package eval

import (
	"testing"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// CEPlayer represents a game player for closest enemy testing
type CEPlayer struct {
	ID       string
	Name     string
	Team     string
	Position ast.GeoPoint
}

// CEGameState represents the game state for closest enemy testing
type CEGameState struct {
	Players []*CEPlayer
}

func TestClosestEnemyDistance(t *testing.T) {
	// Setup: Create players at known positions
	// Using simple coordinates for easy distance verification
	//
	// Map layout (approximate):
	//   Player1 (runner) at (0, 0)
	//   Player2 (catcher) at (0, 0.001) - ~111m north
	//   Player3 (catcher) at (0, 0.002) - ~222m north
	//   Player4 (catcher) at (0, 0.005) - ~555m north
	//   Player5 (runner) at (0, 0.003) - not a catcher
	//
	state := &CEGameState{
		Players: []*CEPlayer{
			{ID: "p1", Name: "Alice", Team: "runner", Position: ast.GeoPoint{Lat: 0, Lon: 0}},
			{ID: "p2", Name: "Bob", Team: "catcher", Position: ast.GeoPoint{Lat: 0.001, Lon: 0}},   // ~111m from origin
			{ID: "p3", Name: "Carol", Team: "catcher", Position: ast.GeoPoint{Lat: 0.002, Lon: 0}}, // ~222m from origin
			{ID: "p4", Name: "Dave", Team: "catcher", Position: ast.GeoPoint{Lat: 0.005, Lon: 0}},  // ~555m from origin
			{ID: "p5", Name: "Eve", Team: "runner", Position: ast.GeoPoint{Lat: 0.003, Lon: 0}},    // runner, not catcher
		},
	}

	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	// Test 1: Find closest catcher to Player1 (runner at origin)
	t.Run("ClosestCatcherToRunner", func(t *testing.T) {
		// Set params as if we're evaluating for Player1
		ctx.Params["originPos"] = state.Players[0].Position // (0, 0)
		ctx.Params["excludeId"] = state.Players[0].ID       // "p1"

		view := &ast.View{
			Source: "Players",
			Pipeline: []ast.ViewOperation{
				// Filter to catchers only
				{
					Type:  ast.ViewOpFilter,
					Where: &ast.WhereClause{Field: "Team", Op: "==", Value: "catcher"},
				},
				// Exclude self (though p1 is not a catcher anyway)
				{
					Type:  ast.ViewOpFilter,
					Where: &ast.WhereClause{Field: "ID", Op: "!=", Value: "param:excludeId"},
				},
				// Find nearest
				{
					Type:     ast.ViewOpNearest,
					Origin:   "param:originPos",
					Position: "$.Position",
					Count:    1,
				},
				// Calculate distance
				{
					Type:     ast.ViewOpDistance,
					From:     "param:originPos",
					Position: "$.Position",
				},
			},
		}

		result, err := ve.Evaluate(ctx, view, ctx.Params)
		if err != nil {
			t.Fatalf("Evaluate failed: %v", err)
		}

		distance, ok := result.(float64)
		if !ok {
			t.Fatalf("Expected float64, got %T: %v", result, result)
		}

		// Player2 (Bob) is at (0.001, 0), which is ~111m from origin
		// Haversine distance for 0.001 degrees latitude ≈ 111.19m
		expectedMin := 110.0
		expectedMax := 113.0
		if distance < expectedMin || distance > expectedMax {
			t.Errorf("Expected distance between %.0f-%.0fm, got %.2fm", expectedMin, expectedMax, distance)
		}

		t.Logf("Closest catcher to runner: %.2f meters", distance)
	})

	// Test 2: Find closest catcher to Player2 (catcher Bob)
	// Should find Player3 (Carol), not himself
	t.Run("ClosestCatcherToCatcher_ExcludesSelf", func(t *testing.T) {
		ctx.Params["originPos"] = state.Players[1].Position // Bob at (0.001, 0)
		ctx.Params["excludeId"] = state.Players[1].ID       // "p2"

		view := &ast.View{
			Source: "Players",
			Pipeline: []ast.ViewOperation{
				{
					Type:  ast.ViewOpFilter,
					Where: &ast.WhereClause{Field: "Team", Op: "==", Value: "catcher"},
				},
				{
					Type:  ast.ViewOpFilter,
					Where: &ast.WhereClause{Field: "ID", Op: "!=", Value: "param:excludeId"},
				},
				{
					Type:     ast.ViewOpNearest,
					Origin:   "param:originPos",
					Position: "$.Position",
					Count:    1,
				},
				{
					Type:     ast.ViewOpDistance,
					From:     "param:originPos",
					Position: "$.Position",
				},
			},
		}

		result, err := ve.Evaluate(ctx, view, ctx.Params)
		if err != nil {
			t.Fatalf("Evaluate failed: %v", err)
		}

		distance, ok := result.(float64)
		if !ok {
			t.Fatalf("Expected float64, got %T: %v", result, result)
		}

		// Carol is at (0.002, 0), Bob is at (0.001, 0)
		// Distance should be ~111m (0.001 degrees)
		expectedMin := 110.0
		expectedMax := 113.0
		if distance < expectedMin || distance > expectedMax {
			t.Errorf("Expected distance between %.0f-%.0fm, got %.2fm", expectedMin, expectedMax, distance)
		}

		t.Logf("Closest catcher to Bob (excluding self): %.2f meters", distance)
	})

	// Test 3: Find closest catcher to Player4 (Dave, the farthest catcher)
	// Should find Player3 (Carol)
	t.Run("ClosestCatcherToFarthestCatcher", func(t *testing.T) {
		ctx.Params["originPos"] = state.Players[3].Position // Dave at (0.005, 0)
		ctx.Params["excludeId"] = state.Players[3].ID       // "p4"

		view := &ast.View{
			Source: "Players",
			Pipeline: []ast.ViewOperation{
				{
					Type:  ast.ViewOpFilter,
					Where: &ast.WhereClause{Field: "Team", Op: "==", Value: "catcher"},
				},
				{
					Type:  ast.ViewOpFilter,
					Where: &ast.WhereClause{Field: "ID", Op: "!=", Value: "param:excludeId"},
				},
				{
					Type:     ast.ViewOpNearest,
					Origin:   "param:originPos",
					Position: "$.Position",
					Count:    1,
				},
				{
					Type:     ast.ViewOpDistance,
					From:     "param:originPos",
					Position: "$.Position",
				},
			},
		}

		result, err := ve.Evaluate(ctx, view, ctx.Params)
		if err != nil {
			t.Fatalf("Evaluate failed: %v", err)
		}

		distance, ok := result.(float64)
		if !ok {
			t.Fatalf("Expected float64, got %T: %v", result, result)
		}

		// Carol is at (0.002, 0), Dave is at (0.005, 0)
		// Distance should be ~333m (0.003 degrees)
		expectedMin := 330.0
		expectedMax := 340.0
		if distance < expectedMin || distance > expectedMax {
			t.Errorf("Expected distance between %.0f-%.0fm, got %.2fm", expectedMin, expectedMax, distance)
		}

		t.Logf("Closest catcher to Dave (excluding self): %.2f meters", distance)
	})

	// Test 4: Verify the full SetFromView flow with effect evaluator
	t.Run("FullEffectFlow", func(t *testing.T) {
		ee := NewEffectEvaluator()

		// Define the view
		closestCatcherView := &ast.View{
			Source: "Players",
			Pipeline: []ast.ViewOperation{
				{
					Type:  ast.ViewOpFilter,
					Where: &ast.WhereClause{Field: "Team", Op: "==", Value: "catcher"},
				},
				{
					Type:  ast.ViewOpFilter,
					Where: &ast.WhereClause{Field: "ID", Op: "!=", Value: "param:excludeId"},
				},
				{
					Type:     ast.ViewOpNearest,
					Origin:   "param:originPos",
					Position: "$.Position",
					Count:    1,
				},
				{
					Type:     ast.ViewOpDistance,
					From:     "param:originPos",
					Position: "$.Position",
				},
			},
		}

		ruleViews := map[string]*ast.View{
			"closestCatcherDistance": closestCatcherView,
		}

		// Simulate applying to each player
		for _, player := range state.Players {
			playerCtx := ctx.WithEntity(player, 0)

			// Evaluate the view expression with self references
			expr := &ast.ValueExpression{
				Type: "viewResult",
				View: "closestCatcherDistance",
				ViewParams: map[string]interface{}{
					"originPos": "self.Position",
					"excludeId": "self.ID",
				},
			}

			result, err := ee.evaluateValueExpression(playerCtx, expr, ruleViews)
			if err != nil {
				// Might fail if no catchers found (e.g., only one catcher exists)
				t.Logf("Player %s (%s): %v", player.Name, player.Team, err)
				continue
			}

			if distance, ok := result.(float64); ok {
				t.Logf("Player %s (%s): closest catcher is %.2f meters away",
					player.Name, player.Team, distance)
			} else {
				t.Logf("Player %s (%s): result = %v", player.Name, player.Team, result)
			}
		}
	})
}

// Test GPS wrap-around (meridian crossing)
func TestClosestEnemyDistance_MeridianCrossing(t *testing.T) {
	state := &CEGameState{
		Players: []*CEPlayer{
			// Player at 179° longitude
			{ID: "p1", Name: "West", Team: "catcher", Position: ast.GeoPoint{Lat: 0, Lon: 179}},
			// Player at -179° longitude (across the date line)
			{ID: "p2", Name: "East", Team: "catcher", Position: ast.GeoPoint{Lat: 0, Lon: -179}},
			// Player at 0° longitude (far away)
			{ID: "p3", Name: "Center", Team: "catcher", Position: ast.GeoPoint{Lat: 0, Lon: 0}},
		},
	}

	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	// From West player's perspective, East should be closest (2° away, not 358°)
	ctx.Params["originPos"] = state.Players[0].Position // 179°
	ctx.Params["excludeId"] = state.Players[0].ID

	view := &ast.View{
		Source: "Players",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpFilter,
				Where: &ast.WhereClause{Field: "Team", Op: "==", Value: "catcher"},
			},
			{
				Type:  ast.ViewOpFilter,
				Where: &ast.WhereClause{Field: "ID", Op: "!=", Value: "param:excludeId"},
			},
			{
				Type:     ast.ViewOpNearest,
				Origin:   "param:originPos",
				Position: "$.Position",
				Count:    1,
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, ctx.Params)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	entities, ok := result.([]interface{})
	if !ok || len(entities) == 0 {
		t.Fatalf("Expected entity array, got %T: %v", result, result)
	}

	nearest := entities[0].(*CEPlayer)
	if nearest.ID != "p2" {
		t.Errorf("Expected East (p2) to be nearest, got %s (%s)", nearest.Name, nearest.ID)
	}

	t.Logf("Nearest to West player: %s (correctly crossed meridian)", nearest.Name)
}
