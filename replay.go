package statesync

import (
	"encoding/json"
	"time"
)

// DiffRecord represents a single state change that can be persisted and replayed.
// All state changes go through the schema, making them replayable.
type DiffRecord struct {
	// Sequence number for ordering
	Seq uint64 `json:"seq"`

	// Tick number when this diff was created
	Tick uint64 `json:"tick"`

	// Timestamp when the diff was created
	Timestamp time.Time `json:"ts"`

	// Source identifies who/what created this change
	// "server" = server-side rule, "player:ID" = player action, "external:type" = external system
	Source string `json:"source"`

	// The encoded diff data (binary format from Encode())
	Data []byte `json:"data"`

	// Optional: Events that were emitted with this diff
	Events []Event `json:"events,omitempty"`

	// Optional: Delta time for deterministic replay (in nanoseconds)
	DeltaNs int64 `json:"deltaNs,omitempty"`
}

// DiffRecorder captures diffs for persistence.
// Attach this to TrackedSession.SetHooks() to capture all state changes.
type DiffRecorder struct {
	records []DiffRecord
	source  string
	tick    uint64
}

// NewDiffRecorder creates a new diff recorder
func NewDiffRecorder() *DiffRecorder {
	return &DiffRecorder{
		source: "server",
	}
}

// SetSource sets the source identifier for subsequent records
func (dr *DiffRecorder) SetSource(source string) {
	dr.source = source
}

// SetTick sets the current tick for subsequent records
func (dr *DiffRecorder) SetTick(tick uint64) {
	dr.tick = tick
}

// Record captures a diff with the current source and tick
func (dr *DiffRecorder) Record(seq uint64, data []byte, events []Event, delta time.Duration) {
	if len(data) == 0 {
		return // Skip empty diffs
	}

	record := DiffRecord{
		Seq:       seq,
		Tick:      dr.tick,
		Timestamp: time.Now(),
		Source:    dr.source,
		Data:      make([]byte, len(data)),
		DeltaNs:   delta.Nanoseconds(),
	}
	copy(record.Data, data)

	if len(events) > 0 {
		record.Events = make([]Event, len(events))
		copy(record.Events, events)
	}

	dr.records = append(dr.records, record)
}

// Records returns all captured records
func (dr *DiffRecorder) Records() []DiffRecord {
	return dr.records
}

// Clear removes all captured records
func (dr *DiffRecorder) Clear() {
	dr.records = dr.records[:0]
}

// Drain returns all records and clears the buffer
func (dr *DiffRecorder) Drain() []DiffRecord {
	records := dr.records
	dr.records = nil
	return records
}

// MarshalRecords serializes records to JSON for storage
func MarshalRecords(records []DiffRecord) ([]byte, error) {
	return json.Marshal(records)
}

// UnmarshalRecords deserializes records from JSON
func UnmarshalRecords(data []byte) ([]DiffRecord, error) {
	var records []DiffRecord
	err := json.Unmarshal(data, &records)
	return records, err
}

// MapReplayer replays recorded diffs on a map-based state.
// Use this for server-side replay, time-travel debugging, or analysis.
// For client-side replay, clients decode diffs using their native decoders.
type MapReplayer struct {
	state    map[string]interface{}
	registry *SchemaRegistry
	decoder  *Decoder
}

// NewMapReplayer creates a replayer for map-based state
func NewMapReplayer(registry *SchemaRegistry) *MapReplayer {
	return &MapReplayer{
		state:    make(map[string]interface{}),
		registry: registry,
		decoder:  NewDecoder(registry),
	}
}

// State returns the current replayed state
func (mr *MapReplayer) State() map[string]interface{} {
	return mr.state
}

// Reset clears the state for a fresh replay
func (mr *MapReplayer) Reset() {
	mr.state = make(map[string]interface{})
}

// Replay applies a single diff record to the state.
// Returns the delta time for deterministic timing.
func (mr *MapReplayer) Replay(record DiffRecord) (time.Duration, error) {
	patch, err := mr.decoder.Decode(record.Data)
	if err != nil {
		return 0, err
	}

	schema := mr.registry.Get(patch.SchemaID)
	if schema == nil {
		return 0, ErrUnknownSchema
	}

	if err := ApplyPatch(mr.state, patch, schema); err != nil {
		return 0, err
	}

	return time.Duration(record.DeltaNs), nil
}

// ReplayAll applies all records in sequence
func (mr *MapReplayer) ReplayAll(records []DiffRecord) error {
	for _, record := range records {
		if _, err := mr.Replay(record); err != nil {
			return err
		}
	}
	return nil
}

// ReplayRange replays records within a sequence range [fromSeq, toSeq]
func (mr *MapReplayer) ReplayRange(records []DiffRecord, fromSeq, toSeq uint64) error {
	for _, record := range records {
		if record.Seq < fromSeq {
			continue
		}
		if record.Seq > toSeq {
			break
		}
		if _, err := mr.Replay(record); err != nil {
			return err
		}
	}
	return nil
}

// ExternalInjector provides a way to inject data from external sources (like API responses)
// into the schema. All injected data goes through the normal diff tracking.
type ExternalInjector[T Trackable, A any, ID comparable] struct {
	session *TrackedSession[T, A, ID]
	source  string
}

// NewExternalInjector creates an injector for external data sources
func NewExternalInjector[T Trackable, A any, ID comparable](session *TrackedSession[T, A, ID], source string) *ExternalInjector[T, A, ID] {
	return &ExternalInjector[T, A, ID]{
		session: session,
		source:  source,
	}
}

// Inject applies external data to the state.
// The updateFn should modify the state to incorporate the external data.
// Returns the diffs that were created (for persistence).
//
// Example:
//
//	injector := NewExternalInjector(session, "external:chatgpt")
//	injector.Inject(func(state *GameState) {
//	    state.AIResponses = append(state.AIResponses, AIResponse{
//	        RequestID: requestID,
//	        Response:  chatGPTResponse,
//	        Status:    "completed",
//	    })
//	})
func (ei *ExternalInjector[T, A, ID]) Inject(updateFn func(*T)) map[ID][]byte {
	ei.session.State().Update(updateFn)
	return ei.session.Tick()
}

// InjectWithEvent applies external data and emits an event.
// This is useful for triggering rules that wait for external data.
func (ei *ExternalInjector[T, A, ID]) InjectWithEvent(updateFn func(*T), eventType string, payload any) (map[ID][]byte, error) {
	ei.session.State().Update(updateFn)
	if err := ei.session.Emit(eventType, payload); err != nil {
		return nil, err
	}
	return ei.session.Tick(), nil
}

// Source returns the source identifier for this injector
func (ei *ExternalInjector[T, A, ID]) Source() string {
	return ei.source
}

// RecordingHooks creates SessionHooks that record all diffs to a DiffRecorder.
// Use this to capture diffs for persistence.
//
// Example:
//
//	recorder := NewDiffRecorder()
//	session.SetHooks(RecordingHooks[GameState, string](recorder))
//
//	// ... game runs ...
//
//	// Save records to database
//	records := recorder.Drain()
//	saveToDatabase(records)
func RecordingHooks[T Trackable, ID comparable](recorder *DiffRecorder) SessionHooks[T, ID] {
	return SessionHooks[T, ID]{
		OnAfterBroadcast: func(diffs map[ID][]byte, seq uint64) {
			// Capture the base diff (we need to get it from one client or encode separately)
			// For simplicity, we record the first non-nil diff
			for _, data := range diffs {
				if len(data) > 0 {
					recorder.Record(seq, data, nil, 0)
					break
				}
			}
		},
	}
}

// AsyncRequestStatus represents the status of an async external request
type AsyncRequestStatus string

const (
	AsyncPending   AsyncRequestStatus = "pending"
	AsyncCompleted AsyncRequestStatus = "completed"
	AsyncFailed    AsyncRequestStatus = "failed"
)

// AsyncRequest represents a pending external request stored in the schema.
// Store these in your schema to track external operations.
type AsyncRequest struct {
	// Unique ID for this request
	ID string `json:"id"`

	// Type of request (e.g., "chatgpt", "http", "db")
	Type string `json:"type"`

	// Current status
	Status AsyncRequestStatus `json:"status"`

	// Request payload (serialized)
	Request json.RawMessage `json:"request,omitempty"`

	// Response payload (serialized, populated when completed)
	Response json.RawMessage `json:"response,omitempty"`

	// Error message (populated when failed)
	Error string `json:"error,omitempty"`

	// Timestamps
	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
}

// NewAsyncRequest creates a new pending async request
func NewAsyncRequest(id, reqType string, request any) (*AsyncRequest, error) {
	var reqData json.RawMessage
	if request != nil {
		var err error
		reqData, err = json.Marshal(request)
		if err != nil {
			return nil, err
		}
	}

	return &AsyncRequest{
		ID:        id,
		Type:      reqType,
		Status:    AsyncPending,
		Request:   reqData,
		CreatedAt: time.Now(),
	}, nil
}

// Complete marks the request as completed with a response
func (ar *AsyncRequest) Complete(response any) error {
	if response != nil {
		data, err := json.Marshal(response)
		if err != nil {
			return err
		}
		ar.Response = data
	}
	ar.Status = AsyncCompleted
	ar.CompletedAt = time.Now()
	return nil
}

// Fail marks the request as failed with an error
func (ar *AsyncRequest) Fail(err error) {
	ar.Status = AsyncFailed
	ar.Error = err.Error()
	ar.CompletedAt = time.Now()
}

// GetRequest unmarshals the request payload
func (ar *AsyncRequest) GetRequest(v any) error {
	if len(ar.Request) == 0 {
		return nil
	}
	return json.Unmarshal(ar.Request, v)
}

// GetResponse unmarshals the response payload
func (ar *AsyncRequest) GetResponse(v any) error {
	if len(ar.Response) == 0 {
		return nil
	}
	return json.Unmarshal(ar.Response, v)
}
