package eval

import (
	"testing"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// Test types for filter evaluation

type FilterTestPlayer struct {
	ID   string
	Name string
	Team string
}

type FilterTestMessage struct {
	ID     string
	Team   string
	Text   string
	Hidden bool
}

type FilterTestState struct {
	Players []FilterTestPlayer
	Chat    []FilterTestMessage
}

func TestFilterEvaluator_KeepWhere(t *testing.T) {
	state := &FilterTestState{
		Players: []FilterTestPlayer{
			{ID: "p1", Name: "Alice", Team: "catcher"},
			{ID: "p2", Name: "Bob", Team: "runner"},
		},
		Chat: []FilterTestMessage{
			{ID: "m1", Team: "catcher", Text: "Catcher message 1"},
			{ID: "m2", Team: "runner", Text: "Runner message 1"},
			{ID: "m3", Team: "catcher", Text: "Catcher message 2"},
		},
	}

	filter := &ast.Filter{
		Name: "TeamChatFilter",
		Params: []ast.FilterParam{
			{Name: "viewerTeam", Type: "string"},
		},
		Operations: []ast.FilterOperation{
			{
				ID:     "filterChat",
				Type:   ast.FilterOpKeepWhere,
				Target: "$.Chat",
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "==",
					Value: "param:viewerTeam",
				},
			},
		},
	}

	evaluator := NewFilterEvaluator()

	// Test for catcher viewer
	params := map[string]interface{}{
		"viewerTeam": "catcher",
	}

	result, err := evaluator.Apply(filter, state, params)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*FilterTestState)
	if len(filteredState.Chat) != 2 {
		t.Errorf("expected 2 catcher messages, got %d", len(filteredState.Chat))
	}

	for _, msg := range filteredState.Chat {
		if msg.Team != "catcher" {
			t.Errorf("found non-catcher message: %s", msg.Team)
		}
	}
}

func TestFilterEvaluator_RemoveWhere(t *testing.T) {
	state := &FilterTestState{
		Chat: []FilterTestMessage{
			{ID: "m1", Team: "catcher", Text: "Catcher msg"},
			{ID: "m2", Team: "runner", Text: "Runner msg"},
			{ID: "m3", Team: "spectator", Text: "Spectator msg"},
		},
	}

	filter := &ast.Filter{
		Name: "HideRunnerMessages",
		Operations: []ast.FilterOperation{
			{
				ID:     "removeRunners",
				Type:   ast.FilterOpRemoveWhere,
				Target: "$.Chat",
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "==",
					Value: "runner",
				},
			},
		},
	}

	evaluator := NewFilterEvaluator()

	result, err := evaluator.Apply(filter, state, nil)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*FilterTestState)
	if len(filteredState.Chat) != 2 {
		t.Errorf("expected 2 non-runner messages, got %d", len(filteredState.Chat))
	}

	for _, msg := range filteredState.Chat {
		if msg.Team == "runner" {
			t.Error("found runner message after filtering")
		}
	}
}

func TestFilterEvaluator_AndCondition(t *testing.T) {
	state := &FilterTestState{
		Chat: []FilterTestMessage{
			{ID: "m1", Team: "catcher", Hidden: false, Text: "Visible catcher"},
			{ID: "m2", Team: "catcher", Hidden: true, Text: "Hidden catcher"},
			{ID: "m3", Team: "runner", Hidden: false, Text: "Visible runner"},
		},
	}

	filter := &ast.Filter{
		Name: "VisibleCatcherMessages",
		Params: []ast.FilterParam{
			{Name: "viewerTeam", Type: "string"},
		},
		Operations: []ast.FilterOperation{
			{
				ID:     "filterChat",
				Type:   ast.FilterOpKeepWhere,
				Target: "$.Chat",
				Where: &ast.WhereClause{
					And: []*ast.WhereClause{
						{Field: "Team", Op: "==", Value: "param:viewerTeam"},
						{Field: "Hidden", Op: "==", Value: false},
					},
				},
			},
		},
	}

	evaluator := NewFilterEvaluator()

	params := map[string]interface{}{
		"viewerTeam": "catcher",
	}

	result, err := evaluator.Apply(filter, state, params)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*FilterTestState)
	if len(filteredState.Chat) != 1 {
		t.Errorf("expected 1 visible catcher message, got %d", len(filteredState.Chat))
	}

	if len(filteredState.Chat) > 0 && filteredState.Chat[0].ID != "m1" {
		t.Errorf("expected message m1, got %s", filteredState.Chat[0].ID)
	}
}

func TestFilterEvaluator_OrCondition(t *testing.T) {
	state := &FilterTestState{
		Chat: []FilterTestMessage{
			{ID: "m1", Team: "catcher", Text: "Catcher msg"},
			{ID: "m2", Team: "runner", Text: "Runner msg"},
			{ID: "m3", Team: "spectator", Text: "Spectator msg"},
		},
	}

	filter := &ast.Filter{
		Name: "CatcherOrRunnerMessages",
		Operations: []ast.FilterOperation{
			{
				ID:     "filterChat",
				Type:   ast.FilterOpKeepWhere,
				Target: "$.Chat",
				Where: &ast.WhereClause{
					Or: []*ast.WhereClause{
						{Field: "Team", Op: "==", Value: "catcher"},
						{Field: "Team", Op: "==", Value: "runner"},
					},
				},
			},
		},
	}

	evaluator := NewFilterEvaluator()

	result, err := evaluator.Apply(filter, state, nil)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*FilterTestState)
	if len(filteredState.Chat) != 2 {
		t.Errorf("expected 2 messages, got %d", len(filteredState.Chat))
	}
}

func TestFilterEvaluator_NotCondition(t *testing.T) {
	state := &FilterTestState{
		Chat: []FilterTestMessage{
			{ID: "m1", Team: "catcher", Text: "Catcher msg"},
			{ID: "m2", Team: "runner", Text: "Runner msg"},
		},
	}

	filter := &ast.Filter{
		Name: "NotCatcherMessages",
		Operations: []ast.FilterOperation{
			{
				ID:     "filterChat",
				Type:   ast.FilterOpKeepWhere,
				Target: "$.Chat",
				Where: &ast.WhereClause{
					Not: &ast.WhereClause{
						Field: "Team",
						Op:    "==",
						Value: "catcher",
					},
				},
			},
		},
	}

	evaluator := NewFilterEvaluator()

	result, err := evaluator.Apply(filter, state, nil)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*FilterTestState)
	if len(filteredState.Chat) != 1 {
		t.Errorf("expected 1 message, got %d", len(filteredState.Chat))
	}

	if len(filteredState.Chat) > 0 && filteredState.Chat[0].Team == "catcher" {
		t.Error("should not have catcher message")
	}
}

func TestFilterEvaluator_EmptyArray(t *testing.T) {
	state := &FilterTestState{
		Chat: []FilterTestMessage{},
	}

	filter := &ast.Filter{
		Name: "FilterEmpty",
		Operations: []ast.FilterOperation{
			{
				ID:     "filterChat",
				Type:   ast.FilterOpKeepWhere,
				Target: "$.Chat",
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "==",
					Value: "catcher",
				},
			},
		},
	}

	evaluator := NewFilterEvaluator()

	result, err := evaluator.Apply(filter, state, nil)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*FilterTestState)
	if len(filteredState.Chat) != 0 {
		t.Errorf("expected 0 messages, got %d", len(filteredState.Chat))
	}
}

func TestFilterEvaluator_NoMatch(t *testing.T) {
	state := &FilterTestState{
		Chat: []FilterTestMessage{
			{ID: "m1", Team: "runner", Text: "Runner only"},
		},
	}

	filter := &ast.Filter{
		Name: "CatcherOnly",
		Operations: []ast.FilterOperation{
			{
				ID:     "filterChat",
				Type:   ast.FilterOpKeepWhere,
				Target: "$.Chat",
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "==",
					Value: "catcher",
				},
			},
		},
	}

	evaluator := NewFilterEvaluator()

	result, err := evaluator.Apply(filter, state, nil)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*FilterTestState)
	if len(filteredState.Chat) != 0 {
		t.Errorf("expected 0 catcher messages, got %d", len(filteredState.Chat))
	}
}

// Test: Replace enemy names with "Unknown Runner"
func TestFilterEvaluator_ReplaceFieldWhere(t *testing.T) {
	state := &FilterTestState{
		Players: []FilterTestPlayer{
			{ID: "p1", Name: "Alice", Team: "catcher"},
			{ID: "p2", Name: "Bob", Team: "runner"},
			{ID: "p3", Name: "Charlie", Team: "runner"},
			{ID: "p4", Name: "Diana", Team: "catcher"},
		},
	}

	// Single declarative operation: replace Name for enemies
	filter := &ast.Filter{
		Name: "HideEnemyNames",
		Params: []ast.FilterParam{
			{Name: "viewerTeam", Type: "string"},
		},
		Operations: []ast.FilterOperation{
			{
				ID:     "hideNames",
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

	params := map[string]interface{}{
		"viewerTeam": "catcher",
	}

	result, err := evaluator.Apply(filter, state, params)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*FilterTestState)

	for _, p := range filteredState.Players {
		if p.Team == "catcher" {
			if p.Name == "Unknown Runner" {
				t.Errorf("catcher %s should keep their name", p.ID)
			}
		} else {
			if p.Name != "Unknown Runner" {
				t.Errorf("runner %s should be 'Unknown Runner', got '%s'", p.ID, p.Name)
			}
		}
	}
}

// Test: Hide enemy locations (set to 0)
func TestFilterEvaluator_HideFieldsWhere(t *testing.T) {
	type PlayerWithLocation struct {
		ID   string
		Name string
		Team string
		Lat  float64
		Lng  float64
	}

	type GameWithLocations struct {
		Players []PlayerWithLocation
	}

	state := &GameWithLocations{
		Players: []PlayerWithLocation{
			{ID: "p1", Name: "Alice", Team: "catcher", Lat: 47.5, Lng: 19.0},
			{ID: "p2", Name: "Bob", Team: "runner", Lat: 48.2, Lng: 20.1},
		},
	}

	// Single declarative operation: hide location fields for enemies
	filter := &ast.Filter{
		Name: "HideEnemyLocations",
		Params: []ast.FilterParam{
			{Name: "viewerTeam", Type: "string"},
		},
		Operations: []ast.FilterOperation{
			{
				ID:     "hideLocations",
				Type:   ast.FilterOpHideFieldsWhere,
				Target: "$.Players",
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "!=",
					Value: "param:viewerTeam",
				},
				Fields: []string{"Lat", "Lng"},
			},
		},
	}

	evaluator := NewFilterEvaluator()

	params := map[string]interface{}{
		"viewerTeam": "catcher",
	}

	result, err := evaluator.Apply(filter, state, params)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*GameWithLocations)

	// Catcher should see their own location
	if filteredState.Players[0].Lat != 47.5 {
		t.Errorf("catcher location should be visible")
	}

	// Runner location should be hidden (0)
	if filteredState.Players[1].Lat != 0 || filteredState.Players[1].Lng != 0 {
		t.Errorf("runner location should be hidden, got Lat=%f Lng=%f",
			filteredState.Players[1].Lat, filteredState.Players[1].Lng)
	}
}

// Test: Legacy FilterArray operation still works
func TestFilterEvaluator_LegacyFilterArray(t *testing.T) {
	state := &FilterTestState{
		Chat: []FilterTestMessage{
			{ID: "m1", Team: "catcher", Text: "Catcher msg"},
			{ID: "m2", Team: "runner", Text: "Runner msg"},
		},
	}

	filter := &ast.Filter{
		Name: "LegacyFilter",
		Operations: []ast.FilterOperation{
			{
				ID:     "filterChat",
				Type:   ast.FilterOpFilterArray, // Legacy type
				Target: "$.Chat",
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "==",
					Value: "catcher",
				},
			},
		},
	}

	evaluator := NewFilterEvaluator()

	result, err := evaluator.Apply(filter, state, nil)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*FilterTestState)
	if len(filteredState.Chat) != 1 {
		t.Errorf("expected 1 catcher message, got %d", len(filteredState.Chat))
	}
}

// Test: Filter Enabled field
func TestFilter_IsEnabled(t *testing.T) {
	// Default: enabled when Enabled is nil
	filter1 := &ast.Filter{
		Name:       "TestFilter",
		Operations: []ast.FilterOperation{},
	}
	if !filter1.IsEnabled() {
		t.Error("filter should be enabled by default (nil)")
	}

	// Explicitly enabled
	enabled := true
	filter2 := &ast.Filter{
		Name:       "EnabledFilter",
		Enabled:    &enabled,
		Operations: []ast.FilterOperation{},
	}
	if !filter2.IsEnabled() {
		t.Error("filter should be enabled when Enabled=true")
	}

	// Explicitly disabled
	disabled := false
	filter3 := &ast.Filter{
		Name:       "DisabledFilter",
		Enabled:    &disabled,
		Operations: []ast.FilterOperation{},
	}
	if filter3.IsEnabled() {
		t.Error("filter should be disabled when Enabled=false")
	}
}

// Test: SetEnabled method
func TestFilter_SetEnabled(t *testing.T) {
	filter := &ast.Filter{
		Name:       "TestFilter",
		Operations: []ast.FilterOperation{},
	}

	// Initially enabled (nil)
	if !filter.IsEnabled() {
		t.Error("filter should be enabled by default")
	}

	// Disable it
	filter.SetEnabled(false)
	if filter.IsEnabled() {
		t.Error("filter should be disabled after SetEnabled(false)")
	}

	// Re-enable it
	filter.SetEnabled(true)
	if !filter.IsEnabled() {
		t.Error("filter should be enabled after SetEnabled(true)")
	}
}

// Test: Combined operations - filter then hide
func TestFilterEvaluator_CombinedOperations(t *testing.T) {
	state := &FilterTestState{
		Players: []FilterTestPlayer{
			{ID: "p1", Name: "Alice", Team: "catcher"},
			{ID: "p2", Name: "Bob", Team: "runner"},
			{ID: "p3", Name: "Charlie", Team: "runner"},
		},
		Chat: []FilterTestMessage{
			{ID: "m1", Team: "catcher", Text: "Team chat"},
			{ID: "m2", Team: "runner", Text: "Enemy chat"},
		},
	}

	// Multiple operations in sequence
	filter := &ast.Filter{
		Name: "CatcherView",
		Params: []ast.FilterParam{
			{Name: "viewerTeam", Type: "string"},
		},
		Operations: []ast.FilterOperation{
			// 1. Keep only own team's chat
			{
				ID:     "filterChat",
				Type:   ast.FilterOpKeepWhere,
				Target: "$.Chat",
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "==",
					Value: "param:viewerTeam",
				},
			},
			// 2. Hide enemy names
			{
				ID:     "hideEnemyNames",
				Type:   ast.FilterOpReplaceFieldWhere,
				Target: "$.Players",
				Where: &ast.WhereClause{
					Field: "Team",
					Op:    "!=",
					Value: "param:viewerTeam",
				},
				Field: "Name",
				Value: "???",
			},
		},
	}

	evaluator := NewFilterEvaluator()

	params := map[string]interface{}{
		"viewerTeam": "catcher",
	}

	result, err := evaluator.Apply(filter, state, params)
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}

	filteredState := result.(*FilterTestState)

	// Chat should only have catcher message
	if len(filteredState.Chat) != 1 {
		t.Errorf("expected 1 chat message, got %d", len(filteredState.Chat))
	}

	// Enemies should have "???" as name
	hiddenCount := 0
	for _, p := range filteredState.Players {
		if p.Team != "catcher" && p.Name == "???" {
			hiddenCount++
		}
	}
	if hiddenCount != 2 {
		t.Errorf("expected 2 hidden names, got %d", hiddenCount)
	}
}

// Test: EnableFilter and DisableFilter effect types are valid
func TestEffect_EnableDisableFilter(t *testing.T) {
	// Test EnableFilter effect
	enableEffect := &ast.Effect{
		Type:   ast.EffectTypeEnableFilter,
		Filter: "HideEnemyLocations",
	}
	if enableEffect.Type != ast.EffectTypeEnableFilter {
		t.Error("EnableFilter type should be set")
	}
	if enableEffect.Filter != "HideEnemyLocations" {
		t.Error("Filter field should be set")
	}

	// Test DisableFilter effect
	disableEffect := &ast.Effect{
		Type:   ast.EffectTypeDisableFilter,
		Filter: "HideEnemyLocations",
	}
	if disableEffect.Type != ast.EffectTypeDisableFilter {
		t.Error("DisableFilter type should be set")
	}
	if disableEffect.Filter != "HideEnemyLocations" {
		t.Error("Filter field should be set")
	}
}

// Test: JSON unmarshaling of EnableFilter/DisableFilter effects
func TestEffect_EnableDisableFilter_JSON(t *testing.T) {
	jsonData := `{"type": "EnableFilter", "filter": "TeamChatFilter"}`

	var effect ast.Effect
	if err := effect.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("failed to unmarshal EnableFilter: %v", err)
	}

	if effect.Type != ast.EffectTypeEnableFilter {
		t.Errorf("expected EnableFilter, got %s", effect.Type)
	}
	if effect.Filter != "TeamChatFilter" {
		t.Errorf("expected TeamChatFilter, got %s", effect.Filter)
	}
}
