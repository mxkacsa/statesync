package eval

import (
	"fmt"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// EffectEvaluator applies effects to state.
// Effects are batch operations - the engine handles per-entity iteration.
type EffectEvaluator struct {
	transformEval  *TransformEvaluator
	viewEval       *ViewEvaluator
	ruleController RuleController
}

// NewEffectEvaluator creates a new effect evaluator
func NewEffectEvaluator() *EffectEvaluator {
	return &EffectEvaluator{
		transformEval: NewTransformEvaluator(),
		viewEval:      NewViewEvaluator(),
	}
}

// SetRuleController sets the rule controller for enabling/disabling rules
func (ee *EffectEvaluator) SetRuleController(rc RuleController) {
	ee.ruleController = rc
}

// Apply applies an effect to the state using the registry.
// For batch effects, this resolves targets and iterates internally.
func (ee *EffectEvaluator) Apply(ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	// Look up effect handler in registry
	def, ok := GetEffect(string(effect.Type))
	if !ok {
		return fmt.Errorf("unknown effect type: %s (not registered)", effect.Type)
	}

	// Call the registered function
	return def.Func(ee, ctx, effect, ruleViews)
}

// resolveTargets resolves the targets for an effect from views
func (ee *EffectEvaluator) resolveTargets(ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) ([]interface{}, error) {
	if effect.Targets == nil {
		// No targets = apply to state directly (single operation)
		return []interface{}{ctx.State}, nil
	}

	// Check if targets is a view name
	if viewName, ok := effect.Targets.(string); ok {
		// Look up view in rule views
		if view, exists := ruleViews[viewName]; exists {
			result, err := ee.viewEval.Evaluate(ctx, view, nil)
			if err != nil {
				return nil, fmt.Errorf("view %s: %w", viewName, err)
			}
			entities, ok := toEntitySlice(result)
			if !ok {
				return nil, fmt.Errorf("view %s did not return entity list", viewName)
			}
			return entities, nil
		}
		// Also check if it's already computed in context
		if viewResult, exists := ctx.Views[viewName]; exists {
			entities, ok := toEntitySlice(viewResult)
			if !ok {
				return nil, fmt.Errorf("view %s did not return entity list", viewName)
			}
			return entities, nil
		}
		return nil, fmt.Errorf("view not found: %s", viewName)
	}

	// Check if targets is an inline view definition
	if viewDef, ok := effect.Targets.(map[string]interface{}); ok {
		view := &ast.View{}
		if source, ok := viewDef["source"].(string); ok {
			view.Source = source
		}
		if pipeline, ok := viewDef["pipeline"].([]interface{}); ok {
			// Parse pipeline operations
			for _, opRaw := range pipeline {
				if opMap, ok := opRaw.(map[string]interface{}); ok {
					op := ast.ViewOperation{}
					if t, ok := opMap["type"].(string); ok {
						op.Type = ast.ViewOperationType(t)
					}
					// TODO: Parse other operation fields
					view.Pipeline = append(view.Pipeline, op)
				}
			}
		}
		result, err := ee.viewEval.Evaluate(ctx, view, nil)
		if err != nil {
			return nil, err
		}
		entities, ok := toEntitySlice(result)
		if !ok {
			return nil, fmt.Errorf("inline view did not return entity list")
		}
		return entities, nil
	}

	return nil, fmt.Errorf("invalid targets specification")
}

// Note: Effect implementations are now in effects_builtin.go
// They are registered via the global registry and called through Apply

// evaluateValueExpression evaluates a value expression in entity context
func (ee *EffectEvaluator) evaluateValueExpression(ctx *Context, expr *ast.ValueExpression, ruleViews map[string]*ast.View) (interface{}, error) {
	switch expr.Type {
	case "literal":
		return expr.Literal, nil

	case "field":
		return ctx.Resolve(string(expr.Field))

	case "viewResult":
		// Get the view
		view, ok := ruleViews[expr.View]
		if !ok {
			return nil, fmt.Errorf("view not found: %s", expr.View)
		}

		// Resolve view params with self references
		params := make(map[string]interface{})
		for key, val := range expr.ViewParams {
			resolved, err := ctx.Resolve(val)
			if err != nil {
				return nil, fmt.Errorf("param %s: %w", key, err)
			}
			params[key] = resolved
		}

		// Evaluate view with params
		return ee.viewEval.Evaluate(ctx, view, params)

	case "distance":
		fromVal, err := ctx.Resolve(expr.From)
		if err != nil {
			return nil, fmt.Errorf("from: %w", err)
		}
		toVal, err := ctx.Resolve(expr.To)
		if err != nil {
			return nil, fmt.Errorf("to: %w", err)
		}

		fromPoint, err := toGeoPoint(fromVal)
		if err != nil {
			return nil, fmt.Errorf("from: %w", err)
		}
		toPoint, err := toGeoPoint(toVal)
		if err != nil {
			return nil, fmt.Errorf("to: %w", err)
		}

		return haversineDistance(fromPoint, toPoint), nil

	case "transform":
		if expr.Transform == nil {
			return nil, fmt.Errorf("transform is required")
		}
		return ee.transformEval.Evaluate(ctx, expr.Transform)

	default:
		return nil, fmt.Errorf("unknown value expression type: %s", expr.Type)
	}
}
