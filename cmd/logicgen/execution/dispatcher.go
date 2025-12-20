package execution

import (
	"log"
	"sync"
)

// HandlerFunc is the signature for event handlers.
// The handler receives an execution context and returns an error.
type HandlerFunc func(ctx *Context) error

// HandlerFuncWithParams is a handler that also receives parameters.
type HandlerFuncWithParams func(ctx *Context, params map[string]any) error

// Dispatcher manages event routing to handlers.
// Each event runs in its own goroutine for concurrent execution.
type Dispatcher struct {
	handlers map[string]HandlerFuncWithParams
	mu       sync.RWMutex

	// Optional error handler
	OnError func(sessionID, event string, err error)
}

// NewDispatcher creates a new event dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string]HandlerFuncWithParams),
		OnError: func(sessionID, event string, err error) {
			log.Printf("[%s] Event %s error: %v", sessionID, event, err)
		},
	}
}

// Register registers a handler for an event type.
func (d *Dispatcher) Register(event string, handler HandlerFuncWithParams) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[event] = handler
}

// RegisterSimple registers a handler without parameters.
func (d *Dispatcher) RegisterSimple(event string, handler HandlerFunc) {
	d.Register(event, func(ctx *Context, params map[string]any) error {
		return handler(ctx)
	})
}

// Dispatch dispatches an event to its handler in a new goroutine.
// Returns immediately; the handler runs asynchronously.
func (d *Dispatcher) Dispatch(sessionID, event string, params map[string]any) {
	d.mu.RLock()
	handler, ok := d.handlers[event]
	d.mu.RUnlock()

	if !ok {
		return
	}

	go func() {
		ctx := NewContext(sessionID, event)
		defer ctx.Cancel()

		if err := handler(ctx, params); err != nil {
			if d.OnError != nil {
				d.OnError(sessionID, event, err)
			}
		}
	}()
}

// DispatchSync dispatches an event synchronously (blocks until complete).
func (d *Dispatcher) DispatchSync(sessionID, event string, params map[string]any) error {
	d.mu.RLock()
	handler, ok := d.handlers[event]
	d.mu.RUnlock()

	if !ok {
		return nil
	}

	ctx := NewContext(sessionID, event)
	defer ctx.Cancel()

	return handler(ctx, params)
}

// ActiveHandler tracks a running handler for cancellation.
type ActiveHandler struct {
	SessionID string
	Event     string
	Context   *Context
}

// TrackedDispatcher extends Dispatcher with handler tracking and cancellation.
type TrackedDispatcher struct {
	*Dispatcher
	active map[string]*ActiveHandler // handlerID -> handler
	mu     sync.RWMutex
}

// NewTrackedDispatcher creates a dispatcher that tracks active handlers.
func NewTrackedDispatcher() *TrackedDispatcher {
	return &TrackedDispatcher{
		Dispatcher: NewDispatcher(),
		active:     make(map[string]*ActiveHandler),
	}
}

// DispatchTracked dispatches an event and tracks it for cancellation.
// Returns a handler ID that can be used to cancel the handler.
func (d *TrackedDispatcher) DispatchTracked(sessionID, event string, params map[string]any) string {
	d.Dispatcher.mu.RLock()
	handler, ok := d.Dispatcher.handlers[event]
	d.Dispatcher.mu.RUnlock()

	if !ok {
		return ""
	}

	ctx := NewContext(sessionID, event)
	handlerID := ctx.HandlerID

	d.mu.Lock()
	d.active[handlerID] = &ActiveHandler{
		SessionID: sessionID,
		Event:     event,
		Context:   ctx,
	}
	d.mu.Unlock()

	go func() {
		defer func() {
			ctx.Cancel()
			d.mu.Lock()
			delete(d.active, handlerID)
			d.mu.Unlock()
		}()

		if err := handler(ctx, params); err != nil {
			if d.Dispatcher.OnError != nil {
				d.Dispatcher.OnError(sessionID, event, err)
			}
		}
	}()

	return handlerID
}

// CancelHandler cancels a running handler by ID.
func (d *TrackedDispatcher) CancelHandler(handlerID string) bool {
	d.mu.RLock()
	active, ok := d.active[handlerID]
	d.mu.RUnlock()

	if ok && active.Context != nil {
		active.Context.Cancel()
		return true
	}
	return false
}

// CancelSession cancels all handlers for a session.
func (d *TrackedDispatcher) CancelSession(sessionID string) int {
	d.mu.RLock()
	var toCancel []*ActiveHandler
	for _, active := range d.active {
		if active.SessionID == sessionID {
			toCancel = append(toCancel, active)
		}
	}
	d.mu.RUnlock()

	for _, active := range toCancel {
		active.Context.Cancel()
	}

	return len(toCancel)
}

// ActiveCount returns the number of active handlers.
func (d *TrackedDispatcher) ActiveCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.active)
}
