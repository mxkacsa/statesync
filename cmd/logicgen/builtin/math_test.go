package builtin

import (
	"math"
	"testing"
)

func TestMathAdd(t *testing.T) {
	tests := []struct {
		a, b, want float64
	}{
		{1, 2, 3},
		{-1, 1, 0},
		{0, 0, 0},
		{1.5, 2.5, 4.0},
		{-5.5, -4.5, -10.0},
	}

	for _, tt := range tests {
		result, err := Call("Math.Add", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Add(%f, %f) = %v, want %f", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestMathAdd_InvalidInput(t *testing.T) {
	_, err := Call("Math.Add", map[string]interface{}{
		"a": "not a number",
		"b": 1.0,
	})
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestMathSubtract(t *testing.T) {
	tests := []struct {
		a, b, want float64
	}{
		{5, 3, 2},
		{0, 5, -5},
		{-3, -7, 4},
		{10.5, 2.5, 8.0},
	}

	for _, tt := range tests {
		result, err := Call("Math.Subtract", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Subtract(%f, %f) = %v, want %f", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestMathMultiply(t *testing.T) {
	tests := []struct {
		a, b, want float64
	}{
		{2, 3, 6},
		{-2, 3, -6},
		{-2, -3, 6},
		{0, 100, 0},
		{1.5, 2, 3.0},
	}

	for _, tt := range tests {
		result, err := Call("Math.Multiply", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Multiply(%f, %f) = %v, want %f", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestMathDivide(t *testing.T) {
	tests := []struct {
		a, b, want float64
	}{
		{6, 2, 3},
		{10, 4, 2.5},
		{-10, 2, -5},
		{7, 2, 3.5},
	}

	for _, tt := range tests {
		result, err := Call("Math.Divide", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Divide(%f, %f) = %v, want %f", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestMathDivide_ByZero(t *testing.T) {
	_, err := Call("Math.Divide", map[string]interface{}{
		"a": 10.0,
		"b": 0.0,
	})
	if err == nil {
		t.Error("expected division by zero error")
	}
}

func TestMathModulo(t *testing.T) {
	tests := []struct {
		a, b, want float64
	}{
		{10, 3, 1},
		{10, 2, 0},
		{7.5, 2.5, 0},
		{-10, 3, -1},
	}

	for _, tt := range tests {
		result, err := Call("Math.Modulo", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Modulo(%f, %f) = %v, want %f", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestMathModulo_ByZero(t *testing.T) {
	_, err := Call("Math.Modulo", map[string]interface{}{
		"a": 10.0,
		"b": 0.0,
	})
	if err == nil {
		t.Error("expected modulo by zero error")
	}
}

func TestMathClamp(t *testing.T) {
	tests := []struct {
		value, min, max, want float64
	}{
		{5, 0, 10, 5},   // Within range
		{-5, 0, 10, 0},  // Below min
		{15, 0, 10, 10}, // Above max
		{0, 0, 10, 0},   // At min
		{10, 0, 10, 10}, // At max
	}

	for _, tt := range tests {
		result, err := Call("Math.Clamp", map[string]interface{}{
			"value": tt.value,
			"min":   tt.min,
			"max":   tt.max,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Clamp(%f, %f, %f) = %v, want %f", tt.value, tt.min, tt.max, result, tt.want)
		}
	}
}

func TestMathMin(t *testing.T) {
	tests := []struct {
		a, b, want float64
	}{
		{1, 2, 1},
		{5, 3, 3},
		{-1, -2, -2},
		{0, 0, 0},
	}

	for _, tt := range tests {
		result, err := Call("Math.Min", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Min(%f, %f) = %v, want %f", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestMathMax(t *testing.T) {
	tests := []struct {
		a, b, want float64
	}{
		{1, 2, 2},
		{5, 3, 5},
		{-1, -2, -1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		result, err := Call("Math.Max", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Max(%f, %f) = %v, want %f", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestMathAbs(t *testing.T) {
	tests := []struct {
		value, want float64
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-3.14, 3.14},
	}

	for _, tt := range tests {
		result, err := Call("Math.Abs", map[string]interface{}{
			"value": tt.value,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Abs(%f) = %v, want %f", tt.value, result, tt.want)
		}
	}
}

func TestMathRound(t *testing.T) {
	tests := []struct {
		value, want float64
	}{
		{1.4, 1},
		{1.5, 2},
		{1.6, 2},
		{-1.4, -1},
		{-1.5, -2},
		{2.0, 2},
	}

	for _, tt := range tests {
		result, err := Call("Math.Round", map[string]interface{}{
			"value": tt.value,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Round(%f) = %v, want %f", tt.value, result, tt.want)
		}
	}
}

func TestMathFloor(t *testing.T) {
	tests := []struct {
		value, want float64
	}{
		{1.9, 1},
		{1.1, 1},
		{1.0, 1},
		{-1.1, -2},
		{-1.9, -2},
	}

	for _, tt := range tests {
		result, err := Call("Math.Floor", map[string]interface{}{
			"value": tt.value,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Floor(%f) = %v, want %f", tt.value, result, tt.want)
		}
	}
}

func TestMathCeil(t *testing.T) {
	tests := []struct {
		value, want float64
	}{
		{1.1, 2},
		{1.9, 2},
		{1.0, 1},
		{-1.1, -1},
		{-1.9, -1},
	}

	for _, tt := range tests {
		result, err := Call("Math.Ceil", map[string]interface{}{
			"value": tt.value,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Ceil(%f) = %v, want %f", tt.value, result, tt.want)
		}
	}
}

func TestMathSqrt(t *testing.T) {
	tests := []struct {
		value, want float64
	}{
		{4, 2},
		{9, 3},
		{2, math.Sqrt(2)},
		{0, 0},
		{1, 1},
	}

	for _, tt := range tests {
		result, err := Call("Math.Sqrt", map[string]interface{}{
			"value": tt.value,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if math.Abs(result.(float64)-tt.want) > 0.0001 {
			t.Errorf("Sqrt(%f) = %v, want %f", tt.value, result, tt.want)
		}
	}
}

func TestMathSqrt_Negative(t *testing.T) {
	_, err := Call("Math.Sqrt", map[string]interface{}{
		"value": -4.0,
	})
	if err == nil {
		t.Error("expected error for negative sqrt")
	}
}

func TestMathPow(t *testing.T) {
	tests := []struct {
		base, exp, want float64
	}{
		{2, 3, 8},
		{3, 2, 9},
		{10, 0, 1},
		{5, 1, 5},
		{2, -1, 0.5},
		{4, 0.5, 2},
	}

	for _, tt := range tests {
		result, err := Call("Math.Pow", map[string]interface{}{
			"base":     tt.base,
			"exponent": tt.exp,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if math.Abs(result.(float64)-tt.want) > 0.0001 {
			t.Errorf("Pow(%f, %f) = %v, want %f", tt.base, tt.exp, result, tt.want)
		}
	}
}

func TestMathRandom(t *testing.T) {
	// Default range 0-1
	result, err := Call("Math.Random", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := result.(float64)
	if r < 0 || r > 1 {
		t.Errorf("Random() = %f, expected 0-1", r)
	}

	// Custom range
	result2, err := Call("Math.Random", map[string]interface{}{
		"min": 10.0,
		"max": 20.0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r2 := result2.(float64)
	if r2 < 10 || r2 > 20 {
		t.Errorf("Random(10, 20) = %f, expected 10-20", r2)
	}
}

func TestMathLerp(t *testing.T) {
	tests := []struct {
		a, b, t, want float64
	}{
		{0, 10, 0, 0},
		{0, 10, 1, 10},
		{0, 10, 0.5, 5},
		{0, 10, 0.25, 2.5},
		{-10, 10, 0.5, 0},
		{0, 100, 0.1, 10},
	}

	for _, tt := range tests {
		result, err := Call("Math.Lerp", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
			"t": tt.t,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Lerp(%f, %f, %f) = %v, want %f", tt.a, tt.b, tt.t, result, tt.want)
		}
	}
}

func TestMathLerp_OutOfRange(t *testing.T) {
	// Extrapolation when t > 1
	result, err := Call("Math.Lerp", map[string]interface{}{
		"a": 0.0,
		"b": 10.0,
		"t": 2.0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extrapolate to 20
	if result != 20.0 {
		t.Errorf("Lerp(0, 10, 2) = %v, want 20", result)
	}

	// Extrapolation when t < 0
	result2, err := Call("Math.Lerp", map[string]interface{}{
		"a": 0.0,
		"b": 10.0,
		"t": -0.5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extrapolate to -5
	if result2 != -5.0 {
		t.Errorf("Lerp(0, 10, -0.5) = %v, want -5", result2)
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  float64
		ok    bool
	}{
		{"float64", 3.14, 3.14, true},
		{"float32", float32(3.14), 3.14, true},
		{"int", 42, 42.0, true},
		{"int32", int32(42), 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"uint", uint(42), 42.0, true},
		{"uint32", uint32(42), 42.0, true},
		{"uint64", uint64(42), 42.0, true},
		{"string", "not a number", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat64(tt.input)
			if ok != tt.ok {
				t.Errorf("toFloat64(%v) ok = %v, want %v", tt.input, ok, tt.ok)
				return
			}
			if ok && math.Abs(result-tt.want) > 0.01 {
				t.Errorf("toFloat64(%v) = %f, want %f", tt.input, result, tt.want)
			}
		})
	}
}
