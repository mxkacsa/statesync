package statesync

import (
	"sync"
)

// TrackedState manages state with automatic change tracking
// T must implement Trackable
type TrackedState[T Trackable, A any] struct {
	mu          sync.RWMutex
	current     T
	effects     []Effect[T, A]
	encoderPool sync.Pool // pool of *Encoder instances (eliminates encoder lock contention)
	registry    *SchemaRegistry
}

// TrackedConfig configuration for TrackedState
type TrackedConfig struct {
	Registry *SchemaRegistry
}

// NewTrackedState creates a new TrackedState
func NewTrackedState[T Trackable, A any](initial T, cfg *TrackedConfig) *TrackedState[T, A] {
	var registry *SchemaRegistry
	if cfg != nil && cfg.Registry != nil {
		registry = cfg.Registry
	} else {
		registry = NewSchemaRegistry()
		registry.Register(initial.Schema())
	}

	ts := &TrackedState[T, A]{
		current:  initial,
		effects:  make([]Effect[T, A], 0),
		registry: registry,
	}
	ts.encoderPool.New = func() interface{} {
		return NewEncoder(registry)
	}
	return ts
}

// Get returns the current state with all effects applied.
// WARNING: If T is a pointer type, the returned value shares memory with internal state.
// Use Read() for safe concurrent access, or ensure single-writer access pattern.
func (s *TrackedState[T, A]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.withEffects(s.current)
}

// GetBase returns the current state without effects.
// WARNING: If T is a pointer type, the returned value shares memory with internal state.
// Use ReadBase() for safe concurrent access, or ensure single-writer access pattern.
func (s *TrackedState[T, A]) GetBase() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// Update modifies the state via a callback.
// The lock is held for the duration of fn - keep it fast.
// Changes are automatically tracked by the Trackable implementation.
func (s *TrackedState[T, A]) Update(fn func(*T)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.current)
}

// UpdateInPlace provides write access to the current state value directly.
// Use this when T is a pointer type (e.g., *GameState) and you want to call methods
// on it without double-pointer indirection: fn receives the pointer, not *pointer.
func (s *TrackedState[T, A]) UpdateInPlace(fn func(T)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.current)
}

// Read provides safe read access to the state.
// The lock is held for the duration of fn - keep it fast.
// Use this instead of Get() when you need to read multiple fields atomically.
func (s *TrackedState[T, A]) Read(fn func(T)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s.withEffects(s.current))
}

// ReadBase provides safe read access to the base state without effects.
func (s *TrackedState[T, A]) ReadBase(fn func(T)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s.current)
}

// Set replaces the entire state
// This marks all fields as changed
func (s *TrackedState[T, A]) Set(newState T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = newState
	s.current.MarkAllDirty()
}

// Encode returns the binary encoded changes
// Returns nil if no changes
func (s *TrackedState[T, A]) Encode() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := s.withEffects(s.current)
	if !state.Changes().HasChanges() {
		return nil
	}
	return s.poolEncode(state)
}

// EncodeAll returns the full state as binary (for initial sync)
func (s *TrackedState[T, A]) EncodeAll() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := s.withEffects(s.current)
	return s.poolEncodeAll(state)
}

// EncodeWithFilter encodes with a filter function
func (s *TrackedState[T, A]) EncodeWithFilter(filter func(T) T) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := s.withEffects(s.current)
	if filter != nil {
		state = filter(state)
	}
	if isNilTrackable(state) || !state.Changes().HasChanges() {
		return nil
	}
	return s.poolEncode(state)
}

// EncodeAllWithFilter encodes full state with filter
func (s *TrackedState[T, A]) EncodeAllWithFilter(filter func(T) T) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := s.withEffects(s.current)
	if filter != nil {
		state = filter(state)
	}
	if isNilTrackable(state) {
		return nil
	}
	return s.poolEncodeAll(state)
}

// poolEncode gets an encoder from the pool, encodes changes, and returns it.
func (s *TrackedState[T, A]) poolEncode(state Trackable) []byte {
	enc := s.encoderPool.Get().(*Encoder)
	data := enc.Encode(state)
	s.encoderPool.Put(enc)
	return data
}

// poolEncodeAll gets an encoder from the pool, encodes full state, and returns it.
func (s *TrackedState[T, A]) poolEncodeAll(state Trackable) []byte {
	enc := s.encoderPool.Get().(*Encoder)
	data := enc.EncodeAll(state)
	s.encoderPool.Put(enc)
	return data
}

// lockedEncode encodes changes for a pre-resolved state using the encoder pool.
func (s *TrackedState[T, A]) lockedEncode(state Trackable) []byte {
	return s.poolEncode(state)
}

// lockedEncodeAll encodes full state for a pre-resolved state using the encoder pool.
func (s *TrackedState[T, A]) lockedEncodeAll(state Trackable) []byte {
	return s.poolEncodeAll(state)
}

// Commit clears all tracked changes
// Call after broadcasting to all clients
func (s *TrackedState[T, A]) Commit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current.ClearChanges()
}

// HasChanges returns true if there are uncommitted changes
func (s *TrackedState[T, A]) HasChanges() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current.Changes().HasChanges()
}

// Effect management

// AddEffect adds an effect to the state.
// If the effect implements Activatable, OnActivate is called once with write access to the base state.
func (s *TrackedState[T, A]) AddEffect(e Effect[T, A], activator A) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.effects {
		if existing.ID() == e.ID() {
			return ErrEffectExists
		}
	}

	e.SetActivator(activator)
	s.effects = append(s.effects, e)

	// Call OnActivate for one-time setup (notifications, initial values, etc.)
	if act, ok := any(e).(Activatable[T, A]); ok {
		s.current = act.OnActivate(s.current, activator)
	}

	return nil
}

// RemoveEffect removes an effect by ID
func (s *TrackedState[T, A]) RemoveEffect(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, e := range s.effects {
		if e.ID() == id {
			// Cancel any scheduled expiration timer
			if sched, ok := any(e).(Schedulable); ok {
				sched.CancelScheduledExpiration()
			}
			s.effects = append(s.effects[:i], s.effects[i+1:]...)
			return true
		}
	}
	return false
}

// HasEffect checks if an effect exists
func (s *TrackedState[T, A]) HasEffect(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.effects {
		if e.ID() == id {
			return true
		}
	}
	return false
}

// GetEffect returns an effect by ID
func (s *TrackedState[T, A]) GetEffect(id string) Effect[T, A] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.effects {
		if e.ID() == id {
			return e
		}
	}
	return nil
}

// ClearEffects removes all effects
func (s *TrackedState[T, A]) ClearEffects() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Cancel all scheduled expiration timers
	for _, e := range s.effects {
		if sched, ok := any(e).(Schedulable); ok {
			sched.CancelScheduledExpiration()
		}
	}
	s.effects = make([]Effect[T, A], 0)
}

// Effects returns a copy of all active effects
func (s *TrackedState[T, A]) Effects() []Effect[T, A] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.effects) == 0 {
		return nil
	}
	return append([]Effect[T, A]{}, s.effects...)
}

// CleanupExpired removes all expired effects.
// Returns the number of effects removed.
func (s *TrackedState[T, A]) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.effects) == 0 {
		return 0
	}

	removed := 0
	active := s.effects[:0]
	for _, e := range s.effects {
		if exp, ok := any(e).(Expirable); ok && exp.Expired() {
			if sched, ok := any(e).(Schedulable); ok {
				sched.CancelScheduledExpiration()
			}
			removed++
			continue
		}
		active = append(active, e)
	}
	// Clear tail to allow GC of removed effects
	for i := len(active); i < len(s.effects); i++ {
		s.effects[i] = nil
	}
	s.effects = active

	return removed
}

// withEffects applies all effects to the state
// Note: This creates a copy to avoid mutating the base state
func (s *TrackedState[T, A]) withEffects(state T) T {
	result := state
	for _, e := range s.effects {
		result = e.Apply(result, e.Activator())
	}
	return result
}

// Registry returns the schema registry
func (s *TrackedState[T, A]) Registry() *SchemaRegistry {
	return s.registry
}

// ErrEffectExists is returned when adding a duplicate effect
var ErrEffectExists = &DuplicateEffectError{}

// DuplicateEffectError indicates an effect with the same ID already exists
type DuplicateEffectError struct{}

func (e *DuplicateEffectError) Error() string {
	return "effect with this ID already exists"
}
