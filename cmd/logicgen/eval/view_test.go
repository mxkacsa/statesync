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

func TestViewEvaluator_Field_Single(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Name: "Sword", Price: 100.0},
	}

	view := &ast.View{
		Type:  ast.ViewTypeField,
		Field: "$.Name",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Sword" {
		t.Errorf("expected 'Sword', got %v", result)
	}
}

func TestViewEvaluator_Field_Multiple(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Name: "Sword"},
		ViewItem{ID: "i2", Name: "Shield"},
		ViewItem{ID: "i3", Name: "Potion"},
	}

	view := &ast.View{
		Type:  ast.ViewTypeField,
		Field: "$.Name",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := result.([]interface{})
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}
	if names[0] != "Sword" || names[1] != "Shield" || names[2] != "Potion" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestViewEvaluator_Field_Empty(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Type:  ast.ViewTypeField,
		Field: "$.Name",
	}

	result, err := ve.Compute(ctx, view, []interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for empty entities, got %v", result)
	}
}

func TestViewEvaluator_Max(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Price: 50.0},
		ViewItem{ID: "i2", Price: 150.0},
		ViewItem{ID: "i3", Price: 75.0},
	}

	view := &ast.View{
		Type:  ast.ViewTypeMax,
		Field: "$.Price",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 150.0 {
		t.Errorf("expected 150.0, got %v", result)
	}
}

func TestViewEvaluator_Max_ReturnEntity(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Price: 50.0},
		ViewItem{ID: "i2", Price: 150.0},
		ViewItem{ID: "i3", Price: 75.0},
	}

	view := &ast.View{
		Type:   ast.ViewTypeMax,
		Field:  "$.Price",
		Return: "entity",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := result.(ViewItem)
	if item.ID != "i2" {
		t.Errorf("expected i2, got %s", item.ID)
	}
}

func TestViewEvaluator_Max_Empty(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Type:  ast.ViewTypeMax,
		Field: "$.Price",
	}

	result, err := ve.Compute(ctx, view, []interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for empty entities, got %v", result)
	}
}

func TestViewEvaluator_Min(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Price: 50.0},
		ViewItem{ID: "i2", Price: 150.0},
		ViewItem{ID: "i3", Price: 25.0},
	}

	view := &ast.View{
		Type:  ast.ViewTypeMin,
		Field: "$.Price",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 25.0 {
		t.Errorf("expected 25.0, got %v", result)
	}
}

func TestViewEvaluator_Min_ReturnEntity(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Price: 50.0},
		ViewItem{ID: "i2", Price: 150.0},
		ViewItem{ID: "i3", Price: 25.0},
	}

	view := &ast.View{
		Type:   ast.ViewTypeMin,
		Field:  "$.Price",
		Return: "entity",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := result.(ViewItem)
	if item.ID != "i3" {
		t.Errorf("expected i3, got %s", item.ID)
	}
}

func TestViewEvaluator_Sum(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Price: 50.0},
		ViewItem{ID: "i2", Price: 30.0},
		ViewItem{ID: "i3", Price: 20.0},
	}

	view := &ast.View{
		Type:  ast.ViewTypeSum,
		Field: "$.Price",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 100.0 {
		t.Errorf("expected 100.0, got %v", result)
	}
}

func TestViewEvaluator_Sum_IntegerField(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Quantity: 5},
		ViewItem{ID: "i2", Quantity: 10},
		ViewItem{ID: "i3", Quantity: 3},
	}

	view := &ast.View{
		Type:  ast.ViewTypeSum,
		Field: "$.Quantity",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 18.0 {
		t.Errorf("expected 18.0, got %v", result)
	}
}

func TestViewEvaluator_Sum_Empty(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Type:  ast.ViewTypeSum,
		Field: "$.Price",
	}

	result, err := ve.Compute(ctx, view, []interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 0.0 {
		t.Errorf("expected 0.0, got %v", result)
	}
}

func TestViewEvaluator_Count(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1"},
		ViewItem{ID: "i2"},
		ViewItem{ID: "i3"},
	}

	view := &ast.View{
		Type: ast.ViewTypeCount,
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 3 {
		t.Errorf("expected 3, got %v", result)
	}
}

func TestViewEvaluator_Count_WithCondition(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Category: "Weapon"},
		ViewItem{ID: "i2", Category: "Armor"},
		ViewItem{ID: "i3", Category: "Weapon"},
		ViewItem{ID: "i4", Category: "Potion"},
	}

	view := &ast.View{
		Type: ast.ViewTypeCount,
		Where: &ast.WhereClause{
			Field: "Category",
			Op:    "==",
			Value: "Weapon",
		},
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 2 {
		t.Errorf("expected 2 weapons, got %v", result)
	}
}

func TestViewEvaluator_Count_WithNotEqual(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewPlayer{ID: "p1", Team: "Red"},
		ViewPlayer{ID: "p2", Team: "Blue"},
		ViewPlayer{ID: "p3", Team: "Red"},
		ViewPlayer{ID: "p4", Team: "Green"},
	}

	view := &ast.View{
		Type: ast.ViewTypeCount,
		Where: &ast.WhereClause{
			Field: "Team",
			Op:    "!=",
			Value: "Red",
		},
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 2 {
		t.Errorf("expected 2 non-red, got %v", result)
	}
}

func TestViewEvaluator_Avg(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewPlayer{ID: "p1", Score: 100},
		ViewPlayer{ID: "p2", Score: 200},
		ViewPlayer{ID: "p3", Score: 300},
	}

	view := &ast.View{
		Type:  ast.ViewTypeAvg,
		Field: "$.Score",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 200.0 {
		t.Errorf("expected 200.0, got %v", result)
	}
}

func TestViewEvaluator_Avg_Empty(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Type:  ast.ViewTypeAvg,
		Field: "$.Score",
	}

	result, err := ve.Compute(ctx, view, []interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != 0.0 {
		t.Errorf("expected 0.0, got %v", result)
	}
}

func TestViewEvaluator_First(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Name: "First"},
		ViewItem{ID: "i2", Name: "Second"},
		ViewItem{ID: "i3", Name: "Third"},
	}

	view := &ast.View{
		Type: ast.ViewTypeFirst,
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := result.(ViewItem)
	if item.Name != "First" {
		t.Errorf("expected 'First', got %s", item.Name)
	}
}

func TestViewEvaluator_First_Empty(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Type: ast.ViewTypeFirst,
	}

	result, err := ve.Compute(ctx, view, []interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for empty, got %v", result)
	}
}

func TestViewEvaluator_Last(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Name: "First"},
		ViewItem{ID: "i2", Name: "Second"},
		ViewItem{ID: "i3", Name: "Third"},
	}

	view := &ast.View{
		Type: ast.ViewTypeLast,
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := result.(ViewItem)
	if item.Name != "Third" {
		t.Errorf("expected 'Third', got %s", item.Name)
	}
}

func TestViewEvaluator_Last_Empty(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Type: ast.ViewTypeLast,
	}

	result, err := ve.Compute(ctx, view, []interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for empty, got %v", result)
	}
}

func TestViewEvaluator_GroupBy(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewPlayer{ID: "p1", Name: "Alice", Team: "Red"},
		ViewPlayer{ID: "p2", Name: "Bob", Team: "Blue"},
		ViewPlayer{ID: "p3", Name: "Charlie", Team: "Red"},
		ViewPlayer{ID: "p4", Name: "Diana", Team: "Blue"},
	}

	view := &ast.View{
		Type:       ast.ViewTypeGroupBy,
		GroupField: "Team",
	}

	result, err := ve.Compute(ctx, view, entities)
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

func TestViewEvaluator_GroupBy_WithAggregate(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewPlayer{ID: "p1", Score: 100, Team: "Red"},
		ViewPlayer{ID: "p2", Score: 200, Team: "Blue"},
		ViewPlayer{ID: "p3", Score: 150, Team: "Red"},
		ViewPlayer{ID: "p4", Score: 300, Team: "Blue"},
	}

	view := &ast.View{
		Type:       ast.ViewTypeGroupBy,
		GroupField: "Team",
		Aggregate: &ast.View{
			Type:  ast.ViewTypeSum,
			Field: "$.Score",
		},
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	groups := result.(map[interface{}]interface{})
	if groups["Red"] != 250.0 {
		t.Errorf("expected Red team sum 250, got %v", groups["Red"])
	}
	if groups["Blue"] != 500.0 {
		t.Errorf("expected Blue team sum 500, got %v", groups["Blue"])
	}
}

func TestViewEvaluator_Distinct(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Category: "Weapon"},
		ViewItem{ID: "i2", Category: "Armor"},
		ViewItem{ID: "i3", Category: "Weapon"},
		ViewItem{ID: "i4", Category: "Potion"},
		ViewItem{ID: "i5", Category: "Armor"},
	}

	view := &ast.View{
		Type:  ast.ViewTypeDistinct,
		Field: "$.Category",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	categories := result.([]interface{})
	if len(categories) != 3 {
		t.Errorf("expected 3 distinct categories, got %d", len(categories))
	}
}

func TestViewEvaluator_Map(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Name: "Sword", Price: 100},
		ViewItem{ID: "i2", Name: "Shield", Price: 75},
	}

	view := &ast.View{
		Type: ast.ViewTypeMap,
		Transform: map[string]interface{}{
			"itemName":  "$.Name",
			"itemPrice": "$.Price",
		},
	}

	result, err := ve.Compute(ctx, view, entities)
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

func TestViewEvaluator_Sort_Ascending(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Price: 100},
		ViewItem{ID: "i2", Price: 25},
		ViewItem{ID: "i3", Price: 75},
	}

	view := &ast.View{
		Type: ast.ViewTypeSort,
		By:   "$.Price",
	}

	result, err := ve.Compute(ctx, view, entities)
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

func TestViewEvaluator_Sort_Descending(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Price: 100},
		ViewItem{ID: "i2", Price: 25},
		ViewItem{ID: "i3", Price: 75},
	}

	view := &ast.View{
		Type:  ast.ViewTypeSort,
		By:    "$.Price",
		Order: "desc",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted := result.([]interface{})
	first := sorted[0].(ViewItem)
	if first.Price != 100 {
		t.Errorf("expected first price 100, got %v", first.Price)
	}
}

func TestViewEvaluator_Sort_WithLimit(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Price: 100},
		ViewItem{ID: "i2", Price: 25},
		ViewItem{ID: "i3", Price: 75},
		ViewItem{ID: "i4", Price: 50},
	}

	view := &ast.View{
		Type:  ast.ViewTypeSort,
		By:    "$.Price",
		Limit: 2,
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted := result.([]interface{})
	if len(sorted) != 2 {
		t.Errorf("expected 2 items (limited), got %d", len(sorted))
	}
}

func TestViewEvaluator_Sort_StringField(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Name: "Zebra"},
		ViewItem{ID: "i2", Name: "Apple"},
		ViewItem{ID: "i3", Name: "Mango"},
	}

	view := &ast.View{
		Type: ast.ViewTypeSort,
		By:   "$.Name",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted := result.([]interface{})
	first := sorted[0].(ViewItem)
	if first.Name != "Apple" {
		t.Errorf("expected first name 'Apple', got %s", first.Name)
	}
}

func TestViewEvaluator_Sort_SingleElement(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Price: 100},
	}

	view := &ast.View{
		Type: ast.ViewTypeSort,
		By:   "$.Price",
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted := result.([]interface{})
	if len(sorted) != 1 {
		t.Errorf("expected 1 item, got %d", len(sorted))
	}
}

func TestViewEvaluator_Distance(t *testing.T) {
	state := &ViewTestState{
		FromPos: ast.GeoPoint{Lat: 47.4979, Lon: 19.0402}, // Budapest
		ToPos:   ast.GeoPoint{Lat: 48.2082, Lon: 16.3738}, // Vienna
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Type: ast.ViewTypeDistance,
		From: "$.FromPos",
		To:   "$.ToPos",
	}

	result, err := ve.Compute(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distance := result.(float64)
	// Budapest to Vienna is approximately 215km
	if distance < 200000 || distance > 250000 {
		t.Errorf("expected distance ~215000 meters, got %v", distance)
	}
}

func TestViewEvaluator_Distance_Kilometers(t *testing.T) {
	state := &ViewTestState{
		FromPos: ast.GeoPoint{Lat: 47.4979, Lon: 19.0402},
		ToPos:   ast.GeoPoint{Lat: 48.2082, Lon: 16.3738},
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Type: ast.ViewTypeDistance,
		From: "$.FromPos",
		To:   "$.ToPos",
		Unit: "km",
	}

	result, err := ve.Compute(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distance := result.(float64)
	if distance < 200 || distance > 250 {
		t.Errorf("expected distance ~215 km, got %v", distance)
	}
}

func TestViewEvaluator_Distance_SamePoint(t *testing.T) {
	point := ast.GeoPoint{Lat: 47.4979, Lon: 19.0402}
	state := &ViewTestState{
		FromPos: point,
		ToPos:   point,
	}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Type: ast.ViewTypeDistance,
		From: "$.FromPos",
		To:   "$.ToPos",
	}

	result, err := ve.Compute(ctx, view, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distance := result.(float64)
	if distance != 0 {
		t.Errorf("expected distance 0 for same point, got %v", distance)
	}
}

func TestViewEvaluator_UnknownType(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	view := &ast.View{
		Type: "Unknown",
	}

	_, err := ve.Compute(ctx, view, []interface{}{})
	if err == nil {
		t.Error("expected error for unknown view type")
	}
}

func TestViewEvaluator_MaxMin_NonNumericField(t *testing.T) {
	state := &ViewTestState{}
	ctx := NewContext(state, 0, 0)
	ve := NewViewEvaluator()

	entities := []interface{}{
		ViewItem{ID: "i1", Name: "Sword"},
		ViewItem{ID: "i2", Name: "Shield"},
	}

	view := &ast.View{
		Type:  ast.ViewTypeMax,
		Field: "$.Name", // String field
	}

	result, err := ve.Compute(ctx, view, entities)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return nil because strings aren't numeric
	if result != nil {
		t.Errorf("expected nil for non-numeric field, got %v", result)
	}
}
