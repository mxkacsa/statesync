package parse

import (
	"fmt"
	"strings"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// Validator validates parsed rules
type Validator interface {
	Validate(ruleSet *ast.RuleSet) error
}

// RequiredFieldsValidator validates that required fields are present
type RequiredFieldsValidator struct{}

// Validate validates required fields
func (v *RequiredFieldsValidator) Validate(ruleSet *ast.RuleSet) error {
	var errors []string

	for i, rule := range ruleSet.Rules {
		if rule.Name == "" {
			errors = append(errors, fmt.Sprintf("rule[%d]: name is required", i))
		}

		// Validate trigger if present
		if rule.Trigger != nil {
			if rule.Trigger.Type == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].trigger: type is required", i))
			}
		}

		// Validate selector if present
		if rule.Selector != nil {
			if rule.Selector.Type == "" && rule.Selector.Entity == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].selector: type or entity is required", i))
			}
		}

		// Validate effects
		for j, effect := range rule.Effects {
			if effect.Type == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: type is required", i, j))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}
	return nil
}

// PathValidator validates path expressions
type PathValidator struct{}

// Validate validates path expressions
func (v *PathValidator) Validate(ruleSet *ast.RuleSet) error {
	var errors []string

	for i, rule := range ruleSet.Rules {
		// Validate effect paths
		for j, effect := range rule.Effects {
			if effect.Path != "" {
				if err := validatePath(string(effect.Path)); err != nil {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d].path: %s", i, j, err))
				}
			}
		}

		// Validate view paths
		for name, view := range rule.Views {
			if view.Field != "" {
				if err := validatePath(string(view.Field)); err != nil {
					errors = append(errors, fmt.Sprintf("rule[%d].views.%s.field: %s", i, name, err))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("path validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}
	return nil
}

// validatePath validates a single path expression
func validatePath(path string) error {
	if path == "" {
		return nil
	}

	// Path must start with $ or be a reference
	if !strings.HasPrefix(path, "$") &&
		!strings.HasPrefix(path, "param:") &&
		!strings.HasPrefix(path, "view:") &&
		!strings.HasPrefix(path, "const:") &&
		!strings.HasPrefix(path, "state:") {
		return fmt.Errorf("invalid path format: %s", path)
	}

	// Check for balanced brackets
	bracketCount := 0
	for _, ch := range path {
		if ch == '[' {
			bracketCount++
		} else if ch == ']' {
			bracketCount--
			if bracketCount < 0 {
				return fmt.Errorf("unbalanced brackets in path: %s", path)
			}
		}
	}
	if bracketCount != 0 {
		return fmt.Errorf("unbalanced brackets in path: %s", path)
	}

	return nil
}

// TriggerValidator validates trigger configurations
type TriggerValidator struct{}

// Validate validates trigger configurations
func (v *TriggerValidator) Validate(ruleSet *ast.RuleSet) error {
	var errors []string

	for i, rule := range ruleSet.Rules {
		if rule.Trigger == nil {
			continue
		}

		trigger := rule.Trigger
		switch trigger.Type {
		case ast.TriggerTypeOnEvent:
			if trigger.Event == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].trigger: OnEvent requires event name", i))
			}
		case ast.TriggerTypeOnChange:
			if len(trigger.Watch) == 0 {
				errors = append(errors, fmt.Sprintf("rule[%d].trigger: OnChange requires watch paths", i))
			}
		case ast.TriggerTypeDistance:
			if trigger.From == "" || trigger.To == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].trigger: Distance requires from and to paths", i))
			}
			if trigger.Value <= 0 {
				errors = append(errors, fmt.Sprintf("rule[%d].trigger: Distance requires positive value", i))
			}
		case ast.TriggerTypeTimer:
			if trigger.Duration <= 0 {
				errors = append(errors, fmt.Sprintf("rule[%d].trigger: Timer requires positive duration", i))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("trigger validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}
	return nil
}

// SelectorValidator validates selector configurations
type SelectorValidator struct{}

// Validate validates selector configurations
func (v *SelectorValidator) Validate(ruleSet *ast.RuleSet) error {
	var errors []string

	for i, rule := range ruleSet.Rules {
		if rule.Selector == nil {
			continue
		}

		selector := rule.Selector
		switch selector.Type {
		case ast.SelectorTypeSingle:
			if selector.ID == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].selector: Single requires id", i))
			}
		case ast.SelectorTypeRelated:
			if selector.From == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].selector: Related requires from", i))
			}
			if selector.Relation == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].selector: Related requires relation", i))
			}
		case ast.SelectorTypeNearest, ast.SelectorTypeFarthest:
			if selector.Origin == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].selector: Nearest/Farthest requires origin", i))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("selector validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}
	return nil
}

// EffectValidator validates effect configurations
type EffectValidator struct{}

// Validate validates effect configurations
func (v *EffectValidator) Validate(ruleSet *ast.RuleSet) error {
	var errors []string

	for i, rule := range ruleSet.Rules {
		for j, effect := range rule.Effects {
			switch effect.Type {
			case ast.EffectTypeSet:
				if effect.Path == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Set requires path", i, j))
				}
				if effect.Value == nil {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Set requires value", i, j))
				}
			case ast.EffectTypeIncrement, ast.EffectTypeDecrement:
				if effect.Path == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Increment/Decrement requires path", i, j))
				}
			case ast.EffectTypeAppend:
				if effect.Path == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Append requires path", i, j))
				}
				if effect.Item == nil {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Append requires item", i, j))
				}
			case ast.EffectTypeEmit:
				if effect.Event == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Emit requires event", i, j))
				}
			case ast.EffectTypeSpawn:
				if effect.Entity == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Spawn requires entity", i, j))
				}
			case ast.EffectTypeTransform:
				if effect.Path == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Transform requires path", i, j))
				}
				if effect.Transform == nil {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Transform requires transform", i, j))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("effect validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}
	return nil
}

// ViewValidator validates view configurations
type ViewValidator struct{}

// Validate validates view configurations
func (v *ViewValidator) Validate(ruleSet *ast.RuleSet) error {
	var errors []string

	for i, rule := range ruleSet.Rules {
		for name, view := range rule.Views {
			if view.Type == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].views.%s: type is required", i, name))
			}

			switch view.Type {
			case ast.ViewTypeField, ast.ViewTypeMax, ast.ViewTypeMin, ast.ViewTypeSum, ast.ViewTypeAvg, ast.ViewTypeDistinct:
				if view.Field == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].views.%s: field is required for %s", i, name, view.Type))
				}
			case ast.ViewTypeGroupBy:
				if view.GroupField == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].views.%s: groupField is required for GroupBy", i, name))
				}
			case ast.ViewTypeSort:
				if view.By == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].views.%s: by is required for Sort", i, name))
				}
			case ast.ViewTypeDistance:
				if view.From == "" || view.To == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].views.%s: from and to are required for Distance", i, name))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("view validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}
	return nil
}

// TransformValidator validates transform configurations
type TransformValidator struct{}

// Validate validates transform configurations
func (v *TransformValidator) Validate(ruleSet *ast.RuleSet) error {
	var errors []string

	for i, rule := range ruleSet.Rules {
		for j, effect := range rule.Effects {
			if effect.Transform != nil {
				if err := validateTransform(effect.Transform); err != nil {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d].transform: %s", i, j, err))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("transform validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}
	return nil
}

// validateTransform validates a single transform
func validateTransform(t *ast.Transform) error {
	switch t.Type {
	case ast.TransformTypeMoveTowards:
		if t.Current == nil || t.Target == nil {
			return fmt.Errorf("MoveTowards requires current and target")
		}
		if t.Speed <= 0 {
			return fmt.Errorf("MoveTowards requires positive speed")
		}
	case ast.TransformTypeGpsDistance:
		if t.From == nil || t.To == nil {
			return fmt.Errorf("GpsDistance requires from and to")
		}
	case ast.TransformTypeClamp:
		if t.Value == nil || t.Min == nil || t.Max == nil {
			return fmt.Errorf("Clamp requires value, min, and max")
		}
	case ast.TransformTypeIf:
		if t.Condition == nil {
			return fmt.Errorf("If requires condition")
		}
	}
	return nil
}

// CompositeValidator combines multiple validators
type CompositeValidator struct {
	validators []Validator
}

// NewCompositeValidator creates a validator that runs multiple validators
func NewCompositeValidator(validators ...Validator) *CompositeValidator {
	return &CompositeValidator{validators: validators}
}

// Validate runs all validators
func (v *CompositeValidator) Validate(ruleSet *ast.RuleSet) error {
	var allErrors []error
	for _, validator := range v.validators {
		if err := validator.Validate(ruleSet); err != nil {
			allErrors = append(allErrors, err)
		}
	}
	if len(allErrors) > 0 {
		return &ValidationErrors{Errors: allErrors}
	}
	return nil
}

// StrictValidator includes all validators
func StrictValidator() Validator {
	return NewCompositeValidator(
		&RequiredFieldsValidator{},
		&PathValidator{},
		&TriggerValidator{},
		&SelectorValidator{},
		&EffectValidator{},
		&ViewValidator{},
		&TransformValidator{},
	)
}
