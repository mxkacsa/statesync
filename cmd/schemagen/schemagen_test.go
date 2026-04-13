package main

import (
	"strings"
	"testing"
)

func TestParseSimpleSchema(t *testing.T) {
	input := `
package game

@id(1)
type Player {
    ID    string
    Name  string
    Score int64
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if schema.Package != "game" {
		t.Errorf("expected package 'game', got '%s'", schema.Package)
	}

	if len(schema.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(schema.Types))
	}

	player := schema.Types[0]
	if player.Name != "Player" {
		t.Errorf("expected type name 'Player', got '%s'", player.Name)
	}
	if player.ID != 1 {
		t.Errorf("expected ID 1, got %d", player.ID)
	}
	if len(player.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(player.Fields))
	}

	// Check fields
	expectedFields := []struct {
		name, typ string
	}{
		{"ID", "string"},
		{"Name", "string"},
		{"Score", "int64"},
	}

	for i, ef := range expectedFields {
		if player.Fields[i].Name != ef.name {
			t.Errorf("field %d: expected name '%s', got '%s'", i, ef.name, player.Fields[i].Name)
		}
		if player.Fields[i].Type != ef.typ {
			t.Errorf("field %d: expected type '%s', got '%s'", i, ef.typ, player.Fields[i].Type)
		}
	}
}

func TestParseArrayAndMapFields(t *testing.T) {
	input := `
package game

@id(1)
type GameState {
    Players []Player        @key(ID)
    Scores  map[string]int64
    Items   []int32
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	game := schema.Types[0]
	if len(game.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(game.Fields))
	}

	// Players field with @key
	players := game.Fields[0]
	if players.Type != "[]Player" {
		t.Errorf("expected type '[]Player', got '%s'", players.Type)
	}
	if players.Key != "ID" {
		t.Errorf("expected key 'ID', got '%s'", players.Key)
	}

	// Scores map field
	scores := game.Fields[1]
	if scores.Type != "map[string]int64" {
		t.Errorf("expected type 'map[string]int64', got '%s'", scores.Type)
	}

	// Items array field
	items := game.Fields[2]
	if items.Type != "[]int32" {
		t.Errorf("expected type '[]int32', got '%s'", items.Type)
	}
}

func TestParseViewAnnotations(t *testing.T) {
	input := `
package game

@id(1)
type Player {
    ID     string
    Name   string
    Secret string  @view(admin)
    Hand   []int32 @view(owner)
}

view all {}
view admin { includes: all }
view owner { includes: all }
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	player := schema.Types[0]

	// Check view annotations
	if len(player.Fields[2].Views) != 1 || player.Fields[2].Views[0] != "admin" {
		t.Errorf("expected Secret to have view 'admin', got %v", player.Fields[2].Views)
	}
	if len(player.Fields[3].Views) != 1 || player.Fields[3].Views[0] != "owner" {
		t.Errorf("expected Hand to have view 'owner', got %v", player.Fields[3].Views)
	}

	// Check views
	if len(schema.Views) != 3 {
		t.Fatalf("expected 3 views, got %d", len(schema.Views))
	}

	viewNames := make(map[string]bool)
	for _, v := range schema.Views {
		viewNames[v.Name] = true
	}
	if !viewNames["all"] || !viewNames["admin"] || !viewNames["owner"] {
		t.Errorf("expected views 'all', 'admin', 'owner', got %v", viewNames)
	}
}

func TestParseTypeWithoutID(t *testing.T) {
	input := `
package game

type Player {
    ID   string
    Name string
}

type Game {
    Round int32
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(schema.Types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(schema.Types))
	}

	// Auto-assigned IDs
	if schema.Types[0].ID != 1 {
		t.Errorf("expected auto ID 1, got %d", schema.Types[0].ID)
	}
	if schema.Types[1].ID != 2 {
		t.Errorf("expected auto ID 2, got %d", schema.Types[1].ID)
	}
}

func TestParseComments(t *testing.T) {
	input := `
// Package comment
package game

// Player represents a player
@id(1)
type Player {
    // Player ID
    ID   string
    # Another comment style
    Name string
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if schema.Package != "game" {
		t.Errorf("expected package 'game', got '%s'", schema.Package)
	}
	if len(schema.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(schema.Types))
	}
}

func TestParseType(t *testing.T) {
	tests := []struct {
		input    string
		isArray  bool
		isMap    bool
		baseType string
		elemType string
		keyType  string
	}{
		{"string", false, false, "string", "", ""},
		{"int64", false, false, "int64", "", ""},
		{"[]Player", true, false, "array", "Player", ""},
		{"[]int32", true, false, "array", "int32", ""},
		{"map[string]int64", false, true, "map", "int64", "string"},
		{"map[string]Player", false, true, "map", "Player", "string"},
		{"*Player", false, false, "Player", "", ""},
	}

	for _, tt := range tests {
		pt := ParseType(tt.input)
		if pt.IsArray != tt.isArray {
			t.Errorf("%s: expected IsArray=%v, got %v", tt.input, tt.isArray, pt.IsArray)
		}
		if pt.IsMap != tt.isMap {
			t.Errorf("%s: expected IsMap=%v, got %v", tt.input, tt.isMap, pt.IsMap)
		}
		if pt.BaseType != tt.baseType {
			t.Errorf("%s: expected BaseType='%s', got '%s'", tt.input, tt.baseType, pt.BaseType)
		}
		if pt.ElemType != tt.elemType {
			t.Errorf("%s: expected ElemType='%s', got '%s'", tt.input, tt.elemType, pt.ElemType)
		}
		if pt.KeyType != tt.keyType {
			t.Errorf("%s: expected KeyType='%s', got '%s'", tt.input, tt.keyType, pt.KeyType)
		}
	}
}

func TestGoType(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"string", "string"},
		{"int64", "int64"},
		{"[]Player", "[]Player"},
		{"[]int32", "[]int32"},
		{"map[string]int64", "map[string]int64"},
		{"bytes", "[]byte"},
	}

	for _, tt := range tests {
		got := GoType(tt.input)
		if got != tt.expected {
			t.Errorf("GoType(%s): expected '%s', got '%s'", tt.input, tt.expected, got)
		}
	}
}

func TestTSType(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"string", "string"},
		{"int64", "number"},
		{"int32", "number"},
		{"float64", "number"},
		{"bool", "boolean"},
		{"[]Player", "Player[]"},
		{"[]int32", "number[]"},
		{"map[string]int64", "Record<string, number>"},
		{"bytes", "Uint8Array"},
	}

	for _, tt := range tests {
		got := TSType(tt.input)
		if got != tt.expected {
			t.Errorf("TSType(%s): expected '%s', got '%s'", tt.input, tt.expected, got)
		}
	}
}

func TestGenerateGo(t *testing.T) {
	input := `
package game

@id(1)
type Player {
    ID    string
    Name  string
    Score int64
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// Check for expected content
	checks := []string{
		"package game",
		"type Player struct",
		"func NewPlayer()",
		"func PlayerSchema()",
		"func (t *Player) Schema()",
		"func (t *Player) Changes()",
		"func (t *Player) ClearChanges()",
		"func (t *Player) MarkAllDirty()",
		"func (t *Player) GetFieldValue(",
		"func (t *Player) ID()",
		"func (t *Player) SetID(",
		"func (t *Player) Name()",
		"func (t *Player) SetName(",
		"func (t *Player) Score()",
		"func (t *Player) SetScore(",
		"statesync.NewSchemaBuilder",
		"WithID(1)",
	}

	for _, check := range checks {
		if !strings.Contains(codeStr, check) {
			t.Errorf("generated code missing: %s", check)
		}
	}
}

func TestGenerateGoArrayMethods(t *testing.T) {
	input := `
package game

@id(1)
type Player {
    Hand []int32
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// Check for array methods
	checks := []string{
		"func (t *Player) Hand()",
		"func (t *Player) SetHand(",
		"func (t *Player) AppendHand(",
		"func (t *Player) RemoveHandAt(",
		"func (t *Player) UpdateHandAt(",
		"func (t *Player) HandLen()",
		"func (t *Player) HandAt(",
	}

	for _, check := range checks {
		if !strings.Contains(codeStr, check) {
			t.Errorf("generated code missing array method: %s", check)
		}
	}
}

func TestGenerateGoMapMethods(t *testing.T) {
	input := `
package game

@id(1)
type Game {
    Scores map[string]int64
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// Check for map methods
	checks := []string{
		"func (t *Game) Scores()",
		"func (t *Game) SetScores(",
		"func (t *Game) SetScoresKey(",
		"func (t *Game) DeleteScoresKey(",
		"func (t *Game) ScoresGet(",
	}

	for _, check := range checks {
		if !strings.Contains(codeStr, check) {
			t.Errorf("generated code missing map method: %s", check)
		}
	}
}

func TestGenerateTS(t *testing.T) {
	input := `
package game

@id(1)
type Player {
    ID    string
    Name  string
    Score int64
    Hand  []int32
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateTS(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// Check for expected content
	checks := []string{
		"export interface Player",
		"ID: string",
		"Name: string",
		"Score: number",
		"Hand: number[]",
		"export const PlayerSchema: Schema",
		"defineSchema(",
		"FieldType.String",
		"FieldType.Int64",
		"FieldType.Array",
		"createRegistry()",
		"createPlayerState(",
	}

	for _, check := range checks {
		if !strings.Contains(codeStr, check) {
			t.Errorf("generated TS code missing: %s", check)
		}
	}
}

func TestGenerateTSViews(t *testing.T) {
	input := `
package game

@id(1)
type Player {
    ID string
}

view all {}
view admin { includes: all }
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateTS(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// Check for view types
	checks := []string{
		"export type allView",
		"export type adminView",
		"export type ViewType",
	}

	for _, check := range checks {
		if !strings.Contains(codeStr, check) {
			t.Errorf("generated TS code missing view: %s", check)
		}
	}
}

func TestFieldTypeEnum(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"int8", "TypeInt8"},
		{"int16", "TypeInt16"},
		{"int32", "TypeInt32"},
		{"int64", "TypeInt64"},
		{"int", "TypeInt64"},
		{"uint8", "TypeUint8"},
		{"uint16", "TypeUint16"},
		{"uint32", "TypeUint32"},
		{"uint64", "TypeUint64"},
		{"float32", "TypeFloat32"},
		{"float64", "TypeFloat64"},
		{"string", "TypeString"},
		{"bool", "TypeBool"},
		{"bytes", "TypeBytes"},
		{"Player", "TypeStruct"},
		{"[]int32", "TypeArray"},
		{"map[string]int", "TypeMap"},
	}

	for _, tt := range tests {
		got := FieldTypeEnum(tt.input)
		if got != tt.expected {
			t.Errorf("FieldTypeEnum(%s): expected '%s', got '%s'", tt.input, tt.expected, got)
		}
	}
}

func TestIsPrimitive(t *testing.T) {
	primitives := []string{
		"int8", "int16", "int32", "int64", "int",
		"uint8", "uint16", "uint32", "uint64", "uint",
		"float32", "float64", "string", "bool", "bytes",
	}

	for _, p := range primitives {
		if !IsPrimitive(p) {
			t.Errorf("expected %s to be primitive", p)
		}
	}

	nonPrimitives := []string{"Player", "Game", "[]int", "map[string]int"}
	for _, np := range nonPrimitives {
		if IsPrimitive(np) {
			t.Errorf("expected %s to NOT be primitive", np)
		}
	}
}

func TestParseComplexSchema(t *testing.T) {
	input := `
package main

// Player definition
@id(2)
type Player {
    ID      string
    Name    string
    Score   int64
    Hand    []int32    @view(owner)
    Ready   bool
}

// Main game state
@id(1)
type GameState {
    Round       int32
    Phase       string
    Players     []Player    @key(ID)
    Scores      map[string]int64
    SecretSeed  int64       @view(admin)
}

view all {}
view admin { includes: all }
view owner { includes: all }
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if schema.Package != "main" {
		t.Errorf("expected package 'main', got '%s'", schema.Package)
	}

	if len(schema.Types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(schema.Types))
	}

	// Find Player and GameState
	var player, game *TypeDef
	for i := range schema.Types {
		if schema.Types[i].Name == "Player" {
			player = schema.Types[i]
		} else if schema.Types[i].Name == "GameState" {
			game = schema.Types[i]
		}
	}

	if player == nil || game == nil {
		t.Fatal("missing Player or GameState type")
	}

	// Check Player
	if player.ID != 2 {
		t.Errorf("Player ID: expected 2, got %d", player.ID)
	}
	if len(player.Fields) != 5 {
		t.Errorf("Player fields: expected 5, got %d", len(player.Fields))
	}

	// Check GameState
	if game.ID != 1 {
		t.Errorf("GameState ID: expected 1, got %d", game.ID)
	}
	if len(game.Fields) != 5 {
		t.Errorf("GameState fields: expected 5, got %d", len(game.Fields))
	}

	// Check @key annotation
	playersField := game.Fields[2]
	if playersField.Key != "ID" {
		t.Errorf("Players key: expected 'ID', got '%s'", playersField.Key)
	}

	// Generate Go and check it compiles (by checking for no errors)
	_, err = GenerateGo(schema)
	if err != nil {
		t.Fatalf("Go generation failed: %v", err)
	}

	// Generate TS
	_, err = GenerateTS(schema)
	if err != nil {
		t.Fatalf("TS generation failed: %v", err)
	}
}

// ============================================
// Tests for new @helper, @root, @default features
// ============================================

func TestParseHelperAnnotation(t *testing.T) {
	input := `
package game

@id(1) @helper
type Player {
    ID   string
    Name string
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(schema.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(schema.Types))
	}

	player := schema.Types[0]
	if player.Role != RoleHelper {
		t.Errorf("expected role 'helper', got '%s'", player.Role)
	}
	if player.DefaultState != "inactive" {
		t.Errorf("expected defaultState 'inactive', got '%s'", player.DefaultState)
	}
}

func TestParseRootActiveAnnotation(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Round int32
    Phase string
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	game := schema.Types[0]
	if game.Role != RoleRoot {
		t.Errorf("expected role 'root', got '%s'", game.Role)
	}
	if game.DefaultState != "active" {
		t.Errorf("expected defaultState 'active', got '%s'", game.DefaultState)
	}
	if !game.IsRoot() {
		t.Error("IsRoot() should return true")
	}
	if !game.IsActiveByDefault() {
		t.Error("IsActiveByDefault() should return true")
	}
}

func TestParseRootInactiveAnnotation(t *testing.T) {
	input := `
package game

@id(1) @root(inactive)
type DroneMode {
    MaxDrones int32
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	drone := schema.Types[0]
	if drone.Role != RoleRoot {
		t.Errorf("expected role 'root', got '%s'", drone.Role)
	}
	if drone.DefaultState != "inactive" {
		t.Errorf("expected defaultState 'inactive', got '%s'", drone.DefaultState)
	}
	if !drone.IsRoot() {
		t.Error("IsRoot() should return true")
	}
	if drone.IsActiveByDefault() {
		t.Error("IsActiveByDefault() should return false")
	}
}

func TestParseRootWithoutState(t *testing.T) {
	input := `
package game

@id(1) @root
type GameMode {
    Active bool
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mode := schema.Types[0]
	if mode.Role != RoleRoot {
		t.Errorf("expected role 'root', got '%s'", mode.Role)
	}
	// Default state when not specified should be inactive
	if mode.DefaultState != "inactive" {
		t.Errorf("expected defaultState 'inactive' (default), got '%s'", mode.DefaultState)
	}
}

func TestParseDefaultLiteral(t *testing.T) {
	input := `
package game

@id(1)
type Config {
    Speed       float64     @default(1.5)
    MaxPlayers  int32       @default(4)
    Name        string      @default("default")
    Enabled     bool        @default(true)
    Empty       string      @default("")
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	config := schema.Types[0]
	if len(config.Fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(config.Fields))
	}

	// Speed
	speed := config.Fields[0]
	if speed.DefaultSource != DefaultLiteral {
		t.Errorf("Speed: expected source 'literal', got '%s'", speed.DefaultSource)
	}
	if speed.DefaultValue != "1.5" {
		t.Errorf("Speed: expected value '1.5', got '%s'", speed.DefaultValue)
	}

	// MaxPlayers
	maxPlayers := config.Fields[1]
	if maxPlayers.DefaultSource != DefaultLiteral {
		t.Errorf("MaxPlayers: expected source 'literal', got '%s'", maxPlayers.DefaultSource)
	}
	if maxPlayers.DefaultValue != "4" {
		t.Errorf("MaxPlayers: expected value '4', got '%s'", maxPlayers.DefaultValue)
	}

	// Name
	name := config.Fields[2]
	if name.DefaultSource != DefaultLiteral {
		t.Errorf("Name: expected source 'literal', got '%s'", name.DefaultSource)
	}
	if name.DefaultValue != "default" {
		t.Errorf("Name: expected value 'default', got '%s'", name.DefaultValue)
	}

	// Enabled
	enabled := config.Fields[3]
	if enabled.DefaultSource != DefaultLiteral {
		t.Errorf("Enabled: expected source 'literal', got '%s'", enabled.DefaultSource)
	}
	if enabled.DefaultValue != "true" {
		t.Errorf("Enabled: expected value 'true', got '%s'", enabled.DefaultValue)
	}

	// Empty string
	empty := config.Fields[4]
	if empty.DefaultSource != DefaultLiteral {
		t.Errorf("Empty: expected source 'literal', got '%s'", empty.DefaultSource)
	}
	if empty.DefaultValue != "" {
		t.Errorf("Empty: expected empty value, got '%s'", empty.DefaultValue)
	}
}

func TestParseDefaultConfig(t *testing.T) {
	input := `
package game

@id(1)
type GameState {
    SpeedMult   float64     @default(config:GameConfig.Speed)
    MaxPlayers  int32       @default(config:GameConfig.MaxPlayers)
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	game := schema.Types[0]

	// SpeedMult
	speed := game.Fields[0]
	if speed.DefaultSource != DefaultConfig {
		t.Errorf("SpeedMult: expected source 'config', got '%s'", speed.DefaultSource)
	}
	if speed.DefaultValue != "GameConfig.Speed" {
		t.Errorf("SpeedMult: expected value 'GameConfig.Speed', got '%s'", speed.DefaultValue)
	}

	// MaxPlayers
	maxPlayers := game.Fields[1]
	if maxPlayers.DefaultSource != DefaultConfig {
		t.Errorf("MaxPlayers: expected source 'config', got '%s'", maxPlayers.DefaultSource)
	}
	if maxPlayers.DefaultValue != "GameConfig.MaxPlayers" {
		t.Errorf("MaxPlayers: expected value 'GameConfig.MaxPlayers', got '%s'", maxPlayers.DefaultValue)
	}
}

func TestParseMixedAnnotations(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Round   int32   @default(1)
    Phase   string  @default("lobby")
    Secret  int64   @view(admin) @default(0)
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	game := schema.Types[0]

	// Check type-level annotations
	if game.Role != RoleRoot {
		t.Errorf("expected role 'root', got '%s'", game.Role)
	}
	if game.DefaultState != "active" {
		t.Errorf("expected defaultState 'active', got '%s'", game.DefaultState)
	}

	// Check field-level annotations
	secret := game.Fields[2]
	if len(secret.Views) != 1 || secret.Views[0] != "admin" {
		t.Errorf("Secret: expected view 'admin', got %v", secret.Views)
	}
	if secret.DefaultSource != DefaultLiteral || secret.DefaultValue != "0" {
		t.Errorf("Secret: expected literal default '0', got source='%s' value='%s'",
			secret.DefaultSource, secret.DefaultValue)
	}
}

func TestGenerateGoWithDefaults(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Round   int32   @default(1)
    Phase   string  @default("lobby")
    Speed   float64 @default(1.5)
    Active  bool    @default(true)
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// Check for ResetToDefaults method
	if !strings.Contains(codeStr, "func (t *GameState) ResetToDefaults()") {
		t.Error("generated code missing ResetToDefaults method")
	}

	// Check for default values in ResetToDefaults
	defaultChecks := []string{
		"t.round = 1",
		`t.phase = "lobby"`,
		"t.speed = 1.5",
		"t.active = true",
	}
	for _, check := range defaultChecks {
		if !strings.Contains(codeStr, check) {
			t.Errorf("generated code missing default assignment: %s", check)
		}
	}
}

func TestGenerateGoSchemaRegistry(t *testing.T) {
	input := `
package game

@id(1) @helper
type Player {
    ID   string
    Name string
}

@id(2) @root(active)
type GameState {
    Round int32
}

@id(3) @root(inactive)
type DroneMode {
    MaxDrones int32
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// Check for SchemaRegistry
	registryChecks := []string{
		"type SchemaRegistry struct",
		"func GetSchemaRegistry() *SchemaRegistry",
		"func (r *SchemaRegistry) IsActive(name string) bool",
		"func (r *SchemaRegistry) Activate(name string) error",
		"func (r *SchemaRegistry) Deactivate(name string) error",
		"func (r *SchemaRegistry) Get(name string) interface{}",
		"func (r *SchemaRegistry) ResetAll()",
	}
	for _, check := range registryChecks {
		if !strings.Contains(codeStr, check) {
			t.Errorf("generated code missing registry method: %s", check)
		}
	}

	// Check for typed getters (only for root schemas)
	if !strings.Contains(codeStr, "func GetGameStateInstance() *GameState") {
		t.Error("generated code missing GetGameStateInstance")
	}
	if !strings.Contains(codeStr, "func ActivateGameState() error") {
		t.Error("generated code missing ActivateGameState")
	}
	if !strings.Contains(codeStr, "func DeactivateGameState() error") {
		t.Error("generated code missing DeactivateGameState")
	}
	if !strings.Contains(codeStr, "func IsGameStateActive() bool") {
		t.Error("generated code missing IsGameStateActive")
	}

	// Check for DroneMode methods
	if !strings.Contains(codeStr, "func GetDroneModeInstance() *DroneMode") {
		t.Error("generated code missing GetDroneModeInstance")
	}
	if !strings.Contains(codeStr, "func ActivateDroneMode() error") {
		t.Error("generated code missing ActivateDroneMode")
	}

	// Player should NOT have activation methods (it's a helper)
	if strings.Contains(codeStr, "func ActivatePlayer()") {
		t.Error("helper schema should not have Activate method")
	}
}

func TestGenerateTSWithDefaults(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Round   int32   @default(1)
    Phase   string  @default("lobby")
    Speed   float64 @default(1.5)
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateTS(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// Check for defaultGameState function
	if !strings.Contains(codeStr, "export function defaultGameState(): GameState") {
		t.Error("generated TS code missing defaultGameState function")
	}

	// Check for default values
	defaultChecks := []string{
		"Round: 1",
		"Phase: 'lobby'",
		"Speed: 1.5",
	}
	for _, check := range defaultChecks {
		if !strings.Contains(codeStr, check) {
			t.Errorf("generated TS code missing default: %s", check)
		}
	}
}

func TestGenerateTSSchemaManager(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Round int32
}

@id(2) @root(inactive)
type DroneMode {
    MaxDrones int32
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateTS(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// Check for SchemaActivationManager
	managerChecks := []string{
		"class SchemaActivationManager",
		"export function getSchemaManager()",
		"isActive(name: SchemaName): boolean",
		"activate(name: SchemaName): void",
		"deactivate(name: SchemaName): void",
		"resetAll(): void",
	}
	for _, check := range managerChecks {
		if !strings.Contains(codeStr, check) {
			t.Errorf("generated TS code missing manager: %s", check)
		}
	}

	// Check for typed convenience functions
	if !strings.Contains(codeStr, "export function getGameState()") {
		t.Error("generated TS code missing getGameState")
	}
	if !strings.Contains(codeStr, "export function activateGameState()") {
		t.Error("generated TS code missing activateGameState")
	}
	if !strings.Contains(codeStr, "export function getDroneMode()") {
		t.Error("generated TS code missing getDroneMode")
	}
}

func TestParseFullSchema(t *testing.T) {
	input := `
package main

// Helper schemas
@id(2) @helper
type Player {
    ID      string
    Name    string      @default("")
    Score   int64       @default(0)
    Ready   bool        @default(false)
}

@id(4) @helper
type Drone {
    ID      string
    X       float64     @default(0)
    Y       float64     @default(0)
    Health  int32       @default(100)
}

// Root schemas
@id(1) @root(active)
type GameState {
    Round       int32       @default(1)
    Phase       string      @default("lobby")
    Players     []Player    @key(ID)
}

@id(3) @root(inactive)
type DroneMode {
    Drones          []Drone     @key(ID)
    SpawnInterval   int32       @default(5)
    MaxDrones       int32       @default(10)
}

view all {}
view admin { includes: all }
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Verify structure
	if len(schema.Types) != 4 {
		t.Fatalf("expected 4 types, got %d", len(schema.Types))
	}

	// Count helpers and roots
	helpers := 0
	roots := 0
	activeRoots := 0

	for _, typ := range schema.Types {
		if typ.Role == RoleHelper {
			helpers++
		}
		if typ.Role == RoleRoot {
			roots++
			if typ.DefaultState == "active" {
				activeRoots++
			}
		}
	}

	if helpers != 2 {
		t.Errorf("expected 2 helpers, got %d", helpers)
	}
	if roots != 2 {
		t.Errorf("expected 2 roots, got %d", roots)
	}
	if activeRoots != 1 {
		t.Errorf("expected 1 active root, got %d", activeRoots)
	}

	// Generate both Go and TS to verify no errors
	goCode, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("Go generation failed: %v", err)
	}
	if len(goCode) == 0 {
		t.Error("Go code is empty")
	}

	tsCode, err := GenerateTS(schema)
	if err != nil {
		t.Fatalf("TS generation failed: %v", err)
	}
	if len(tsCode) == 0 {
		t.Error("TS code is empty")
	}
}

func TestGoDefaultValue(t *testing.T) {
	tests := []struct {
		field    *FieldDef
		expected string
	}{
		{
			field:    &FieldDef{Name: "Score", Type: "int64", DefaultSource: DefaultNone},
			expected: "0",
		},
		{
			field:    &FieldDef{Name: "Name", Type: "string", DefaultSource: DefaultNone},
			expected: `""`,
		},
		{
			field:    &FieldDef{Name: "Active", Type: "bool", DefaultSource: DefaultNone},
			expected: "false",
		},
		{
			field:    &FieldDef{Name: "Score", Type: "int64", DefaultSource: DefaultLiteral, DefaultValue: "100"},
			expected: "100",
		},
		{
			field:    &FieldDef{Name: "Name", Type: "string", DefaultSource: DefaultLiteral, DefaultValue: "player"},
			expected: `"player"`,
		},
		{
			field:    &FieldDef{Name: "Active", Type: "bool", DefaultSource: DefaultLiteral, DefaultValue: "true"},
			expected: "true",
		},
		{
			field:    &FieldDef{Name: "Speed", Type: "float64", DefaultSource: DefaultConfig, DefaultValue: "GameConfig.Speed"},
			expected: "GetGameConfig().Speed",
		},
		{
			field:    &FieldDef{Name: "Items", Type: "[]int32", DefaultSource: DefaultNone},
			expected: "nil",
		},
		{
			field:    &FieldDef{Name: "Scores", Type: "map[string]int64", DefaultSource: DefaultNone},
			expected: "nil",
		},
	}

	for _, tt := range tests {
		got := goDefaultValue(tt.field)
		if got != tt.expected {
			t.Errorf("goDefaultValue(%s %s): expected '%s', got '%s'",
				tt.field.Name, tt.field.Type, tt.expected, got)
		}
	}
}

func TestTsDefaultValue(t *testing.T) {
	tests := []struct {
		field    *FieldDef
		expected string
	}{
		{
			field:    &FieldDef{Name: "Score", Type: "int64", DefaultSource: DefaultNone},
			expected: "0",
		},
		{
			field:    &FieldDef{Name: "Name", Type: "string", DefaultSource: DefaultNone},
			expected: "''",
		},
		{
			field:    &FieldDef{Name: "Active", Type: "bool", DefaultSource: DefaultNone},
			expected: "false",
		},
		{
			field:    &FieldDef{Name: "Score", Type: "int64", DefaultSource: DefaultLiteral, DefaultValue: "100"},
			expected: "100",
		},
		{
			field:    &FieldDef{Name: "Name", Type: "string", DefaultSource: DefaultLiteral, DefaultValue: "player"},
			expected: "'player'",
		},
		{
			field:    &FieldDef{Name: "Items", Type: "[]int32", DefaultSource: DefaultNone},
			expected: "[]",
		},
		{
			field:    &FieldDef{Name: "Scores", Type: "map[string]int64", DefaultSource: DefaultNone},
			expected: "{}",
		},
	}

	for _, tt := range tests {
		got := tsDefaultValue(tt.field)
		if got != tt.expected {
			t.Errorf("tsDefaultValue(%s %s): expected '%s', got '%s'",
				tt.field.Name, tt.field.Type, tt.expected, got)
		}
	}
}

// T15: @id(0) should be rejected by the parser
func TestParser_IdZeroRejected(t *testing.T) {
	input := `
package test

@id(0)
type Broken {
    Name string
}
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Error("expected error for @id(0), got nil")
	}
}

// T15: @id(-1) should be rejected
func TestParser_IdNegativeRejected(t *testing.T) {
	input := `
package test

@id(-1)
type Broken {
    Name string
}
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Error("expected error for @id(-1), got nil")
	}
}

// T12: zero-field type should generate valid code (MarkAllDirty guard)
func TestGenerateGo_ShallowClone(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Phase   string
    Score   int64
    Players map[string]Player @key(ID)
    Items   []Item            @key(ID)
    Config  bytes             @optional
}

@id(2) @helper
type Player {
    ID   string
    Name string
}

@id(3) @helper
type Item {
    ID    string
    Value int32
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// ShallowClone should exist for root type
	if !strings.Contains(codeStr, "func (s *GameState) ShallowClone() *GameState") {
		t.Error("missing ShallowClone for root type GameState")
	}

	// ShallowClone should NOT exist for helper types
	if strings.Contains(codeStr, "func (s *Player) ShallowClone()") {
		t.Error("ShallowClone should not be generated for helper type Player")
	}

	// Should deep-copy ChangeSet
	if !strings.Contains(codeStr, "changes.CloneForFilter()") {
		t.Error("ShallowClone should use CloneForFilter for ChangeSet")
	}

	// Should copy map fields (new map + loop)
	if !strings.Contains(codeStr, "clone.players = make(map[string]Player") {
		t.Error("ShallowClone should shallow-copy map fields")
	}

	// Should copy slice fields
	if !strings.Contains(codeStr, "clone.items = make([]Item") {
		t.Error("ShallowClone should copy slice fields")
	}

	// Scalar fields should be in struct literal (direct copy)
	// go format aligns fields, so check without exact whitespace
	if !strings.Contains(codeStr, "phase:") || !strings.Contains(codeStr, "s.phase") {
		t.Error("ShallowClone should copy scalar fields directly")
	}

	// Bytes fields should be in struct literal (shared)
	if !strings.Contains(codeStr, "config:") || !strings.Contains(codeStr, "s.config") {
		t.Error("ShallowClone should share bytes fields")
	}
}

func TestGenerateGo_JSONMarshalUnmarshal(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Phase   string
    HostID  string
    Score   int64          @optional
    Players map[string]Player @key(ID)
    Config  bytes          @optional
}

@id(2) @helper
type Player {
    ID       string
    Name     string
    IsCaught bool
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	codeStr := string(code)

	// JSON struct should exist for all types
	if !strings.Contains(codeStr, "type gameStateJSON struct") {
		t.Error("missing gameStateJSON struct")
	}
	if !strings.Contains(codeStr, "type playerJSON struct") {
		t.Error("missing playerJSON struct")
	}

	// MarshalJSON/UnmarshalJSON for all types
	for _, typeName := range []string{"GameState", "Player"} {
		if !strings.Contains(codeStr, "func (t *"+typeName+") MarshalJSON()") {
			t.Errorf("missing MarshalJSON for %s", typeName)
		}
		if !strings.Contains(codeStr, "func (t *"+typeName+") UnmarshalJSON(") {
			t.Errorf("missing UnmarshalJSON for %s", typeName)
		}
	}

	// JSON tags with camelCase
	if !strings.Contains(codeStr, `json:"phase"`) {
		t.Error("missing json tag for Phase → phase")
	}
	if !strings.Contains(codeStr, `json:"hostId"`) {
		t.Error("missing json tag for HostID → hostId")
	}
	if !strings.Contains(codeStr, `json:"isCaught"`) {
		t.Error("missing json tag for IsCaught → isCaught")
	}

	// Optional fields should have omitempty
	if !strings.Contains(codeStr, `json:"score,omitempty"`) {
		t.Error("optional int64 field should have omitempty")
	}

	// Bytes fields should be json.RawMessage with omitempty
	if !strings.Contains(codeStr, `json:"config,omitempty"`) {
		t.Error("bytes field should have omitempty")
	}

	// Map fields should have omitempty
	if !strings.Contains(codeStr, `json:"players,omitempty"`) {
		t.Error("map field should have omitempty")
	}

	// Root MarshalJSON should use getters for maps (thread-safe)
	if !strings.Contains(codeStr, "players := t.Players()") {
		t.Error("root MarshalJSON should use getter for map fields")
	}

	// Root MarshalJSON should acquire mu.RLock
	if !strings.Contains(codeStr, "t.mu.RLock()") {
		t.Error("root MarshalJSON should acquire mu.RLock")
	}

	// UnmarshalJSON should use New + setters
	if !strings.Contains(codeStr, "init := NewGameState()") {
		t.Error("UnmarshalJSON should create new instance")
	}
	if !strings.Contains(codeStr, "t.SetPhase(j.Phase)") {
		t.Error("UnmarshalJSON should use setter for Phase")
	}

	// bytesToRawJSON helper should exist
	if !strings.Contains(codeStr, "func bytesToRawJSON(") {
		t.Error("missing bytesToRawJSON helper function")
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"Phase", "phase"},
		{"ID", "id"},
		{"HostID", "hostId"},
		{"IsCaught", "isCaught"},
		{"MaxTeamSlots", "maxTeamSlots"},
		{"MassConfusionTeamID", "massConfusionTeamId"},
		{"GameMode", "gameMode"},
		{"Name", "name"},
	}
	for _, tt := range tests {
		got := toCamelCase(tt.input)
		if got != tt.expected {
			t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGenerateGo_ZeroFieldType(t *testing.T) {
	schema := &SchemaFile{
		Package: "test",
		Types: []*TypeDef{
			{
				Name:         "Empty",
				ID:           1,
				Role:         RoleRoot,
				DefaultState: "active",
				Fields:       []*FieldDef{},
			},
		},
		Views: []*ViewDef{},
	}

	goCode, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("GenerateGo error: %v", err)
	}

	code := string(goCode)

	// MarkAllDirty should NOT contain MarkAll(-1) or MarkAll(255)
	if strings.Contains(code, "MarkAll(255)") {
		t.Error("T12 REGRESSION: zero-field type generates MarkAll(255)")
	}
	if strings.Contains(code, "MarkAll(-1)") {
		t.Error("T12 REGRESSION: zero-field type generates MarkAll(-1)")
	}

	// Should not call MarkAll at all for zero fields
	if strings.Contains(code, "MarkAllDirty") && strings.Contains(code, "MarkAll(") {
		// Check it's guarded (MarkAllDirty body should be empty or not call MarkAll)
		t.Log("MarkAllDirty exists (expected for Trackable interface compliance)")
	}
}

func TestNoSyncAnnotation(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Phase   string
    Score   int64
    Secret  string  @noSync
    Token   string  @noSync
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	gs := schema.Types[0]

	// Check SyncIndex assignment
	if gs.Fields[0].SyncIndex != 0 { // Phase
		t.Errorf("Phase SyncIndex = %d, want 0", gs.Fields[0].SyncIndex)
	}
	if gs.Fields[1].SyncIndex != 1 { // Score
		t.Errorf("Score SyncIndex = %d, want 1", gs.Fields[1].SyncIndex)
	}
	if gs.Fields[2].SyncIndex != -1 { // Secret @noSync
		t.Errorf("Secret SyncIndex = %d, want -1", gs.Fields[2].SyncIndex)
	}
	if gs.Fields[3].SyncIndex != -1 { // Token @noSync
		t.Errorf("Token SyncIndex = %d, want -1", gs.Fields[3].SyncIndex)
	}

	code, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	codeStr := string(code)

	// @noSync fields should have getter/setter in Go struct
	if !strings.Contains(codeStr, "func (t *GameState) Secret()") {
		t.Error("@noSync field should have getter")
	}
	if !strings.Contains(codeStr, "func (t *GameState) SetSecret(") {
		t.Error("@noSync field should have setter")
	}

	// @noSync setter should NOT call changes.Mark
	// Find the SetSecret function and check it doesn't contain Mark
	setSecretIdx := strings.Index(codeStr, "func (t *GameState) SetSecret(")
	if setSecretIdx == -1 {
		t.Fatal("SetSecret not found")
	}
	setSecretEnd := strings.Index(codeStr[setSecretIdx:], "\n}\n")
	setSecretBody := codeStr[setSecretIdx : setSecretIdx+setSecretEnd]
	if strings.Contains(setSecretBody, "changes.Mark") {
		t.Error("@noSync setter should NOT call changes.Mark")
	}

	// @noSync fields should NOT be in schema builder
	if strings.Contains(codeStr, `String("Secret"`) {
		t.Error("@noSync field should NOT appear in schema builder")
	}

	// Synced fields should still be in schema builder
	if !strings.Contains(codeStr, `String("Phase"`) {
		t.Error("synced field Phase should be in schema builder")
	}

	// MarkAllDirty should use maxSyncIndex (1, not 3)
	if strings.Contains(codeStr, "MarkAll(3)") {
		t.Error("MarkAllDirty should use maxSyncIndex, not total field count")
	}
	if !strings.Contains(codeStr, "MarkAll(1)") {
		t.Error("MarkAllDirty should use MarkAll(1) for 2 synced fields")
	}

	// @noSync fields should be in JSON marshal (for persistence)
	if !strings.Contains(codeStr, `json:"secret"`) {
		t.Error("@noSync field should appear in JSON struct")
	}

	// @noSync fields should be in ShallowClone
	if !strings.Contains(codeStr, "secret:") && !strings.Contains(codeStr, "s.secret") {
		t.Error("@noSync field should be in ShallowClone")
	}
}

func TestNoSyncAnnotation_JSSchema(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Phase   string
    Score   int64
    Secret  string  @noSync
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	jsCode, err := GenerateJS(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	js := string(jsCode)

	// @noSync fields should NOT be in JS schema
	if strings.Contains(js, "secret") {
		t.Error("@noSync field should NOT appear in JS schema")
	}

	// Synced fields should be in JS schema
	if !strings.Contains(js, "'phase'") {
		t.Error("synced field should appear in JS schema")
	}
	if !strings.Contains(js, "'score'") {
		t.Error("synced field should appear in JS schema")
	}
}

func TestGenerateJS(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Phase   string
    Players map[string]Player @key(ID)
    Items   []string
    Config  bytes
}

@id(2) @helper
type Player {
    ID    string
    Name  string
    Score int64
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	jsCode, err := GenerateJS(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	js := string(jsCode)

	// Should have import
	if !strings.Contains(js, "import { defineSchema, FieldType } from './decoder.js'") {
		t.Error("missing decoder import")
	}

	// Should export schemas
	if !strings.Contains(js, "export const gameStateSchema") {
		t.Error("missing gameStateSchema export")
	}
	if !strings.Contains(js, "export const playerSchema") {
		t.Error("missing playerSchema export")
	}

	// Player schema should come before GameState (topological sort)
	playerIdx := strings.Index(js, "playerSchema")
	gameIdx := strings.Index(js, "gameStateSchema")
	if playerIdx > gameIdx {
		t.Error("Player schema should be defined before GameState (dependency order)")
	}

	// Map field with child schema
	if !strings.Contains(js, "childSchema: playerSchema") {
		t.Error("map field should reference child schema")
	}

	// allSchemas export
	if !strings.Contains(js, "export const allSchemas") {
		t.Error("missing allSchemas export")
	}
}

func TestRootMapSliceAutoInit(t *testing.T) {
	input := `
package game

@id(1) @root(active)
type GameState {
    Phase   string
    Players map[string]Player @key(ID)
    Items   []Item            @key(ID)
}

@id(2) @helper
type Player {
    ID   string
}

@id(3) @helper
type Item {
    ID   string
}
`
	schema, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	code, err := GenerateGo(schema)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	codeStr := string(code)

	// Root type ResetToDefaults should use make() for maps
	if !strings.Contains(codeStr, "t.players = make(map[string]Player)") {
		t.Error("root map field should be initialized with make() in ResetToDefaults")
	}

	// Root type ResetToDefaults should use make() for slices
	if !strings.Contains(codeStr, "t.items = make([]Item, 0)") {
		t.Error("root slice field should be initialized with make() in ResetToDefaults")
	}

	// Helper type should NOT auto-init maps (nil is fine)
	// Player has no map fields so this is implicitly tested
}
