package main

// ConfigFile represents a parsed .config file
type ConfigFile struct {
	Package string       `json:"package"`
	Configs []*ConfigDef `json:"configs"`
}

// ConfigDef represents a config group definition
type ConfigDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Fields      []*FieldDef `json:"fields"`
}

// FieldDef represents a config field definition
type FieldDef struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`              // int32, float64, string, bool, []string, etc.
	Default     interface{} `json:"default,omitempty"` // Default value
	Min         *float64    `json:"min,omitempty"`     // Min value for numbers
	Max         *float64    `json:"max,omitempty"`     // Max value for numbers
	Options     []string    `json:"options,omitempty"` // Valid options for strings
	Description string      `json:"description,omitempty"`
	Env         string      `json:"env,omitempty"`      // Environment variable name override
	Required    bool        `json:"required,omitempty"` // Field is required (no default)
}

// ParsedType holds parsed type information
type ParsedType struct {
	IsArray  bool
	IsMap    bool
	BaseType string
	ElemType string // For arrays/maps
	KeyType  string // For maps
}

// ParseType parses a type string like "[]string" or "map[string]int"
func ParseType(s string) ParsedType {
	pt := ParsedType{}

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

	// Primitive type
	pt.BaseType = s
	return pt
}

// GoType converts a config type to Go type
func GoType(t string) string {
	pt := ParseType(t)

	if pt.IsArray {
		return "[]" + GoType(pt.ElemType)
	}
	if pt.IsMap {
		return "map[" + GoType(pt.KeyType) + "]" + GoType(pt.ElemType)
	}

	switch pt.BaseType {
	case "duration":
		return "time.Duration"
	default:
		return pt.BaseType
	}
}

// TSType converts a config type to TypeScript type
func TSType(t string) string {
	pt := ParseType(t)

	if pt.IsArray {
		return TSType(pt.ElemType) + "[]"
	}
	if pt.IsMap {
		return "Record<" + TSType(pt.KeyType) + ", " + TSType(pt.ElemType) + ">"
	}

	switch pt.BaseType {
	case "int8", "int16", "int32", "uint8", "uint16", "uint32", "float32", "float64":
		return "number"
	case "int64", "uint64", "int", "uint", "duration":
		return "number"
	case "string":
		return "string"
	case "bool":
		return "boolean"
	default:
		return pt.BaseType
	}
}

// IsPrimitive checks if a type is a primitive
func IsPrimitive(t string) bool {
	switch t {
	case "int8", "int16", "int32", "int64", "int",
		"uint8", "uint16", "uint32", "uint64", "uint",
		"float32", "float64",
		"string", "bool", "duration":
		return true
	}
	return false
}

// DefaultForType returns the Go zero value for a type
func DefaultForType(t string) string {
	pt := ParseType(t)

	if pt.IsArray {
		return "nil"
	}
	if pt.IsMap {
		return "nil"
	}

	switch pt.BaseType {
	case "int8", "int16", "int32", "int64", "int",
		"uint8", "uint16", "uint32", "uint64", "uint":
		return "0"
	case "float32", "float64":
		return "0.0"
	case "string":
		return `""`
	case "bool":
		return "false"
	case "duration":
		return "0"
	default:
		return "nil"
	}
}
