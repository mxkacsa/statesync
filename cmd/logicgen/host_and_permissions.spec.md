# Host Player & Event Permissions Specification

## Overview

This document specifies:
1. **Host Player Tracking** - Session host management with auto-promotion
2. **Event Permissions** - Restrict who can trigger events
3. **Kick Player** - Remove players from session

---

## 1. Host Player Tracking

### Requirements

- Every session has a `hostPlayerID`
- Host is the first player to join (or explicitly set)
- When host leaves, next player is auto-promoted
- Host ID is synced to all clients with state

### statesync Changes

```go
type TrackedSession[S, M, P comparable] interface {
    // Existing
    State() StateAccessor[S]
    ID() string
    Emit(event string, data any)
    EmitTo(playerID P, event string, data any)

    // NEW: Host management
    HostPlayerID() P                    // Get current host
    SetHostPlayerID(playerID P)         // Manually set host
    IsHost(playerID P) bool             // Check if player is host

    // NEW: Player management
    Players() []P                       // List all players
    PlayerCount() int                   // Count players
    Kick(playerID P, reason string)     // Remove player

    // Called internally when player leaves
    OnPlayerLeave(playerID P)           // Auto-promotes host if needed
}
```

### Auto-Promotion Logic

```go
func (s *Session) OnPlayerLeave(playerID P) {
    // Remove player from session
    s.removePlayer(playerID)

    // If host left, promote next player
    if s.hostPlayerID == playerID {
        if len(s.players) > 0 {
            s.hostPlayerID = s.players[0]  // First remaining player
            s.Emit("HostChanged", map[string]any{
                "oldHost": playerID,
                "newHost": s.hostPlayerID,
            })
        } else {
            // No players left, session can be cleaned up
            s.hostPlayerID = ""  // Zero value
        }
    }
}
```

### Schema Integration

The hostPlayerID is a **system field** - automatically managed, not defined in schema.

```go
// statesync automatically adds to sync messages:
type SyncMessage struct {
    Schema       SchemaDefinition  `json:"schema"`
    HostPlayerID string            `json:"hostPlayerId"`  // NEW
    State        StateData         `json:"state"`
}

// On host change, broadcast:
type HostChangedMessage struct {
    Type      string `json:"type"`      // "host_changed"
    OldHostID string `json:"oldHostId"`
    NewHostID string `json:"newHostId"`
}
```

---

## 2. Event Permissions

### Permission Types

| Permission | Description |
|------------|-------------|
| `hostOnly` | Only host can trigger this event |
| `allowedPlayers` | Only specific players can trigger |
| `playerParam` | The event must have a playerID param, verified against sender |
| `anyone` | Default - any player can trigger |

### JSON Definition

```json
{
  "name": "OnStartGame",
  "event": "StartGame",
  "permissions": {
    "hostOnly": true
  },
  "parameters": [],
  "nodes": [...]
}

{
  "name": "OnPlayerMove",
  "event": "PlayerMove",
  "permissions": {
    "playerParam": "playerID"  // Verify sender matches this param
  },
  "parameters": [
    {"name": "playerID", "type": "string"},
    {"name": "x", "type": "float64"},
    {"name": "y", "type": "float64"}
  ],
  "nodes": [...]
}

{
  "name": "OnAdminCommand",
  "event": "AdminCommand",
  "permissions": {
    "allowedPlayers": ["admin1", "admin2"]  // Hardcoded list
  },
  "parameters": [],
  "nodes": [...]
}

{
  "name": "OnKickPlayer",
  "event": "KickPlayer",
  "permissions": {
    "hostOnly": true
  },
  "parameters": [
    {"name": "targetPlayerID", "type": "string"},
    {"name": "reason", "type": "string"}
  ],
  "nodes": [...]
}
```

### Generated Code

**hostOnly:**
```go
func OnStartGame(session *statesync.TrackedSession[*GameState, any, string], senderID string) error {
    // Permission check: hostOnly
    if !session.IsHost(senderID) {
        return ErrNotHost
    }

    state := session.State().Get()
    // ... rest of handler
}
```

**playerParam:**
```go
func OnPlayerMove(session *statesync.TrackedSession[*GameState, any, string], senderID string, playerID string, x, y float64) error {
    // Permission check: playerParam must match sender
    if playerID != senderID {
        return ErrNotAllowed
    }

    state := session.State().Get()
    // ... rest of handler
}
```

**allowedPlayers:**
```go
var _OnAdminCommand_allowedPlayers = map[string]bool{
    "admin1": true,
    "admin2": true,
}

func OnAdminCommand(session *statesync.TrackedSession[*GameState, any, string], senderID string) error {
    // Permission check: allowedPlayers
    if !_OnAdminCommand_allowedPlayers[senderID] {
        return ErrNotAllowed
    }

    state := session.State().Get()
    // ... rest of handler
}
```

### Handler Signature Change

All handlers now receive `senderID` as first parameter after session:

```go
// Before
func OnEvent(session *TrackedSession, params...) error

// After (with permissions)
func OnEvent(session *TrackedSession, senderID string, params...) error
```

---

## 3. Kick Player

### Node Type

```go
const NodeKickPlayer NodeType = "KickPlayer"  // Kick a player from session
```

### Node Definition

```go
NodeKickPlayer: {
    Type:        NodeKickPlayer,
    Category:    "Session",
    Description: "Kicks a player from the session",
    Inputs: []PortDefinition{
        {Name: "playerID", Type: "string", Required: true},
        {Name: "reason", Type: "string", Required: false, Default: ""},
    },
    Outputs: []PortDefinition{
        {Name: "kicked", Type: "bool", Required: true},
    },
},
```

### JSON Example

```json
{
  "id": "kickBadPlayer",
  "type": "KickPlayer",
  "inputs": {
    "playerID": "param:targetPlayerID",
    "reason": {"constant": "Cheating detected"}
  }
}
```

### Generated Code

```go
// Node: kickBadPlayer (KickPlayer)
kickBadPlayer_kicked := session.Kick(targetPlayerID, "Cheating detected")
```

### statesync Kick Implementation

```go
func (s *Session) Kick(playerID P, reason string) bool {
    if !s.hasPlayer(playerID) {
        return false
    }

    // Notify the kicked player
    s.EmitTo(playerID, "Kicked", map[string]any{
        "reason": reason,
    })

    // Remove player (triggers OnPlayerLeave -> host promotion if needed)
    s.OnPlayerLeave(playerID)

    // Disconnect their connection
    s.disconnectPlayer(playerID)

    // Notify others
    s.Emit("PlayerKicked", map[string]any{
        "playerID": playerID,
        "reason":   reason,
    })

    return true
}
```

---

## 4. Implementation Changes Summary

### logicgen/types.go

```go
// Add to EventHandler struct
type EventHandler struct {
    Name        string            `json:"name"`
    Event       string            `json:"event"`
    Permissions *EventPermissions `json:"permissions,omitempty"`  // NEW
    Parameters  []Parameter       `json:"parameters"`
    Nodes       []Node            `json:"nodes"`
    Flow        []FlowEdge        `json:"flow"`
}

type EventPermissions struct {
    HostOnly       bool     `json:"hostOnly,omitempty"`
    PlayerParam    string   `json:"playerParam,omitempty"`    // Param name to verify
    AllowedPlayers []string `json:"allowedPlayers,omitempty"` // Static list
}

// Add new node type
const NodeKickPlayer NodeType = "KickPlayer"
```

### logicgen/generator.go

```go
// generateHandler changes:
// 1. Always add senderID parameter after session
// 2. Generate permission checks at start of handler
// 3. Handle hostOnly, playerParam, allowedPlayers

func (g *CodeGenerator) generateHandler(handler *EventHandler) error {
    // ... existing signature generation ...

    // Always add senderID
    g.write(", senderID string")

    // ... rest of signature ...

    // Generate permission checks
    if handler.Permissions != nil {
        g.generatePermissionChecks(handler)
    }

    // ... rest of handler ...
}

func (g *CodeGenerator) generatePermissionChecks(handler *EventHandler) {
    perm := handler.Permissions

    if perm.HostOnly {
        g.writeLine("// Permission: hostOnly")
        g.writeLine("if !session.IsHost(senderID) {")
        g.indent++
        g.writeLine("return ErrNotHost")
        g.indent--
        g.writeLine("}")
        g.writeLine("")
    }

    if perm.PlayerParam != "" {
        g.writeLine("// Permission: playerParam")
        g.writeLine("if %s != senderID {", perm.PlayerParam)
        g.indent++
        g.writeLine("return ErrNotAllowed")
        g.indent--
        g.writeLine("}")
        g.writeLine("")
    }

    if len(perm.AllowedPlayers) > 0 {
        g.writeLine("// Permission: allowedPlayers")
        mapName := fmt.Sprintf("_%s_allowedPlayers", handler.Name)
        g.writeLine("if !%s[senderID] {", mapName)
        g.indent++
        g.writeLine("return ErrNotAllowed")
        g.indent--
        g.writeLine("}")
        g.writeLine("")
    }
}
```

### statesync Interface Requirements

```go
// TrackedSession must implement these methods:
type TrackedSession[S, M, P comparable] interface {
    // Host management
    HostPlayerID() P
    SetHostPlayerID(playerID P)
    IsHost(playerID P) bool

    // Player management
    Players() []P
    PlayerCount() int
    Kick(playerID P, reason string) bool
    OnPlayerLeave(playerID P)

    // ... existing methods ...
}
```

---

## 5. Example: Complete Kick Handler

### JSON Definition

```json
{
  "name": "OnKickPlayer",
  "event": "KickPlayer",
  "permissions": {
    "hostOnly": true
  },
  "parameters": [
    {"name": "targetPlayerID", "type": "string"},
    {"name": "reason", "type": "string"}
  ],
  "nodes": [
    {
      "id": "checkNotSelf",
      "type": "Compare",
      "inputs": {
        "left": "param:targetPlayerID",
        "op": {"constant": "!="},
        "right": "param:senderID"
      }
    },
    {
      "id": "ifNotSelf",
      "type": "If",
      "inputs": { "condition": "node:checkNotSelf:result" }
    },
    {
      "id": "doKick",
      "type": "KickPlayer",
      "inputs": {
        "playerID": "param:targetPlayerID",
        "reason": "param:reason"
      }
    },
    {
      "id": "emitKickFailed",
      "type": "EmitToPlayer",
      "inputs": {
        "playerID": "param:senderID",
        "event": {"constant": "KickFailed"},
        "data": {"constant": "Cannot kick yourself"}
      }
    }
  ],
  "flow": [
    { "from": "start", "to": "checkNotSelf" },
    { "from": "checkNotSelf", "to": "ifNotSelf" },
    { "from": "ifNotSelf", "to": "doKick", "label": "true" },
    { "from": "ifNotSelf", "to": "emitKickFailed", "label": "false" },
    { "from": "doKick", "to": "end" },
    { "from": "emitKickFailed", "to": "end" }
  ]
}
```

### Generated Code

```go
// Permission error types
var (
    ErrNotHost    = errors.New("only host can perform this action")
    ErrNotAllowed = errors.New("player not allowed to perform this action")
)

// OnKickPlayer handles the KickPlayer event
func OnKickPlayer(session *statesync.TrackedSession[*GameState, any, string], senderID string, targetPlayerID string, reason string) error {
    // Permission: hostOnly
    if !session.IsHost(senderID) {
        return ErrNotHost
    }

    state := session.State().Get()
    _ = state

    // Node: checkNotSelf (Compare)
    checkNotSelf_result := targetPlayerID != senderID

    // Node: ifNotSelf (If)
    if checkNotSelf_result {
        // Node: doKick (KickPlayer)
        doKick_kicked := session.Kick(targetPlayerID, reason)
        _ = doKick_kicked
    } else {
        // Node: emitKickFailed (EmitToPlayer)
        session.EmitTo(senderID, "KickFailed", "Cannot kick yourself")
    }

    return nil
}
```

---

## 6. Client-Side Integration

### JavaScript Example

```javascript
class GameSession {
    constructor(ws) {
        this.ws = ws;
        this.hostPlayerId = null;
        this.myPlayerId = null;
    }

    onMessage(msg) {
        switch (msg.type) {
            case "sync":
                this.hostPlayerId = msg.hostPlayerId;
                this.updateState(msg.state);
                break;

            case "host_changed":
                this.hostPlayerId = msg.newHostId;
                this.onHostChanged(msg.oldHostId, msg.newHostId);
                break;

            case "Kicked":
                this.onKicked(msg.reason);
                this.ws.close();
                break;

            case "PlayerKicked":
                this.onPlayerKicked(msg.playerID, msg.reason);
                break;
        }
    }

    isHost() {
        return this.myPlayerId === this.hostPlayerId;
    }

    // Only send if we're host (client-side check, server also validates)
    startGame() {
        if (!this.isHost()) {
            console.error("Only host can start game");
            return;
        }
        this.send({ event: "StartGame" });
    }

    kickPlayer(targetId, reason) {
        if (!this.isHost()) {
            console.error("Only host can kick");
            return;
        }
        this.send({
            event: "KickPlayer",
            targetPlayerID: targetId,
            reason: reason
        });
    }
}
```
