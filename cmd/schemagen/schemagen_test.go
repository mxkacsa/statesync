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
