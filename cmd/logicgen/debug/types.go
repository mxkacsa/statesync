package debug

import "time"

// NodeStatus represents the current state of a node execution
type NodeStatus string

const (
	NodeStatusPending   NodeStatus = "pending"
	NodeStatusExecuting NodeStatus = "executing"
	NodeStatusWaiting   NodeStatus = "waiting"
	NodeStatusCompleted NodeStatus = "completed"
	NodeStatusError     NodeStatus = "error"
)

// DebugHook is the interface for receiving execution trace events.
// Implementations can send these events to a WebSocket for UI visualization.
type DebugHook interface {
	// Event lifecycle
	OnEventStart(sessionID, handler string, params map[string]any)
	OnEventEnd(sessionID, handler string, durationMs float64, err error)

	// Node lifecycle
	OnNodeStart(sessionID, handler, nodeID, nodeType string, inputs map[string]any)
	OnNodeEnd(sessionID, handler, nodeID string, outputs map[string]any, durationMs float64)
	OnNodeError(sessionID, handler, nodeID string, err error)

	// Wait/async nodes
	OnNodeWait(sessionID, handler, nodeID string, duration time.Duration)
	OnNodeResume(sessionID, handler, nodeID string)
}

// Message types for WebSocket protocol
type MessageType string

const (
	MsgEventStart MessageType = "event:start"
	MsgEventEnd   MessageType = "event:end"
	MsgNodeStart  MessageType = "node:start"
	MsgNodeEnd    MessageType = "node:end"
	MsgNodeError  MessageType = "node:error"
	MsgNodeWait   MessageType = "node:wait"
	MsgNodeResume MessageType = "node:resume"
)

// DebugMessage is the JSON structure sent over WebSocket
type DebugMessage struct {
	Type       MessageType    `json:"type"`
	SessionID  string         `json:"sessionId"`
	Handler    string         `json:"handler"`
	NodeID     string         `json:"nodeId,omitempty"`
	NodeType   string         `json:"nodeType,omitempty"`
	Inputs     map[string]any `json:"inputs,omitempty"`
	Outputs    map[string]any `json:"outputs,omitempty"`
	Params     map[string]any `json:"params,omitempty"`
	DurationMs float64        `json:"durationMs,omitempty"`
	WaitMs     int64          `json:"waitMs,omitempty"`
	ResumeAt   int64          `json:"resumeAt,omitempty"`
	Error      string         `json:"error,omitempty"`
	Timestamp  int64          `json:"timestamp"`
}
