package statesync

import (
	"sync"
	"testing"
)

// BenchState is a realistic game state for benchmarking
type BenchState struct {
	mu      sync.RWMutex
	changes *ChangeSet
	schema  *Schema

	phase   string
	round   int64
	players map[string]BenchPlayer
	items   []BenchItem
	config  []byte
}

type BenchPlayer struct {
	changes *ChangeSet
	schema  *Schema

	id       string
	name     string
	score    int64
	position []byte
	isCaught bool
	teamId   string
}

type BenchItem struct {
	changes *ChangeSet
	schema  *Schema

	id       string
	position []byte
	isActive bool
}

func BenchPlayerSchema() *Schema {
	return NewSchemaBuilder("BenchPlayer").WithID(2).
		String("ID").String("Name").Int64("Score").
		Bytes("Position").Bool("IsCaught").String("TeamId").
		Build()
}

func BenchItemSchema() *Schema {
	return NewSchemaBuilder("BenchItem").WithID(3).
		String("ID").Bytes("Position").Bool("IsActive").
		Build()
}

func BenchStateSchema() *Schema {
	return NewSchemaBuilder("BenchState").WithID(1).
		String("Phase").Int64("Round").
		Map("Players", TypeStruct, BenchPlayerSchema()).
		Array("Items", TypeStruct, BenchItemSchema()).
		Bytes("Config").
		Build()
}

func NewBenchState() *BenchState {
	return &BenchState{
		changes: NewChangeSet(),
		schema:  BenchStateSchema(),
		phase:   "playing",
		round:   42,
		players: make(map[string]BenchPlayer),
		items:   make([]BenchItem, 0),
	}
}

func (s *BenchState) Schema() *Schema        { return s.schema }
func (s *BenchState) Changes() *ChangeSet     { return s.changes }
func (s *BenchState) ClearChanges()           { s.changes.Clear() }
func (s *BenchState) MarkAllDirty()           { s.changes.MarkAll(4) }

func (s *BenchState) GetFieldValue(index uint8) interface{} {
	switch index {
	case 0:
		return s.phase
	case 1:
		return s.round
	case 2:
		if s.players == nil {
			return nil
		}
		cp := make(map[string]BenchPlayer, len(s.players))
		for k, v := range s.players {
			cp[k] = v
		}
		return cp
	case 3:
		if s.items == nil {
			return nil
		}
		cp := make([]BenchItem, len(s.items))
		copy(cp, s.items)
		return cp
	case 4:
		return s.config
	}
	return nil
}

func (bp *BenchPlayer) Schema() *Schema        { return bp.schema }
func (bp *BenchPlayer) Changes() *ChangeSet     { return bp.changes }
func (bp *BenchPlayer) ClearChanges()           { bp.changes.Clear() }
func (bp *BenchPlayer) MarkAllDirty()           { bp.changes.MarkAll(5) }

func (bp *BenchPlayer) GetFieldValue(index uint8) interface{} {
	switch index {
	case 0:
		return bp.id
	case 1:
		return bp.name
	case 2:
		return bp.score
	case 3:
		return bp.position
	case 4:
		return bp.isCaught
	case 5:
		return bp.teamId
	}
	return nil
}

func (bi *BenchItem) Schema() *Schema        { return bi.schema }
func (bi *BenchItem) Changes() *ChangeSet     { return bi.changes }
func (bi *BenchItem) ClearChanges()           { bi.changes.Clear() }
func (bi *BenchItem) MarkAllDirty()           { bi.changes.MarkAll(2) }

func (bi *BenchItem) GetFieldValue(index uint8) interface{} {
	switch index {
	case 0:
		return bi.id
	case 1:
		return bi.position
	case 2:
		return bi.isActive
	}
	return nil
}

func (s *BenchState) ShallowClone() *BenchState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clone := &BenchState{
		changes: s.changes.CloneForFilter(),
		schema:  s.schema,
		phase:   s.phase,
		round:   s.round,
		config:  s.config,
	}
	if s.players != nil {
		clone.players = make(map[string]BenchPlayer, len(s.players))
		for k, v := range s.players {
			clone.players[k] = v
		}
	}
	if s.items != nil {
		clone.items = make([]BenchItem, len(s.items))
		copy(clone.items, s.items)
	}
	return clone
}

func populateBenchState(s *BenchState, nPlayers, nItems int) {
	posBytes := []byte(`{"lat":47.5,"lng":19.04,"t":1700000000}`)
	for i := 0; i < nPlayers; i++ {
		id := "p" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		p := BenchPlayer{
			changes:  NewChangeSet(),
			schema:   BenchPlayerSchema(),
			id:       id,
			name:     "Player " + id,
			score:    int64(i * 10),
			position: posBytes,
			teamId:   "team1",
		}
		p.MarkAllDirty()
		s.players[id] = p
	}
	for i := 0; i < nItems; i++ {
		id := "item" + string(rune('0'+i%10))
		item := BenchItem{
			changes:  NewChangeSet(),
			schema:   BenchItemSchema(),
			id:       id,
			position: posBytes,
			isActive: true,
		}
		item.MarkAllDirty()
		s.items = append(s.items, item)
	}
	s.MarkAllDirty()
	p0 := s.players["pA0"]
	s.changes.GetOrCreateMap(2).MarkReplace("pA0", &p0)
}

// --- Benchmarks ---

func BenchmarkEncodeAll_10Players(b *testing.B) {
	benchEncodeAll(b, 10, 5)
}

func BenchmarkEncodeAll_50Players(b *testing.B) {
	benchEncodeAll(b, 50, 20)
}

func BenchmarkEncodeAll_100Players(b *testing.B) {
	benchEncodeAll(b, 100, 50)
}

func benchEncodeAll(b *testing.B, nPlayers, nItems int) {
	s := NewBenchState()
	populateBenchState(s, nPlayers, nItems)
	enc := NewEncoder(nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		enc.Reset()
		enc.EncodeAll(s)
	}
}

func BenchmarkEncodePatch_SingleField(b *testing.B) {
	s := NewBenchState()
	populateBenchState(s, 10, 5)
	enc := NewEncoder(nil)

	// Only one field changed
	s.ClearChanges()
	s.changes.Mark(1, OpReplace) // round changed

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		enc.Reset()
		enc.Encode(s)
	}
}

func BenchmarkShallowClone_10Players(b *testing.B) {
	benchClone(b, 10, 5)
}

func BenchmarkShallowClone_50Players(b *testing.B) {
	benchClone(b, 50, 20)
}

func BenchmarkShallowClone_100Players(b *testing.B) {
	benchClone(b, 100, 50)
}

func benchClone(b *testing.B, nPlayers, nItems int) {
	s := NewBenchState()
	populateBenchState(s, nPlayers, nItems)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = s.ShallowClone()
	}
}

func BenchmarkChangeSetCloneForFilter(b *testing.B) {
	cs := NewChangeSet()
	cs.Mark(0, OpReplace)
	cs.Mark(3, OpReplace)
	m := cs.GetOrCreateMap(2)
	m.MarkReplace("key1", "val1")
	m.MarkReplace("key2", "val2")
	m.MarkReplace("key3", "val3")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cs.CloneForFilter()
	}
}

func BenchmarkBroadcast_10Players_NoFilter(b *testing.B) {
	benchBroadcast(b, 10, false)
}

func BenchmarkBroadcast_50Players_NoFilter(b *testing.B) {
	benchBroadcast(b, 50, false)
}

func BenchmarkBroadcast_10Players_WithFilter(b *testing.B) {
	benchBroadcast(b, 10, true)
}

func BenchmarkBroadcast_50Players_WithFilter(b *testing.B) {
	benchBroadcast(b, 50, true)
}

func benchBroadcast(b *testing.B, nClients int, withFilter bool) {
	s := NewBenchState()
	populateBenchState(s, 10, 5)

	ts := NewTrackedState[*BenchState, any](s, nil)
	session := NewTrackedSession[*BenchState, any, string](ts)

	var filter FilterFunc[*BenchState]
	if withFilter {
		filter = func(state *BenchState) *BenchState {
			clone := state.ShallowClone()
			// Simulate hiding some players
			delete(clone.players, "pA0")
			return clone
		}
	}

	for i := 0; i < nClients; i++ {
		id := "c" + string(rune('0'+i%10)) + string(rune('A'+i/10))
		session.Connect(id, filter)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Simulate a state change
		ts.UpdateInPlace(func(state *BenchState) {
			state.round++
			state.changes.Mark(1, OpReplace)
		})
		session.Tick()
	}
}

func BenchmarkEncoderPool_Parallel(b *testing.B) {
	s := NewBenchState()
	populateBenchState(s, 10, 5)

	ts := NewTrackedState[*BenchState, any](s, nil)
	ts.UpdateInPlace(func(state *BenchState) {
		state.changes.Mark(0, OpReplace)
	})

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = ts.Encode()
		}
	})
}
