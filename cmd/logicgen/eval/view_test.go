package eval

import (
	"testing"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// Test types for view tests
type ViewTestState struct {
	Items   []ViewItem
	Players []ViewPlayer
	FromPos ast.GeoPoint
	ToPos   ast.GeoPoint
}

type ViewItem struct {
	ID       string
	Name     string
	Price    float64
	Quantity int
	Category string
	Position ast.GeoPoint
}

type ViewPlayer struct {
	ID    string
	Name  string
	Score int
	Team  string
	Level int
}

func TestViewEvaluator_Filter(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Category: "Weapon", Price: 100},
			{ID: "i2", Category: "Armor", Price: 50},
			{ID: "i3", Category: "Weapon", Price: 150},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpFilter,
				Where: &ast.WhereClause{
					Field: "Category",
					Op:    "==",
					Value: "Weapon",
				},
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 weapons, got %d", len(items))
	}
}

func TestViewEvaluator_Filter_Comparison(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Price: 50},
			{ID: "i2", Price: 100},
			{ID: "i3", Price: 150},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpFilter,
				Where: &ast.WhereClause{
					Field: "Price",
					Op:    ">",
					Value: 75.0,
				},
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items with Price > 75, got %d", len(items))
	}
}

func TestViewEvaluator_Max(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Price: 50.0},
			{ID: "i2", Price: 150.0},
			{ID: "i3", Price: 75.0},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpMax,
				Field: "Price",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 150.0 {
		t.Errorf("expected 150.0, got %v", result)
	}
}

func TestViewEvaluator_Max_ReturnEntity(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Price: 50.0},
			{ID: "i2", Price: 150.0},
			{ID: "i3", Price: 75.0},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:   ast.ViewOpMax,
				Field:  "Price",
				Return: "entity",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := result.(ViewItem)
	if item.ID != "i2" {
		t.Errorf("expected i2, got %s", item.ID)
	}
}

func TestViewEvaluator_Max_Empty(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpMax,
				Field: "Price",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for empty entities, got %v", result)
	}
}

func TestViewEvaluator_Min(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Price: 50.0},
			{ID: "i2", Price: 150.0},
			{ID: "i3", Price: 25.0},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpMin,
				Field: "Price",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 25.0 {
		t.Errorf("expected 25.0, got %v", result)
	}
}

func TestViewEvaluator_Min_ReturnEntity(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Price: 50.0},
			{ID: "i2", Price: 150.0},
			{ID: "i3", Price: 25.0},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:   ast.ViewOpMin,
				Field:  "Price",
				Return: "entity",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := result.(ViewItem)
	if item.ID != "i3" {
		t.Errorf("expected i3, got %s", item.ID)
	}
}

func TestViewEvaluator_Sum(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Price: 50.0},
			{ID: "i2", Price: 30.0},
			{ID: "i3", Price: 20.0},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpSum,
				Field: "Price",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 100.0 {
		t.Errorf("expected 100.0, got %v", result)
	}
}

func TestViewEvaluator_Sum_IntegerField(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Quantity: 5},
			{ID: "i2", Quantity: 10},
			{ID: "i3", Quantity: 3},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpSum,
				Field: "Quantity",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 18.0 {
		t.Errorf("expected 18.0, got %v", result)
	}
}

func TestViewEvaluator_Sum_Empty(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpSum,
				Field: "Price",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 0.0 {
		t.Errorf("expected 0.0, got %v", result)
	}
}

func TestViewEvaluator_Count(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1"},
			{ID: "i2"},
			{ID: "i3"},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpCount,
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 3 {
		t.Errorf("expected 3, got %v", result)
	}
}

func TestViewEvaluator_FilterThenCount(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Category: "Weapon"},
			{ID: "i2", Category: "Armor"},
			{ID: "i3", Category: "Weapon"},
			{ID: "i4", Category: "Potion"},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpFilter,
				Where: &ast.WhereClause{
					Field: "Category",
					Op:    "==",
					Value: "Weapon",
				},
			},
			{
				Type: ast.ViewOpCount,
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 2 {
		t.Errorf("expected 2 weapons, got %v", result)
	}
}

func TestViewEvaluator_Avg(t *testing.T) {
	state := &ViewTestState{
		Players: []ViewPlayer{
			{ID: "p1", Score: 100},
			{ID: "p2", Score: 200},
			{ID: "p3", Score: 300},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Players",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpAvg,
				Field: "Score",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 200.0 {
		t.Errorf("expected 200.0, got %v", result)
	}
}

func TestViewEvaluator_Avg_Empty(t *testing.T) {
	state := &ViewTestState{
		Players: []ViewPlayer{},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Players",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpAvg,
				Field: "Score",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 0.0 {
		t.Errorf("expected 0.0, got %v", result)
	}
}

func TestViewEvaluator_First(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Name: "First"},
			{ID: "i2", Name: "Second"},
			{ID: "i3", Name: "Third"},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpFirst,
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := result.(ViewItem)
	if item.Name != "First" {
		t.Errorf("expected 'First', got %s", item.Name)
	}
}

func TestViewEvaluator_First_Empty(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpFirst,
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for empty, got %v", result)
	}
}

func TestViewEvaluator_Last(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Name: "First"},
			{ID: "i2", Name: "Second"},
			{ID: "i3", Name: "Third"},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpLast,
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := result.(ViewItem)
	if item.Name != "Third" {
		t.Errorf("expected 'Third', got %s", item.Name)
	}
}

func TestViewEvaluator_Last_Empty(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpLast,
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for empty, got %v", result)
	}
}

func TestViewEvaluator_GroupBy(t *testing.T) {
	state := &ViewTestState{
		Players: []ViewPlayer{
			{ID: "p1", Name: "Alice", Team: "Red"},
			{ID: "p2", Name: "Bob", Team: "Blue"},
			{ID: "p3", Name: "Charlie", Team: "Red"},
			{ID: "p4", Name: "Diana", Team: "Blue"},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Players",
		Pipeline: []ast.ViewOperation{
			{
				Type:       ast.ViewOpGroupBy,
				GroupField: "Team",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	groups := result.(map[interface{}][]interface{})
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}

	if len(groups["Red"]) != 2 {
		t.Errorf("expected 2 players in Red team, got %d", len(groups["Red"]))
	}
	if len(groups["Blue"]) != 2 {
		t.Errorf("expected 2 players in Blue team, got %d", len(groups["Blue"]))
	}
}

func TestViewEvaluator_Distinct(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Category: "Weapon"},
			{ID: "i2", Category: "Armor"},
			{ID: "i3", Category: "Weapon"},
			{ID: "i4", Category: "Potion"},
			{ID: "i5", Category: "Armor"},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpDistinct,
				Field: "Category",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	categories := result.([]interface{})
	if len(categories) != 3 {
		t.Errorf("expected 3 distinct categories, got %d", len(categories))
	}
}

func TestViewEvaluator_Map(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Name: "Sword", Price: 100},
			{ID: "i2", Name: "Shield", Price: 75},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpMap,
				Fields: map[string]interface{}{
					"itemName":  "$.Name",
					"itemPrice": "$.Price",
				},
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mapped := result.([]map[string]interface{})
	if len(mapped) != 2 {
		t.Fatalf("expected 2 mapped items, got %d", len(mapped))
	}

	if mapped[0]["itemName"] != "Sword" {
		t.Errorf("expected 'Sword', got %v", mapped[0]["itemName"])
	}
	if mapped[1]["itemPrice"] != 75.0 {
		t.Errorf("expected 75.0, got %v (type: %T)", mapped[1]["itemPrice"], mapped[1]["itemPrice"])
	}
}

func TestViewEvaluator_OrderBy_Ascending(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Price: 100},
			{ID: "i2", Price: 25},
			{ID: "i3", Price: 75},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpOrderBy,
				By:   "$.Price",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted := result.([]interface{})
	if len(sorted) != 3 {
		t.Fatalf("expected 3 items, got %d", len(sorted))
	}

	first := sorted[0].(ViewItem)
	if first.Price != 25 {
		t.Errorf("expected first price 25, got %v", first.Price)
	}

	last := sorted[2].(ViewItem)
	if last.Price != 100 {
		t.Errorf("expected last price 100, got %v", last.Price)
	}
}

func TestViewEvaluator_OrderBy_Descending(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Price: 100},
			{ID: "i2", Price: 25},
			{ID: "i3", Price: 75},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpOrderBy,
				By:    "$.Price",
				Order: "desc",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted := result.([]interface{})
	first := sorted[0].(ViewItem)
	if first.Price != 100 {
		t.Errorf("expected first price 100, got %v", first.Price)
	}
}

func TestViewEvaluator_Limit(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1"},
			{ID: "i2"},
			{ID: "i3"},
			{ID: "i4"},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpLimit,
				Count: 2,
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items (limited), got %d", len(items))
	}
}

func TestViewEvaluator_OrderByString(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Name: "Zebra"},
			{ID: "i2", Name: "Apple"},
			{ID: "i3", Name: "Mango"},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpOrderBy,
				By:   "$.Name",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted := result.([]interface{})
	first := sorted[0].(ViewItem)
	if first.Name != "Apple" {
		t.Errorf("expected first name 'Apple', got %s", first.Name)
	}
}

func TestViewEvaluator_OrderBySingle(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Price: 100},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpOrderBy,
				By:   "$.Price",
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted := result.([]interface{})
	if len(sorted) != 1 {
		t.Errorf("expected 1 item, got %d", len(sorted))
	}
}

func TestViewEvaluator_Pipeline_FilterOrderByLimit(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Category: "Weapon", Price: 100},
			{ID: "i2", Category: "Armor", Price: 50},
			{ID: "i3", Category: "Weapon", Price: 200},
			{ID: "i4", Category: "Weapon", Price: 150},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	// Filter weapons, order by price desc, take top 2
	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: ast.ViewOpFilter,
				Where: &ast.WhereClause{
					Field: "Category",
					Op:    "==",
					Value: "Weapon",
				},
			},
			{
				Type:  ast.ViewOpOrderBy,
				By:    "$.Price",
				Order: "desc",
			},
			{
				Type:  ast.ViewOpLimit,
				Count: 2,
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := result.([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	first := items[0].(ViewItem)
	if first.Price != 200 {
		t.Errorf("expected first price 200 (most expensive weapon), got %v", first.Price)
	}

	second := items[1].(ViewItem)
	if second.Price != 150 {
		t.Errorf("expected second price 150, got %v", second.Price)
	}
}

func TestViewEvaluator_UnknownOperation(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{{ID: "i1"}},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type: "Unknown",
			},
		},
	}

	_, err := ve.Evaluate(ctx, view, nil)
	if err == nil {
		t.Error("expected error for unknown operation type")
	}
}

func TestViewEvaluator_MaxMin_NonNumericField(t *testing.T) {
	state := &ViewTestState{
		Items: []ViewItem{
			{ID: "i1", Name: "Sword"},
			{ID: "i2", Name: "Shield"},
		},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Source: "Items",
		Pipeline: []ast.ViewOperation{
			{
				Type:  ast.ViewOpMax,
				Field: "Name", // String field
			},
		},
	}

	result, err := ve.Evaluate(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return nil because strings aren't numeric
	if result != nil {
		t.Errorf("expected nil for non-numeric field, got %v", result)
	}
}
