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

		// Validate view pipeline paths
		for name, view := range rule.Views {
			for k, op := range view.Pipeline {
				if op.Field != "" {
					if err := validatePath(string(op.Field)); err != nil {
						errors = append(errors, fmt.Sprintf("rule[%d].views.%s.pipeline[%d].field: %s", i, name, k, err))
					}
				}
				if op.By != "" {
					if err := validatePath(string(op.By)); err != nil {
						errors = append(errors, fmt.Sprintf("rule[%d].views.%s.pipeline[%d].by: %s", i, name, k, err))
					}
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

	// Path must start with $ or be a reference or self, or be a simple field name
	if !strings.HasPrefix(path, "$") &&
		!strings.HasPrefix(path, "param:") &&
		!strings.HasPrefix(path, "view:") &&
		!strings.HasPrefix(path, "const:") &&
		!strings.HasPrefix(path, "state:") &&
		!strings.HasPrefix(path, "self.") &&
		path != "self" &&
		!isSimpleFieldName(path) {
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

// isSimpleFieldName checks if a path is a simple field name (letters, digits, underscore)
func isSimpleFieldName(path string) bool {
	if len(path) == 0 {
		return false
	}
	for _, ch := range path {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	return true
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
			if trigger.Value < 0 {
				errors = append(errors, fmt.Sprintf("rule[%d].trigger: Distance value cannot be negative", i))
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
			case ast.EffectTypeSetFromView:
				if effect.Path == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: SetFromView requires path", i, j))
				}
				if effect.ValueExpression == nil {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: SetFromView requires valueExpression", i, j))
				}
			case ast.EffectTypeEmit:
				if effect.Event == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Emit requires event", i, j))
				}
			case ast.EffectTypeSpawn:
				if effect.Entity == "" {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d]: Spawn requires entity", i, j))
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
			if view.Source == "" {
				errors = append(errors, fmt.Sprintf("rule[%d].views.%s: source is required", i, name))
			}

			// Validate pipeline operations
			for k, op := range view.Pipeline {
				switch op.Type {
				case ast.ViewOpFilter:
					if op.Where == nil {
						errors = append(errors, fmt.Sprintf("rule[%d].views.%s.pipeline[%d]: Filter requires where clause", i, name, k))
					}
				case ast.ViewOpOrderBy:
					if op.By == "" {
						errors = append(errors, fmt.Sprintf("rule[%d].views.%s.pipeline[%d]: OrderBy requires by field", i, name, k))
					}
				case ast.ViewOpGroupBy:
					if op.GroupField == "" {
						errors = append(errors, fmt.Sprintf("rule[%d].views.%s.pipeline[%d]: GroupBy requires groupField", i, name, k))
					}
				case ast.ViewOpMin, ast.ViewOpMax, ast.ViewOpSum, ast.ViewOpAvg:
					if op.Field == "" {
						errors = append(errors, fmt.Sprintf("rule[%d].views.%s.pipeline[%d]: %s requires field", i, name, k, op.Type))
					}
				case ast.ViewOpNearest, ast.ViewOpFarthest:
					if op.Origin == nil {
						errors = append(errors, fmt.Sprintf("rule[%d].views.%s.pipeline[%d]: %s requires origin", i, name, k, op.Type))
					}
					if op.Position == "" {
						errors = append(errors, fmt.Sprintf("rule[%d].views.%s.pipeline[%d]: %s requires position field", i, name, k, op.Type))
					}
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
			// Check transform in Value field
			if effect.Value != nil {
				if transform, ok := effect.Value.(*ast.Transform); ok {
					if err := validateTransform(transform); err != nil {
						errors = append(errors, fmt.Sprintf("rule[%d].effects[%d].value: %s", i, j, err))
					}
				}
			}
			// Check transform in ValueExpression field
			if effect.ValueExpression != nil && effect.ValueExpression.Transform != nil {
				if err := validateTransform(effect.ValueExpression.Transform); err != nil {
					errors = append(errors, fmt.Sprintf("rule[%d].effects[%d].valueExpression.transform: %s", i, j, err))
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

// ValueExpressionValidator validates view parameter usage in effects
type ValueExpressionValidator struct{}

// Validate validates that effects provide required view parameters
func (v *ValueExpressionValidator) Validate(ruleSet *ast.RuleSet) error {
	var errors []string

	for i, rule := range ruleSet.Rules {
		// Build map of views with their required params
		viewRequiredParams := make(map[string][]string)
		for name, view := range rule.Views {
			for paramName, paramDef := range view.Params {
				if paramDef.Default == nil {
					viewRequiredParams[name] = append(viewRequiredParams[name], paramName)
				}
			}
		}

		// Check effects that use ValueExpression with views
		for j, effect := range rule.Effects {
			if effect.ValueExpression == nil {
				continue
			}
			if effect.ValueExpression.View == "" {
				continue
			}

			viewName := effect.ValueExpression.View
			requiredParams, exists := viewRequiredParams[viewName]
			if !exists {
				// View not found in this rule - might be external
				continue
			}

			// Check that all required params are provided
			for _, requiredParam := range requiredParams {
				if effect.ValueExpression.ViewParams == nil {
					errors = append(errors, fmt.Sprintf(
						"rule[%d].effects[%d]: view %q requires parameter %q but no viewParams provided",
						i, j, viewName, requiredParam))
					continue
				}
				if _, provided := effect.ValueExpression.ViewParams[requiredParam]; !provided {
					errors = append(errors, fmt.Sprintf(
						"rule[%d].effects[%d]: view %q requires parameter %q",
						i, j, viewName, requiredParam))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("value expression validation failed:\n  - %s", strings.Join(errors, "\n  - "))
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
		&EffectValidator{},
		&ViewValidator{},
		&TransformValidator{},
		&ValueExpressionValidator{},
	)
}
