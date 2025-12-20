# Logicgen Update Specification

## Overview

This document specifies three major features for the logicgen code generator:
1. **Debug Tracing** - Real-time execution visualization for UI editor ✅ IMPLEMENTED
2. **Wait Nodes** - Node-level timing/delay capabilities ✅ IMPLEMENTED
3. **Event Concurrency** - Parallel event execution with goroutines ✅ IMPLEMENTED

---

## 1. Debug Tracing

### Purpose
Enable real-time debugging in the visual node editor by tracing execution flow, node inputs/outputs, timing, and state changes.

### Build-time Flag
Debug code is **only included** when built with `-tags debug` flag:

```bash
# Development build (with debug)
go build -tags debug ./...

# Production build (no debug overhead)
go build ./...
```

### Generated Code Structure

**Without debug flag** (production):
```go
func OnGameTick(session *statesync.TrackedSession[*GameState, any, string]) error {
    state := session.State().Get()

    // Node: getCurrentTime
    getCurrentTime_timestamp := time.Now().Unix()
    // Node: checkCondition
    checkCondition_result := getCurrentTime_timestamp > 1000
    // ...
    return nil
}
```

**With debug flag** (development):
```go
func OnGameTick(session *statesync.TrackedSession[*GameState, any, string], dbg DebugHook) error {
    state := session.State().Get()

    // Node: getCurrentTime
    if dbg != nil { dbg.OnNodeStart("getCurrentTime", "GetCurrentTime", nil) }
    getCurrentTime_timestamp := time.Now().Unix()
    if dbg != nil { dbg.OnNodeEnd("getCurrentTime", map[string]any{"timestamp": getCurrentTime_timestamp}) }

    // Node: checkCondition
    if dbg != nil { dbg.OnNodeStart("checkCondition", "Compare", map[string]any{"left": getCurrentTime_timestamp, "right": 1000}) }
    checkCondition_result := getCurrentTime_timestamp > 1000
    if dbg != nil { dbg.OnNodeEnd("checkCondition", map[string]any{"result": checkCondition_result}) }

    // ...
    return nil
}
```

### DebugHook Interface

```go
// debug_types.go (always present, but implementations only in debug builds)

type DebugHook interface {
    // Event lifecycle
    OnEventStart(handler string, params map[string]any)
    OnEventEnd(handler string, err error)

    // Node lifecycle
    OnNodeStart(nodeID, nodeType string, inputs map[string]any)
    OnNodeEnd(nodeID string, outputs map[string]any)
    OnNodeError(nodeID string, err error)

    // Wait/async
    OnNodeWait(nodeID string, duration time.Duration)
    OnNodeResume(nodeID string)
}

// NoopDebugHook for when debug is disabled
type NoopDebugHook struct{}
func (NoopDebugHook) OnEventStart(string, map[string]any) {}
func (NoopDebugHook) OnEventEnd(string, error) {}
// ... all methods are no-ops
```

### Debug Message Protocol (WebSocket)

Messages sent to UI editor:

```json
{
  "type": "event:start",
  "sessionId": "room-123",
  "handler": "OnGameTick",
  "params": {},
  "timestamp": 1703001234567
}

{
  "type": "node:start",
  "sessionId": "room-123",
  "handler": "OnGameTick",
  "nodeId": "getCurrentTime",
  "nodeType": "GetCurrentTime",
  "inputs": {},
  "timestamp": 1703001234568
}

{
  "type": "node:end",
  "sessionId": "room-123",
  "handler": "OnGameTick",
  "nodeId": "getCurrentTime",
  "outputs": {"timestamp": 1703001234},
  "durationMs": 0.1,
  "timestamp": 1703001234568
}

{
  "type": "node:wait",
  "sessionId": "room-123",
  "handler": "OnPlayerAttack",
  "nodeId": "waitBeforeHit",
  "durationMs": 5000,
  "resumeAt": 1703001239568,
  "timestamp": 1703001234568
}

{
  "type": "event:end",
  "sessionId": "room-123",
  "handler": "OnGameTick",
  "error": null,
  "durationMs": 1.5,
  "timestamp": 1703001234570
}
```

### Generator Changes

New flag for logicgen:
```bash
logicgen -input game.json -output game_gen.go -debug
```

This generates TWO files:
- `game_gen.go` - Production code (no debug)
- `game_gen_debug.go` - Debug-enabled code with build tag `//go:build debug`

---

## 2. Wait Nodes (Node-level Timing)

### Purpose
Allow nodes to pause execution for a duration, enabling timed game logic like cooldowns, delays, animations.

### Approach: Goroutine + Channel (Simple)

Since each event handler runs in its own goroutine, we can simply use `select` with timeout:

```go
// Node: waitBeforeDamage (Wait)
select {
case <-time.After(5 * time.Second):
    // Continue after wait
case <-ctx.Done():
    return ctx.Err() // Handler cancelled
}
```

### New Node Types

| Node Type | Inputs | Outputs | Description |
|-----------|--------|---------|-------------|
| `Wait` | `duration: int`, `unit: string` | - | Pauses execution for fixed time |
| `WaitUntil` | `condition: bool`, `checkInterval: int`, `timeout: int` | `timedOut: bool` | Polls condition until true |
| `WaitForEvent` | `eventType: string`, `timeout: int` | `received: bool`, `data: any` | Waits for specific event |
| `Timeout` | `duration: int`, `unit: string` | `timedOut: bool` | Wraps subsequent nodes with timeout |

### Wait Node JSON Example

```json
{
  "id": "cooldownWait",
  "type": "Wait",
  "inputs": {
    "duration": {"constant": 500},
    "unit": {"constant": "ms"}
  }
}
```

### Generated Code

```go
// Node: cooldownWait (Wait)
select {
case <-time.After(500 * time.Millisecond):
    // Wait completed
case <-ctx.Done():
    return ctx.Err()
}
```

### Execution Context

Handlers receive a context for cancellation:

```go
type ExecutionContext struct {
    context.Context
    Session   *statesync.TrackedSession[S, M, P]
    SessionID string
    Debug     DebugHook // nil in production
}

func OnPlayerAttack(ctx *ExecutionContext, playerID string) error {
    // ... can use ctx.Done() for cancellation
}
```

---

## 3. Event Concurrency

### Purpose
Allow multiple events to execute in parallel without blocking each other.

### Approach: Simple Goroutine per Event

Since `statesync.TrackedSession` is internally thread-safe, we simply spawn each event in its own goroutine:

```go
type EventDispatcher struct {
    handlers map[string]HandlerFunc
}

func (d *EventDispatcher) Dispatch(session *TrackedSession, event string, params map[string]any) {
    handler, ok := d.handlers[event]
    if !ok {
        return
    }

    // Each event runs in its own goroutine
    go func() {
        ctx := &ExecutionContext{
            Context:   context.Background(),
            Session:   session,
            SessionID: session.ID(),
        }

        if err := handler(ctx, params); err != nil {
            log.Printf("Event %s error: %v", event, err)
        }
    }()
}
```

### Characteristics

| Aspect | Behavior |
|--------|----------|
| Event isolation | Each event runs independently |
| Blocking | Events don't block each other |
| State access | Thread-safe via statesync |
| Wait nodes | Don't block other events |
| Cancellation | Via context.Context |

### Optional: Event Priority Queue

For cases where certain events should process first:

```go
type PriorityDispatcher struct {
    highPriority chan *EventRequest  // System events
    normalQueue  chan *EventRequest  // Game events
    workers      int
}

func (d *PriorityDispatcher) Start() {
    for i := 0; i < d.workers; i++ {
        go d.worker()
    }
}

func (d *PriorityDispatcher) worker() {
    for {
        select {
        case req := <-d.highPriority:
            d.process(req)
        default:
            select {
            case req := <-d.highPriority:
                d.process(req)
            case req := <-d.normalQueue:
                d.process(req)
            }
        }
    }
}
```

---

## Implementation Order

### Phase 1: Debug Tracing ✅ DONE
1. ✅ Add `DebugHook` interface to types (`debug/types.go`)
2. ✅ Modify generator to emit debug code
3. ✅ Add `-debug` flag to generator
4. ✅ Use build tags for conditional compilation (`//go:build debug`)
5. ✅ Create SSE debug server (`debug/server.go`) - uses Server-Sent Events (no external deps)

### Phase 2: Wait Nodes ✅ DONE
1. ✅ Add `ExecutionContext` type (`execution/context.go`)
2. ✅ Add Wait node types to definitions (`types.go`)
3. ✅ Implement `generateWait`, `generateWaitUntil`, `generateTimeout`
4. ✅ Update handler signatures to receive context (auto-detected)
5. ✅ Test with wait_test example

### Phase 3: Event Concurrency ✅ DONE
1. ✅ Create `EventDispatcher` (`execution/dispatcher.go`)
2. ✅ Add cancellation support via context.Context
3. ✅ TrackedDispatcher with handler tracking
4. ✅ Thread-safety documentation (`execution/THREAD_SAFETY.md`)

---

## File Structure After Implementation

```
cmd/logicgen/
├── main.go                    ✅ Updated with -debug flag
├── generator.go               ✅ Updated with Wait nodes + context + debug
├── types.go                   ✅ Added Wait/WaitUntil/Timeout node types
├── validator.go
├── schema_loader.go
├── debug/
│   ├── types.go               ✅ DebugHook interface + message types
│   ├── noop.go                ✅ NoopDebugHook
│   ├── hooks.go               ✅ ChannelHook, WriterHook, PrintHook, MultiHook
│   └── server.go              ✅ SSE server (build tag: debug)
├── execution/
│   ├── context.go             ✅ ExecutionContext with Wait helpers
│   ├── dispatcher.go          ✅ EventDispatcher + TrackedDispatcher
│   └── THREAD_SAFETY.md       ✅ Thread-safety documentation
├── testdata/
│   ├── chase_game.schema.json
│   ├── chase_game_logic.json
│   ├── chase_game_gen.go      ✅ Generated (no Wait nodes, no context)
│   ├── chase_game_gen_debug.go✅ Generated with debug hooks
│   ├── wait_test.schema.json  ✅ Test schema
│   ├── wait_test_logic.json   ✅ Test logic with Wait nodes
│   └── wait_test_gen.go       ✅ Generated with context parameter
└── logicgen_update.spec.md    ✅ This file
```
