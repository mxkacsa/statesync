package statesync

import "testing"

func TestMarshalField(t *testing.T) {
	type Config struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	cfg := &Config{Name: "test", Value: 42}
	data := MarshalField(cfg)
	if data == nil {
		t.Fatal("MarshalField returned nil")
	}

	decoded := UnmarshalField[Config](data)
	if decoded == nil {
		t.Fatal("UnmarshalField returned nil")
	}
	if decoded.Name != "test" || decoded.Value != 42 {
		t.Errorf("got %+v, want {Name:test Value:42}", decoded)
	}
}

func TestMarshalFieldNil(t *testing.T) {
	type Foo struct{ X int }
	data := MarshalField[Foo](nil)
	if data != nil {
		t.Error("MarshalField(nil) should return nil")
	}

	decoded := UnmarshalField[Foo](nil)
	if decoded != nil {
		t.Error("UnmarshalField(nil) should return nil")
	}

	decoded2 := UnmarshalField[Foo]([]byte{})
	if decoded2 != nil {
		t.Error("UnmarshalField(empty) should return nil")
	}
}

func TestMarshalFieldValue(t *testing.T) {
	type Pos struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}

	pos := Pos{X: 1.5, Y: 2.5}
	data := MarshalFieldValue(pos)
	if data == nil {
		t.Fatal("MarshalFieldValue returned nil")
	}

	decoded := UnmarshalFieldValue[Pos](data)
	if decoded.X != 1.5 || decoded.Y != 2.5 {
		t.Errorf("got %+v, want {X:1.5 Y:2.5}", decoded)
	}
}

func TestUnmarshalFieldValueEmpty(t *testing.T) {
	type Foo struct{ X int }
	v := UnmarshalFieldValue[Foo](nil)
	if v.X != 0 {
		t.Error("UnmarshalFieldValue(nil) should return zero value")
	}
}

func TestUnmarshalFieldInvalidJSON(t *testing.T) {
	type Foo struct{ X int }
	decoded := UnmarshalField[Foo]([]byte("not json"))
	if decoded != nil {
		t.Error("UnmarshalField with invalid JSON should return nil")
	}
}
