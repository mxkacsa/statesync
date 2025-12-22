package eval

import (
	"testing"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// Test types for selector tests
type SelectorTestState struct {
	Drones     []Drone
	Players    []SelectorPlayer
	Buildings  map[string]Building
	HQ         Building
	ActiveUnit *Drone
}

type Drone struct {
	ID       string
	Name     string
	Status   string
	Health   int
	Position ast.GeoPoint
	OwnerID  string
	Targets  []string
}

type SelectorPlayer struct {
	ID    string
	Name  string
	Score int
	Team  string
}

type Building struct {
	ID       string
	Name     string
	Position ast.GeoPoint
}

func TestSelectorEvaluator_NilSelector(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{{ID: "d1"}},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	// Nil selector should return the state itself
	result, err := se.Select(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
	if result[0] != state {
		t.Error("expected state to be returned for nil selector")
	}
}

func TestSelectorEvaluator_SelectAll(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d1", Name: "Alpha"},
			{ID: "d2", Name: "Beta"},
			{ID: "d3", Name: "Gamma"},
		},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   ast.SelectorTypeAll,
		Entity: "Drones",
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 drones, got %d", len(result))
	}
}

func TestSelectorEvaluator_SelectAll_EmptySlice(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   ast.SelectorTypeAll,
		Entity: "Drones",
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 drones, got %d", len(result))
	}
}

func TestSelectorEvaluator_SelectAll_Map(t *testing.T) {
	state := &SelectorTestState{
		Buildings: map[string]Building{
			"b1": {ID: "b1", Name: "HQ"},
			"b2": {ID: "b2", Name: "Factory"},
		},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   ast.SelectorTypeAll,
		Entity: "Buildings",
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 buildings, got %d", len(result))
	}
}

func TestSelectorEvaluator_SelectAll_SingleEntity(t *testing.T) {
	state := &SelectorTestState{
		HQ: Building{ID: "hq1", Name: "Headquarters"},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   ast.SelectorTypeAll,
		Entity: "HQ",
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 building, got %d", len(result))
	}
}

func TestSelectorEvaluator_SelectAll_UnknownEntity(t *testing.T) {
	state := &SelectorTestState{}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   ast.SelectorTypeAll,
		Entity: "NonExistent",
	}

	_, err := se.Select(ctx, selector)
	if err == nil {
		t.Error("expected error for unknown entity")
	}
}

func TestSelectorEvaluator_Filter_StringEqual(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d1", Status: "Active"},
			{ID: "d2", Status: "Idle"},
			{ID: "d3", Status: "Active"},
			{ID: "d4", Status: "Destroyed"},
		},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   ast.SelectorTypeFilter,
		Entity: "Drones",
		Where: &ast.WhereClause{
			Field: "Status",
			Op:    "==",
			Value: "Active",
		},
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 active drones, got %d", len(result))
	}
}

func TestSelectorEvaluator_Filter_NumericComparison(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d1", Health: 100},
			{ID: "d2", Health: 50},
			{ID: "d3", Health: 75},
			{ID: "d4", Health: 25},
		},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	tests := []struct {
		name     string
		op       string
		value    interface{}
		expected int
	}{
		{"greater than", ">", 50, 2},
		{"greater equal", ">=", 50, 3},
		{"less than", "<", 75, 2},
		{"less equal", "<=", 75, 3},
		{"equal", "==", 100, 1},
		{"not equal", "!=", 100, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := &ast.Selector{
				Type:   ast.SelectorTypeFilter,
				Entity: "Drones",
				Where: &ast.WhereClause{
					Field: "Health",
					Op:    tt.op,
					Value: tt.value,
				},
			}

			result, err := se.Select(ctx, selector)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != tt.expected {
				t.Errorf("expected %d drones, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestSelectorEvaluator_Filter_NoWhere(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d1"},
			{ID: "d2"},
		},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   ast.SelectorTypeFilter,
		Entity: "Drones",
		Where:  nil,
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 drones (all), got %d", len(result))
	}
}

func TestSelectorEvaluator_Filter_NonexistentField(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d1", Status: "Active"},
		},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   ast.SelectorTypeFilter,
		Entity: "Drones",
		Where: &ast.WhereClause{
			Field: "NonExistent",
			Op:    "==",
			Value: "test",
		},
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty - no matches when field doesn't exist
	if len(result) != 0 {
		t.Errorf("expected 0 matches for non-existent field, got %d", len(result))
	}
}

// Note: Single selector tests require ID to be a resolvable path ($.SomeField).
// The current implementation doesn't support param: prefix in Path type fields.
// These tests are skipped until the API is updated to handle this case.

func TestSelectorEvaluator_Related(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d1", Name: "Alpha", OwnerID: "p1"},
			{ID: "d2", Name: "Beta", OwnerID: "p1"},
			{ID: "d3", Name: "Gamma", OwnerID: "p2"},
		},
		Players: []SelectorPlayer{
			{ID: "p1", Name: "Player1"},
			{ID: "p2", Name: "Player2"},
		},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	// Set up a drone as the source entity ($ will refer to CurrentEntity)
	droneCtx := ctx.WithEntity(state.Drones[0], 0)

	selector := &ast.Selector{
		Type:     ast.SelectorTypeRelated,
		Entity:   "Players",
		From:     "$", // $ refers to current entity
		Relation: "OwnerID",
	}

	result, err := se.Select(droneCtx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 related player, got %d", len(result))
	}

	player := result[0].(SelectorPlayer)
	if player.ID != "p1" {
		t.Errorf("expected player p1, got %s", player.ID)
	}
}

func TestSelectorEvaluator_Related_MultipleTargets(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d1", Name: "Alpha", Targets: []string{"d2", "d3"}},
			{ID: "d2", Name: "Beta"},
			{ID: "d3", Name: "Gamma"},
			{ID: "d4", Name: "Delta"},
		},
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	// Set up drone d1 as the source
	droneCtx := ctx.WithEntity(state.Drones[0], 0)

	selector := &ast.Selector{
		Type:     ast.SelectorTypeRelated,
		Entity:   "Drones",
		From:     "$", // $ refers to current entity
		Relation: "Targets",
	}

	result, err := se.Select(droneCtx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 related drones, got %d", len(result))
	}
}

func TestSelectorEvaluator_Related_NoRelationField(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{{ID: "d1"}},
	}
	ctx := NewContext(state, 0, 0).WithEntity(state.Drones[0], 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:     ast.SelectorTypeRelated,
		Entity:   "Drones",
		From:     "$.Entity",
		Relation: "", // Missing relation
	}

	_, err := se.Select(ctx, selector)
	if err == nil {
		t.Error("expected error for missing relation field")
	}
}

func TestSelectorEvaluator_Nearest(t *testing.T) {
	// Create state with an origin reference drone (we'll use first drone's position as origin)
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d1", Position: ast.GeoPoint{Lat: 47.4979, Lon: 19.0402}}, // Same as origin
			{ID: "d2", Position: ast.GeoPoint{Lat: 47.5, Lon: 19.05}},      // ~1km away
			{ID: "d3", Position: ast.GeoPoint{Lat: 47.51, Lon: 19.06}},     // ~2km away
			{ID: "d4", Position: ast.GeoPoint{Lat: 48.0, Lon: 19.5}},       // ~60km away
		},
	}
	ctx := NewContext(state, 0, 0)
	// Set origin in current entity context
	ctx = ctx.WithEntity(state.Drones[0], 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   ast.SelectorTypeNearest,
		Entity: "Drones",
		Origin: "$.Position", // Use current entity's position as origin
		Limit:  2,
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 nearest drones, got %d", len(result))
	}

	// First should be d1 (same position)
	first := result[0].(Drone)
	if first.ID != "d1" {
		t.Errorf("expected d1 as nearest, got %s", first.ID)
	}
}

func TestSelectorEvaluator_Farthest(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d1", Position: ast.GeoPoint{Lat: 47.4979, Lon: 19.0402}}, // Same as origin
			{ID: "d2", Position: ast.GeoPoint{Lat: 47.5, Lon: 19.05}},      // ~1km away
			{ID: "d3", Position: ast.GeoPoint{Lat: 48.0, Lon: 19.5}},       // ~60km away
		},
	}
	ctx := NewContext(state, 0, 0)
	ctx = ctx.WithEntity(state.Drones[0], 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   ast.SelectorTypeFarthest,
		Entity: "Drones",
		Origin: "$.Position",
		Limit:  1,
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 farthest drone, got %d", len(result))
	}

	first := result[0].(Drone)
	if first.ID != "d3" {
		t.Errorf("expected d3 as farthest, got %s", first.ID)
	}
}

func TestSelectorEvaluator_Nearest_WithDistanceConstraints(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d1", Position: ast.GeoPoint{Lat: 47.4979, Lon: 19.0402}}, // 0m (origin)
			{ID: "d2", Position: ast.GeoPoint{Lat: 47.498, Lon: 19.041}},   // ~50m
			{ID: "d3", Position: ast.GeoPoint{Lat: 47.51, Lon: 19.06}},     // ~2km
			{ID: "d4", Position: ast.GeoPoint{Lat: 48.0, Lon: 19.5}},       // ~60km
		},
	}
	ctx := NewContext(state, 0, 0)
	ctx = ctx.WithEntity(state.Drones[0], 0)
	se := NewSelectorEvaluator()

	// Only entities between 100m and 10000m (10km)
	selector := &ast.Selector{
		Type:        ast.SelectorTypeNearest,
		Entity:      "Drones",
		Origin:      "$.Position",
		MinDistance: 100,
		MaxDistance: 10000,
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// d1 (0m) and d2 (~50m) are too close, d4 (~60km) is too far
	// Only d3 (~2km) should match
	if len(result) != 1 {
		t.Errorf("expected 1 drone in distance range, got %d", len(result))
	}

	if len(result) > 0 {
		drone := result[0].(Drone)
		if drone.ID != "d3" {
			t.Errorf("expected d3, got %s", drone.ID)
		}
	}
}

func TestSelectorEvaluator_Nearest_CustomPositionField(t *testing.T) {
	state := &SelectorTestState{
		Drones: []Drone{
			{ID: "d0", Position: ast.GeoPoint{Lat: 47.4979, Lon: 19.0402}}, // Origin
			{ID: "d1", Position: ast.GeoPoint{Lat: 47.5, Lon: 19.05}},
			{ID: "d2", Position: ast.GeoPoint{Lat: 48.0, Lon: 19.5}},
		},
	}
	ctx := NewContext(state, 0, 0)
	ctx = ctx.WithEntity(state.Drones[0], 0) // Use d0 as the origin
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:     ast.SelectorTypeNearest,
		Entity:   "Drones",
		Origin:   "$.Position",
		Position: "$.Position",
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 drones, got %d", len(result))
	}

	// First should be d0 (same position as origin)
	if len(result) > 0 {
		first := result[0].(Drone)
		if first.ID != "d0" {
			t.Errorf("expected d0 as nearest, got %s", first.ID)
		}
	}
}

func TestSelectorEvaluator_UnknownType(t *testing.T) {
	state := &SelectorTestState{}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	selector := &ast.Selector{
		Type:   "Unknown",
		Entity: "Drones",
	}

	_, err := se.Select(ctx, selector)
	if err == nil {
		t.Error("expected error for unknown selector type")
	}
}

func TestSelectorEvaluator_WithPointerEntity(t *testing.T) {
	drone := &Drone{ID: "d1", Status: "Active"}
	state := &SelectorTestState{
		ActiveUnit: drone,
	}
	ctx := NewContext(state, 0, 0)
	se := NewSelectorEvaluator()

	// Can't directly select pointer entities in slice form
	// but we should handle them gracefully
	selector := &ast.Selector{
		Type:   ast.SelectorTypeAll,
		Entity: "ActiveUnit",
	}

	result, err := se.Select(ctx, selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}
