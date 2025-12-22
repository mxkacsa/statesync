# LogicGen v2 - Deklaratív Szabály-alapú Játéklogika Specifikáció

## 1. Bevezető

### 1.1 Motiváció

A jelenlegi imperatív, Blueprint-stílus logicgen számos problémával küzd multiplayer környezetben:
- Végtelen ciklus lehetőség (`While`, `ForEach`)
- Nem determinisztikus viselkedés
- Race condition-ök kockázata
- Nehezen auditálható/debugolható

Az új rendszer **deklaratív, szabály-alapú, tick-driven** architektúrát használ.

### 1.2 Alapelvek

1. **"Ha egy node-nak 'next node' kell, rossz node"** - Minden node adatot ad vissza, csak Rule-ok kompozitálják őket
2. **State-driven** - A logika a state-et transzformálja, nem procedurális flow-t követ
3. **Determinisztikus** - Ugyanaz az input mindig ugyanazt az outputot adja
4. **Tick-alapú** - Időfüggő műveletek deltaTime-mal számolnak, nem while loop-pal
5. **Server authoritative** - Minden számítás szerveren történik, kliens csak renderel

### 1.3 Fő komponensek

```
┌─────────────────────────────────────────────────────────────────┐
│                         RULE ENGINE                              │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────┐    ┌──────────┐    ┌─────────┐    ┌────────┐      │
│  │ TRIGGER │ -> │ SELECTOR │ -> │  VIEW   │ -> │ EFFECT │      │
│  └─────────┘    └──────────┘    └─────────┘    └────────┘      │
│       │              │               │              │           │
│   Mikor?         Kire?           Mit?          Csináld!        │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Architektúra

### 2.1 Könyvtár struktúra

```
cmd/logicgen/
├── ast/                    # Abstract Syntax Tree - definíciók
│   ├── rule.go            # Rule, Trigger, Effect struktúrák
│   ├── selector.go        # Selector node-ok
│   ├── view.go            # View/Aggregate node-ok
│   ├── transform.go       # Transform node-ok (MoveTowards, stb.)
│   └── schema.go          # Schema kontextus
│
├── eval/                   # Evaluation - végrehajtás
│   ├── engine.go          # Rule engine, tick loop
│   ├── context.go         # EvalContext - runtime kontextus
│   ├── trigger.go         # Trigger kiértékelés
│   ├── selector.go        # Selector végrehajtás
│   ├── view.go            # View számítás
│   └── effect.go          # Effect alkalmazás
│
├── gen/                    # Code generation
│   ├── generator.go       # Go kód generálás
│   ├── templates.go       # Go template-ek
│   └── optimize.go        # Optimalizációk
│
├── parse/                  # JSON/YAML parsing
│   ├── parser.go          # Rule fájl parser
│   └── validate.go        # Validáció
│
├── builtin/               # Beépített node-ok
│   ├── triggers.go        # OnTick, OnEvent, Distance, stb.
│   ├── selectors.go       # Entity, Filter, All, stb.
│   ├── views.go           # Max, Min, Sum, Count, GroupBy
│   ├── transforms.go      # MoveTowards, SetField, stb.
│   └── effects.go         # Mutate, Emit, Spawn, stb.
│
├── registry.go            # Node regisztráció
├── types.go               # Közös típusok
└── main.go                # CLI
```

### 2.2 Végrehajtási modell

```
┌────────────────────────────────────────────────────────────────┐
│                        TICK CYCLE                               │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. INPUT COLLECTION                                            │
│     - Player events (move, action, etc.)                        │
│     - External events (time, network)                           │
│                                                                 │
│  2. TRIGGER EVALUATION                                          │
│     - Minden rule trigger-je kiértékelődik                      │
│     - Determinisztikus sorrend (rule priority)                  │
│                                                                 │
│  3. RULE EXECUTION (fired triggers)                             │
│     a) Selector: entitások kiválasztása                         │
│     b) View: derived értékek számítása                          │
│     c) Transform: state transzformáció (pure)                   │
│     d) Effect: state mutáció alkalmazása                        │
│                                                                 │
│  4. STATE COMMIT                                                │
│     - Összes változás atomikusan commitolva                     │
│     - ChangeSet generálás (statesync)                           │
│                                                                 │
│  5. BROADCAST                                                   │
│     - Delta küldés klienseknek                                  │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

---

## 3. Node Típusok

### 3.1 Trigger Node-ok

Trigger-ek határozzák meg, **mikor** fut le egy szabály.

```go
// ast/rule.go
type Trigger interface {
    Type() TriggerType
    DependsOn() []Path  // Mely state path-ektől függ (optimalizációhoz)
}

type TriggerType string

const (
    TriggerOnTick     TriggerType = "OnTick"      // Minden tick-ben
    TriggerOnEvent    TriggerType = "OnEvent"     // Adott event beérkezésekor
    TriggerOnChange   TriggerType = "OnChange"    // State változáskor
    TriggerDistance   TriggerType = "Distance"    // GPS távolság feltétel
    TriggerTimer      TriggerType = "Timer"       // Időzített (cooldown, delay)
    TriggerCondition  TriggerType = "Condition"   // Általános feltétel
)
```

#### 3.1.1 OnTick
```json
{
    "type": "OnTick",
    "interval": 100  // ms, opcionális (default: minden tick)
}
```

#### 3.1.2 OnEvent
```json
{
    "type": "OnEvent",
    "event": "PlayerMove",
    "params": ["playerID", "targetPosition"]
}
```

#### 3.1.3 Distance Trigger
```json
{
    "type": "Distance",
    "from": "$.Drones[*].Position",
    "to": "$.Drones[*].Target",
    "operator": "<=",
    "value": 5.0,
    "unit": "meters"
}
```

#### 3.1.4 Condition Trigger
```json
{
    "type": "Condition",
    "expression": {
        "left": "$.Player.Health",
        "op": "<=",
        "right": 0
    }
}
```

### 3.2 Selector Node-ok

Selector-ok határozzák meg, **mely entitásokra** vonatkozik a szabály.

```go
// ast/selector.go
type Selector interface {
    Type() SelectorType
    Select(ctx *EvalContext) []Entity
}

type SelectorType string

const (
    SelectorAll      SelectorType = "All"       // Minden entitás adott típusból
    SelectorFilter   SelectorType = "Filter"    // Szűrt entitások
    SelectorSingle   SelectorType = "Single"    // Egyetlen entitás ID alapján
    SelectorRelated  SelectorType = "Related"   // Kapcsolódó entitások
    SelectorNearest  SelectorType = "Nearest"   // Legközelebbi (GPS)
)
```

#### 3.2.1 All Selector
```json
{
    "type": "All",
    "entity": "Drones"
}
```

#### 3.2.2 Filter Selector
```json
{
    "type": "Filter",
    "entity": "Players",
    "where": {
        "field": "Team",
        "op": "==",
        "value": "Red"
    }
}
```

#### 3.2.3 Nearest Selector (GPS)
```json
{
    "type": "Nearest",
    "entity": "Enemies",
    "from": "param:playerPosition",
    "limit": 5,
    "maxDistance": 100.0
}
```

### 3.3 View Node-ok

View-k **derived értékeket** számítanak az entitásokból. Read-only, nincs side effect.

```go
// ast/view.go
type View interface {
    Type() ViewType
    Compute(ctx *EvalContext, entities []Entity) Value
    DependsOn() []Path
}

type ViewType string

const (
    ViewField     ViewType = "Field"     // Egyszerű field kiolvasás
    ViewMax       ViewType = "Max"       // Maximum érték
    ViewMin       ViewType = "Min"       // Minimum érték
    ViewSum       ViewType = "Sum"       // Összeg
    ViewCount     ViewType = "Count"     // Darabszám
    ViewAvg       ViewType = "Avg"       // Átlag
    ViewFirst     ViewType = "First"     // Első elem
    ViewLast      ViewType = "Last"      // Utolsó elem
    ViewGroupBy   ViewType = "GroupBy"   // Csoportosítás
    ViewDistinct  ViewType = "Distinct"  // Egyedi értékek
    ViewMap       ViewType = "Map"       // Transzformáció (pure)
    ViewReduce    ViewType = "Reduce"    // Redukció
)
```

#### 3.3.1 Max View
```json
{
    "type": "Max",
    "field": "Score",
    "return": "entity"  // vagy "value"
}
```

**Generált kód:**
```go
func viewMaxScore(entities []Player) (Player, int) {
    var maxEntity Player
    maxValue := math.MinInt
    for _, e := range entities {
        if e.Score > maxValue {
            maxValue = e.Score
            maxEntity = e
        }
    }
    return maxEntity, maxValue
}
```

#### 3.3.2 GroupBy View
```json
{
    "type": "GroupBy",
    "field": "Team",
    "aggregate": {
        "type": "Sum",
        "field": "Score"
    }
}
```

**Eredmény:** `map[Team]int` - csapatonkénti összpontszám

#### 3.3.3 Map View
```json
{
    "type": "Map",
    "transform": {
        "ID": "$.ID",
        "Distance": {
            "type": "GpsDistance",
            "from": "$.Position",
            "to": "param:targetPosition"
        }
    }
}
```

### 3.4 Transform Node-ok

Transform-ok **pure függvények** amik értékeket transzformálnak. Nincs side effect.

```go
// ast/transform.go
type Transform interface {
    Type() TransformType
    Apply(ctx *EvalContext, input Value) Value
}

type TransformType string

const (
    // Math
    TransformAdd       TransformType = "Add"
    TransformSubtract  TransformType = "Subtract"
    TransformMultiply  TransformType = "Multiply"
    TransformDivide    TransformType = "Divide"
    TransformClamp     TransformType = "Clamp"
    TransformRound     TransformType = "Round"

    // GPS
    TransformMoveTowards   TransformType = "MoveTowards"
    TransformGpsDistance   TransformType = "GpsDistance"
    TransformGpsBearing    TransformType = "GpsBearing"
    TransformPointInRadius TransformType = "PointInRadius"

    // String
    TransformConcat    TransformType = "Concat"
    TransformFormat    TransformType = "Format"

    // Logic
    TransformIf        TransformType = "If"      // Ternary: condition ? then : else
    TransformCoalesce  TransformType = "Coalesce" // Első nem-null érték
)
```

#### 3.4.1 MoveTowards Transform

```json
{
    "type": "MoveTowards",
    "current": "$.Position",
    "target": "$.Target",
    "speed": 15.0,
    "unit": "km/h"
}
```

**Generált kód:**
```go
func transformMoveTowards(current, target GeoPoint, speed float64, dt time.Duration) GeoPoint {
    dir := target.Subtract(current)
    dist := dir.Length()
    maxStep := speed * dt.Seconds() / 3600.0 // km/h -> km/s
    if dist <= maxStep {
        return target
    }
    return current.Add(dir.Normalize().Scale(maxStep))
}
```

#### 3.4.2 Clamp Transform
```json
{
    "type": "Clamp",
    "value": "$.Health",
    "min": 0,
    "max": 100
}
```

#### 3.4.3 Conditional Transform
```json
{
    "type": "If",
    "condition": {
        "left": "$.Health",
        "op": ">",
        "right": 50
    },
    "then": "Healthy",
    "else": "Wounded"
}
```

### 3.5 Effect Node-ok

Effect-ek **mutációkat** alkalmaznak a state-re. Ez az egyetlen hely ahol side effect történhet.

```go
// ast/effect.go
type Effect interface {
    Type() EffectType
    Apply(ctx *EvalContext, entities []Entity, views map[string]Value) error
}

type EffectType string

const (
    EffectSet       EffectType = "Set"        // Érték beállítása
    EffectIncrement EffectType = "Increment"  // Érték növelése
    EffectDecrement EffectType = "Decrement"  // Érték csökkentése
    EffectAppend    EffectType = "Append"     // Elemhez hozzáadás
    EffectRemove    EffectType = "Remove"     // Elem eltávolítása
    EffectEmit      EffectType = "Emit"       // Event kiküldése
    EffectSpawn     EffectType = "Spawn"      // Új entitás létrehozása
    EffectDestroy   EffectType = "Destroy"    // Entitás törlése
    EffectTransform EffectType = "Transform"  // Transform alkalmazása és mentése
)
```

#### 3.5.1 Set Effect
```json
{
    "type": "Set",
    "path": "$.Winner",
    "value": "view:maxScorePlayer.ID"
}
```

#### 3.5.2 Transform Effect (GPS mozgás)
```json
{
    "type": "Transform",
    "path": "$.Position",
    "transform": {
        "type": "MoveTowards",
        "current": "$.Position",
        "target": "$.Target",
        "speed": 15.0
    }
}
```

#### 3.5.3 Emit Effect
```json
{
    "type": "Emit",
    "event": "DroneArrived",
    "to": "all",
    "payload": {
        "droneID": "$.ID",
        "position": "$.Position"
    }
}
```

#### 3.5.4 Conditional Effect
```json
{
    "type": "If",
    "condition": {
        "left": "view:distance",
        "op": "<=",
        "right": 5.0
    },
    "then": {
        "type": "Set",
        "path": "$.Status",
        "value": "Arrived"
    },
    "else": null
}
```

---

## 4. Rule Definíció

### 4.1 Rule struktúra

```go
// ast/rule.go
type Rule struct {
    Name        string            `json:"name"`
    Description string            `json:"description,omitempty"`
    Priority    int               `json:"priority,omitempty"`    // Magasabb = előbb fut
    Enabled     bool              `json:"enabled"`

    Trigger     Trigger           `json:"trigger"`
    Selector    Selector          `json:"selector,omitempty"`    // Opcionális
    Views       map[string]View   `json:"views,omitempty"`       // Opcionális
    Effects     []Effect          `json:"effects"`
}

type RuleSet struct {
    Version  string   `json:"version"`
    Package  string   `json:"package"`
    Imports  []string `json:"imports,omitempty"`
    Rules    []Rule   `json:"rules"`
}
```

### 4.2 Teljes példa - Drone mozgás és érkezés

```json
{
    "version": "2.0",
    "package": "game",
    "rules": [
        {
            "name": "MoveDronesTowardsTarget",
            "description": "Drone-ok mozgatása a célpont felé minden tick-ben",
            "priority": 100,
            "enabled": true,
            "trigger": {
                "type": "OnTick"
            },
            "selector": {
                "type": "Filter",
                "entity": "Drones",
                "where": {
                    "field": "Status",
                    "op": "==",
                    "value": "Moving"
                }
            },
            "effects": [
                {
                    "type": "Transform",
                    "path": "$.Position",
                    "transform": {
                        "type": "MoveTowards",
                        "current": "$.Position",
                        "target": "$.Target",
                        "speed": 15.0,
                        "unit": "km/h"
                    }
                }
            ]
        },
        {
            "name": "DroneArrivedAtTarget",
            "description": "Drone megérkezik ha 5 méteren belül van",
            "priority": 90,
            "enabled": true,
            "trigger": {
                "type": "Distance",
                "from": "$.Drones[*].Position",
                "to": "$.Drones[*].Target",
                "operator": "<=",
                "value": 5.0,
                "unit": "meters"
            },
            "selector": {
                "type": "Filter",
                "entity": "Drones",
                "where": {
                    "field": "Status",
                    "op": "==",
                    "value": "Moving"
                }
            },
            "views": {
                "arrivedDrone": {
                    "type": "First"
                }
            },
            "effects": [
                {
                    "type": "Set",
                    "path": "$.Status",
                    "value": "Arrived"
                },
                {
                    "type": "Set",
                    "path": "$.Position",
                    "value": "$.Target"
                },
                {
                    "type": "Emit",
                    "event": "DroneArrived",
                    "to": "all",
                    "payload": {
                        "droneID": "$.ID",
                        "position": "$.Position"
                    }
                }
            ]
        }
    ]
}
```

---

## 5. Path Expression Szintaxis

### 5.1 Alapok

```
$                   - Root state
$.Field             - Mező elérés
$.Array[0]          - Index szerinti elérés
$.Array[*]          - Összes elem (wildcard)
$.Map["key"]        - Map kulcs elérés
$.Parent.Child      - Nested path
```

### 5.2 Kontextus referenciák

```
$                   - Aktuális entitás (selector context)
$.Field             - Entitás mezője
param:name          - Rule paraméter (event payload-ból)
view:name           - Számított view érték
view:name.field     - View mező
const:value         - Konstans érték
state:Path          - Explicit state elérés (root-tól)
```

### 5.3 Példák

```json
// Aktuális entitás Score mezője
"$.Score"

// Event paraméter
"param:playerID"

// View eredmény
"view:maxScorePlayer"

// View mező
"view:maxScorePlayer.ID"

// Konstans
"const:100"

// Root state explicit elérés
"state:$.GamePhase"
```

---

## 6. Generált Kód

### 6.1 Engine struktúra

```go
// Generated code structure

package game

import (
    "context"
    "time"
    "github.com/mxkacsa/statesync"
)

// RuleEngine manages rule evaluation and execution
type RuleEngine struct {
    state      *GameState
    rules      []Rule
    tickRate   time.Duration
    lastTick   time.Time
}

// Rule interface for all rules
type Rule interface {
    Name() string
    Priority() int
    Evaluate(ctx *EvalContext) bool
    Execute(ctx *EvalContext) error
}

// EvalContext provides runtime context for rule evaluation
type EvalContext struct {
    State      *GameState
    DeltaTime  time.Duration
    Tick       uint64
    Event      *Event        // nil if OnTick
    Views      map[string]interface{}
    Params     map[string]interface{}
}
```

### 6.2 Generált Rule példa

```go
// MoveDronesTowardsTarget rule
type MoveDronesTowardsTargetRule struct{}

func (r *MoveDronesTowardsTargetRule) Name() string { return "MoveDronesTowardsTarget" }
func (r *MoveDronesTowardsTargetRule) Priority() int { return 100 }

func (r *MoveDronesTowardsTargetRule) Evaluate(ctx *EvalContext) bool {
    // OnTick trigger - always fires
    return true
}

func (r *MoveDronesTowardsTargetRule) Execute(ctx *EvalContext) error {
    // Selector: Filter Drones where Status == "Moving"
    for i := 0; i < ctx.State.DronesLen(); i++ {
        drone := ctx.State.DronesAt(i)
        if drone.Status != "Moving" {
            continue
        }

        // Transform: MoveTowards
        newPos := gps.MoveTowards(
            drone.Position,
            drone.Target,
            15.0, // km/h
            ctx.DeltaTime,
        )

        // Effect: Set Position
        drone.Position = newPos
        ctx.State.UpdateDronesAt(i, drone)
    }
    return nil
}
```

### 6.3 Engine tick loop

```go
func (e *RuleEngine) Tick(ctx context.Context) error {
    now := time.Now()
    dt := now.Sub(e.lastTick)
    e.lastTick = now

    evalCtx := &EvalContext{
        State:     e.state,
        DeltaTime: dt,
        Tick:      e.tick,
        Views:     make(map[string]interface{}),
        Params:    make(map[string]interface{}),
    }
    e.tick++

    // Sort rules by priority (descending)
    sort.Slice(e.rules, func(i, j int) bool {
        return e.rules[i].Priority() > e.rules[j].Priority()
    })

    // Evaluate and execute rules
    for _, rule := range e.rules {
        if rule.Evaluate(evalCtx) {
            if err := rule.Execute(evalCtx); err != nil {
                return fmt.Errorf("rule %s failed: %w", rule.Name(), err)
            }
        }
    }

    return nil
}
```

---

## 7. GPS Modul

### 7.1 GeoPoint típus

```go
// builtin/gps.go

package builtin

import "math"

const (
    EarthRadius = 6371000.0 // meters
)

type GeoPoint struct {
    Lat float64 `json:"lat"` // Latitude in degrees
    Lon float64 `json:"lon"` // Longitude in degrees
}

// DistanceTo calculates distance in meters using Haversine formula
func (p GeoPoint) DistanceTo(other GeoPoint) float64 {
    lat1 := p.Lat * math.Pi / 180
    lat2 := other.Lat * math.Pi / 180
    dLat := (other.Lat - p.Lat) * math.Pi / 180
    dLon := (other.Lon - p.Lon) * math.Pi / 180

    a := math.Sin(dLat/2)*math.Sin(dLat/2) +
        math.Cos(lat1)*math.Cos(lat2)*
        math.Sin(dLon/2)*math.Sin(dLon/2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

    return EarthRadius * c
}

// BearingTo calculates bearing in degrees
func (p GeoPoint) BearingTo(other GeoPoint) float64 {
    lat1 := p.Lat * math.Pi / 180
    lat2 := other.Lat * math.Pi / 180
    dLon := (other.Lon - p.Lon) * math.Pi / 180

    y := math.Sin(dLon) * math.Cos(lat2)
    x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)

    bearing := math.Atan2(y, x) * 180 / math.Pi
    return math.Mod(bearing+360, 360)
}

// MoveTowards moves towards target by distance in meters
func (p GeoPoint) MoveTowards(target GeoPoint, meters float64) GeoPoint {
    dist := p.DistanceTo(target)
    if dist <= meters {
        return target
    }

    fraction := meters / dist
    lat := p.Lat + (target.Lat-p.Lat)*fraction
    lon := p.Lon + (target.Lon-p.Lon)*fraction

    return GeoPoint{Lat: lat, Lon: lon}
}

// MoveTowardsSpeed moves towards target at given speed (km/h) for duration
func MoveTowards(current, target GeoPoint, speedKmh float64, dt time.Duration) GeoPoint {
    meters := speedKmh * 1000.0 / 3600.0 * dt.Seconds()
    return current.MoveTowards(target, meters)
}

// PointInRadius checks if point is within radius (meters) of center
func PointInRadius(point, center GeoPoint, radiusMeters float64) bool {
    return point.DistanceTo(center) <= radiusMeters
}
```

---

## 8. Implementációs Terv

### 8.1 Fázis 1: Alap infrastruktúra (1-2 hét)

1. **ast/ package létrehozása**
   - [ ] `types.go` - Közös típusok (Path, Value, Entity)
   - [ ] `rule.go` - Rule, RuleSet struktúrák
   - [ ] `trigger.go` - Trigger interface és típusok
   - [ ] `selector.go` - Selector interface és típusok
   - [ ] `view.go` - View interface és típusok
   - [ ] `transform.go` - Transform interface és típusok
   - [ ] `effect.go` - Effect interface és típusok

2. **parse/ package**
   - [ ] `parser.go` - JSON parsing Rule struktúrákba
   - [ ] `validate.go` - Szintaktikai és szemantikai validáció

3. **Tesztek**
   - [ ] Unit tesztek minden AST típushoz
   - [ ] Parser tesztek

### 8.2 Fázis 2: Kiértékelés (1-2 hét)

1. **eval/ package**
   - [ ] `context.go` - EvalContext implementáció
   - [ ] `path.go` - Path expression kiértékelés
   - [ ] `trigger.go` - Trigger kiértékelők
   - [ ] `selector.go` - Selector végrehajtók
   - [ ] `view.go` - View számítók
   - [ ] `effect.go` - Effect alkalmazók
   - [ ] `engine.go` - Rule engine, tick loop

2. **Tesztek**
   - [ ] Kiértékelési tesztek minden node típushoz
   - [ ] Integráció tesztek teljes rule-okra

### 8.3 Fázis 3: Beépített node-ok (1 hét)

1. **builtin/ package**
   - [ ] `triggers.go` - OnTick, OnEvent, Distance, Condition
   - [ ] `selectors.go` - All, Filter, Single, Nearest
   - [ ] `views.go` - Max, Min, Sum, Count, GroupBy, First, Last
   - [ ] `transforms.go` - Math ops, GPS ops, String ops
   - [ ] `effects.go` - Set, Increment, Append, Remove, Emit

2. **GPS modul**
   - [ ] `gps.go` - GeoPoint, distance, bearing, move

### 8.4 Fázis 4: Kód generálás (1-2 hét)

1. **gen/ package**
   - [ ] `generator.go` - Fő generátor logika
   - [ ] `templates.go` - Go kód template-ek
   - [ ] `rule_gen.go` - Rule -> Go kód
   - [ ] `optimize.go` - Optimalizációk (inline, stb.)

2. **Tesztek**
   - [ ] Generált kód tesztek
   - [ ] Benchmark tesztek

### 8.5 Fázis 5: CLI és integráció (0.5 hét)

1. **main.go frissítés**
   - [ ] Új flag-ek (--v2, --format)
   - [ ] Backwards compatibility warning

2. **Dokumentáció**
   - [ ] README frissítés
   - [ ] Példák

---

## 9. Tesztelési Stratégia

### 9.1 Unit tesztek

Minden node típushoz külön teszt fájl:

```go
// eval/trigger_test.go
func TestOnTickTrigger(t *testing.T) { ... }
func TestDistanceTrigger(t *testing.T) { ... }
func TestConditionTrigger(t *testing.T) { ... }

// eval/selector_test.go
func TestAllSelector(t *testing.T) { ... }
func TestFilterSelector(t *testing.T) { ... }

// eval/view_test.go
func TestMaxView(t *testing.T) { ... }
func TestGroupByView(t *testing.T) { ... }
```

### 9.2 Integrációs tesztek

Teljes rule-ok tesztelése:

```go
func TestDroneMovementRule(t *testing.T) {
    state := &GameState{
        Drones: []Drone{
            {ID: "d1", Position: GeoPoint{0, 0}, Target: GeoPoint{0.001, 0}, Status: "Moving"},
        },
    }

    engine := NewRuleEngine(state, rules)
    engine.Tick(context.Background())

    // Verify drone moved towards target
    assert.Greater(t, state.Drones[0].Position.Lon, 0.0)
}
```

### 9.3 Determinizmus tesztek

Ugyanaz az input mindig ugyanazt az outputot adja:

```go
func TestDeterminism(t *testing.T) {
    state1 := initialState()
    state2 := initialState()

    engine1 := NewRuleEngine(state1, rules)
    engine2 := NewRuleEngine(state2, rules)

    for i := 0; i < 100; i++ {
        engine1.Tick(ctx)
        engine2.Tick(ctx)
    }

    assert.Equal(t, state1, state2)
}
```

### 9.4 Performance benchmark

```go
func BenchmarkRuleEngine(b *testing.B) {
    state := largeGameState() // 1000 drones
    engine := NewRuleEngine(state, rules)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        engine.Tick(context.Background())
    }
}
```

---

## 10. Migráció a jelenlegi rendszerről

### 10.1 Konverziós guide

| Régi (v1)           | Új (v2)                           |
|---------------------|-----------------------------------|
| `ForEach` + body    | `Selector` + `Effect` per entity  |
| `While` loop        | `OnTick` trigger + condition      |
| `If` node           | `Condition` trigger vagy `If` effect |
| `SetField`          | `Set` effect                      |
| `GetField`          | Path expression (`$.Field`)       |
| `Compare`           | Expression a trigger/effect-ben   |
| `Wait`              | `Timer` trigger                   |
| `EmitToAll`         | `Emit` effect `to: "all"`         |

### 10.2 Példa konverzió

**Régi (v1):**
```json
{
    "handlers": [{
        "name": "UpdateDronePositions",
        "event": "OnTick",
        "nodes": [
            {"id": "getDrones", "type": "GetField", "inputs": {"path": "Drones"}},
            {"id": "loop", "type": "ForEach", "inputs": {"array": "node:getDrones:value"}},
            {"id": "checkMoving", "type": "Compare", ...},
            {"id": "ifMoving", "type": "If", ...},
            {"id": "move", "type": "GpsMoveToward", ...},
            {"id": "setPos", "type": "SetField", ...}
        ],
        "flow": [...]
    }]
}
```

**Új (v2):**
```json
{
    "rules": [{
        "name": "UpdateDronePositions",
        "trigger": {"type": "OnTick"},
        "selector": {
            "type": "Filter",
            "entity": "Drones",
            "where": {"field": "Status", "op": "==", "value": "Moving"}
        },
        "effects": [{
            "type": "Transform",
            "path": "$.Position",
            "transform": {
                "type": "MoveTowards",
                "current": "$.Position",
                "target": "$.Target",
                "speed": 15.0
            }
        }]
    }]
}
```

---

## 11. Bővíthetőség

### 11.1 Custom Trigger regisztráció

```go
// registry.go
func RegisterTrigger(name string, factory TriggerFactory) {
    triggerRegistry[name] = factory
}

// Custom trigger
type CustomDistanceTrigger struct {
    FromPath   string
    ToPath     string
    MaxDist    float64
}

func init() {
    RegisterTrigger("CustomDistance", func(config map[string]interface{}) Trigger {
        return &CustomDistanceTrigger{...}
    })
}
```

### 11.2 Custom Transform regisztráció

```go
func RegisterTransform(name string, fn TransformFunc) {
    transformRegistry[name] = fn
}

// Custom transform
func init() {
    RegisterTransform("CustomLerp", func(ctx *EvalContext, args map[string]Value) Value {
        a := args["a"].(float64)
        b := args["b"].(float64)
        t := args["t"].(float64)
        return a + (b-a)*t
    })
}
```

---

## 12. Összefoglaló

### 12.1 Előnyök az új rendszerrel

1. **Biztonság** - Nincs végtelen ciklus, determinisztikus
2. **Egyszerűség** - Deklaratív szabályok, nem procedurális kód
3. **Teljesítmény** - Optimalizálható, cache-elhető
4. **Tesztelhetőség** - Minden node izoláltan tesztelhető
5. **Debugolhatóság** - Rule-onként nyomon követhető
6. **Multiplayer-ready** - Server authoritative, replayable

### 12.2 Trade-off-ok

1. **Tanulási görbe** - Új paradigma
2. **Kifejezőerő** - Komplex szekvenciális logika nehezebb
3. **Újraírás** - Teljes refaktor szükséges

### 12.3 Következő lépések

1. Review és jóváhagyás
2. Fázis 1 implementáció kezdése
3. Iteratív fejlesztés és tesztelés
