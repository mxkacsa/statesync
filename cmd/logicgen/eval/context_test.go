package eval

import (
	"testing"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// TestState is a test state structure
type TestState struct {
	Players []TestPlayer
	Score   int
	Config  TestConfig
}

type TestPlayer struct {
	ID       string
	Name     string
	Health   int
	Position ast.GeoPoint
	Status   string
}

type TestConfig struct {
	MaxPlayers int
	GameMode   string
}

func TestContext_ResolvePath(t *testing.T) {
	state := &TestState{
		Players: []TestPlayer{
			{ID: "p1", Name: "Alice", Health: 100, Position: ast.GeoPoint{Lat: 47.5, Lon: 19.0}, Status: "Active"},
			{ID: "p2", Name: "Bob", Health: 80, Position: ast.GeoPoint{Lat: 47.6, Lon: 19.1}, Status: "Active"},
		},
		Score:  42,
		Config: TestConfig{MaxPlayers: 10, GameMode: "deathmatch"},
	}

	ctx := NewContext(state, 100*time.Millisecond, 1)

	tests := []struct {
		name     string
		path     ast.Path
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "simple field",
			path:     "$.Score",
			expected: 42,
		},
		{
			name:     "nested field",
			path:     "$.Config.MaxPlayers",
			expected: 10,
		},
		{
			name:     "array access",
			path:     "$.Players[0].Name",
			expected: "Alice",
		},
		{
			name:     "second array element",
			path:     "$.Players[1].Health",
			expected: 80,
		},
		{
			name:    "invalid path",
			path:    "$.NonExistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ctx.ResolvePath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestContext_Resolve(t *testing.T) {
	state := &TestState{
		Score: 100,
	}

	ctx := NewContext(state, 100*time.Millisecond, 1)
	ctx.Params["playerId"] = "p1"
	ctx.Views["totalScore"] = 500

	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "literal string",
			value:    "hello",
			expected: "hello",
		},
		{
			name:     "literal number",
			value:    42.5,
			expected: 42.5,
		},
		{
			name:     "state path",
			value:    "$.Score",
			expected: 100,
		},
		{
			name:     "param reference",
			value:    "param:playerId",
			expected: "p1",
		},
		{
			name:     "view reference",
			value:    "view:totalScore",
			expected: 500,
		},
		{
			name:     "const number",
			value:    "const:123",
			expected: int64(123),
		},
		{
			name:     "const bool",
			value:    "const:true",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ctx.Resolve(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestContext_SetPath(t *testing.T) {
	state := &TestState{
		Score: 100,
		Players: []TestPlayer{
			{ID: "p1", Name: "Alice", Health: 100},
		},
	}

	ctx := NewContext(state, 100*time.Millisecond, 1)

	// Set a simple field
	if err := ctx.SetPath("$.Score", 200); err != nil {
		t.Fatalf("SetPath failed: %v", err)
	}

	if state.Score != 200 {
		t.Errorf("expected Score to be 200, got %d", state.Score)
	}
}

func TestContext_WithEntity(t *testing.T) {
	state := &TestState{
		Players: []TestPlayer{
			{ID: "p1", Name: "Alice", Health: 100},
		},
	}

	ctx := NewContext(state, 100*time.Millisecond, 1)
	entityCtx := ctx.WithEntity(state.Players[0], 0)

	// Resolve path from entity context should use current entity
	result, err := entityCtx.ResolvePath("$.Name")
	if err != nil {
		t.Fatalf("ResolvePath failed: %v", err)
	}

	if result != "Alice" {
		t.Errorf("expected 'Alice', got %v", result)
	}
}
