# Thread Safety Guide for Logicgen Execution

## Overview

This document describes the thread-safety characteristics of the logicgen execution model and provides guidelines for safe concurrent usage.

## Execution Model

### Event Dispatching

Each event handler runs in its own goroutine when using `Dispatcher.Dispatch()`:

```go
dispatcher := execution.NewDispatcher()
dispatcher.Register("Attack", OnAttack)

// Each call spawns a new goroutine
dispatcher.Dispatch(sessionID, "Attack", params)  // goroutine 1
dispatcher.Dispatch(sessionID, "Attack", params)  // goroutine 2
dispatcher.Dispatch(sessionID, "Move", params)    // goroutine 3
```

### Concurrency Levels

| Component | Thread-Safe? | Notes |
|-----------|--------------|-------|
| `Dispatcher` | ✅ Yes | Registration and dispatch are protected by mutex |
| `TrackedDispatcher` | ✅ Yes | Active handler tracking is mutex-protected |
| `Context` | ✅ Yes | Based on Go's context.Context |
| Generated Handlers | ⚠️ Depends | See State Access section |
| `statesync.TrackedSession` | ✅ Yes | Must be implemented as thread-safe |

## State Access Requirements

### TrackedSession Implementation

The `statesync.TrackedSession` must implement thread-safe access to state. All getter and setter methods must be safe for concurrent use:

```go
type TrackedSession[S, M, P any] interface {
    // These must all be goroutine-safe:
    State() StateAccessor[S]
    ID() string
    Emit(event string, data any)
    EmitTo(playerID P, event string, data any)
}

type StateAccessor[S any] interface {
    Get() S  // Must return a thread-safe state or copy
}
```

### Recommended Implementation Patterns

#### Option 1: Mutex-Protected State

```go
type ThreadSafeSession struct {
    mu    sync.RWMutex
    state *GameState
}

func (s *ThreadSafeSession) State() *GameState {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.state
}

// All state setters must lock:
func (state *GameState) SetScore(v int) {
    state.session.mu.Lock()
    defer state.session.mu.Unlock()
    state.score = v
}
```

#### Option 2: Copy-on-Read State

```go
func (s *Session) State() *GameState {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.state.DeepCopy()
}
```

#### Option 3: Immutable State with Atomic Swap

```go
type Session struct {
    state atomic.Pointer[GameState]
}

func (s *Session) State() *GameState {
    return s.state.Load()
}

func (s *Session) UpdateState(fn func(*GameState) *GameState) {
    for {
        old := s.state.Load()
        new := fn(old.DeepCopy())
        if s.state.CompareAndSwap(old, new) {
            return
        }
    }
}
```

## Handler Safety Rules

### DO: Safe Patterns

```go
// ✅ Reading state is safe (assuming thread-safe session)
value := state.GetScore()

// ✅ Writing state is safe (assuming thread-safe setters)
state.SetScore(100)

// ✅ Local variables are always safe
localVar := value * 2

// ✅ Context cancellation is safe
select {
case <-ctx.Done():
    return ctx.Err()
}

// ✅ Emitting events is safe (assuming thread-safe session)
session.Emit("ScoreChanged", nil)
```

### DON'T: Unsafe Patterns

```go
// ❌ Don't store session/state in global variables
var globalState *GameState  // UNSAFE

// ❌ Don't share data between handlers via closures
sharedCounter := 0
handler1 := func() { sharedCounter++ }  // RACE CONDITION
handler2 := func() { sharedCounter++ }  // RACE CONDITION

// ❌ Don't read-modify-write without synchronization
score := state.GetScore()
state.SetScore(score + 1)  // RACE if concurrent!

// Use atomic operations or transactions instead:
state.AddScore(1)  // Safe if implemented atomically
```

## Wait Nodes and Concurrency

Wait nodes interact with Go's runtime scheduler:

```go
// Wait node - yields goroutine
select {
case <-time.After(500 * time.Millisecond):
    // Other goroutines run during this wait
case <-ctx.Done():
    return ctx.Err()
}

// After wait, state may have changed!
// Always re-read state after waits if needed
newValue := state.GetValue()
```

### Wait Node Safety

| Aspect | Safe? | Notes |
|--------|-------|-------|
| Multiple handlers waiting | ✅ | Each has independent timer |
| State changes during wait | ⚠️ | Re-read state after wait |
| Cancellation during wait | ✅ | Context propagates cancel |

## Debug Hooks and Thread Safety

All debug hook implementations must be thread-safe:

```go
// ✅ Server uses mutex for client map
type Server struct {
    clients map[*websocket.Conn]bool
    mu      sync.RWMutex
}

// ✅ WriterHook uses mutex for writer
type WriterHook struct {
    w  io.Writer
    mu sync.Mutex
}

// ✅ MultiHook uses RWMutex for hook list
type MultiHook struct {
    hooks []DebugHook
    mu    sync.RWMutex
}
```

## Cancellation

### Handler Cancellation

```go
// Cancel a specific handler
tracked := execution.NewTrackedDispatcher()
handlerID := tracked.DispatchTracked(sessionID, "LongTask", nil)
// Later...
tracked.CancelHandler(handlerID)
```

### Session Cancellation

```go
// Cancel all handlers in a session (e.g., player disconnect)
tracked.CancelSession(sessionID)
```

### Graceful Shutdown

```go
// In generated handler with Wait nodes:
select {
case <-time.After(duration):
    // Normal completion
case <-ctx.Done():
    // Cleanup and exit
    return ctx.Err()
}
```

## Testing for Race Conditions

Use Go's race detector during development:

```bash
# Run tests with race detector
go test -race ./...

# Build with race detector
go build -race ./...
```

## Performance Considerations

### Lock Contention

High event rates may cause contention on:
- Session state mutex
- Dispatcher's handler map (read-mostly, low contention)
- Debug hook broadcast channel

### Recommendations

1. **Batch state updates** when possible
2. **Use read-locks** for read-only operations
3. **Keep critical sections small**
4. **Consider per-session dispatchers** for high-load scenarios

```go
// Per-session dispatcher reduces contention
type GameServer struct {
    sessions map[string]*SessionDispatcher
}

type SessionDispatcher struct {
    session    *TrackedSession
    dispatcher *execution.Dispatcher
}
```

## Summary

| Requirement | Implementation |
|-------------|----------------|
| State access | TrackedSession must be thread-safe |
| Event handlers | Run in separate goroutines |
| Wait nodes | Use context for cancellation |
| Debug hooks | All provided hooks are thread-safe |
| Testing | Use `-race` flag during development |
