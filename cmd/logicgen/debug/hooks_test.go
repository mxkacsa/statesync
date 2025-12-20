package debug

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// ChannelHook Tests
// ============================================================================

func TestNewChannelHook(t *testing.T) {
	hook := NewChannelHook(10)
	if hook == nil {
		t.Fatal("NewChannelHook returned nil")
	}
	if cap(hook.C) != 10 {
		t.Errorf("Channel capacity = %d, want 10", cap(hook.C))
	}
}

func TestChannelHook_OnEventStart(t *testing.T) {
	hook := NewChannelHook(10)
	params := map[string]any{"playerID": "p1"}

	hook.OnEventStart("session1", "OnJoin", params)

	select {
	case msg := <-hook.C:
		if msg.Type != MsgEventStart {
			t.Errorf("Type = %v, want %v", msg.Type, MsgEventStart)
		}
		if msg.SessionID != "session1" {
			t.Errorf("SessionID = %v, want session1", msg.SessionID)
		}
		if msg.Handler != "OnJoin" {
			t.Errorf("Handler = %v, want OnJoin", msg.Handler)
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestChannelHook_OnEventEnd(t *testing.T) {
	hook := NewChannelHook(10)

	hook.OnEventEnd("session1", "OnJoin", 5.5, nil)

	msg := <-hook.C
	if msg.Type != MsgEventEnd {
		t.Errorf("Type = %v, want %v", msg.Type, MsgEventEnd)
	}
	if msg.DurationMs != 5.5 {
		t.Errorf("DurationMs = %v, want 5.5", msg.DurationMs)
	}
	if msg.Error != "" {
		t.Errorf("Error should be empty, got %v", msg.Error)
	}
}

func TestChannelHook_OnEventEnd_WithError(t *testing.T) {
	hook := NewChannelHook(10)
	testErr := errors.New("test error")

	hook.OnEventEnd("session1", "OnJoin", 5.5, testErr)

	msg := <-hook.C
	if msg.Error != "test error" {
		t.Errorf("Error = %v, want 'test error'", msg.Error)
	}
}

func TestChannelHook_OnNodeStart(t *testing.T) {
	hook := NewChannelHook(10)
	inputs := map[string]any{"value": 42}

	hook.OnNodeStart("session1", "OnJoin", "node1", "Compare", inputs)

	msg := <-hook.C
	if msg.Type != MsgNodeStart {
		t.Errorf("Type = %v, want %v", msg.Type, MsgNodeStart)
	}
	if msg.NodeID != "node1" {
		t.Errorf("NodeID = %v, want node1", msg.NodeID)
	}
	if msg.NodeType != "Compare" {
		t.Errorf("NodeType = %v, want Compare", msg.NodeType)
	}
}

func TestChannelHook_OnNodeEnd(t *testing.T) {
	hook := NewChannelHook(10)
	outputs := map[string]any{"result": true}

	hook.OnNodeEnd("session1", "OnJoin", "node1", outputs, 1.5)

	msg := <-hook.C
	if msg.Type != MsgNodeEnd {
		t.Errorf("Type = %v, want %v", msg.Type, MsgNodeEnd)
	}
	if msg.DurationMs != 1.5 {
		t.Errorf("DurationMs = %v, want 1.5", msg.DurationMs)
	}
}

func TestChannelHook_OnNodeError(t *testing.T) {
	hook := NewChannelHook(10)
	testErr := errors.New("node error")

	hook.OnNodeError("session1", "OnJoin", "node1", testErr)

	msg := <-hook.C
	if msg.Type != MsgNodeError {
		t.Errorf("Type = %v, want %v", msg.Type, MsgNodeError)
	}
	if msg.Error != "node error" {
		t.Errorf("Error = %v, want 'node error'", msg.Error)
	}
}

func TestChannelHook_OnNodeWait(t *testing.T) {
	hook := NewChannelHook(10)

	hook.OnNodeWait("session1", "OnJoin", "node1", 100*time.Millisecond)

	msg := <-hook.C
	if msg.Type != MsgNodeWait {
		t.Errorf("Type = %v, want %v", msg.Type, MsgNodeWait)
	}
	if msg.WaitMs != 100 {
		t.Errorf("WaitMs = %v, want 100", msg.WaitMs)
	}
}

func TestChannelHook_OnNodeResume(t *testing.T) {
	hook := NewChannelHook(10)

	hook.OnNodeResume("session1", "OnJoin", "node1")

	msg := <-hook.C
	if msg.Type != MsgNodeResume {
		t.Errorf("Type = %v, want %v", msg.Type, MsgNodeResume)
	}
}

func TestChannelHook_SessionFilter(t *testing.T) {
	hook := NewChannelHook(10)
	hook.SessionID = "session1" // Only accept session1

	hook.OnEventStart("session1", "OnJoin", nil)
	hook.OnEventStart("session2", "OnJoin", nil) // Should be filtered

	// Should only receive one message
	select {
	case msg := <-hook.C:
		if msg.SessionID != "session1" {
			t.Errorf("SessionID = %v, want session1", msg.SessionID)
		}
	default:
		t.Fatal("Expected one message")
	}

	// Channel should be empty now
	select {
	case <-hook.C:
		t.Fatal("Unexpected second message")
	default:
		// Expected
	}
}

func TestChannelHook_FullChannel(t *testing.T) {
	hook := NewChannelHook(1)

	// Fill the channel
	hook.OnEventStart("session1", "Handler1", nil)
	// This should not block, just drop the message
	hook.OnEventStart("session2", "Handler2", nil)

	// Only first message should be in channel
	msg := <-hook.C
	if msg.SessionID != "session1" {
		t.Errorf("SessionID = %v, want session1", msg.SessionID)
	}
}

// ============================================================================
// WriterHook Tests
// ============================================================================

func TestNewWriterHook(t *testing.T) {
	var buf bytes.Buffer
	hook := NewWriterHook(&buf)
	if hook == nil {
		t.Fatal("NewWriterHook returned nil")
	}
}

func TestWriterHook_OnEventStart(t *testing.T) {
	var buf bytes.Buffer
	hook := NewWriterHook(&buf)

	hook.OnEventStart("session1", "OnJoin", map[string]any{"id": "p1"})

	var msg DebugMessage
	if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if msg.Type != MsgEventStart {
		t.Errorf("Type = %v, want %v", msg.Type, MsgEventStart)
	}
}

func TestWriterHook_OnEventEnd(t *testing.T) {
	var buf bytes.Buffer
	hook := NewWriterHook(&buf)

	hook.OnEventEnd("session1", "OnJoin", 10.5, nil)

	var msg DebugMessage
	json.Unmarshal(buf.Bytes(), &msg)
	if msg.Type != MsgEventEnd {
		t.Errorf("Type = %v, want %v", msg.Type, MsgEventEnd)
	}
	if msg.DurationMs != 10.5 {
		t.Errorf("DurationMs = %v, want 10.5", msg.DurationMs)
	}
}

func TestWriterHook_OnEventEnd_WithError(t *testing.T) {
	var buf bytes.Buffer
	hook := NewWriterHook(&buf)

	hook.OnEventEnd("session1", "OnJoin", 10.5, errors.New("test error"))

	var msg DebugMessage
	json.Unmarshal(buf.Bytes(), &msg)
	if msg.Error != "test error" {
		t.Errorf("Error = %v, want 'test error'", msg.Error)
	}
}

func TestWriterHook_OnNodeStart(t *testing.T) {
	var buf bytes.Buffer
	hook := NewWriterHook(&buf)

	hook.OnNodeStart("session1", "OnJoin", "node1", "Compare", nil)

	var msg DebugMessage
	json.Unmarshal(buf.Bytes(), &msg)
	if msg.Type != MsgNodeStart {
		t.Errorf("Type = %v, want %v", msg.Type, MsgNodeStart)
	}
	if msg.NodeID != "node1" {
		t.Errorf("NodeID = %v, want node1", msg.NodeID)
	}
}

func TestWriterHook_OnNodeEnd(t *testing.T) {
	var buf bytes.Buffer
	hook := NewWriterHook(&buf)

	hook.OnNodeEnd("session1", "OnJoin", "node1", map[string]any{"result": true}, 2.5)

	var msg DebugMessage
	json.Unmarshal(buf.Bytes(), &msg)
	if msg.Type != MsgNodeEnd {
		t.Errorf("Type = %v, want %v", msg.Type, MsgNodeEnd)
	}
}

func TestWriterHook_OnNodeError(t *testing.T) {
	var buf bytes.Buffer
	hook := NewWriterHook(&buf)

	hook.OnNodeError("session1", "OnJoin", "node1", errors.New("node failed"))

	var msg DebugMessage
	json.Unmarshal(buf.Bytes(), &msg)
	if msg.Type != MsgNodeError {
		t.Errorf("Type = %v, want %v", msg.Type, MsgNodeError)
	}
	if msg.Error != "node failed" {
		t.Errorf("Error = %v, want 'node failed'", msg.Error)
	}
}

func TestWriterHook_OnNodeWait(t *testing.T) {
	var buf bytes.Buffer
	hook := NewWriterHook(&buf)

	hook.OnNodeWait("session1", "OnJoin", "node1", 500*time.Millisecond)

	var msg DebugMessage
	json.Unmarshal(buf.Bytes(), &msg)
	if msg.Type != MsgNodeWait {
		t.Errorf("Type = %v, want %v", msg.Type, MsgNodeWait)
	}
	if msg.WaitMs != 500 {
		t.Errorf("WaitMs = %v, want 500", msg.WaitMs)
	}
}

func TestWriterHook_OnNodeResume(t *testing.T) {
	var buf bytes.Buffer
	hook := NewWriterHook(&buf)

	hook.OnNodeResume("session1", "OnJoin", "node1")

	var msg DebugMessage
	json.Unmarshal(buf.Bytes(), &msg)
	if msg.Type != MsgNodeResume {
		t.Errorf("Type = %v, want %v", msg.Type, MsgNodeResume)
	}
}

// ============================================================================
// PrintHook Tests
// ============================================================================

func TestNewPrintHook(t *testing.T) {
	var buf bytes.Buffer
	hook := NewPrintHook(&buf)
	if hook == nil {
		t.Fatal("NewPrintHook returned nil")
	}
}

func TestPrintHook_OnEventStart(t *testing.T) {
	var buf bytes.Buffer
	hook := NewPrintHook(&buf)

	hook.OnEventStart("session1", "OnJoin", nil)

	output := buf.String()
	if !strings.Contains(output, "session1") {
		t.Error("Output should contain sessionID")
	}
	if !strings.Contains(output, "OnJoin") {
		t.Error("Output should contain handler name")
	}
}

func TestPrintHook_OnEventEnd_Success(t *testing.T) {
	var buf bytes.Buffer
	hook := NewPrintHook(&buf)

	hook.OnEventEnd("session1", "OnJoin", 5.5, nil)

	output := buf.String()
	if !strings.Contains(output, "completed") {
		t.Error("Output should contain 'completed'")
	}
}

func TestPrintHook_OnEventEnd_Error(t *testing.T) {
	var buf bytes.Buffer
	hook := NewPrintHook(&buf)

	hook.OnEventEnd("session1", "OnJoin", 5.5, errors.New("test error"))

	output := buf.String()
	if !strings.Contains(output, "failed") {
		t.Error("Output should contain 'failed'")
	}
	if !strings.Contains(output, "test error") {
		t.Error("Output should contain error message")
	}
}

func TestPrintHook_OnNodeStart(t *testing.T) {
	var buf bytes.Buffer
	hook := NewPrintHook(&buf)

	hook.OnNodeStart("session1", "OnJoin", "node1", "Compare", nil)

	output := buf.String()
	if !strings.Contains(output, "node1") {
		t.Error("Output should contain node ID")
	}
	if !strings.Contains(output, "Compare") {
		t.Error("Output should contain node type")
	}
}

func TestPrintHook_OnNodeError(t *testing.T) {
	var buf bytes.Buffer
	hook := NewPrintHook(&buf)

	hook.OnNodeError("session1", "OnJoin", "node1", errors.New("node error"))

	output := buf.String()
	if !strings.Contains(output, "error") {
		t.Error("Output should contain 'error'")
	}
}

func TestPrintHook_OnNodeWait(t *testing.T) {
	var buf bytes.Buffer
	hook := NewPrintHook(&buf)

	hook.OnNodeWait("session1", "OnJoin", "node1", 100*time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "waiting") {
		t.Error("Output should contain 'waiting'")
	}
}

func TestPrintHook_OnNodeResume(t *testing.T) {
	var buf bytes.Buffer
	hook := NewPrintHook(&buf)

	hook.OnNodeResume("session1", "OnJoin", "node1")

	output := buf.String()
	if !strings.Contains(output, "resumed") {
		t.Error("Output should contain 'resumed'")
	}
}

// ============================================================================
// MultiHook Tests
// ============================================================================

func TestNewMultiHook(t *testing.T) {
	hook := NewMultiHook()
	if hook == nil {
		t.Fatal("NewMultiHook returned nil")
	}
}

func TestMultiHook_Add(t *testing.T) {
	multi := NewMultiHook()
	ch := NewChannelHook(10)

	multi.Add(ch)

	multi.OnEventStart("session1", "OnJoin", nil)

	select {
	case <-ch.C:
		// Success
	default:
		t.Fatal("Expected message in channel")
	}
}

func TestMultiHook_Broadcasts(t *testing.T) {
	ch1 := NewChannelHook(10)
	ch2 := NewChannelHook(10)
	multi := NewMultiHook(ch1, ch2)

	multi.OnEventStart("session1", "OnJoin", nil)

	// Both channels should receive the message
	select {
	case <-ch1.C:
	default:
		t.Error("Channel 1 should receive message")
	}

	select {
	case <-ch2.C:
	default:
		t.Error("Channel 2 should receive message")
	}
}

func TestMultiHook_AllMethods(t *testing.T) {
	ch := NewChannelHook(100)
	multi := NewMultiHook(ch)

	multi.OnEventStart("s", "h", nil)
	multi.OnEventEnd("s", "h", 1.0, nil)
	multi.OnNodeStart("s", "h", "n", "Compare", nil)
	multi.OnNodeEnd("s", "h", "n", nil, 1.0)
	multi.OnNodeError("s", "h", "n", errors.New("err"))
	multi.OnNodeWait("s", "h", "n", time.Second)
	multi.OnNodeResume("s", "h", "n")

	// Should have 7 messages
	count := 0
	for {
		select {
		case <-ch.C:
			count++
		default:
			goto done
		}
	}
done:
	if count != 7 {
		t.Errorf("Message count = %d, want 7", count)
	}
}

// ============================================================================
// NoopHook Tests
// ============================================================================

func TestNoopHook(t *testing.T) {
	hook := NoopHook{}

	// These should not panic
	hook.OnEventStart("s", "h", nil)
	hook.OnEventEnd("s", "h", 1.0, nil)
	hook.OnNodeStart("s", "h", "n", "t", nil)
	hook.OnNodeEnd("s", "h", "n", nil, 1.0)
	hook.OnNodeError("s", "h", "n", nil)
	hook.OnNodeWait("s", "h", "n", time.Second)
	hook.OnNodeResume("s", "h", "n")
}
