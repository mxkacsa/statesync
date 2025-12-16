package statesync

import (
	"sync"
	"testing"
)

// TestGameState is a manually implemented Trackable for testing
type TestGameState struct {
	mu      sync.RWMutex
	changes *ChangeSet
	schema  *Schema

	// Fields
	round   int
	phase   string
	players []TestPlayer
	scores  map[string]int
}

type TestPlayer struct {
	ID    string
	Name  string
	Score int
}

func NewTestGameState() *TestGameState {
	return &TestGameState{
		changes: NewChangeSet(),
		schema:  testGameStateSchema(),
		players: make([]TestPlayer, 0),
		scores:  make(map[string]int),
	}
}

func testGameStateSchema() *Schema {
	playerSchema := NewSchemaBuilder("TestPlayer").
		String("ID").
		String("Name").
		Int64("Score").
		Build()

	return NewSchemaBuilder("TestGameState").
		WithID(1).
		Int64("round").
		String("phase").
		ArrayByKey("players", TypeStruct, playerSchema, "ID").
		Map("scores", TypeInt64, nil).
		Build()
}

// Trackable implementation
func (g *TestGameState) Schema() *Schema     { return g.schema }
func (g *TestGameState) Changes() *ChangeSet { return g.changes }
func (g *TestGameState) ClearChanges()       { g.changes.Clear() }
func (g *TestGameState) MarkAllDirty()       { g.changes.MarkAll(3) }

func (g *TestGameState) GetFieldValue(index uint8) interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()
	switch index {
	case 0:
		return int64(g.round)
	case 1:
		return g.phase
	case 2:
		return g.players
	case 3:
		return g.scores
	}
	return nil
}

// FastEncoder implementation - zero allocation encoding

func (g *TestGameState) EncodeChangesTo(e *Encoder) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Count changes
	count := 0
	if g.changes.IsFieldDirty(0) {
		count++
	}
	if g.changes.IsFieldDirty(1) {
		count++
	}
	// Skip complex types (2: players array, 3: scores map) for now
	e.WriteChangeCount(count)

	// Encode primitive fields directly - no interface{} boxing!
	if g.changes.IsFieldDirty(0) {
		e.WriteFieldHeader(0, OpReplace)
		e.WriteInt64(int64(g.round))
	}
	if g.changes.IsFieldDirty(1) {
		e.WriteFieldHeader(1, OpReplace)
		e.WriteString(g.phase)
	}
}

func (g *TestGameState) EncodeAllTo(e *Encoder) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Encode all fields directly
	e.WriteInt64(int64(g.round))
	e.WriteString(g.phase)
	// Skip complex types for benchmark simplicity
}

// Getters
func (g *TestGameState) Round() int             { g.mu.RLock(); defer g.mu.RUnlock(); return g.round }
func (g *TestGameState) Phase() string          { g.mu.RLock(); defer g.mu.RUnlock(); return g.phase }
func (g *TestGameState) Players() []TestPlayer  { g.mu.RLock(); defer g.mu.RUnlock(); return g.players }
func (g *TestGameState) Scores() map[string]int { g.mu.RLock(); defer g.mu.RUnlock(); return g.scores }

// Tracked setters
func (g *TestGameState) SetRound(v int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.round != v {
		g.round = v
		g.changes.Mark(0, OpReplace)
	}
}

func (g *TestGameState) SetPhase(v string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != v {
		g.phase = v
		g.changes.Mark(1, OpReplace)
	}
}

func (g *TestGameState) AddPlayer(p TestPlayer) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.players = append(g.players, p)
	arr := g.changes.GetOrCreateArray(2)
	arr.MarkAdd(len(g.players)-1, p)
}

func (g *TestGameState) SetScore(key string, v int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, existed := g.scores[key]
	g.scores[key] = v
	m := g.changes.GetOrCreateMap(3)
	if existed {
		m.MarkReplace(key, v)
	} else {
		m.MarkAdd(key, v)
	}
}

// Tests

func TestChangeSet(t *testing.T) {
	cs := NewChangeSet()

	// Initially empty
	if cs.HasChanges() {
		t.Error("new ChangeSet should have no changes")
	}

	// Mark a field
	cs.Mark(0, OpReplace)
	if !cs.HasChanges() {
		t.Error("should have changes after Mark")
	}

	// Check field change
	change := cs.GetFieldChange(0)
	if change.Op != OpReplace {
		t.Errorf("expected OpReplace, got %v", change.Op)
	}

	// Clear
	cs.Clear()
	if cs.HasChanges() {
		t.Error("should have no changes after Clear")
	}
}

func TestArrayChangeSet(t *testing.T) {
	cs := NewChangeSet()
	arr := cs.GetOrCreateArray(0)

	arr.MarkAdd(0, "first")
	arr.MarkReplace(0, "updated")
	arr.MarkRemove(1)

	if !arr.HasChanges() {
		t.Error("should have array changes")
	}

	arr.Clear()
	if arr.HasChanges() {
		t.Error("should have no array changes after Clear")
	}
}

func TestMapChangeSet(t *testing.T) {
	cs := NewChangeSet()
	m := cs.GetOrCreateMap(0)

	m.MarkAdd("key1", "value1")
	m.MarkReplace("key1", "value2")
	m.MarkRemove("key2")

	if !m.HasChanges() {
		t.Error("should have map changes")
	}

	m.Clear()
	if m.HasChanges() {
		t.Error("should have no map changes after Clear")
	}
}

func TestSchemaBuilder(t *testing.T) {
	schema := NewSchemaBuilder("TestType").
		WithID(42).
		Int32("count").
		String("name").
		Bool("active").
		Build()

	if schema.ID != 42 {
		t.Errorf("expected ID 42, got %d", schema.ID)
	}
	if schema.Name != "TestType" {
		t.Errorf("expected name TestType, got %s", schema.Name)
	}
	if schema.FieldCount() != 3 {
		t.Errorf("expected 3 fields, got %d", schema.FieldCount())
	}

	// Check fields
	f0 := schema.Field(0)
	if f0.Name != "count" || f0.Type != TypeInt32 {
		t.Errorf("field 0 mismatch: %+v", f0)
	}
	f1 := schema.Field(1)
	if f1.Name != "name" || f1.Type != TypeString {
		t.Errorf("field 1 mismatch: %+v", f1)
	}
	f2 := schema.Field(2)
	if f2.Name != "active" || f2.Type != TypeBool {
		t.Errorf("field 2 mismatch: %+v", f2)
	}

	// Lookup by name
	fn := schema.FieldByName("name")
	if fn == nil || fn.Index != 1 {
		t.Error("FieldByName failed")
	}
}

func TestEncoder(t *testing.T) {
	registry := NewSchemaRegistry()
	state := NewTestGameState()
	registry.Register(state.Schema())

	encoder := NewEncoder(registry)

	// Set some values
	state.SetRound(1)
	state.SetPhase("draw")

	// Encode changes
	data := encoder.Encode(state)
	if data == nil {
		t.Fatal("expected encoded data")
	}

	// Check message type
	if data[0] != MsgPatch {
		t.Errorf("expected MsgPatch, got %d", data[0])
	}

	t.Logf("Encoded %d bytes for 2 field changes", len(data))
}

func TestEncoderFull(t *testing.T) {
	registry := NewSchemaRegistry()
	state := NewTestGameState()
	registry.Register(state.Schema())

	encoder := NewEncoder(registry)

	// Set some values
	state.SetRound(5)
	state.SetPhase("play")
	state.AddPlayer(TestPlayer{ID: "p1", Name: "Alice", Score: 100})

	// Encode full state
	data := encoder.EncodeAll(state)
	if data == nil {
		t.Fatal("expected encoded data")
	}

	// Check message type
	if data[0] != MsgFullState {
		t.Errorf("expected MsgFullState, got %d", data[0])
	}

	t.Logf("Encoded full state: %d bytes", len(data))
}

func TestDecoder(t *testing.T) {
	registry := NewSchemaRegistry()
	state := NewTestGameState()
	registry.Register(state.Schema())

	encoder := NewEncoder(registry)
	decoder := NewDecoder(registry)

	// Set values
	state.SetRound(42)
	state.SetPhase("betting")

	// Encode
	data := encoder.Encode(state)

	// Decode
	patch, err := decoder.Decode(data)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if patch.SchemaID != 1 {
		t.Errorf("expected schema ID 1, got %d", patch.SchemaID)
	}
	if len(patch.Changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(patch.Changes))
	}

	// Verify changes
	for _, change := range patch.Changes {
		switch change.FieldIndex {
		case 0:
			if change.Value.(int64) != 42 {
				t.Errorf("round mismatch: %v", change.Value)
			}
		case 1:
			if change.Value.(string) != "betting" {
				t.Errorf("phase mismatch: %v", change.Value)
			}
		}
	}
}

func TestTrackedState(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)

	// Update via pointer
	tracked.Update(func(s **TestGameState) {
		(*s).SetRound(10)
		(*s).SetPhase("draw")
	})

	// Check changes
	if !tracked.HasChanges() {
		t.Error("should have changes")
	}

	// Encode
	data := tracked.Encode()
	if data == nil {
		t.Error("expected encoded data")
	}

	// Commit
	tracked.Commit()
	if tracked.HasChanges() {
		t.Error("should have no changes after Commit")
	}
}

func TestTrackedSession(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	// Connect clients
	session.Connect("client1", nil)
	session.Connect("client2", nil)

	if session.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", session.ClientCount())
	}

	// First broadcast should be full state for new clients
	state.SetRound(1)
	diffs := session.Tick()

	if len(diffs) != 2 {
		t.Errorf("expected 2 diffs, got %d", len(diffs))
	}

	// Check that data was sent
	for id, data := range diffs {
		if len(data) == 0 {
			t.Errorf("client %s got empty data", id)
		}
		// First message should be full state
		if data[0] != MsgFullState {
			t.Errorf("client %s: expected MsgFullState, got %d", id, data[0])
		}
	}

	// Update and broadcast
	state.SetRound(2)
	diffs = session.Tick()

	for id, data := range diffs {
		if data[0] != MsgPatch {
			t.Errorf("client %s: expected MsgPatch, got %d", id, data[0])
		}
	}

	// Disconnect
	session.Disconnect("client1")
	if session.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", session.ClientCount())
	}
}

func TestTrackedTransaction(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	session.Connect("client1", nil)

	// Use transaction
	diffs := session.Transaction(func(tx *TrackedTx[*TestGameState, string]) {
		tx.Update(func(s **TestGameState) {
			(*s).SetRound(5)
		})
		tx.Update(func(s **TestGameState) {
			(*s).SetPhase("play")
		})
	})

	if len(diffs) != 1 {
		t.Errorf("expected 1 diff, got %d", len(diffs))
	}
}

func TestReconnectionWithHistory(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	// Enable history with 10 entries
	session.SetHistorySize(10)

	// Connect client
	session.Connect("client1", nil)

	// Initial sync - round 1
	state.SetRound(1)
	session.Tick() // seq becomes 2

	// Round 2
	state.SetRound(2)
	session.Tick() // seq becomes 3

	// Round 3
	state.SetRound(3)
	session.Tick()                   // seq becomes 4
	lastSeenSeq := session.Seq() - 1 // Client received this tick (seq 3)

	// Round 4 - client receives this but then disconnects
	state.SetRound(4)
	session.Tick() // seq becomes 5

	// Simulate disconnect - client last saw seq 4 (round 4)
	session.Disconnect("client1")
	lastSeenSeq = session.Seq() - 1 // seq 4

	// More updates happen while client is disconnected
	state.SetRound(5)
	session.Tick() // seq becomes 6

	state.SetRound(6)
	session.Tick() // seq becomes 7

	// Client reconnects with their last known sequence (4)
	// They should get updates for seq 5 and 6 (rounds 5 and 6)
	updates, isFull := session.Reconnect("client1", lastSeenSeq, nil)

	// Should get incremental updates
	if isFull {
		t.Error("expected incremental updates, got full state")
	}

	// Should have 2 pending updates (rounds 5 and 6)
	if len(updates) != 2 {
		t.Errorf("expected 2 pending updates, got %d", len(updates))
	}

	// Verify client is reconnected
	if session.ClientCount() != 1 {
		t.Errorf("expected 1 client after reconnect, got %d", session.ClientCount())
	}
}

func TestReconnectionHistoryTooOld(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	// Enable small history (only 3 entries)
	session.SetHistorySize(3)

	session.Connect("client1", nil)

	// Initial sync
	state.SetRound(1)
	session.Tick()
	veryOldSeq := session.Seq()

	// Make many updates to overflow history
	for i := 2; i <= 10; i++ {
		state.SetRound(i)
		session.Tick()
	}

	session.Disconnect("client1")

	// Client tries to reconnect with very old sequence
	updates, isFull := session.Reconnect("client1", veryOldSeq, nil)

	// Should get full state because history is too old
	if !isFull {
		t.Error("expected full state, got incremental updates")
	}

	if len(updates) != 1 {
		t.Errorf("expected 1 full state update, got %d", len(updates))
	}

	// Check it's actually a full state message
	if updates[0][0] != MsgFullState {
		t.Errorf("expected MsgFullState, got %d", updates[0][0])
	}
}

func TestReconnectionUpToDate(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	session.SetHistorySize(10)
	session.Connect("client1", nil)

	state.SetRound(1)
	session.Tick()

	state.SetRound(2)
	session.Tick()
	currentSeq := session.Seq() - 1 // Last completed tick

	session.Disconnect("client1")

	// Client reconnects but is already up to date
	updates, isFull := session.Reconnect("client1", currentSeq, nil)

	if isFull {
		t.Error("expected no full state when up to date")
	}

	if updates != nil {
		t.Errorf("expected nil updates when up to date, got %d", len(updates))
	}
}

func TestSequenceNumbers(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	// Initial sequence should be 1
	if session.Seq() != 1 {
		t.Errorf("initial sequence should be 1, got %d", session.Seq())
	}

	session.Connect("client1", nil)
	state.SetRound(1)
	session.Tick()

	// Sequence should increment
	if session.Seq() != 2 {
		t.Errorf("sequence should be 2 after tick, got %d", session.Seq())
	}

	// TickWithSeq should return the pre-tick sequence
	state.SetRound(2)
	_, seq := session.TickWithSeq()
	if seq != 2 {
		t.Errorf("TickWithSeq should return 2, got %d", seq)
	}

	// Client acknowledges sequence
	session.AckSeq("client1", seq)
	if session.ClientSeq("client1") != seq {
		t.Errorf("client sequence should be %d, got %d", seq, session.ClientSeq("client1"))
	}
}

// Test filter and hooks

func TestTrackedSessionFilter(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	// Simple filter that verifies it's being called
	filterCalled := false
	filterPhase := func(s *TestGameState) *TestGameState {
		filterCalled = true
		// Return the original state but with phase modified
		// Keep the same changes pointer
		s.phase = "hidden"
		return s
	}

	// Connect admin (no filter) and player (with filter)
	session.Connect("admin", nil)
	session.Connect("player", filterPhase)

	// First tick sends full state for new clients
	state.SetRound(1)
	state.SetPhase("draw")
	diffs := session.Tick()

	// Both should get full state data
	if len(diffs) != 2 {
		t.Errorf("expected 2 diffs on first tick, got %d", len(diffs))
	}

	// Verify filter was called
	if !filterCalled {
		t.Error("filter should have been called during first Tick")
	}

	// Check both got full state
	for id, data := range diffs {
		if len(data) == 0 {
			t.Errorf("client %s got empty data", id)
		}
		if data[0] != MsgFullState {
			t.Errorf("client %s: expected MsgFullState (0x%02x), got 0x%02x", id, MsgFullState, data[0])
		}
	}
}

func TestTrackedSessionFilterWithExistingClient(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	// Connect clients
	session.Connect("admin", nil)
	session.Connect("player", nil)

	// First tick to clear "needs full" flag
	state.SetRound(1)
	session.Tick()

	// Now update player's filter
	filterCalled := false
	session.SetFilter("player", func(s *TestGameState) *TestGameState {
		filterCalled = true
		return s
	})

	// Second update
	state.SetRound(2)
	diffs := session.Tick()

	// Both should get patch data
	if len(diffs) != 2 {
		t.Errorf("expected 2 diffs, got %d", len(diffs))
	}

	// Verify filter was called
	if !filterCalled {
		t.Error("updated filter should have been called")
	}

	// Verify patches (not full state)
	for id, data := range diffs {
		if len(data) > 0 && data[0] == MsgFullState {
			t.Errorf("client %s: expected MsgPatch, got MsgFullState", id)
		}
	}
}

func TestTrackedSessionHooks(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	// Track hook calls
	var beforeFilterCalls []string
	var afterFilterCalls []string
	var beforeEncodeCalls []string
	var afterEncodeCalls []string
	var beforeBroadcastCalled bool
	var afterBroadcastCalled bool
	var afterBroadcastSeq uint64

	session.SetHooks(SessionHooks[*TestGameState, string]{
		OnBeforeFilter: func(clientID string, state *TestGameState) {
			beforeFilterCalls = append(beforeFilterCalls, clientID)
		},
		OnAfterFilter: func(clientID string, filtered *TestGameState) {
			afterFilterCalls = append(afterFilterCalls, clientID)
		},
		OnBeforeEncode: func(clientID string, state *TestGameState) {
			beforeEncodeCalls = append(beforeEncodeCalls, clientID)
		},
		OnAfterEncode: func(clientID string, data []byte) []byte {
			afterEncodeCalls = append(afterEncodeCalls, clientID)
			return data // Return unchanged
		},
		OnBeforeBroadcast: func(diffs map[string][]byte) map[string][]byte {
			beforeBroadcastCalled = true
			return diffs
		},
		OnAfterBroadcast: func(diffs map[string][]byte, seq uint64) {
			afterBroadcastCalled = true
			afterBroadcastSeq = seq
		},
	})

	// Connect clients
	session.Connect("client1", nil)
	session.Connect("client2", nil)

	// Update and tick
	state.SetRound(1)
	session.Tick()

	// Verify hooks were called
	if len(beforeFilterCalls) != 2 {
		t.Errorf("OnBeforeFilter should be called twice, got %d", len(beforeFilterCalls))
	}
	if len(afterFilterCalls) != 2 {
		t.Errorf("OnAfterFilter should be called twice, got %d", len(afterFilterCalls))
	}
	if len(beforeEncodeCalls) != 2 {
		t.Errorf("OnBeforeEncode should be called twice, got %d", len(beforeEncodeCalls))
	}
	if len(afterEncodeCalls) != 2 {
		t.Errorf("OnAfterEncode should be called twice, got %d", len(afterEncodeCalls))
	}
	if !beforeBroadcastCalled {
		t.Error("OnBeforeBroadcast should be called")
	}
	if !afterBroadcastCalled {
		t.Error("OnAfterBroadcast should be called")
	}
	if afterBroadcastSeq != 1 {
		t.Errorf("OnAfterBroadcast seq should be 1, got %d", afterBroadcastSeq)
	}
}

func TestTrackedSessionHooksModifyData(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	// Hook that adds a prefix to encoded data
	prefix := []byte{0xFF, 0xFE}
	session.SetHooks(SessionHooks[*TestGameState, string]{
		OnAfterEncode: func(clientID string, data []byte) []byte {
			result := make([]byte, len(prefix)+len(data))
			copy(result, prefix)
			copy(result[len(prefix):], data)
			return result
		},
	})

	session.Connect("client1", nil)
	state.SetRound(1)
	diffs := session.Tick()

	// Verify prefix was added
	data := diffs["client1"]
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xFE {
		t.Error("OnAfterEncode should have added prefix")
	}
}

func TestTrackedSessionHooksFilterClient(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	// Hook that filters out client2
	session.SetHooks(SessionHooks[*TestGameState, string]{
		OnBeforeBroadcast: func(diffs map[string][]byte) map[string][]byte {
			delete(diffs, "client2")
			return diffs
		},
	})

	session.Connect("client1", nil)
	session.Connect("client2", nil)
	state.SetRound(1)
	diffs := session.Tick()

	// client2 should be filtered out
	if _, ok := diffs["client2"]; ok {
		t.Error("client2 should be filtered out by OnBeforeBroadcast")
	}
	if _, ok := diffs["client1"]; !ok {
		t.Error("client1 should still receive data")
	}
}

func TestTrackedSessionHelperMethods(t *testing.T) {
	state := NewTestGameState()
	tracked := NewTrackedState[*TestGameState, string](state, nil)
	session := NewTrackedSession[*TestGameState, string, string](tracked)

	// HasClient
	if session.HasClient("client1") {
		t.Error("client1 should not exist yet")
	}

	session.Connect("client1", nil)

	if !session.HasClient("client1") {
		t.Error("client1 should exist")
	}

	// GetFilter should be nil
	if session.GetFilter("client1") != nil {
		t.Error("client1 filter should be nil")
	}

	// SetFilter
	filterCalled := false
	session.SetFilter("client1", func(s *TestGameState) *TestGameState {
		filterCalled = true
		return s
	})

	if session.GetFilter("client1") == nil {
		t.Error("client1 filter should not be nil after SetFilter")
	}

	// Trigger broadcast to verify filter is called
	state.SetRound(1)
	session.Tick()

	if !filterCalled {
		t.Error("filter should have been called during broadcast")
	}
}

// Benchmarks

func BenchmarkChangeSetMark(b *testing.B) {
	cs := NewChangeSet()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cs.Mark(uint8(i%256), OpReplace)
	}
}

func BenchmarkEncoderSingleField(b *testing.B) {
	registry := NewSchemaRegistry()
	state := NewTestGameState()
	registry.Register(state.Schema())
	encoder := NewEncoder(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.SetRound(i)
		encoder.Encode(state)
		state.ClearChanges()
	}
}

func BenchmarkEncoderMultipleFields(b *testing.B) {
	registry := NewSchemaRegistry()
	state := NewTestGameState()
	registry.Register(state.Schema())
	encoder := NewEncoder(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.SetRound(i)
		state.SetPhase("phase")
		state.SetScore("p1", i*10)
		encoder.Encode(state)
		state.ClearChanges()
	}
}

func BenchmarkEncoderFullState(b *testing.B) {
	registry := NewSchemaRegistry()
	state := NewTestGameState()
	registry.Register(state.Schema())

	// Add some data
	for i := 0; i < 10; i++ {
		state.AddPlayer(TestPlayer{
			ID:    string(rune('a' + i)),
			Name:  "Player",
			Score: i * 100,
		})
	}
	state.ClearChanges()

	encoder := NewEncoder(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.EncodeAll(state)
	}
}

func BenchmarkDecoderPatch(b *testing.B) {
	registry := NewSchemaRegistry()
	state := NewTestGameState()
	registry.Register(state.Schema())

	encoder := NewEncoder(registry)
	decoder := NewDecoder(registry)

	state.SetRound(42)
	state.SetPhase("play")
	data := encoder.Encode(state)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoder.Decode(data)
	}
}

func BenchmarkNewTrackedDiff(b *testing.B) {
	registry := NewSchemaRegistry()
	state := NewTestGameState()
	registry.Register(state.Schema())
	encoder := NewEncoder(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.SetRound(i)
		encoder.Encode(state)
		state.ClearChanges()
	}
}

func BenchmarkNewTrackedDiffLargeState(b *testing.B) {
	registry := NewSchemaRegistry()
	state := NewTestGameState()
	registry.Register(state.Schema())

	// Add 100 players
	for i := 0; i < 100; i++ {
		state.AddPlayer(TestPlayer{ID: string(rune('a' + i)), Name: "Player", Score: i})
		state.SetScore(string(rune('a'+i)), i*10)
	}
	state.ClearChanges()

	encoder := NewEncoder(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.SetRound(i) // Only change one field
		encoder.Encode(state)
		state.ClearChanges()
	}
}
