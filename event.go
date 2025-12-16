package statesync

import (
	"encoding/binary"
	"math"
	"sync"
	"sync/atomic"
)

// Event represents a one-time message sent to clients.
// Unlike state updates (which are diffs), events are discrete messages
// that don't persist in state - they're fire-and-forget notifications.
//
// Common use cases:
//   - Game events: "CardPlayed", "PlayerJoined", "RoundStarted"
//   - UI triggers: "ShowAnimation", "PlaySound", "Toast"
//   - Errors/warnings that don't affect state
type Event struct {
	Type    string // Event type identifier (e.g., "CardPlayed")
	Payload []byte // Binary-encoded payload (use EncodeEventPayload helper)
}

// EventTarget specifies who receives an event
type EventTarget uint8

const (
	// TargetAll sends to all connected clients
	TargetAll EventTarget = iota
	// TargetOne sends to a specific client
	TargetOne
	// TargetExcept sends to all except a specific client
	TargetExcept
	// TargetMany sends to multiple specific clients
	TargetMany
)

// PendingEvent is an event waiting to be broadcast
type PendingEvent[ID comparable] struct {
	Event  Event
	Target EventTarget
	To     ID   // For TargetOne
	Except ID   // For TargetExcept
	ToMany []ID // For TargetMany
}

// EventBuffer collects events between Tick() calls.
// Optimized for low-allocation operation with atomic counter.
type EventBuffer[ID comparable] struct {
	mu     sync.Mutex
	events []PendingEvent[ID]
	swap   []PendingEvent[ID] // Pre-allocated swap buffer
	count  atomic.Int32       // Lock-free count for HasEvents check
}

// NewEventBuffer creates a new event buffer
func NewEventBuffer[ID comparable]() *EventBuffer[ID] {
	return &EventBuffer[ID]{
		events: make([]PendingEvent[ID], 0, 8),
		swap:   make([]PendingEvent[ID], 0, 8),
	}
}

// Add adds an event to the buffer
func (eb *EventBuffer[ID]) Add(event PendingEvent[ID]) {
	eb.mu.Lock()
	eb.events = append(eb.events, event)
	eb.count.Store(int32(len(eb.events)))
	eb.mu.Unlock()
}

// Drain returns all pending events and clears the buffer.
// Uses swap buffer to minimize allocations.
func (eb *EventBuffer[ID]) Drain() []PendingEvent[ID] {
	// Fast path: check atomic counter first (no lock)
	if eb.count.Load() == 0 {
		return nil
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	if len(eb.events) == 0 {
		return nil
	}

	// Swap buffers instead of allocating
	events := eb.events
	eb.events = eb.swap[:0]
	eb.swap = events[:0] // Will be reused next Drain
	eb.count.Store(0)
	return events
}

// Count returns the number of pending events (lock-free)
func (eb *EventBuffer[ID]) Count() int {
	return int(eb.count.Load())
}

// HasEvents returns true if there are pending events (lock-free)
func (eb *EventBuffer[ID]) HasEvents() bool {
	return eb.count.Load() > 0
}

// Clear removes all pending events without returning them
func (eb *EventBuffer[ID]) Clear() {
	eb.mu.Lock()
	eb.events = eb.events[:0]
	eb.count.Store(0)
	eb.mu.Unlock()
}

// EventEmitter is the interface for emitting events
type EventEmitter[ID comparable] interface {
	// Emit sends an event to all connected clients
	Emit(eventType string, payload any) error

	// EmitTo sends an event to a specific client
	EmitTo(clientID ID, eventType string, payload any) error

	// EmitExcept sends an event to all clients except one
	EmitExcept(exceptID ID, eventType string, payload any) error

	// EmitToMany sends an event to multiple specific clients
	EmitToMany(clientIDs []ID, eventType string, payload any) error

	// EmitRaw sends a pre-encoded event to all clients
	EmitRaw(event Event) error

	// EmitRawTo sends a pre-encoded event to a specific client
	EmitRawTo(clientID ID, event Event) error
}

// Binary protocol for events
const (
	// MsgEvent is the message type for events
	MsgEvent uint8 = 0x10

	// MsgEventBatch is for multiple events in one message
	MsgEventBatch uint8 = 0x11
)

// EncodeEvent encodes an event to binary format
func EncodeEvent(event Event) []byte {
	// Format: [MsgEvent][typeLen:varint][type:bytes][payloadLen:varint][payload:bytes]
	typeBytes := []byte(event.Type)
	size := 1 + varIntSize(uint64(len(typeBytes))) + len(typeBytes) +
		varIntSize(uint64(len(event.Payload))) + len(event.Payload)

	buf := make([]byte, size)
	pos := 0

	// Message type
	buf[pos] = MsgEvent
	pos++

	// Event type string
	pos += putVarUint(buf[pos:], uint64(len(typeBytes)))
	copy(buf[pos:], typeBytes)
	pos += len(typeBytes)

	// Payload
	pos += putVarUint(buf[pos:], uint64(len(event.Payload)))
	copy(buf[pos:], event.Payload)

	return buf
}

// EncodeEventBatch encodes multiple events into a single message
func EncodeEventBatch(events []Event) []byte {
	if len(events) == 0 {
		return nil
	}
	if len(events) == 1 {
		return EncodeEvent(events[0])
	}

	// Calculate total size
	size := 1 + varIntSize(uint64(len(events))) // header + count
	for _, e := range events {
		typeBytes := []byte(e.Type)
		size += varIntSize(uint64(len(typeBytes))) + len(typeBytes)
		size += varIntSize(uint64(len(e.Payload))) + len(e.Payload)
	}

	buf := make([]byte, size)
	pos := 0

	// Message type
	buf[pos] = MsgEventBatch
	pos++

	// Event count
	pos += putVarUint(buf[pos:], uint64(len(events)))

	// Each event
	for _, e := range events {
		typeBytes := []byte(e.Type)
		pos += putVarUint(buf[pos:], uint64(len(typeBytes)))
		copy(buf[pos:], typeBytes)
		pos += len(typeBytes)

		pos += putVarUint(buf[pos:], uint64(len(e.Payload)))
		copy(buf[pos:], e.Payload)
		pos += len(e.Payload)
	}

	return buf
}

// DecodeEvent decodes a single event from binary format
func DecodeEvent(data []byte) (Event, error) {
	if len(data) < 2 || data[0] != MsgEvent {
		return Event{}, ErrInvalidEventFormat
	}

	pos := 1

	// Event type
	typeLen, n := readVarUint(data[pos:])
	pos += n
	if pos+int(typeLen) > len(data) {
		return Event{}, ErrInvalidEventFormat
	}
	eventType := string(data[pos : pos+int(typeLen)])
	pos += int(typeLen)

	// Payload
	payloadLen, n := readVarUint(data[pos:])
	pos += n
	if pos+int(payloadLen) > len(data) {
		return Event{}, ErrInvalidEventFormat
	}
	payload := data[pos : pos+int(payloadLen)]

	return Event{Type: eventType, Payload: payload}, nil
}

// DecodeEventBatch decodes multiple events from binary format
func DecodeEventBatch(data []byte) ([]Event, error) {
	if len(data) < 2 {
		return nil, ErrInvalidEventFormat
	}

	// Single event
	if data[0] == MsgEvent {
		e, err := DecodeEvent(data)
		if err != nil {
			return nil, err
		}
		return []Event{e}, nil
	}

	// Batch
	if data[0] != MsgEventBatch {
		return nil, ErrInvalidEventFormat
	}

	pos := 1
	count, n := readVarUint(data[pos:])
	pos += n

	events := make([]Event, 0, count)
	for i := uint64(0); i < count; i++ {
		// Event type
		typeLen, n := readVarUint(data[pos:])
		pos += n
		if pos+int(typeLen) > len(data) {
			return nil, ErrInvalidEventFormat
		}
		eventType := string(data[pos : pos+int(typeLen)])
		pos += int(typeLen)

		// Payload
		payloadLen, n := readVarUint(data[pos:])
		pos += n
		if pos+int(payloadLen) > len(data) {
			return nil, ErrInvalidEventFormat
		}
		payload := make([]byte, payloadLen)
		copy(payload, data[pos:pos+int(payloadLen)])
		pos += int(payloadLen)

		events = append(events, Event{Type: eventType, Payload: payload})
	}

	return events, nil
}

// EventPayloadEncoder helps encode event payloads
type EventPayloadEncoder struct {
	buf []byte
	pos int
}

// NewEventPayloadEncoder creates a new payload encoder
func NewEventPayloadEncoder() *EventPayloadEncoder {
	return &EventPayloadEncoder{
		buf: make([]byte, 256),
	}
}

// Reset resets the encoder for reuse
func (e *EventPayloadEncoder) Reset() {
	e.pos = 0
}

// Bytes returns the encoded bytes
func (e *EventPayloadEncoder) Bytes() []byte {
	return e.buf[:e.pos]
}

func (e *EventPayloadEncoder) grow(n int) {
	needed := e.pos + n
	if needed <= len(e.buf) {
		return
	}
	newBuf := make([]byte, needed*2)
	copy(newBuf, e.buf[:e.pos])
	e.buf = newBuf
}

// WriteString writes a string to the payload
func (e *EventPayloadEncoder) WriteString(s string) {
	e.grow(len(s) + 10)
	e.pos += putVarUint(e.buf[e.pos:], uint64(len(s)))
	copy(e.buf[e.pos:], s)
	e.pos += len(s)
}

// WriteInt64 writes an int64 to the payload
func (e *EventPayloadEncoder) WriteInt64(v int64) {
	e.grow(8)
	binary.LittleEndian.PutUint64(e.buf[e.pos:], uint64(v))
	e.pos += 8
}

// WriteInt32 writes an int32 to the payload
func (e *EventPayloadEncoder) WriteInt32(v int32) {
	e.grow(4)
	binary.LittleEndian.PutUint32(e.buf[e.pos:], uint32(v))
	e.pos += 4
}

// WriteFloat64 writes a float64 to the payload
func (e *EventPayloadEncoder) WriteFloat64(v float64) {
	e.grow(8)
	binary.LittleEndian.PutUint64(e.buf[e.pos:], math.Float64bits(v))
	e.pos += 8
}

// WriteBool writes a bool to the payload
func (e *EventPayloadEncoder) WriteBool(v bool) {
	e.grow(1)
	if v {
		e.buf[e.pos] = 1
	} else {
		e.buf[e.pos] = 0
	}
	e.pos++
}

// WriteBytes writes raw bytes to the payload
func (e *EventPayloadEncoder) WriteBytes(b []byte) {
	e.grow(len(b) + 10)
	e.pos += putVarUint(e.buf[e.pos:], uint64(len(b)))
	copy(e.buf[e.pos:], b)
	e.pos += len(b)
}

// Errors
var (
	ErrInvalidEventFormat = &EventError{msg: "invalid event format"}
)

// EventError represents an event-related error
type EventError struct {
	msg string
}

func (e *EventError) Error() string {
	return "statesync: " + e.msg
}

// Helper functions for varint encoding (duplicated to avoid circular deps)

func varIntSize(v uint64) int {
	size := 1
	for v >= 0x80 {
		v >>= 7
		size++
	}
	return size
}

func putVarUint(buf []byte, v uint64) int {
	i := 0
	for v >= 0x80 {
		buf[i] = byte(v) | 0x80
		v >>= 7
		i++
	}
	buf[i] = byte(v)
	return i + 1
}

func readVarUint(buf []byte) (uint64, int) {
	var v uint64
	var shift uint
	for i, b := range buf {
		v |= uint64(b&0x7F) << shift
		if b < 0x80 {
			return v, i + 1
		}
		shift += 7
		if shift >= 64 {
			return 0, 0 // Overflow
		}
	}
	return 0, 0 // Incomplete
}
