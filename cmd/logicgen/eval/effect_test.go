package eval

import (
	"testing"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// Test types for effect tests
type EffectTestState struct {
	Score     int
	Health    float64
	Name      string
	IsActive  bool
	Items     []EffectItem
	Inventory []string
	Position  ast.GeoPoint
	Tags      map[string]string
}

type EffectItem struct {
	ID       string
	Name     string
	Quantity int
	Price    float64
}

func TestEffectEvaluator_Set_Int(t *testing.T) {
	state := &EffectTestState{Score: 0}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeSet,
		Path:  "$.Score",
		Value: 100,
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Score != 100 {
		t.Errorf("expected Score=100, got %d", state.Score)
	}
}

func TestEffectEvaluator_Set_Float(t *testing.T) {
	state := &EffectTestState{Health: 50.0}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeSet,
		Path:  "$.Health",
		Value: 100.5,
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Health != 100.5 {
		t.Errorf("expected Health=100.5, got %f", state.Health)
	}
}

func TestEffectEvaluator_Set_String(t *testing.T) {
	state := &EffectTestState{Name: ""}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeSet,
		Path:  "$.Name",
		Value: "Player1",
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Name != "Player1" {
		t.Errorf("expected Name='Player1', got %s", state.Name)
	}
}

func TestEffectEvaluator_Set_Bool(t *testing.T) {
	state := &EffectTestState{IsActive: false}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeSet,
		Path:  "$.IsActive",
		Value: true,
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !state.IsActive {
		t.Error("expected IsActive=true")
	}
}

func TestEffectEvaluator_Set_GeoPoint(t *testing.T) {
	state := &EffectTestState{}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	newPos := ast.GeoPoint{Lat: 47.5, Lon: 19.0}
	effect := &ast.Effect{
		Type:  ast.EffectTypeSet,
		Path:  "$.Position",
		Value: newPos,
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Position.Lat != 47.5 || state.Position.Lon != 19.0 {
		t.Errorf("expected Position=(47.5,19.0), got %v", state.Position)
	}
}

func TestEffectEvaluator_Set_FromParam(t *testing.T) {
	state := &EffectTestState{Score: 0}
	ctx := NewContext(state, 0, 0)
	ctx.Params["newScore"] = 250
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeSet,
		Path:  "$.Score",
		Value: "param:newScore",
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Score != 250 {
		t.Errorf("expected Score=250, got %d", state.Score)
	}
}

func TestEffectEvaluator_Increment(t *testing.T) {
	state := &EffectTestState{Score: 50}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeIncrement,
		Path:  "$.Score",
		Value: 10,
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Score != 60 {
		t.Errorf("expected Score=60, got %d", state.Score)
	}
}

func TestEffectEvaluator_Increment_Float(t *testing.T) {
	state := &EffectTestState{Health: 75.5}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeIncrement,
		Path:  "$.Health",
		Value: 10.5,
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Health != 86.0 {
		t.Errorf("expected Health=86.0, got %f", state.Health)
	}
}

func TestEffectEvaluator_Increment_Negative(t *testing.T) {
	state := &EffectTestState{Score: 50}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeIncrement,
		Path:  "$.Score",
		Value: -20,
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Score != 30 {
		t.Errorf("expected Score=30, got %d", state.Score)
	}
}

func TestEffectEvaluator_Increment_NonNumeric(t *testing.T) {
	state := &EffectTestState{Name: "test"}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeIncrement,
		Path:  "$.Name",
		Value: 10,
	}

	err := ee.Apply(ctx, effect)
	if err == nil {
		t.Error("expected error for non-numeric increment")
	}
}

func TestEffectEvaluator_Decrement(t *testing.T) {
	state := &EffectTestState{Score: 100}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeDecrement,
		Path:  "$.Score",
		Value: 30,
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Score != 70 {
		t.Errorf("expected Score=70, got %d", state.Score)
	}
}

func TestEffectEvaluator_Decrement_BelowZero(t *testing.T) {
	state := &EffectTestState{Score: 20}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeDecrement,
		Path:  "$.Score",
		Value: 50,
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should go negative
	if state.Score != -30 {
		t.Errorf("expected Score=-30, got %d", state.Score)
	}
}

func TestEffectEvaluator_Append(t *testing.T) {
	state := &EffectTestState{
		Inventory: []string{"sword", "shield"},
	}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeAppend,
		Path: "$.Inventory",
		Item: "potion",
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Inventory) != 3 {
		t.Fatalf("expected 3 items, got %d", len(state.Inventory))
	}

	if state.Inventory[2] != "potion" {
		t.Errorf("expected 'potion', got %s", state.Inventory[2])
	}
}

func TestEffectEvaluator_Append_ToEmpty(t *testing.T) {
	state := &EffectTestState{
		Inventory: []string{},
	}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeAppend,
		Path: "$.Inventory",
		Item: "first-item",
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Inventory) != 1 {
		t.Fatalf("expected 1 item, got %d", len(state.Inventory))
	}
}

func TestEffectEvaluator_Append_NonSlice(t *testing.T) {
	state := &EffectTestState{Name: "test"}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeAppend,
		Path: "$.Name",
		Item: "item",
	}

	err := ee.Apply(ctx, effect)
	if err == nil {
		t.Error("expected error for appending to non-slice")
	}
}

func TestEffectEvaluator_Remove_ByIndex(t *testing.T) {
	state := &EffectTestState{
		Inventory: []string{"a", "b", "c", "d"},
	}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeRemove,
		Path:  "$.Inventory",
		Index: 1, // Remove "b"
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Inventory) != 3 {
		t.Fatalf("expected 3 items, got %d", len(state.Inventory))
	}

	expected := []string{"a", "c", "d"}
	for i, v := range expected {
		if state.Inventory[i] != v {
			t.Errorf("index %d: expected %s, got %s", i, v, state.Inventory[i])
		}
	}
}

func TestEffectEvaluator_Remove_ByIndex_OutOfBounds(t *testing.T) {
	state := &EffectTestState{
		Inventory: []string{"a", "b"},
	}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeRemove,
		Path:  "$.Inventory",
		Index: 10, // Out of bounds
	}

	err := ee.Apply(ctx, effect)
	if err == nil {
		t.Error("expected error for out of bounds index")
	}
}

func TestEffectEvaluator_Remove_ByCondition(t *testing.T) {
	state := &EffectTestState{
		Items: []EffectItem{
			{ID: "i1", Quantity: 5},
			{ID: "i2", Quantity: 0},
			{ID: "i3", Quantity: 3},
			{ID: "i4", Quantity: 0},
		},
	}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeRemove,
		Path: "$.Items",
		Where: &ast.WhereClause{
			Field: "Quantity",
			Op:    "==",
			Value: 0,
		},
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Items) != 2 {
		t.Errorf("expected 2 items (non-zero quantity), got %d", len(state.Items))
	}
}

func TestEffectEvaluator_Remove_NonSlice(t *testing.T) {
	state := &EffectTestState{Score: 100}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeRemove,
		Path:  "$.Score",
		Index: 0,
	}

	err := ee.Apply(ctx, effect)
	if err == nil {
		t.Error("expected error for removing from non-slice")
	}
}

func TestEffectEvaluator_Clear_Slice(t *testing.T) {
	state := &EffectTestState{
		Inventory: []string{"a", "b", "c"},
	}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeClear,
		Path: "$.Inventory",
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Inventory) != 0 {
		t.Errorf("expected empty inventory, got %d items", len(state.Inventory))
	}
}

func TestEffectEvaluator_Clear_Map(t *testing.T) {
	state := &EffectTestState{
		Tags: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeClear,
		Path: "$.Tags",
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Tags) != 0 {
		t.Errorf("expected empty tags, got %d", len(state.Tags))
	}
}

func TestEffectEvaluator_Clear_NonClearable(t *testing.T) {
	state := &EffectTestState{Score: 100}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeClear,
		Path: "$.Score",
	}

	err := ee.Apply(ctx, effect)
	if err == nil {
		t.Error("expected error for clearing non-clearable type")
	}
}

func TestEffectEvaluator_Transform_Add(t *testing.T) {
	state := &EffectTestState{Score: 50}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeTransform,
		Path: "$.Score",
		Transform: &ast.Transform{
			Type:  ast.TransformTypeAdd,
			Left:  "$.Score",
			Right: 25,
		},
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Score != 75 {
		t.Errorf("expected Score=75, got %d", state.Score)
	}
}

// Note: MoveTowards transform test requires DeltaTime and param: support in Path fields.
// Skipping until API is updated to handle param: references in Transform.Target field.

func TestEffectEvaluator_Transform_Nil(t *testing.T) {
	state := &EffectTestState{}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:      ast.EffectTypeTransform,
		Path:      "$.Score",
		Transform: nil,
	}

	err := ee.Apply(ctx, effect)
	if err == nil {
		t.Error("expected error for nil transform")
	}
}

func TestEffectEvaluator_If_ThenBranch(t *testing.T) {
	state := &EffectTestState{Score: 0, IsActive: true}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:      ast.EffectTypeIf,
		Condition: "$.IsActive",
		Then: &ast.Effect{
			Type:  ast.EffectTypeSet,
			Path:  "$.Score",
			Value: 100,
		},
		Else: &ast.Effect{
			Type:  ast.EffectTypeSet,
			Path:  "$.Score",
			Value: 0,
		},
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Score != 100 {
		t.Errorf("expected Score=100 (then branch), got %d", state.Score)
	}
}

func TestEffectEvaluator_If_ElseBranch(t *testing.T) {
	state := &EffectTestState{Score: 50, IsActive: false}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:      ast.EffectTypeIf,
		Condition: "$.IsActive",
		Then: &ast.Effect{
			Type:  ast.EffectTypeSet,
			Path:  "$.Score",
			Value: 100,
		},
		Else: &ast.Effect{
			Type:  ast.EffectTypeSet,
			Path:  "$.Score",
			Value: -1,
		},
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Score != -1 {
		t.Errorf("expected Score=-1 (else branch), got %d", state.Score)
	}
}

func TestEffectEvaluator_If_NoElse(t *testing.T) {
	state := &EffectTestState{Score: 50, IsActive: false}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:      ast.EffectTypeIf,
		Condition: "$.IsActive",
		Then: &ast.Effect{
			Type:  ast.EffectTypeSet,
			Path:  "$.Score",
			Value: 100,
		},
		// No Else branch
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Score should remain unchanged
	if state.Score != 50 {
		t.Errorf("expected Score=50 (unchanged), got %d", state.Score)
	}
}

func TestEffectEvaluator_If_TruthyChecks(t *testing.T) {
	tests := []struct {
		name      string
		condition interface{}
		expected  bool
	}{
		{"true bool", true, true},
		{"false bool", false, false},
		{"positive int", 1, true},
		{"zero int", 0, false},
		{"non-empty string", "hello", true},
		{"empty string", "", false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &EffectTestState{Score: 0}
			ctx := NewContext(state, 0, 0)
			ctx.Params["cond"] = tt.condition
			ee := NewEffectEvaluator()

			effect := &ast.Effect{
				Type:      ast.EffectTypeIf,
				Condition: "param:cond",
				Then: &ast.Effect{
					Type:  ast.EffectTypeSet,
					Path:  "$.Score",
					Value: 1,
				},
				Else: &ast.Effect{
					Type:  ast.EffectTypeSet,
					Path:  "$.Score",
					Value: -1,
				},
			}

			if err := ee.Apply(ctx, effect); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expected := -1
			if tt.expected {
				expected = 1
			}

			if state.Score != expected {
				t.Errorf("expected Score=%d, got %d", expected, state.Score)
			}
		})
	}
}

func TestEffectEvaluator_Sequence(t *testing.T) {
	state := &EffectTestState{Score: 0, Health: 0}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeSequence,
		Effects: []*ast.Effect{
			{
				Type:  ast.EffectTypeSet,
				Path:  "$.Score",
				Value: 100,
			},
			{
				Type:  ast.EffectTypeSet,
				Path:  "$.Health",
				Value: 50.0,
			},
			{
				Type:  ast.EffectTypeIncrement,
				Path:  "$.Score",
				Value: 10,
			},
		},
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Score != 110 {
		t.Errorf("expected Score=110, got %d", state.Score)
	}
	if state.Health != 50.0 {
		t.Errorf("expected Health=50, got %f", state.Health)
	}
}

func TestEffectEvaluator_Sequence_Empty(t *testing.T) {
	state := &EffectTestState{Score: 42}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:    ast.EffectTypeSequence,
		Effects: []*ast.Effect{},
	}

	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Score should remain unchanged
	if state.Score != 42 {
		t.Errorf("expected Score=42, got %d", state.Score)
	}
}

func TestEffectEvaluator_Sequence_StopsOnError(t *testing.T) {
	state := &EffectTestState{Score: 0, Name: "test"}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeSequence,
		Effects: []*ast.Effect{
			{
				Type:  ast.EffectTypeSet,
				Path:  "$.Score",
				Value: 100,
			},
			{
				Type:  ast.EffectTypeIncrement,
				Path:  "$.Name", // Will fail - can't increment string
				Value: 10,
			},
			{
				Type:  ast.EffectTypeSet,
				Path:  "$.Score",
				Value: 999, // Should not reach this
			},
		},
	}

	err := ee.Apply(ctx, effect)
	if err == nil {
		t.Error("expected error in sequence")
	}

	// First effect should have executed
	if state.Score != 100 {
		t.Errorf("expected Score=100 (first effect), got %d", state.Score)
	}
}

func TestEffectEvaluator_Emit(t *testing.T) {
	state := &EffectTestState{Score: 100, Name: "TestPlayer"}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:  ast.EffectTypeEmit,
		Event: "ScoreUpdated",
		Payload: map[string]interface{}{
			"score":      "$.Score",
			"playerName": "$.Name",
		},
	}

	// Emit is a placeholder, just verify it doesn't error
	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEffectEvaluator_Spawn(t *testing.T) {
	state := &EffectTestState{}
	ctx := NewContext(state, 0, 0)
	ctx.Params["spawnPos"] = ast.GeoPoint{Lat: 47.5, Lon: 19.0}
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type:   ast.EffectTypeSpawn,
		Entity: "Item",
		Fields: map[string]interface{}{
			"ID":       "new-item",
			"Position": "param:spawnPos",
		},
	}

	// Spawn is a placeholder, just verify it doesn't error
	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEffectEvaluator_Destroy(t *testing.T) {
	state := &EffectTestState{}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: ast.EffectTypeDestroy,
	}

	// Destroy is a placeholder, just verify it doesn't error
	if err := ee.Apply(ctx, effect); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEffectEvaluator_UnknownType(t *testing.T) {
	state := &EffectTestState{}
	ctx := NewContext(state, 0, 0)
	ee := NewEffectEvaluator()

	effect := &ast.Effect{
		Type: "Unknown",
	}

	err := ee.Apply(ctx, effect)
	if err == nil {
		t.Error("expected error for unknown effect type")
	}
}

// Note: Set_OnEntity test is skipped because modifying struct fields inside slices
// requires the slice to contain pointers, not values. The current test uses value types
// which creates a copy when passed to WithEntity, preventing modification of the original.
