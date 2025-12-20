package execution

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Context Tests
// ============================================================================

func TestNewContext(t *testing.T) {
	ctx := NewContext("session1", "handler1")

	if ctx == nil {
		t.Fatal("NewContext returned nil")
	}
	if ctx.SessionID != "session1" {
		t.Errorf("SessionID = %v, want session1", ctx.SessionID)
	}
	if ctx.HandlerID != "handler1" {
		t.Errorf("HandlerID = %v, want handler1", ctx.HandlerID)
	}
	if ctx.Context == nil {
		t.Error("Context should not be nil")
	}
}

func TestNewContextWithTimeout(t *testing.T) {
	ctx := NewContextWithTimeout("session1", "handler1", 100*time.Millisecond)

	if ctx == nil {
		t.Fatal("NewContextWithTimeout returned nil")
	}

	// Should have a deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}
	if deadline.Before(time.Now()) {
		t.Error("Deadline should be in the future")
	}
}

func TestNewContextWithParent(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx := NewContextWithParent(parent, "session1", "handler1")

	if ctx == nil {
		t.Fatal("NewContextWithParent returned nil")
	}

	// Cancel parent should cancel child
	cancel()

	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Child context should be cancelled when parent is cancelled")
	}
}

func TestContext_Cancel(t *testing.T) {
	ctx := NewContext("session1", "handler1")

	// Should not be done yet
	select {
	case <-ctx.Done():
		t.Error("Context should not be done yet")
	default:
		// Expected
	}

	ctx.Cancel()

	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled")
	}
}

func TestContext_Cancel_Nil(t *testing.T) {
	ctx := &Context{}
	// Should not panic
	ctx.Cancel()
}

func TestContext_Wait(t *testing.T) {
	ctx := NewContext("session1", "handler1")

	start := time.Now()
	err := ctx.Wait(50 * time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Wait returned error: %v", err)
	}
	if elapsed < 40*time.Millisecond {
		t.Error("Wait should have waited at least 40ms")
	}
}

func TestContext_Wait_Cancelled(t *testing.T) {
	ctx := NewContext("session1", "handler1")

	// Cancel after 20ms
	go func() {
		time.Sleep(20 * time.Millisecond)
		ctx.Cancel()
	}()

	start := time.Now()
	err := ctx.Wait(200 * time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Wait should return error when cancelled")
	}
	if elapsed > 100*time.Millisecond {
		t.Error("Wait should have been interrupted by cancellation")
	}
}

func TestContext_WaitUntil_ConditionMet(t *testing.T) {
	ctx := NewContext("session1", "handler1")

	counter := 0
	condition := func() bool {
		counter++
		return counter >= 3
	}

	timedOut, err := ctx.WaitUntil(condition, 10*time.Millisecond, time.Second)

	if err != nil {
		t.Errorf("WaitUntil returned error: %v", err)
	}
	if timedOut {
		t.Error("Should not have timed out")
	}
	if counter < 3 {
		t.Errorf("Condition was checked %d times, expected at least 3", counter)
	}
}

func TestContext_WaitUntil_Timeout(t *testing.T) {
	ctx := NewContext("session1", "handler1")

	condition := func() bool {
		return false // Never true
	}

	timedOut, err := ctx.WaitUntil(condition, 10*time.Millisecond, 50*time.Millisecond)

	if err != nil {
		t.Errorf("WaitUntil returned error: %v", err)
	}
	if !timedOut {
		t.Error("Should have timed out")
	}
}

func TestContext_WaitUntil_Cancelled(t *testing.T) {
	ctx := NewContext("session1", "handler1")

	// Cancel after 20ms
	go func() {
		time.Sleep(20 * time.Millisecond)
		ctx.Cancel()
	}()

	condition := func() bool {
		return false // Never true
	}

	_, err := ctx.WaitUntil(condition, 10*time.Millisecond, time.Second)

	if err == nil {
		t.Error("WaitUntil should return error when cancelled")
	}
}

// ============================================================================
// Dispatcher Tests
// ============================================================================

func TestNewDispatcher(t *testing.T) {
	d := NewDispatcher()

	if d == nil {
		t.Fatal("NewDispatcher returned nil")
	}
	if d.handlers == nil {
		t.Error("handlers map should not be nil")
	}
	if d.OnError == nil {
		t.Error("OnError should have a default handler")
	}
}

func TestDispatcher_Register(t *testing.T) {
	d := NewDispatcher()

	called := false
	d.Register("TestEvent", func(ctx *Context, params map[string]any) error {
		called = true
		return nil
	})

	// Dispatch synchronously to verify registration
	err := d.DispatchSync("session1", "TestEvent", nil)
	if err != nil {
		t.Errorf("DispatchSync returned error: %v", err)
	}
	if !called {
		t.Error("Handler was not called")
	}
}

func TestDispatcher_RegisterSimple(t *testing.T) {
	d := NewDispatcher()

	called := false
	d.RegisterSimple("TestEvent", func(ctx *Context) error {
		called = true
		return nil
	})

	err := d.DispatchSync("session1", "TestEvent", nil)
	if err != nil {
		t.Errorf("DispatchSync returned error: %v", err)
	}
	if !called {
		t.Error("Handler was not called")
	}
}

func TestDispatcher_Dispatch_Async(t *testing.T) {
	d := NewDispatcher()

	var wg sync.WaitGroup
	wg.Add(1)

	var calledSessionID string
	d.Register("TestEvent", func(ctx *Context, params map[string]any) error {
		calledSessionID = ctx.SessionID
		wg.Done()
		return nil
	})

	d.Dispatch("session1", "TestEvent", nil)

	// Wait for async handler
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if calledSessionID != "session1" {
			t.Errorf("SessionID = %v, want session1", calledSessionID)
		}
	case <-time.After(time.Second):
		t.Error("Handler was not called within timeout")
	}
}

func TestDispatcher_Dispatch_UnknownEvent(t *testing.T) {
	d := NewDispatcher()

	// Should not panic
	d.Dispatch("session1", "UnknownEvent", nil)
}

func TestDispatcher_DispatchSync_UnknownEvent(t *testing.T) {
	d := NewDispatcher()

	err := d.DispatchSync("session1", "UnknownEvent", nil)
	if err != nil {
		t.Errorf("DispatchSync for unknown event should return nil, got %v", err)
	}
}

func TestDispatcher_DispatchSync_WithError(t *testing.T) {
	d := NewDispatcher()

	expectedErr := errors.New("test error")
	d.Register("TestEvent", func(ctx *Context, params map[string]any) error {
		return expectedErr
	})

	err := d.DispatchSync("session1", "TestEvent", nil)
	if err != expectedErr {
		t.Errorf("DispatchSync should return handler error, got %v", err)
	}
}

func TestDispatcher_Dispatch_WithError(t *testing.T) {
	d := NewDispatcher()

	var errorReceived error
	var errorSession string
	var errorEvent string

	d.OnError = func(sessionID, event string, err error) {
		errorSession = sessionID
		errorEvent = event
		errorReceived = err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	expectedErr := errors.New("test error")
	d.Register("TestEvent", func(ctx *Context, params map[string]any) error {
		defer wg.Done()
		return expectedErr
	})

	d.Dispatch("session1", "TestEvent", nil)
	wg.Wait()

	// Give time for error handler to be called
	time.Sleep(10 * time.Millisecond)

	if errorReceived != expectedErr {
		t.Errorf("OnError should receive error, got %v", errorReceived)
	}
	if errorSession != "session1" {
		t.Errorf("OnError session = %v, want session1", errorSession)
	}
	if errorEvent != "TestEvent" {
		t.Errorf("OnError event = %v, want TestEvent", errorEvent)
	}
}

func TestDispatcher_WithParams(t *testing.T) {
	d := NewDispatcher()

	var receivedParams map[string]any
	d.Register("TestEvent", func(ctx *Context, params map[string]any) error {
		receivedParams = params
		return nil
	})

	params := map[string]any{"key": "value", "num": 42}
	d.DispatchSync("session1", "TestEvent", params)

	if receivedParams["key"] != "value" {
		t.Errorf("Params key = %v, want value", receivedParams["key"])
	}
	if receivedParams["num"] != 42 {
		t.Errorf("Params num = %v, want 42", receivedParams["num"])
	}
}

// ============================================================================
// TrackedDispatcher Tests
// ============================================================================

func TestNewTrackedDispatcher(t *testing.T) {
	d := NewTrackedDispatcher()

	if d == nil {
		t.Fatal("NewTrackedDispatcher returned nil")
	}
	if d.Dispatcher == nil {
		t.Error("Dispatcher should not be nil")
	}
	if d.active == nil {
		t.Error("active map should not be nil")
	}
}

func TestTrackedDispatcher_DispatchTracked(t *testing.T) {
	d := NewTrackedDispatcher()

	done := make(chan struct{})
	d.Register("TestEvent", func(ctx *Context, params map[string]any) error {
		<-done
		return nil
	})

	handlerID := d.DispatchTracked("session1", "TestEvent", nil)

	if handlerID == "" {
		t.Error("DispatchTracked should return a handler ID")
	}

	// Handler should be active
	if d.ActiveCount() != 1 {
		t.Errorf("ActiveCount = %d, want 1", d.ActiveCount())
	}

	close(done)
	time.Sleep(50 * time.Millisecond) // Wait for handler to complete

	if d.ActiveCount() != 0 {
		t.Errorf("ActiveCount after completion = %d, want 0", d.ActiveCount())
	}
}

func TestTrackedDispatcher_DispatchTracked_UnknownEvent(t *testing.T) {
	d := NewTrackedDispatcher()

	handlerID := d.DispatchTracked("session1", "UnknownEvent", nil)

	if handlerID != "" {
		t.Errorf("DispatchTracked for unknown event should return empty string, got %v", handlerID)
	}
}

func TestTrackedDispatcher_CancelHandler(t *testing.T) {
	d := NewTrackedDispatcher()

	var cancelled int32
	d.Register("TestEvent", func(ctx *Context, params map[string]any) error {
		<-ctx.Done()
		atomic.StoreInt32(&cancelled, 1)
		return ctx.Err()
	})

	handlerID := d.DispatchTracked("session1", "TestEvent", nil)
	time.Sleep(20 * time.Millisecond) // Let handler start

	ok := d.CancelHandler(handlerID)
	if !ok {
		t.Error("CancelHandler should return true for active handler")
	}

	time.Sleep(50 * time.Millisecond) // Wait for handler to process cancellation

	if atomic.LoadInt32(&cancelled) != 1 {
		t.Error("Handler should have been cancelled")
	}
}

func TestTrackedDispatcher_CancelHandler_Unknown(t *testing.T) {
	d := NewTrackedDispatcher()

	ok := d.CancelHandler("unknown-id")
	if ok {
		t.Error("CancelHandler should return false for unknown handler")
	}
}

func TestTrackedDispatcher_CancelSession(t *testing.T) {
	d := NewTrackedDispatcher()

	var cancelCount int32
	started := make(chan struct{}, 3)

	// Register different events to avoid handler ID collision
	// (HandlerID is set to event name in current implementation)
	d.Register("Event1", func(ctx *Context, params map[string]any) error {
		started <- struct{}{}
		<-ctx.Done()
		atomic.AddInt32(&cancelCount, 1)
		return nil
	})
	d.Register("Event2", func(ctx *Context, params map[string]any) error {
		started <- struct{}{}
		<-ctx.Done()
		atomic.AddInt32(&cancelCount, 1)
		return nil
	})
	d.Register("Event3", func(ctx *Context, params map[string]any) error {
		started <- struct{}{}
		<-ctx.Done()
		atomic.AddInt32(&cancelCount, 1)
		return nil
	})

	// Start multiple handlers for session1 and one for session2
	d.DispatchTracked("session1", "Event1", nil)
	d.DispatchTracked("session1", "Event2", nil)
	d.DispatchTracked("session2", "Event3", nil) // Different session

	// Wait for all handlers to start
	for i := 0; i < 3; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for handlers to start")
		}
	}

	count := d.CancelSession("session1")
	if count != 2 {
		t.Errorf("CancelSession should cancel 2 handlers, got %d", count)
	}

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&cancelCount) != 2 {
		t.Errorf("Should have cancelled 2 handlers, got %d", atomic.LoadInt32(&cancelCount))
	}

	// session2 handler should still be active
	if d.ActiveCount() != 1 {
		t.Errorf("Should still have 1 active handler (session2), got %d", d.ActiveCount())
	}
}

func TestTrackedDispatcher_ActiveCount(t *testing.T) {
	d := NewTrackedDispatcher()

	if d.ActiveCount() != 0 {
		t.Errorf("Initial ActiveCount = %d, want 0", d.ActiveCount())
	}

	done := make(chan struct{})
	started := make(chan struct{}, 2)

	// Register different events to avoid handler ID collision
	// (HandlerID is set to event name in current implementation)
	d.Register("Event1", func(ctx *Context, params map[string]any) error {
		started <- struct{}{}
		<-done
		return nil
	})
	d.Register("Event2", func(ctx *Context, params map[string]any) error {
		started <- struct{}{}
		<-done
		return nil
	})

	d.DispatchTracked("session1", "Event1", nil)
	d.DispatchTracked("session2", "Event2", nil)

	// Wait for both handlers to start
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for handlers to start")
		}
	}

	if d.ActiveCount() != 2 {
		t.Errorf("ActiveCount = %d, want 2", d.ActiveCount())
	}

	close(done)
	time.Sleep(50 * time.Millisecond)

	if d.ActiveCount() != 0 {
		t.Errorf("ActiveCount after completion = %d, want 0", d.ActiveCount())
	}
}

func TestTrackedDispatcher_ConcurrentOperations(t *testing.T) {
	d := NewTrackedDispatcher()

	d.Register("TestEvent", func(ctx *Context, params map[string]any) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d.DispatchTracked("session", "TestEvent", nil)
		}(i)
	}
	wg.Wait()

	// All handlers should complete without race conditions
	time.Sleep(100 * time.Millisecond)
	if d.ActiveCount() != 0 {
		t.Errorf("All handlers should have completed, got %d active", d.ActiveCount())
	}
}
