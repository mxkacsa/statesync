package statesync

import (
	"bytes"
	"reflect"
	"testing"
)

// TestDecoderFullState tests decoding of full state messages
func TestDecoderFullState(t *testing.T) {
	registry := NewSchemaRegistry()

	schema := NewSchemaBuilder("TestFull").
		WithID(100).
		Int32("intVal").
		String("strVal").
		Bool("boolVal").
		Build()
	registry.Register(schema)

	encoder := NewEncoder(registry)
	decoder := NewDecoder(registry)

	// Create a test trackable
	state := &simpleTrackable{
		schema:  schema,
		changes: NewChangeSet(),
		intVal:  42,
		strVal:  "hello",
		boolVal: true,
	}

	// Encode full state
	encoded := encoder.EncodeAll(state)
	if len(encoded) == 0 {
		t.Fatal("EncodeAll returned empty bytes")
	}

	// Decode full state
	patch, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if patch.SchemaID != 100 {
		t.Errorf("Expected schema ID 100, got %d", patch.SchemaID)
	}

	if len(patch.Changes) != 3 {
		t.Errorf("Expected 3 changes, got %d", len(patch.Changes))
	}

	// Verify values
	for _, change := range patch.Changes {
		switch change.FieldIndex {
		case 0:
			if v, ok := change.Value.(int32); !ok || v != 42 {
				t.Errorf("Expected int32(42), got %v", change.Value)
			}
		case 1:
			if v, ok := change.Value.(string); !ok || v != "hello" {
				t.Errorf("Expected 'hello', got %v", change.Value)
			}
		case 2:
			if v, ok := change.Value.(bool); !ok || v != true {
				t.Errorf("Expected true, got %v", change.Value)
			}
		}
	}
}

// TestDecoderPatch tests decoding of patch messages
func TestDecoderPatch(t *testing.T) {
	registry := NewSchemaRegistry()

	schema := NewSchemaBuilder("TestPatch").
		WithID(101).
		Int64("value").
		String("name").
		Build()
	registry.Register(schema)

	encoder := NewEncoder(registry)
	decoder := NewDecoder(registry)

	state := &simpleTrackable{
		schema:  schema,
		changes: NewChangeSet(),
		intVal:  100,
		strVal:  "test",
	}

	// Mark only one field as changed
	state.changes.Mark(0, OpReplace)
	state.intVal = 200

	encoded := encoder.Encode(state)
	if len(encoded) == 0 {
		t.Fatal("Encode returned empty bytes")
	}

	patch, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(patch.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(patch.Changes))
	}
}

// TestDecoderAllPrimitiveTypes tests encoding/decoding of all primitive types
func TestDecoderAllPrimitiveTypes(t *testing.T) {
	registry := NewSchemaRegistry()

	schema := NewSchemaBuilder("AllTypes").
		WithID(102).
		Int8("int8Val").
		Int16("int16Val").
		Int32("int32Val").
		Int64("int64Val").
		Uint8("uint8Val").
		Uint16("uint16Val").
		Uint32("uint32Val").
		Uint64("uint64Val").
		Float32("float32Val").
		Float64("float64Val").
		String("stringVal").
		Bool("boolVal").
		Bytes("bytesVal").
		Build()
	registry.Register(schema)

	encoder := NewEncoder(registry)
	decoder := NewDecoder(registry)

	state := &allTypesTrackable{
		schema:     schema,
		changes:    NewChangeSet(),
		int8Val:    -128,
		int16Val:   -32768,
		int32Val:   -2147483648,
		int64Val:   -9223372036854775808,
		uint8Val:   255,
		uint16Val:  65535,
		uint32Val:  4294967295,
		uint64Val:  18446744073709551615,
		float32Val: 3.14,
		float64Val: 2.718281828,
		stringVal:  "test string",
		boolVal:    true,
		bytesVal:   []byte{0x01, 0x02, 0x03},
	}

	encoded := encoder.EncodeAll(state)
	patch, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(patch.Changes) != 13 {
		t.Errorf("Expected 13 changes, got %d", len(patch.Changes))
	}

	// Verify each type
	for _, change := range patch.Changes {
		switch change.FieldIndex {
		case 0:
			if v, ok := change.Value.(int8); !ok || v != -128 {
				t.Errorf("int8: expected -128, got %v (%T)", change.Value, change.Value)
			}
		case 1:
			if v, ok := change.Value.(int16); !ok || v != -32768 {
				t.Errorf("int16: expected -32768, got %v", change.Value)
			}
		case 2:
			if v, ok := change.Value.(int32); !ok || v != -2147483648 {
				t.Errorf("int32: expected -2147483648, got %v", change.Value)
			}
		case 3:
			if v, ok := change.Value.(int64); !ok || v != -9223372036854775808 {
				t.Errorf("int64: expected min int64, got %v", change.Value)
			}
		case 4:
			if v, ok := change.Value.(uint8); !ok || v != 255 {
				t.Errorf("uint8: expected 255, got %v", change.Value)
			}
		case 5:
			if v, ok := change.Value.(uint16); !ok || v != 65535 {
				t.Errorf("uint16: expected 65535, got %v", change.Value)
			}
		case 6:
			if v, ok := change.Value.(uint32); !ok || v != 4294967295 {
				t.Errorf("uint32: expected max uint32, got %v", change.Value)
			}
		case 7:
			if v, ok := change.Value.(uint64); !ok || v != 18446744073709551615 {
				t.Errorf("uint64: expected max uint64, got %v", change.Value)
			}
		case 10:
			if v, ok := change.Value.(string); !ok || v != "test string" {
				t.Errorf("string: expected 'test string', got %v", change.Value)
			}
		case 11:
			if v, ok := change.Value.(bool); !ok || v != true {
				t.Errorf("bool: expected true, got %v", change.Value)
			}
		case 12:
			if v, ok := change.Value.([]byte); !ok || !bytes.Equal(v, []byte{0x01, 0x02, 0x03}) {
				t.Errorf("bytes: expected [1,2,3], got %v", change.Value)
			}
		}
	}
}

// TestDecoderArrayFull tests decoding of full array replacement
func TestDecoderArrayFull(t *testing.T) {
	registry := NewSchemaRegistry()

	schema := NewSchemaBuilder("ArrayTest").
		WithID(103).
		Array("items", TypeInt32, nil).
		Build()
	registry.Register(schema)

	encoder := NewEncoder(registry)
	decoder := NewDecoder(registry)

	state := &arrayTrackable{
		schema:  schema,
		changes: NewChangeSet(),
		items:   []int32{10, 20, 30, 40, 50},
	}

	encoded := encoder.EncodeAll(state)
	patch, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(patch.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(patch.Changes))
	}

	arr, ok := patch.Changes[0].Value.([]interface{})
	if !ok {
		t.Fatalf("Expected []interface{}, got %T", patch.Changes[0].Value)
	}

	if len(arr) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(arr))
	}
}

// TestDecoderMapFull tests decoding of full map replacement
func TestDecoderMapFull(t *testing.T) {
	registry := NewSchemaRegistry()

	schema := NewSchemaBuilder("MapTest").
		WithID(104).
		Map("data", TypeString, nil).
		Build()
	registry.Register(schema)

	encoder := NewEncoder(registry)
	decoder := NewDecoder(registry)

	state := &mapTrackable{
		schema:  schema,
		changes: NewChangeSet(),
		data:    map[string]string{"key1": "value1", "key2": "value2"},
	}

	encoded := encoder.EncodeAll(state)
	patch, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(patch.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(patch.Changes))
	}

	m, ok := patch.Changes[0].Value.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", patch.Changes[0].Value)
	}

	if len(m) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(m))
	}
}

// TestDecoderBufferErrors tests decoder behavior with invalid/truncated buffers
func TestDecoderBufferErrors(t *testing.T) {
	registry := NewSchemaRegistry()
	decoder := NewDecoder(registry)

	tests := []struct {
		name string
		data []byte
	}{
		{"empty buffer", []byte{}},
		{"too small", []byte{0x01}},
		{"invalid message type", []byte{0xFF, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decoder.Decode(tt.data)
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

// TestDecoderUnknownSchema tests decoder with unknown schema ID
func TestDecoderUnknownSchema(t *testing.T) {
	registry := NewSchemaRegistry()
	decoder := NewDecoder(registry)

	// Valid full state message but unknown schema ID
	data := []byte{MsgFullState, 0xFF, 0xFF, 0x00}
	_, err := decoder.Decode(data)
	if err == nil {
		t.Error("Expected error for unknown schema")
	}
}

// TestApplyPatch tests the ApplyPatch function
func TestApplyPatch(t *testing.T) {
	schema := NewSchemaBuilder("PatchTest").
		WithID(105).
		Int32("value").
		String("name").
		Array("items", TypeInt32, nil).
		Map("data", TypeString, nil).
		Build()

	state := map[string]interface{}{
		"value": int32(0),
		"name":  "",
		"items": []interface{}{},
		"data":  map[string]interface{}{},
	}

	// Test OpReplace
	patch := &DecodedPatch{
		SchemaID: 105,
		Changes: []DecodedChange{
			{FieldIndex: 0, Op: OpReplace, Value: int32(42)},
			{FieldIndex: 1, Op: OpReplace, Value: "hello"},
		},
	}

	err := ApplyPatch(state, patch, schema)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	if state["value"] != int32(42) {
		t.Errorf("Expected value=42, got %v", state["value"])
	}
	if state["name"] != "hello" {
		t.Errorf("Expected name='hello', got %v", state["name"])
	}
}

// TestApplyPatchArrayChanges tests ApplyPatch with array operations
func TestApplyPatchArrayChanges(t *testing.T) {
	schema := NewSchemaBuilder("ArrayPatch").
		WithID(106).
		Array("items", TypeInt32, nil).
		Build()

	state := map[string]interface{}{
		"items": []interface{}{int32(1), int32(2), int32(3)},
	}

	// Test array add
	patch := &DecodedPatch{
		SchemaID: 106,
		Changes: []DecodedChange{
			{
				FieldIndex: 0,
				ArrayChanges: []DecodedArrayChange{
					{Index: 1, Op: OpAdd, Value: int32(10)},
				},
			},
		},
	}

	err := ApplyPatch(state, patch, schema)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	items := state["items"].([]interface{})
	if len(items) != 4 {
		t.Errorf("Expected 4 items, got %d", len(items))
	}

	// Test array remove
	state["items"] = []interface{}{int32(1), int32(2), int32(3)}
	patch = &DecodedPatch{
		SchemaID: 106,
		Changes: []DecodedChange{
			{
				FieldIndex: 0,
				ArrayChanges: []DecodedArrayChange{
					{Index: 1, Op: OpRemove},
				},
			},
		},
	}

	err = ApplyPatch(state, patch, schema)
	if err != nil {
		t.Fatalf("ApplyPatch remove failed: %v", err)
	}

	items = state["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("Expected 2 items after remove, got %d", len(items))
	}

	// Test array replace
	state["items"] = []interface{}{int32(1), int32(2), int32(3)}
	patch = &DecodedPatch{
		SchemaID: 106,
		Changes: []DecodedChange{
			{
				FieldIndex: 0,
				ArrayChanges: []DecodedArrayChange{
					{Index: 1, Op: OpReplace, Value: int32(99)},
				},
			},
		},
	}

	err = ApplyPatch(state, patch, schema)
	if err != nil {
		t.Fatalf("ApplyPatch replace failed: %v", err)
	}

	items = state["items"].([]interface{})
	if items[1] != int32(99) {
		t.Errorf("Expected items[1]=99, got %v", items[1])
	}

	// Test array move
	state["items"] = []interface{}{int32(1), int32(2), int32(3)}
	patch = &DecodedPatch{
		SchemaID: 106,
		Changes: []DecodedChange{
			{
				FieldIndex: 0,
				ArrayChanges: []DecodedArrayChange{
					{Index: 0, Op: OpMove, OldIndex: 2},
				},
			},
		},
	}

	err = ApplyPatch(state, patch, schema)
	if err != nil {
		t.Fatalf("ApplyPatch move failed: %v", err)
	}
}

// TestApplyPatchMapChanges tests ApplyPatch with map operations
func TestApplyPatchMapChanges(t *testing.T) {
	schema := NewSchemaBuilder("MapPatch").
		WithID(107).
		Map("data", TypeInt32, nil).
		Build()

	state := map[string]interface{}{
		"data": map[string]interface{}{"a": int32(1), "b": int32(2)},
	}

	// Test map add/replace
	patch := &DecodedPatch{
		SchemaID: 107,
		Changes: []DecodedChange{
			{
				FieldIndex: 0,
				MapChanges: []DecodedMapChange{
					{Key: "c", Op: OpAdd, Value: int32(3)},
					{Key: "a", Op: OpReplace, Value: int32(10)},
				},
			},
		},
	}

	err := ApplyPatch(state, patch, schema)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	data := state["data"].(map[string]interface{})
	if data["c"] != int32(3) {
		t.Errorf("Expected data['c']=3, got %v", data["c"])
	}
	if data["a"] != int32(10) {
		t.Errorf("Expected data['a']=10, got %v", data["a"])
	}

	// Test map remove
	patch = &DecodedPatch{
		SchemaID: 107,
		Changes: []DecodedChange{
			{
				FieldIndex: 0,
				MapChanges: []DecodedMapChange{
					{Key: "b", Op: OpRemove},
				},
			},
		},
	}

	err = ApplyPatch(state, patch, schema)
	if err != nil {
		t.Fatalf("ApplyPatch remove failed: %v", err)
	}

	data = state["data"].(map[string]interface{})
	if _, exists := data["b"]; exists {
		t.Error("Expected 'b' to be removed")
	}
}

// TestApplyPatchOnNilCollection tests ApplyPatch when collection doesn't exist
func TestApplyPatchOnNilCollection(t *testing.T) {
	schema := NewSchemaBuilder("NilTest").
		WithID(108).
		Array("items", TypeInt32, nil).
		Map("data", TypeString, nil).
		Build()

	// State without the collections
	state := map[string]interface{}{}

	// Should create array
	patch := &DecodedPatch{
		SchemaID: 108,
		Changes: []DecodedChange{
			{
				FieldIndex: 0,
				ArrayChanges: []DecodedArrayChange{
					{Index: 0, Op: OpAdd, Value: int32(1)},
				},
			},
		},
	}

	err := ApplyPatch(state, patch, schema)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}

	// Should create map
	patch = &DecodedPatch{
		SchemaID: 108,
		Changes: []DecodedChange{
			{
				FieldIndex: 1,
				MapChanges: []DecodedMapChange{
					{Key: "test", Op: OpAdd, Value: "value"},
				},
			},
		},
	}

	err = ApplyPatch(state, patch, schema)
	if err != nil {
		t.Fatalf("ApplyPatch failed: %v", err)
	}
}

// TestVarIntEdgeCases tests varint encoding/decoding edge cases
func TestVarIntEdgeCases(t *testing.T) {
	registry := NewSchemaRegistry()
	encoder := NewEncoder(registry)

	tests := []uint64{
		0,
		1,
		127,               // Max 1-byte
		128,               // Min 2-byte
		16383,             // Max 2-byte
		16384,             // Min 3-byte
		2097151,           // Max 3-byte
		268435455,         // Max 4-byte
		34359738367,       // Max 5-byte
		4398046511103,     // Max 6-byte
		562949953421311,   // Max 7-byte
		72057594037927935, // Max 8-byte
		^uint64(0),        // Max uint64
	}

	for _, val := range tests {
		encoder.Reset()
		encoder.WriteVarUint(val)

		decoder := NewDecoder(registry)
		decoder.buf = encoder.Bytes()
		decoder.pos = 0

		decoded, err := decoder.readVarUint()
		if err != nil {
			t.Errorf("Failed to decode %d: %v", val, err)
			continue
		}

		if decoded != val {
			t.Errorf("Varint mismatch: expected %d, got %d", val, decoded)
		}
	}
}

// TestEncoderBufferGrowth tests encoder buffer growth strategies
func TestEncoderBufferGrowth(t *testing.T) {
	registry := NewSchemaRegistry()
	encoder := NewEncoder(registry)

	// Write a large amount of data to trigger buffer growth
	largeString := make([]byte, 100000)
	for i := range largeString {
		largeString[i] = 'x'
	}

	encoder.WriteString(string(largeString))

	if encoder.pos != len(largeString)+3 { // +3 for varint length
		t.Errorf("Unexpected position after large write: %d", encoder.pos)
	}
}

// TestChangeSetEdgeCases tests ChangeSet edge cases
func TestChangeSetEdgeCases(t *testing.T) {
	cs := NewChangeSet()

	// Test marking many fields (beyond first bitmap)
	for i := 0; i < 100; i++ {
		cs.Mark(uint8(i), OpReplace)
	}

	if !cs.HasChanges() {
		t.Error("Expected changes after marking 100 fields")
	}

	// Test GetFieldChange for unmarked field (returns default with OpNone)
	change := cs.GetFieldChange(200)
	if change.Op != 0 { // OpNone is 0
		t.Errorf("Expected default Op (0) for unmarked field, got %v", change.Op)
	}

	// Test Clear
	cs.Clear()
	if cs.HasChanges() {
		t.Error("Expected no changes after Clear")
	}

	// Test MarkAll
	cs.MarkAll(10)
	count := 0
	for i := 0; i < 10; i++ {
		if cs.IsFieldDirty(uint8(i)) {
			count++
		}
	}
	if count != 10 {
		t.Errorf("Expected 10 dirty fields, got %d", count)
	}
}

// TestArrayChangeSetEdgeCases tests ArrayChangeSet edge cases
func TestArrayChangeSetEdgeCases(t *testing.T) {
	cs := NewChangeSet()
	arr := cs.GetOrCreateArray(0)

	// Test all operations
	arr.MarkAdd(0, "value1")
	arr.MarkReplace(1, "value2")
	arr.MarkRemove(2)
	arr.MarkMove(3, 5)

	if !arr.HasChanges() {
		t.Error("Expected changes")
	}

	// Test clear
	arr.Clear()
	if arr.HasChanges() {
		t.Error("Expected no changes after Clear")
	}
}

// TestMapChangeSetEdgeCases tests MapChangeSet edge cases
func TestMapChangeSetEdgeCases(t *testing.T) {
	cs := NewChangeSet()
	m := cs.GetOrCreateMap(0)

	// Test all operations
	m.MarkAdd("a", "value1")
	m.MarkReplace("b", "value2")
	m.MarkRemove("c")

	if !m.HasChanges() {
		t.Error("Expected changes")
	}

	// Test clear
	m.Clear()
	if m.HasChanges() {
		t.Error("Expected no changes after Clear")
	}
}

// TestSchemaEdgeCases tests Schema edge cases
func TestSchemaEdgeCases(t *testing.T) {
	schema := NewSchemaBuilder("EdgeCase").
		WithID(200).
		Int32("field1").
		String("field2").
		Build()

	// Test Field with invalid index
	if schema.Field(100) != nil {
		t.Error("Expected nil for invalid field index")
	}

	// Test FieldByName with invalid name
	if schema.FieldByName("nonexistent") != nil {
		t.Error("Expected nil for invalid field name")
	}

	// Test FieldByName with valid name
	f := schema.FieldByName("field1")
	if f == nil || f.Name != "field1" {
		t.Error("Expected to find field1")
	}

	// Test MaxIndex
	if schema.MaxIndex() != 1 {
		t.Errorf("Expected MaxIndex=1, got %d", schema.MaxIndex())
	}
}

// TestSchemaRegistryEdgeCases tests SchemaRegistry edge cases
func TestSchemaRegistryEdgeCases(t *testing.T) {
	registry := NewSchemaRegistry()

	schema := NewSchemaBuilder("Test").WithID(1).Int32("val").Build()
	registry.Register(schema)

	// Test GetByName
	found := registry.GetByName("Test")
	if found == nil {
		t.Error("Expected to find schema by name")
	}

	// Test Get with invalid ID
	if registry.Get(999) != nil {
		t.Error("Expected nil for invalid ID")
	}

	// Test GetByName with invalid name
	if registry.GetByName("NonExistent") != nil {
		t.Error("Expected nil for invalid name")
	}
}

// Helper trackable types for testing

type simpleTrackable struct {
	schema  *Schema
	changes *ChangeSet
	intVal  int64
	strVal  string
	boolVal bool
}

func (s *simpleTrackable) Schema() *Schema     { return s.schema }
func (s *simpleTrackable) Changes() *ChangeSet { return s.changes }
func (s *simpleTrackable) ClearChanges()       { s.changes.Clear() }
func (s *simpleTrackable) MarkAllDirty()       { s.changes.MarkAll(3) }
func (s *simpleTrackable) GetFieldValue(idx uint8) interface{} {
	switch idx {
	case 0:
		return int32(s.intVal)
	case 1:
		return s.strVal
	case 2:
		return s.boolVal
	}
	return nil
}

type allTypesTrackable struct {
	schema     *Schema
	changes    *ChangeSet
	int8Val    int8
	int16Val   int16
	int32Val   int32
	int64Val   int64
	uint8Val   uint8
	uint16Val  uint16
	uint32Val  uint32
	uint64Val  uint64
	float32Val float32
	float64Val float64
	stringVal  string
	boolVal    bool
	bytesVal   []byte
}

func (a *allTypesTrackable) Schema() *Schema     { return a.schema }
func (a *allTypesTrackable) Changes() *ChangeSet { return a.changes }
func (a *allTypesTrackable) ClearChanges()       { a.changes.Clear() }
func (a *allTypesTrackable) MarkAllDirty()       { a.changes.MarkAll(13) }
func (a *allTypesTrackable) GetFieldValue(idx uint8) interface{} {
	switch idx {
	case 0:
		return a.int8Val
	case 1:
		return a.int16Val
	case 2:
		return a.int32Val
	case 3:
		return a.int64Val
	case 4:
		return a.uint8Val
	case 5:
		return a.uint16Val
	case 6:
		return a.uint32Val
	case 7:
		return a.uint64Val
	case 8:
		return a.float32Val
	case 9:
		return a.float64Val
	case 10:
		return a.stringVal
	case 11:
		return a.boolVal
	case 12:
		return a.bytesVal
	}
	return nil
}

type arrayTrackable struct {
	schema  *Schema
	changes *ChangeSet
	items   []int32
}

func (a *arrayTrackable) Schema() *Schema     { return a.schema }
func (a *arrayTrackable) Changes() *ChangeSet { return a.changes }
func (a *arrayTrackable) ClearChanges()       { a.changes.Clear() }
func (a *arrayTrackable) MarkAllDirty()       { a.changes.MarkAll(1) }
func (a *arrayTrackable) GetFieldValue(idx uint8) interface{} {
	if idx == 0 {
		return a.items
	}
	return nil
}

type mapTrackable struct {
	schema  *Schema
	changes *ChangeSet
	data    map[string]string
}

func (m *mapTrackable) Schema() *Schema     { return m.schema }
func (m *mapTrackable) Changes() *ChangeSet { return m.changes }
func (m *mapTrackable) ClearChanges()       { m.changes.Clear() }
func (m *mapTrackable) MarkAllDirty()       { m.changes.MarkAll(1) }
func (m *mapTrackable) GetFieldValue(idx uint8) interface{} {
	if idx == 0 {
		return m.data
	}
	return nil
}

// TestEncoderSlowPathTypeConversions tests the type conversion functions in encoder
func TestEncoderSlowPathTypeConversions(t *testing.T) {
	registry := NewSchemaRegistry()

	// Test int conversions (schema expects int32 but we pass int)
	schema := NewSchemaBuilder("IntConvert").
		WithID(200).
		Int32("val").
		Build()
	registry.Register(schema)

	encoder := NewEncoder(registry)

	state := &intConvertTrackable{
		schema:  schema,
		changes: NewChangeSet(),
		val:     int(42), // Pass int, should convert to int32
	}

	encoded := encoder.EncodeAll(state)
	if len(encoded) == 0 {
		t.Error("Expected non-empty encoding")
	}

	decoder := NewDecoder(registry)
	patch, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if v, ok := patch.Changes[0].Value.(int32); !ok || v != 42 {
		t.Errorf("Expected int32(42), got %v", patch.Changes[0].Value)
	}
}

type intConvertTrackable struct {
	schema  *Schema
	changes *ChangeSet
	val     int
}

func (i *intConvertTrackable) Schema() *Schema     { return i.schema }
func (i *intConvertTrackable) Changes() *ChangeSet { return i.changes }
func (i *intConvertTrackable) ClearChanges()       { i.changes.Clear() }
func (i *intConvertTrackable) MarkAllDirty()       { i.changes.MarkAll(1) }
func (i *intConvertTrackable) GetFieldValue(idx uint8) interface{} {
	if idx == 0 {
		return i.val
	}
	return nil
}

// TestTrackedSessionDebounce tests TrackedSession debounce functionality
func TestTrackedSessionDebounce(t *testing.T) {
	state := NewTestGameState()
	ts := NewTrackedState[*TestGameState, any](state, nil)
	session := NewTrackedSession[*TestGameState, any, string](ts)

	// Test SetDebounce
	session.SetDebounce(0) // No debounce

	// Test SetBroadcastCallback
	called := false
	session.SetBroadcastCallback(func(diffs map[string][]byte) {
		called = true
	})

	session.Connect("client1", nil)
	ts.Update(func(s **TestGameState) {
		(*s).round = 5
		(*s).changes.Mark(0, OpReplace)
	})

	// ScheduleBroadcast with no debounce should call immediately
	session.ScheduleBroadcast()

	if !called {
		t.Error("Expected broadcast callback to be called")
	}
}

// TestTrackedSessionMethods tests TrackedSession proxy methods
func TestTrackedSessionMethods(t *testing.T) {
	state := NewTestGameState()
	ts := NewTrackedState[*TestGameState, any](state, nil)
	session := NewTrackedSession[*TestGameState, any, string](ts)

	// Test Clients()
	session.Connect("client1", nil)
	session.Connect("client2", nil)

	clients := session.Clients()
	if len(clients) != 2 {
		t.Errorf("Expected 2 clients, got %d", len(clients))
	}

	// Test State()
	if session.State() != ts {
		t.Error("State() should return underlying TrackedState")
	}

	// Test Full()
	full := session.Full("client1")
	if len(full) == 0 {
		t.Error("Full() should return encoded state")
	}

	// Test Diff() for new client (should return full state)
	diff := session.Diff("client1")
	if len(diff) == 0 {
		t.Error("Diff() should return data for new client")
	}

	// Test Get() and GetBase()
	_ = session.Get()
	_ = session.GetBase()

	// Note: UpdateAndBroadcast is tested indirectly through ApplyUpdate
}

// TestTrackedStateEffects tests effect management in TrackedState
func TestTrackedStateEffects(t *testing.T) {
	state := NewTestGameState()
	ts := NewTrackedState[*TestGameState, any](state, nil)

	// Test AddEffect using Func effect
	effect := Func[*TestGameState, any]("test-effect", func(s *TestGameState, a any) *TestGameState {
		s.round += 10
		return s
	})

	err := ts.AddEffect(effect, nil)
	if err != nil {
		t.Errorf("AddEffect failed: %v", err)
	}

	// Test HasEffect
	if !ts.HasEffect("test-effect") {
		t.Error("Expected effect to exist")
	}

	// Test GetEffect
	got := ts.GetEffect("test-effect")
	if got == nil {
		t.Error("GetEffect should return the effect")
	}

	// Test Effects()
	effects := ts.Effects()
	if len(effects) != 1 {
		t.Errorf("Expected 1 effect, got %d", len(effects))
	}

	// Test RemoveEffect
	if !ts.RemoveEffect("test-effect") {
		t.Error("RemoveEffect should return true")
	}

	if ts.HasEffect("test-effect") {
		t.Error("Effect should be removed")
	}

	// Test ClearEffects
	ts.AddEffect(effect, nil)
	ts.ClearEffects()
	if ts.HasEffect("test-effect") {
		t.Error("ClearEffects should remove all effects")
	}
}

// TestTrackedSessionEffectProxies tests TrackedSession effect proxy methods
func TestTrackedSessionEffectProxies(t *testing.T) {
	state := NewTestGameState()
	ts := NewTrackedState[*TestGameState, any](state, nil)
	session := NewTrackedSession[*TestGameState, any, string](ts)

	effect := Func[*TestGameState, any]("proxy-test", func(s *TestGameState, a any) *TestGameState {
		return s
	})

	// Test AddEffect via session
	err := session.AddEffect(effect, nil)
	if err != nil {
		t.Errorf("AddEffect via session failed: %v", err)
	}

	// Test HasEffect via session
	if !session.HasEffect("proxy-test") {
		t.Error("HasEffect via session should work")
	}

	// Test GetEffect via session
	if session.GetEffect("proxy-test") == nil {
		t.Error("GetEffect via session should work")
	}

	// Test RemoveEffect via session
	if !session.RemoveEffect("proxy-test") {
		t.Error("RemoveEffect via session should work")
	}

	// Test ClearEffects via session
	session.AddEffect(effect, nil)
	session.ClearEffects()
	if session.HasEffect("proxy-test") {
		t.Error("ClearEffects via session should work")
	}
}

// TestTrackedStateGetMethods tests Get and GetBase
func TestTrackedStateGetMethods(t *testing.T) {
	state := NewTestGameState()
	ts := NewTrackedState[*TestGameState, any](state, nil)

	// Initial state
	got := ts.Get()
	if got.round != 0 {
		t.Errorf("Expected round=0, got %d", got.round)
	}

	// Update and check
	ts.Update(func(s **TestGameState) {
		(*s).round = 5
		(*s).changes.Mark(0, OpReplace)
	})

	got = ts.GetBase()
	if got.round != 5 {
		t.Errorf("Expected round=5, got %d", got.round)
	}
}

// TestTrackedStateSet tests Set method
func TestTrackedStateSet(t *testing.T) {
	state := NewTestGameState()
	ts := NewTrackedState[*TestGameState, any](state, nil)

	newState := NewTestGameState()
	newState.round = 100
	newState.phase = "new"

	ts.Set(newState)

	got := ts.GetBase()
	if got.round != 100 || got.phase != "new" {
		t.Errorf("Set didn't work properly: round=%d, phase=%s", got.round, got.phase)
	}
}

// TestTrackedStateRegistry tests Registry method
func TestTrackedStateRegistry(t *testing.T) {
	state := NewTestGameState()
	registry := NewSchemaRegistry()
	cfg := &TrackedConfig{Registry: registry}
	ts := NewTrackedState[*TestGameState, any](state, cfg)

	if ts.Registry() != registry {
		t.Error("Registry should return the provided registry")
	}
}

// TestTrackedSessionApplyUpdate tests ApplyUpdate
func TestTrackedSessionApplyUpdate(t *testing.T) {
	state := NewTestGameState()
	ts := NewTrackedState[*TestGameState, any](state, nil)
	session := NewTrackedSession[*TestGameState, any, string](ts)

	session.Connect("client1", nil)

	diffs := session.ApplyUpdate(func(s **TestGameState) {
		(*s).round = 42
		(*s).changes.Mark(0, OpReplace)
	})

	if len(diffs) == 0 {
		t.Error("ApplyUpdate should return diffs")
	}
}

// TestEncoderSlowPathArrayMap tests encoder slow path for arrays and maps
func TestEncoderSlowPathArrayMap(t *testing.T) {
	registry := NewSchemaRegistry()

	// Test with various array types
	schema := NewSchemaBuilder("ArrayTypes").
		WithID(201).
		Array("ints", TypeInt32, nil).
		Array("strings", TypeString, nil).
		Array("floats", TypeFloat64, nil).
		Map("strMap", TypeString, nil).
		Map("intMap", TypeInt64, nil).
		Build()
	registry.Register(schema)

	encoder := NewEncoder(registry)

	state := &arrayMapTrackable{
		schema:  schema,
		changes: NewChangeSet(),
		ints:    []int{1, 2, 3},
		strings: []string{"a", "b", "c"},
		floats:  []float64{1.1, 2.2, 3.3},
		strMap:  map[string]string{"k1": "v1", "k2": "v2"},
		intMap:  map[string]int64{"a": 100, "b": 200},
	}

	encoded := encoder.EncodeAll(state)
	if len(encoded) == 0 {
		t.Error("Expected non-empty encoding")
	}

	decoder := NewDecoder(registry)
	_, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
}

type arrayMapTrackable struct {
	schema  *Schema
	changes *ChangeSet
	ints    []int
	strings []string
	floats  []float64
	strMap  map[string]string
	intMap  map[string]int64
}

func (a *arrayMapTrackable) Schema() *Schema     { return a.schema }
func (a *arrayMapTrackable) Changes() *ChangeSet { return a.changes }
func (a *arrayMapTrackable) ClearChanges()       { a.changes.Clear() }
func (a *arrayMapTrackable) MarkAllDirty()       { a.changes.MarkAll(5) }
func (a *arrayMapTrackable) GetFieldValue(idx uint8) interface{} {
	switch idx {
	case 0:
		return a.ints
	case 1:
		return a.strings
	case 2:
		return a.floats
	case 3:
		return a.strMap
	case 4:
		return a.intMap
	}
	return nil
}

// TestOperationString tests Operation.String() method
func TestOperationString(t *testing.T) {
	tests := []struct {
		op   Operation
		want string
	}{
		{OpReplace, "replace"},
		{OpAdd, "add"},
		{OpRemove, "remove"},
		{OpMove, "move"},
		{Operation(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.op.String()
		if got != tt.want {
			t.Errorf("Operation(%d).String() = %s, want %s", tt.op, got, tt.want)
		}
	}
}

// TestFieldTypeString tests FieldType.String() method
func TestFieldTypeString(t *testing.T) {
	tests := []struct {
		ft   FieldType
		want string
	}{
		{TypeInt8, "int8"},
		{TypeInt16, "int16"},
		{TypeInt32, "int32"},
		{TypeInt64, "int64"},
		{TypeUint8, "uint8"},
		{TypeUint16, "uint16"},
		{TypeUint32, "uint32"},
		{TypeUint64, "uint64"},
		{TypeFloat32, "float32"},
		{TypeFloat64, "float64"},
		{TypeString, "string"},
		{TypeBool, "bool"},
		{TypeBytes, "bytes"},
		{TypeStruct, "struct"},
		{TypeArray, "array"},
		{TypeMap, "map"},
		{TypeVarInt, "varint"},
		{TypeVarUint, "varuint"},
		{TypeTimestamp, "timestamp"},
		{FieldType(99), "unknown(99)"},
	}

	for _, tt := range tests {
		got := tt.ft.String()
		if got != tt.want {
			t.Errorf("FieldType(%d).String() = %s, want %s", tt.ft, got, tt.want)
		}
	}
}

// TestFieldTypeSize tests FieldType.Size() method
func TestFieldTypeSize(t *testing.T) {
	tests := []struct {
		ft   FieldType
		want int
	}{
		{TypeInt8, 1},
		{TypeUint8, 1},
		{TypeBool, 1},
		{TypeInt16, 2},
		{TypeUint16, 2},
		{TypeInt32, 4},
		{TypeUint32, 4},
		{TypeFloat32, 4},
		{TypeInt64, 8},
		{TypeUint64, 8},
		{TypeFloat64, 8},
		{TypeTimestamp, 8},
		{TypeString, 0}, // Variable length returns 0
		{TypeBytes, 0},
		{TypeStruct, 0},
		{TypeArray, 0},
		{TypeMap, 0},
		{TypeVarInt, 0},
		{TypeVarUint, 0},
	}

	for _, tt := range tests {
		got := tt.ft.Size()
		if got != tt.want {
			t.Errorf("FieldType(%d).Size() = %d, want %d", tt.ft, got, tt.want)
		}
	}
}

// TestTrackedFuncEffectBuiltin tests the built-in TrackedFuncEffect
func TestTrackedFuncEffectBuiltin(t *testing.T) {
	effect := Func[*TestGameState, string]("test-func", func(s *TestGameState, a string) *TestGameState {
		s.round += 100
		return s
	})

	if effect.ID() != "test-func" {
		t.Errorf("Expected ID 'test-func', got %s", effect.ID())
	}

	state := NewTestGameState()
	result := effect.Apply(state, "activator")

	if result.round != 100 {
		t.Errorf("Expected round=100 after Apply, got %d", result.round)
	}

	// Test Activator
	effect.SetActivator("test-activator")
	if effect.Activator() != "test-activator" {
		t.Error("SetActivator/Activator not working")
	}
}

// TestInferFieldType tests the InferFieldType function
func TestInferFieldTypeFunc(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  FieldType
	}{
		{"int8", int8(1), TypeInt8},
		{"int16", int16(1), TypeInt16},
		{"int32", int32(1), TypeInt32},
		{"int64", int64(1), TypeInt64},
		{"int", int(1), TypeInt64},
		{"uint8", uint8(1), TypeUint8},
		{"uint16", uint16(1), TypeUint16},
		{"uint32", uint32(1), TypeUint32},
		{"uint64", uint64(1), TypeUint64},
		{"float32", float32(1.0), TypeFloat32},
		{"float64", float64(1.0), TypeFloat64},
		{"string", "test", TypeString},
		{"bool", true, TypeBool},
		{"bytes", []byte{1, 2, 3}, TypeBytes},
		{"struct", struct{}{}, TypeStruct},
		{"slice", []int{1, 2}, TypeArray},
		{"map", map[string]int{}, TypeMap},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferFieldType(reflect.TypeOf(tt.value))
			if got != tt.want {
				t.Errorf("InferFieldType(%T) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
