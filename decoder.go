package statesync

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Decoder errors
var (
	ErrBufferTooSmall = errors.New("buffer too small")
	ErrInvalidMessage = errors.New("invalid message type")
	ErrUnknownSchema  = errors.New("unknown schema ID")
	ErrInvalidField   = errors.New("invalid field index")
	ErrInvalidType    = errors.New("invalid field type")
	ErrInvalidVarInt  = errors.New("invalid varint encoding")
	ErrInvalidOp      = errors.New("invalid operation")
)

// Decoder decodes binary messages
type Decoder struct {
	buf      []byte
	pos      int
	registry *SchemaRegistry
}

// NewDecoder creates a new decoder
func NewDecoder(registry *SchemaRegistry) *Decoder {
	return &Decoder{
		registry: registry,
	}
}

// DecodedPatch represents a decoded patch message
type DecodedPatch struct {
	SchemaID uint16
	Changes  []DecodedChange
}

// DecodedChange represents a single field change
type DecodedChange struct {
	FieldIndex uint8
	Op         Operation
	Value      interface{}

	// For array changes
	ArrayChanges []DecodedArrayChange

	// For map changes
	MapChanges []DecodedMapChange
}

// DecodedArrayChange represents a change to an array element
type DecodedArrayChange struct {
	Index    int
	Op       Operation
	OldIndex int         // For moves
	Value    interface{} // For add/replace
}

// DecodedMapChange represents a change to a map entry
type DecodedMapChange struct {
	Key   string
	Op    Operation
	Value interface{}
}

// Decode decodes a binary message
func (d *Decoder) Decode(data []byte) (*DecodedPatch, error) {
	d.buf = data
	d.pos = 0

	if len(data) < 3 {
		return nil, ErrBufferTooSmall
	}

	msgType, err := d.readByte()
	if err != nil {
		return nil, err
	}

	switch msgType {
	case MsgFullState:
		return d.decodeFullState()
	case MsgPatch:
		return d.decodePatch()
	default:
		return nil, ErrInvalidMessage
	}
}

// decodeFullState decodes a full state message
func (d *Decoder) decodeFullState() (*DecodedPatch, error) {
	schemaID, err := d.readUint16()
	if err != nil {
		return nil, err
	}

	schema := d.registry.Get(schemaID)
	if schema == nil {
		return nil, fmt.Errorf("%w: %d", ErrUnknownSchema, schemaID)
	}

	fieldCount, err := d.readByte()
	if err != nil {
		return nil, err
	}

	changes := make([]DecodedChange, fieldCount)
	for i := uint8(0); i < fieldCount; i++ {
		field := schema.Field(i)
		if field == nil {
			return nil, fmt.Errorf("%w: %d", ErrInvalidField, i)
		}

		value, err := d.decodeField(field)
		if err != nil {
			return nil, err
		}

		changes[i] = DecodedChange{
			FieldIndex: i,
			Op:         OpReplace,
			Value:      value,
		}
	}

	return &DecodedPatch{
		SchemaID: schemaID,
		Changes:  changes,
	}, nil
}

// decodePatch decodes an incremental patch message
func (d *Decoder) decodePatch() (*DecodedPatch, error) {
	schemaID, err := d.readUint16()
	if err != nil {
		return nil, err
	}

	schema := d.registry.Get(schemaID)
	if schema == nil {
		return nil, fmt.Errorf("%w: %d", ErrUnknownSchema, schemaID)
	}

	changeCount, err := d.readVarUint()
	if err != nil {
		return nil, err
	}

	changes := make([]DecodedChange, changeCount)
	for i := uint64(0); i < changeCount; i++ {
		fieldIndex, err := d.readByte()
		if err != nil {
			return nil, err
		}

		field := schema.Field(fieldIndex)
		if field == nil {
			return nil, fmt.Errorf("%w: %d", ErrInvalidField, fieldIndex)
		}

		change := DecodedChange{FieldIndex: fieldIndex}

		// Handle array/map fields with mode marker
		if field.Type == TypeArray {
			mode, err := d.readByte()
			if err != nil {
				return nil, err
			}
			if mode == ArrayModeIncremental {
				// Incremental changes
				arrayChanges, err := d.decodeArrayChanges(field)
				if err != nil {
					return nil, err
				}
				change.ArrayChanges = arrayChanges
			} else {
				// Full replacement
				change.Op = OpReplace
				value, err := d.decodeArrayFull(field)
				if err != nil {
					return nil, err
				}
				change.Value = value
			}
			changes[i] = change
			continue
		}
		if field.Type == TypeMap {
			mode, err := d.readByte()
			if err != nil {
				return nil, err
			}
			if mode == ArrayModeIncremental {
				// Incremental changes
				mapChanges, err := d.decodeMapChanges(field)
				if err != nil {
					return nil, err
				}
				change.MapChanges = mapChanges
			} else {
				// Full replacement
				change.Op = OpReplace
				value, err := d.decodeMapFull(field)
				if err != nil {
					return nil, err
				}
				change.Value = value
			}
			changes[i] = change
			continue
		}

		// Simple field change (primitives, structs)
		op, err := d.readByte()
		if err != nil {
			return nil, err
		}
		change.Op = Operation(op)

		if change.Op != OpRemove {
			value, err := d.decodeField(field)
			if err != nil {
				return nil, err
			}
			change.Value = value
		}

		changes[i] = change
	}

	return &DecodedPatch{
		SchemaID: schemaID,
		Changes:  changes,
	}, nil
}

// decodeField decodes a single field value
func (d *Decoder) decodeField(field *FieldMeta) (interface{}, error) {
	switch field.Type {
	case TypeInt8:
		return d.readInt8()
	case TypeInt16:
		return d.readInt16()
	case TypeInt32:
		return d.readInt32()
	case TypeInt64:
		return d.readInt64()
	case TypeUint8:
		return d.readByte()
	case TypeUint16:
		return d.readUint16()
	case TypeUint32:
		return d.readUint32()
	case TypeUint64:
		return d.readUint64()
	case TypeFloat32:
		return d.readFloat32()
	case TypeFloat64:
		return d.readFloat64()
	case TypeString:
		return d.readString()
	case TypeBool:
		return d.readBool()
	case TypeBytes:
		return d.readBytes()
	case TypeVarInt:
		return d.readVarInt()
	case TypeVarUint:
		return d.readVarUint()
	case TypeStruct:
		return d.decodeStruct(field.ChildSchema)
	case TypeArray:
		return d.decodeArray(field)
	case TypeMap:
		return d.decodeMap(field)
	default:
		return nil, fmt.Errorf("%w: %v", ErrInvalidType, field.Type)
	}
}

// decodeStruct decodes a nested struct
func (d *Decoder) decodeStruct(schema *Schema) (map[string]interface{}, error) {
	// Null check
	isNull, err := d.readByte()
	if err != nil {
		return nil, err
	}
	if isNull == 0 {
		return nil, nil
	}

	result := make(map[string]interface{})
	for i := range schema.Fields {
		field := &schema.Fields[i]
		value, err := d.decodeField(field)
		if err != nil {
			return nil, err
		}
		result[field.Name] = value
	}
	return result, nil
}

// decodeArray decodes an array in full state encoding
// Used in EncodeAll path where there's no mode marker
func (d *Decoder) decodeArray(field *FieldMeta) ([]interface{}, error) {
	return d.decodeArrayFull(field)
}

// decodeArrayFull decodes a full array replacement
func (d *Decoder) decodeArrayFull(field *FieldMeta) ([]interface{}, error) {
	length, err := d.readVarUint()
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, length)
	for i := uint64(0); i < length; i++ {
		value, err := d.decodeArrayElement(field)
		if err != nil {
			return nil, err
		}
		result[i] = value
	}
	return result, nil
}

// decodeArrayChanges decodes incremental array changes
func (d *Decoder) decodeArrayChanges(field *FieldMeta) ([]DecodedArrayChange, error) {
	changeCount, err := d.readVarUint()
	if err != nil {
		return nil, err
	}

	changes := make([]DecodedArrayChange, changeCount)
	for i := uint64(0); i < changeCount; i++ {
		index, err := d.readVarUint()
		if err != nil {
			return nil, err
		}

		op, err := d.readByte()
		if err != nil {
			return nil, err
		}

		change := DecodedArrayChange{
			Index: int(index),
			Op:    Operation(op),
		}

		switch change.Op {
		case OpAdd, OpReplace:
			value, err := d.decodeArrayElement(field)
			if err != nil {
				return nil, err
			}
			change.Value = value
		case OpMove:
			oldIdx, err := d.readVarUint()
			if err != nil {
				return nil, err
			}
			change.OldIndex = int(oldIdx)
		}

		changes[i] = change
	}
	return changes, nil
}

// decodeArrayElement decodes a single array element
func (d *Decoder) decodeArrayElement(field *FieldMeta) (interface{}, error) {
	if field.ElemType == TypeStruct {
		return d.decodeStruct(field.ChildSchema)
	}
	tempField := &FieldMeta{Type: field.ElemType}
	return d.decodeField(tempField)
}

// decodeMapFull decodes a full map replacement (no op byte prefix)
// Used for patches with ArrayModeFull marker
func (d *Decoder) decodeMapFull(field *FieldMeta) (map[string]interface{}, error) {
	length, err := d.readVarUint()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{}, length)
	for i := uint64(0); i < length; i++ {
		key, err := d.readString()
		if err != nil {
			return nil, err
		}
		value, err := d.decodeMapValue(field)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

// decodeMap decodes a map in full state encoding
// Used in EncodeAll path where there's no mode marker
func (d *Decoder) decodeMap(field *FieldMeta) (map[string]interface{}, error) {
	return d.decodeMapFull(field)
}

// decodeMapChanges decodes incremental map changes
func (d *Decoder) decodeMapChanges(field *FieldMeta) ([]DecodedMapChange, error) {
	changeCount, err := d.readVarUint()
	if err != nil {
		return nil, err
	}

	changes := make([]DecodedMapChange, changeCount)
	for i := uint64(0); i < changeCount; i++ {
		key, err := d.readString()
		if err != nil {
			return nil, err
		}

		op, err := d.readByte()
		if err != nil {
			return nil, err
		}

		change := DecodedMapChange{
			Key: key,
			Op:  Operation(op),
		}

		if change.Op != OpRemove {
			value, err := d.decodeMapValue(field)
			if err != nil {
				return nil, err
			}
			change.Value = value
		}

		changes[i] = change
	}
	return changes, nil
}

// decodeMapValue decodes a single map value
func (d *Decoder) decodeMapValue(field *FieldMeta) (interface{}, error) {
	if field.ElemType == TypeStruct {
		return d.decodeStruct(field.ChildSchema)
	}
	tempField := &FieldMeta{Type: field.ElemType}
	return d.decodeField(tempField)
}

// Primitive read methods

func (d *Decoder) readByte() (uint8, error) {
	if d.pos >= len(d.buf) {
		return 0, ErrBufferTooSmall
	}
	v := d.buf[d.pos]
	d.pos++
	return v, nil
}

func (d *Decoder) readInt8() (int8, error) {
	v, err := d.readByte()
	return int8(v), err
}

func (d *Decoder) readInt16() (int16, error) {
	if d.pos+2 > len(d.buf) {
		return 0, ErrBufferTooSmall
	}
	v := int16(binary.LittleEndian.Uint16(d.buf[d.pos:]))
	d.pos += 2
	return v, nil
}

func (d *Decoder) readUint16() (uint16, error) {
	if d.pos+2 > len(d.buf) {
		return 0, ErrBufferTooSmall
	}
	v := binary.LittleEndian.Uint16(d.buf[d.pos:])
	d.pos += 2
	return v, nil
}

func (d *Decoder) readInt32() (int32, error) {
	if d.pos+4 > len(d.buf) {
		return 0, ErrBufferTooSmall
	}
	v := int32(binary.LittleEndian.Uint32(d.buf[d.pos:]))
	d.pos += 4
	return v, nil
}

func (d *Decoder) readUint32() (uint32, error) {
	if d.pos+4 > len(d.buf) {
		return 0, ErrBufferTooSmall
	}
	v := binary.LittleEndian.Uint32(d.buf[d.pos:])
	d.pos += 4
	return v, nil
}

func (d *Decoder) readInt64() (int64, error) {
	if d.pos+8 > len(d.buf) {
		return 0, ErrBufferTooSmall
	}
	v := int64(binary.LittleEndian.Uint64(d.buf[d.pos:]))
	d.pos += 8
	return v, nil
}

func (d *Decoder) readUint64() (uint64, error) {
	if d.pos+8 > len(d.buf) {
		return 0, ErrBufferTooSmall
	}
	v := binary.LittleEndian.Uint64(d.buf[d.pos:])
	d.pos += 8
	return v, nil
}

func (d *Decoder) readFloat32() (float32, error) {
	if d.pos+4 > len(d.buf) {
		return 0, ErrBufferTooSmall
	}
	v := math.Float32frombits(binary.LittleEndian.Uint32(d.buf[d.pos:]))
	d.pos += 4
	return v, nil
}

func (d *Decoder) readFloat64() (float64, error) {
	if d.pos+8 > len(d.buf) {
		return 0, ErrBufferTooSmall
	}
	v := math.Float64frombits(binary.LittleEndian.Uint64(d.buf[d.pos:]))
	d.pos += 8
	return v, nil
}

func (d *Decoder) readBool() (bool, error) {
	v, err := d.readByte()
	return v != 0, err
}

func (d *Decoder) readString() (string, error) {
	length, err := d.readVarUint()
	if err != nil {
		return "", err
	}
	if d.pos+int(length) > len(d.buf) {
		return "", ErrBufferTooSmall
	}
	s := string(d.buf[d.pos : d.pos+int(length)])
	d.pos += int(length)
	return s, nil
}

func (d *Decoder) readBytes() ([]byte, error) {
	length, err := d.readVarUint()
	if err != nil {
		return nil, err
	}
	if d.pos+int(length) > len(d.buf) {
		return nil, ErrBufferTooSmall
	}
	b := make([]byte, length)
	copy(b, d.buf[d.pos:d.pos+int(length)])
	d.pos += int(length)
	return b, nil
}

// readVarInt reads a variable-length signed integer (zigzag encoding)
func (d *Decoder) readVarInt() (int64, error) {
	uv, err := d.readVarUint()
	if err != nil {
		return 0, err
	}
	// Zigzag decode: (uv >> 1) ^ -(uv & 1)
	return int64((uv >> 1) ^ -(uv & 1)), nil
}

// readVarUint reads a variable-length unsigned integer
func (d *Decoder) readVarUint() (uint64, error) {
	var result uint64
	var shift uint
	for {
		if d.pos >= len(d.buf) {
			return 0, ErrBufferTooSmall
		}
		b := d.buf[d.pos]
		d.pos++
		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 64 {
			return 0, ErrInvalidVarInt
		}
	}
	return result, nil
}

// ApplyPatch applies a decoded patch to a map-based state
func ApplyPatch(state map[string]interface{}, patch *DecodedPatch, schema *Schema) error {
	for _, change := range patch.Changes {
		field := schema.Field(change.FieldIndex)
		if field == nil {
			continue
		}

		switch change.Op {
		case OpReplace:
			state[field.Name] = change.Value
		case OpRemove:
			delete(state, field.Name)
		case OpAdd:
			state[field.Name] = change.Value
		}

		// Handle array changes
		if len(change.ArrayChanges) > 0 {
			arr, ok := state[field.Name].([]interface{})
			if !ok {
				arr = make([]interface{}, 0)
			}
			arr = applyArrayChanges(arr, change.ArrayChanges)
			state[field.Name] = arr
		}

		// Handle map changes
		if len(change.MapChanges) > 0 {
			m, ok := state[field.Name].(map[string]interface{})
			if !ok {
				m = make(map[string]interface{})
			}
			applyMapChanges(m, change.MapChanges)
			state[field.Name] = m
		}
	}
	return nil
}

func applyArrayChanges(arr []interface{}, changes []DecodedArrayChange) []interface{} {
	for _, change := range changes {
		switch change.Op {
		case OpAdd:
			if change.Index >= len(arr) {
				arr = append(arr, change.Value)
			} else {
				arr = append(arr[:change.Index+1], arr[change.Index:]...)
				arr[change.Index] = change.Value
			}
		case OpReplace:
			if change.Index < len(arr) {
				arr[change.Index] = change.Value
			}
		case OpRemove:
			if change.Index < len(arr) {
				arr = append(arr[:change.Index], arr[change.Index+1:]...)
			}
		case OpMove:
			if change.OldIndex < len(arr) && change.Index < len(arr) {
				elem := arr[change.OldIndex]
				arr = append(arr[:change.OldIndex], arr[change.OldIndex+1:]...)
				arr = append(arr[:change.Index], append([]interface{}{elem}, arr[change.Index:]...)...)
			}
		}
	}
	return arr
}

func applyMapChanges(m map[string]interface{}, changes []DecodedMapChange) {
	for _, change := range changes {
		switch change.Op {
		case OpAdd, OpReplace:
			m[change.Key] = change.Value
		case OpRemove:
			delete(m, change.Key)
		}
	}
}
