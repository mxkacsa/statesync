package parse

import (
	"testing"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

func TestParser_ParseSingleRule(t *testing.T) {
	input := []byte(`{
		"name": "TestRule",
		"description": "A test rule",
		"priority": 10,
		"trigger": {
			"type": "OnTick",
			"interval": 1000
		},
		"views": {
			"allPlayers": {
				"source": "Players"
			}
		},
		"effects": [
			{
				"type": "Increment",
				"targets": "allPlayers",
				"path": "$.Score",
				"value": 1
			}
		]
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ruleSet.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ruleSet.Rules))
	}

	rule := ruleSet.Rules[0]
	if rule.Name != "TestRule" {
		t.Errorf("expected name 'TestRule', got '%s'", rule.Name)
	}
	if rule.Description != "A test rule" {
		t.Errorf("expected description 'A test rule', got '%s'", rule.Description)
	}
	if rule.GetPriority() != 10 {
		t.Errorf("expected priority 10, got %d", rule.GetPriority())
	}

	// Check trigger
	if rule.Trigger == nil {
		t.Fatal("expected trigger to be set")
	}
	if rule.Trigger.Type != ast.TriggerTypeOnTick {
		t.Errorf("expected trigger type 'OnTick', got '%s'", rule.Trigger.Type)
	}
	if rule.Trigger.Interval != 1000 {
		t.Errorf("expected interval 1000, got %d", rule.Trigger.Interval)
	}

	// Check views
	if rule.Views == nil || rule.Views["allPlayers"] == nil {
		t.Fatal("expected views to be set")
	}
	if rule.Views["allPlayers"].Source != "Players" {
		t.Errorf("expected source 'Players', got '%s'", rule.Views["allPlayers"].Source)
	}

	// Check effects
	if len(rule.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(rule.Effects))
	}
	if rule.Effects[0].Type != ast.EffectTypeIncrement {
		t.Errorf("expected effect type 'Increment', got '%s'", rule.Effects[0].Type)
	}
}

func TestParser_ParseRuleArray(t *testing.T) {
	input := []byte(`[
		{
			"name": "Rule1",
			"trigger": {"type": "OnTick"},
			"effects": [{"type": "Set", "path": "$.A", "value": 1}]
		},
		{
			"name": "Rule2",
			"trigger": {"type": "OnTick"},
			"effects": [{"type": "Set", "path": "$.B", "value": 2}]
		}
	]`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ruleSet.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(ruleSet.Rules))
	}

	if ruleSet.Rules[0].Name != "Rule1" {
		t.Errorf("expected first rule name 'Rule1', got '%s'", ruleSet.Rules[0].Name)
	}
	if ruleSet.Rules[1].Name != "Rule2" {
		t.Errorf("expected second rule name 'Rule2', got '%s'", ruleSet.Rules[1].Name)
	}
}

func TestParser_ParseRuleSet(t *testing.T) {
	input := []byte(`{
		"version": "2.0",
		"name": "TestRuleSet",
		"rules": [
			{
				"name": "Rule1",
				"trigger": {"type": "OnTick"},
				"effects": []
			}
		]
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ruleSet.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ruleSet.Rules))
	}
}

func TestParser_ParseTriggerTypes(t *testing.T) {
	tests := []struct {
		name        string
		triggerJSON string
		expected    ast.TriggerType
	}{
		{
			name:        "OnTick",
			triggerJSON: `{"type": "OnTick"}`,
			expected:    ast.TriggerTypeOnTick,
		},
		{
			name:        "OnEvent",
			triggerJSON: `{"type": "OnEvent", "event": "PlayerJoined"}`,
			expected:    ast.TriggerTypeOnEvent,
		},
		{
			name:        "Timer",
			triggerJSON: `{"type": "Timer", "duration": 5000}`,
			expected:    ast.TriggerTypeTimer,
		},
		{
			name:        "Distance",
			triggerJSON: `{"type": "Distance", "from": "$.Player.Position", "to": "$.Target.Position", "value": 100, "operator": "<="}`,
			expected:    ast.TriggerTypeDistance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(`{
				"name": "TestRule",
				"trigger": ` + tt.triggerJSON + `,
				"effects": []
			}`)

			parser := NewParser()
			ruleSet, err := parser.Parse(input, "test.json")
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if ruleSet.Rules[0].Trigger.Type != tt.expected {
				t.Errorf("expected trigger type '%s', got '%s'", tt.expected, ruleSet.Rules[0].Trigger.Type)
			}
		})
	}
}

func TestParser_ParseEffectTypes(t *testing.T) {
	tests := []struct {
		name       string
		effectJSON string
		expected   ast.EffectType
	}{
		{
			name:       "Set",
			effectJSON: `{"type": "Set", "path": "$.Score", "value": 100}`,
			expected:   ast.EffectTypeSet,
		},
		{
			name:       "Increment",
			effectJSON: `{"type": "Increment", "path": "$.Score", "value": 10}`,
			expected:   ast.EffectTypeIncrement,
		},
		{
			name:       "Decrement",
			effectJSON: `{"type": "Decrement", "path": "$.Score", "value": 5}`,
			expected:   ast.EffectTypeDecrement,
		},
		{
			name:       "Emit",
			effectJSON: `{"type": "Emit", "event": "ScoreUpdated", "payload": {"score": "$.Score"}}`,
			expected:   ast.EffectTypeEmit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(`{
				"name": "TestRule",
				"trigger": {"type": "OnTick"},
				"effects": [` + tt.effectJSON + `]
			}`)

			parser := NewParser()
			ruleSet, err := parser.Parse(input, "test.json")
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(ruleSet.Rules[0].Effects) != 1 {
				t.Fatalf("expected 1 effect, got %d", len(ruleSet.Rules[0].Effects))
			}
			if ruleSet.Rules[0].Effects[0].Type != tt.expected {
				t.Errorf("expected effect type '%s', got '%s'", tt.expected, ruleSet.Rules[0].Effects[0].Type)
			}
		})
	}
}

func TestParser_ParseViews(t *testing.T) {
	input := []byte(`{
		"name": "TestRule",
		"trigger": {"type": "OnTick"},
		"views": {
			"allPlayers": {
				"source": "Players"
			},
			"highScorers": {
				"source": "Players",
				"pipeline": [
					{"type": "Filter", "where": {"field": "Score", "op": ">=", "value": 100}}
				]
			},
			"topScorer": {
				"source": "Players",
				"pipeline": [
					{"type": "OrderBy", "by": "Score", "order": "desc"},
					{"type": "First"}
				]
			}
		},
		"effects": []
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	rule := ruleSet.Rules[0]
	if len(rule.Views) != 3 {
		t.Fatalf("expected 3 views, got %d", len(rule.Views))
	}

	if rule.Views["allPlayers"] == nil || rule.Views["allPlayers"].Source != "Players" {
		t.Error("expected 'allPlayers' view with source Players")
	}
	if rule.Views["highScorers"] == nil || len(rule.Views["highScorers"].Pipeline) != 1 {
		t.Error("expected 'highScorers' view with 1 pipeline operation")
	}
	if rule.Views["topScorer"] == nil || len(rule.Views["topScorer"].Pipeline) != 2 {
		t.Error("expected 'topScorer' view with 2 pipeline operations")
	}
}

func TestParser_InvalidJSON(t *testing.T) {
	input := []byte(`{invalid json}`)

	parser := NewParser()
	_, err := parser.Parse(input, "test.json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParser_MissingRequiredFields(t *testing.T) {
	// Missing name
	input := []byte(`{
		"trigger": {"type": "OnTick"},
		"effects": []
	}`)

	parser := NewParser()
	_, err := parser.Parse(input, "test.json")
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestParser_RuleSetWithName_NotConfusedWithSingleRule(t *testing.T) {
	// RuleSet with "name" field should not be confused with single Rule
	input := []byte(`{
		"version": "2.0",
		"name": "MyRuleSet",
		"description": "Test rule set",
		"rules": [
			{
				"name": "Rule1",
				"trigger": {"type": "OnTick"},
				"effects": [{"type": "Set", "path": "$.Test", "value": 1}]
			},
			{
				"name": "Rule2",
				"trigger": {"type": "OnTick"},
				"effects": [{"type": "Set", "path": "$.Test", "value": 2}]
			}
		]
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ruleSet.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d (should not be confused as single rule)", len(ruleSet.Rules))
	}
}

func TestParser_EmptyRuleSet(t *testing.T) {
	input := []byte(`{
		"version": "2.0",
		"name": "Empty",
		"rules": []
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ruleSet.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(ruleSet.Rules))
	}
}

func TestParser_InvalidRuleFormat(t *testing.T) {
	// Valid JSON but not a valid rule format
	input := []byte(`{"foo": "bar"}`)

	parser := NewParser()
	_, err := parser.Parse(input, "test.json")
	if err == nil {
		t.Error("expected error for invalid rule format")
	}
}

func TestParser_RuleWithConditionTrigger(t *testing.T) {
	input := []byte(`{
		"name": "ConditionRule",
		"trigger": {
			"type": "Condition",
			"condition": {
				"and": [
					{"field": "$.Health", "op": ">", "value": 0},
					{"field": "$.Status", "op": "==", "value": "Active"}
				]
			}
		},
		"effects": []
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if ruleSet.Rules[0].Trigger.Type != ast.TriggerTypeCondition {
		t.Errorf("expected Condition trigger type, got %s", ruleSet.Rules[0].Trigger.Type)
	}
}

func TestParser_RuleWithSetFromView(t *testing.T) {
	input := []byte(`{
		"name": "SetFromViewRule",
		"trigger": {"type": "OnTick"},
		"views": {
			"allPlayers": {"source": "Players"}
		},
		"effects": [
			{
				"type": "SetFromView",
				"targets": "allPlayers",
				"path": "$.Distance",
				"valueExpression": {
					"type": "distance",
					"from": "self.position",
					"to": "view:nearestEnemy"
				}
			}
		]
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	effect := ruleSet.Rules[0].Effects[0]
	if effect.Type != ast.EffectTypeSetFromView {
		t.Errorf("expected SetFromView effect type, got %s", effect.Type)
	}
	if effect.ValueExpression == nil {
		t.Fatal("expected valueExpression to be set")
	}
}

func TestParser_RuleWithViewPipeline(t *testing.T) {
	input := []byte(`{
		"name": "PipelineRule",
		"trigger": {"type": "OnTick"},
		"views": {
			"topRunners": {
				"source": "Players",
				"pipeline": [
					{"type": "Filter", "where": {"field": "Team", "op": "==", "value": "runner"}},
					{"type": "OrderBy", "by": "Score", "order": "desc"},
					{"type": "Limit", "count": 3}
				]
			}
		},
		"effects": []
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	view := ruleSet.Rules[0].Views["topRunners"]
	if view == nil {
		t.Fatal("expected topRunners view")
	}
	if len(view.Pipeline) != 3 {
		t.Errorf("expected 3 pipeline operations, got %d", len(view.Pipeline))
	}
}

func TestParser_RuleWithDisabledFlag(t *testing.T) {
	input := []byte(`{
		"name": "DisabledRule",
		"enabled": false,
		"trigger": {"type": "OnTick"},
		"effects": [{"type": "Set", "path": "$.Test", "value": 1}]
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	rule := ruleSet.Rules[0]
	if rule.Enabled == nil {
		t.Fatal("expected Enabled to be set")
	}
	if *rule.Enabled != false {
		t.Error("expected Enabled to be false")
	}
}

func TestParser_RuleWithIfElseEffect(t *testing.T) {
	input := []byte(`{
		"name": "IfElseRule",
		"trigger": {"type": "OnTick"},
		"effects": [
			{
				"type": "If",
				"condition": "$.Health > 0",
				"then": {
					"type": "Set",
					"path": "$.Status",
					"value": "Alive"
				},
				"else": {
					"type": "Set",
					"path": "$.Status",
					"value": "Dead"
				}
			}
		]
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	effect := ruleSet.Rules[0].Effects[0]
	if effect.Type != ast.EffectTypeIf {
		t.Errorf("expected If effect type, got %s", effect.Type)
	}
	if effect.Then == nil {
		t.Error("expected Then effect")
	}
	if effect.Else == nil {
		t.Error("expected Else effect")
	}
}

func TestParser_RuleWithSequenceEffect(t *testing.T) {
	input := []byte(`{
		"name": "SequenceRule",
		"trigger": {"type": "OnTick"},
		"effects": [
			{
				"type": "Sequence",
				"effects": [
					{"type": "Set", "path": "$.A", "value": 1},
					{"type": "Set", "path": "$.B", "value": 2},
					{"type": "Increment", "path": "$.C", "value": 1}
				]
			}
		]
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	effect := ruleSet.Rules[0].Effects[0]
	if effect.Type != ast.EffectTypeSequence {
		t.Errorf("expected Sequence effect type, got %s", effect.Type)
	}
	if len(effect.Effects) != 3 {
		t.Errorf("expected 3 sub-effects, got %d", len(effect.Effects))
	}
}

func TestParser_ViewWithWhereClause(t *testing.T) {
	input := []byte(`{
		"name": "FilterRule",
		"trigger": {"type": "OnTick"},
		"views": {
			"highScorers": {
				"source": "Players",
				"pipeline": [
					{
						"type": "Filter",
						"where": {
							"field": "Score",
							"op": ">=",
							"value": 100
						}
					}
				]
			}
		},
		"effects": []
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	view := ruleSet.Rules[0].Views["highScorers"]
	if view == nil || len(view.Pipeline) != 1 {
		t.Fatal("expected view with 1 pipeline operation")
	}

	op := view.Pipeline[0]
	if op.Where == nil {
		t.Fatal("expected Where clause")
	}
	if op.Where.Field != "Score" {
		t.Errorf("expected field 'Score', got '%s'", op.Where.Field)
	}
	if op.Where.Op != ">=" {
		t.Errorf("expected op '>=', got '%s'", op.Where.Op)
	}
}

func TestParser_AddValidator(t *testing.T) {
	parser := NewParser()
	initialCount := len(parser.validators)

	customValidator := &RequiredFieldsValidator{}
	parser.AddValidator(customValidator)

	if len(parser.validators) != initialCount+1 {
		t.Errorf("expected %d validators, got %d", initialCount+1, len(parser.validators))
	}
}

func TestParseRule_Function(t *testing.T) {
	data := map[string]interface{}{
		"name":     "ParsedRule",
		"priority": float64(50),
		"trigger": map[string]interface{}{
			"type": "OnTick",
		},
		"effects": []interface{}{
			map[string]interface{}{
				"type":  "Set",
				"path":  "$.Value",
				"value": float64(100),
			},
		},
	}

	rule, err := ParseRule(data)
	if err != nil {
		t.Fatalf("ParseRule failed: %v", err)
	}

	if rule.Name != "ParsedRule" {
		t.Errorf("expected name 'ParsedRule', got %s", rule.Name)
	}
	if rule.Priority != 50 {
		t.Errorf("expected priority 50, got %d", rule.Priority)
	}
}

func TestParseTrigger_Function(t *testing.T) {
	data := map[string]interface{}{
		"type":     "OnEvent",
		"event":    "PlayerJoined",
		"interval": float64(1000),
	}

	trigger, err := ParseTrigger(data)
	if err != nil {
		t.Fatalf("ParseTrigger failed: %v", err)
	}

	if trigger.Type != ast.TriggerTypeOnEvent {
		t.Errorf("expected type OnEvent, got %s", trigger.Type)
	}
	if trigger.Event != "PlayerJoined" {
		t.Errorf("expected event 'PlayerJoined', got %s", trigger.Event)
	}
}

func TestParseView_Function(t *testing.T) {
	data := map[string]interface{}{
		"source": "Players",
		"pipeline": []interface{}{
			map[string]interface{}{
				"type":  "Filter",
				"where": map[string]interface{}{"field": "Score", "op": ">=", "value": float64(100)},
			},
		},
	}

	view, err := ParseView(data)
	if err != nil {
		t.Fatalf("ParseView failed: %v", err)
	}

	if view.Source != "Players" {
		t.Errorf("expected source 'Players', got %s", view.Source)
	}
}

func TestParseEffect_Function(t *testing.T) {
	data := map[string]interface{}{
		"type":  "Set",
		"path":  "$.Health",
		"value": float64(100),
	}

	effect, err := ParseEffect(data)
	if err != nil {
		t.Fatalf("ParseEffect failed: %v", err)
	}

	if effect.Type != ast.EffectTypeSet {
		t.Errorf("expected type Set, got %s", effect.Type)
	}
}

func TestParseTransform_Function(t *testing.T) {
	data := map[string]interface{}{
		"type": "Add",
		"operands": []interface{}{
			"$.Score",
			float64(10),
		},
	}

	transform, err := ParseTransform(data)
	if err != nil {
		t.Fatalf("ParseTransform failed: %v", err)
	}

	if transform.Type != ast.TransformTypeAdd {
		t.Errorf("expected type Add, got %s", transform.Type)
	}
}

func TestParser_EffectWithTargets(t *testing.T) {
	input := []byte(`{
		"name": "BatchEffectRule",
		"trigger": {"type": "OnTick"},
		"views": {
			"allPlayers": {"source": "Players"}
		},
		"effects": [
			{
				"type": "Increment",
				"targets": "allPlayers",
				"path": "$.Score",
				"value": 10
			}
		]
	}`)

	parser := NewParser()
	ruleSet, err := parser.Parse(input, "test.json")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	effect := ruleSet.Rules[0].Effects[0]
	if effect.Targets != "allPlayers" {
		t.Errorf("expected targets 'allPlayers', got '%v'", effect.Targets)
	}
}
