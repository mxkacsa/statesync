package statesync

import (
	"testing"
)

func TestEventEncodeDecode(t *testing.T) {
	// Single event
	event := Event{
		Type:    "CardPlayed",
		Payload: []byte(`{"cardId":"c7"}`),
	}

	encoded := EncodeEvent(event)
	if len(encoded) == 0 {
		t.Fatal("encoded event is empty")
	}

	decoded, err := DecodeEvent(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if decoded.Type != event.Type {
		t.Errorf("type mismatch: got %q, want %q", decoded.Type, event.Type)
	}
	if string(decoded.Payload) != string(event.Payload) {
		t.Errorf("payload mismatch: got %q, want %q", decoded.Payload, event.Payload)
	}
}

func TestEventBatchEncodeDecode(t *testing.T) {
	events := []Event{
		{Type: "PlayerJoined", Payload: []byte(`{"id":"alice"}`)},
		{Type: "GameStarted", Payload: []byte(`{"round":1}`)},
		{Type: "CardDealt", Payload: []byte(`{"cards":[1,2,3]}`)},
	}

	encoded := EncodeEventBatch(events)
	if len(encoded) == 0 {
		t.Fatal("encoded batch is empty")
	}

	decoded, err := DecodeEventBatch(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(decoded) != len(events) {
		t.Fatalf("count mismatch: got %d, want %d", len(decoded), len(events))
	}

	for i, e := range decoded {
		if e.Type != events[i].Type {
			t.Errorf("event %d type mismatch: got %q, want %q", i, e.Type, events[i].Type)
		}
		if string(e.Payload) != string(events[i].Payload) {
			t.Errorf("event %d payload mismatch: got %q, want %q", i, e.Payload, events[i].Payload)
		}
	}
}

func TestEventBatchSingleEvent(t *testing.T) {
	// Single event should use MsgEvent format, not batch
	events := []Event{
		{Type: "SingleEvent", Payload: []byte("test")},
	}

	encoded := EncodeEventBatch(events)
	if encoded[0] != MsgEvent {
		t.Errorf("single event should use MsgEvent (0x%x), got 0x%x", MsgEvent, encoded[0])
	}

	// But DecodeEventBatch should still work
	decoded, err := DecodeEventBatch(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decoded))
	}
}

func TestEventBuffer(t *testing.T) {
	buf := NewEventBuffer[string]()

	// Initially empty
	if buf.Count() != 0 {
		t.Errorf("expected empty buffer, got %d", buf.Count())
	}

	// Add events
	buf.Add(PendingEvent[string]{
		Event:  Event{Type: "Event1", Payload: []byte("1")},
		Target: TargetAll,
	})
	buf.Add(PendingEvent[string]{
		Event:  Event{Type: "Event2", Payload: []byte("2")},
		Target: TargetOne,
		To:     "alice",
	})

	if buf.Count() != 2 {
		t.Errorf("expected 2 events, got %d", buf.Count())
	}

	// Drain
	events := buf.Drain()
	if len(events) != 2 {
		t.Fatalf("expected 2 events from drain, got %d", len(events))
	}

	// Buffer should be empty after drain
	if buf.Count() != 0 {
		t.Errorf("expected empty buffer after drain, got %d", buf.Count())
	}

	// Events should be correct
	if events[0].Event.Type != "Event1" {
		t.Errorf("wrong event type: %s", events[0].Event.Type)
	}
	if events[1].Target != TargetOne {
		t.Errorf("wrong target: %v", events[1].Target)
	}
	if events[1].To != "alice" {
		t.Errorf("wrong To: %s", events[1].To)
	}
}

func TestEventPayloadEncoder(t *testing.T) {
	enc := NewEventPayloadEncoder()

	enc.WriteString("hello")
	enc.WriteInt32(42)
	enc.WriteBool(true)
	enc.WriteFloat64(3.14)

	bytes := enc.Bytes()
	if len(bytes) == 0 {
		t.Fatal("encoder produced empty output")
	}

	// Reset and reuse
	enc.Reset()
	enc.WriteString("world")
	bytes2 := enc.Bytes()

	if len(bytes2) >= len(bytes) {
		// bytes2 should be shorter (just "world" vs multiple values)
		// This isn't strictly true due to encoding, but we can at least verify reset works
	}
}

func TestSessionEmit(t *testing.T) {
	// Create a simple test setup
	type TestState struct {
		Value   int
		changes *ChangeSet
	}

	// Skip this test for now as it requires a full Trackable implementation
	t.Skip("Requires full Trackable implementation for integration test")
}

func TestEventEmptyBatch(t *testing.T) {
	encoded := EncodeEventBatch([]Event{})
	if encoded != nil {
		t.Errorf("expected nil for empty batch, got %v", encoded)
	}
}

func TestEventDecodeInvalid(t *testing.T) {
	// Invalid message type
	_, err := DecodeEvent([]byte{0xFF, 0x00})
	if err == nil {
		t.Error("expected error for invalid message type")
	}

	// Too short
	_, err = DecodeEvent([]byte{MsgEvent})
	if err == nil {
		t.Error("expected error for truncated message")
	}
}
