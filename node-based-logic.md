# Node-Based Server Logic System

## Áttekintés

Vizuális node-based editor rendszer server-side játék logika készítéséhez. Build time-ban Go kódot generál a node graph alapján.

## Architektúra

```
Node Graph (JSON) → logicgen → Go Code → Compile
                                  ↓
                         Server Logic Handlers
```

## Node Típusok

### 1. Event Nodes (Trigger Points)
- **OnPlayerConnect** - játékos csatlakozásakor
- **OnPlayerDisconnect** - játékos lecsatlakozásakor
- **OnCustomEvent** - egyéni event fogadása
- **OnTick** - időzített trigger (pl. 60 fps)

Input: context (playerID, eventData, timestamp)
Output: execution flow

### 2. State Access Nodes
- **GetField** - mező olvasása a state-ből
- **SetField** - mező írása
- **GetPlayer** - játékos adatok lekérése
- **GetCurrentState** - teljes state lekérése

Input: field path (pl. "GameState.Round")
Output: value

### 3. Array/List Operations
- **AddToArray** - elem hozzáadása tömbhöz
- **RemoveFromArray** - elem törlése tömbből (index vagy predicate alapján)
- **FilterArray** - tömb szűrése predicate alapján
- **MapArray** - tömb transzformálása
- **FindInArray** - elem keresése tömbben
- **ArrayLength** - tömb hossza
- **ArrayAt** - elem index alapján

### 4. Map/Dictionary Operations
- **SetMapValue** - kulcs-érték pár beállítása
- **GetMapValue** - érték lekérése kulcs alapján
- **RemoveMapKey** - kulcs törlése
- **HasMapKey** - kulcs létezésének ellenőrzése
- **MapKeys** - összes kulcs lekérése
- **MapValues** - összes érték lekérése
- **FilterMap** - map szűrése

### 5. Control Flow Nodes
- **If** - feltételes elágazás
- **Switch** - többirányú elágazás
- **ForEach** - ciklus tömb/map elemein
- **While** - feltételes ciklus
- **Break** - ciklus megszakítása
- **Continue** - következő iteráció

### 6. Logic Nodes
- **And** - logikai ÉS
- **Or** - logikai VAGY
- **Not** - logikai NEM
- **Compare** - összehasonlítás (==, !=, <, >, <=, >=)
- **IsNull** - null ellenőrzés
- **IsEmpty** - üresség ellenőrzés (array, string)

### 7. Math Nodes
- **Add** - összeadás
- **Subtract** - kivonás
- **Multiply** - szorzás
- **Divide** - osztás
- **Modulo** - maradékos osztás
- **Min/Max** - minimum/maximum
- **Random** - véletlenszám generálás
- **Round/Floor/Ceil** - kerekítés

### 8. String Nodes
- **Concat** - string összefűzés
- **Format** - string formázás template-tel
- **Contains** - tartalmaz-e substring-et
- **Split** - string szétvágás
- **ToUpper/ToLower** - case konverzió
- **Trim** - whitespace eltávolítás

### 9. Variable Nodes
- **GetVariable** - változó olvasása
- **SetVariable** - változó írása
- **Constant** - konstans érték

### 10. Function Nodes
- **CallFunction** - egyéni függvény hívása
- **Return** - visszatérési érték

### 11. Event Emission Nodes
- **EmitEvent** - event küldése
- **EmitToPlayer** - event egy játékosnak
- **EmitToAll** - event mindenkinek
- **EmitExcept** - event mindenkinek egy kivételével
- **EmitToMany** - event több játékosnak

### 12. Effect Nodes
- **AddEffect** - effect hozzáadása a state-hez
- **RemoveEffect** - effect eltávolítása
- **HasEffect** - effect létezésének ellenőrzése

## Node Graph Formátum (JSON)

```json
{
  "version": "1.0",
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
          "id": "node1",
          "type": "GetPlayer",
          "inputs": {
            "playerID": {"source": "param:playerID"}
          },
          "outputs": {
            "player": "player"
          }
        },
        {
          "id": "node2",
          "type": "RemoveFromArray",
          "inputs": {
            "array": {"source": "node1:player.Hand"},
            "predicate": {"source": "node3:output"}
          },
          "outputs": {
            "result": "updatedHand"
          }
        },
        {
          "id": "node3",
          "type": "Compare",
          "inputs": {
            "left": {"source": "foreach:item.ID"},
            "op": "==",
            "right": {"source": "param:cardID"}
          },
          "outputs": {
            "result": "output"
          }
        },
        {
          "id": "node4",
          "type": "SetField",
          "inputs": {
            "path": "player.Hand",
            "value": {"source": "node2:updatedHand"}
          }
        },
        {
          "id": "node5",
          "type": "EmitToAll",
          "inputs": {
            "eventType": "CardPlayedEvent",
            "payload": {
              "playerID": {"source": "param:playerID"},
              "cardID": {"source": "param:cardID"}
            }
          }
        }
      ],
      "flow": [
        {"from": "start", "to": "node1"},
        {"from": "node1", "to": "node2"},
        {"from": "node2", "to": "node4"},
        {"from": "node4", "to": "node5"}
      ]
    }
  ]
}
```

## Generált Go Kód Példa

```go
package generated

import (
    "github.com/mxkacsa/statesync"
)

// OnCardPlayed handles the CardPlayed event
func OnCardPlayed(session *statesync.TrackedSession[*GameState, any, string], playerID string, cardID int32) error {
    // Node1: GetPlayer
    state := session.State().Get()
    var player *Player
    for i := range state.Players {
        if state.Players[i].ID == playerID {
            player = &state.Players[i]
            break
        }
    }
    if player == nil {
        return fmt.Errorf("player not found: %s", playerID)
    }

    // Node2-3: RemoveFromArray with predicate
    updatedHand := make([]Card, 0, len(player.Hand))
    for _, card := range player.Hand {
        // Node3: Compare
        if card.ID != cardID {
            updatedHand = append(updatedHand, card)
        }
    }

    // Node4: SetField
    session.State().Update(func(s **GameState) {
        for i := range (*s).Players {
            if (*s).Players[i].ID == playerID {
                (*s).Players[i].Hand = updatedHand
                (*s).changes.Mark(2, statesync.OpReplace) // Players field
                break
            }
        }
    })

    // Node5: EmitToAll
    enc := statesync.NewEventPayloadEncoder()
    enc.WriteString(playerID)
    enc.WriteInt32(cardID)
    session.Emit("CardPlayedEvent", enc.Bytes())

    return nil
}
```

## Node Connection Rules

1. **Type Safety**: output type must match input type
2. **Flow Validation**: no circular dependencies
3. **Execution Order**: topological sort of nodes
4. **Parameter Binding**: event parameters accessible to all nodes

## Code Generator Features

- **Type inference**: automatikus típus meghatározás
- **Optimization**: felesleges változók eliminálása
- **Error handling**: hibakezelés minden node-nál
- **Validation**: node graph validálás generálás előtt
- **Debug info**: source map generálás (node ID → Go kód sor)

## Integration Points

1. **Schema Integration**: használja a schemagen által generált típusokat
2. **Event System**: integráció a statesync event rendszerrel
3. **State Tracking**: automatikus change tracking
4. **Session API**: használja a TrackedSession API-t

## Future Extensions

- **Visual Editor**: web-based node editor
- **Hot Reload**: runtime node graph frissítés development módban
- **Profiling**: node execution time mérés
- **Breakpoints**: debug breakpoint-ok node-okon
- **Custom Nodes**: user-defined node types plugin rendszerrel
