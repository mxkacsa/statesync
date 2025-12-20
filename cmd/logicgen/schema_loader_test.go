package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ============================================================================
// ParseFieldType Tests
// ============================================================================

func TestParseFieldType_Primitives(t *testing.T) {
	tests := []struct {
		input    string
		wantBase string
		isArray  bool
		isMap    bool
		isPtr    bool
	}{
		{"string", "string", false, false, false},
		{"int", "int", false, false, false},
		{"int32", "int32", false, false, false},
		{"int64", "int64", false, false, false},
		{"float32", "float32", false, false, false},
		{"float64", "float64", false, false, false},
		{"bool", "bool", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pt := ParseFieldType(tt.input)
			if pt.BaseType != tt.wantBase {
				t.Errorf("BaseType = %v, want %v", pt.BaseType, tt.wantBase)
			}
			if pt.IsArray != tt.isArray {
				t.Errorf("IsArray = %v, want %v", pt.IsArray, tt.isArray)
			}
			if pt.IsMap != tt.isMap {
				t.Errorf("IsMap = %v, want %v", pt.IsMap, tt.isMap)
			}
			if pt.IsPointer != tt.isPtr {
				t.Errorf("IsPointer = %v, want %v", pt.IsPointer, tt.isPtr)
			}
		})
	}
}

func TestParseFieldType_Arrays(t *testing.T) {
	tests := []struct {
		input    string
		elemType string
	}{
		{"[]string", "string"},
		{"[]int", "int"},
		{"[]Player", "Player"},
		{"[]SurpriseBox", "SurpriseBox"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pt := ParseFieldType(tt.input)
			if !pt.IsArray {
				t.Error("IsArray should be true")
			}
			if pt.BaseType != "array" {
				t.Errorf("BaseType = %v, want array", pt.BaseType)
			}
			if pt.ElemType != tt.elemType {
				t.Errorf("ElemType = %v, want %v", pt.ElemType, tt.elemType)
			}
		})
	}
}

func TestParseFieldType_Maps(t *testing.T) {
	tests := []struct {
		input    string
		keyType  string
		elemType string
	}{
		{"map[string]int", "string", "int"},
		{"map[string]Player", "string", "Player"},
		{"map[int]string", "int", "string"},
		{"map[string]bool", "string", "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pt := ParseFieldType(tt.input)
			if !pt.IsMap {
				t.Error("IsMap should be true")
			}
			if pt.BaseType != "map" {
				t.Errorf("BaseType = %v, want map", pt.BaseType)
			}
			if pt.KeyType != tt.keyType {
				t.Errorf("KeyType = %v, want %v", pt.KeyType, tt.keyType)
			}
			if pt.ElemType != tt.elemType {
				t.Errorf("ElemType = %v, want %v", pt.ElemType, tt.elemType)
			}
		})
	}
}

func TestParseFieldType_Pointers(t *testing.T) {
	tests := []struct {
		input    string
		baseType string
		isPtr    bool
	}{
		{"*string", "string", true},
		{"*Player", "Player", true},
		{"*int", "int", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pt := ParseFieldType(tt.input)
			if pt.IsPointer != tt.isPtr {
				t.Errorf("IsPointer = %v, want %v", pt.IsPointer, tt.isPtr)
			}
			if pt.BaseType != tt.baseType {
				t.Errorf("BaseType = %v, want %v", pt.BaseType, tt.baseType)
			}
		})
	}
}

// ============================================================================
// IsPrimitiveType Tests
// ============================================================================

func TestIsPrimitiveType(t *testing.T) {
	primitives := []string{
		"int8", "int16", "int32", "int64", "int",
		"uint8", "uint16", "uint32", "uint64", "uint",
		"float32", "float64",
		"string", "bool", "bytes", "[]byte",
	}

	for _, p := range primitives {
		t.Run(p, func(t *testing.T) {
			if !IsPrimitiveType(p) {
				t.Errorf("%s should be primitive", p)
			}
		})
	}

	notPrimitives := []string{"Player", "GameState", "[]Player", "map[string]int"}
	for _, np := range notPrimitives {
		t.Run(np, func(t *testing.T) {
			if IsPrimitiveType(np) {
				t.Errorf("%s should not be primitive", np)
			}
		})
	}
}

// ============================================================================
// ParsePath Tests
// ============================================================================

func TestParsePath_Simple(t *testing.T) {
	path, err := ParsePath("Players")
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}
	if len(path.Segments) != 1 {
		t.Fatalf("Segments len = %d, want 1", len(path.Segments))
	}
	if path.Segments[0].FieldName != "Players" {
		t.Errorf("FieldName = %v, want Players", path.Segments[0].FieldName)
	}
	if path.Segments[0].IndexType != "" {
		t.Errorf("IndexType = %v, want empty", path.Segments[0].IndexType)
	}
}

func TestParsePath_Nested(t *testing.T) {
	path, err := ParsePath("Game.Players.Score")
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}
	if len(path.Segments) != 3 {
		t.Fatalf("Segments len = %d, want 3", len(path.Segments))
	}
	if path.Segments[0].FieldName != "Game" {
		t.Errorf("Segment 0 FieldName = %v, want Game", path.Segments[0].FieldName)
	}
	if path.Segments[1].FieldName != "Players" {
		t.Errorf("Segment 1 FieldName = %v, want Players", path.Segments[1].FieldName)
	}
	if path.Segments[2].FieldName != "Score" {
		t.Errorf("Segment 2 FieldName = %v, want Score", path.Segments[2].FieldName)
	}
}

func TestParsePath_LiteralIndex(t *testing.T) {
	path, err := ParsePath("Cards[0]")
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}
	if len(path.Segments) != 1 {
		t.Fatalf("Segments len = %d, want 1", len(path.Segments))
	}
	seg := path.Segments[0]
	if seg.FieldName != "Cards" {
		t.Errorf("FieldName = %v, want Cards", seg.FieldName)
	}
	if seg.IndexType != "literal" {
		t.Errorf("IndexType = %v, want literal", seg.IndexType)
	}
	if seg.IndexValue != 0 {
		t.Errorf("IndexValue = %v, want 0", seg.IndexValue)
	}
}

func TestParsePath_VariableIndex(t *testing.T) {
	path, err := ParsePath("Players[playerId]")
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}
	seg := path.Segments[0]
	if seg.IndexType != "variable" {
		t.Errorf("IndexType = %v, want variable", seg.IndexType)
	}
	if seg.IndexValue != "playerId" {
		t.Errorf("IndexValue = %v, want playerId", seg.IndexValue)
	}
}

func TestParsePath_KeyLookup(t *testing.T) {
	path, err := ParsePath("Players[playerId:ID]")
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}
	seg := path.Segments[0]
	if seg.IndexType != "key_lookup" {
		t.Errorf("IndexType = %v, want key_lookup", seg.IndexType)
	}
	if seg.IndexValue != "playerId" {
		t.Errorf("IndexValue = %v, want playerId", seg.IndexValue)
	}
	if seg.KeyField != "ID" {
		t.Errorf("KeyField = %v, want ID", seg.KeyField)
	}
}

func TestParsePath_Complex(t *testing.T) {
	// Path "Game.Players[id].Cards[0]" splits into 3 segments:
	// 1. Game (simple field)
	// 2. Players[id] (field with variable index)
	// 3. Cards[0] (field with literal index)
	path, err := ParsePath("Game.Players[id].Cards[0]")
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}
	if len(path.Segments) != 3 {
		t.Fatalf("Segments len = %d, want 3", len(path.Segments))
	}

	// Game
	if path.Segments[0].FieldName != "Game" {
		t.Errorf("Segment 0 = %v, want Game", path.Segments[0].FieldName)
	}

	// Players[id]
	if path.Segments[1].FieldName != "Players" {
		t.Errorf("Segment 1 = %v, want Players", path.Segments[1].FieldName)
	}
	if path.Segments[1].IndexType != "variable" {
		t.Errorf("Segment 1 IndexType = %v, want variable", path.Segments[1].IndexType)
	}

	// Cards[0]
	if path.Segments[2].FieldName != "Cards" {
		t.Errorf("Segment 2 = %v, want Cards", path.Segments[2].FieldName)
	}
	if path.Segments[2].IndexType != "literal" {
		t.Errorf("Segment 2 IndexType = %v, want literal", path.Segments[2].IndexType)
	}
}

func TestParsePath_InvalidBracket(t *testing.T) {
	_, err := ParsePath("Players[invalid")
	if err == nil {
		t.Error("Expected error for invalid bracket")
	}
}

// ============================================================================
// LoadSchema Tests
// ============================================================================

func TestLoadSchema_ValidFile(t *testing.T) {
	// Use existing test file
	schemaPath := filepath.Join("testdata", "permissions_test.schema.json")
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Skip("Test schema file not found")
	}

	ctx, err := LoadSchema(schemaPath)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	if ctx.Schema == nil {
		t.Fatal("Schema should not be nil")
	}
	if ctx.RootType == nil {
		t.Fatal("RootType should not be nil")
	}
	if len(ctx.TypeIndex) == 0 {
		t.Error("TypeIndex should not be empty")
	}
}

func TestLoadSchema_InvalidPath(t *testing.T) {
	_, err := LoadSchema("nonexistent.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadSchema_InvalidJSON(t *testing.T) {
	// Create a temp file with invalid JSON
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(tmpFile, []byte("{invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err = LoadSchema(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// ============================================================================
// SchemaContext Methods Tests
// ============================================================================

func TestSchemaContext_GetType(t *testing.T) {
	ctx := &SchemaContext{
		Schema: &SchemaFile{},
		TypeIndex: map[string]*TypeDef{
			"Player": {Name: "Player", ID: 1},
			"Game":   {Name: "Game", ID: 2},
		},
	}

	player := ctx.GetType("Player")
	if player == nil {
		t.Fatal("GetType(Player) returned nil")
	}
	if player.Name != "Player" {
		t.Errorf("Name = %v, want Player", player.Name)
	}

	unknown := ctx.GetType("Unknown")
	if unknown != nil {
		t.Error("GetType(Unknown) should return nil")
	}
}

func TestSchemaContext_GetField(t *testing.T) {
	ctx := &SchemaContext{
		Schema: &SchemaFile{},
		TypeIndex: map[string]*TypeDef{
			"Player": {
				Name: "Player",
				ID:   1,
				Fields: []*FieldDef{
					{Name: "ID", Type: "string"},
					{Name: "Score", Type: "int32"},
				},
			},
		},
	}

	field := ctx.GetField("Player", "Score")
	if field == nil {
		t.Fatal("GetField returned nil")
	}
	if field.Name != "Score" {
		t.Errorf("Name = %v, want Score", field.Name)
	}
	if field.Type != "int32" {
		t.Errorf("Type = %v, want int32", field.Type)
	}

	// Non-existent field
	unknown := ctx.GetField("Player", "Unknown")
	if unknown != nil {
		t.Error("GetField(Unknown) should return nil")
	}

	// Non-existent type
	noType := ctx.GetField("NoType", "Field")
	if noType != nil {
		t.Error("GetField for non-existent type should return nil")
	}
}

func TestSchemaContext_GetFieldAccessInfo(t *testing.T) {
	ctx := &SchemaContext{
		Schema: &SchemaFile{},
		TypeIndex: map[string]*TypeDef{
			"Game": {
				Name: "Game",
				ID:   1,
				Fields: []*FieldDef{
					{Name: "Score", Type: "int32"},
					{Name: "Players", Type: "[]Player"},
					{Name: "Scores", Type: "map[string]int"},
				},
			},
		},
	}

	// Simple field
	scoreInfo := ctx.GetFieldAccessInfo("Game", "Score")
	if scoreInfo == nil {
		t.Fatal("GetFieldAccessInfo returned nil")
	}
	if scoreInfo.GetterName != "Score" {
		t.Errorf("GetterName = %v, want Score", scoreInfo.GetterName)
	}
	if scoreInfo.SetterName != "SetScore" {
		t.Errorf("SetterName = %v, want SetScore", scoreInfo.SetterName)
	}

	// Array field
	playersInfo := ctx.GetFieldAccessInfo("Game", "Players")
	if playersInfo == nil {
		t.Fatal("GetFieldAccessInfo returned nil for array")
	}
	if !playersInfo.ParsedType.IsArray {
		t.Error("ParsedType.IsArray should be true")
	}
	if playersInfo.AppendName != "AppendPlayers" {
		t.Errorf("AppendName = %v, want AppendPlayers", playersInfo.AppendName)
	}
	if playersInfo.RemoveAtName != "RemovePlayersAt" {
		t.Errorf("RemoveAtName = %v, want RemovePlayersAt", playersInfo.RemoveAtName)
	}

	// Map field
	scoresInfo := ctx.GetFieldAccessInfo("Game", "Scores")
	if scoresInfo == nil {
		t.Fatal("GetFieldAccessInfo returned nil for map")
	}
	if !scoresInfo.ParsedType.IsMap {
		t.Error("ParsedType.IsMap should be true")
	}
	if scoresInfo.SetKeyName != "SetScoresKey" {
		t.Errorf("SetKeyName = %v, want SetScoresKey", scoresInfo.SetKeyName)
	}
}

// ============================================================================
// isNumeric Tests
// ============================================================================

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"0", true},
		{"123", true},
		{"999", true},
		{"", false},
		{"abc", false},
		{"1a2", false},
		{"12.3", false},
		{"-1", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isNumeric(tt.input)
			if got != tt.want {
				t.Errorf("isNumeric(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ============================================================================
// ParsedPath String Tests
// ============================================================================

func TestParsedPath_String(t *testing.T) {
	path, _ := ParsePath("Game.Players[0]")
	if path.String() != "Game.Players[0]" {
		t.Errorf("String() = %v, want Game.Players[0]", path.String())
	}
}
