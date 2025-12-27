package parse

import (
	"testing"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

func TestRequiredFieldsValidator(t *testing.T) {
	v := &RequiredFieldsValidator{}

	tests := []struct {
		name    string
		ruleSet *ast.RuleSet
		wantErr bool
	}{
		{
			name: "valid rule",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name:    "ValidRule",
						Trigger: &ast.Trigger{Type: ast.TriggerTypeOnTick},
						Effects: []*ast.Effect{{Type: ast.EffectTypeSet}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing rule name",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name:    "",
						Trigger: &ast.Trigger{Type: ast.TriggerTypeOnTick},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing trigger type",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name:    "Test",
						Trigger: &ast.Trigger{Type: ""},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing effect type",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name:    "Test",
						Effects: []*ast.Effect{{Type: ""}},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.ruleSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathValidator(t *testing.T) {
	v := &PathValidator{}

	tests := []struct {
		name    string
		ruleSet *ast.RuleSet
		wantErr bool
	}{
		{
			name: "valid JSONPath",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{Type: ast.EffectTypeSet, Path: "$.Field"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid param reference",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{Type: ast.EffectTypeSet, Path: "param:value"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid view reference",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{Type: ast.EffectTypeSet, Path: "view:total"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid path format",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{Type: ast.EffectTypeSet, Path: "invalid.path"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "unbalanced brackets - missing close",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{Type: ast.EffectTypeSet, Path: "$.Items[0"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "unbalanced brackets - extra close",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{Type: ast.EffectTypeSet, Path: "$.Items]"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty path is valid",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name:    "Test",
						Effects: []*ast.Effect{{Type: ast.EffectTypeSet}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "view pipeline field validation",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Views: map[string]*ast.View{
							"total": {
								Source: "Players",
								Pipeline: []ast.ViewOperation{
									{Type: ast.ViewOpSum, Field: "$.Score"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.ruleSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTriggerValidator(t *testing.T) {
	v := &TriggerValidator{}

	tests := []struct {
		name    string
		ruleSet *ast.RuleSet
		wantErr bool
	}{
		{
			name: "OnTick is always valid",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeOnTick}},
				},
			},
			wantErr: false,
		},
		{
			name: "OnEvent requires event name",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeOnEvent, Event: ""}},
				},
			},
			wantErr: true,
		},
		{
			name: "OnEvent with event name is valid",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeOnEvent, Event: "PlayerJoined"}},
				},
			},
			wantErr: false,
		},
		{
			name: "OnChange requires watch paths",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeOnChange}},
				},
			},
			wantErr: true,
		},
		{
			name: "OnChange with watch paths is valid",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeOnChange, Watch: []ast.Path{"$.Field"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "Distance requires from and to",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeDistance, Value: 100}},
				},
			},
			wantErr: true,
		},
		{
			name: "Distance rejects negative value",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeDistance, From: "$.A", To: "$.B", Value: -1}},
				},
			},
			wantErr: true,
		},
		{
			name: "Distance allows zero value",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeDistance, From: "$.A", To: "$.B", Value: 0}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid Distance trigger",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeDistance, From: "$.A", To: "$.B", Value: 100}},
				},
			},
			wantErr: false,
		},
		{
			name: "Timer requires positive duration",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeTimer, Duration: 0}},
				},
			},
			wantErr: true,
		},
		{
			name: "valid Timer trigger",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: &ast.Trigger{Type: ast.TriggerTypeTimer, Duration: 5000}},
				},
			},
			wantErr: false,
		},
		{
			name: "nil trigger is valid",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Trigger: nil},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.ruleSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEffectValidator(t *testing.T) {
	v := &EffectValidator{}

	tests := []struct {
		name    string
		ruleSet *ast.RuleSet
		wantErr bool
	}{
		{
			name: "Set requires path",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Effects: []*ast.Effect{{Type: ast.EffectTypeSet, Value: 100}}},
				},
			},
			wantErr: true,
		},
		{
			name: "Set requires value",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Effects: []*ast.Effect{{Type: ast.EffectTypeSet, Path: "$.Field"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "valid Set effect",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Effects: []*ast.Effect{{Type: ast.EffectTypeSet, Path: "$.Field", Value: 100}}},
				},
			},
			wantErr: false,
		},
		{
			name: "Increment requires path",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Effects: []*ast.Effect{{Type: ast.EffectTypeIncrement, Value: 10}}},
				},
			},
			wantErr: true,
		},
		{
			name: "Emit requires event",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Effects: []*ast.Effect{{Type: ast.EffectTypeEmit}}},
				},
			},
			wantErr: true,
		},
		{
			name: "Spawn requires entity",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Effects: []*ast.Effect{{Type: ast.EffectTypeSpawn}}},
				},
			},
			wantErr: true,
		},
		{
			name: "SetFromView requires path",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Effects: []*ast.Effect{{Type: ast.EffectTypeSetFromView, ValueExpression: &ast.ValueExpression{}}}},
				},
			},
			wantErr: true,
		},
		{
			name: "SetFromView requires valueExpression",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Effects: []*ast.Effect{{Type: ast.EffectTypeSetFromView, Path: "$.Field"}}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.ruleSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestViewValidator(t *testing.T) {
	v := &ViewValidator{}

	tests := []struct {
		name    string
		ruleSet *ast.RuleSet
		wantErr bool
	}{
		{
			name: "view requires source",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Views: map[string]*ast.View{"v": {}}},
				},
			},
			wantErr: true,
		},
		{
			name: "valid view with source",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{Name: "Test", Views: map[string]*ast.View{"v": {Source: "Players"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "Filter requires where clause",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Views: map[string]*ast.View{
							"v": {
								Source:   "Players",
								Pipeline: []ast.ViewOperation{{Type: ast.ViewOpFilter}},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "OrderBy requires by field",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Views: map[string]*ast.View{
							"v": {
								Source:   "Players",
								Pipeline: []ast.ViewOperation{{Type: ast.ViewOpOrderBy}},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "GroupBy requires groupField",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Views: map[string]*ast.View{
							"v": {
								Source:   "Players",
								Pipeline: []ast.ViewOperation{{Type: ast.ViewOpGroupBy}},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Sum requires field",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Views: map[string]*ast.View{
							"v": {
								Source:   "Players",
								Pipeline: []ast.ViewOperation{{Type: ast.ViewOpSum}},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Count doesn't require field",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Views: map[string]*ast.View{
							"v": {
								Source:   "Players",
								Pipeline: []ast.ViewOperation{{Type: ast.ViewOpCount}},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "First doesn't require field",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Views: map[string]*ast.View{
							"v": {
								Source:   "Players",
								Pipeline: []ast.ViewOperation{{Type: ast.ViewOpFirst}},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Nearest requires origin and position",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Views: map[string]*ast.View{
							"v": {
								Source:   "Players",
								Pipeline: []ast.ViewOperation{{Type: ast.ViewOpNearest}},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.ruleSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransformValidator(t *testing.T) {
	v := &TransformValidator{}

	tests := []struct {
		name    string
		ruleSet *ast.RuleSet
		wantErr bool
	}{
		{
			name: "MoveTowards requires current and target",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{
								Type: ast.EffectTypeSetFromView,
								Path: "$.Pos",
								ValueExpression: &ast.ValueExpression{
									Transform: &ast.Transform{Type: ast.TransformTypeMoveTowards, Speed: 10},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "MoveTowards requires positive speed",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{
								Type: ast.EffectTypeSetFromView,
								Path: "$.Pos",
								ValueExpression: &ast.ValueExpression{
									Transform: &ast.Transform{Type: ast.TransformTypeMoveTowards, Current: "$.Pos", Target: "$.Target", Speed: 0},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "GpsDistance requires from and to",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{
								Type: ast.EffectTypeSetFromView,
								Path: "$.Dist",
								ValueExpression: &ast.ValueExpression{
									Transform: &ast.Transform{Type: ast.TransformTypeGpsDistance},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Clamp requires value, min, and max",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{
								Type: ast.EffectTypeSetFromView,
								Path: "$.Val",
								ValueExpression: &ast.ValueExpression{
									Transform: &ast.Transform{Type: ast.TransformTypeClamp},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "If requires condition",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{
								Type: ast.EffectTypeSetFromView,
								Path: "$.Val",
								ValueExpression: &ast.ValueExpression{
									Transform: &ast.Transform{Type: ast.TransformTypeIf},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Add doesn't have strict requirements",
			ruleSet: &ast.RuleSet{
				Rules: []*ast.Rule{
					{
						Name: "Test",
						Effects: []*ast.Effect{
							{
								Type: ast.EffectTypeSetFromView,
								Path: "$.Val",
								ValueExpression: &ast.ValueExpression{
									Transform: &ast.Transform{Type: ast.TransformTypeAdd},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.ruleSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompositeValidator(t *testing.T) {
	cv := NewCompositeValidator(
		&RequiredFieldsValidator{},
		&PathValidator{},
	)

	// Valid rule
	validRuleSet := &ast.RuleSet{
		Rules: []*ast.Rule{
			{Name: "Test", Effects: []*ast.Effect{{Type: ast.EffectTypeSet, Path: "$.Field", Value: 1}}},
		},
	}

	err := cv.Validate(validRuleSet)
	if err != nil {
		t.Errorf("unexpected error for valid ruleset: %v", err)
	}

	// Invalid rule - multiple errors
	invalidRuleSet := &ast.RuleSet{
		Rules: []*ast.Rule{
			{Name: "", Effects: []*ast.Effect{{Type: "", Path: "invalid.path"}}},
		},
	}

	err = cv.Validate(invalidRuleSet)
	if err == nil {
		t.Error("expected error for invalid ruleset")
	}

	// Check it returns ValidationErrors
	if _, ok := err.(*ValidationErrors); !ok {
		t.Errorf("expected *ValidationErrors, got %T", err)
	}
}

func TestStrictValidator(t *testing.T) {
	sv := StrictValidator()

	// Should have all validators
	cv, ok := sv.(*CompositeValidator)
	if !ok {
		t.Fatal("StrictValidator should return *CompositeValidator")
	}

	if len(cv.validators) != 7 {
		t.Errorf("expected 7 validators, got %d", len(cv.validators))
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"", false},                    // Empty is valid
		{"$.Field", false},             // JSONPath
		{"$.Field.Nested", false},      // Nested
		{"$.Items[0]", false},          // Array index
		{"$.Items[0].Name", false},     // Array with field
		{"param:value", false},         // Param reference
		{"view:total", false},          // View reference
		{"const:100", false},           // Const reference
		{"state:global.config", false}, // State reference
		{"self.field", false},          // Self reference
		{"invalid.path", true},         // No prefix
		{"$.Items[0", true},            // Unbalanced - missing ]
		{"$.Items]", true},             // Unbalanced - extra ]
		{"$.Items[0]]", true},          // Unbalanced - extra ]
		{"$.Items[[0]", true},          // Unbalanced - extra [
		{"$.A[0].B[1].C[2]", false},    // Multiple array accesses
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
