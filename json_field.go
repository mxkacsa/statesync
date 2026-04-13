package statesync

import "encoding/json"

// MarshalField encodes a Go value to JSON bytes for storing in a bytes field.
// Returns nil for nil pointers. Use with struct field setters:
//
//	gs.SetConfig(statesync.MarshalField(cfg))
func MarshalField[T any](v *T) []byte {
	if v == nil {
		return nil
	}
	data, _ := json.Marshal(v)
	return data
}

// UnmarshalField decodes JSON bytes from a bytes field into a Go value.
// Returns nil for empty/nil data. Use with struct field getters:
//
//	cfg := statesync.UnmarshalField[domain.GameConfig](gs.Config())
func UnmarshalField[T any](data []byte) *T {
	if len(data) == 0 {
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil
	}
	return &v
}

// MarshalFieldValue encodes a non-pointer Go value to JSON bytes.
// Use for value types (not pointers):
//
//	gs.SetPosition(statesync.MarshalFieldValue(pos))
func MarshalFieldValue[T any](v T) []byte {
	data, _ := json.Marshal(v)
	return data
}

// UnmarshalFieldValue decodes JSON bytes into a Go value (non-pointer).
// Returns the zero value on error. Use when nil is not meaningful:
//
//	pos := statesync.UnmarshalFieldValue[domain.Position](gs.Position())
func UnmarshalFieldValue[T any](data []byte) T {
	var v T
	if len(data) == 0 {
		return v
	}
	json.Unmarshal(data, &v)
	return v
}
