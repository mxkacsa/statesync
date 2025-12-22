package eval

import (
	"context"
	"testing"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// Test state for trigger tests
type TriggerTestState struct {
	Counter int
	Players []TriggerTestPlayer
}

type TriggerTestPlayer struct {
	ID       string
	Health   int
	Position ast.GeoPoint
}

func TestOnChangeTrigger(t *testing.T) {
	state := &TriggerTestState{
		Counter: 0,
		Players: []TriggerTestPlayer{
			{ID: "p1", Health: 100},
		},
	}

	te := NewTriggerEvaluator()
	ctx := NewContext(state, 100*time.Millisecond, 1)

	trigger := &ast.Trigger{
		Type:     ast.TriggerTypeOnChange,
		Watch:    []ast.Path{"$.Counter"},
		RuleName: "TestOnChange",
	}

	// First call - should not fire (initial state)
	fired, err := te.Evaluate(ctx, trigger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fired {
		t.Error("OnChange should not fire on first evaluation (initial state)")
	}

	// Second call - no change, should not fire
	ctx = NewContext(state, 100*time.Millisecond, 2)
	fired, err = te.Evaluate(ctx, trigger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fired {
		t.Error("OnChange should not fire when value hasn't changed")
	}

	// Change value
	state.Counter = 5
	ctx = NewContext(state, 100*time.Millisecond, 3)
	fired, err = te.Evaluate(ctx, trigger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fired {
		t.Error("OnChange should fire when value changes")
	}

	// Same value again - should not fire
	ctx = NewContext(state, 100*time.Millisecond, 4)
	fired, err = te.Evaluate(ctx, trigger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fired {
		t.Error("OnChange should not fire when value hasn't changed again")
	}
}

func TestWaitTrigger(t *testing.T) {
	state := &TriggerTestState{}
	te := NewTriggerEvaluator()

	trigger := &ast.Trigger{
		Type:     ast.TriggerTypeWait,
		Duration: 500, // 500ms = 5 ticks at 100ms
		RuleName: "TestWait",
	}

	// Tick 1 - should not fire (timer starts, elapsed = 0)
	ctx := NewContext(state, 100*time.Millisecond, 1)
	fired, err := te.Evaluate(ctx, trigger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fired {
		t.Error("Wait should not fire immediately")
	}

	// Tick 2-5 - should not fire yet (elapsed = tick - 1 < 5)
	for i := uint64(2); i <= 5; i++ {
		ctx = NewContext(state, 100*time.Millisecond, i)
		fired, _ = te.Evaluate(ctx, trigger)
		if fired {
			t.Errorf("Wait should not fire at tick %d (elapsed = %d, need >= 5)", i, i-1)
		}
	}

	// Tick 6 - should fire (elapsed = 6 - 1 = 5 >= 5)
	ctx = NewContext(state, 100*time.Millisecond, 6)
	fired, err = te.Evaluate(ctx, trigger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fired {
		t.Error("Wait should fire after duration elapsed")
	}

	// Tick 7+ - should NOT fire again (one-shot)
	for i := uint64(7); i <= 10; i++ {
		ctx = NewContext(state, 100*time.Millisecond, i)
		fired, _ = te.Evaluate(ctx, trigger)
		if fired {
			t.Errorf("Wait should not fire again after first fire (tick %d)", i)
		}
	}
}

func TestTimerTriggerRepeat(t *testing.T) {
	state := &TriggerTestState{}
	te := NewTriggerEvaluator()

	trigger := &ast.Trigger{
		Type:     ast.TriggerTypeTimer,
		Duration: 300, // 300ms
		Repeat:   true,
		RuleName: "TestTimerRepeat",
	}

	fireCount := 0
	// Tick through 10 ticks (1000ms total)
	for i := uint64(1); i <= 10; i++ {
		ctx := NewContext(state, 100*time.Millisecond, i)
		fired, _ := te.Evaluate(ctx, trigger)
		if fired {
			fireCount++
		}
	}

	// Should fire approximately 3 times (at 300ms, 600ms, 900ms intervals)
	if fireCount < 2 || fireCount > 4 {
		t.Errorf("Timer with repeat should fire ~3 times in 1000ms with 300ms interval, got %d", fireCount)
	}
}

func TestTimerResetTimer(t *testing.T) {
	state := &TriggerTestState{}
	te := NewTriggerEvaluator()

	trigger := &ast.Trigger{
		Type:     ast.TriggerTypeTimer,
		Duration: 500, // 500ms = 5 ticks
		Repeat:   false,
		RuleName: "TestResetTimer",
	}

	// Run for 4 ticks - timer starts on tick 1, elapsed = 3 after tick 4
	for i := uint64(1); i <= 4; i++ {
		ctx := NewContext(state, 100*time.Millisecond, i)
		te.Evaluate(ctx, trigger)
	}

	// Reset the timer
	te.ResetTimer("TestResetTimer")

	// Now it should need another 6 ticks to fire (start at tick 5, fire when elapsed >= 5)
	// Ticks 5-9: timer restarts at tick 5, elapsed = 0,1,2,3,4 - should not fire
	fired := false
	for i := uint64(5); i <= 9; i++ {
		ctx := NewContext(state, 100*time.Millisecond, i)
		f, _ := te.Evaluate(ctx, trigger)
		if f {
			fired = true
			t.Errorf("Timer should not fire at tick %d after reset", i)
		}
	}

	// Tick 10 - should fire (elapsed = 10 - 5 = 5 >= 5)
	ctx := NewContext(state, 100*time.Millisecond, 10)
	fired, _ = te.Evaluate(ctx, trigger)
	if !fired {
		t.Error("Timer should fire after full duration from reset")
	}
}

func TestScheduleTriggerEvery(t *testing.T) {
	state := &TriggerTestState{}
	te := NewTriggerEvaluator()

	trigger := &ast.Trigger{
		Type:     ast.TriggerTypeSchedule,
		Every:    "100ms",
		RuleName: "TestScheduleEvery",
	}

	// First evaluation should fire (initial)
	ctx := NewContext(state, 100*time.Millisecond, 1)
	fired, err := te.Evaluate(ctx, trigger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fired {
		t.Error("Schedule with 'every' should fire immediately on first call")
	}

	// Next immediate call should not fire (not enough time elapsed)
	fired, _ = te.Evaluate(ctx, trigger)
	if fired {
		t.Error("Schedule should not fire again immediately")
	}
}

func TestCronFieldMatching(t *testing.T) {
	tests := []struct {
		expr     string
		value    int
		expected bool
	}{
		{"*", 5, true},
		{"*", 0, true},
		{"5", 5, true},
		{"5", 6, false},
		{"*/5", 0, true},
		{"*/5", 5, true},
		{"*/5", 10, true},
		{"*/5", 7, false},
		{"1,3,5", 3, true},
		{"1,3,5", 2, false},
		{"1-5", 3, true},
		{"1-5", 0, false},
		{"1-5", 6, false},
	}

	for _, tt := range tests {
		result := matchCronField(tt.expr, tt.value)
		if result != tt.expected {
			t.Errorf("matchCronField(%q, %d) = %v, expected %v", tt.expr, tt.value, result, tt.expected)
		}
	}
}

func TestEnableDisableRule(t *testing.T) {
	state := &TriggerTestState{Counter: 0}

	// Create rules
	rules := []*ast.Rule{
		{
			Name: "IncrementCounter",
			Trigger: &ast.Trigger{
				Type: ast.TriggerTypeOnTick,
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeIncrement,
					Path:  "$.Counter",
					Value: 1,
				},
			},
		},
		{
			Name: "DisableIncrement",
			Trigger: &ast.Trigger{
				Type: ast.TriggerTypeCondition,
				Condition: &ast.Expression{
					Left:  "$.Counter",
					Op:    ">=",
					Right: float64(3),
				},
			},
			Effects: []*ast.Effect{
				{
					Type: ast.EffectTypeDisableRule,
					Rule: "IncrementCounter",
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// Run ticks
	for i := 0; i < 10; i++ {
		engine.TickWithDelta(ctx, 100*time.Millisecond)
	}

	// Counter should stop at 3 because the rule gets disabled
	// After reaching 3, DisableIncrement fires and disables IncrementCounter
	if state.Counter > 4 {
		t.Errorf("Counter should have stopped around 3-4, got %d (rule should be disabled)", state.Counter)
	}
	if state.Counter < 3 {
		t.Errorf("Counter should have reached at least 3, got %d", state.Counter)
	}
}

func TestRuleControlFromEffect(t *testing.T) {
	state := &TriggerTestState{Counter: 0}

	// A rule that fires once using Wait trigger
	rules := []*ast.Rule{
		{
			Name: "OneTimeAction",
			Trigger: &ast.Trigger{
				Type:     ast.TriggerTypeWait,
				Duration: 200, // Fire after 200ms (elapsed >= 2)
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeSet,
					Path:  "$.Counter",
					Value: 100,
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// Run 3 ticks - Wait fires on tick 3 (elapsed = 3-1 = 2 >= 2)
	for i := 0; i < 3; i++ {
		engine.TickWithDelta(ctx, 100*time.Millisecond)
	}

	// Counter should be 100 (set once by Wait trigger)
	if state.Counter != 100 {
		t.Errorf("Counter should be 100, got %d", state.Counter)
	}
}

func TestTriggerDefaultDisabled(t *testing.T) {
	state := &TriggerTestState{Counter: 0}

	disabled := false
	rules := []*ast.Rule{
		{
			Name: "DisabledByDefault",
			Trigger: &ast.Trigger{
				Type:    ast.TriggerTypeOnTick,
				Enabled: &disabled, // Disabled by default
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeIncrement,
					Path:  "$.Counter",
					Value: 1,
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// Run 5 ticks - trigger is disabled, counter should stay 0
	for i := 0; i < 5; i++ {
		engine.TickWithDelta(ctx, 100*time.Millisecond)
	}

	if state.Counter != 0 {
		t.Errorf("Counter should be 0 (trigger disabled), got %d", state.Counter)
	}

	// Enable the trigger
	engine.EnableTrigger("DisabledByDefault")

	// Run 5 more ticks - now it should increment
	for i := 0; i < 5; i++ {
		engine.TickWithDelta(ctx, 100*time.Millisecond)
	}

	if state.Counter != 5 {
		t.Errorf("Counter should be 5 after enabling trigger, got %d", state.Counter)
	}
}

func TestEnableDisableTriggerEffect(t *testing.T) {
	state := &TriggerTestState{Counter: 0}

	disabled := false
	rules := []*ast.Rule{
		{
			Name: "CounterRule",
			Trigger: &ast.Trigger{
				Type:    ast.TriggerTypeOnTick,
				Enabled: &disabled, // Start disabled
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeIncrement,
					Path:  "$.Counter",
					Value: 1,
				},
			},
		},
		{
			Name: "EnableCounterAfter3Ticks",
			Trigger: &ast.Trigger{
				Type:     ast.TriggerTypeWait,
				Duration: 300, // Fire after 300ms
			},
			Effects: []*ast.Effect{
				{
					Type: ast.EffectTypeEnableTrigger,
					Rule: "CounterRule",
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// Run 3 ticks - CounterRule is disabled
	for i := 0; i < 3; i++ {
		engine.TickWithDelta(ctx, 100*time.Millisecond)
	}

	if state.Counter != 0 {
		t.Errorf("Counter should still be 0 before EnableTrigger fires, got %d", state.Counter)
	}

	// Run tick 4 - EnableCounterAfter3Ticks fires and enables CounterRule
	engine.TickWithDelta(ctx, 100*time.Millisecond)

	// Counter might still be 0 if EnableTrigger ran in same tick
	// Run 3 more ticks to see the counter increment
	for i := 0; i < 3; i++ {
		engine.TickWithDelta(ctx, 100*time.Millisecond)
	}

	if state.Counter < 3 {
		t.Errorf("Counter should be at least 3 after trigger was enabled, got %d", state.Counter)
	}
}
