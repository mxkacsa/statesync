package statesync

import (
	"sync"
)

// Effect is a reversible state transformation.
// Effects don't mutate base state - they transform on read.
// T is the state type, A is the activator type (e.g., string for playerID).
// Use *A or a wrapper type if you need to distinguish "no activator" from zero value.
type Effect[T, A any] interface {
	ID() string
	Apply(state T, activator A) T
	Activator() A
	SetActivator(activator A)
}

// Expirable interface for effects that can expire.
// Implement this on your effect type to enable CleanupExpired().
type Expirable interface {
	Expired() bool
}

// Schedulable is an interface for effects that can schedule automatic expiration.
// Implement this on your effect type to enable automatic timer-based cleanup.
type Schedulable interface {
	// ScheduleExpiration starts a timer that calls the callback when the effect expires.
	// Returns false if scheduling is not possible (no expiration, already expired, etc).
	ScheduleExpiration(onExpire func(effectID string)) bool

	// CancelScheduledExpiration stops any pending expiration timer.
	CancelScheduledExpiration()
}

// Func creates a simple effect from a function.
// This is the basic building block - use it directly or as a base for custom effects.
func Func[T, A any](id string, fn func(state T, activator A) T) *FuncEffect[T, A] {
	return &FuncEffect[T, A]{id: id, fn: fn}
}

// FuncEffect is a simple function-based effect.
// Thread-safe: Activator() and SetActivator() can be called concurrently.
type FuncEffect[T, A any] struct {
	mu        sync.RWMutex
	id        string
	fn        func(T, A) T
	activator A
}

func (e *FuncEffect[T, A]) ID() string { return e.id }

func (e *FuncEffect[T, A]) Apply(s T, activator A) T {
	return e.fn(s, activator)
}

func (e *FuncEffect[T, A]) Activator() A {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.activator
}

func (e *FuncEffect[T, A]) SetActivator(activator A) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.activator = activator
}
