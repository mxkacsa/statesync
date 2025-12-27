package eval

import (
	"testing"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// =============================================================================
// Nested Schema Test: Player with Cards
// =============================================================================

type Card struct {
	ID    string
	Value int
	Suit  string
}

type CardPlayer struct {
	ID    string
	Name  string
	Team  string
	Cards []Card
}

type CardGameState struct {
	Players []CardPlayer
}

// TestView_NestedMax tests finding the highest card among catcher players
func TestView_NestedMax(t *testing.T) {
	state := &CardGameState{
		Players: []CardPlayer{
			{
				ID:   "p1",
				Name: "Alice",
				Team: "catcher",
				Cards: []Card{
					{ID: "c1", Value: 10, Suit: "hearts"},
					{ID: "c2", Value: 5, Suit: "spades"},
				},
			},
			{
				ID:   "p2",
				Name: "Bob",
				Team: "runner",
				Cards: []Card{
					{ID: "c3", Value: 15, Suit: "diamonds"}, // Highest overall, but runner
				},
			},
			{
				ID:   "p3",
				Name: "Charlie",
				Team: "catcher",
				Cards: []Card{
					{ID: "c4", Value: 12, Suit: "clubs"}, // Highest among catchers
					{ID: "c5", Value: 3, Suit: "hearts"},
				},
			},
		},
	}

	// View: Get the highest card value among catcher players
	// Pipeline: Filter(Team==catcher) -> FlatMap(Cards) -> Max(Value)
	view := &ast.View{
		Name:   "HighestCatcherCard",
		Source: "Players",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpFilter,
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "==",
					Value: "catcher",
				},
			},
			{
				Type:  ast.ViewOpFlatMap,
				Field: "Cards",
			},
			{
				Type:  ast.ViewOpMax,
				Field: "Value",
			},
		},
	}

	ctx := NewContext(state, time.Second, 1)
	evaluator := NewViewEvaluator()

	result, err := evaluator.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("view error: %v", err)
	}

	maxVal, ok := result.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", result)
	}

	// Highest card among catchers should be 12 (Charlie's card)
	if maxVal != 12 {
		t.Errorf("expected max card value 12, got %v", maxVal)
	}

	t.Logf("Highest catcher card value: %v", maxVal)
}

// TestView_NestedPath tests using nested path syntax Cards[*].Value
func TestView_NestedPathMax(t *testing.T) {
	state := &CardGameState{
		Players: []CardPlayer{
			{
				ID:   "p1",
				Team: "catcher",
				Cards: []Card{
					{Value: 10},
					{Value: 8},
				},
			},
			{
				ID:   "p2",
				Team: "catcher",
				Cards: []Card{
					{Value: 15}, // Highest
				},
			},
		},
	}

	// Alternative: Use nested path with wildcard directly
	// This collects all card values into a flat array
	view := &ast.View{
		Name:   "AllCardValues",
		Source: "Players",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpFlatMap,
				Field: "Cards",
			},
			{
				Type:  ast.ViewOpOrderBy,
				By:    "Value",
				Order: "desc",
			},
			{
				Type: ast.ViewOpFirst,
			},
		},
	}

	ctx := NewContext(state, time.Second, 1)
	evaluator := NewViewEvaluator()

	result, err := evaluator.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("view error: %v", err)
	}

	// Result should be the card with highest value
	card, ok := result.(Card)
	if !ok {
		t.Fatalf("expected Card, got %T: %v", result, result)
	}

	if card.Value != 15 {
		t.Errorf("expected card value 15, got %d", card.Value)
	}

	t.Logf("Highest card: %+v", card)
}

// TestView_SumNestedValues tests summing all card values for a team
func TestView_SumNestedValues(t *testing.T) {
	state := &CardGameState{
		Players: []CardPlayer{
			{
				Team: "catcher",
				Cards: []Card{
					{Value: 10},
					{Value: 5},
				},
			},
			{
				Team: "catcher",
				Cards: []Card{
					{Value: 7},
				},
			},
		},
	}

	// Sum all card values: Filter -> FlatMap -> Sum
	view := &ast.View{
		Name:   "TotalCatcherCardValue",
		Source: "Players",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpFilter,
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "==",
					Value: "catcher",
				},
			},
			{
				Type:  ast.ViewOpFlatMap,
				Field: "Cards",
			},
			{
				Type:  ast.ViewOpSum,
				Field: "Value",
			},
		},
	}

	ctx := NewContext(state, time.Second, 1)
	evaluator := NewViewEvaluator()

	result, err := evaluator.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("view error: %v", err)
	}

	sum, ok := result.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", result)
	}

	// 10 + 5 + 7 = 22
	if sum != 22 {
		t.Errorf("expected sum 22, got %v", sum)
	}

	t.Logf("Total catcher card value: %v", sum)
}

// TestView_CountNestedItems tests counting all cards for a team
func TestView_CountNestedItems(t *testing.T) {
	state := &CardGameState{
		Players: []CardPlayer{
			{
				Team:  "catcher",
				Cards: []Card{{}, {}, {}}, // 3 cards
			},
			{
				Team:  "runner",
				Cards: []Card{{}, {}}, // 2 cards (should be excluded)
			},
			{
				Team:  "catcher",
				Cards: []Card{{}}, // 1 card
			},
		},
	}

	// Count cards for catchers: Filter -> FlatMap -> Count
	view := &ast.View{
		Name:   "CatcherCardCount",
		Source: "Players",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpFilter,
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "==",
					Value: "catcher",
				},
			},
			{
				Type:  ast.ViewOpFlatMap,
				Field: "Cards",
			},
			{
				Type: ast.ViewOpCount,
			},
		},
	}

	ctx := NewContext(state, time.Second, 1)
	evaluator := NewViewEvaluator()

	result, err := evaluator.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("view error: %v", err)
	}

	count, ok := result.(int)
	if !ok {
		t.Fatalf("expected int, got %T", result)
	}

	// 3 + 1 = 4 cards for catchers
	if count != 4 {
		t.Errorf("expected count 4, got %d", count)
	}

	t.Logf("Catcher card count: %d", count)
}

// TestGetFieldValue_NestedPath tests the nested path functionality directly
func TestGetFieldValue_NestedPath(t *testing.T) {
	player := CardPlayer{
		ID:   "p1",
		Name: "Alice",
		Cards: []Card{
			{ID: "c1", Value: 10},
			{ID: "c2", Value: 20},
		},
	}

	// Test nested path with wildcard
	values, err := getFieldValue(player, "Cards[*].Value")
	if err != nil {
		t.Fatalf("nested path error: %v", err)
	}

	valSlice, ok := values.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", values)
	}

	if len(valSlice) != 2 {
		t.Errorf("expected 2 values, got %d", len(valSlice))
	}

	t.Logf("Nested values: %v", valSlice)

	// Test simple nested path (no wildcard)
	cards, err := getFieldValue(player, "Cards")
	if err != nil {
		t.Fatalf("simple path error: %v", err)
	}

	cardSlice, ok := cards.([]Card)
	if !ok {
		t.Fatalf("expected []Card, got %T", cards)
	}

	if len(cardSlice) != 2 {
		t.Errorf("expected 2 cards, got %d", len(cardSlice))
	}
}
