package eval

import (
	"encoding/json"
	"testing"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// =============================================================================
// Realistic Game Example: Tag Game with Team Chat and Hidden Locations
// =============================================================================

// TagGameState represents the full game state
type TagGameState struct {
	Players  []TagPlayer  `json:"players"`
	Chat     []TagMessage `json:"chat"`
	GameInfo TagGameInfo  `json:"gameInfo"`
}

type TagPlayer struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Team     string  `json:"team"` // "catcher" or "runner"
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Score    int     `json:"score"`
	IsTagged bool    `json:"isTagged"`
}

type TagMessage struct {
	ID        string `json:"id"`
	SenderID  string `json:"senderId"`
	Team      string `json:"team"` // which team can see this
	Text      string `json:"text"`
	IsPrivate bool   `json:"isPrivate"`
}

type TagGameInfo struct {
	RoundNumber int  `json:"roundNumber"`
	TimeLeft    int  `json:"timeLeft"`
	IsActive    bool `json:"isActive"`
}

// TestFilter_RealisticGameScenario tests a complete game filtering scenario
func TestFilter_RealisticGameScenario(t *testing.T) {
	// Full game state - what the server knows
	gameState := &TagGameState{
		Players: []TagPlayer{
			{ID: "p1", Name: "Alice", Team: "catcher", Lat: 47.5, Lng: 19.0, Score: 10, IsTagged: false},
			{ID: "p2", Name: "Bob", Team: "runner", Lat: 47.6, Lng: 19.1, Score: 5, IsTagged: false},
			{ID: "p3", Name: "Charlie", Team: "runner", Lat: 47.7, Lng: 19.2, Score: 8, IsTagged: true},
			{ID: "p4", Name: "Diana", Team: "catcher", Lat: 47.8, Lng: 19.3, Score: 15, IsTagged: false},
		},
		Chat: []TagMessage{
			{ID: "m1", SenderID: "p1", Team: "catcher", Text: "I see Bob near the park!", IsPrivate: false},
			{ID: "m2", SenderID: "p2", Team: "runner", Text: "Hide behind the building!", IsPrivate: false},
			{ID: "m3", SenderID: "p3", Team: "runner", Text: "I got tagged :(", IsPrivate: false},
			{ID: "m4", SenderID: "p1", Team: "all", Text: "Good game everyone!", IsPrivate: false},
			{ID: "m5", SenderID: "p4", Team: "catcher", Text: "Charlie is near the fountain", IsPrivate: false},
		},
		GameInfo: TagGameInfo{
			RoundNumber: 3,
			TimeLeft:    120,
			IsActive:    true,
		},
	}

	// Define filter for catcher view - defined as JSON (like from node editor)
	filterJSON := `{
		"name": "CatcherViewFilter",
		"params": [
			{"name": "viewerTeam", "type": "string"}
		],
		"operations": [
			{
				"id": "filterTeamChat",
				"type": "KeepWhere",
				"target": "$.Chat",
				"where": {
					"or": [
						{"field": "Team", "op": "==", "value": "param:viewerTeam"},
						{"field": "Team", "op": "==", "value": "all"}
					]
				}
			},
			{
				"id": "hideEnemyLocations",
				"type": "HideFieldsWhere",
				"target": "$.Players",
				"where": {"field": "Team", "op": "!=", "value": "param:viewerTeam"},
				"fields": ["Lat", "Lng"]
			},
			{
				"id": "hideEnemyNames",
				"type": "ReplaceFieldWhere",
				"target": "$.Players",
				"where": {"field": "Team", "op": "!=", "value": "param:viewerTeam"},
				"field": "Name",
				"value": "Runner"
			}
		]
	}`

	// Parse filter from JSON
	var filter ast.Filter
	if err := json.Unmarshal([]byte(filterJSON), &filter); err != nil {
		t.Fatalf("failed to parse filter JSON: %v", err)
	}

	// Apply filter for catcher viewer
	evaluator := NewFilterEvaluator()
	params := map[string]interface{}{
		"viewerTeam": "catcher",
	}

	result, err := evaluator.Apply(&filter, gameState, params)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filtered := result.(*TagGameState)

	// === Verify Chat Filtering ===
	t.Run("ChatFiltering", func(t *testing.T) {
		// Catchers should only see: catcher team messages + "all" messages
		// m1 (catcher), m4 (all), m5 (catcher) = 3 messages
		if len(filtered.Chat) != 3 {
			t.Errorf("expected 3 chat messages for catcher, got %d", len(filtered.Chat))
			for _, m := range filtered.Chat {
				t.Logf("  - %s: %s (team: %s)", m.ID, m.Text, m.Team)
			}
		}

		// Verify no runner messages leaked
		for _, msg := range filtered.Chat {
			if msg.Team == "runner" {
				t.Errorf("runner message leaked to catcher view: %s", msg.Text)
			}
		}
	})

	// === Verify Location Hiding ===
	t.Run("LocationHiding", func(t *testing.T) {
		for _, p := range filtered.Players {
			if p.Team == "catcher" {
				// Catchers should see their own locations
				if p.Lat == 0 && p.Lng == 0 {
					t.Errorf("catcher %s location should be visible", p.ID)
				}
			} else {
				// Runners should have hidden locations
				if p.Lat != 0 || p.Lng != 0 {
					t.Errorf("runner %s location should be hidden (0,0), got (%f,%f)", p.ID, p.Lat, p.Lng)
				}
			}
		}
	})

	// === Verify Name Hiding ===
	t.Run("NameHiding", func(t *testing.T) {
		for _, p := range filtered.Players {
			if p.Team == "catcher" {
				// Catchers keep their real names
				if p.Name == "Runner" {
					t.Errorf("catcher %s should keep their real name", p.ID)
				}
			} else {
				// Runners should be anonymized
				if p.Name != "Runner" {
					t.Errorf("runner %s should be named 'Runner', got '%s'", p.ID, p.Name)
				}
			}
		}
	})

	// === Verify Other Data Unchanged ===
	t.Run("UnchangedData", func(t *testing.T) {
		// Scores should be visible
		for _, p := range filtered.Players {
			if p.Score == 0 && p.ID == "p1" {
				t.Error("scores should not be hidden")
			}
		}

		// GameInfo should be unchanged
		if filtered.GameInfo.RoundNumber != 3 {
			t.Error("GameInfo should not be modified")
		}
	})

	// Print filtered state for visual verification
	t.Log("\n=== Filtered State for Catcher ===")
	t.Log("Players:")
	for _, p := range filtered.Players {
		t.Logf("  %s: %s (team: %s, loc: %.1f,%.1f, score: %d)",
			p.ID, p.Name, p.Team, p.Lat, p.Lng, p.Score)
	}
	t.Log("Chat:")
	for _, m := range filtered.Chat {
		t.Logf("  [%s] %s", m.Team, m.Text)
	}
}

// TestFilter_RunnerView tests filtering from runner perspective
func TestFilter_RunnerView(t *testing.T) {
	gameState := &TagGameState{
		Players: []TagPlayer{
			{ID: "p1", Name: "Alice", Team: "catcher", Lat: 47.5, Lng: 19.0, Score: 10},
			{ID: "p2", Name: "Bob", Team: "runner", Lat: 47.6, Lng: 19.1, Score: 5},
		},
		Chat: []TagMessage{
			{ID: "m1", Team: "catcher", Text: "Catcher strategy"},
			{ID: "m2", Team: "runner", Text: "Runner strategy"},
			{ID: "m3", Team: "all", Text: "Public message"},
		},
	}

	filter := &ast.Filter{
		Name: "RunnerView",
		Params: []ast.FilterParam{
			{Name: "viewerTeam", Type: "string"},
		},
		Operations: []ast.FilterOperation{
			{
				ID:     "filterChat",
				Type:   ast.FilterOpKeepWhere,
				Target: "$.Chat",
				Where: &ast.WhereClause{
					Or: []*ast.WhereClause{
						{Field: "Team", Op: "==", Value: "param:viewerTeam"},
						{Field: "Team", Op: "==", Value: "all"},
					},
				},
			},
			{
				ID:     "hideCatcherLocations",
				Type:   ast.FilterOpHideFieldsWhere,
				Target: "$.Players",
				Where:  &ast.WhereClause{Field: "Team", Op: "!=", Value: "param:viewerTeam"},
				Fields: []string{"Lat", "Lng"},
			},
		},
	}

	evaluator := NewFilterEvaluator()
	result, err := evaluator.Apply(filter, gameState, map[string]interface{}{
		"viewerTeam": "runner",
	})
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filtered := result.(*TagGameState)

	// Runner should see runner + all messages (2 messages)
	if len(filtered.Chat) != 2 {
		t.Errorf("expected 2 messages for runner, got %d", len(filtered.Chat))
	}

	// Catcher location should be hidden
	for _, p := range filtered.Players {
		if p.Team == "catcher" && (p.Lat != 0 || p.Lng != 0) {
			t.Errorf("catcher location should be hidden for runner view")
		}
		if p.Team == "runner" && p.Lat == 0 {
			t.Errorf("runner should see own location")
		}
	}
}

// TestFilter_SpectatorView tests a spectator who sees everything but no private data
func TestFilter_SpectatorView(t *testing.T) {
	gameState := &TagGameState{
		Players: []TagPlayer{
			{ID: "p1", Name: "Alice", Team: "catcher", Lat: 47.5, Lng: 19.0},
			{ID: "p2", Name: "Bob", Team: "runner", Lat: 47.6, Lng: 19.1},
		},
		Chat: []TagMessage{
			{ID: "m1", Team: "catcher", Text: "Team secret"},
			{ID: "m2", Team: "all", Text: "Public"},
		},
	}

	// Spectator filter: only see "all" team messages, but see all locations
	filter := &ast.Filter{
		Name: "SpectatorView",
		Operations: []ast.FilterOperation{
			{
				ID:     "publicChatOnly",
				Type:   ast.FilterOpKeepWhere,
				Target: "$.Chat",
				Where:  &ast.WhereClause{Field: "Team", Op: "==", Value: "all"},
			},
			// Spectators can see all player locations (no hiding needed)
		},
	}

	evaluator := NewFilterEvaluator()
	result, err := evaluator.Apply(filter, gameState, nil)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filtered := result.(*TagGameState)

	// Only public messages
	if len(filtered.Chat) != 1 {
		t.Errorf("spectator should only see 1 public message, got %d", len(filtered.Chat))
	}

	// But all player locations visible
	for _, p := range filtered.Players {
		if p.Lat == 0 {
			t.Errorf("spectator should see all locations, but %s has 0", p.ID)
		}
	}
}

// TestFilter_CatcherSeesRunnersAsUnknown demonstrates the specific use case:
// Catchers see all runner names replaced with "Unknown Runner"
func TestFilter_CatcherSeesRunnersAsUnknown(t *testing.T) {
	// Game state with 4 players
	state := &TagGameState{
		Players: []TagPlayer{
			{ID: "p1", Name: "Alice", Team: "catcher", Lat: 47.5, Lng: 19.0},
			{ID: "p2", Name: "Bob", Team: "runner", Lat: 47.6, Lng: 19.1},
			{ID: "p3", Name: "Charlie", Team: "runner", Lat: 47.7, Lng: 19.2},
			{ID: "p4", Name: "Diana", Team: "catcher", Lat: 47.8, Lng: 19.3},
		},
	}

	// Filter: replace enemy team names with "Unknown Runner"
	filter := &ast.Filter{
		Name: "HideEnemyNames",
		Params: []ast.FilterParam{
			{Name: "viewerTeam", Type: "string"},
		},
		Operations: []ast.FilterOperation{
			{
				ID:     "replaceEnemyNames",
				Type:   ast.FilterOpReplaceFieldWhere,
				Target: "$.Players",
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "!=",
					Value: "param:viewerTeam",
				},
				Field: "Name",
				Value: "Unknown Runner",
			},
		},
	}

	evaluator := NewFilterEvaluator()

	// Apply as catcher
	result, err := evaluator.Apply(filter, state, map[string]interface{}{
		"viewerTeam": "catcher",
	})
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filtered := result.(*TagGameState)

	t.Log("=== Catcher's View ===")
	for _, p := range filtered.Players {
		t.Logf("  %s: Name=%q Team=%s", p.ID, p.Name, p.Team)
	}

	// Verify
	for _, p := range filtered.Players {
		switch p.Team {
		case "catcher":
			// Catchers keep their real names
			if p.Name == "Unknown Runner" {
				t.Errorf("FAIL: Catcher %s should keep real name, got %q", p.ID, p.Name)
			} else {
				t.Logf("OK: Catcher %s keeps name %q", p.ID, p.Name)
			}
		case "runner":
			// Runners should be anonymized
			if p.Name != "Unknown Runner" {
				t.Errorf("FAIL: Runner %s should be 'Unknown Runner', got %q", p.ID, p.Name)
			} else {
				t.Logf("OK: Runner %s hidden as %q", p.ID, p.Name)
			}
		}
	}
}

// TestFilter_RunnerSeesAllNames shows that runners see catcher names normally
// (symmetric test - both teams can hide each other)
func TestFilter_RunnerSeesAllNames(t *testing.T) {
	state := &TagGameState{
		Players: []TagPlayer{
			{ID: "p1", Name: "Alice", Team: "catcher"},
			{ID: "p2", Name: "Bob", Team: "runner"},
		},
	}

	// No name hiding filter - runners see everyone
	filter := &ast.Filter{
		Name:       "NoHiding",
		Operations: []ast.FilterOperation{
			// No operations - all data visible
		},
	}

	evaluator := NewFilterEvaluator()
	result, err := evaluator.Apply(filter, state, nil)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filtered := result.(*TagGameState)

	// All names should be visible
	for _, p := range filtered.Players {
		if p.Name == "" || p.Name == "Unknown Runner" {
			t.Errorf("Player %s should have visible name, got %q", p.ID, p.Name)
		}
	}
}
