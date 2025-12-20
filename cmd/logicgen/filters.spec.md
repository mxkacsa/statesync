# Node-Based Filter System Specification

## Overview

This document specifies how filters work with the node-based logic system:
1. **Filter Definitions** - Define filter logic as node graphs
2. **AddFilter/RemoveFilter Nodes** - Dynamically add/remove filters
3. **Parameter Passing** - Pass playerID, teamID, etc. to filters

---

## 1. Filter Definition

Filters are defined alongside handlers in the logic JSON. Each filter:
- Has a name and description
- Takes parameters (playerID, teamID, etc.)
- Contains nodes that transform state for a viewer

### JSON Structure

```json
{
  "schemaPath": "./game.schema.json",
  "stateType": "GameState",

  "filters": [
    {
      "name": "HideEnemyLocations",
      "description": "Hides enemy team locations from viewer",
      "parameters": [
        {"name": "viewerTeam", "type": "string"}
      ],
      "nodes": [
        {
          "id": "iteratePlayers",
          "type": "ForEach",
          "inputs": {
            "array": "state:Players"
          }
        },
        {
          "id": "getPlayerTeam",
          "type": "GetStructField",
          "inputs": {
            "struct": "node:iteratePlayers:item",
            "field": {"constant": "Team"}
          }
        },
        {
          "id": "isEnemy",
          "type": "Compare",
          "inputs": {
            "left": "node:getPlayerTeam:value",
            "op": {"constant": "!="},
            "right": "param:viewerTeam"
          }
        },
        {
          "id": "ifEnemy",
          "type": "If",
          "inputs": {
            "condition": "node:isEnemy:result"
          }
        },
        {
          "id": "hideLat",
          "type": "UpdateStruct",
          "inputs": {
            "struct": "node:iteratePlayers:item",
            "field": {"constant": "Lat"},
            "value": {"constant": 0}
          }
        },
        {
          "id": "hideLng",
          "type": "UpdateStruct",
          "inputs": {
            "struct": "node:iteratePlayers:item",
            "field": {"constant": "Lng"},
            "value": {"constant": 0}
          }
        }
      ],
      "flow": [
        {"from": "start", "to": "iteratePlayers"},
        {"from": "iteratePlayers", "to": "getPlayerTeam", "label": "body"},
        {"from": "getPlayerTeam", "to": "isEnemy"},
        {"from": "isEnemy", "to": "ifEnemy"},
        {"from": "ifEnemy", "to": "hideLat", "label": "true"},
        {"from": "hideLat", "to": "hideLng"},
        {"from": "hideLng", "to": "end"},
        {"from": "ifEnemy", "to": "end", "label": "false"}
      ]
    },
    {
      "name": "HidePlayerHand",
      "description": "Hides a specific player's hand from others",
      "parameters": [
        {"name": "playerID", "type": "string"},
        {"name": "viewerID", "type": "string"}
      ],
      "nodes": [
        {
          "id": "checkNotSelf",
          "type": "Compare",
          "inputs": {
            "left": "param:playerID",
            "op": {"constant": "!="},
            "right": "param:viewerID"
          }
        },
        {
          "id": "ifNotSelf",
          "type": "If",
          "inputs": {"condition": "node:checkNotSelf:result"}
        },
        {
          "id": "getPlayer",
          "type": "GetPlayer",
          "inputs": {"playerID": "param:playerID"}
        },
        {
          "id": "hideHand",
          "type": "UpdateStruct",
          "inputs": {
            "struct": "node:getPlayer:player",
            "field": {"constant": "Hand"},
            "value": {"constant": null}
          }
        }
      ],
      "flow": [
        {"from": "start", "to": "checkNotSelf"},
        {"from": "checkNotSelf", "to": "ifNotSelf"},
        {"from": "ifNotSelf", "to": "getPlayer", "label": "true"},
        {"from": "getPlayer", "to": "hideHand"},
        {"from": "hideHand", "to": "end"},
        {"from": "ifNotSelf", "to": "end", "label": "false"}
      ]
    }
  ],

  "handlers": [...]
}
```

---

## 2. Generated Filter Code

For each filter definition, generate a factory function:

```go
// HideEnemyLocations creates a filter that hides enemy team locations
func HideEnemyLocations(viewerTeam string) statesync.FilterFunc[*GameState] {
    return func(state *GameState) *GameState {
        // Clone state for modification
        filtered := state.ShallowClone()

        // Node: iteratePlayers (ForEach)
        for i := range filtered.Players {
            item := &filtered.Players[i]

            // Node: getPlayerTeam (GetStructField)
            getPlayerTeam_value := item.Team

            // Node: isEnemy (Compare)
            isEnemy_result := getPlayerTeam_value != viewerTeam

            // Node: ifEnemy (If)
            if isEnemy_result {
                // Node: hideLat (UpdateStruct)
                item.Lat = 0
                // Node: hideLng (UpdateStruct)
                item.Lng = 0
            }
        }

        return filtered
    }
}

// HidePlayerHand creates a filter that hides a player's hand from others
func HidePlayerHand(playerID, viewerID string) statesync.FilterFunc[*GameState] {
    return func(state *GameState) *GameState {
        // Node: checkNotSelf (Compare)
        checkNotSelf_result := playerID != viewerID

        if !checkNotSelf_result {
            return state // Self can see own hand
        }

        // Clone and modify
        filtered := state.ShallowClone()

        // Node: getPlayer (GetPlayer)
        for i := range filtered.Players {
            if filtered.Players[i].ID == playerID {
                // Node: hideHand (UpdateStruct)
                filtered.Players[i].Hand = nil
                break
            }
        }

        return filtered
    }
}
```

---

## 3. Filter Registry & Composition

Multiple filters can be active per viewer. We need a composition system:

```go
// FilterRegistry manages active filters per viewer
type FilterRegistry[T any, ID comparable] struct {
    mu      sync.RWMutex
    filters map[ID]map[string]statesync.FilterFunc[T] // viewerID -> filterID -> filter
}

// Add adds a filter instance for a viewer
func (r *FilterRegistry[T, ID]) Add(viewerID ID, filterID string, filter statesync.FilterFunc[T]) {
    r.mu.Lock()
    defer r.mu.Unlock()
    if r.filters[viewerID] == nil {
        r.filters[viewerID] = make(map[string]statesync.FilterFunc[T])
    }
    r.filters[viewerID][filterID] = filter
}

// Remove removes a filter instance
func (r *FilterRegistry[T, ID]) Remove(viewerID ID, filterID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()
    if r.filters[viewerID] == nil {
        return false
    }
    _, ok := r.filters[viewerID][filterID]
    delete(r.filters[viewerID], filterID)
    return ok
}

// GetComposed returns a composed filter for a viewer
func (r *FilterRegistry[T, ID]) GetComposed(viewerID ID) statesync.FilterFunc[T] {
    r.mu.RLock()
    filters := r.filters[viewerID]
    if len(filters) == 0 {
        r.mu.RUnlock()
        return nil
    }
    // Copy filter slice
    fns := make([]statesync.FilterFunc[T], 0, len(filters))
    for _, f := range filters {
        fns = append(fns, f)
    }
    r.mu.RUnlock()

    // Return composed filter
    return func(state T) T {
        for _, fn := range fns {
            state = fn(state)
        }
        return state
    }
}
```

---

## 4. AddFilter / RemoveFilter Nodes

### Node Types

```go
const (
    NodeAddFilter    NodeType = "AddFilter"    // Add a filter for a viewer
    NodeRemoveFilter NodeType = "RemoveFilter" // Remove a filter
    NodeHasFilter    NodeType = "HasFilter"    // Check if filter exists
)
```

### Node Definitions

```go
NodeAddFilter: {
    Type:        NodeAddFilter,
    Category:    "Session",
    Description: "Adds a filter for a viewer",
    Inputs: []PortDefinition{
        {Name: "viewerID", Type: "string", Required: true},
        {Name: "filterID", Type: "string", Required: true},   // Unique instance ID
        {Name: "filterName", Type: "string", Required: true}, // Filter type name
        {Name: "params", Type: "map[string]any", Required: false}, // Filter parameters
    },
    Outputs: []PortDefinition{},
},
NodeRemoveFilter: {
    Type:        NodeRemoveFilter,
    Category:    "Session",
    Description: "Removes a filter from a viewer",
    Inputs: []PortDefinition{
        {Name: "viewerID", Type: "string", Required: true},
        {Name: "filterID", Type: "string", Required: true},
    },
    Outputs: []PortDefinition{
        {Name: "removed", Type: "bool", Required: true},
    },
},
NodeHasFilter: {
    Type:        NodeHasFilter,
    Category:    "Session",
    Description: "Checks if a filter exists for a viewer",
    Inputs: []PortDefinition{
        {Name: "viewerID", Type: "string", Required: true},
        {Name: "filterID", Type: "string", Required: true},
    },
    Outputs: []PortDefinition{
        {Name: "exists", Type: "bool", Required: true},
    },
},
```

### JSON Examples

**Add filter with parameters:**
```json
{
  "id": "hideEnemyLocs",
  "type": "AddFilter",
  "inputs": {
    "viewerID": "param:playerID",
    "filterID": {"template": "hide_enemy_{{playerID}}"},
    "filterName": {"constant": "HideEnemyLocations"},
    "params": {
      "viewerTeam": "param:playerTeam"
    }
  }
}
```

**Remove filter:**
```json
{
  "id": "revealLocs",
  "type": "RemoveFilter",
  "inputs": {
    "viewerID": "param:playerID",
    "filterID": {"template": "hide_enemy_{{playerID}}"}
  }
}
```

### Generated Code

**AddFilter:**
```go
// Node: hideEnemyLocs (AddFilter)
hideEnemyLocs_filterID := fmt.Sprintf("hide_enemy_%s", playerID)
hideEnemyLocs_filter := HideEnemyLocations(playerTeam)
session.FilterRegistry().Add(playerID, hideEnemyLocs_filterID, hideEnemyLocs_filter)
session.UpdateFilter(playerID) // Refresh composed filter
```

**RemoveFilter:**
```go
// Node: revealLocs (RemoveFilter)
revealLocs_filterID := fmt.Sprintf("hide_enemy_%s", playerID)
revealLocs_removed := session.FilterRegistry().Remove(playerID, revealLocs_filterID)
session.UpdateFilter(playerID) // Refresh composed filter
```

---

## 5. Integration with TrackedSession

The session needs to integrate with the filter registry:

```go
type TrackedSession[T, A, ID] struct {
    // ... existing fields ...
    filterRegistry *FilterRegistry[T, ID]
}

// FilterRegistry returns the filter registry
func (s *TrackedSession[T, A, ID]) FilterRegistry() *FilterRegistry[T, ID] {
    return s.filterRegistry
}

// UpdateFilter refreshes the composed filter for a viewer
func (s *TrackedSession[T, A, ID]) UpdateFilter(viewerID ID) {
    filter := s.filterRegistry.GetComposed(viewerID)
    s.SetFilter(viewerID, filter)
}
```

---

## 6. Special Filter Inputs

Filters have access to special inputs:

| Input | Description | Example |
|-------|-------------|---------|
| `state:FieldName` | Access state field | `state:Players` |
| `param:name` | Filter parameter | `param:viewerTeam` |
| `viewer:id` | Current viewer's ID | `viewer:id` |
| `viewer:team` | Current viewer's team | `viewer:team` |

---

## 7. Complete Example

### JSON Definition

```json
{
  "schemaPath": "./chase_game.schema.json",
  "stateType": "ChaseGameState",

  "filters": [
    {
      "name": "HideRunnerFromChaser",
      "description": "Hide runner's exact location from chasers",
      "parameters": [
        {"name": "runnerID", "type": "string"}
      ],
      "nodes": [
        {
          "id": "getRunner",
          "type": "FindInArray",
          "inputs": {
            "array": "state:Players",
            "field": {"constant": "ID"},
            "value": "param:runnerID"
          }
        },
        {
          "id": "checkFound",
          "type": "IsNull",
          "inputs": {"value": "node:getRunner:item"}
        },
        {
          "id": "ifFound",
          "type": "If",
          "inputs": {"condition": "node:checkFound:isNull"}
        },
        {
          "id": "hidePublicLat",
          "type": "SetStructField",
          "inputs": {
            "struct": "node:getRunner:item",
            "field": {"constant": "PublicLat"},
            "value": {"constant": 0}
          }
        },
        {
          "id": "hidePublicLng",
          "type": "SetStructField",
          "inputs": {
            "struct": "node:getRunner:item",
            "field": {"constant": "PublicLng"},
            "value": {"constant": 0}
          }
        }
      ],
      "flow": [
        {"from": "start", "to": "getRunner"},
        {"from": "getRunner", "to": "checkFound"},
        {"from": "checkFound", "to": "ifFound"},
        {"from": "ifFound", "to": "hidePublicLat", "label": "false"},
        {"from": "hidePublicLat", "to": "hidePublicLng"},
        {"from": "hidePublicLng", "to": "end"},
        {"from": "ifFound", "to": "end", "label": "true"}
      ]
    }
  ],

  "handlers": [
    {
      "name": "OnPlayerCollectInvisibility",
      "event": "CollectInvisibility",
      "parameters": [
        {"name": "playerID", "type": "string"}
      ],
      "nodes": [
        {
          "id": "getPlayer",
          "type": "GetPlayer",
          "inputs": {"playerID": "param:playerID"}
        },
        {
          "id": "getTeam",
          "type": "GetStructField",
          "inputs": {
            "struct": "node:getPlayer:player",
            "field": {"constant": "Team"}
          }
        },
        {
          "id": "isRunner",
          "type": "Compare",
          "inputs": {
            "left": "node:getTeam:value",
            "op": {"constant": "=="},
            "right": {"constant": "runner"}
          }
        },
        {
          "id": "ifRunner",
          "type": "If",
          "inputs": {"condition": "node:isRunner:result"}
        },
        {
          "id": "iterateChasers",
          "type": "ForEachWhere",
          "inputs": {
            "array": "state:Players",
            "whereField": {"constant": "Team"},
            "whereOp": {"constant": "=="},
            "whereValue": {"constant": "chaser"}
          }
        },
        {
          "id": "addHideFilter",
          "type": "AddFilter",
          "inputs": {
            "viewerID": "node:iterateChasers:item.ID",
            "filterID": {"template": "hide_runner_{{playerID}}_from_{{node:iterateChasers:item.ID}}"},
            "filterName": {"constant": "HideRunnerFromChaser"},
            "params": {
              "runnerID": "param:playerID"
            }
          }
        },
        {
          "id": "scheduleReveal",
          "type": "Wait",
          "inputs": {
            "duration": {"constant": 30},
            "unit": {"constant": "s"}
          }
        },
        {
          "id": "removeHideFilter",
          "type": "RemoveFilter",
          "inputs": {
            "viewerID": "node:iterateChasers:item.ID",
            "filterID": {"template": "hide_runner_{{playerID}}_from_{{node:iterateChasers:item.ID}}"}
          }
        }
      ],
      "flow": [
        {"from": "start", "to": "getPlayer"},
        {"from": "getPlayer", "to": "getTeam"},
        {"from": "getTeam", "to": "isRunner"},
        {"from": "isRunner", "to": "ifRunner"},
        {"from": "ifRunner", "to": "iterateChasers", "label": "true"},
        {"from": "iterateChasers", "to": "addHideFilter", "label": "body"},
        {"from": "addHideFilter", "to": "scheduleReveal"},
        {"from": "scheduleReveal", "to": "removeHideFilter"},
        {"from": "removeHideFilter", "to": "end"},
        {"from": "ifRunner", "to": "end", "label": "false"}
      ]
    }
  ]
}
```

### Generated Code

```go
package chase

import (
    "fmt"
    "sync"

    "github.com/mxkacsa/statesync"
)

// ============================================================================
// Filter Factories
// ============================================================================

// HideRunnerFromChaser creates a filter that hides runner's exact location
func HideRunnerFromChaser(runnerID string) statesync.FilterFunc[*ChaseGameState] {
    return func(state *ChaseGameState) *ChaseGameState {
        // Find runner
        var runnerIdx int = -1
        for i := range state.Players {
            if state.Players[i].ID == runnerID {
                runnerIdx = i
                break
            }
        }

        if runnerIdx < 0 {
            return state // Runner not found
        }

        // Clone and hide location
        filtered := state.ShallowClone()
        filtered.Players[runnerIdx].PublicLat = 0
        filtered.Players[runnerIdx].PublicLng = 0

        return filtered
    }
}

// ============================================================================
// Filter Registry
// ============================================================================

type FilterRegistry struct {
    mu      sync.RWMutex
    filters map[string]map[string]statesync.FilterFunc[*ChaseGameState]
}

func NewFilterRegistry() *FilterRegistry {
    return &FilterRegistry{
        filters: make(map[string]map[string]statesync.FilterFunc[*ChaseGameState]),
    }
}

func (r *FilterRegistry) Add(viewerID, filterID string, filter statesync.FilterFunc[*ChaseGameState]) {
    r.mu.Lock()
    defer r.mu.Unlock()
    if r.filters[viewerID] == nil {
        r.filters[viewerID] = make(map[string]statesync.FilterFunc[*ChaseGameState])
    }
    r.filters[viewerID][filterID] = filter
}

func (r *FilterRegistry) Remove(viewerID, filterID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()
    if r.filters[viewerID] == nil {
        return false
    }
    _, ok := r.filters[viewerID][filterID]
    delete(r.filters[viewerID], filterID)
    return ok
}

func (r *FilterRegistry) Has(viewerID, filterID string) bool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    if r.filters[viewerID] == nil {
        return false
    }
    _, ok := r.filters[viewerID][filterID]
    return ok
}

func (r *FilterRegistry) GetComposed(viewerID string) statesync.FilterFunc[*ChaseGameState] {
    r.mu.RLock()
    filters := r.filters[viewerID]
    if len(filters) == 0 {
        r.mu.RUnlock()
        return nil
    }
    fns := make([]statesync.FilterFunc[*ChaseGameState], 0, len(filters))
    for _, f := range filters {
        fns = append(fns, f)
    }
    r.mu.RUnlock()

    return func(state *ChaseGameState) *ChaseGameState {
        for _, fn := range fns {
            state = fn(state)
        }
        return state
    }
}

// ============================================================================
// Handlers
// ============================================================================

// Global filter registry (initialized by session setup)
var filterRegistry *FilterRegistry

func OnPlayerCollectInvisibility(
    session *statesync.TrackedSession[*ChaseGameState, any, string],
    senderID string,
    playerID string,
) error {
    state := session.State().Get()
    _ = state
    _ = senderID

    // Node: getPlayer (GetPlayer)
    var getPlayer_player *Player
    for i := range state.Players {
        if state.Players[i].ID == playerID {
            getPlayer_player = &state.Players[i]
            break
        }
    }

    // Node: getTeam (GetStructField)
    getTeam_value := ""
    if getPlayer_player != nil {
        getTeam_value = getPlayer_player.Team
    }

    // Node: isRunner (Compare)
    isRunner_result := getTeam_value == "runner"

    // Node: ifRunner (If)
    if isRunner_result {
        // Node: iterateChasers (ForEachWhere)
        for i := range state.Players {
            if state.Players[i].Team == "chaser" {
                item := &state.Players[i]

                // Node: addHideFilter (AddFilter)
                addHideFilter_filterID := fmt.Sprintf("hide_runner_%s_from_%s", playerID, item.ID)
                addHideFilter_filter := HideRunnerFromChaser(playerID)
                filterRegistry.Add(item.ID, addHideFilter_filterID, addHideFilter_filter)
                session.SetFilter(item.ID, filterRegistry.GetComposed(item.ID))

                // Node: scheduleReveal (Wait) - spawn goroutine for delayed removal
                go func(viewerID, filterID string) {
                    time.Sleep(30 * time.Second)

                    // Node: removeHideFilter (RemoveFilter)
                    filterRegistry.Remove(viewerID, filterID)
                    session.SetFilter(viewerID, filterRegistry.GetComposed(viewerID))
                }(item.ID, addHideFilter_filterID)
            }
        }
    }

    return nil
}
```

---

## 8. Implementation Plan

### Phase 1: Types & Definitions
1. Add `FilterDefinition` struct to `types.go`
2. Add `NodeAddFilter`, `NodeRemoveFilter`, `NodeHasFilter` node types
3. Add node definitions

### Phase 2: Filter Generation
1. Parse filter definitions from JSON
2. Generate filter factory functions
3. Generate `FilterRegistry` type

### Phase 3: Node Generation
1. Implement `generateAddFilter()`
2. Implement `generateRemoveFilter()`
3. Implement `generateHasFilter()`

### Phase 4: Integration
1. Handle `params` input (map of parameters)
2. Handle `template` strings for dynamic IDs
3. Integrate with session's `SetFilter()`

---

## 9. Types Summary

### logicgen/types.go additions

```go
// FilterDefinition defines a filter that transforms state for viewers
type FilterDefinition struct {
    Name        string      `json:"name"`
    Description string      `json:"description,omitempty"`
    Parameters  []Parameter `json:"parameters"`
    Nodes       []Node      `json:"nodes"`
    Flow        []FlowEdge  `json:"flow"`
}

// LogicDefinition is the root structure for logic JSON
type LogicDefinition struct {
    SchemaPath string             `json:"schemaPath"`
    StateType  string             `json:"stateType"`
    Filters    []FilterDefinition `json:"filters,omitempty"`  // NEW
    Handlers   []EventHandler     `json:"handlers"`
}

// Node types
const (
    NodeAddFilter    NodeType = "AddFilter"
    NodeRemoveFilter NodeType = "RemoveFilter"
    NodeHasFilter    NodeType = "HasFilter"
)
```
