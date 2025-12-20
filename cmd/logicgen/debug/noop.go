package debug

import "time"

// NoopHook is a debug hook that does nothing.
// Used in production builds where debug is disabled.
type NoopHook struct{}

var _ DebugHook = (*NoopHook)(nil)

func (NoopHook) OnEventStart(sessionID, handler string, params map[string]any)       {}
func (NoopHook) OnEventEnd(sessionID, handler string, durationMs float64, err error) {}
func (NoopHook) OnNodeStart(sessionID, handler, nodeID, nodeType string, inputs map[string]any) {
}
func (NoopHook) OnNodeEnd(sessionID, handler, nodeID string, outputs map[string]any, durationMs float64) {
}
func (NoopHook) OnNodeError(sessionID, handler, nodeID string, err error)             {}
func (NoopHook) OnNodeWait(sessionID, handler, nodeID string, duration time.Duration) {}
func (NoopHook) OnNodeResume(sessionID, handler, nodeID string)                       {}
