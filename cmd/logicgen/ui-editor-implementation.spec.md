# Node-Based Logic Editor - Implementation Specification

## Overview

This document specifies how to implement a visual node-based editor compatible with the `logicgen` code generator. The editor produces JSON node graphs that `logicgen` transforms into Go code.

### Tool Ecosystem

The editor integrates with three code generators:

| Tool | Purpose | Input | Output |
|------|---------|-------|--------|
| **schemagen** | State schema definitions | `.schema` | Go + TypeScript types |
| **logicgen** | Event handler logic | `.json` | Go handlers |
| **envgen** | Runtime configuration | `.config` | Go + TypeScript config |

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Visual Editor (UI)                                 │
├────────────────────┬────────────────────┬────────────────────────────────────┤
│   Schema Editor    │   Logic Editor     │       Config Editor                │
│   (game.schema)    │   (logic.json)     │       (game.config)               │
├────────────────────┼────────────────────┼────────────────────────────────────┤
│     schemagen      │     logicgen       │         envgen                     │
├────────────────────┼────────────────────┼────────────────────────────────────┤
│  schema_gen.go     │  handlers_gen.go   │      config_gen.go                │
│  schema.ts         │                    │      config.ts                     │
└────────────────────┴────────────────────┴────────────────────────────────────┘
```

### Schema Roles

Schemas are categorized into two roles:

| Role | Annotation | Activatable | Use Case |
|------|------------|-------------|----------|
| **Helper** | `@helper` | No | Reusable types (Player, Item, etc.) |
| **Root** | `@root(active\|inactive)` | Yes | Game modes, optional features |

### Default Value Sources

Field defaults can come from three sources:

| Source | Syntax | Example |
|--------|--------|---------|
| **Literal** | `@default(value)` | `@default(100)`, `@default("lobby")` |
| **Config** | `@default(config:X.Y)` | `@default(config:GameConfig.Speed)` |
| **Zero** | (none) | Zero value for type |

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Visual Editor │────▶│   JSON Graph    │────▶│   Go Code       │
│   (UI)          │     │   (NodeGraph)   │     │   (Generated)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │
        ▼                       ▼                       ▼
   User interaction       logicgen input          Runtime code
```

## Data Format

### Root Structure: NodeGraph

```typescript
interface NodeGraph {
  version: string;           // Schema version (e.g., "1.0")
  package: string;           // Go package name for generated code
  imports?: string[];        // Additional Go imports
  filters?: FilterDefinition[];
  functions?: FunctionDefinition[];
  handlers: EventHandler[];
}
```

### EventHandler

```typescript
interface EventHandler {
  name: string;              // Go function name (e.g., "OnPlayerJoin")
  event: string;             // Event type (e.g., "PlayerJoin")
  permissions?: EventPermissions;
  parameters: Parameter[];
  nodes: Node[];
  flow: FlowEdge[];
}
```

### FunctionDefinition

```typescript
interface FunctionDefinition {
  name: string;              // Function name (e.g., "CalculateDamage")
  description?: string;
  parameters: Parameter[];
  returnType?: string;       // "int", "string", "bool", etc.
  nodes: Node[];
  flow: FlowEdge[];
}
```

### FilterDefinition

```typescript
interface FilterDefinition {
  name: string;              // Filter factory name
  description?: string;
  parameters: Parameter[];   // Filter parameters (e.g., viewerTeam)
  nodes: Node[];
  flow: FlowEdge[];
}
```

### Node

```typescript
interface Node {
  id: string;                // Unique within handler/function/filter
  type: string;              // Node type (e.g., "Compare", "SetField")
  inputs: Record<string, InputValue>;
  outputs?: Record<string, string>;

  // Editor-only metadata (ignored by logicgen)
  _position?: { x: number; y: number };
  _comment?: string;
}
```

### InputValue Types

```typescript
type InputValue =
  | string                           // Reference: "param:name", "node:id:output", "state:Path"
  | { constant: any }                // Constant value
  | { source: string }               // Alternative reference syntax
  | Record<string, InputValue>;      // Nested (for args, params maps)
```

### FlowEdge

```typescript
interface FlowEdge {
  from: string;              // "start" or node ID
  to: string;                // "end" or node ID
  label?: string;            // "true", "false", "body", "done" for control flow
  condition?: string;        // For conditional edges
}
```

### Parameter

```typescript
interface Parameter {
  name: string;
  type: string;              // "string", "int", "float64", "bool", "*PlayerType"
}
```

### EventPermissions

```typescript
interface EventPermissions {
  hostOnly?: boolean;        // Only session host can trigger
  playerParam?: string;      // Parameter that must match senderID
  allowedPlayers?: string[]; // Static list of allowed player IDs
}
```

---

## Node Categories and Definitions

### State Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `GetField` | `path: string` | `value: any` | Get value at state path |
| `SetField` | `path: string`, `value: any` | - | Set value at state path |
| `GetPlayer` | `playerID: string` | `player: *Player`, `found: bool` | Find player by ID |
| `GetCurrentState` | - | `state: *State` | Get current state reference |

**Path syntax:**
- Simple: `Score`, `Phase`
- Nested: `Player.Health`, `Game.Round`
- Array access: `Players[0]`, `Cards[idx]`
- Key lookup: `Players[playerID:ID]` (find where ID field equals playerID)

### Array Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `AddToArray` | `path: string`, `value: any` | - | Append to array at path |
| `RemoveFromArray` | `path: string`, `index: int` | - | Remove element by index |
| `FindInArray` | `path: string`, `field: string`, `value: any` | `index: int`, `found: bool` | Find element index |
| `ArrayLength` | `path: string` | `length: int` | Get array length |
| `ArrayAt` | `path: string`, `index: int` | `value: any` | Get element at index |
| `ForEachWhere` | `path: string`, `condition: bool` | `item: any`, `index: int` | Iterate with filter |

### Map Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `SetMapValue` | `path: string`, `key: string`, `value: any` | - | Set map entry |
| `GetMapValue` | `path: string`, `key: string` | `value: any`, `exists: bool` | Get map entry |
| `RemoveMapKey` | `path: string`, `key: string` | `removed: bool` | Remove map entry |
| `HasMapKey` | `path: string`, `key: string` | `exists: bool` | Check if key exists |

### Control Flow Nodes

| Type | Inputs | Outputs | Flow Labels | Description |
|------|--------|---------|-------------|-------------|
| `If` | `condition: bool` | - | `true`, `false` | Conditional branch |
| `ForEach` | `path: string` | `item: any`, `index: int` | `body`, `done` | Loop over array |
| `While` | `condition: bool` | - | `body`, `done` | While loop |

### Logic Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `Compare` | `left: any`, `right: any`, `op: string` | `result: bool` | Compare values (==, !=, <, >, <=, >=) |
| `And` | `a: bool`, `b: bool` | `result: bool` | Logical AND |
| `Or` | `a: bool`, `b: bool` | `result: bool` | Logical OR |
| `Not` | `value: bool` | `result: bool` | Logical NOT |
| `IsNull` | `value: any` | `result: bool` | Check if nil |
| `IsEmpty` | `value: any` | `result: bool` | Check if empty (string/array) |

### Math Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `Add` | `a: number`, `b: number` | `result: number` | Addition |
| `Subtract` | `a: number`, `b: number` | `result: number` | Subtraction |
| `Multiply` | `a: number`, `b: number` | `result: number` | Multiplication |
| `Divide` | `a: number`, `b: number` | `result: number` | Division |
| `Modulo` | `a: number`, `b: number` | `result: number` | Modulo |
| `Min` | `a: number`, `b: number` | `result: number` | Minimum |
| `Max` | `a: number`, `b: number` | `result: number` | Maximum |
| `RandomInt` | `min: int`, `max: int` | `result: int` | Random integer [min, max] |
| `RandomFloat` | `min: float`, `max: float` | `result: float` | Random float [min, max) |
| `RandomBool` | `probability: float` | `result: bool` | Random boolean |

### String Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `Concat` | `a: string`, `b: string` | `result: string` | Concatenate strings |
| `Format` | `format: string`, `args: map` | `result: string` | Printf-style format |
| `Contains` | `str: string`, `substr: string` | `result: bool` | Check substring |

### Event Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `EmitToAll` | `eventType: string`, `data: map` | - | Emit to all players |
| `EmitToPlayer` | `playerID: string`, `eventType: string`, `data: map` | - | Emit to one player |
| `EmitToMany` | `playerIDs: []string`, `eventType: string`, `data: map` | - | Emit to multiple players |

### Session Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `KickPlayer` | `playerID: string`, `reason: string` | - | Kick player from session |
| `GetHostPlayer` | - | `hostID: string` | Get host player ID |
| `IsHost` | `playerID: string` | `result: bool` | Check if player is host |

### Filter Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `AddFilter` | `viewerID: string`, `filterID: string`, `filterName: string`, `params: map` | - | Add filter for viewer |
| `RemoveFilter` | `viewerID: string`, `filterID: string` | `removed: bool` | Remove filter |
| `HasFilter` | `viewerID: string`, `filterID: string` | `exists: bool` | Check filter exists |

### Effect Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `AddEffect` | `effectId: string`, `effectName: string`, `activator: any`, `params: map` | - | Add effect to session |
| `RemoveEffect` | `effectId: string` | `removed: bool` | Remove effect |
| `HasEffect` | `effectId: string` | `exists: bool` | Check effect exists |

### Function Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `CallFunction` | `function: string`, `args: map` | `result: any` | Call defined function |
| `Return` | `value?: any` | - | Return from function (value required if function has returnType) |

**Important Function Rules:**
- `CallFunction.result` output type matches the called function's `returnType`
- If function has no `returnType`, `CallFunction` has no `result` output
- `Return.value` is required when the containing function has a `returnType`
- `Return.value` should match the function's declared `returnType`

### Async Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `Wait` | `duration: int` | - | Wait milliseconds |
| `WaitUntil` | `condition: bool`, `timeout: int` | `completed: bool` | Wait for condition |
| `Timeout` | `duration: int` | - | Set timeout context |

### GPS/Geometry Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `GpsDistance` | `lat1, lon1, lat2, lon2: float` | `distance: float` | Distance in meters |
| `GpsMoveToward` | `fromLat, fromLon, toLat, toLon, distance: float` | `newLat, newLon: float` | Move toward point |
| `PointInCircle` | `pointLat, pointLon, centerLat, centerLon, radius: float` | `inside: bool` | Check if in circle |

### Variable Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `SetVariable` | `name: string`, `value: any` | - | Set local variable |
| `GetVariable` | `name: string` | `value: any` | Get local variable |
| `Constant` | `value: any`, `type: string` | `value: any` | Constant value |

### Time Nodes

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `GetCurrentTime` | - | `timestamp: int64` | Current Unix timestamp (ms) |
| `TimeSince` | `timestamp: int64` | `elapsed: int64` | Time since timestamp (ms) |

---

## Function System Details

### Function Definition Structure

Functions are reusable node-based logic that can be called from handlers or other functions. They support:

1. **Parameters** - Input values passed when calling the function
2. **Return Type** - Optional output value type
3. **Local Scope** - Parameters accessible via `param:` prefix within function nodes

### Function Editor Tab

The editor should have separate tabs/panels for:
- **Handlers** - Event-triggered logic
- **Functions** - Reusable logic blocks
- **Filters** - State transformation logic

```
┌─────────────────────────────────────────────────────────────────┐
│ [Handlers ▼] [Functions] [Filters]           [+ New Function]  │
├─────────────────────────────────────────────────────────────────┤
│ Function: GetPlayerHighestCard                                  │
│ Parameters: playerID (string)                                   │
│ Returns: int                                                    │
├─────────────────────────────────────────────────────────────────┤
│                        ┌────────────────┐                       │
│  [Start] ─────────────▶│   GetPlayer    │                       │
│                        │ playerID: ●────┼── param:playerID      │
│                        │         player ○┼─────────┐            │
│                        └────────────────┘          │            │
│                                                    ▼            │
│                        ┌────────────────┐   ┌────────────┐      │
│                        │     Max        │   │  ForEach   │      │
│                        │     a: ●───────┼───┼─ item      │      │
│                        │     b: ●       │   │ Hand: ●────┼──────│
│                        │  result ○──────┼───┼───────────▶│      │
│                        └────────────────┘   └────────────┘      │
│                               │                                  │
│                               ▼                                  │
│                        ┌────────────────┐                       │
│                        │    Return      │                       │
│                        │  value: ●──────┼── node:max:result     │
│                        └────────────────┘                       │
│                               │                                  │
│                               ▼                                  │
│                            [End]                                │
└─────────────────────────────────────────────────────────────────┘
```

### CallFunction Node Visualization

The `CallFunction` node should dynamically display argument inputs based on the selected function:

```
┌─────────────────────────────────────────┐
│ [⚡] CallFunction                        │
├─────────────────────────────────────────┤
│ function: [GetPlayerHighestCard ▼]      │  ← Dropdown with available functions
├─────────────────────────────────────────┤
│ Arguments:                              │
│   ● playerID ─────── param:player1ID    │  ← Dynamic based on function params
├─────────────────────────────────────────┤
│                           result ○──────│  ← Only if function has returnType
│                           (int)         │  ← Type shown from function definition
└─────────────────────────────────────────┘
```

**Dynamic Behavior:**
1. When `function` input changes, regenerate argument inputs based on function definition
2. Show return type in output port if function has `returnType`
3. Hide result output if function has no `returnType` (void function)

### Return Node in Functions

```
┌─────────────────────────────────────────┐
│ [↩] Return                              │
├─────────────────────────────────────────┤
│   ● value ─────── node:max:result       │  ← Required if function has returnType
│     (int)                               │  ← Type from function's returnType
└─────────────────────────────────────────┘
```

**Validation Rules:**
- If function has `returnType` and `Return` node has no `value`: ERROR
- If function has `returnType` but no `Return` node in flow: WARNING
- If function has no `returnType` but `Return` has `value`: WARNING (ignored)

### Function Call Flow Example

```
Handler: OnCompareHighestCards
Parameters: player1ID (string), player2ID (string)

  [Start]
     │
     ▼
┌────────────────────────┐
│ CallFunction           │
│ function: GetPlayer... │
│ args.playerID: ●───────┼── param:player1ID
│              result ○──┼─────────────────────────────┐
└────────────────────────┘                             │
     │                                                  │
     ▼                                                  │
┌────────────────────────┐                             │
│ CallFunction           │                             │
│ function: GetPlayer... │                             │
│ args.playerID: ●───────┼── param:player2ID          │
│              result ○──┼──────────────────────┐     │
└────────────────────────┘                      │     │
     │                                           │     │
     ▼                                           ▼     ▼
┌────────────────────────┐  ┌────────────────────────────────┐
│ Compare                │  │ Both results used for comparison│
│ left:  ●───────────────┼──┘                                │
│ right: ●───────────────┼────────────────────────────────────┘
│ op:    ●─── ">"        │
│      result ○──────────┼─────▶
└────────────────────────┘
```

---

## Visual Editor Requirements

### Canvas

1. **Infinite canvas** with pan and zoom
2. **Grid snapping** for node alignment
3. **Minimap** for navigation in large graphs
4. **Multi-select** for group operations

### Nodes

1. **Visual representation:**
   ```
   ┌─────────────────────────────────┐
   │ [Icon] NodeType                 │  ← Header with category color
   ├─────────────────────────────────┤
   │ ● input1 (required)       out1 ○│  ← Input/Output ports
   │ ○ input2 (optional)       out2 ○│
   │ ○ input3: [inline value]        │  ← Inline constant editing
   └─────────────────────────────────┘
   ```

2. **Port colors by type:**
   - `bool` - Red
   - `int/float` - Blue
   - `string` - Green
   - `any` - Gray
   - `flow` - White (execution flow)

3. **Category colors:**
   - State - Purple
   - Array - Orange
   - Control Flow - Yellow
   - Logic - Red
   - Math - Blue
   - Events - Green
   - Session - Teal
   - Effects - Pink

### Connections

1. **Data connections** (colored by type)
2. **Flow connections** (white/gray, thicker)
3. **Validation indicators:**
   - Valid connection: solid line
   - Type mismatch warning: dashed orange
   - Missing required: red highlight on port

### Node Palette

1. **Searchable** node list
2. **Category grouping** with collapsible sections
3. **Drag-and-drop** to canvas
4. **Quick-add** via right-click context menu
5. **Recent nodes** section

### Properties Panel

1. **Node properties** when node selected
2. **Input editing:**
   - Reference picker (parameters, node outputs, state paths)
   - Constant value editor (with type-appropriate UI)
   - Expression builder for complex values
3. **Handler/Function properties** when canvas background selected

### Toolbar

1. **File operations:** New, Open, Save, Export
2. **Edit operations:** Undo, Redo, Cut, Copy, Paste, Delete
3. **View operations:** Zoom, Fit to Screen, Toggle Grid
4. **Validate:** Check graph for errors
5. **Generate:** Preview/generate Go code

---

## Reference System

### Reference Syntax

```
param:<parameterName>           → Handler/function parameter
node:<nodeId>:<outputName>      → Output from another node
state:<path>                    → State field access
loop:item                       → Current ForEach item
loop:index                      → Current ForEach index
```

### Reference Picker UI

```
┌─────────────────────────────────────┐
│ 🔍 Search references...             │
├─────────────────────────────────────┤
│ ▼ Parameters                        │
│   ○ playerID (string)               │
│   ○ amount (int)                    │
│ ▼ Node Outputs                      │
│   ○ checkScore → result (bool)      │
│   ○ getPlayer → player (*Player)    │
│ ▼ State Fields                      │
│   ○ Phase (string)                  │
│   ○ Players ([]Player)              │
│ ▼ Loop Context                      │
│   ○ item (Player)                   │
│   ○ index (int)                     │
└─────────────────────────────────────┘
```

---

## Validation Rules

### Compile-time Validation (Editor)

1. **Unique node IDs** within each handler/function/filter
2. **All required inputs connected** or have inline values
3. **Type compatibility** between connected ports
4. **No cycles** in data flow (except loops)
5. **Flow graph connected** - all nodes reachable from start
6. **Valid references** - referenced nodes/params exist
7. **Handler names valid** Go identifiers

### Warnings (Non-blocking)

1. **Unused node outputs** - generated but not consumed
2. **Unreachable nodes** - not in any flow path
3. **Potential nil access** - GetPlayer result used without found check
4. **Type coercion** - implicit conversions

### Error Display

```
┌─────────────────────────────────────────────┐
│ ⚠ Validation Errors                    [x]  │
├─────────────────────────────────────────────┤
│ ❌ Node "checkTeam": missing required input │
│    "left" is not connected                  │
│    [Go to node]                             │
├─────────────────────────────────────────────┤
│ ⚠ Node "unused": output "result" unused     │
│    Consider connecting or removing          │
│    [Go to node] [Delete node]               │
└─────────────────────────────────────────────┘
```

---

## State Path Builder

### Schema Integration

Editor should load the schema (from schemagen) to provide:
1. **Autocomplete** for state paths
2. **Type information** for fields
3. **Validation** of path expressions
4. **Schema activation state** - which root schemas are active
5. **Default values** - field defaults for reset operations

### Path Builder UI

```
┌─────────────────────────────────────┐
│ State Path Builder                  │
├─────────────────────────────────────┤
│ Players [ playerID : ID ] . Score   │
│ └─array─┘ └───key lookup──┘ └field─┘│
├─────────────────────────────────────┤
│ Available fields:                   │
│  ○ Score (int)                      │
│  ○ Health (int)                     │
│  ○ Position (Position)              │
│  ○ Inventory ([]Item)               │
└─────────────────────────────────────┘
```

---

## Flow Editor

### Special Flow Labels

| Control Node | Label | Meaning |
|--------------|-------|---------|
| `If` | `true` | Condition is true |
| `If` | `false` | Condition is false |
| `ForEach` | `body` | Loop body (per iteration) |
| `ForEach` | `done` | After loop completes |
| `While` | `body` | Loop body |
| `While` | `done` | After loop exits |

### Flow Visualization

```
[start] ──────────────────────────────────────────▶ [end]
           │                                          ▲
           ▼                                          │
    ┌─────────────┐                                   │
    │ checkScore  │                                   │
    └─────────────┘                                   │
           │                                          │
           ▼                                          │
    ┌─────────────┐  true   ┌─────────────┐          │
    │    If       │────────▶│  addPoints  │──────────┘
    └─────────────┘         └─────────────┘
           │ false
           ▼
    ┌─────────────┐
    │  logFailure │──────────────────────────────────▶
    └─────────────┘
```

---

## Code Preview

### Generated Code Panel

```go
// Preview: OnScoreUpdate
func (h *Handlers) OnScoreUpdate(
    ctx context.Context,
    session *statesync.TrackedSession[*GameState],
    senderID string,
    playerID string,
    points int,
) error {
    state := session.State()

    // Node: checkScore
    checkScore_result := points > 0

    // Node: if
    if checkScore_result {
        // Node: addPoints
        // ... (highlighted when hovering node in editor)
    }

    return nil
}
```

### Bidirectional Highlighting

1. Hover node in editor → highlight corresponding code
2. Click code line → select corresponding node
3. Errors in code → show on originating node

---

## Import/Export

### File Format

```json
{
  "version": "1.0",
  "package": "game",
  "handlers": [...],
  "functions": [...],
  "filters": [...],
  "_editor": {
    "viewport": { "x": 0, "y": 0, "zoom": 1.0 },
    "selectedNodes": [],
    "expandedCategories": ["State", "Logic"]
  }
}
```

### Export Options

1. **JSON only** - Node graph for version control
2. **JSON + Go** - Both graph and generated code
3. **Go only** - Just the generated code (for CI/CD)

### CLI Integration

```bash
# Generate from JSON
logicgen -input logic.json -output handlers.go -schema schema.go

# Validate only
logicgen -input logic.json -validate

# Watch mode for development
logicgen -input logic.json -output handlers.go -watch
```

---

## Keyboard Shortcuts

| Action | Shortcut |
|--------|----------|
| Delete selected | `Delete` / `Backspace` |
| Copy | `Ctrl+C` |
| Paste | `Ctrl+V` |
| Undo | `Ctrl+Z` |
| Redo | `Ctrl+Shift+Z` |
| Select all | `Ctrl+A` |
| Deselect | `Escape` |
| Search nodes | `Ctrl+F` |
| Add node | `Space` or `Tab` |
| Duplicate | `Ctrl+D` |
| Group | `Ctrl+G` |
| Fit view | `F` |
| Zoom in | `Ctrl++` or scroll |
| Zoom out | `Ctrl+-` or scroll |
| Pan | `Middle mouse` or `Space+drag` |
| Validate | `Ctrl+Shift+V` |
| Generate | `Ctrl+Shift+G` |

---

## Example Node Graph JSON

```json
{
  "version": "1.0",
  "package": "game",
  "functions": [
    {
      "name": "CalculateDamage",
      "description": "Calculate damage based on attack and defense",
      "parameters": [
        { "name": "attackPower", "type": "int" },
        { "name": "defense", "type": "int" }
      ],
      "returnType": "int",
      "nodes": [
        {
          "id": "subtract",
          "type": "Subtract",
          "inputs": {
            "a": "param:attackPower",
            "b": "param:defense"
          }
        },
        {
          "id": "clamp",
          "type": "Max",
          "inputs": {
            "a": "node:subtract:result",
            "b": { "constant": 0 }
          }
        },
        {
          "id": "return",
          "type": "Return",
          "inputs": {
            "value": "node:clamp:result"
          }
        }
      ],
      "flow": [
        { "from": "start", "to": "subtract" },
        { "from": "subtract", "to": "clamp" },
        { "from": "clamp", "to": "return" }
      ]
    }
  ],
  "handlers": [
    {
      "name": "OnPlayerAttack",
      "event": "PlayerAttack",
      "permissions": {
        "playerParam": "attackerID"
      },
      "parameters": [
        { "name": "attackerID", "type": "string" },
        { "name": "targetID", "type": "string" }
      ],
      "nodes": [
        {
          "id": "getAttacker",
          "type": "GetPlayer",
          "inputs": {
            "playerID": "param:attackerID"
          }
        },
        {
          "id": "getTarget",
          "type": "GetPlayer",
          "inputs": {
            "playerID": "param:targetID"
          }
        },
        {
          "id": "checkFound",
          "type": "And",
          "inputs": {
            "a": "node:getAttacker:found",
            "b": "node:getTarget:found"
          }
        },
        {
          "id": "ifFound",
          "type": "If",
          "inputs": {
            "condition": "node:checkFound:result"
          }
        },
        {
          "id": "calcDamage",
          "type": "CallFunction",
          "inputs": {
            "function": { "constant": "CalculateDamage" },
            "args": {
              "attackPower": { "constant": 10 },
              "defense": { "constant": 3 }
            }
          }
        },
        {
          "id": "applyDamage",
          "type": "SetField",
          "inputs": {
            "path": { "constant": "Players[param:targetID:ID].Health" },
            "value": "node:calcDamage:result"
          }
        }
      ],
      "flow": [
        { "from": "start", "to": "getAttacker" },
        { "from": "getAttacker", "to": "getTarget" },
        { "from": "getTarget", "to": "checkFound" },
        { "from": "checkFound", "to": "ifFound" },
        { "from": "ifFound", "to": "calcDamage", "label": "true" },
        { "from": "calcDamage", "to": "applyDamage" },
        { "from": "applyDamage", "to": "end" },
        { "from": "ifFound", "to": "end", "label": "false" }
      ]
    }
  ],
  "filters": [
    {
      "name": "HideEnemyLocations",
      "description": "Hides location data for players on other teams",
      "parameters": [
        { "name": "viewerTeam", "type": "string" }
      ],
      "nodes": [
        {
          "id": "loop",
          "type": "ForEach",
          "inputs": {
            "path": { "constant": "Players" }
          }
        },
        {
          "id": "checkTeam",
          "type": "Compare",
          "inputs": {
            "left": "loop:item.Team",
            "right": "param:viewerTeam",
            "op": { "constant": "!=" }
          }
        },
        {
          "id": "ifEnemy",
          "type": "If",
          "inputs": {
            "condition": "node:checkTeam:result"
          }
        },
        {
          "id": "hideLocation",
          "type": "SetField",
          "inputs": {
            "path": "loop:item.Location",
            "value": { "constant": null }
          }
        }
      ],
      "flow": [
        { "from": "start", "to": "loop" },
        { "from": "loop", "to": "checkTeam", "label": "body" },
        { "from": "checkTeam", "to": "ifEnemy" },
        { "from": "ifEnemy", "to": "hideLocation", "label": "true" },
        { "from": "hideLocation", "to": "loop" },
        { "from": "ifEnemy", "to": "loop", "label": "false" },
        { "from": "loop", "to": "end", "label": "done" }
      ]
    }
  ]
}
```

---

## Schema & Config Management

### Schema Types (schemagen)

The editor integrates with schemagen which now supports two schema roles:

#### Helper Schemas
Helper schemas are used by other schemas but cannot be activated independently:

```schema
@id(2) @helper
type Player {
    ID      string
    Name    string      @default("")
    Score   int64       @default(0)
    Ready   bool        @default(false)
}
```

#### Root Schemas (Activatable)
Root schemas represent game modes that can be activated/deactivated at runtime:

```schema
// Active by default
@id(1) @root(active)
type GameState {
    Round       int32       @default(1)
    Phase       string      @default("lobby")
    Players     []Player    @key(ID)
    SpeedMult   float64     @default(config:GameConfig.Speed)
}

// Inactive by default (optional game mode)
@id(3) @root(inactive)
type DroneMode {
    Drones          []Drone     @key(ID)
    SpawnInterval   int32       @default(5)
    MaxDrones       int32       @default(10)
}
```

### Default Value Types

Fields support three default value sources:

| Source | Syntax | Example | Description |
|--------|--------|---------|-------------|
| **Literal** | `@default(value)` | `@default(100)` | Hardcoded value |
| **Empty** | `@default("")` | `@default("")` | Empty string/zero |
| **Config** | `@default(config:X.Y)` | `@default(config:GameConfig.Speed)` | From envgen config |

### Schema Activation Nodes

New nodes for managing schema activation:

| Type | Inputs | Outputs | Description |
|------|--------|---------|-------------|
| `ActivateSchema` | `schemaName: string` | `success: bool` | Activate a root schema (resets to defaults) |
| `DeactivateSchema` | `schemaName: string` | `success: bool` | Deactivate a root schema |
| `IsSchemaActive` | `schemaName: string` | `active: bool` | Check if schema is active |
| `ResetSchema` | `schemaName: string` | - | Reset active schema to defaults |

### Schema Activation UI

```
┌─────────────────────────────────────────────────────────────────┐
│ Schema Manager                                           [⚙]   │
├─────────────────────────────────────────────────────────────────┤
│ Root Schemas:                                                   │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ ● GameState            [Active]      [Deactivate] [Reset]  │ │
│ │   Default: active                                          │ │
│ ├─────────────────────────────────────────────────────────────┤ │
│ │ ○ DroneMode            [Inactive]    [Activate]            │ │
│ │   Default: inactive                                        │ │
│ ├─────────────────────────────────────────────────────────────┤ │
│ │ ○ TournamentMode       [Inactive]    [Activate]            │ │
│ │   Default: inactive                                        │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ Helper Schemas: Player, Drone, Item (non-activatable)          │
└─────────────────────────────────────────────────────────────────┘
```

### Config Integration (envgen)

The editor can load config definitions from envgen:

```config
package main

config GameConfig {
    Speed           float64     @default(1.0)    @min(0.1) @max(10.0)
    MaxPlayers      int32       @default(4)      @min(2)   @max(16)
    EnableDebug     bool        @default(false)
    GameMode        string      @default("classic") @options("classic","ranked","casual")
}
```

### Config Editor Panel

```
┌─────────────────────────────────────────────────────────────────┐
│ Config: GameConfig                              [Save] [Reset]  │
├─────────────────────────────────────────────────────────────────┤
│ Speed           [━━━━━━━●━━━━━] 1.5    (0.1 - 10.0)            │
│ MaxPlayers      [━━●━━━━━━━━━━] 4      (2 - 16)                │
│ EnableDebug     [○ Off  ● On ]                                 │
│ GameMode        [classic ▼]    (classic, ranked, casual)       │
│ RoundDuration   [━━━━━━●━━━━━] 60      (10 - 600)              │
└─────────────────────────────────────────────────────────────────┘
```

### Reference Picker with Config

The reference picker now includes config values:

```
┌─────────────────────────────────────┐
│ 🔍 Search references...             │
├─────────────────────────────────────┤
│ ▼ Parameters                        │
│   ○ playerID (string)               │
│ ▼ Node Outputs                      │
│   ○ getPlayer → result (Player)     │
│ ▼ State Fields                      │
│   ○ GameState.Phase (string)        │
│   ○ GameState.Round (int)           │
│ ▼ Config Values                     │  ← NEW
│   ○ GameConfig.Speed (float64)      │
│   ○ GameConfig.MaxPlayers (int32)   │
│   ○ GameConfig.EnableDebug (bool)   │
└─────────────────────────────────────┘
```

### Generated TypeScript Types

The schema generates TypeScript with activation support:

```typescript
// Schema types
export type SchemaName = 'GameState' | 'DroneMode' | 'TournamentMode';

// Activation manager
export function getSchemaManager(): SchemaActivationManager;
export function isGameStateActive(): boolean;
export function activateGameState(): void;
export function deactivateGameState(): void;
export function getGameState(): SyncState<GameState> | null;

// Defaults
export function defaultGameState(): GameState;
export function defaultDroneMode(): DroneMode;
```

### Generated Go Code

```go
// Schema registry
func GetSchemaRegistry() *SchemaRegistry
func ActivateGameState() error
func DeactivateGameState() error
func IsGameStateActive() bool
func GetGameStateInstance() *GameState

// Reset to defaults
func (t *GameState) ResetToDefaults()
```

---

## Technology Recommendations

### Frontend Framework
- **React** with TypeScript for component architecture
- **React Flow** or **rete.js** for node graph canvas
- **Zustand** or **Redux** for state management
- **Monaco Editor** for code preview (same as VS Code)

### Persistence
- **Local**: IndexedDB for browser storage
- **Cloud**: REST API to save/load projects
- **Version Control**: Export as JSON for Git

### Schema Loading
- Load Go schema file via API endpoint
- Parse struct definitions for autocomplete
- Cache schema in browser storage

---

## Implementation Phases

### Phase 1: Core Editor (4-6 weeks)
- [ ] Canvas with pan/zoom
- [ ] Node rendering
- [ ] Connection drawing
- [ ] Basic node palette
- [ ] Properties panel
- [ ] JSON import/export

### Phase 2: Validation & Generation (2-3 weeks)
- [ ] Real-time validation
- [ ] Error highlighting
- [ ] Code preview panel
- [ ] logicgen CLI integration

### Phase 3: Advanced Features (3-4 weeks)
- [ ] Schema integration
- [ ] Path builder
- [ ] Reference picker
- [ ] Undo/redo
- [ ] Keyboard shortcuts

### Phase 4: Polish (2-3 weeks)
- [ ] Dark/light themes
- [ ] Node grouping
- [ ] Comments
- [ ] Minimap
- [ ] Performance optimization

---

## API Endpoints (Backend)

```
POST /api/validate
  Request: { nodeGraph: NodeGraph }
  Response: { valid: boolean, errors: ValidationError[] }

POST /api/generate
  Request: { nodeGraph: NodeGraph, schema?: string }
  Response: { code: string, errors: GenerationError[] }

GET /api/schema
  Response: { types: TypeDefinition[], fields: FieldDefinition[] }

POST /api/save
  Request: { projectId: string, nodeGraph: NodeGraph }
  Response: { success: boolean }

GET /api/load/:projectId
  Response: { nodeGraph: NodeGraph }
```

---

## Appendix: All Node Types

### Quick Reference

| Category | Nodes |
|----------|-------|
| State | GetField, SetField, GetPlayer, GetCurrentState |
| Array | AddToArray, RemoveFromArray, FindInArray, ArrayLength, ArrayAt, FilterArray, MapArray, ForEachWhere, UpdateWhere, FindWhere, CountWhere |
| Map | SetMapValue, GetMapValue, RemoveMapKey, HasMapKey, MapKeys, MapValues, FilterMap |
| Control Flow | If, ForEach, While, Break, Continue |
| Logic | Compare, And, Or, Not, IsNull, IsEmpty |
| Math | Add, Subtract, Multiply, Divide, Modulo, Min, Max, Sqrt, Pow, Abs, Sin, Cos, Atan2, Round, Floor, Ceil |
| Random | RandomInt, RandomFloat, RandomBool |
| String | Concat, Format, Contains, Split, ToUpper, ToLower, Trim |
| Events | EmitToAll, EmitToPlayer, EmitToMany |
| Session | KickPlayer, GetHostPlayer, IsHost |
| Filters | AddFilter, RemoveFilter, HasFilter |
| Effects | AddEffect, RemoveEffect, HasEffect |
| Functions | CallFunction, Return |
| Variables | SetVariable, GetVariable, Constant |
| Object | CreateStruct, UpdateStruct, GetStructField |
| Time | GetCurrentTime, TimeSince |
| Async | Wait, WaitUntil, Timeout |
| GPS | GpsDistance, GpsMoveToward, PointInCircle, PointInPolygon |
| **Schema** | **ActivateSchema, DeactivateSchema, IsSchemaActive, ResetSchema** |
| **Config** | **GetConfig, SetConfig, ReloadConfig** |

**Total: 80+ node types**

---

## Rete.js Implementation Guide

### Why Rete.js

Rete.js v2 is recommended for this editor because:
1. **TypeScript first** - Full type safety
2. **Framework agnostic** - Works with React, Vue, Angular
3. **Plugin architecture** - Easy to extend
4. **Active community** - Well maintained

### Rete.js v2 Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Rete Editor                             │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Nodes     │  │ Connections │  │     Controls            │  │
│  │  (Custom)   │  │  (Sockets)  │  │  (Input widgets)        │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│  Plugins:                                                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────────┐   │
│  │AreaPlugin│ │Connection│ │ Render   │ │ Auto-arrange      │   │
│  │ (canvas) │ │  Plugin  │ │ (React)  │ │ (dagre layout)    │   │
│  └──────────┘ └──────────┘ └──────────┘ └───────────────────┘   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────────┐   │
│  │ Minimap  │ │ History  │ │ Context  │ │ SelectionPlugin   │   │
│  │          │ │ (undo)   │ │  Menu    │ │                   │   │
│  └──────────┘ └──────────┘ └──────────┘ └───────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Required Packages

```bash
npm install rete rete-area-plugin rete-connection-plugin rete-react-plugin
npm install rete-auto-arrange-plugin rete-minimap-plugin rete-history-plugin
npm install rete-context-menu-plugin
npm install dagre  # For auto-layout
npm install styled-components  # For custom styling
```

### Socket Types Definition

```typescript
// sockets.ts
import { ClassicPreset } from 'rete';

// Define socket types for type checking
export const Sockets = {
  // Primitive types
  Boolean: new ClassicPreset.Socket('boolean'),
  Number: new ClassicPreset.Socket('number'),
  String: new ClassicPreset.Socket('string'),

  // Complex types
  Any: new ClassicPreset.Socket('any'),
  Array: new ClassicPreset.Socket('array'),
  Map: new ClassicPreset.Socket('map'),
  Player: new ClassicPreset.Socket('player'),

  // Execution flow (special)
  Flow: new ClassicPreset.Socket('flow'),
};

// Socket compatibility matrix
export function canConnect(output: ClassicPreset.Socket, input: ClassicPreset.Socket): boolean {
  // Any accepts everything
  if (input.name === 'any') return true;

  // Number accepts number
  if (input.name === 'number' && output.name === 'number') return true;

  // Flow only connects to flow
  if (input.name === 'flow') return output.name === 'flow';

  // Same types connect
  return output.name === input.name;
}
```

### Base Node Class

```typescript
// nodes/BaseNode.ts
import { ClassicPreset } from 'rete';

export class BaseNode extends ClassicPreset.Node {
  // Position for serialization
  position: { x: number; y: number } = { x: 0, y: 0 };

  // For JSON export
  toJSON(): object {
    return {
      id: this.id,
      type: this.constructor.name.replace('Node', ''),
      inputs: this.serializeInputs(),
      _position: this.position,
    };
  }

  protected serializeInputs(): Record<string, any> {
    const inputs: Record<string, any> = {};
    for (const [key, input] of Object.entries(this.inputs)) {
      const control = input.control;
      if (control && 'value' in control) {
        inputs[key] = { constant: control.value };
      }
      // Connection references handled separately
    }
    return inputs;
  }
}
```

### CallFunction Node (Dynamic Inputs)

```typescript
// nodes/CallFunctionNode.ts
import { ClassicPreset } from 'rete';
import { BaseNode } from './BaseNode';
import { Sockets } from '../sockets';

export class CallFunctionNode extends BaseNode {
  private functionDefinitions: Map<string, FunctionDefinition>;
  private currentFunction: string = '';

  constructor(
    functionDefinitions: Map<string, FunctionDefinition>,
    initialFunction?: string
  ) {
    super('CallFunction');
    this.functionDefinitions = functionDefinitions;

    // Flow ports
    this.addInput('flow_in', new ClassicPreset.Input(Sockets.Flow, '', false));
    this.addOutput('flow_out', new ClassicPreset.Output(Sockets.Flow, ''));

    // Function selector control
    const functionSelect = new DropdownControl(
      Array.from(functionDefinitions.keys()),
      initialFunction || '',
      (value) => this.onFunctionChange(value)
    );
    this.addControl('function', functionSelect);

    if (initialFunction) {
      this.onFunctionChange(initialFunction);
    }
  }

  private onFunctionChange(functionName: string) {
    this.currentFunction = functionName;
    const funcDef = this.functionDefinitions.get(functionName);

    // Remove old argument inputs
    for (const key of Object.keys(this.inputs)) {
      if (key.startsWith('arg_')) {
        this.removeInput(key);
      }
    }

    // Remove old result output
    if (this.outputs['result']) {
      this.removeOutput('result');
    }

    if (!funcDef) return;

    // Add argument inputs based on function parameters
    for (const param of funcDef.parameters) {
      const socket = this.getSocketForType(param.type);
      const input = new ClassicPreset.Input(socket, param.name, true);

      // Add inline control for constants
      input.addControl(new InputControl(param.type, null));
      this.addInput(`arg_${param.name}`, input);
    }

    // Add result output if function has return type
    if (funcDef.returnType) {
      const socket = this.getSocketForType(funcDef.returnType);
      this.addOutput('result', new ClassicPreset.Output(socket, 'result'));
    }

    // Trigger re-render
    this.update?.();
  }

  private getSocketForType(type: string): ClassicPreset.Socket {
    switch (type) {
      case 'int':
      case 'float':
      case 'float64':
      case 'int32':
      case 'int64':
        return Sockets.Number;
      case 'string':
        return Sockets.String;
      case 'bool':
        return Sockets.Boolean;
      default:
        return Sockets.Any;
    }
  }

  toJSON() {
    const base = super.toJSON();
    const args: Record<string, any> = {};

    // Serialize arguments
    for (const [key, input] of Object.entries(this.inputs)) {
      if (key.startsWith('arg_')) {
        const paramName = key.replace('arg_', '');
        const control = input.control;
        if (control && 'value' in control) {
          args[paramName] = { constant: control.value };
        }
        // Connections handled by editor serialization
      }
    }

    return {
      ...base,
      inputs: {
        function: { constant: this.currentFunction },
        args,
      },
    };
  }
}
```

### Return Node (Context-Aware)

```typescript
// nodes/ReturnNode.ts
import { ClassicPreset } from 'rete';
import { BaseNode } from './BaseNode';
import { Sockets } from '../sockets';

export class ReturnNode extends BaseNode {
  private returnType: string | null = null;

  constructor(returnType?: string) {
    super('Return');

    // Flow in only (terminal node)
    this.addInput('flow_in', new ClassicPreset.Input(Sockets.Flow, '', false));

    if (returnType) {
      this.setReturnType(returnType);
    }
  }

  setReturnType(type: string | null) {
    this.returnType = type;

    // Remove existing value input
    if (this.inputs['value']) {
      this.removeInput('value');
    }

    // Add value input if function has return type
    if (type) {
      const socket = this.getSocketForType(type);
      const input = new ClassicPreset.Input(socket, 'value', true);
      input.addControl(new InputControl(type, null));
      this.addInput('value', input);
    }

    this.update?.();
  }

  private getSocketForType(type: string): ClassicPreset.Socket {
    switch (type) {
      case 'int':
      case 'float':
      case 'float64':
        return Sockets.Number;
      case 'string':
        return Sockets.String;
      case 'bool':
        return Sockets.Boolean;
      default:
        return Sockets.Any;
    }
  }
}
```

### Editor Setup

```typescript
// editor.ts
import { createRoot } from 'react-dom/client';
import { NodeEditor, GetSchemes, ClassicPreset } from 'rete';
import { AreaPlugin, AreaExtensions } from 'rete-area-plugin';
import { ConnectionPlugin, Presets as ConnectionPresets } from 'rete-connection-plugin';
import { ReactPlugin, Presets, ReactArea2D } from 'rete-react-plugin';
import { AutoArrangePlugin, Presets as ArrangePresets } from 'rete-auto-arrange-plugin';
import { HistoryPlugin, HistoryExtensions } from 'rete-history-plugin';
import { MinimapPlugin } from 'rete-minimap-plugin';
import { ContextMenuPlugin, Presets as ContextMenuPresets } from 'rete-context-menu-plugin';

type Schemes = GetSchemes<BaseNode, Connection<BaseNode, BaseNode>>;
type AreaExtra = ReactArea2D<Schemes>;

export async function createEditor(container: HTMLElement) {
  const editor = new NodeEditor<Schemes>();
  const area = new AreaPlugin<Schemes, AreaExtra>(container);
  const connection = new ConnectionPlugin<Schemes, AreaExtra>();
  const render = new ReactPlugin<Schemes, AreaExtra>({ createRoot });
  const arrange = new AutoArrangePlugin<Schemes>();
  const history = new HistoryPlugin<Schemes>();
  const minimap = new MinimapPlugin<Schemes>();
  const contextMenu = new ContextMenuPlugin<Schemes>({
    items: createContextMenuItems(),
  });

  // Register plugins
  editor.use(area);
  area.use(connection);
  area.use(render);
  area.use(arrange);
  area.use(history);
  area.use(minimap);
  area.use(contextMenu);

  // Configure presets
  render.addPreset(Presets.classic.setup());
  connection.addPreset(ConnectionPresets.classic.setup());
  arrange.addPreset(ArrangePresets.classic.setup());

  // Enable history
  HistoryExtensions.keyboard(history);

  // Enable selection
  AreaExtensions.selectableNodes(area, AreaExtensions.selector(), {
    accumulating: AreaExtensions.accumulateOnCtrl(),
  });

  // Socket compatibility check
  editor.addPipe((context) => {
    if (context.type === 'connectioncreate') {
      const { data } = context;
      if (!canConnect(data.source, data.target)) {
        return; // Block incompatible connections
      }
    }
    return context;
  });

  return { editor, area, arrange };
}
```

### Context Menu Items

```typescript
// contextMenu.ts
function createContextMenuItems() {
  return {
    searchBar: true,
    delay: 100,
    items(context: any) {
      const nodeCategories = {
        'State': [
          ['GetField', () => new GetFieldNode()],
          ['SetField', () => new SetFieldNode()],
          ['GetPlayer', () => new GetPlayerNode()],
        ],
        'Logic': [
          ['Compare', () => new CompareNode()],
          ['And', () => new AndNode()],
          ['Or', () => new OrNode()],
          ['Not', () => new NotNode()],
        ],
        'Math': [
          ['Add', () => new AddNode()],
          ['Subtract', () => new SubtractNode()],
          ['Multiply', () => new MultiplyNode()],
          ['Divide', () => new DivideNode()],
          ['Max', () => new MaxNode()],
          ['Min', () => new MinNode()],
        ],
        'Control Flow': [
          ['If', () => new IfNode()],
          ['ForEach', () => new ForEachNode()],
          ['While', () => new WhileNode()],
        ],
        'Functions': [
          ['CallFunction', () => new CallFunctionNode(functionDefinitions)],
          ['Return', () => new ReturnNode()],
        ],
        'Events': [
          ['EmitToAll', () => new EmitToAllNode()],
          ['EmitToPlayer', () => new EmitToPlayerNode()],
        ],
      };

      return Object.entries(nodeCategories).map(([category, nodes]) => ({
        label: category,
        submenu: nodes.map(([label, factory]) => ({
          label,
          key: label,
          handler: () => factory(),
        })),
      }));
    },
  };
}
```

### JSON Serialization

```typescript
// serialization.ts
export function serializeGraph(
  editor: NodeEditor<Schemes>,
  area: AreaPlugin<Schemes, AreaExtra>,
  metadata: { name: string; type: 'handler' | 'function' | 'filter'; parameters: Parameter[] }
): NodeGraph {
  const nodes = editor.getNodes();
  const connections = editor.getConnections();

  // Serialize nodes
  const serializedNodes = nodes.map((node) => {
    const view = area.nodeViews.get(node.id);
    return {
      ...node.toJSON(),
      _position: view ? { x: view.position.x, y: view.position.y } : undefined,
    };
  });

  // Serialize connections as inputs
  for (const conn of connections) {
    const targetNode = serializedNodes.find((n) => n.id === conn.target);
    if (targetNode) {
      const inputKey = conn.targetInput.replace('arg_', '');
      if (conn.targetInput.startsWith('arg_')) {
        // Function argument
        targetNode.inputs.args = targetNode.inputs.args || {};
        targetNode.inputs.args[inputKey] = `node:${conn.source}:${conn.sourceOutput}`;
      } else if (conn.targetInput !== 'flow_in') {
        // Regular input
        targetNode.inputs[inputKey] = `node:${conn.source}:${conn.sourceOutput}`;
      }
    }
  }

  // Build flow edges from flow connections
  const flowEdges = connections
    .filter((c) => c.sourceOutput === 'flow_out' || c.sourceOutput.startsWith('flow_'))
    .map((c) => ({
      from: c.source === 'start' ? 'start' : c.source,
      to: c.target,
      label: c.sourceOutput.replace('flow_', ''),
    }));

  return {
    nodes: serializedNodes.filter((n) => n.type !== 'Start'),
    flow: flowEdges,
  };
}
```

### Deserialization (Loading)

```typescript
// deserialization.ts
export async function loadGraph(
  editor: NodeEditor<Schemes>,
  area: AreaPlugin<Schemes, AreaExtra>,
  arrange: AutoArrangePlugin<Schemes>,
  graphData: { nodes: any[]; flow: FlowEdge[] },
  functionDefinitions: Map<string, FunctionDefinition>
) {
  // Clear existing
  await editor.clear();

  // Create Start node
  const startNode = new StartNode();
  await editor.addNode(startNode);
  await area.translate(startNode.id, { x: 0, y: 200 });

  // Create nodes
  const nodeMap = new Map<string, BaseNode>();
  for (const nodeData of graphData.nodes) {
    const node = createNodeFromData(nodeData, functionDefinitions);
    if (node) {
      await editor.addNode(node);
      nodeMap.set(nodeData.id, node);

      // Apply position if available
      if (nodeData._position) {
        await area.translate(node.id, nodeData._position);
      }
    }
  }

  // Create connections from inputs
  for (const nodeData of graphData.nodes) {
    const targetNode = nodeMap.get(nodeData.id);
    if (!targetNode) continue;

    for (const [inputKey, inputValue] of Object.entries(nodeData.inputs || {})) {
      if (typeof inputValue === 'string' && inputValue.startsWith('node:')) {
        const [, sourceId, sourceOutput] = inputValue.split(':');
        const sourceNode = nodeMap.get(sourceId);
        if (sourceNode) {
          await editor.addConnection(
            new Connection(sourceNode, sourceOutput, targetNode, inputKey)
          );
        }
      }
    }
  }

  // Create flow connections
  for (const edge of graphData.flow) {
    const fromNode = edge.from === 'start' ? startNode : nodeMap.get(edge.from);
    const toNode = nodeMap.get(edge.to);

    if (fromNode && toNode) {
      const outputName = edge.label ? `flow_${edge.label}` : 'flow_out';
      await editor.addConnection(
        new Connection(fromNode, outputName, toNode, 'flow_in')
      );
    }
  }

  // Auto-arrange if no positions
  const hasPositions = graphData.nodes.some((n) => n._position);
  if (!hasPositions) {
    await arrange.layout();
  }

  // Fit view
  AreaExtensions.zoomAt(area, editor.getNodes());
}

function createNodeFromData(
  data: any,
  functionDefinitions: Map<string, FunctionDefinition>
): BaseNode | null {
  switch (data.type) {
    case 'CallFunction':
      const funcName = data.inputs?.function?.constant || data.inputs?.function;
      return new CallFunctionNode(functionDefinitions, funcName);
    case 'Return':
      return new ReturnNode();
    case 'Compare':
      return new CompareNode();
    case 'Add':
      return new AddNode();
    // ... all other node types
    default:
      console.warn(`Unknown node type: ${data.type}`);
      return null;
  }
}
```

### Validation Integration

```typescript
// validation.ts
export async function validateGraph(editor: NodeEditor<Schemes>): Promise<ValidationResult> {
  const errors: ValidationError[] = [];
  const warnings: ValidationWarning[] = [];

  const nodes = editor.getNodes();
  const connections = editor.getConnections();

  // Check required inputs
  for (const node of nodes) {
    for (const [key, input] of Object.entries(node.inputs)) {
      if (key === 'flow_in') continue;

      const hasConnection = connections.some(
        (c) => c.target === node.id && c.targetInput === key
      );
      const hasValue = input.control && 'value' in input.control && input.control.value != null;

      if (!hasConnection && !hasValue && input.socket.name !== 'flow') {
        // Check if required
        const def = getNodeDefinition(node.constructor.name);
        const inputDef = def?.inputs.find((i) => i.name === key);
        if (inputDef?.required) {
          errors.push({
            nodeId: node.id,
            message: `Missing required input: ${key}`,
          });
        }
      }
    }
  }

  // Check Return nodes in functions with return type
  // This would need context about the current function

  // Check for unreachable nodes
  const reachable = new Set<string>();
  const startNode = nodes.find((n) => n instanceof StartNode);
  if (startNode) {
    traverseFlow(startNode.id, connections, reachable);
  }

  for (const node of nodes) {
    if (!reachable.has(node.id) && !(node instanceof StartNode)) {
      warnings.push({
        nodeId: node.id,
        message: 'Node is unreachable from Start',
      });
    }
  }

  return { valid: errors.length === 0, errors, warnings };
}
```

### Custom Node Styling

```typescript
// CustomNode.tsx (React component)
import styled from 'styled-components';
import { Presets } from 'rete-react-plugin';

const NodeContainer = styled.div<{ $category: string; $selected: boolean }>`
  background: ${(p) => getCategoryColor(p.$category)};
  border: 2px solid ${(p) => (p.$selected ? '#fff' : 'transparent')};
  border-radius: 8px;
  min-width: 180px;

  .title {
    padding: 8px 12px;
    font-weight: bold;
    border-bottom: 1px solid rgba(255,255,255,0.2);
  }

  .inputs, .outputs {
    padding: 8px;
  }
`;

function getCategoryColor(category: string): string {
  const colors: Record<string, string> = {
    'State': '#9b59b6',
    'Logic': '#e74c3c',
    'Math': '#3498db',
    'Control Flow': '#f39c12',
    'Functions': '#1abc9c',
    'Events': '#2ecc71',
    'Session': '#00bcd4',
  };
  return colors[category] || '#95a5a6';
}

export function CustomNode(props: { data: BaseNode }) {
  const { data } = props;
  const category = getNodeCategory(data.constructor.name);

  return (
    <NodeContainer $category={category} $selected={data.selected}>
      <div className="title">{data.label}</div>
      <div className="inputs">
        {Object.entries(data.inputs).map(([key, input]) => (
          <Presets.classic.Input key={key} data={input} />
        ))}
      </div>
      <div className="outputs">
        {Object.entries(data.outputs).map(([key, output]) => (
          <Presets.classic.Output key={key} data={output} />
        ))}
      </div>
    </NodeContainer>
  );
}
```

### Project Structure

```
src/
├── editor/
│   ├── index.ts              # Editor setup
│   ├── sockets.ts            # Socket definitions
│   ├── serialization.ts      # JSON export
│   ├── deserialization.ts    # JSON import
│   └── validation.ts         # Graph validation
├── nodes/
│   ├── BaseNode.ts           # Base class
│   ├── StartNode.ts          # Start node
│   ├── controls/
│   │   ├── InputControl.ts   # Text/number input
│   │   ├── DropdownControl.ts # Dropdown select
│   │   └── ReferenceControl.ts # Reference picker
│   ├── state/
│   │   ├── GetFieldNode.ts
│   │   ├── SetFieldNode.ts
│   │   └── GetPlayerNode.ts
│   ├── logic/
│   │   ├── CompareNode.ts
│   │   ├── AndNode.ts
│   │   └── ...
│   ├── math/
│   │   ├── AddNode.ts
│   │   └── ...
│   ├── functions/
│   │   ├── CallFunctionNode.ts
│   │   └── ReturnNode.ts
│   └── index.ts              # Export all nodes
├── components/
│   ├── CustomNode.tsx        # Node renderer
│   ├── NodePalette.tsx       # Node list
│   ├── PropertiesPanel.tsx   # Selected node props
│   ├── FunctionEditor.tsx    # Function tab
│   └── CodePreview.tsx       # Generated code
├── store/
│   ├── editorStore.ts        # Zustand store
│   └── types.ts              # TypeScript types
└── App.tsx                   # Main app
```

---

*Document Version: 1.2*
*Compatible with logicgen v1.0, schemagen v1.1, envgen v1.0*
*Rete.js v2.x recommended*
*Last Updated: 2025-12-20*

---

## Changelog

### v1.2 (2025-12-20)
- Added Schema & Config Management section
- Added `@helper` and `@root(active|inactive)` schema annotations
- Added `@default(value)` and `@default(config:X.Y)` field annotations
- Added Schema activation nodes (ActivateSchema, DeactivateSchema, etc.)
- Added Config nodes (GetConfig, SetConfig, ReloadConfig)
- Added Schema Manager UI mockup
- Added Config Editor Panel mockup
- Added envgen integration documentation
- Updated Appendix with new node types (80+ total)

### v1.1 (2025-12-19)
- Added Rete.js Implementation Guide
- Added Function System Details
- Added CallFunction dynamic inputs

### v1.0 (2025-12-18)
- Initial specification
