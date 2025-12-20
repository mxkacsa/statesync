package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SchemaFile represents a parsed .schema file (mirrors schemagen)
type SchemaFile struct {
	Package string     `json:"package"`
	Types   []*TypeDef `json:"types"`
}

// TypeDef represents a schema type definition
type TypeDef struct {
	Name   string      `json:"name"`
	ID     int         `json:"id"`
	Fields []*FieldDef `json:"fields"`
}

// FieldDef represents a field definition
type FieldDef struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Key      string `json:"key,omitempty"`      // For arrays: key field for tracking
	Optional bool   `json:"optional,omitempty"` // Pointer/nullable
}

// SchemaContext provides schema-aware type information for code generation
type SchemaContext struct {
	Schema    *SchemaFile
	TypeIndex map[string]*TypeDef // Type name -> TypeDef
	RootType  *TypeDef            // The root state type (e.g., GameState)
}

// LoadSchema loads and parses a schema JSON file
func LoadSchema(path string) (*SchemaContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read schema file: %w", err)
	}

	var schema SchemaFile
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("cannot parse schema JSON: %w", err)
	}

	ctx := &SchemaContext{
		Schema:    &schema,
		TypeIndex: make(map[string]*TypeDef),
	}

	// Build type index
	for _, t := range schema.Types {
		ctx.TypeIndex[t.Name] = t
		// First type is usually the root state
		if ctx.RootType == nil {
			ctx.RootType = t
		}
	}

	return ctx, nil
}

// GetType returns the type definition for a type name
func (ctx *SchemaContext) GetType(name string) *TypeDef {
	return ctx.TypeIndex[name]
}

// GetField returns the field definition for a type and field name
func (ctx *SchemaContext) GetField(typeName, fieldName string) *FieldDef {
	t := ctx.GetType(typeName)
	if t == nil {
		return nil
	}
	for _, f := range t.Fields {
		if f.Name == fieldName {
			return f
		}
	}
	return nil
}

// ParsedFieldType holds parsed type information
type ParsedFieldType struct {
	IsArray   bool
	IsMap     bool
	IsPointer bool
	BaseType  string
	ElemType  string // For arrays/maps
	KeyType   string // For maps
}

// ParseFieldType parses a type string like "[]Player" or "map[string]int"
func ParseFieldType(s string) ParsedFieldType {
	pt := ParsedFieldType{}

	// Check for pointer
	if len(s) > 0 && s[0] == '*' {
		pt.IsPointer = true
		s = s[1:]
	}

	// Check for array
	if len(s) >= 2 && s[0] == '[' && s[1] == ']' {
		pt.IsArray = true
		pt.ElemType = s[2:]
		pt.BaseType = "array"
		return pt
	}

	// Check for map
	if len(s) >= 4 && s[0:4] == "map[" {
		pt.IsMap = true
		pt.BaseType = "map"
		// Parse map[K]V
		depth := 1
		for i := 4; i < len(s); i++ {
			if s[i] == '[' {
				depth++
			} else if s[i] == ']' {
				depth--
				if depth == 0 {
					pt.KeyType = s[4:i]
					pt.ElemType = s[i+1:]
					break
				}
			}
		}
		return pt
	}

	// Primitive or struct type
	pt.BaseType = s
	return pt
}

// IsPrimitive checks if a type is a primitive
func IsPrimitiveType(t string) bool {
	switch t {
	case "int8", "int16", "int32", "int64", "int",
		"uint8", "uint16", "uint32", "uint64", "uint",
		"float32", "float64",
		"string", "bool", "bytes", "[]byte":
		return true
	}
	return false
}

// FieldAccessInfo describes how to access/modify a field
type FieldAccessInfo struct {
	TypeName      string          // The type containing this field
	FieldName     string          // The field name
	FieldDef      *FieldDef       // The field definition
	ParsedType    ParsedFieldType // Parsed type info
	GetterName    string          // e.g., "Players"
	SetterName    string          // e.g., "SetPlayers"
	AppendName    string          // e.g., "AppendPlayers" (for arrays)
	RemoveAtName  string          // e.g., "RemovePlayersAt" (for arrays)
	UpdateAtName  string          // e.g., "UpdatePlayersAt" (for arrays)
	AtName        string          // e.g., "PlayersAt" (for arrays)
	LenName       string          // e.g., "PlayersLen" (for arrays)
	SetKeyName    string          // e.g., "SetScoresKey" (for maps)
	DeleteKeyName string          // e.g., "DeleteScoresKey" (for maps)
	GetKeyName    string          // e.g., "ScoresGet" (for maps)
}

// GetFieldAccessInfo returns access info for a field
func (ctx *SchemaContext) GetFieldAccessInfo(typeName, fieldName string) *FieldAccessInfo {
	field := ctx.GetField(typeName, fieldName)
	if field == nil {
		return nil
	}

	pt := ParseFieldType(field.Type)
	info := &FieldAccessInfo{
		TypeName:   typeName,
		FieldName:  fieldName,
		FieldDef:   field,
		ParsedType: pt,
		GetterName: fieldName,
	}

	if pt.IsArray {
		info.SetterName = "Set" + fieldName
		info.AppendName = "Append" + fieldName
		info.RemoveAtName = "Remove" + fieldName + "At"
		info.UpdateAtName = "Update" + fieldName + "At"
		info.AtName = fieldName + "At"
		info.LenName = fieldName + "Len"
	} else if pt.IsMap {
		info.SetterName = "Set" + fieldName
		info.SetKeyName = "Set" + fieldName + "Key"
		info.DeleteKeyName = "Delete" + fieldName + "Key"
		info.GetKeyName = fieldName + "Get"
	} else {
		info.SetterName = "Set" + fieldName
	}

	return info
}

// PathSegment represents one part of a field path
type PathSegment struct {
	FieldName  string      // The field name (e.g., "Players", "Cards")
	IndexType  string      // "", "literal", "variable", "key_lookup"
	IndexValue interface{} // The index value (int, string, or variable name)
	KeyField   string      // For key_lookup: the field to match (e.g., "ID")
}

// ParsedPath represents a parsed field path like "state.players[id].cards[0]"
type ParsedPath struct {
	Segments []PathSegment
	RawPath  string
}

// ParsePath parses a field path string
// Supported formats:
//   - "Players" - simple field access
//   - "Players[0]" - array index access
//   - "Players[playerId]" - variable index (detected by non-numeric)
//   - "Players[playerId:ID]" - key lookup (find element where ID == playerId)
//   - "Game.Players[id].Cards[0]" - nested path
func ParsePath(path string) (*ParsedPath, error) {
	result := &ParsedPath{
		RawPath:  path,
		Segments: []PathSegment{},
	}

	// Split by dot, but need to handle brackets
	current := ""
	for i := 0; i < len(path); i++ {
		ch := path[i]
		if ch == '.' {
			if current != "" {
				seg, err := parseSegment(current)
				if err != nil {
					return nil, err
				}
				result.Segments = append(result.Segments, seg)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	// Don't forget the last segment
	if current != "" {
		seg, err := parseSegment(current)
		if err != nil {
			return nil, err
		}
		result.Segments = append(result.Segments, seg)
	}

	return result, nil
}

// parseSegment parses a single segment like "Players[id]" or "Cards[0]"
func parseSegment(s string) (PathSegment, error) {
	seg := PathSegment{}

	// Check for bracket
	bracketStart := strings.Index(s, "[")
	if bracketStart == -1 {
		// Simple field access
		seg.FieldName = s
		return seg, nil
	}

	bracketEnd := strings.Index(s, "]")
	if bracketEnd == -1 || bracketEnd < bracketStart {
		return seg, fmt.Errorf("invalid bracket in path segment: %s", s)
	}

	seg.FieldName = s[:bracketStart]
	indexStr := s[bracketStart+1 : bracketEnd]

	// Check if it's a key lookup (contains :)
	if colonIdx := strings.Index(indexStr, ":"); colonIdx != -1 {
		seg.IndexType = "key_lookup"
		seg.IndexValue = indexStr[:colonIdx]
		seg.KeyField = indexStr[colonIdx+1:]
		return seg, nil
	}

	// Check if it's a number (literal index) or variable
	if isNumeric(indexStr) {
		seg.IndexType = "literal"
		// Parse as int
		var idx int
		fmt.Sscanf(indexStr, "%d", &idx)
		seg.IndexValue = idx
	} else {
		seg.IndexType = "variable"
		seg.IndexValue = indexStr
	}

	return seg, nil
}

func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// String returns a string representation of the path
func (p *ParsedPath) String() string {
	return p.RawPath
}
