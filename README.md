# statesync

[![Go](https://github.com/mxkacsa/statesync/actions/workflows/ci.yml/badge.svg)](https://github.com/mxkacsa/statesync/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/mxkacsa/statesync/graph/badge.svg)](https://codecov.io/gh/mxkacsa/statesync)
[![Go Reference](https://pkg.go.dev/badge/github.com/mxkacsa/statesync.svg)](https://pkg.go.dev/github.com/mxkacsa/statesync)

Deterministic state synchronization for Go with high-performance binary encoding. Features automatic change tracking, reversible effects, player-specific filters, reconnection support, debounced broadcasts, events, and pipeline hooks.

```
┌────────────────────────────────────────────────────────────────────────────┐
│                              SERVER                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         State[T]                                    │   │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐   │   │
│  │  │  Base State  │──▶│   Effects    │──▶ │  Effective State     │   │   │
│  │  │  {round: 5}  │    │  [buff x2]   │    │  {round: 5, hp: 200} │   │   │
│  │  └──────────────┘    └──────────────┘    └──────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                       │
│                                    ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Session[T, ID]                                 │   │
│  │                                                                     │   │
│  │   ┌─────────────┐      ┌─────────────┐      ┌─────────────┐         │   │
│  │   │   Alice     │      │    Bob      │      │   Admin     │         │   │
│  │   │  filter     │      │  filter     │      │ (no filter) │         │   │
│  │   │ hides other │      │ hides other │      │ sees all    │         │   │
│  │   │   hands     │      │   hands     │      │             │         │   │
│  │   └──────┬──────┘      └──────┬──────┘      └──────┬──────┘         │   │
│  └──────────│─────────────────────│─────────────────────│──────────────┘   │
│             │                     │                     │                  │
│             ▼                     ▼                     ▼                  │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐          │
│  │  Binary Patch    │  │  Binary Patch    │  │  Binary Patch    │          │
│  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘          │
└───────────│─────────────────────│─────────────────────│────────────────────┘
            │                     │                     │
            ▼                     ▼                     ▼
┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐
│  Alice's Client   │  │   Bob's Client    │  │  Admin's Client   │
└───────────────────┘  └───────────────────┘  └───────────────────┘
```

## Features

- **Deterministic** - Same inputs, same outputs. Always.
- **Binary encoding** - High-performance compact binary protocol
- **Minimal memory** - One previous state, not per-session
- **Reversible effects** - Add/remove without mutating base state
- **Filters** - Each viewer sees customized data (hide secrets, filter by visibility)
- **Event system** - Fire-and-forget messages with targeting (all, one, except, many)
- **Pipeline hooks** - Intercept broadcast pipeline for logging, debugging, modification
- **Reconnection support** - History buffer for seamless client reconnection
- **Debounced broadcasts** - Batch rapid updates efficiently
- **Code generation** - `schemagen`, `trackgen`, `logicgen` tools
- **Node-based logic** - Visual scripting for server logic with `logicgen`
- **Fast** - ~20 byte patches vs ~200 byte JSON, sub-microsecond encoding

## Install

```bash
go get github.com/mxkacsa/statesync
```

## Quick Start

```go
// Define your state with tracking annotations
//go:generate schemagen -type=GameState -output=schema_gen.go
//go:generate trackgen -type=GameState -output=tracking_gen.go

type GameState struct {
    Round   int64  `track:"0"`
    Phase   string `track:"1"`
    Players []Player `track:"2" key:"ID"`
}

// Create tracked state
state := NewGameState() // Generated constructor
ts := statesync.NewTrackedState[*GameState, any](state, nil)

// Create session with binary encoding
session := statesync.NewTrackedSession[*GameState, any, string](ts)

// Connect clients
session.Connect("alice", nil)

// Update and broadcast binary diffs
diffs := session.ApplyUpdate(func(s **GameState) {
    (*s).Round++
    (*s).changes.Mark(0, statesync.OpReplace)
})

for clientID, data := range diffs {
    ws.SendBinary(clientID, data) // Much smaller than JSON!
}
```

## Debounced Broadcasts

Batch rapid updates to reduce network traffic:

```go
session.SetDebounce(50 * time.Millisecond)
session.SetBroadcastCallback(func(diffs map[string][]byte) {
    for clientID, data := range diffs {
        ws.Send(clientID, data)
    }
})

// Updates within 50ms window are batched
session.ScheduleBroadcast() // Called after each state change
```

## Reconnection Support

Handle client reconnections without full state resync:

```go
// Enable history (keep last 60 ticks for 20 tick/s = 3 seconds)
session.SetHistorySize(60)

// On tick, get sequence number
diffs, seq := session.TickWithSeq()
for id, data := range diffs {
    ws.Send(id, WithSeq(data, seq))
}

// Client reconnects with last known sequence
updates, isFull := session.Reconnect("alice", lastSeq, filter)
if isFull {
    // Client was disconnected too long, send full state
    ws.Send("alice", updates[0])
} else {
    // Send only missed updates
    for _, update := range updates {
        ws.Send("alice", update)
    }
}
```

## Event System

Events are fire-and-forget messages that don't persist in state. Use them for notifications, animations, sounds, toasts, etc.

```go
// Emit to all clients
session.Emit("GameStarted", map[string]any{"round": 1})

// Emit to specific client
session.EmitTo("alice", "YourTurn", nil)

// Emit to all except one (e.g., the sender)
session.EmitExcept("bob", "PlayerAction", map[string]any{"player": "bob", "action": "fold"})

// Emit to multiple specific clients
session.EmitToMany([]string{"alice", "charlie"}, "TeamMessage", "Go team!")
```

### Using TickWithEvents

```go
// Update state and emit events
session.State().Update(func(g *Game) {
    g.Phase = "playing"
})
session.Emit("PhaseChanged", map[string]any{"phase": "playing"})
session.EmitTo(currentPlayer, "YourTurn", nil)

// Get both state diffs and events
result := session.TickWithEvents()

// Send to clients
for clientID, diff := range result.Diffs {
    if diff != nil {
        ws.SendBinary(clientID, diff)
    }
}
for clientID, events := range result.Events {
    ws.SendBinary(clientID, events)
}
```

### Pre-encoded Events (Zero-copy)

For maximum performance, pre-encode payloads:

```go
// Create encoder once, reuse
enc := statesync.NewEventPayloadEncoder()
enc.WriteString("alice")
enc.WriteInt32(100)
enc.WriteBool(true)

// Emit raw bytes (no JSON marshaling)
session.EmitRaw(statesync.Event{
    Type:    "ScoreUpdate",
    Payload: enc.Bytes(),
})
enc.Reset() // Reuse for next event
```

## Code Generators

### schemagen - Schema Generator

Generate binary encoding schemas from struct definitions:

```bash
go install github.com/mxkacsa/statesync/cmd/schemagen@latest
```

```go
//go:generate schemagen -type=GameState -output=schema_gen.go

type GameState struct {
    Round   int64    `track:"0"`
    Phase   string   `track:"1"`
    Players []Player `track:"2" key:"ID"`
}
```

### trackgen - Tracking Code Generator

Generate `Trackable` interface implementations with automatic visibility filtering:

```bash
go install github.com/mxkacsa/statesync/cmd/trackgen@latest
```

```go
//go:generate trackgen -type=Player,GameState -output=tracking_gen.go

type Player struct {
    ID       string   `track:"0" identity:"true"`        // Marks this as the owner field
    Team     string   `track:"1" teamKey:"true"`         // Marks this as the team field
    Name     string   `track:"2"`                        // Public (default)
    Hand     []Card   `track:"3" visible:"self"`         // Only visible to self
    Position Point    `track:"4" visible:"team"`         // Visible to teammates
    Health   int      `track:"5" visible:"self|team"`    // Visible to self OR team
    Secret   string   `track:"6" visible:"private"`      // Never visible to clients
    MapData  []Item   `track:"7" visible:"hasMap"`       // Custom predicate
}
```

Generated code includes:
- `PlayerVisibilityCtx` - Context struct with ViewerID, ViewerTeam, and custom predicates
- `ShallowClone()` - Efficient cloning (only slices/maps are copied)
- `FilterFor(ctx)` - Returns filtered copy based on visibility rules
- `FilterPlayerSliceFor(items, ctx)` - Filters entire slice

### logicgen - Node-Based Logic Generator

Generate server logic from visual node graphs (build-time visual scripting):

```bash
go install github.com/mxkacsa/statesync/cmd/logicgen@latest
```

Create a node graph JSON file:

```json
{
  "version": "1.0",
  "package": "main",
  "handlers": [
    {
      "name": "OnCardPlayed",
      "event": "CardPlayed",
      "parameters": [
        {"name": "playerID", "type": "string"},
        {"name": "cardID", "type": "int32"}
      ],
      "nodes": [
        {
          "id": "getPlayer",
          "type": "GetPlayer",
          "inputs": {"playerID": "param:playerID"},
          "outputs": {"player": "player"}
        },
        {
          "id": "addPoints",
          "type": "Add",
          "inputs": {
            "a": "node:getPlayer:player.Score",
            "b": 10
          },
          "outputs": {"result": "newScore"}
        },
        {
          "id": "notify",
          "type": "EmitToAll",
          "inputs": {
            "eventType": {"constant": "ScoreUpdated"},
            "payload": {"score": "node:addPoints:newScore"}
          }
        }
      ],
      "flow": [
        {"from": "start", "to": "getPlayer"},
        {"from": "getPlayer", "to": "addPoints"},
        {"from": "addPoints", "to": "notify"}
      ]
    }
  ]
}
```

Generate Go code:

```bash
logicgen -input=game_logic.json -output=game_logic_gen.go
```

Node types include:
- **State Access**: GetField, SetField, GetPlayer
- **Array Operations**: AddToArray, RemoveFromArray, FilterArray, MapArray
- **Map Operations**: SetMapValue, GetMapValue, RemoveMapKey, FilterMap
- **Logic**: Compare, And, Or, Not, If, Switch
- **Math**: Add, Subtract, Multiply, Divide, Min, Max
- **Events**: EmitToAll, EmitToPlayer, EmitExcept
- **Control Flow**: ForEach, While, Break, Continue

See [cmd/logicgen/README.md](cmd/logicgen/README.md) for complete documentation.

## Visibility System

The visibility system provides compile-time generated filtering with ~86ns per entity.

### Visibility Tags

| Tag | Description |
|-----|-------------|
| `visible:"self"` | Only visible to owner (requires `identity:"true"` field) |
| `visible:"team"` | Only visible to same team (requires `teamKey:"true"` field) |
| `visible:"self\|team"` | Visible to owner OR teammates |
| `visible:"private"` | Never visible (server-only) |
| `visible:"public"` | Always visible (default) |
| `visible:"customName"` | Custom predicate function |

### Usage Example

```go
// Create visibility context
ctx := &PlayerVisibilityCtx{
    ViewerID:   "alice",
    ViewerTeam: "red",
    HasMap: func(viewerID string) bool {
        // Custom logic: does viewer have the map item?
        return inventory[viewerID].Has("map")
    },
}

// Filter single player
filtered := player.FilterFor(ctx)

// Filter all players
players := FilterPlayerSliceFor(gameState.Players, ctx)
```

### Performance

| Operation | Time | Allocs | Memory |
|-----------|------|--------|--------|
| FilterFor (opponent view) | **13 ns** | 0 | 0 B |
| FilterFor (self view) | **1.3 ns** | 0 | 0 B |
| FilterSlice100 (new alloc) | 3.4 µs | 1 | 13 KB |
| FilterSliceTo (reuse buffer) | **1.6 µs** | 0 | 0 B |
| FilterForInPlace | 13 ns | 0 | 0 B |

**Zero allocation filtering!** This is achieved by:
- Value return instead of pointer (enables stack allocation)
- Conditional slice cloning (only clone if data will be kept)
- Pre-allocated output buffer option for batch operations
- In-place modification for maximum performance

## Pipeline Hooks

Intercept the broadcast pipeline for logging, debugging, or modification:

```go
session.SetHooks(statesync.SessionHooks[*GameState, string]{
    // Called before filtering for each client
    OnBeforeFilter: func(clientID string, state *GameState) {
        log.Printf("filtering for %s", clientID)
    },

    // Called after filtering for each client
    OnAfterFilter: func(clientID string, filtered *GameState) {
        log.Printf("filtered state for %s", clientID)
    },

    // Called before encoding for each client
    OnBeforeEncode: func(clientID string, state *GameState) {
        metrics.RecordEncode(clientID)
    },

    // Called after encoding - can modify the bytes
    OnAfterEncode: func(clientID string, data []byte) []byte {
        // Add custom header, compress, encrypt, etc.
        return append([]byte{0x01}, data...) // Version prefix
    },

    // Called once before sending to all clients - can filter clients
    OnBeforeBroadcast: func(diffs map[string][]byte) map[string][]byte {
        // Remove spectators from critical updates
        delete(diffs, "spectator1")
        return diffs
    },

    // Called after broadcast completes
    OnAfterBroadcast: func(diffs map[string][]byte, seq uint64) {
        log.Printf("broadcast seq %d to %d clients", seq, len(diffs))
    },
})
```

## API

### TrackedState

```go
ts := statesync.NewTrackedState[T, A](initial, &statesync.TrackedConfig{
    Registry: registry, // Optional schema registry
})

ts.Get()                       // Current state with effects
ts.GetBase()                   // Without effects
ts.Update(func(s *T) {...})    // Modify
ts.Encode()                    // Binary diff
ts.EncodeAll()                 // Full binary state
ts.Commit()                    // Clear change tracking
```

### TrackedSession

```go
session := statesync.NewTrackedSession[T, A, string](trackedState)

session.Connect(id, filter)    // Connect with optional filter
session.Disconnect(id)
session.Tick()                 // Broadcast + commit
session.Broadcast()            // Get diffs without commit

// Client management
session.HasClient(id)          // Check if client exists
session.GetFilter(id)          // Get client's filter
session.SetFilter(id, filter)  // Update filter at runtime

// Pipeline hooks
session.SetHooks(hooks)        // Set pipeline callbacks

// Debouncing
session.SetDebounce(50 * time.Millisecond)
session.SetBroadcastCallback(fn)
session.ScheduleBroadcast()

// Reconnection
session.SetHistorySize(60)
session.TickWithSeq()
session.Reconnect(id, lastSeq, filter)
session.AckSeq(id, seq)

// Events
session.Emit(eventType, payload)           // To all
session.EmitTo(id, eventType, payload)     // To one
session.EmitExcept(id, eventType, payload) // To all except
session.EmitToMany(ids, eventType, payload)// To many
session.EmitRaw(event)                     // Pre-encoded
session.TickWithEvents()                   // Returns TickResult with Diffs + Events
```

### Effects

```go
// Create effect from function
state.AddEffect(statesync.Func("buff", func(s T, activator A) T {
    s.Damage *= 2
    return s
}), activator)

// Query & Remove
state.HasEffect("id")
state.GetEffect("id")
state.RemoveEffect("id")
state.CleanupExpired() // Removes effects implementing Expirable interface
```

Custom effect types (timed, conditional, toggle, stack) can be implemented
by satisfying the `Effect[T, A]` interface. Optionally implement `Expirable`
for automatic cleanup with `CleanupExpired()`, or `Schedulable` for timer-based expiration.

## Thread Safety

All types are safe for concurrent access from multiple goroutines.

- **TrackedState**: Internal `sync.RWMutex` protects all operations
- **TrackedSession**: Client map protected, safe concurrent access
- **Effects**: Internal mutexes protect mutable state
- **EventBuffer**: Atomic counter + mutex for safe concurrent emit/drain

### Safe Access Patterns

```go
// UNSAFE: Get() returns pointer that can race with Update()
state := ts.Get()
fmt.Println(state.Round) // Race if Update() called concurrently

// SAFE: Read() holds lock during callback
ts.Read(func(s *GameState) {
    fmt.Println(s.Round) // Protected
})

// SAFE: Update() holds lock during callback
ts.Update(func(s **GameState) {
    (*s).SetRound(5)
})
```

For single-writer patterns (one goroutine does all writes), `Get()` is safe.
For concurrent read/write from multiple goroutines, use `Read()`/`Update()`.

## Performance

| Operation | Time | Notes |
|-----------|------|-------|
| Diff cycle | <1μs | Binary encoding with change tracking |
| Message size | ~20 bytes | Compact binary format |
| FilterFor | ~13ns | Zero-allocation visibility filtering |

## Files

```
tracked_state.go   - State management with effects
tracked_session.go - Multi-client session + event emitter
effect.go          - Effect types (Timed, Toggle, Stack, etc.)
event.go           - Event system (emit, encode, decode)
encoder.go         - Binary encoder
decoder.go         - Binary decoder
schema.go          - Schema definitions
changeset.go       - Change tracking
persist.go         - Save/load

cmd/schemagen/     - Schema code generator
cmd/trackgen/      - Trackable code generator
cmd/logicgen/      - Node-based logic code generator
```

## License

MIT
