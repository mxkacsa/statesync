package statesync

import (
	"fmt"
	"reflect"
)

// FieldType represents the wire type for encoding
type FieldType uint8

const (
	TypeInvalid FieldType = iota
	TypeInt8
	TypeInt16
	TypeInt32
	TypeInt64
	TypeUint8
	TypeUint16
	TypeUint32
	TypeUint64
	TypeFloat32
	TypeFloat64
	TypeString
	TypeBool
	TypeBytes     // []byte
	TypeStruct    // Nested Trackable struct
	TypeArray     // Slice of values
	TypeMap       // Map of values
	TypeVarInt    // Variable-length integer (like protobuf)
	TypeVarUint   // Variable-length unsigned integer
	TypeTimestamp // time.Time encoded as Unix millis
)

func (ft FieldType) String() string {
	names := []string{
		"invalid", "int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "string", "bool", "bytes",
		"struct", "array", "map", "varint", "varuint", "timestamp",
	}
	if int(ft) < len(names) {
		return names[ft]
	}
	return fmt.Sprintf("unknown(%d)", ft)
}

// Size returns the fixed size in bytes, or 0 for variable-length types
func (ft FieldType) Size() int {
	switch ft {
	case TypeInt8, TypeUint8, TypeBool:
		return 1
	case TypeInt16, TypeUint16:
		return 2
	case TypeInt32, TypeUint32, TypeFloat32:
		return 4
	case TypeInt64, TypeUint64, TypeFloat64, TypeTimestamp:
		return 8
	default:
		return 0 // Variable length
	}
}

// FieldMeta describes a single field in a schema
type FieldMeta struct {
	Index    uint8     // Field index (0-255)
	Name     string    // Field name (for JSON compat)
	Type     FieldType // Wire type
	ElemType FieldType // Element type (for arrays/maps)

	// For nested structs
	ChildSchema *Schema

	// For arrays with key-based tracking
	KeyField string // Field name to use as key (e.g., "ID")

	// For projections (future DSL feature)
	Projections []string // Which views can see this field
}

// Schema describes a trackable type
type Schema struct {
	ID     uint16         // Unique schema identifier
	Name   string         // Type name
	Fields []FieldMeta    // Fields in index order
	byName map[string]int // name -> field index lookup
}

// NewSchema creates a new schema definition
func NewSchema(id uint16, name string) *Schema {
	return &Schema{
		ID:     id,
		Name:   name,
		Fields: make([]FieldMeta, 0),
		byName: make(map[string]int),
	}
}

// AddField adds a field to the schema
func (s *Schema) AddField(field FieldMeta) *Schema {
	if field.Index != uint8(len(s.Fields)) {
		panic(fmt.Sprintf("field index %d doesn't match position %d", field.Index, len(s.Fields)))
	}
	s.byName[field.Name] = len(s.Fields)
	s.Fields = append(s.Fields, field)
	return s
}

// Field returns field meta by index
func (s *Schema) Field(index uint8) *FieldMeta {
	if int(index) >= len(s.Fields) {
		return nil
	}
	return &s.Fields[index]
}

// FieldByName returns field meta by name
func (s *Schema) FieldByName(name string) *FieldMeta {
	if idx, ok := s.byName[name]; ok {
		return &s.Fields[idx]
	}
	return nil
}

// FieldCount returns the number of fields
func (s *Schema) FieldCount() int {
	return len(s.Fields)
}

// MaxIndex returns the maximum field index
func (s *Schema) MaxIndex() uint8 {
	if len(s.Fields) == 0 {
		return 0
	}
	return uint8(len(s.Fields) - 1)
}

// Trackable interface for types that support change tracking
type Trackable interface {
	// Schema returns the type's schema definition
	Schema() *Schema

	// Changes returns the current ChangeSet
	Changes() *ChangeSet

	// ClearChanges clears all tracked changes
	ClearChanges()

	// MarkAllDirty marks all fields as changed (for initial sync)
	MarkAllDirty()

	// GetFieldValue returns the value of a field by index
	// SLOW PATH: uses interface{} boxing, prefer FastEncoder if available
	GetFieldValue(index uint8) interface{}
}

// FastEncoder is an optional interface for zero-allocation encoding
// Generated types should implement this for maximum performance
type FastEncoder interface {
	// EncodeChangesTo encodes only changed fields directly to the encoder
	// This avoids interface{} boxing and double type switches
	EncodeChangesTo(e *Encoder)

	// EncodeAllTo encodes all fields directly to the encoder
	EncodeAllTo(e *Encoder)
}

// SchemaRegistry maintains schema ID mappings
type SchemaRegistry struct {
	schemas map[uint16]*Schema
	byName  map[string]*Schema
	nextID  uint16
}

// NewSchemaRegistry creates a new registry
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		schemas: make(map[uint16]*Schema),
		byName:  make(map[string]*Schema),
		nextID:  1,
	}
}

// Register adds a schema to the registry
func (r *SchemaRegistry) Register(schema *Schema) {
	if schema.ID == 0 {
		schema.ID = r.nextID
		r.nextID++
	}
	r.schemas[schema.ID] = schema
	r.byName[schema.Name] = schema
}

// Get returns a schema by ID
func (r *SchemaRegistry) Get(id uint16) *Schema {
	return r.schemas[id]
}

// GetByName returns a schema by name
func (r *SchemaRegistry) GetByName(name string) *Schema {
	return r.byName[name]
}

// InferFieldType infers the FieldType from a Go reflect.Type
func InferFieldType(t reflect.Type) FieldType {
	switch t.Kind() {
	case reflect.Int8:
		return TypeInt8
	case reflect.Int16:
		return TypeInt16
	case reflect.Int32:
		return TypeInt32
	case reflect.Int, reflect.Int64:
		return TypeInt64
	case reflect.Uint8:
		return TypeUint8
	case reflect.Uint16:
		return TypeUint16
	case reflect.Uint32:
		return TypeUint32
	case reflect.Uint, reflect.Uint64:
		return TypeUint64
	case reflect.Float32:
		return TypeFloat32
	case reflect.Float64:
		return TypeFloat64
	case reflect.String:
		return TypeString
	case reflect.Bool:
		return TypeBool
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return TypeBytes
		}
		return TypeArray
	case reflect.Map:
		return TypeMap
	case reflect.Struct:
		// Check for time.Time
		if t.PkgPath() == "time" && t.Name() == "Time" {
			return TypeTimestamp
		}
		return TypeStruct
	case reflect.Ptr:
		return InferFieldType(t.Elem())
	default:
		return TypeInvalid
	}
}

// SchemaBuilder provides a fluent API for building schemas
type SchemaBuilder struct {
	schema *Schema
}

// NewSchemaBuilder creates a new builder
func NewSchemaBuilder(name string) *SchemaBuilder {
	return &SchemaBuilder{
		schema: NewSchema(0, name),
	}
}

// WithID sets the schema ID
func (b *SchemaBuilder) WithID(id uint16) *SchemaBuilder {
	b.schema.ID = id
	return b
}

// Int8 adds an int8 field
func (b *SchemaBuilder) Int8(name string) *SchemaBuilder {
	return b.field(name, TypeInt8)
}

// Int16 adds an int16 field
func (b *SchemaBuilder) Int16(name string) *SchemaBuilder {
	return b.field(name, TypeInt16)
}

// Int32 adds an int32 field
func (b *SchemaBuilder) Int32(name string) *SchemaBuilder {
	return b.field(name, TypeInt32)
}

// Int64 adds an int64 field
func (b *SchemaBuilder) Int64(name string) *SchemaBuilder {
	return b.field(name, TypeInt64)
}

// Uint8 adds a uint8 field
func (b *SchemaBuilder) Uint8(name string) *SchemaBuilder {
	return b.field(name, TypeUint8)
}

// Uint16 adds a uint16 field
func (b *SchemaBuilder) Uint16(name string) *SchemaBuilder {
	return b.field(name, TypeUint16)
}

// Uint32 adds a uint32 field
func (b *SchemaBuilder) Uint32(name string) *SchemaBuilder {
	return b.field(name, TypeUint32)
}

// Uint64 adds a uint64 field
func (b *SchemaBuilder) Uint64(name string) *SchemaBuilder {
	return b.field(name, TypeUint64)
}

// Float32 adds a float32 field
func (b *SchemaBuilder) Float32(name string) *SchemaBuilder {
	return b.field(name, TypeFloat32)
}

// Float64 adds a float64 field
func (b *SchemaBuilder) Float64(name string) *SchemaBuilder {
	return b.field(name, TypeFloat64)
}

// String adds a string field
func (b *SchemaBuilder) String(name string) *SchemaBuilder {
	return b.field(name, TypeString)
}

// Bool adds a bool field
func (b *SchemaBuilder) Bool(name string) *SchemaBuilder {
	return b.field(name, TypeBool)
}

// Bytes adds a []byte field
func (b *SchemaBuilder) Bytes(name string) *SchemaBuilder {
	return b.field(name, TypeBytes)
}

// Struct adds a nested struct field
func (b *SchemaBuilder) Struct(name string, childSchema *Schema) *SchemaBuilder {
	b.schema.AddField(FieldMeta{
		Index:       uint8(len(b.schema.Fields)),
		Name:        name,
		Type:        TypeStruct,
		ChildSchema: childSchema,
	})
	return b
}

// Array adds an array field
func (b *SchemaBuilder) Array(name string, elemType FieldType, childSchema *Schema) *SchemaBuilder {
	b.schema.AddField(FieldMeta{
		Index:       uint8(len(b.schema.Fields)),
		Name:        name,
		Type:        TypeArray,
		ElemType:    elemType,
		ChildSchema: childSchema,
	})
	return b
}

// ArrayByKey adds an array field with key-based tracking
func (b *SchemaBuilder) ArrayByKey(name string, elemType FieldType, childSchema *Schema, keyField string) *SchemaBuilder {
	b.schema.AddField(FieldMeta{
		Index:       uint8(len(b.schema.Fields)),
		Name:        name,
		Type:        TypeArray,
		ElemType:    elemType,
		ChildSchema: childSchema,
		KeyField:    keyField,
	})
	return b
}

// Map adds a map field
func (b *SchemaBuilder) Map(name string, elemType FieldType, childSchema *Schema) *SchemaBuilder {
	b.schema.AddField(FieldMeta{
		Index:       uint8(len(b.schema.Fields)),
		Name:        name,
		Type:        TypeMap,
		ElemType:    elemType,
		ChildSchema: childSchema,
	})
	return b
}

// Build returns the completed schema
func (b *SchemaBuilder) Build() *Schema {
	return b.schema
}

func (b *SchemaBuilder) field(name string, typ FieldType) *SchemaBuilder {
	b.schema.AddField(FieldMeta{
		Index: uint8(len(b.schema.Fields)),
		Name:  name,
		Type:  typ,
	})
	return b
}
