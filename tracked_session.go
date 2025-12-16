package statesync

import (
	"encoding/json"
	"sync"
	"time"
)

// FilterFunc transforms state for a specific viewer (hides private data)
type FilterFunc[T any] func(T) T

// Hooks for intercepting the broadcast pipeline
type SessionHooks[T Trackable, ID comparable] struct {
	// OnBeforeFilter is called before filtering state for each client
	// Receives: clientID, raw state
	OnBeforeFilter func(clientID ID, state T)

	// OnAfterFilter is called after filtering state for each client
	// Receives: clientID, filtered state
	OnAfterFilter func(clientID ID, filtered T)

	// OnBeforeEncode is called before encoding state to binary
	// Receives: clientID, state to encode
	OnBeforeEncode func(clientID ID, state T)

	// OnAfterEncode is called after encoding, before adding to result map
	// Receives: clientID, encoded bytes
	// Return modified bytes or nil to skip this client
	OnAfterEncode func(clientID ID, data []byte) []byte

	// OnBeforeBroadcast is called once before broadcasting to all clients
	// Receives: map of clientID -> encoded data
	// Return modified map (can filter/modify clients)
	OnBeforeBroadcast func(diffs map[ID][]byte) map[ID][]byte

	// OnAfterBroadcast is called after broadcast completes
	// Receives: final map sent to clients, sequence number
	OnAfterBroadcast func(diffs map[ID][]byte, seq uint64)
}

// TrackedSession manages multiple clients with binary state sync
type TrackedSession[T Trackable, A any, ID comparable] struct {
	mu    sync.RWMutex
	state *TrackedState[T, A]

	// Client filters (nil = full state)
	clients map[ID]FilterFunc[T]

	// Full state tracking for new clients
	clientNeedsFull map[ID]bool

	// Sequence tracking for reconnection support
	seq         uint64             // Current sequence number (increments on each Tick)
	clientSeq   map[ID]uint64      // Last acknowledged sequence per client
	history     []historyEntry[ID] // Ring buffer of recent updates
	historySize int                // Max history entries (0 = disabled)

	// Debounce support
	debounceMu    sync.Mutex
	debounce      time.Duration
	debounceTimer *time.Timer
	onBroadcast   func(map[ID][]byte)

	// Pipeline hooks
	hooks SessionHooks[T, ID]

	// Event system
	events *EventBuffer[ID]
}

// historyEntry stores diffs at a specific sequence number
type historyEntry[ID comparable] struct {
	seq      uint64
	baseDiff []byte        // Diff without filter (for reconnection)
	diffs    map[ID][]byte // Per-client diffs (for clients with filter)
}

// NewTrackedSession creates a new session with binary state sync
func NewTrackedSession[T Trackable, A any, ID comparable](state *TrackedState[T, A]) *TrackedSession[T, A, ID] {
	return &TrackedSession[T, A, ID]{
		state:           state,
		clients:         make(map[ID]FilterFunc[T]),
		clientNeedsFull: make(map[ID]bool),
		clientSeq:       make(map[ID]uint64),
		seq:             1, // Start at 1 so 0 means "no previous sequence"
		events:          NewEventBuffer[ID](),
	}
}

// SetHooks configures pipeline hooks for interception/logging
func (s *TrackedSession[T, A, ID]) SetHooks(hooks SessionHooks[T, ID]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hooks = hooks
}

// SetHistorySize configures the reconnection history buffer size.
// Set to 0 to disable history (clients always get full state on reconnect).
// Recommended: 30-60 for 20 tick/s games (1.5-3 seconds of history).
func (s *TrackedSession[T, A, ID]) SetHistorySize(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.historySize = size
	if size > 0 && s.history == nil {
		s.history = make([]historyEntry[ID], 0, size)
	}
}

// Seq returns the current sequence number
func (s *TrackedSession[T, A, ID]) Seq() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.seq
}

// ClientSeq returns the last acknowledged sequence for a client
func (s *TrackedSession[T, A, ID]) ClientSeq(id ID) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clientSeq[id]
}

// AckSeq acknowledges that a client has received updates up to seq.
// This should be called when the client confirms receipt of updates.
func (s *TrackedSession[T, A, ID]) AckSeq(id ID, seq uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if seq > s.clientSeq[id] {
		s.clientSeq[id] = seq
	}
}

// Connect adds a client with optional filter function.
// Filter transforms state to hide private data from this client.
// Pass nil for full state access (admin/spectator).
func (s *TrackedSession[T, A, ID]) Connect(id ID, filter FilterFunc[T]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[id] = filter
	s.clientNeedsFull[id] = true
}

// Disconnect removes a client
func (s *TrackedSession[T, A, ID]) Disconnect(id ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, id)
	delete(s.clientNeedsFull, id)
	delete(s.clientSeq, id)
}

// ClientCount returns the number of connected clients
func (s *TrackedSession[T, A, ID]) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// Clients returns a list of connected client IDs
func (s *TrackedSession[T, A, ID]) Clients() []ID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]ID, 0, len(s.clients))
	for id := range s.clients {
		ids = append(ids, id)
	}
	return ids
}

// HasClient checks if a client is connected
func (s *TrackedSession[T, A, ID]) HasClient(id ID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.clients[id]
	return ok
}

// GetFilter returns the filter function for a client (nil if none)
func (s *TrackedSession[T, A, ID]) GetFilter(id ID) FilterFunc[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients[id]
}

// SetFilter updates the filter function for a connected client
func (s *TrackedSession[T, A, ID]) SetFilter(id ID, filter FilterFunc[T]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[id]; ok {
		s.clients[id] = filter
	}
}

// State returns the underlying TrackedState
func (s *TrackedSession[T, A, ID]) State() *TrackedState[T, A] {
	return s.state
}

// Full returns the full binary state for a specific client (for initial sync)
func (s *TrackedSession[T, A, ID]) Full(id ID) []byte {
	s.mu.RLock()
	filter := s.clients[id]
	hooks := s.hooks
	s.mu.RUnlock()

	state := s.state.Get()

	// Hook: before filter
	if hooks.OnBeforeFilter != nil {
		hooks.OnBeforeFilter(id, state)
	}

	// Apply filter
	if filter != nil {
		state = filter(state)
	}

	// Hook: after filter
	if hooks.OnAfterFilter != nil {
		hooks.OnAfterFilter(id, state)
	}

	// Hook: before encode
	if hooks.OnBeforeEncode != nil {
		hooks.OnBeforeEncode(id, state)
	}

	// Encode
	data := s.state.encoder.EncodeAll(state)

	// Hook: after encode
	if hooks.OnAfterEncode != nil {
		data = hooks.OnAfterEncode(id, data)
	}

	return data
}

// Diff returns the binary diff for a specific client
func (s *TrackedSession[T, A, ID]) Diff(id ID) []byte {
	s.mu.Lock()
	filter := s.clients[id]
	needsFull := s.clientNeedsFull[id]
	if needsFull {
		s.clientNeedsFull[id] = false
	}
	s.mu.Unlock()

	if needsFull {
		return s.Full(id)
	}

	if filter != nil {
		return s.state.EncodeWithFilter(filter)
	}
	return s.state.Encode()
}

// Broadcast returns binary diffs for all connected clients
func (s *TrackedSession[T, A, ID]) Broadcast() map[ID][]byte {
	s.mu.Lock()
	clients := make(map[ID]FilterFunc[T], len(s.clients))
	needsFullMap := make(map[ID]bool, len(s.clients))
	for id, filter := range s.clients {
		clients[id] = filter
		needsFullMap[id] = s.clientNeedsFull[id]
		if s.clientNeedsFull[id] {
			s.clientNeedsFull[id] = false
		}
	}
	hooks := s.hooks
	s.mu.Unlock()

	if len(clients) == 0 {
		return nil
	}

	result := make(map[ID][]byte, len(clients))

	// Get raw state once
	rawState := s.state.Get()

	// Cache for nil filter (full state view)
	var fullDiff []byte
	var fullDiffComputed bool

	for id, filter := range clients {
		needsFull := needsFullMap[id]

		var data []byte

		// Hook: before filter
		if hooks.OnBeforeFilter != nil {
			hooks.OnBeforeFilter(id, rawState)
		}

		// Apply filter
		state := rawState
		if filter != nil {
			state = filter(rawState)
		}

		// Hook: after filter
		if hooks.OnAfterFilter != nil {
			hooks.OnAfterFilter(id, state)
		}

		// Hook: before encode
		if hooks.OnBeforeEncode != nil {
			hooks.OnBeforeEncode(id, state)
		}

		// Encode
		if needsFull {
			// New client needs full state
			data = s.state.encoder.EncodeAll(state)
		} else if filter == nil {
			// Use cached full diff for unfiltered clients
			if !fullDiffComputed {
				fullDiff = s.state.Encode()
				fullDiffComputed = true
			}
			data = fullDiff
		} else {
			// Filtered diff
			if !state.Changes().HasChanges() {
				continue
			}
			data = s.state.encoder.Encode(state)
		}

		// Hook: after encode
		if hooks.OnAfterEncode != nil {
			data = hooks.OnAfterEncode(id, data)
		}

		if len(data) > 0 {
			result[id] = data
		}
	}

	// Hook: before broadcast
	if hooks.OnBeforeBroadcast != nil {
		result = hooks.OnBeforeBroadcast(result)
	}

	return result
}

// Tick performs a full update cycle: broadcast + commit changes + increment sequence
func (s *TrackedSession[T, A, ID]) Tick() map[ID][]byte {
	diffs := s.Broadcast()

	// Store base diff before commit (for reconnection without filter)
	var baseDiff []byte
	s.mu.RLock()
	storeHistory := s.historySize > 0
	hooks := s.hooks
	s.mu.RUnlock()

	if storeHistory {
		baseDiff = s.state.Encode() // Get diff without filter
	}

	s.state.Commit()

	// Handle sequence and history (under lock)
	s.mu.Lock()
	currentSeq := s.seq
	s.seq++

	// Store in history if enabled and there are changes
	if s.historySize > 0 && len(baseDiff) > 0 {
		// Deep copy diffs for history (original may be reused)
		historyDiffs := make(map[ID][]byte, len(diffs))
		for id, data := range diffs {
			cp := make([]byte, len(data))
			copy(cp, data)
			historyDiffs[id] = cp
		}

		entry := historyEntry[ID]{
			seq:      currentSeq,
			baseDiff: baseDiff,
			diffs:    historyDiffs,
		}

		if len(s.history) < s.historySize {
			s.history = append(s.history, entry)
		} else {
			// Ring buffer: shift left and add at end
			copy(s.history, s.history[1:])
			s.history[len(s.history)-1] = entry
		}
	}
	s.mu.Unlock()

	// Hook: after broadcast
	if hooks.OnAfterBroadcast != nil {
		hooks.OnAfterBroadcast(diffs, currentSeq)
	}

	return diffs
}

// TickWithSeq performs Tick and returns both diffs and the sequence number.
// The returned sequence should be sent to clients so they can acknowledge receipt.
func (s *TrackedSession[T, A, ID]) TickWithSeq() (map[ID][]byte, uint64) {
	s.mu.RLock()
	seq := s.seq
	s.mu.RUnlock()

	diffs := s.Tick()
	return diffs, seq
}

// GetPendingSince returns all updates since the given sequence number for a client.
// If the sequence is too old (not in history), returns nil and false.
// If the client is up to date, returns empty slice and true.
// Otherwise returns the pending diffs and true.
// Note: For clients that were disconnected, this returns the base diff (no filter).
func (s *TrackedSession[T, A, ID]) GetPendingSince(id ID, sinceSeq uint64) ([][]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Client is up to date
	if sinceSeq >= s.seq-1 {
		return [][]byte{}, true
	}

	// No history configured
	if s.historySize == 0 || len(s.history) == 0 {
		return nil, false
	}

	// Check if we have history going back far enough
	oldestSeq := s.history[0].seq
	if sinceSeq < oldestSeq {
		// Requested sequence is too old, client needs full state
		return nil, false
	}

	// Collect all diffs since the requested sequence
	var pending [][]byte
	for _, entry := range s.history {
		if entry.seq > sinceSeq {
			// Try client-specific diff first (has filter applied)
			if data, ok := entry.diffs[id]; ok && len(data) > 0 {
				pending = append(pending, data)
			} else if len(entry.baseDiff) > 0 {
				// Fall back to base diff (no filter)
				pending = append(pending, entry.baseDiff)
			}
		}
	}

	return pending, true
}

// Reconnect handles a client reconnecting with their last known sequence.
// Returns the data to send and whether it's a full state (true) or incremental updates (false).
// If updates is nil, there's nothing to send (client is up to date).
func (s *TrackedSession[T, A, ID]) Reconnect(id ID, lastSeq uint64, filter FilterFunc[T]) (updates [][]byte, isFull bool) {
	// Try to get incremental updates from history
	pending, ok := s.GetPendingSince(id, lastSeq)
	if ok && len(pending) > 0 {
		// Can resume with incremental updates
		s.mu.Lock()
		s.clients[id] = filter
		s.clientNeedsFull[id] = false
		s.clientSeq[id] = lastSeq
		s.mu.Unlock()
		return pending, false
	}

	if ok && len(pending) == 0 {
		// Client is up to date, just reconnect
		s.mu.Lock()
		s.clients[id] = filter
		s.clientNeedsFull[id] = false
		s.clientSeq[id] = s.seq - 1
		s.mu.Unlock()
		return nil, false
	}

	// Need full state sync
	s.mu.Lock()
	s.clients[id] = filter
	s.clientNeedsFull[id] = false
	s.clientSeq[id] = s.seq - 1
	s.mu.Unlock()

	fullData := s.Full(id)
	return [][]byte{fullData}, true
}

// Debounce support

// SetDebounce configures debounce duration
func (s *TrackedSession[T, A, ID]) SetDebounce(d time.Duration) {
	s.debounceMu.Lock()
	defer s.debounceMu.Unlock()
	s.debounce = d
}

// SetBroadcastCallback sets the callback for debounced broadcasts
func (s *TrackedSession[T, A, ID]) SetBroadcastCallback(fn func(map[ID][]byte)) {
	s.debounceMu.Lock()
	defer s.debounceMu.Unlock()
	s.onBroadcast = fn
}

// ScheduleBroadcast schedules a debounced broadcast
func (s *TrackedSession[T, A, ID]) ScheduleBroadcast() {
	s.debounceMu.Lock()

	if s.debounce == 0 {
		// No debounce, broadcast immediately
		// Get callback and release lock before calling Tick/callback
		// to avoid holding lock during potentially long operations
		callback := s.onBroadcast
		s.debounceMu.Unlock()

		if callback != nil {
			diffs := s.Tick()
			if len(diffs) > 0 {
				callback(diffs)
			}
		}
		return
	}

	// Cancel existing timer
	if s.debounceTimer != nil {
		s.debounceTimer.Stop()
	}

	// Schedule new broadcast
	s.debounceTimer = time.AfterFunc(s.debounce, func() {
		s.debounceMu.Lock()
		callback := s.onBroadcast
		s.debounceMu.Unlock()

		if callback != nil {
			diffs := s.Tick()
			if len(diffs) > 0 {
				callback(diffs)
			}
		}
	})
	s.debounceMu.Unlock()
}

// Transaction-based updates

// TrackedTx represents a transaction for batched updates
type TrackedTx[T Trackable, A any] struct {
	state *TrackedState[T, A]
}

// Transaction executes a batch of updates and returns binary diffs
func (s *TrackedSession[T, A, ID]) Transaction(fn func(tx *TrackedTx[T, A])) map[ID][]byte {
	tx := &TrackedTx[T, A]{state: s.state}
	fn(tx)
	return s.Tick()
}

// Update modifies the state within a transaction
func (tx *TrackedTx[T, A]) Update(fn func(*T)) {
	tx.state.Update(fn)
}

// Get returns the current state
func (tx *TrackedTx[T, A]) Get() T {
	return tx.state.Get()
}

// GetBase returns the base state without effects
func (tx *TrackedTx[T, A]) GetBase() T {
	return tx.state.GetBase()
}

// ApplyUpdate is a shorthand for a single-update transaction
func (s *TrackedSession[T, A, ID]) ApplyUpdate(fn func(*T)) map[ID][]byte {
	return s.Transaction(func(tx *TrackedTx[T, A]) {
		tx.Update(fn)
	})
}

// Effect management proxies

// AddEffect adds an effect to the underlying state
func (s *TrackedSession[T, A, ID]) AddEffect(e Effect[T, A], activator A) error {
	return s.state.AddEffect(e, activator)
}

// RemoveEffect removes an effect from the underlying state
func (s *TrackedSession[T, A, ID]) RemoveEffect(id string) bool {
	return s.state.RemoveEffect(id)
}

// HasEffect checks if an effect exists
func (s *TrackedSession[T, A, ID]) HasEffect(id string) bool {
	return s.state.HasEffect(id)
}

// GetEffect returns an effect by ID
func (s *TrackedSession[T, A, ID]) GetEffect(id string) Effect[T, A] {
	return s.state.GetEffect(id)
}

// ClearEffects removes all effects
func (s *TrackedSession[T, A, ID]) ClearEffects() {
	s.state.ClearEffects()
}

// Convenience methods

// UpdateAndBroadcast updates the state and returns diffs in one call
func (s *TrackedSession[T, A, ID]) UpdateAndBroadcast(fn func(*T)) map[ID][]byte {
	s.state.Update(fn)
	return s.Tick()
}

// Get returns the current state with effects.
// WARNING: If T is a pointer type, use Read() for safe concurrent access.
func (s *TrackedSession[T, A, ID]) Get() T {
	return s.state.Get()
}

// GetBase returns the base state without effects.
// WARNING: If T is a pointer type, use ReadBase() for safe concurrent access.
func (s *TrackedSession[T, A, ID]) GetBase() T {
	return s.state.GetBase()
}

// Read provides safe read access to the state with effects.
func (s *TrackedSession[T, A, ID]) Read(fn func(T)) {
	s.state.Read(fn)
}

// ReadBase provides safe read access to the base state without effects.
func (s *TrackedSession[T, A, ID]) ReadBase(fn func(T)) {
	s.state.ReadBase(fn)
}

// ============================================================================
// EventEmitter implementation
// ============================================================================

// Emit sends an event to all connected clients.
// The event will be included in the next Tick() result.
func (s *TrackedSession[T, A, ID]) Emit(eventType string, payload any) error {
	encoded, err := encodePayload(payload)
	if err != nil {
		return err
	}
	s.events.Add(PendingEvent[ID]{
		Event:  Event{Type: eventType, Payload: encoded},
		Target: TargetAll,
	})
	return nil
}

// EmitTo sends an event to a specific client.
// The event will be included in the next Tick() result for that client only.
func (s *TrackedSession[T, A, ID]) EmitTo(clientID ID, eventType string, payload any) error {
	encoded, err := encodePayload(payload)
	if err != nil {
		return err
	}
	s.events.Add(PendingEvent[ID]{
		Event:  Event{Type: eventType, Payload: encoded},
		Target: TargetOne,
		To:     clientID,
	})
	return nil
}

// EmitExcept sends an event to all clients except one.
// The event will be included in the next Tick() result for all clients except exceptID.
func (s *TrackedSession[T, A, ID]) EmitExcept(exceptID ID, eventType string, payload any) error {
	encoded, err := encodePayload(payload)
	if err != nil {
		return err
	}
	s.events.Add(PendingEvent[ID]{
		Event:  Event{Type: eventType, Payload: encoded},
		Target: TargetExcept,
		Except: exceptID,
	})
	return nil
}

// EmitToMany sends an event to multiple specific clients.
// The event will be included in the next Tick() result for the specified clients.
func (s *TrackedSession[T, A, ID]) EmitToMany(clientIDs []ID, eventType string, payload any) error {
	encoded, err := encodePayload(payload)
	if err != nil {
		return err
	}
	s.events.Add(PendingEvent[ID]{
		Event:  Event{Type: eventType, Payload: encoded},
		Target: TargetMany,
		ToMany: clientIDs,
	})
	return nil
}

// EmitRaw sends a pre-encoded event to all clients.
func (s *TrackedSession[T, A, ID]) EmitRaw(event Event) error {
	s.events.Add(PendingEvent[ID]{
		Event:  event,
		Target: TargetAll,
	})
	return nil
}

// EmitRawTo sends a pre-encoded event to a specific client.
func (s *TrackedSession[T, A, ID]) EmitRawTo(clientID ID, event Event) error {
	s.events.Add(PendingEvent[ID]{
		Event:  event,
		Target: TargetOne,
		To:     clientID,
	})
	return nil
}

// PendingEvents returns the number of events waiting to be broadcast
func (s *TrackedSession[T, A, ID]) PendingEvents() int {
	return s.events.Count()
}

// ClearEvents removes all pending events without broadcasting them
func (s *TrackedSession[T, A, ID]) ClearEvents() {
	s.events.Clear()
}

// TickResult contains both state diffs and events for each client
type TickResult[ID comparable] struct {
	// Diffs contains state changes per client (same as Tick() return value)
	Diffs map[ID][]byte

	// Events contains encoded events per client
	// Multiple events are batched into a single []byte using EncodeEventBatch
	Events map[ID][]byte

	// Seq is the sequence number for this tick
	Seq uint64
}

// TickWithEvents performs a full update cycle and returns both diffs and events.
// This is the recommended method when using events.
//
// Example:
//
//	session.Emit("CardPlayed", map[string]any{"cardId": "c7"})
//	session.State().Update(func(g *Game) { g.Phase = "next" })
//	result := session.TickWithEvents()
//	for clientID, diff := range result.Diffs {
//	    sendToClient(clientID, diff)
//	}
//	for clientID, events := range result.Events {
//	    sendToClient(clientID, events)
//	}
func (s *TrackedSession[T, A, ID]) TickWithEvents() TickResult[ID] {
	// Fast path: check for events first (lock-free)
	hasEvents := s.events.HasEvents()

	// Get state diffs
	diffs := s.Tick()

	// Get sequence (already incremented by Tick)
	s.mu.RLock()
	seq := s.seq - 1 // Tick incremented it, we want the seq for this tick
	s.mu.RUnlock()

	if !hasEvents {
		return TickResult[ID]{Diffs: diffs, Seq: seq}
	}

	// Drain events
	pending := s.events.Drain()
	if len(pending) == 0 {
		return TickResult[ID]{Diffs: diffs, Seq: seq}
	}

	// Get client list under lock
	s.mu.RLock()
	clientIDs := make([]ID, 0, len(s.clients))
	for id := range s.clients {
		clientIDs = append(clientIDs, id)
	}
	s.mu.RUnlock()

	// Build client lookup set for TargetOne/TargetMany validation
	clientSet := make(map[ID]struct{}, len(clientIDs))
	for _, id := range clientIDs {
		clientSet[id] = struct{}{}
	}

	// Group events by client
	clientEvents := make(map[ID][]Event, len(clientIDs))
	for _, pe := range pending {
		switch pe.Target {
		case TargetAll:
			for _, id := range clientIDs {
				clientEvents[id] = append(clientEvents[id], pe.Event)
			}
		case TargetOne:
			if _, ok := clientSet[pe.To]; ok {
				clientEvents[pe.To] = append(clientEvents[pe.To], pe.Event)
			}
		case TargetExcept:
			for _, id := range clientIDs {
				if id != pe.Except {
					clientEvents[id] = append(clientEvents[id], pe.Event)
				}
			}
		case TargetMany:
			for _, id := range pe.ToMany {
				if _, ok := clientSet[id]; ok {
					clientEvents[id] = append(clientEvents[id], pe.Event)
				}
			}
		}
	}

	// Encode events per client
	events := make(map[ID][]byte, len(clientEvents))
	for id, evts := range clientEvents {
		events[id] = EncodeEventBatch(evts)
	}

	return TickResult[ID]{
		Diffs:  diffs,
		Events: events,
		Seq:    seq,
	}
}

// encodePayload encodes a payload to bytes
// Supports: []byte (passthrough), string, nil, or JSON-serializable types
func encodePayload(payload any) ([]byte, error) {
	if payload == nil {
		return nil, nil
	}
	switch p := payload.(type) {
	case []byte:
		return p, nil
	case string:
		return []byte(p), nil
	default:
		// For complex types, use JSON encoding
		// This is a simple approach; for better performance, users can pre-encode
		return json.Marshal(p)
	}
}
