package statesync

import (
	"encoding/json"
	"testing"
	"time"
)

// Test state for replay tests
type ReplayTestState struct {
	changes  *ChangeSet
	Score    int64
	Name     string
	Items    []string
	Messages []ReplayMessage
}

type ReplayMessage struct {
	changes *ChangeSet
	ID      string
	Text    string
	Status  string
}

func NewReplayTestState() *ReplayTestState {
	return &ReplayTestState{
		changes: NewChangeSet(),
		Items:   make([]string, 0),
	}
}

func (s *ReplayTestState) Changes() *ChangeSet { return s.changes }
func (s *ReplayTestState) ClearChanges()       { s.changes.Clear() }
func (s *ReplayTestState) MarkAllDirty()       { s.changes.MarkAll(3) }
func (s *ReplayTestState) Schema() *Schema {
	return &Schema{
		ID:   100,
		Name: "ReplayTestState",
		Fields: []FieldMeta{
			{Index: 0, Name: "Score", Type: TypeInt64},
			{Index: 1, Name: "Name", Type: TypeString},
			{Index: 2, Name: "Items", Type: TypeArray, ElemType: TypeString},
			{Index: 3, Name: "Messages", Type: TypeArray, ElemType: TypeStruct},
		},
	}
}

func (s *ReplayTestState) SetScore(v int64) {
	s.Score = v
	s.changes.Mark(0, OpReplace)
}

func (s *ReplayTestState) SetName(v string) {
	s.Name = v
	s.changes.Mark(1, OpReplace)
}

func (s *ReplayTestState) GetFieldValue(index uint8) interface{} {
	switch index {
	case 0:
		return s.Score
	case 1:
		return s.Name
	case 2:
		return s.Items
	case 3:
		return s.Messages
	default:
		return nil
	}
}

func TestDiffRecorder_Basic(t *testing.T) {
	recorder := NewDiffRecorder()

	// Record some diffs
	recorder.SetSource("server")
	recorder.SetTick(1)
	recorder.Record(1, []byte{1, 2, 3}, nil, 100*time.Millisecond)

	recorder.SetSource("player:p1")
	recorder.SetTick(2)
	recorder.Record(2, []byte{4, 5, 6}, nil, 50*time.Millisecond)

	records := recorder.Records()
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Check first record
	if records[0].Seq != 1 {
		t.Errorf("expected seq 1, got %d", records[0].Seq)
	}
	if records[0].Source != "server" {
		t.Errorf("expected source 'server', got %s", records[0].Source)
	}
	if records[0].Tick != 1 {
		t.Errorf("expected tick 1, got %d", records[0].Tick)
	}

	// Check second record
	if records[1].Seq != 2 {
		t.Errorf("expected seq 2, got %d", records[1].Seq)
	}
	if records[1].Source != "player:p1" {
		t.Errorf("expected source 'player:p1', got %s", records[1].Source)
	}
}

func TestDiffRecorder_Drain(t *testing.T) {
	recorder := NewDiffRecorder()
	recorder.Record(1, []byte{1}, nil, 0)
	recorder.Record(2, []byte{2}, nil, 0)

	records := recorder.Drain()
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// After drain, records should be empty
	if len(recorder.Records()) != 0 {
		t.Error("expected empty records after drain")
	}
}

func TestDiffRecorder_SkipsEmptyDiffs(t *testing.T) {
	recorder := NewDiffRecorder()
	recorder.Record(1, nil, nil, 0)
	recorder.Record(2, []byte{}, nil, 0)
	recorder.Record(3, []byte{1, 2, 3}, nil, 0)

	records := recorder.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 record (empty diffs skipped), got %d", len(records))
	}
	if records[0].Seq != 3 {
		t.Errorf("expected seq 3, got %d", records[0].Seq)
	}
}

func TestDiffRecord_Serialization(t *testing.T) {
	records := []DiffRecord{
		{
			Seq:       1,
			Tick:      10,
			Timestamp: time.Now(),
			Source:    "server",
			Data:      []byte{1, 2, 3, 4, 5},
			DeltaNs:   100000000,
		},
		{
			Seq:       2,
			Tick:      11,
			Timestamp: time.Now(),
			Source:    "player:p1",
			Data:      []byte{6, 7, 8},
			Events:    []Event{{Type: "test", Payload: []byte("payload")}},
			DeltaNs:   50000000,
		},
	}

	// Marshal
	data, err := MarshalRecords(records)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Unmarshal
	decoded, err := UnmarshalRecords(data)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(decoded) != 2 {
		t.Fatalf("expected 2 records, got %d", len(decoded))
	}

	// Check first record
	if decoded[0].Seq != 1 || decoded[0].Tick != 10 || decoded[0].Source != "server" {
		t.Error("first record mismatch")
	}
	if string(decoded[0].Data) != string(records[0].Data) {
		t.Error("first record data mismatch")
	}

	// Check second record
	if decoded[1].Seq != 2 || len(decoded[1].Events) != 1 {
		t.Error("second record mismatch")
	}
}

func TestAsyncRequest_Lifecycle(t *testing.T) {
	// Create request
	req, err := NewAsyncRequest("req-1", "chatgpt", map[string]string{"prompt": "Hello"})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	if req.ID != "req-1" {
		t.Errorf("expected ID 'req-1', got %s", req.ID)
	}
	if req.Type != "chatgpt" {
		t.Errorf("expected type 'chatgpt', got %s", req.Type)
	}
	if req.Status != AsyncPending {
		t.Errorf("expected status 'pending', got %s", req.Status)
	}

	// Get request payload
	var reqPayload map[string]string
	if err := req.GetRequest(&reqPayload); err != nil {
		t.Fatalf("get request error: %v", err)
	}
	if reqPayload["prompt"] != "Hello" {
		t.Errorf("expected prompt 'Hello', got %s", reqPayload["prompt"])
	}

	// Complete request
	if err := req.Complete(map[string]string{"response": "Hi there!"}); err != nil {
		t.Fatalf("complete error: %v", err)
	}

	if req.Status != AsyncCompleted {
		t.Errorf("expected status 'completed', got %s", req.Status)
	}

	// Get response payload
	var respPayload map[string]string
	if err := req.GetResponse(&respPayload); err != nil {
		t.Fatalf("get response error: %v", err)
	}
	if respPayload["response"] != "Hi there!" {
		t.Errorf("expected response 'Hi there!', got %s", respPayload["response"])
	}
}

func TestAsyncRequest_Failure(t *testing.T) {
	req, _ := NewAsyncRequest("req-2", "http", nil)

	// Fail request
	req.Fail(ErrUnknownSchema)

	if req.Status != AsyncFailed {
		t.Errorf("expected status 'failed', got %s", req.Status)
	}
	if req.Error != ErrUnknownSchema.Error() {
		t.Errorf("expected error message, got %s", req.Error)
	}
}

func TestExternalInjector_Basic(t *testing.T) {
	state := NewReplayTestState()
	tracked := NewTrackedState[*ReplayTestState, any](state, nil)
	session := NewTrackedSession[*ReplayTestState, any, string](tracked)

	// Create injector
	injector := NewExternalInjector(session, "external:chatgpt")

	if injector.Source() != "external:chatgpt" {
		t.Errorf("expected source 'external:chatgpt', got %s", injector.Source())
	}

	// Inject data
	diffs := injector.Inject(func(s **ReplayTestState) {
		(*s).SetScore(100)
		(*s).SetName("Injected")
	})

	// Check state was updated
	if state.Score != 100 {
		t.Errorf("expected score 100, got %d", state.Score)
	}
	if state.Name != "Injected" {
		t.Errorf("expected name 'Injected', got %s", state.Name)
	}

	// Diffs should be non-empty (state was changed and broadcast)
	// Note: diffs is empty if no clients connected
	_ = diffs
}

func TestExternalInjector_WithEvent(t *testing.T) {
	state := NewReplayTestState()
	tracked := NewTrackedState[*ReplayTestState, any](state, nil)
	session := NewTrackedSession[*ReplayTestState, any, string](tracked)

	// Connect a client to receive events
	session.Connect("client1", nil)

	injector := NewExternalInjector(session, "external:api")

	// Inject with event
	diffs, err := injector.InjectWithEvent(
		func(s **ReplayTestState) {
			(*s).SetScore(42)
		},
		"DataReceived",
		map[string]string{"source": "api"},
	)
	if err != nil {
		t.Fatalf("inject error: %v", err)
	}

	// Should have diffs for the connected client
	if len(diffs) == 0 {
		t.Error("expected diffs for connected client")
	}
}

func TestMapReplayer_Basic(t *testing.T) {
	// Create schema registry
	registry := NewSchemaRegistry()
	schema := &Schema{
		ID:   1,
		Name: "TestState",
		Fields: []FieldMeta{
			{Index: 0, Name: "Score", Type: TypeInt64},
			{Index: 1, Name: "Name", Type: TypeString},
		},
	}
	registry.Register(schema)

	// Create encoder to generate test data
	encoder := NewEncoder(registry)

	// Create a simple state and encode full state
	testState := &simpleTestState{
		changes: NewChangeSet(),
		Score:   100,
		Name:    "Player1",
	}
	testState.changes.MarkAll(1) // Mark all fields dirty
	fullData := encoder.EncodeAll(testState)

	// Create replayer
	replayer := NewMapReplayer(registry)

	// Create a diff record
	record := DiffRecord{
		Seq:     1,
		Tick:    1,
		Source:  "server",
		Data:    fullData,
		DeltaNs: int64(100 * time.Millisecond),
	}

	// Replay
	delta, err := replayer.Replay(record)
	if err != nil {
		t.Fatalf("replay error: %v", err)
	}

	if delta != 100*time.Millisecond {
		t.Errorf("expected delta 100ms, got %v", delta)
	}

	// Check state
	state := replayer.State()
	if score, ok := state["Score"].(int64); !ok || score != 100 {
		t.Errorf("expected Score 100, got %v", state["Score"])
	}
	if name, ok := state["Name"].(string); !ok || name != "Player1" {
		t.Errorf("expected Name 'Player1', got %v", state["Name"])
	}
}

// Simple test state for encoding
type simpleTestState struct {
	changes *ChangeSet
	Score   int64
	Name    string
}

func (s *simpleTestState) Changes() *ChangeSet { return s.changes }
func (s *simpleTestState) ClearChanges()       { s.changes.Clear() }
func (s *simpleTestState) MarkAllDirty()       { s.changes.MarkAll(1) }
func (s *simpleTestState) Schema() *Schema {
	return &Schema{
		ID:   1,
		Name: "TestState",
		Fields: []FieldMeta{
			{Index: 0, Name: "Score", Type: TypeInt64},
			{Index: 1, Name: "Name", Type: TypeString},
		},
	}
}

func (s *simpleTestState) GetFieldValue(index uint8) interface{} {
	switch index {
	case 0:
		return s.Score
	case 1:
		return s.Name
	default:
		return nil
	}
}

func TestRecordingHooks_Integration(t *testing.T) {
	state := NewReplayTestState()
	tracked := NewTrackedState[*ReplayTestState, any](state, nil)
	session := NewTrackedSession[*ReplayTestState, any, string](tracked)

	// Setup recording
	recorder := NewDiffRecorder()
	session.SetHooks(RecordingHooks[*ReplayTestState, string](recorder))

	// Connect a client
	session.Connect("client1", nil)

	// Make some changes
	state.SetScore(100)
	state.SetName("Test")
	session.Tick()

	recorder.SetTick(2)
	state.SetScore(200)
	session.Tick()

	// Check recorded diffs
	records := recorder.Records()
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

func TestAsyncRequest_JSONRoundTrip(t *testing.T) {
	req, _ := NewAsyncRequest("req-json", "test", map[string]int{"count": 42})
	req.Complete(map[string]bool{"success": true})

	// Marshal to JSON
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Unmarshal
	var decoded AsyncRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ID != "req-json" {
		t.Errorf("expected ID 'req-json', got %s", decoded.ID)
	}
	if decoded.Status != AsyncCompleted {
		t.Errorf("expected status 'completed', got %s", decoded.Status)
	}

	var resp map[string]bool
	if err := decoded.GetResponse(&resp); err != nil {
		t.Fatalf("get response error: %v", err)
	}
	if !resp["success"] {
		t.Error("expected success=true")
	}
}

// Test realistic ChatGPT API flow
func TestChatGPTFlow_Simulation(t *testing.T) {
	// This simulates how a ChatGPT API integration would work:
	// 1. Game creates a pending request in schema
	// 2. External system (multiplayer framework) sees pending request
	// 3. External system calls ChatGPT API
	// 4. External system injects response back into schema
	// 5. Game rule triggers on status change

	// Step 1: Create request
	req, err := NewAsyncRequest("chat-1", "chatgpt", map[string]string{
		"prompt": "What's the best move?",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify initial state
	if req.Status != AsyncPending {
		t.Errorf("expected pending, got %s", req.Status)
	}

	// Step 3 & 4: Simulate API response (in real code, this would be in multiplayer framework)
	// The response is injected into schema, creating a diff
	if err := req.Complete(map[string]string{
		"response": "Move your knight to E5",
		"model":    "gpt-4",
	}); err != nil {
		t.Fatal(err)
	}

	// Step 5: Game can now read the response
	if req.Status != AsyncCompleted {
		t.Errorf("expected completed, got %s", req.Status)
	}

	var response map[string]string
	if err := req.GetResponse(&response); err != nil {
		t.Fatal(err)
	}

	if response["response"] != "Move your knight to E5" {
		t.Errorf("unexpected response: %s", response["response"])
	}

	// The key insight: This entire flow goes through the schema,
	// creating diffs that can be saved and replayed.
	// On replay, we don't call ChatGPT again - we just replay the diffs.
}
