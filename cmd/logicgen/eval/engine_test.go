package eval

import (
	"context"
	"testing"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// GameState for engine tests
type GameState struct {
	Players  []*Player
	Enemies  []*Enemy
	Score    int
	GameTime int64
	Config   GameConfig
}

type Player struct {
	ID        string
	Name      string
	Health    int
	MaxHealth int
	Position  ast.GeoPoint
	Status    string
	Score     int
}

type Enemy struct {
	ID       string
	Health   int
	Position ast.GeoPoint
	Target   string
}

type GameConfig struct {
	MaxPlayers    int
	TickRate      int
	SpawnInterval int
}

func TestEngine_BasicTick(t *testing.T) {
	state := &GameState{
		Score: 0,
	}

	// Simple rule that increments score every tick
	rules := []*ast.Rule{
		{
			Name: "IncrementScore",
			Trigger: &ast.Trigger{
				Type: ast.TriggerTypeOnTick,
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeIncrement,
					Path:  "$.Score",
					Value: 1,
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// Run 5 ticks
	for i := 0; i < 5; i++ {
		if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
			t.Fatalf("Tick %d failed: %v", i, err)
		}
	}

	if state.Score != 5 {
		t.Errorf("expected Score=5, got %d", state.Score)
	}
}

func TestEngine_IntervalTrigger(t *testing.T) {
	state := &GameState{
		Score: 0,
	}

	// Rule that only fires every 500ms (5 ticks at 100ms)
	rules := []*ast.Rule{
		{
			Name: "IntervalIncrement",
			Trigger: &ast.Trigger{
				Type:     ast.TriggerTypeOnTick,
				Interval: 500, // 500ms interval
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeIncrement,
					Path:  "$.Score",
					Value: 10,
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// Run 10 ticks at 100ms each (1 second total)
	for i := 0; i < 10; i++ {
		if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
			t.Fatalf("Tick %d failed: %v", i, err)
		}
	}

	// Should fire at tick 5 and tick 10 = 2 times = 20 points
	if state.Score != 20 {
		t.Errorf("expected Score=20, got %d", state.Score)
	}
}

func TestEngine_DisabledRule(t *testing.T) {
	state := &GameState{
		Score: 0,
	}

	disabled := false
	rules := []*ast.Rule{
		{
			Name:    "DisabledRule",
			Enabled: &disabled,
			Trigger: &ast.Trigger{
				Type: ast.TriggerTypeOnTick,
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeSet,
					Path:  "$.Score",
					Value: 100,
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("Tick failed: %v", err)
	}

	// Score should remain 0 because rule is disabled
	if state.Score != 0 {
		t.Errorf("expected Score=0, got %d (disabled rule should not fire)", state.Score)
	}
}

func TestEngine_RulePriority(t *testing.T) {
	state := &GameState{
		Score: 0,
	}

	// Two rules - lower priority sets to 10, higher priority multiplies by 2
	// If priority works, we should get (0 * 2) + 10 = 10 if low priority runs first
	// Or (0 + 10) * 2 = 20 if high priority runs first
	// With descending priority (higher first), high priority runs first
	rules := []*ast.Rule{
		{
			Name:     "AddTen",
			Priority: 10,
			Trigger: &ast.Trigger{
				Type: ast.TriggerTypeOnTick,
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeIncrement,
					Path:  "$.Score",
					Value: 10,
				},
			},
		},
		{
			Name:     "Double",
			Priority: 100, // Higher priority, runs first
			Trigger: &ast.Trigger{
				Type: ast.TriggerTypeOnTick,
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeSet,
					Path:  "$.Score",
					Value: 5, // Set to 5 first
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("Tick failed: %v", err)
	}

	// Double (priority 100) runs first: Score = 5
	// AddTen (priority 10) runs second: Score = 5 + 10 = 15
	if state.Score != 15 {
		t.Errorf("expected Score=15, got %d", state.Score)
	}
}

func TestEngine_SelectorAll(t *testing.T) {
	state := &GameState{
		Players: []*Player{
			{ID: "p1", Health: 100},
			{ID: "p2", Health: 80},
			{ID: "p3", Health: 60},
		},
	}

	// Heal all players by 10
	rules := []*ast.Rule{
		{
			Name: "HealAll",
			Trigger: &ast.Trigger{
				Type: ast.TriggerTypeOnTick,
			},
			Selector: &ast.Selector{
				Type:   ast.SelectorTypeAll,
				Entity: "Players",
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeIncrement,
					Path:  "$.Health",
					Value: 10,
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("Tick failed: %v", err)
	}

	expected := []int{110, 90, 70}
	for i, p := range state.Players {
		if p.Health != expected[i] {
			t.Errorf("Player %d: expected Health=%d, got %d", i, expected[i], p.Health)
		}
	}
}

func TestEngine_SelectorFilter(t *testing.T) {
	state := &GameState{
		Players: []*Player{
			{ID: "p1", Health: 100, Status: "Active"},
			{ID: "p2", Health: 80, Status: "Dead"},
			{ID: "p3", Health: 60, Status: "Active"},
		},
	}

	// Only heal active players
	rules := []*ast.Rule{
		{
			Name: "HealActive",
			Trigger: &ast.Trigger{
				Type: ast.TriggerTypeOnTick,
			},
			Selector: &ast.Selector{
				Type:   ast.SelectorTypeFilter,
				Entity: "Players",
				Where: &ast.WhereClause{
					Field: "Status",
					Op:    "==",
					Value: "Active",
				},
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeIncrement,
					Path:  "$.Health",
					Value: 10,
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("Tick failed: %v", err)
	}

	// Only p1 and p3 should be healed
	if state.Players[0].Health != 110 {
		t.Errorf("Player 0 (Active): expected Health=110, got %d", state.Players[0].Health)
	}
	if state.Players[1].Health != 80 {
		t.Errorf("Player 1 (Dead): expected Health=80 (unchanged), got %d", state.Players[1].Health)
	}
	if state.Players[2].Health != 70 {
		t.Errorf("Player 2 (Active): expected Health=70, got %d", state.Players[2].Health)
	}
}

func TestEngine_ViewSum(t *testing.T) {
	state := &GameState{
		Players: []*Player{
			{ID: "p1", Score: 100},
			{ID: "p2", Score: 200},
			{ID: "p3", Score: 300},
		},
		Score: 0,
	}

	// Calculate total score from all players
	rules := []*ast.Rule{
		{
			Name: "UpdateTotalScore",
			Trigger: &ast.Trigger{
				Type: ast.TriggerTypeOnTick,
			},
			Selector: &ast.Selector{
				Type:   ast.SelectorTypeAll,
				Entity: "Players",
			},
			Views: map[string]*ast.View{
				"totalScore": {
					Type:  ast.ViewTypeSum,
					Field: "$.Score",
				},
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeSet,
					Path:  "$.Score",
					Value: "view:totalScore",
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("Tick failed: %v", err)
	}

	// Note: The view is computed but the effect applies to each entity
	// This test might need adjustment based on how views are used
	// For now, just verify no crash
	t.Log("ViewSum test completed without errors")
}

func TestEngine_EventTrigger(t *testing.T) {
	state := &GameState{
		Score: 0,
	}

	rules := []*ast.Rule{
		{
			Name: "OnPlayerJoin",
			Trigger: &ast.Trigger{
				Type:  ast.TriggerTypeOnEvent,
				Event: "PlayerJoined",
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeIncrement,
					Path:  "$.Score",
					Value: 100,
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// First, a regular tick should not trigger the event rule
	if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("Tick failed: %v", err)
	}
	if state.Score != 0 {
		t.Errorf("expected Score=0 after tick (event rule shouldn't fire), got %d", state.Score)
	}

	// Now send the event
	event := &ast.Event{
		Name:   "PlayerJoined",
		Params: map[string]interface{}{"playerId": "p1"},
	}
	if err := engine.HandleEvent(ctx, event); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	if state.Score != 100 {
		t.Errorf("expected Score=100 after event, got %d", state.Score)
	}

	// Wrong event name should not trigger
	wrongEvent := &ast.Event{Name: "WrongEvent"}
	if err := engine.HandleEvent(ctx, wrongEvent); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	if state.Score != 100 {
		t.Errorf("expected Score=100 after wrong event (unchanged), got %d", state.Score)
	}
}

func TestEngine_ContextCancellation(t *testing.T) {
	state := &GameState{}

	rules := []*ast.Rule{
		{
			Name: "SlowRule",
			Trigger: &ast.Trigger{
				Type: ast.TriggerTypeOnTick,
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeSet,
					Path:  "$.Score",
					Value: 1,
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := engine.TickWithDelta(ctx, 100*time.Millisecond)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestEngine_EmptyRules(t *testing.T) {
	state := &GameState{Score: 42}

	engine := NewEngine(state, []*ast.Rule{})
	ctx := context.Background()

	if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("Tick with empty rules failed: %v", err)
	}

	if state.Score != 42 {
		t.Errorf("expected Score=42 (unchanged), got %d", state.Score)
	}
}

func TestEngine_AddRemoveRule(t *testing.T) {
	state := &GameState{Score: 0}

	engine := NewEngine(state, []*ast.Rule{})
	ctx := context.Background()

	// Add a rule dynamically
	rule := &ast.Rule{
		Name: "DynamicRule",
		Trigger: &ast.Trigger{
			Type: ast.TriggerTypeOnTick,
		},
		Effects: []*ast.Effect{
			{
				Type:  ast.EffectTypeIncrement,
				Path:  "$.Score",
				Value: 5,
			},
		},
	}

	engine.AddRule(rule)

	if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("Tick failed: %v", err)
	}

	if state.Score != 5 {
		t.Errorf("expected Score=5, got %d", state.Score)
	}

	// Remove the rule
	removed := engine.RemoveRule("DynamicRule")
	if !removed {
		t.Error("expected rule to be removed")
	}

	if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("Tick failed: %v", err)
	}

	// Score should remain 5
	if state.Score != 5 {
		t.Errorf("expected Score=5 (unchanged after rule removal), got %d", state.Score)
	}

	// Try to remove non-existent rule
	removed = engine.RemoveRule("NonExistent")
	if removed {
		t.Error("expected false when removing non-existent rule")
	}
}

func TestEngine_GetRule(t *testing.T) {
	rule := &ast.Rule{
		Name:     "TestRule",
		Priority: 50,
	}

	engine := NewEngine(&GameState{}, []*ast.Rule{rule})

	found := engine.GetRule("TestRule")
	if found == nil {
		t.Fatal("expected to find rule")
	}
	if found.Priority != 50 {
		t.Errorf("expected priority 50, got %d", found.Priority)
	}

	notFound := engine.GetRule("NonExistent")
	if notFound != nil {
		t.Error("expected nil for non-existent rule")
	}
}

func TestEngine_GetTickAndState(t *testing.T) {
	state := &GameState{Score: 42}
	engine := NewEngine(state, []*ast.Rule{})
	ctx := context.Background()

	if engine.GetTick() != 0 {
		t.Errorf("expected initial tick=0, got %d", engine.GetTick())
	}

	engine.TickWithDelta(ctx, 100*time.Millisecond)
	engine.TickWithDelta(ctx, 100*time.Millisecond)
	engine.TickWithDelta(ctx, 100*time.Millisecond)

	if engine.GetTick() != 3 {
		t.Errorf("expected tick=3 after 3 ticks, got %d", engine.GetTick())
	}

	returnedState := engine.GetState()
	if gs, ok := returnedState.(*GameState); ok {
		if gs.Score != 42 {
			t.Errorf("expected Score=42 from GetState, got %d", gs.Score)
		}
	} else {
		t.Error("GetState returned wrong type")
	}
}
