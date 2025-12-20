package debug

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// ChannelHook sends debug messages to a channel.
// Useful for testing or custom processing.
type ChannelHook struct {
	C         chan *DebugMessage
	SessionID string // Optional filter
}

// NewChannelHook creates a new channel-based debug hook.
func NewChannelHook(bufferSize int) *ChannelHook {
	return &ChannelHook{
		C: make(chan *DebugMessage, bufferSize),
	}
}

func (h *ChannelHook) send(msg *DebugMessage) {
	if h.SessionID != "" && msg.SessionID != h.SessionID {
		return
	}
	select {
	case h.C <- msg:
	default:
		// Channel full, drop message
	}
}

func (h *ChannelHook) OnEventStart(sessionID, handler string, params map[string]any) {
	h.send(&DebugMessage{
		Type:      MsgEventStart,
		SessionID: sessionID,
		Handler:   handler,
		Params:    params,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *ChannelHook) OnEventEnd(sessionID, handler string, durationMs float64, err error) {
	msg := &DebugMessage{
		Type:       MsgEventEnd,
		SessionID:  sessionID,
		Handler:    handler,
		DurationMs: durationMs,
		Timestamp:  time.Now().UnixMilli(),
	}
	if err != nil {
		msg.Error = err.Error()
	}
	h.send(msg)
}

func (h *ChannelHook) OnNodeStart(sessionID, handler, nodeID, nodeType string, inputs map[string]any) {
	h.send(&DebugMessage{
		Type:      MsgNodeStart,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		NodeType:  nodeType,
		Inputs:    inputs,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *ChannelHook) OnNodeEnd(sessionID, handler, nodeID string, outputs map[string]any, durationMs float64) {
	h.send(&DebugMessage{
		Type:       MsgNodeEnd,
		SessionID:  sessionID,
		Handler:    handler,
		NodeID:     nodeID,
		Outputs:    outputs,
		DurationMs: durationMs,
		Timestamp:  time.Now().UnixMilli(),
	})
}

func (h *ChannelHook) OnNodeError(sessionID, handler, nodeID string, err error) {
	msg := &DebugMessage{
		Type:      MsgNodeError,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		Timestamp: time.Now().UnixMilli(),
	}
	if err != nil {
		msg.Error = err.Error()
	}
	h.send(msg)
}

func (h *ChannelHook) OnNodeWait(sessionID, handler, nodeID string, duration time.Duration) {
	h.send(&DebugMessage{
		Type:      MsgNodeWait,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		WaitMs:    duration.Milliseconds(),
		ResumeAt:  time.Now().Add(duration).UnixMilli(),
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *ChannelHook) OnNodeResume(sessionID, handler, nodeID string) {
	h.send(&DebugMessage{
		Type:      MsgNodeResume,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		Timestamp: time.Now().UnixMilli(),
	})
}

// ============================================================================
// WriterHook - writes JSON to an io.Writer (file, stdout, etc)
// ============================================================================

// WriterHook writes debug messages as JSON lines to an io.Writer.
type WriterHook struct {
	w  io.Writer
	mu sync.Mutex
}

// NewWriterHook creates a hook that writes to the given writer.
func NewWriterHook(w io.Writer) *WriterHook {
	return &WriterHook{w: w}
}

func (h *WriterHook) write(msg *DebugMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.w.Write(data)
	h.w.Write([]byte("\n"))
}

func (h *WriterHook) OnEventStart(sessionID, handler string, params map[string]any) {
	h.write(&DebugMessage{
		Type:      MsgEventStart,
		SessionID: sessionID,
		Handler:   handler,
		Params:    params,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *WriterHook) OnEventEnd(sessionID, handler string, durationMs float64, err error) {
	msg := &DebugMessage{
		Type:       MsgEventEnd,
		SessionID:  sessionID,
		Handler:    handler,
		DurationMs: durationMs,
		Timestamp:  time.Now().UnixMilli(),
	}
	if err != nil {
		msg.Error = err.Error()
	}
	h.write(msg)
}

func (h *WriterHook) OnNodeStart(sessionID, handler, nodeID, nodeType string, inputs map[string]any) {
	h.write(&DebugMessage{
		Type:      MsgNodeStart,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		NodeType:  nodeType,
		Inputs:    inputs,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *WriterHook) OnNodeEnd(sessionID, handler, nodeID string, outputs map[string]any, durationMs float64) {
	h.write(&DebugMessage{
		Type:       MsgNodeEnd,
		SessionID:  sessionID,
		Handler:    handler,
		NodeID:     nodeID,
		Outputs:    outputs,
		DurationMs: durationMs,
		Timestamp:  time.Now().UnixMilli(),
	})
}

func (h *WriterHook) OnNodeError(sessionID, handler, nodeID string, err error) {
	msg := &DebugMessage{
		Type:      MsgNodeError,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		Timestamp: time.Now().UnixMilli(),
	}
	if err != nil {
		msg.Error = err.Error()
	}
	h.write(msg)
}

func (h *WriterHook) OnNodeWait(sessionID, handler, nodeID string, duration time.Duration) {
	h.write(&DebugMessage{
		Type:      MsgNodeWait,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		WaitMs:    duration.Milliseconds(),
		ResumeAt:  time.Now().Add(duration).UnixMilli(),
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *WriterHook) OnNodeResume(sessionID, handler, nodeID string) {
	h.write(&DebugMessage{
		Type:      MsgNodeResume,
		SessionID: sessionID,
		Handler:   handler,
		NodeID:    nodeID,
		Timestamp: time.Now().UnixMilli(),
	})
}

// ============================================================================
// PrintHook - prints human-readable debug output
// ============================================================================

// PrintHook prints debug messages in a human-readable format.
type PrintHook struct {
	w io.Writer
}

// NewPrintHook creates a hook that prints to the given writer.
func NewPrintHook(w io.Writer) *PrintHook {
	return &PrintHook{w: w}
}

func (h *PrintHook) OnEventStart(sessionID, handler string, params map[string]any) {
	fmt.Fprintf(h.w, "[%s] ▶ %s started\n", sessionID, handler)
}

func (h *PrintHook) OnEventEnd(sessionID, handler string, durationMs float64, err error) {
	if err != nil {
		fmt.Fprintf(h.w, "[%s] ✗ %s failed (%.2fms): %v\n", sessionID, handler, durationMs, err)
	} else {
		fmt.Fprintf(h.w, "[%s] ✓ %s completed (%.2fms)\n", sessionID, handler, durationMs)
	}
}

func (h *PrintHook) OnNodeStart(sessionID, handler, nodeID, nodeType string, inputs map[string]any) {
	fmt.Fprintf(h.w, "[%s]   → %s (%s)\n", sessionID, nodeID, nodeType)
}

func (h *PrintHook) OnNodeEnd(sessionID, handler, nodeID string, outputs map[string]any, durationMs float64) {
	// Typically silent for less noise
}

func (h *PrintHook) OnNodeError(sessionID, handler, nodeID string, err error) {
	fmt.Fprintf(h.w, "[%s]   ✗ %s error: %v\n", sessionID, nodeID, err)
}

func (h *PrintHook) OnNodeWait(sessionID, handler, nodeID string, duration time.Duration) {
	fmt.Fprintf(h.w, "[%s]   ⏳ %s waiting %v\n", sessionID, nodeID, duration)
}

func (h *PrintHook) OnNodeResume(sessionID, handler, nodeID string) {
	fmt.Fprintf(h.w, "[%s]   ▶ %s resumed\n", sessionID, nodeID)
}

// ============================================================================
// MultiHook - sends to multiple hooks
// ============================================================================

// MultiHook broadcasts debug events to multiple hooks.
type MultiHook struct {
	hooks []DebugHook
	mu    sync.RWMutex
}

// NewMultiHook creates a hook that broadcasts to multiple hooks.
func NewMultiHook(hooks ...DebugHook) *MultiHook {
	return &MultiHook{hooks: hooks}
}

// Add adds a hook to the multi-hook.
func (h *MultiHook) Add(hook DebugHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hooks = append(h.hooks, hook)
}

func (h *MultiHook) OnEventStart(sessionID, handler string, params map[string]any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, hook := range h.hooks {
		hook.OnEventStart(sessionID, handler, params)
	}
}

func (h *MultiHook) OnEventEnd(sessionID, handler string, durationMs float64, err error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, hook := range h.hooks {
		hook.OnEventEnd(sessionID, handler, durationMs, err)
	}
}

func (h *MultiHook) OnNodeStart(sessionID, handler, nodeID, nodeType string, inputs map[string]any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, hook := range h.hooks {
		hook.OnNodeStart(sessionID, handler, nodeID, nodeType, inputs)
	}
}

func (h *MultiHook) OnNodeEnd(sessionID, handler, nodeID string, outputs map[string]any, durationMs float64) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, hook := range h.hooks {
		hook.OnNodeEnd(sessionID, handler, nodeID, outputs, durationMs)
	}
}

func (h *MultiHook) OnNodeError(sessionID, handler, nodeID string, err error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, hook := range h.hooks {
		hook.OnNodeError(sessionID, handler, nodeID, err)
	}
}

func (h *MultiHook) OnNodeWait(sessionID, handler, nodeID string, duration time.Duration) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, hook := range h.hooks {
		hook.OnNodeWait(sessionID, handler, nodeID, duration)
	}
}

func (h *MultiHook) OnNodeResume(sessionID, handler, nodeID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, hook := range h.hooks {
		hook.OnNodeResume(sessionID, handler, nodeID)
	}
}
