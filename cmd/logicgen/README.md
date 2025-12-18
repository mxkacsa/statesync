# logicgen - Node-Based Server Logic Generator

`logicgen` egy build-time kód generátor, amely node-based visual scripting graph-ból Go server logikát készít.

## Telepítés

```bash
go install github.com/mxkacsa/statesync/cmd/logicgen@latest
```

## Használat

### Alap használat

```bash
logicgen -input=game_logic.json -output=game_logic_gen.go
```

### Csak validálás

```bash
logicgen -input=game_logic.json -validate
```

## Node Graph Formátum

A node graph egy JSON fájl, amely tartalmazza az event handler-eket és a node-okat:

```json
{
  "version": "1.0",
  "package": "main",
  "imports": [],
  "handlers": [
    {
      "name": "OnCardPlayed",
      "event": "CardPlayed",
      "parameters": [
        {"name": "playerID", "type": "string"},
        {"name": "cardID", "type": "int32"}
      ],
      "nodes": [...],
      "flow": [...]
    }
  ]
}
```

## Node Típusok

### State Access Nodes

#### GetField
Mező olvasása a state-ből.

**Inputs:**
- `path` (string): mező útvonala (pl. "GameState.Round")

**Outputs:**
- `value` (any): a mező értéke

**Példa:**
```json
{
  "id": "getRound",
  "type": "GetField",
  "inputs": {
    "path": "GameState.Round"
  },
  "outputs": {
    "value": "currentRound"
  }
}
```

#### SetField
Mező írása a state-be.

**Inputs:**
- `path` (string): mező útvonala
- `value` (any): az új érték

**Példa:**
```json
{
  "id": "setRound",
  "type": "SetField",
  "inputs": {
    "path": "GameState.Round",
    "value": "node:incrementRound:result"
  }
}
```

#### GetPlayer
Játékos lekérése ID alapján.

**Inputs:**
- `playerID` (string): játékos azonosító

**Outputs:**
- `player` (*Player): a játékos objektum

**Példa:**
```json
{
  "id": "getPlayer",
  "type": "GetPlayer",
  "inputs": {
    "playerID": "param:playerID"
  },
  "outputs": {
    "player": "player"
  }
}
```

### Array Operations

#### AddToArray
Elem hozzáadása tömbhöz.

**Inputs:**
- `array` ([]any): a tömb
- `element` (any): a hozzáadandó elem

**Outputs:**
- `result` ([]any): az új tömb

**Példa:**
```json
{
  "id": "addCard",
  "type": "AddToArray",
  "inputs": {
    "array": "node:getPlayer:player.Hand",
    "element": "param:card"
  },
  "outputs": {
    "result": "newHand"
  }
}
```

#### RemoveFromArray
Elem törlése tömbből.

**Inputs:**
- `array` ([]any): a tömb
- `index` (int): a törlendő elem indexe (opcionális)
- `predicate` (func): szűrő függvény (opcionális)

**Outputs:**
- `result` ([]any): az új tömb

**Példa index alapján:**
```json
{
  "id": "removeCard",
  "type": "RemoveFromArray",
  "inputs": {
    "array": "node:getPlayer:player.Hand",
    "index": 0
  },
  "outputs": {
    "result": "newHand"
  }
}
```

**Példa predicate alapján:**
```json
{
  "id": "removeCard",
  "type": "RemoveFromArray",
  "inputs": {
    "array": "node:getPlayer:player.Hand",
    "predicate": "node:cardFilter:result"
  },
  "outputs": {
    "result": "newHand"
  }
}
```

#### FilterArray
Tömb szűrése predicate alapján.

**Inputs:**
- `array` ([]any): a tömb
- `predicate` (func): szűrő függvény

**Outputs:**
- `result` ([]any): a szűrt tömb

**Példa:**
```json
{
  "id": "filterHighScore",
  "type": "FilterArray",
  "inputs": {
    "array": "node:getPlayers:allPlayers",
    "predicate": "node:scoreCheck:result"
  },
  "outputs": {
    "result": "filteredPlayers"
  }
}
```

### Logic Nodes

#### Compare
Két érték összehasonlítása.

**Inputs:**
- `left` (any): bal oldali érték
- `op` (string): operátor ("==", "!=", "<", ">", "<=", ">=")
- `right` (any): jobb oldali érték

**Outputs:**
- `result` (bool): az összehasonlítás eredménye

**Példa:**
```json
{
  "id": "checkScore",
  "type": "Compare",
  "inputs": {
    "left": "node:getPlayer:player.Score",
    "op": ">=",
    "right": 100
  },
  "outputs": {
    "result": "hasHighScore"
  }
}
```

### Math Nodes

#### Add
Két szám összeadása.

**Inputs:**
- `a` (number): első szám
- `b` (number): második szám

**Outputs:**
- `result` (number): az összeg

**Példa:**
```json
{
  "id": "incrementScore",
  "type": "Add",
  "inputs": {
    "a": "node:getPlayer:player.Score",
    "b": 10
  },
  "outputs": {
    "result": "newScore"
  }
}
```

### Event Emission Nodes

#### EmitToAll
Event küldése minden játékosnak.

**Inputs:**
- `eventType` (string): az event típusa
- `payload` (map): az event adatai (opcionális)

**Példa:**
```json
{
  "id": "notifyAll",
  "type": "EmitToAll",
  "inputs": {
    "eventType": "GameStarted",
    "payload": {
      "round": "node:getRound:value"
    }
  }
}
```

#### EmitToPlayer
Event küldése egy játékosnak.

**Inputs:**
- `playerID` (string): játékos azonosító
- `eventType` (string): az event típusa
- `payload` (map): az event adatai (opcionális)

**Példa:**
```json
{
  "id": "notifyPlayer",
  "type": "EmitToPlayer",
  "inputs": {
    "playerID": "param:playerID",
    "eventType": "YourTurn",
    "payload": {}
  }
}
```

## Value References

A node input-ok értékei lehetnek:

### 1. Paraméter referencia
```json
"param:parameterName"
```

Példa:
```json
"playerID": "param:playerID"
```

### 2. Node output referencia
```json
"node:nodeID:outputName"
```

Példa:
```json
"value": "node:getPlayer:player"
```

### 3. Konstans érték
```json
42
"string value"
true
{"key": "value"}
```

Példa:
```json
"minScore": 100
```

### 4. State referencia
```json
"state:fieldPath"
```

Példa:
```json
"array": "state:Players"
```

## Flow Edges

A flow edge-ek határozzák meg a node-ok végrehajtási sorrendjét:

```json
{
  "from": "nodeID1",
  "to": "nodeID2"
}
```

Speciális node-ok:
- `"start"`: az event handler kezdete
- `"end"`: az event handler vége

Példa:
```json
"flow": [
  {"from": "start", "to": "getPlayer"},
  {"from": "getPlayer", "to": "updateScore"},
  {"from": "updateScore", "to": "notifyAll"}
]
```

## Példa: Kártya játék logika

Teljes példa egy kártya játék event handler-re:

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
          "id": "removeCard",
          "type": "RemoveFromArray",
          "inputs": {
            "array": "node:getPlayer:player.Hand",
            "index": "param:cardID"
          },
          "outputs": {"result": "updatedHand"}
        },
        {
          "id": "updateHand",
          "type": "SetField",
          "inputs": {
            "path": "player.Hand",
            "value": "node:removeCard:updatedHand"
          }
        },
        {
          "id": "notify",
          "type": "EmitToAll",
          "inputs": {
            "eventType": "CardPlayed",
            "payload": {
              "playerID": "param:playerID",
              "cardID": "param:cardID"
            }
          }
        }
      ],
      "flow": [
        {"from": "start", "to": "getPlayer"},
        {"from": "getPlayer", "to": "removeCard"},
        {"from": "removeCard", "to": "updateHand"},
        {"from": "updateHand", "to": "notify"}
      ]
    }
  ]
}
```

Ez a következő Go kódot generálja:

```go
package main

import (
    "fmt"
    "github.com/mxkacsa/statesync"
)

// OnCardPlayed handles the CardPlayed event
func OnCardPlayed(session *statesync.TrackedSession[*GameState, any, string], playerID string, cardID int32) error {
    // Node: getPlayer (GetPlayer)
    state := session.State().Get()
    var getPlayer_player *Player
    for i := range state.Players {
        if state.Players[i].ID == playerID {
            getPlayer_player = &state.Players[i]
            break
        }
    }
    if getPlayer_player == nil {
        return fmt.Errorf("player not found: %s", playerID)
    }

    // Node: removeCard (RemoveFromArray)
    removeCard_result := append(getPlayer_player.Hand[:cardID], getPlayer_player.Hand[cardID+1:]...)

    // Node: updateHand (SetField)
    session.State().Update(func(s **GameState) {
        // TODO: Set player.Hand = removeCard_result
    })

    // Node: notify (EmitToAll)
    enc := statesync.NewEventPayloadEncoder()
    session.Emit("CardPlayed", enc.Bytes())

    return nil
}
```

## Best Practices

1. **Egyértelmű node ID-k**: Használj beszédes neveket (pl. `getPlayer`, `incrementScore`)
2. **Lineáris flow**: Kerüld a túl komplex flow graph-okat
3. **Error handling**: A generált kód automatikusan kezeli a hibákat
4. **Type safety**: A validátor ellenőrzi a típusokat
5. **Dokumentáció**: Használj beszédes event és node neveket

## Hibaüzenetek

### "unknown node type"
A megadott node típus nem létezik. Ellenőrizd a `type` mezőt.

### "cycle detected in node graph"
A flow graph körfüggőséget tartalmaz. Ellenőrizd a flow edge-eket.

### "missing required input"
Egy kötelező input hiányzik. Ellenőrizd a node definition-t.

### "duplicate node ID"
Két node azonos ID-val rendelkezik. Használj egyedi ID-kat.

## Roadmap

- [ ] Visual editor web UI
- [ ] További node típusok (Loop, Switch, stb.)
- [ ] Custom node plugins
- [ ] Hot reload development módban
- [ ] Debug breakpoints
- [ ] Performance profiling
