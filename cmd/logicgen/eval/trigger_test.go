package eval

import (
	"testing"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

func TestTriggerEvaluator_OnTick(t *testing.T) {
	te := NewTriggerEvaluator()

	t.Run("every tick", func(t *testing.T) {
		trigger := &ast.Trigger{
			Type:     ast.TriggerTypeOnTick,
			Interval: 0, // Every tick
		}

		for tick := uint64(1); tick <= 10; tick++ {
			ctx := NewContext(&struct{}{}, 100*time.Millisecond, tick)

			result, err := te.Evaluate(ctx, trigger)
			if err != nil {
				t.Fatalf("tick %d: unexpected error: %v", tick, err)
			}

			if !result {
				t.Errorf("tick %d: expected trigger to fire", tick)
			}
		}
	})

	t.Run("interval trigger", func(t *testing.T) {
		trigger := &ast.Trigger{
			Type:     ast.TriggerTypeOnTick,
			Interval: 200, // 200ms = 2 ticks at 100ms
		}

		expectedResults := map[uint64]bool{
			1: false,
			2: true,
			3: false,
			4: true,
			5: false,
			6: true,
		}

		for tick, expected := range expectedResults {
			ctx := NewContext(&struct{}{}, 100*time.Millisecond, tick)

			result, err := te.Evaluate(ctx, trigger)
			if err != nil {
				t.Fatalf("tick %d: unexpected error: %v", tick, err)
			}

			if result != expected {
				t.Errorf("tick %d: expected %v, got %v", tick, expected, result)
			}
		}
	})
}

func TestTriggerEvaluator_OnEvent(t *testing.T) {
	te := NewTriggerEvaluator()

	trigger := &ast.Trigger{
		Type:  ast.TriggerTypeOnEvent,
		Event: "PlayerJoined",
	}

	t.Run("matching event", func(t *testing.T) {
		ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)
		ctx.Event = &ast.Event{Name: "PlayerJoined"}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result {
			t.Error("expected trigger to fire for matching event")
		}
	})

	t.Run("non-matching event", func(t *testing.T) {
		ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)
		ctx.Event = &ast.Event{Name: "PlayerLeft"}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result {
			t.Error("expected trigger NOT to fire for non-matching event")
		}
	})

	t.Run("no event", func(t *testing.T) {
		ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result {
			t.Error("expected trigger NOT to fire when no event")
		}
	})
}

func TestTriggerEvaluator_Distance(t *testing.T) {
	te := NewTriggerEvaluator()

	type TestState struct {
		Player struct {
			Position ast.GeoPoint
		}
		Target struct {
			Position ast.GeoPoint
		}
	}

	t.Run("within distance", func(t *testing.T) {
		state := &TestState{}
		state.Player.Position = ast.GeoPoint{Lat: 47.5, Lon: 19.0}
		state.Target.Position = ast.GeoPoint{Lat: 47.504, Lon: 19.0} // ~445m away

		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type:     ast.TriggerTypeDistance,
			From:     "$.Player.Position",
			To:       "$.Target.Position",
			Operator: "<=",
			Value:    1000, // 1000 meters
			Unit:     "meters",
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result {
			t.Error("expected trigger to fire (within 1000m)")
		}
	})

	t.Run("outside distance", func(t *testing.T) {
		state := &TestState{}
		state.Player.Position = ast.GeoPoint{Lat: 47.5, Lon: 19.0}
		state.Target.Position = ast.GeoPoint{Lat: 47.6, Lon: 19.0} // ~11km away

		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type:     ast.TriggerTypeDistance,
			From:     "$.Player.Position",
			To:       "$.Target.Position",
			Operator: "<=",
			Value:    1000,
			Unit:     "meters",
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result {
			t.Error("expected trigger NOT to fire (outside 1000m)")
		}
	})

	t.Run("greater than operator", func(t *testing.T) {
		state := &TestState{}
		state.Player.Position = ast.GeoPoint{Lat: 47.5, Lon: 19.0}
		state.Target.Position = ast.GeoPoint{Lat: 47.6, Lon: 19.0} // ~11km away

		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type:     ast.TriggerTypeDistance,
			From:     "$.Player.Position",
			To:       "$.Target.Position",
			Operator: ">",
			Value:    5000, // 5km
			Unit:     "meters",
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result {
			t.Error("expected trigger to fire (> 5km)")
		}
	})

	t.Run("kilometers unit", func(t *testing.T) {
		state := &TestState{}
		state.Player.Position = ast.GeoPoint{Lat: 47.5, Lon: 19.0}
		state.Target.Position = ast.GeoPoint{Lat: 47.504, Lon: 19.0} // ~445m away

		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type:     ast.TriggerTypeDistance,
			From:     "$.Player.Position",
			To:       "$.Target.Position",
			Operator: "<=",
			Value:    1, // 1 km
			Unit:     "kilometers",
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result {
			t.Error("expected trigger to fire (within 1km)")
		}
	})
}

func TestTriggerEvaluator_Timer(t *testing.T) {
	te := NewTriggerEvaluator()

	t.Run("timer fires after duration", func(t *testing.T) {
		trigger := &ast.Trigger{
			Type:     ast.TriggerTypeTimer,
			Duration: 500, // 500ms = 5 ticks at 100ms
			Repeat:   false,
		}

		// Ticks 1-4 should not fire
		for tick := uint64(1); tick <= 4; tick++ {
			ctx := NewContext(&struct{}{}, 100*time.Millisecond, tick)

			result, err := te.Evaluate(ctx, trigger)
			if err != nil {
				t.Fatalf("tick %d: unexpected error: %v", tick, err)
			}

			if result {
				t.Errorf("tick %d: timer should not fire yet", tick)
			}
		}

		// Tick 5 should NOT fire (elapsed = 5-1 = 4 < 5)
		ctx := NewContext(&struct{}{}, 100*time.Millisecond, 5)
		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("tick 5: unexpected error: %v", err)
		}

		if result {
			t.Error("tick 5: timer should not fire yet (elapsed = 4)")
		}

		// Tick 6 should fire (elapsed = 6-1 = 5 >= 5)
		ctx = NewContext(&struct{}{}, 100*time.Millisecond, 6)
		result, err = te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("tick 6: unexpected error: %v", err)
		}

		if !result {
			t.Error("tick 6: timer should fire (elapsed = 5)")
		}

		// Tick 7 should not fire (non-repeating, already fired)
		ctx = NewContext(&struct{}{}, 100*time.Millisecond, 7)
		result, err = te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("tick 7: unexpected error: %v", err)
		}

		if result {
			t.Error("tick 7: non-repeating timer should not fire again")
		}
	})

	t.Run("repeating timer", func(t *testing.T) {
		te := NewTriggerEvaluator() // Fresh evaluator

		trigger := &ast.Trigger{
			Type:     ast.TriggerTypeTimer,
			Duration: 300, // 300ms = 3 ticks at 100ms
			Repeat:   true,
		}

		fireCount := 0
		// Run for 12 ticks to see pattern:
		// Tick 1: start, elapsed=0
		// Tick 4: elapsed=3 >= 3, FIRE, reset to tick 4
		// Tick 7: elapsed=3 >= 3, FIRE, reset to tick 7
		// Tick 10: elapsed=3 >= 3, FIRE, reset to tick 10
		for tick := uint64(1); tick <= 12; tick++ {
			ctx := NewContext(&struct{}{}, 100*time.Millisecond, tick)

			result, err := te.Evaluate(ctx, trigger)
			if err != nil {
				t.Fatalf("tick %d: unexpected error: %v", tick, err)
			}

			if result {
				fireCount++
			}
		}

		// Should fire at ticks 4, 7, 10 = 3 times
		if fireCount != 3 {
			t.Errorf("expected 3 fires, got %d", fireCount)
		}
	})
}

func TestTriggerEvaluator_Condition(t *testing.T) {
	te := NewTriggerEvaluator()

	type TestState struct {
		Score  int
		Active bool
	}

	t.Run("simple comparison - true", func(t *testing.T) {
		state := &TestState{Score: 100}
		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type: ast.TriggerTypeCondition,
			Condition: &ast.Expression{
				Left:  "$.Score",
				Op:    ">=",
				Right: 50,
			},
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result {
			t.Error("expected condition to be true (Score >= 50)")
		}
	})

	t.Run("simple comparison - false", func(t *testing.T) {
		state := &TestState{Score: 30}
		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type: ast.TriggerTypeCondition,
			Condition: &ast.Expression{
				Left:  "$.Score",
				Op:    ">=",
				Right: 50,
			},
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result {
			t.Error("expected condition to be false (Score < 50)")
		}
	})

	t.Run("AND condition", func(t *testing.T) {
		state := &TestState{Score: 100, Active: true}
		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type: ast.TriggerTypeCondition,
			Condition: &ast.Expression{
				And: []ast.Expression{
					{Left: "$.Score", Op: ">=", Right: 50},
					{Left: "$.Active", Op: "==", Right: true},
				},
			},
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result {
			t.Error("expected AND condition to be true")
		}
	})

	t.Run("AND condition - one false", func(t *testing.T) {
		state := &TestState{Score: 100, Active: false}
		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type: ast.TriggerTypeCondition,
			Condition: &ast.Expression{
				And: []ast.Expression{
					{Left: "$.Score", Op: ">=", Right: 50},
					{Left: "$.Active", Op: "==", Right: true},
				},
			},
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result {
			t.Error("expected AND condition to be false (Active is false)")
		}
	})

	t.Run("OR condition", func(t *testing.T) {
		state := &TestState{Score: 30, Active: true}
		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type: ast.TriggerTypeCondition,
			Condition: &ast.Expression{
				Or: []ast.Expression{
					{Left: "$.Score", Op: ">=", Right: 50},    // false
					{Left: "$.Active", Op: "==", Right: true}, // true
				},
			},
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result {
			t.Error("expected OR condition to be true (Active is true)")
		}
	})

	t.Run("NOT condition", func(t *testing.T) {
		state := &TestState{Active: false}
		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type: ast.TriggerTypeCondition,
			Condition: &ast.Expression{
				Not: &ast.Expression{
					Left:  "$.Active",
					Op:    "==",
					Right: true,
				},
			},
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result {
			t.Error("expected NOT(Active==true) to be true")
		}
	})

	t.Run("nil condition", func(t *testing.T) {
		state := &TestState{}
		ctx := NewContext(state, 100*time.Millisecond, 1)

		trigger := &ast.Trigger{
			Type:      ast.TriggerTypeCondition,
			Condition: nil,
		}

		result, err := te.Evaluate(ctx, trigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result {
			t.Error("expected nil condition to be true (default)")
		}
	})
}

func TestTriggerEvaluator_NilTrigger(t *testing.T) {
	te := NewTriggerEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	result, err := te.Evaluate(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result {
		t.Error("expected nil trigger to always fire")
	}
}

func TestTriggerEvaluator_UnknownType(t *testing.T) {
	te := NewTriggerEvaluator()
	ctx := NewContext(&struct{}{}, 100*time.Millisecond, 1)

	trigger := &ast.Trigger{
		Type: "UnknownType",
	}

	_, err := te.Evaluate(ctx, trigger)
	if err == nil {
		t.Error("expected error for unknown trigger type")
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		left     interface{}
		right    interface{}
		op       string
		expected bool
	}{
		// Numeric comparisons
		{"int equal", 10, 10, "==", true},
		{"int not equal", 10, 20, "==", false},
		{"int greater", 20, 10, ">", true},
		{"int less", 10, 20, "<", true},
		{"int >=", 10, 10, ">=", true},
		{"int <=", 10, 10, "<=", true},
		{"float equal", 10.5, 10.5, "==", true},
		{"mixed types", 10, 10.0, "==", true},

		// String comparisons
		{"string equal", "hello", "hello", "==", true},
		{"string not equal", "hello", "world", "!=", true},
		{"string greater", "world", "hello", ">", true},
		{"string less", "hello", "world", "<", true},

		// Nil comparisons
		{"nil equal", nil, nil, "==", true},
		{"nil not equal", nil, "value", "!=", true},
		{"value not equal nil", "value", nil, "!=", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compare(tt.left, tt.right, tt.op)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("compare(%v, %v, %s) = %v, expected %v",
					tt.left, tt.right, tt.op, result, tt.expected)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		ok       bool
	}{
		{"float64", float64(3.14), 3.14, true},
		{"float32", float32(3.14), float64(float32(3.14)), true},
		{"int", 42, 42.0, true},
		{"int32", int32(42), 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"uint", uint(42), 42.0, true},
		{"uint32", uint32(42), 42.0, true},
		{"uint64", uint64(42), 42.0, true},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat64(tt.input)

			if ok != tt.ok {
				t.Errorf("ok = %v, expected %v", ok, tt.ok)
			}

			if ok && result != tt.expected {
				t.Errorf("result = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestToGeoPoint(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{"GeoPoint value", ast.GeoPoint{Lat: 47.5, Lon: 19.0}, false},
		{"GeoPoint pointer", &ast.GeoPoint{Lat: 47.5, Lon: 19.0}, false},
		{"map lowercase", map[string]interface{}{"lat": 47.5, "lon": 19.0}, false},
		{"map uppercase", map[string]interface{}{"Lat": 47.5, "Lon": 19.0}, false},
		{"invalid map", map[string]interface{}{"x": 47.5, "y": 19.0}, true},
		{"nil pointer", (*ast.GeoPoint)(nil), true},
		{"string", "invalid", true},
		{"number", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toGeoPoint(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if result.Lat != 47.5 || result.Lon != 19.0 {
					t.Errorf("unexpected result: %+v", result)
				}
			}
		})
	}
}
