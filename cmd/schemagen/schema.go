package main

// SchemaFile represents a parsed .schema file
type SchemaFile struct {
	Package string     `json:"package"`
	Types   []*TypeDef `json:"types"`
	Views   []*ViewDef `json:"views,omitempty"`
}

// SchemaRole indicates whether a type is a root (activatable) or helper type
type SchemaRole string

const (
	RoleHelper SchemaRole = "helper" // Used by other schemas, not independently activatable
	RoleRoot   SchemaRole = "root"   // Main schema, can be activated/deactivated
)

// TypeDef represents a schema type definition
type TypeDef struct {
	Name         string      `json:"name"`
	ID           int         `json:"id"`
	Fields       []*FieldDef `json:"fields"`
	Role         SchemaRole  `json:"role,omitempty"`         // helper or root (default: helper)
	DefaultState string      `json:"defaultState,omitempty"` // For root: "active" or "inactive" (default: inactive)
	OwnerField   string      `json:"ownerField,omitempty"`   // Field that holds the owner player ID (for @write(owner) checks)
}

// IsRoot returns true if this is an activatable root schema
func (t *TypeDef) IsRoot() bool {
	return t.Role == RoleRoot
}

// IsActiveByDefault returns true if this root schema should be active on start
func (t *TypeDef) IsActiveByDefault() bool {
	return t.Role == RoleRoot && t.DefaultState == "active"
}

// DefaultSource indicates where a default value comes from
type DefaultSource string

const (
	DefaultNone    DefaultSource = ""        // No default (zero value)
	DefaultLiteral DefaultSource = "literal" // Hardcoded value: @default(42)
	DefaultConfig  DefaultSource = "config"  // From config: @default(config:GameConfig.Speed)
)

// AutoGenType specifies how a field value is automatically generated
type AutoGenType string

const (
	AutoGenNone AutoGenType = ""     // No auto-generation
	AutoGenUUID AutoGenType = "uuid" // Generate UUID v4
)

// WritePermission specifies who can modify a field
type WritePermission string

const (
	WriteAnyone WritePermission = ""       // Anyone can write (default)
	WriteServer WritePermission = "server" // Only server/rules can write, no player
	WriteOwner  WritePermission = "owner"  // Only the entity owner can write
)

// FieldDef represents a field definition
type FieldDef struct {
	Name          string          `json:"name"`
	Type          string          `json:"type"`                    // int32, string, uuid, []Player, map[string]int, etc.
	Key           string          `json:"key,omitempty"`           // For arrays: key field for tracking
	Views         []string        `json:"views,omitempty"`         // Which views can see this field
	Write         WritePermission `json:"write,omitempty"`         // Who can modify this field (server, owner, or anyone)
	Optional      bool            `json:"optional,omitempty"`      // Pointer/nullable
	DefaultSource DefaultSource   `json:"defaultSource,omitempty"` // Where default comes from
	DefaultValue  string          `json:"defaultValue,omitempty"`  // Literal value or config path (e.g., "GameConfig.Speed")
	AutoGen       AutoGenType     `json:"autoGen,omitempty"`       // Auto-generation type (e.g., "uuid")
}

// ViewDef represents a view/projection definition
type ViewDef struct {
	Name     string   `json:"name"`
	Includes []string `json:"includes,omitempty"` // Views this view includes
}

// ParsedType holds parsed type information
type ParsedType struct {
	IsArray   bool
	IsMap     bool
	IsPointer bool
	BaseType  string
	ElemType  string // For arrays/maps
	KeyType   string // For maps
}

// ParseType parses a type string like "[]Player" or "map[string]int"
func ParseType(s string) ParsedType {
	pt := ParsedType{}

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
func IsPrimitive(t string) bool {
	switch t {
	case "int8", "int16", "int32", "int64", "int",
		"uint8", "uint16", "uint32", "uint64", "uint",
		"float32", "float64",
		"string", "bool", "bytes", "uuid":
		return true
	}
	return false
}

// GoType converts a schema type to Go type
func GoType(t string) string {
	pt := ParseType(t)

	if pt.IsPointer {
		return "*" + GoType(t[1:])
	}
	if pt.IsArray {
		return "[]" + GoType(pt.ElemType)
	}
	if pt.IsMap {
		return "map[" + GoType(pt.KeyType) + "]" + GoType(pt.ElemType)
	}

	switch pt.BaseType {
	case "bytes":
		return "[]byte"
	case "uuid":
		return "string" // UUID is stored as string in Go
	default:
		return pt.BaseType
	}
}

// TSType converts a schema type to TypeScript type
func TSType(t string) string {
	pt := ParseType(t)

	if pt.IsPointer {
		return TSType(t[1:]) + " | null"
	}
	if pt.IsArray {
		return TSType(pt.ElemType) + "[]"
	}
	if pt.IsMap {
		return "Record<" + TSType(pt.KeyType) + ", " + TSType(pt.ElemType) + ">"
	}

	switch pt.BaseType {
	case "int8", "int16", "int32", "uint8", "uint16", "uint32", "float32", "float64":
		return "number"
	case "int64", "uint64", "int", "uint":
		return "number" // or bigint for large values
	case "string", "uuid":
		return "string"
	case "bool":
		return "boolean"
	case "bytes":
		return "Uint8Array"
	default:
		return pt.BaseType // Struct name
	}
}

// FieldTypeEnum returns the statediff.FieldType enum value
func FieldTypeEnum(t string) string {
	pt := ParseType(t)

	if pt.IsArray {
		return "TypeArray"
	}
	if pt.IsMap {
		return "TypeMap"
	}

	switch pt.BaseType {
	case "int8":
		return "TypeInt8"
	case "int16":
		return "TypeInt16"
	case "int32":
		return "TypeInt32"
	case "int64", "int":
		return "TypeInt64"
	case "uint8":
		return "TypeUint8"
	case "uint16":
		return "TypeUint16"
	case "uint32":
		return "TypeUint32"
	case "uint64", "uint":
		return "TypeUint64"
	case "float32":
		return "TypeFloat32"
	case "float64":
		return "TypeFloat64"
	case "string", "uuid":
		return "TypeString"
	case "bool":
		return "TypeBool"
	case "bytes":
		return "TypeBytes"
	default:
		return "TypeStruct"
	}
}

// TSFieldType returns the TypeScript FieldType enum value
func TSFieldType(t string) string {
	pt := ParseType(t)

	if pt.IsArray {
		return "FieldType.Array"
	}
	if pt.IsMap {
		return "FieldType.Map"
	}

	switch pt.BaseType {
	case "int8":
		return "FieldType.Int8"
	case "int16":
		return "FieldType.Int16"
	case "int32":
		return "FieldType.Int32"
	case "int64", "int":
		return "FieldType.Int64"
	case "uint8":
		return "FieldType.Uint8"
	case "uint16":
		return "FieldType.Uint16"
	case "uint32":
		return "FieldType.Uint32"
	case "uint64", "uint":
		return "FieldType.Uint64"
	case "float32":
		return "FieldType.Float32"
	case "float64":
		return "FieldType.Float64"
	case "string", "uuid":
		return "FieldType.String"
	case "bool":
		return "FieldType.Bool"
	case "bytes":
		return "FieldType.Bytes"
	default:
		return "FieldType.Struct"
	}
}
