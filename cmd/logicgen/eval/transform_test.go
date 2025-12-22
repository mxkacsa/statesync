package eval

import (
	"math"
	"testing"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

func TestTransformEvaluator_Add(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		left     interface{}
		right    interface{}
		expected float64
	}{
		{"integers", 10, 20, 30},
		{"floats", 10.5, 20.5, 31},
		{"mixed", 10, 20.5, 30.5},
		{"negative", -10, 20, 10},
		{"zero", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type:  ast.TransformTypeAdd,
				Left:  tt.left,
				Right: tt.right,
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTransformEvaluator_Subtract(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	transform := &ast.Transform{
		Type:  ast.TransformTypeSubtract,
		Left:  100,
		Right: 30,
	}

	result, err := te.Evaluate(ctx, transform)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != float64(70) {
		t.Errorf("expected 70, got %v", result)
	}
}

func TestTransformEvaluator_Multiply(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		left     interface{}
		right    interface{}
		expected float64
	}{
		{"positive", 5, 6, 30},
		{"negative", -5, 6, -30},
		{"zero", 100, 0, 0},
		{"decimal", 2.5, 4, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type:  ast.TransformTypeMultiply,
				Left:  tt.left,
				Right: tt.right,
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTransformEvaluator_Divide(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	t.Run("normal division", func(t *testing.T) {
		transform := &ast.Transform{
			Type:  ast.TransformTypeDivide,
			Left:  100,
			Right: 4,
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != float64(25) {
			t.Errorf("expected 25, got %v", result)
		}
	})

	t.Run("division by zero", func(t *testing.T) {
		transform := &ast.Transform{
			Type:  ast.TransformTypeDivide,
			Left:  100,
			Right: 0,
		}

		_, err := te.Evaluate(ctx, transform)
		if err == nil {
			t.Error("expected division by zero error")
		}
	})
}

func TestTransformEvaluator_Modulo(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		left     interface{}
		right    interface{}
		expected float64
	}{
		{"normal", 17, 5, 2},
		{"exact division", 20, 5, 0},
		{"decimal", 17.5, 5, 2.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type:  ast.TransformTypeModulo,
				Left:  tt.left,
				Right: tt.right,
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}

	t.Run("modulo by zero", func(t *testing.T) {
		transform := &ast.Transform{
			Type:  ast.TransformTypeModulo,
			Left:  17,
			Right: 0,
		}

		_, err := te.Evaluate(ctx, transform)
		if err == nil {
			t.Error("expected modulo by zero error")
		}
	})
}

func TestTransformEvaluator_Clamp(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		value    interface{}
		min      interface{}
		max      interface{}
		expected float64
	}{
		{"within range", 50, 0, 100, 50},
		{"below min", -10, 0, 100, 0},
		{"above max", 150, 0, 100, 100},
		{"at min", 0, 0, 100, 0},
		{"at max", 100, 0, 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type:  ast.TransformTypeClamp,
				Value: tt.value,
				Min:   tt.min,
				Max:   tt.max,
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTransformEvaluator_Round(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		value    interface{}
		expected float64
	}{
		{"round down", 3.2, 3},
		{"round up", 3.7, 4},
		{"round half up", 3.5, 4},
		{"negative round", -3.7, -4},
		{"integer", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type:  ast.TransformTypeRound,
				Value: tt.value,
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTransformEvaluator_Floor(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		value    interface{}
		expected float64
	}{
		{"positive", 3.9, 3},
		{"negative", -3.1, -4},
		{"integer", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type:  ast.TransformTypeFloor,
				Value: tt.value,
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTransformEvaluator_Ceil(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		value    interface{}
		expected float64
	}{
		{"positive", 3.1, 4},
		{"negative", -3.9, -3},
		{"integer", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type:  ast.TransformTypeCeil,
				Value: tt.value,
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTransformEvaluator_Abs(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		value    interface{}
		expected float64
	}{
		{"positive", 42, 42},
		{"negative", -42, 42},
		{"zero", 0, 0},
		{"negative decimal", -3.14, 3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type:  ast.TransformTypeAbs,
				Value: tt.value,
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTransformEvaluator_MinMax(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	t.Run("Min", func(t *testing.T) {
		transform := &ast.Transform{
			Type:  ast.TransformTypeMin,
			Left:  10,
			Right: 20,
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != float64(10) {
			t.Errorf("expected 10, got %v", result)
		}
	})

	t.Run("Max", func(t *testing.T) {
		transform := &ast.Transform{
			Type:  ast.TransformTypeMax,
			Left:  10,
			Right: 20,
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != float64(20) {
			t.Errorf("expected 20, got %v", result)
		}
	})
}

func TestTransformEvaluator_Random(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	transform := &ast.Transform{
		Type:     ast.TransformTypeRandom,
		MinValue: 10,
		MaxValue: 20,
	}

	// Test multiple times to verify range
	for i := 0; i < 100; i++ {
		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		f := result.(float64)
		if f < 10 || f > 20 {
			t.Errorf("random value %v out of range [10, 20]", f)
		}
	}
}

func TestTransformEvaluator_MoveTowards(t *testing.T) {
	te := NewTransformEvaluator()

	type TestEntity struct {
		Position ast.GeoPoint
		Target   ast.GeoPoint
	}

	entity := &TestEntity{
		Position: ast.GeoPoint{Lat: 47.5, Lon: 19.0},
		Target:   ast.GeoPoint{Lat: 47.6, Lon: 19.0}, // ~11km north
	}

	ctx := NewContext(entity, 1000*time.Millisecond, 1) // 1 second delta
	ctx.CurrentEntity = entity

	transform := &ast.Transform{
		Type:    ast.TransformTypeMoveTowards,
		Current: entity.Position,
		Target:  entity.Target,
		Speed:   1000, // 1000 m/s
		Unit:    "m/s",
	}

	result, err := te.Evaluate(ctx, transform)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newPos, ok := result.(ast.GeoPoint)
	if !ok {
		t.Fatalf("expected GeoPoint, got %T", result)
	}

	// Should have moved ~1000m towards target
	originalDist := haversineDistance(entity.Position, entity.Target)
	newDist := haversineDistance(newPos, entity.Target)

	if newDist >= originalDist {
		t.Errorf("expected to move closer to target, originalDist=%v, newDist=%v", originalDist, newDist)
	}

	// Should have moved approximately 1000m
	movedDist := haversineDistance(entity.Position, newPos)
	if math.Abs(movedDist-1000) > 50 { // Allow 50m tolerance
		t.Errorf("expected to move ~1000m, moved %v", movedDist)
	}
}

func TestTransformEvaluator_GpsDistance(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	from := ast.GeoPoint{Lat: 47.5, Lon: 19.0}
	to := ast.GeoPoint{Lat: 47.6, Lon: 19.0} // ~11km north

	t.Run("meters", func(t *testing.T) {
		transform := &ast.Transform{
			Type: ast.TransformTypeGpsDistance,
			From: from,
			To:   to,
			Unit: "meters",
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		dist := result.(float64)
		if dist < 10000 || dist > 12000 {
			t.Errorf("expected ~11000m, got %v", dist)
		}
	})

	t.Run("kilometers", func(t *testing.T) {
		transform := &ast.Transform{
			Type: ast.TransformTypeGpsDistance,
			From: from,
			To:   to,
			Unit: "km",
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		dist := result.(float64)
		if dist < 10 || dist > 12 {
			t.Errorf("expected ~11km, got %v", dist)
		}
	})
}

func TestTransformEvaluator_GpsBearing(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		from     ast.GeoPoint
		to       ast.GeoPoint
		expected float64
		delta    float64
	}{
		{
			name:     "north",
			from:     ast.GeoPoint{Lat: 0, Lon: 0},
			to:       ast.GeoPoint{Lat: 1, Lon: 0},
			expected: 0,
			delta:    1,
		},
		{
			name:     "east",
			from:     ast.GeoPoint{Lat: 0, Lon: 0},
			to:       ast.GeoPoint{Lat: 0, Lon: 1},
			expected: 90,
			delta:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type: ast.TransformTypeGpsBearing,
				From: tt.from,
				To:   tt.to,
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			bearing := result.(float64)
			if math.Abs(bearing-tt.expected) > tt.delta {
				t.Errorf("expected ~%v, got %v", tt.expected, bearing)
			}
		})
	}
}

func TestTransformEvaluator_PointInRadius(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	center := ast.GeoPoint{Lat: 47.5, Lon: 19.0}

	tests := []struct {
		name     string
		point    ast.GeoPoint
		radius   float64
		expected bool
	}{
		{
			name:     "inside",
			point:    ast.GeoPoint{Lat: 47.504, Lon: 19.0}, // ~445m
			radius:   1000,
			expected: true,
		},
		{
			name:     "outside",
			point:    ast.GeoPoint{Lat: 47.6, Lon: 19.0}, // ~11km
			radius:   1000,
			expected: false,
		},
		{
			name:     "at center",
			point:    center,
			radius:   100,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type:   ast.TransformTypePointInRadius,
				Value:  tt.point,
				Center: center,
				Radius: tt.radius,
				Unit:   "meters",
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTransformEvaluator_StringOperations(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	t.Run("Concat", func(t *testing.T) {
		transform := &ast.Transform{
			Type:    ast.TransformTypeConcat,
			Strings: []interface{}{"Hello", " ", "World"},
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "Hello World" {
			t.Errorf("expected 'Hello World', got %v", result)
		}
	})

	t.Run("Format", func(t *testing.T) {
		transform := &ast.Transform{
			Type:   ast.TransformTypeFormat,
			Format: "Player %s scored %d points",
			Args:   []interface{}{"Alice", 100},
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "Player Alice scored 100 points" {
			t.Errorf("expected 'Player Alice scored 100 points', got %v", result)
		}
	})

	t.Run("ToUpper", func(t *testing.T) {
		transform := &ast.Transform{
			Type:  ast.TransformTypeToUpper,
			Value: "hello",
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "HELLO" {
			t.Errorf("expected 'HELLO', got %v", result)
		}
	})

	t.Run("ToLower", func(t *testing.T) {
		transform := &ast.Transform{
			Type:  ast.TransformTypeToLower,
			Value: "HELLO",
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "hello" {
			t.Errorf("expected 'hello', got %v", result)
		}
	})

	t.Run("Trim", func(t *testing.T) {
		transform := &ast.Transform{
			Type:  ast.TransformTypeTrim,
			Value: "  hello  ",
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "hello" {
			t.Errorf("expected 'hello', got %v", result)
		}
	})

	t.Run("Substring", func(t *testing.T) {
		transform := &ast.Transform{
			Type:   ast.TransformTypeSubstring,
			Value:  "Hello World",
			Start:  0,
			Length: 5,
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "Hello" {
			t.Errorf("expected 'Hello', got %v", result)
		}
	})
}

func TestTransformEvaluator_If(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	t.Run("true condition", func(t *testing.T) {
		transform := &ast.Transform{
			Type:      ast.TransformTypeIf,
			Condition: true,
			Then:      "yes",
			Else:      "no",
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "yes" {
			t.Errorf("expected 'yes', got %v", result)
		}
	})

	t.Run("false condition", func(t *testing.T) {
		transform := &ast.Transform{
			Type:      ast.TransformTypeIf,
			Condition: false,
			Then:      "yes",
			Else:      "no",
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "no" {
			t.Errorf("expected 'no', got %v", result)
		}
	})

	t.Run("truthy number", func(t *testing.T) {
		transform := &ast.Transform{
			Type:      ast.TransformTypeIf,
			Condition: 42, // non-zero is truthy
			Then:      "yes",
			Else:      "no",
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "yes" {
			t.Errorf("expected 'yes', got %v", result)
		}
	})

	t.Run("falsy zero", func(t *testing.T) {
		transform := &ast.Transform{
			Type:      ast.TransformTypeIf,
			Condition: 0, // zero is falsy
			Then:      "yes",
			Else:      "no",
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "no" {
			t.Errorf("expected 'no', got %v", result)
		}
	})
}

func TestTransformEvaluator_Coalesce(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	t.Run("first non-nil", func(t *testing.T) {
		transform := &ast.Transform{
			Type:   ast.TransformTypeCoalesce,
			Values: []interface{}{nil, nil, "found", "ignored"},
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "found" {
			t.Errorf("expected 'found', got %v", result)
		}
	})

	t.Run("first value non-nil", func(t *testing.T) {
		transform := &ast.Transform{
			Type:   ast.TransformTypeCoalesce,
			Values: []interface{}{"first", nil, "second"},
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "first" {
			t.Errorf("expected 'first', got %v", result)
		}
	})

	t.Run("all nil", func(t *testing.T) {
		transform := &ast.Transform{
			Type:   ast.TransformTypeCoalesce,
			Values: []interface{}{nil, nil, nil},
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestTransformEvaluator_Not(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"not true", true, false},
		{"not false", false, true},
		{"not 0", 0, true},
		{"not 1", 1, false},
		{"not empty string", "", true},
		{"not non-empty string", "hello", false},
		{"not nil", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &ast.Transform{
				Type:  ast.TransformTypeNot,
				Value: tt.value,
			}

			result, err := te.Evaluate(ctx, transform)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTransformEvaluator_UUID(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	transform := &ast.Transform{
		Type: ast.TransformTypeUUID,
	}

	// Generate multiple UUIDs and verify format
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		uuid, ok := result.(string)
		if !ok {
			t.Fatalf("expected string, got %T", result)
		}

		// Verify format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
		if len(uuid) != 36 {
			t.Errorf("expected length 36, got %d", len(uuid))
		}

		if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
			t.Errorf("invalid UUID format: %s", uuid)
		}

		if uuid[14] != '4' {
			t.Errorf("expected version 4, got %c", uuid[14])
		}

		// Verify uniqueness
		if seen[uuid] {
			t.Errorf("duplicate UUID generated: %s", uuid)
		}
		seen[uuid] = true
	}
}

func TestTransformEvaluator_TimeOperations(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	t.Run("Now", func(t *testing.T) {
		transform := &ast.Transform{Type: ast.TransformTypeNow}

		before := time.Now().UnixMilli()
		result, err := te.Evaluate(ctx, transform)
		after := time.Now().UnixMilli()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ts := result.(int64)
		if ts < before || ts > after {
			t.Errorf("timestamp %d not in range [%d, %d]", ts, before, after)
		}
	})

	t.Run("TimeAdd", func(t *testing.T) {
		transform := &ast.Transform{
			Type:     ast.TransformTypeTimeAdd,
			Value:    int64(1000),
			Duration: 500,
		}

		result, err := te.Evaluate(ctx, transform)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != int64(1500) {
			t.Errorf("expected 1500, got %v", result)
		}
	})
}

func TestTransformEvaluator_NilTransform(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	_, err := te.Evaluate(ctx, nil)
	if err == nil {
		t.Error("expected error for nil transform")
	}
}

func TestTransformEvaluator_UnknownType(t *testing.T) {
	te := NewTransformEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	transform := &ast.Transform{
		Type: "UnknownType",
	}

	_, err := te.Evaluate(ctx, transform)
	if err == nil {
		t.Error("expected error for unknown transform type")
	}
}
