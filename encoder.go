package statesync

import (
	"encoding/binary"
	"math"
	"sort"
)

// Binary protocol constants
const (
	// Message types
	MsgFullState  uint8 = 0x01
	MsgPatch      uint8 = 0x02
	MsgPatchBatch uint8 = 0x03

	// Array/Map encoding modes (first byte after field index)
	ArrayModeIncremental uint8 = 0x00 // Incremental changes follow
	ArrayModeFull        uint8 = 0x01 // Full replacement follows

	// Initial buffer size
	DefaultBufferSize = 4096
)

// Encoder encodes Trackable objects to binary format
type Encoder struct {
	buf      []byte
	pos      int
	registry *SchemaRegistry
}

// NewEncoder creates a new encoder
func NewEncoder(registry *SchemaRegistry) *Encoder {
	return &Encoder{
		buf:      make([]byte, DefaultBufferSize),
		registry: registry,
	}
}

// Reset resets the encoder for reuse
func (e *Encoder) Reset() {
	e.pos = 0
}

// Bytes returns the encoded bytes
func (e *Encoder) Bytes() []byte {
	return e.buf[:e.pos]
}

// grow ensures buffer has capacity for n more bytes
func (e *Encoder) grow(n int) {
	needed := e.pos + n
	if needed <= len(e.buf) {
		return
	}
	// Smart growth strategy:
	// - Small buffers (<64KB): double the size for amortized O(1)
	// - Medium buffers (64KB-1MB): grow by 50%
	// - Large buffers (>1MB): grow by 25% to avoid excessive memory usage
	newSize := len(e.buf)
	for newSize < needed {
		if newSize < 65536 {
			newSize *= 2
		} else if newSize < 1048576 {
			newSize += newSize / 2
		} else {
			newSize += newSize / 4
		}
	}
	newBuf := make([]byte, newSize)
	copy(newBuf, e.buf[:e.pos])
	e.buf = newBuf
}

// Encode encodes only the changed fields of a Trackable object
func (e *Encoder) Encode(t Trackable) []byte {
	e.Reset()

	changes := t.Changes()
	if !changes.HasChanges() {
		return nil
	}

	schema := t.Schema()

	// Message type
	e.writeByte(MsgPatch)

	// Schema ID
	e.writeUint16(schema.ID)

	// FAST PATH: Use generated encoder if available (no interface{} boxing)
	if fast, ok := t.(FastEncoder); ok {
		fast.EncodeChangesTo(e)
		return e.Bytes()
	}

	// SLOW PATH: Use reflection-based encoding
	e.encodeChanges(t, schema, changes)

	return e.Bytes()
}

// EncodeAll encodes all fields of a Trackable object (for initial sync)
func (e *Encoder) EncodeAll(t Trackable) []byte {
	e.Reset()

	schema := t.Schema()

	// Message type
	e.writeByte(MsgFullState)

	// Schema ID
	e.writeUint16(schema.ID)

	// FAST PATH: Use generated encoder if available
	if fast, ok := t.(FastEncoder); ok {
		e.writeByte(uint8(len(schema.Fields)))
		fast.EncodeAllTo(e)
		return e.Bytes()
	}

	// SLOW PATH: Use reflection-based encoding
	// Number of fields
	e.writeByte(uint8(len(schema.Fields)))

	// Encode all fields
	for i := range schema.Fields {
		field := &schema.Fields[i]
		value := t.GetFieldValue(uint8(i))
		e.encodeField(field, value)
	}

	return e.Bytes()
}

// encodeChanges encodes only the changed fields
func (e *Encoder) encodeChanges(t Trackable, schema *Schema, changes *ChangeSet) {
	changedFields := changes.ChangedFields()

	// Number of changes
	e.writeVarUint(uint64(len(changedFields)))

	for _, idx := range changedFields {
		field := schema.Field(idx)
		if field == nil {
			continue
		}

		// Field index
		e.writeByte(idx)

		// Check if it's an array/map change or simple field change
		if field.Type == TypeArray {
			if arrChanges := changes.GetArray(idx); arrChanges != nil && arrChanges.HasChanges() {
				// Incremental array changes
				e.writeByte(ArrayModeIncremental)
				e.encodeArrayChanges(field, arrChanges, t.GetFieldValue(idx))
				continue
			}
			// Full array replacement
			e.writeByte(ArrayModeFull)
			e.encodeArray(field, t.GetFieldValue(idx))
			continue
		}
		if field.Type == TypeMap {
			if mapChanges := changes.GetMap(idx); mapChanges != nil && mapChanges.HasChanges() {
				// Incremental map changes
				e.writeByte(ArrayModeIncremental)
				e.encodeMapChanges(field, mapChanges, t.GetFieldValue(idx))
				continue
			}
			// Full map replacement
			e.writeByte(ArrayModeFull)
			e.encodeMap(field, t.GetFieldValue(idx))
			continue
		}

		// Simple field replacement (primitives, structs)
		change := changes.GetFieldChange(idx)
		e.writeByte(uint8(change.Op))

		if change.Op != OpRemove {
			value := t.GetFieldValue(idx)
			e.encodeField(field, value)
		}
	}
}

// encodeField encodes a single field value
func (e *Encoder) encodeField(field *FieldMeta, value interface{}) {
	switch field.Type {
	case TypeInt8:
		e.writeInt8(toInt8(value))
	case TypeInt16:
		e.writeInt16(toInt16(value))
	case TypeInt32:
		e.writeInt32(toInt32(value))
	case TypeInt64:
		e.writeInt64(toInt64(value))
	case TypeUint8:
		e.writeByte(toUint8(value))
	case TypeUint16:
		e.writeUint16(toUint16(value))
	case TypeUint32:
		e.writeUint32(toUint32(value))
	case TypeUint64:
		e.writeUint64(toUint64(value))
	case TypeFloat32:
		e.writeFloat32(toFloat32(value))
	case TypeFloat64:
		e.writeFloat64(toFloat64(value))
	case TypeString:
		e.writeString(toString(value))
	case TypeBool:
		e.writeBool(toBool(value))
	case TypeBytes:
		e.writeBytes(toBytes(value))
	case TypeVarInt:
		e.writeVarInt(toInt64(value))
	case TypeVarUint:
		e.writeVarUint(toUint64(value))
	case TypeStruct:
		if t, ok := value.(Trackable); ok {
			e.encodeStruct(t, field.ChildSchema)
		}
	case TypeArray:
		e.encodeArray(field, value)
	case TypeMap:
		e.encodeMap(field, value)
	}
}

// encodeStruct encodes a nested struct
func (e *Encoder) encodeStruct(t Trackable, schema *Schema) {
	if t == nil {
		e.writeByte(0) // Null marker
		return
	}
	e.writeByte(1) // Non-null marker

	// Encode all fields of nested struct
	for i := range schema.Fields {
		field := &schema.Fields[i]
		value := t.GetFieldValue(uint8(i))
		e.encodeField(field, value)
	}
}

// encodeArray encodes an array field (full replacement)
func (e *Encoder) encodeArray(field *FieldMeta, value interface{}) {
	// Get array length using reflection or type assertion
	length := getArrayLength(value)
	e.writeVarUint(uint64(length))

	// Encode each element
	for i := 0; i < length; i++ {
		elem := getArrayElement(value, i)
		e.encodeArrayElement(field, elem)
	}
}

// encodeArrayChanges encodes incremental array changes
func (e *Encoder) encodeArrayChanges(field *FieldMeta, changes *ArrayChangeSet, value interface{}) {
	// Number of changes
	e.writeVarUint(uint64(len(changes.changes)))

	// Sort indices for deterministic output
	indices := make([]int, 0, len(changes.changes))
	for idx := range changes.changes {
		indices = append(indices, idx)
	}
	sortInts(indices)

	for _, idx := range indices {
		change := changes.changes[idx]

		// Index
		e.writeVarUint(uint64(idx))
		// Operation
		e.writeByte(uint8(change.Op))

		switch change.Op {
		case OpAdd, OpReplace:
			elem := getArrayElement(value, idx)
			e.encodeArrayElement(field, elem)
		case OpMove:
			e.writeVarUint(uint64(change.OldIndex))
		}
	}
}

// encodeArrayElement encodes a single array element
func (e *Encoder) encodeArrayElement(field *FieldMeta, elem interface{}) {
	if field.ElemType == TypeStruct {
		if t, ok := elem.(Trackable); ok {
			e.encodeStruct(t, field.ChildSchema)
		}
	} else {
		// Primitive element
		tempField := &FieldMeta{Type: field.ElemType}
		e.encodeField(tempField, elem)
	}
}

// encodeMap encodes a map field (full replacement)
func (e *Encoder) encodeMap(field *FieldMeta, value interface{}) {
	// Get map keys and values
	keys, values := getMapKeysValues(value)
	e.writeVarUint(uint64(len(keys)))

	for i, key := range keys {
		e.writeString(key)
		e.encodeMapValue(field, values[i])
	}
}

// encodeMapChanges encodes incremental map changes
func (e *Encoder) encodeMapChanges(field *FieldMeta, changes *MapChangeSet, value interface{}) {
	// Number of changes
	e.writeVarUint(uint64(len(changes.changes)))

	// Sort keys for deterministic output
	keys := make([]string, 0, len(changes.changes))
	for key := range changes.changes {
		keys = append(keys, key)
	}
	sortStrings(keys)

	for _, key := range keys {
		change := changes.changes[key]

		// Key
		e.writeString(key)
		// Operation
		e.writeByte(uint8(change.Op))

		if change.Op != OpRemove {
			e.encodeMapValue(field, change.Value)
		}
	}
}

// encodeMapValue encodes a single map value
func (e *Encoder) encodeMapValue(field *FieldMeta, value interface{}) {
	if field.ElemType == TypeStruct {
		if t, ok := value.(Trackable); ok {
			e.encodeStruct(t, field.ChildSchema)
		}
	} else {
		tempField := &FieldMeta{Type: field.ElemType}
		e.encodeField(tempField, value)
	}
}

// Primitive write methods (private - internal use)

func (e *Encoder) writeByte(v uint8) {
	e.grow(1)
	e.buf[e.pos] = v
	e.pos++
}

// ============================================================================
// Public write methods for generated FastEncoder implementations
// These are type-safe, zero-allocation methods that avoid interface{} boxing
// ============================================================================

// WriteFieldHeader writes the field index and operation
func (e *Encoder) WriteFieldHeader(fieldIndex uint8, op Operation) {
	e.writeByte(fieldIndex)
	e.writeByte(uint8(op))
}

// WriteInt8 writes an int8 value
func (e *Encoder) WriteInt8(v int8) { e.writeInt8(v) }

// WriteInt16 writes an int16 value
func (e *Encoder) WriteInt16(v int16) { e.writeInt16(v) }

// WriteInt32 writes an int32 value
func (e *Encoder) WriteInt32(v int32) { e.writeInt32(v) }

// WriteInt64 writes an int64 value
func (e *Encoder) WriteInt64(v int64) { e.writeInt64(v) }

// WriteUint8 writes a uint8 value
func (e *Encoder) WriteUint8(v uint8) { e.writeByte(v) }

// WriteUint16 writes a uint16 value
func (e *Encoder) WriteUint16(v uint16) { e.writeUint16(v) }

// WriteUint32 writes a uint32 value
func (e *Encoder) WriteUint32(v uint32) { e.writeUint32(v) }

// WriteUint64 writes a uint64 value
func (e *Encoder) WriteUint64(v uint64) { e.writeUint64(v) }

// WriteFloat32 writes a float32 value
func (e *Encoder) WriteFloat32(v float32) { e.writeFloat32(v) }

// WriteFloat64 writes a float64 value
func (e *Encoder) WriteFloat64(v float64) { e.writeFloat64(v) }

// WriteBool writes a bool value
func (e *Encoder) WriteBool(v bool) { e.writeBool(v) }

// WriteString writes a string value
func (e *Encoder) WriteString(v string) { e.writeString(v) }

// WriteBytes writes a []byte value
func (e *Encoder) WriteBytes(v []byte) { e.writeBytes(v) }

// WriteVarInt writes a variable-length signed integer
func (e *Encoder) WriteVarInt(v int64) { e.writeVarInt(v) }

// WriteVarUint writes a variable-length unsigned integer
func (e *Encoder) WriteVarUint(v uint64) { e.writeVarUint(v) }

// WriteChangeCount writes the number of changes (for patches)
func (e *Encoder) WriteChangeCount(count int) { e.writeVarUint(uint64(count)) }

func (e *Encoder) writeInt8(v int8) {
	e.writeByte(uint8(v))
}

func (e *Encoder) writeInt16(v int16) {
	e.grow(2)
	binary.LittleEndian.PutUint16(e.buf[e.pos:], uint16(v))
	e.pos += 2
}

func (e *Encoder) writeUint16(v uint16) {
	e.grow(2)
	binary.LittleEndian.PutUint16(e.buf[e.pos:], v)
	e.pos += 2
}

func (e *Encoder) writeInt32(v int32) {
	e.grow(4)
	binary.LittleEndian.PutUint32(e.buf[e.pos:], uint32(v))
	e.pos += 4
}

func (e *Encoder) writeUint32(v uint32) {
	e.grow(4)
	binary.LittleEndian.PutUint32(e.buf[e.pos:], v)
	e.pos += 4
}

func (e *Encoder) writeInt64(v int64) {
	e.grow(8)
	binary.LittleEndian.PutUint64(e.buf[e.pos:], uint64(v))
	e.pos += 8
}

func (e *Encoder) writeUint64(v uint64) {
	e.grow(8)
	binary.LittleEndian.PutUint64(e.buf[e.pos:], v)
	e.pos += 8
}

func (e *Encoder) writeFloat32(v float32) {
	e.grow(4)
	binary.LittleEndian.PutUint32(e.buf[e.pos:], math.Float32bits(v))
	e.pos += 4
}

func (e *Encoder) writeFloat64(v float64) {
	e.grow(8)
	binary.LittleEndian.PutUint64(e.buf[e.pos:], math.Float64bits(v))
	e.pos += 8
}

func (e *Encoder) writeBool(v bool) {
	if v {
		e.writeByte(1)
	} else {
		e.writeByte(0)
	}
}

func (e *Encoder) writeString(v string) {
	e.writeVarUint(uint64(len(v)))
	e.grow(len(v))
	copy(e.buf[e.pos:], v)
	e.pos += len(v)
}

func (e *Encoder) writeBytes(v []byte) {
	e.writeVarUint(uint64(len(v)))
	e.grow(len(v))
	copy(e.buf[e.pos:], v)
	e.pos += len(v)
}

// writeVarInt writes a variable-length signed integer (zigzag encoding)
func (e *Encoder) writeVarInt(v int64) {
	// Zigzag encoding: (v << 1) ^ (v >> 63)
	uv := uint64((v << 1) ^ (v >> 63))
	e.writeVarUint(uv)
}

// writeVarUint writes a variable-length unsigned integer
func (e *Encoder) writeVarUint(v uint64) {
	e.grow(10) // Max 10 bytes for uint64
	for v >= 0x80 {
		e.buf[e.pos] = byte(v) | 0x80
		e.pos++
		v >>= 7
	}
	e.buf[e.pos] = byte(v)
	e.pos++
}

// Type conversion helpers - simplified for common cases
// These are only used in the slow path; generated code uses direct types

func toInt8(v interface{}) int8 {
	if x, ok := v.(int8); ok {
		return x
	}
	if x, ok := v.(int); ok {
		return int8(x)
	}
	return 0
}

func toInt16(v interface{}) int16 {
	if x, ok := v.(int16); ok {
		return x
	}
	if x, ok := v.(int); ok {
		return int16(x)
	}
	return 0
}

func toInt32(v interface{}) int32 {
	if x, ok := v.(int32); ok {
		return x
	}
	if x, ok := v.(int); ok {
		return int32(x)
	}
	return 0
}

func toInt64(v interface{}) int64 {
	if x, ok := v.(int64); ok {
		return x
	}
	if x, ok := v.(int); ok {
		return int64(x)
	}
	return 0
}

func toUint8(v interface{}) uint8 {
	if x, ok := v.(uint8); ok {
		return x
	}
	if x, ok := v.(int); ok {
		return uint8(x)
	}
	return 0
}

func toUint16(v interface{}) uint16 {
	if x, ok := v.(uint16); ok {
		return x
	}
	if x, ok := v.(int); ok {
		return uint16(x)
	}
	return 0
}

func toUint32(v interface{}) uint32 {
	if x, ok := v.(uint32); ok {
		return x
	}
	if x, ok := v.(int); ok {
		return uint32(x)
	}
	return 0
}

func toUint64(v interface{}) uint64 {
	if x, ok := v.(uint64); ok {
		return x
	}
	if x, ok := v.(int); ok {
		return uint64(x)
	}
	return 0
}

func toFloat32(v interface{}) float32 {
	if x, ok := v.(float32); ok {
		return x
	}
	return 0
}

func toFloat64(v interface{}) float64 {
	if x, ok := v.(float64); ok {
		return x
	}
	return 0
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func toBytes(v interface{}) []byte {
	if b, ok := v.([]byte); ok {
		return b
	}
	return nil
}

// Array/map helpers using type switches (faster than reflection)

func getArrayLength(v interface{}) int {
	switch arr := v.(type) {
	case []interface{}:
		return len(arr)
	case []string:
		return len(arr)
	case []int:
		return len(arr)
	case []int8:
		return len(arr)
	case []int16:
		return len(arr)
	case []int32:
		return len(arr)
	case []int64:
		return len(arr)
	case []uint:
		return len(arr)
	case []uint8:
		return len(arr)
	case []uint16:
		return len(arr)
	case []uint32:
		return len(arr)
	case []uint64:
		return len(arr)
	case []float32:
		return len(arr)
	case []float64:
		return len(arr)
	case []bool:
		return len(arr)
	case []Trackable:
		return len(arr)
	}
	// Unknown type - return 0 (caller should use schema-generated types)
	return 0
}

func getArrayElement(v interface{}, i int) interface{} {
	switch arr := v.(type) {
	case []interface{}:
		return arr[i]
	case []string:
		return arr[i]
	case []int:
		return arr[i]
	case []int8:
		return arr[i]
	case []int16:
		return arr[i]
	case []int32:
		return arr[i]
	case []int64:
		return arr[i]
	case []uint:
		return arr[i]
	case []uint8:
		return arr[i]
	case []uint16:
		return arr[i]
	case []uint32:
		return arr[i]
	case []uint64:
		return arr[i]
	case []float32:
		return arr[i]
	case []float64:
		return arr[i]
	case []bool:
		return arr[i]
	case []Trackable:
		return arr[i]
	}
	return nil
}

func getMapKeysValues(v interface{}) ([]string, []interface{}) {
	switch m := v.(type) {
	case map[string]interface{}:
		return extractMapKV(m)
	case map[string]string:
		return extractMapKVTyped(m)
	case map[string]int:
		return extractMapKVTyped(m)
	case map[string]int32:
		return extractMapKVTyped(m)
	case map[string]int64:
		return extractMapKVTyped(m)
	case map[string]uint:
		return extractMapKVTyped(m)
	case map[string]uint32:
		return extractMapKVTyped(m)
	case map[string]uint64:
		return extractMapKVTyped(m)
	case map[string]float32:
		return extractMapKVTyped(m)
	case map[string]float64:
		return extractMapKVTyped(m)
	case map[string]bool:
		return extractMapKVTyped(m)
	}
	return nil, nil
}

// extractMapKV extracts keys and values from map[string]interface{}
func extractMapKV(m map[string]interface{}) ([]string, []interface{}) {
	keys := make([]string, 0, len(m))
	values := make([]interface{}, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	sortStringsWithValues(keys, values)
	return keys, values
}

// extractMapKVTyped is a generic helper for typed maps
func extractMapKVTyped[V any](m map[string]V) ([]string, []interface{}) {
	keys := make([]string, 0, len(m))
	values := make([]interface{}, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	sortStringsWithValues(keys, values)
	return keys, values
}

// Sort helpers using standard library (O(N log N))

func sortInts(a []int) {
	sort.Ints(a)
}

func sortStrings(a []string) {
	sort.Strings(a)
}

// keyValueSorter implements sort.Interface for sorting keys while maintaining
// the corresponding values in sync
type keyValueSorter struct {
	keys   []string
	values []interface{}
}

func (s *keyValueSorter) Len() int           { return len(s.keys) }
func (s *keyValueSorter) Less(i, j int) bool { return s.keys[i] < s.keys[j] }
func (s *keyValueSorter) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	s.values[i], s.values[j] = s.values[j], s.values[i]
}

func sortStringsWithValues(keys []string, values []interface{}) {
	sort.Sort(&keyValueSorter{keys: keys, values: values})
}
