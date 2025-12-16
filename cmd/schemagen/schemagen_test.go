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
