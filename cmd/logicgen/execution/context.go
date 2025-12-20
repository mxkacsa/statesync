// Package execution provides runtime support for generated logicgen handlers.
package execution

import (
	"context"
	"time"
)

// Context wraps context.Context with additional execution metadata.
// Generated handlers receive this for cancellation and async operations.
type Context struct {
	context.Context
	SessionID string
	HandlerID string
	cancel    context.CancelFunc
}

// NewContext creates a new execution context with cancellation support.
func NewContext(sessionID, handlerID string) *Context {
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{
		Context:   ctx,
		SessionID: sessionID,
		HandlerID: handlerID,
		cancel:    cancel,
	}
}

// NewContextWithTimeout creates an execution context with a timeout.
func NewContextWithTimeout(sessionID, handlerID string, timeout time.Duration) *Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &Context{
		Context:   ctx,
		SessionID: sessionID,
		HandlerID: handlerID,
		cancel:    cancel,
	}
}

// NewContextWithParent creates a child context from a parent context.
func NewContextWithParent(parent context.Context, sessionID, handlerID string) *Context {
	ctx, cancel := context.WithCancel(parent)
	return &Context{
		Context:   ctx,
		SessionID: sessionID,
		HandlerID: handlerID,
		cancel:    cancel,
	}
}

// Cancel cancels the execution context.
func (c *Context) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Wait pauses execution for the specified duration.
// Returns an error if the context is cancelled during the wait.
func (c *Context) Wait(d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-c.Done():
		return c.Err()
	}
}

// WaitUntil waits until the condition function returns true or timeout.
// checkInterval is how often to check the condition.
// Returns true if timed out, false if condition was met.
func (c *Context) WaitUntil(condition func() bool, checkInterval, timeout time.Duration) (timedOut bool, err error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		if condition() {
			return false, nil
		}

		if timeout > 0 && time.Now().After(deadline) {
			return true, nil
		}

		select {
		case <-ticker.C:
			// Check again
		case <-c.Done():
			return false, c.Err()
		}
	}
}
