package eval

import (
	"context"
	"testing"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// BatchGameState represents a game state with players for batch tests
type BatchGameState struct {
	Players []*BatchPlayer
	Round   int
}

// BatchPlayer represents a player with current and public positions
type BatchPlayer struct {
	ID              string
	Name            string
	CurrentPosition ast.GeoPoint // Real-time position (private)
	PublicPosition  ast.GeoPoint // Position visible to others (updated periodically)
}

// TestBatchPlayerPositionUpdate tests that every player's publicPosition
// gets updated from their currentPosition after a timer fires.
// This simulates: "every 20 minutes, sync all players' public positions"
// (using smaller time for testing: 500ms)
func TestBatchPlayerPositionUpdate(t *testing.T) {
	// Setup initial state with 3 players
	state := &BatchGameState{
		Round: 1,
		Players: []*BatchPlayer{
			{
				ID:              "player1",
				Name:            "Alice",
				CurrentPosition: ast.GeoPoint{Lat: 47.5, Lon: 19.0}, // Budapest
				PublicPosition:  ast.GeoPoint{Lat: 0, Lon: 0},       // Unknown initially
			},
			{
				ID:              "player2",
				Name:            "Bob",
				CurrentPosition: ast.GeoPoint{Lat: 48.8, Lon: 2.3}, // Paris
				PublicPosition:  ast.GeoPoint{Lat: 0, Lon: 0},
			},
			{
				ID:              "player3",
				Name:            "Charlie",
				CurrentPosition: ast.GeoPoint{Lat: 51.5, Lon: -0.1}, // London
				PublicPosition:  ast.GeoPoint{Lat: 0, Lon: 0},
			},
		},
	}

	// Create rule: Every 500ms, update each player's PublicPosition from CurrentPosition
	rules := []*ast.Rule{
		{
			Name:        "SyncPublicPositions",
			Description: "Periodically sync public positions from current positions",
			Trigger: &ast.Trigger{
				Type:     ast.TriggerTypeTimer,
				Duration: 500, // 500ms for testing (would be 20 minutes = 1200000 in real game)
				Repeat:   true,
			},
			Selector: &ast.Selector{
				Type:   ast.SelectorTypeAll,
				Entity: "Players",
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeSet,
					Path:  "$.PublicPosition",
					Value: "$.CurrentPosition", // Copy current to public for each player
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// Verify initial state - all public positions should be (0,0)
	for _, p := range state.Players {
		if p.PublicPosition.Lat != 0 || p.PublicPosition.Lon != 0 {
			t.Errorf("Player %s should have (0,0) public position initially", p.ID)
		}
	}

	// Run 5 ticks (500ms) - timer should NOT have fired yet (starts at tick 1, needs 5 elapsed)
	for i := 0; i < 5; i++ {
		if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
			t.Fatalf("tick failed: %v", err)
		}
	}

	// Public positions should still be (0,0)
	for _, p := range state.Players {
		if p.PublicPosition.Lat != 0 || p.PublicPosition.Lon != 0 {
			t.Errorf("Player %s public position should still be (0,0) before timer fires, got (%f,%f)",
				p.ID, p.PublicPosition.Lat, p.PublicPosition.Lon)
		}
	}

	// Run tick 6 (600ms total) - timer SHOULD fire (elapsed = 6 - 1 = 5 >= 5)
	if err := engine.TickWithDelta(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	// Now verify all players have their public positions updated
	expectedPositions := map[string]ast.GeoPoint{
		"player1": {Lat: 47.5, Lon: 19.0},
		"player2": {Lat: 48.8, Lon: 2.3},
		"player3": {Lat: 51.5, Lon: -0.1},
	}

	for _, p := range state.Players {
		expected := expectedPositions[p.ID]
		if p.PublicPosition.Lat != expected.Lat || p.PublicPosition.Lon != expected.Lon {
			t.Errorf("Player %s public position should be (%f,%f), got (%f,%f)",
				p.ID, expected.Lat, expected.Lon, p.PublicPosition.Lat, p.PublicPosition.Lon)
		}
	}
}

// TestBatchPositionUpdateWithMovement tests that positions sync correctly
// even when players move between sync intervals
func TestBatchPositionUpdateWithMovement(t *testing.T) {
	state := &BatchGameState{
		Players: []*BatchPlayer{
			{
				ID:              "mover",
				CurrentPosition: ast.GeoPoint{Lat: 10.0, Lon: 10.0},
				PublicPosition:  ast.GeoPoint{Lat: 0, Lon: 0},
			},
		},
	}

	rules := []*ast.Rule{
		{
			Name: "SyncPositions",
			Trigger: &ast.Trigger{
				Type:     ast.TriggerTypeTimer,
				Duration: 300, // 300ms
				Repeat:   true,
			},
			Selector: &ast.Selector{
				Type:   ast.SelectorTypeAll,
				Entity: "Players",
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeSet,
					Path:  "$.PublicPosition",
					Value: "$.CurrentPosition",
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// Run 4 ticks - timer fires on tick 4 (elapsed = 4-1 = 3 >= 3)
	for i := 0; i < 4; i++ {
		engine.TickWithDelta(ctx, 100*time.Millisecond)
	}

	// Should have synced to (10, 10)
	if state.Players[0].PublicPosition.Lat != 10.0 {
		t.Errorf("First sync failed, got lat=%f", state.Players[0].PublicPosition.Lat)
	}

	// Player moves to new position
	state.Players[0].CurrentPosition = ast.GeoPoint{Lat: 20.0, Lon: 20.0}

	// Run 2 ticks - timer shouldn't fire again yet
	for i := 0; i < 2; i++ {
		engine.TickWithDelta(ctx, 100*time.Millisecond)
	}
	if state.Players[0].PublicPosition.Lat != 10.0 {
		t.Errorf("Public position should not update between syncs, got lat=%f", state.Players[0].PublicPosition.Lat)
	}

	// Run 1 more tick - should sync again (300ms from last fire)
	engine.TickWithDelta(ctx, 100*time.Millisecond)

	// Now should have synced to (20, 20)
	if state.Players[0].PublicPosition.Lat != 20.0 {
		t.Errorf("Second sync failed, expected lat=20.0, got lat=%f", state.Players[0].PublicPosition.Lat)
	}
}

// TestWaitThenBatchUpdate tests using Wait trigger for one-time batch update
func TestWaitThenBatchUpdate(t *testing.T) {
	state := &BatchGameState{
		Players: []*BatchPlayer{
			{ID: "p1", CurrentPosition: ast.GeoPoint{Lat: 1, Lon: 1}, PublicPosition: ast.GeoPoint{}},
			{ID: "p2", CurrentPosition: ast.GeoPoint{Lat: 2, Lon: 2}, PublicPosition: ast.GeoPoint{}},
		},
	}

	// Rule: After 200ms, sync all positions ONCE
	rules := []*ast.Rule{
		{
			Name: "OneTimeSync",
			Trigger: &ast.Trigger{
				Type:     ast.TriggerTypeWait,
				Duration: 200, // Fire once after 200ms
			},
			Selector: &ast.Selector{
				Type:   ast.SelectorTypeAll,
				Entity: "Players",
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeSet,
					Path:  "$.PublicPosition",
					Value: "$.CurrentPosition",
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// Run 3 ticks (300ms) - should fire on tick 3 (elapsed = 3-1 = 2 >= 2)
	for i := 0; i < 3; i++ {
		engine.TickWithDelta(ctx, 100*time.Millisecond)
	}

	// Both players should have their positions synced
	if state.Players[0].PublicPosition.Lat != 1 {
		t.Errorf("Player 1 sync failed, got %f", state.Players[0].PublicPosition.Lat)
	}
	if state.Players[1].PublicPosition.Lat != 2 {
		t.Errorf("Player 2 sync failed, got %f", state.Players[1].PublicPosition.Lat)
	}

	// Change positions
	state.Players[0].CurrentPosition = ast.GeoPoint{Lat: 99, Lon: 99}

	// Run more ticks - should NOT sync again (Wait is one-shot)
	for i := 0; i < 10; i++ {
		engine.TickWithDelta(ctx, 100*time.Millisecond)
	}

	// Position should NOT have updated to 99
	if state.Players[0].PublicPosition.Lat == 99 {
		t.Errorf("Wait trigger fired more than once!")
	}
}

// TestScheduleEveryForPositionSync tests using Schedule with "every" for position sync
func TestScheduleEveryForPositionSync(t *testing.T) {
	state := &BatchGameState{
		Players: []*BatchPlayer{
			{ID: "p1", CurrentPosition: ast.GeoPoint{Lat: 5, Lon: 5}, PublicPosition: ast.GeoPoint{}},
		},
	}

	rules := []*ast.Rule{
		{
			Name: "ScheduledSync",
			Trigger: &ast.Trigger{
				Type:  ast.TriggerTypeSchedule,
				Every: "200ms",
			},
			Selector: &ast.Selector{
				Type:   ast.SelectorTypeAll,
				Entity: "Players",
			},
			Effects: []*ast.Effect{
				{
					Type:  ast.EffectTypeSet,
					Path:  "$.PublicPosition",
					Value: "$.CurrentPosition",
				},
			},
		},
	}

	engine := NewEngine(state, rules)
	ctx := context.Background()

	// First tick should trigger (Schedule fires immediately on first call)
	engine.TickWithDelta(ctx, 100*time.Millisecond)

	if state.Players[0].PublicPosition.Lat != 5 {
		t.Errorf("Schedule should fire immediately on first evaluation")
	}
}
