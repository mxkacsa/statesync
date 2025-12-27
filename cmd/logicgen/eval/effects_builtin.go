package eval

import (
	"fmt"
	"reflect"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

func init() {
	registerStateEffects()
	registerEventEffects()
	registerControlEffects()
}

// =============================================================================
// State Effects
// =============================================================================

func registerStateEffects() {
	// Set
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeSet),
		Category:    CategoryState,
		Description: "Sets a value on target entities",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
			{Name: "value", Type: "interface{}", Required: true},
			{Name: "targets", Type: "string|View", Required: false},
		},
		Func: effectSet,
	})

	// Increment
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeIncrement),
		Category:    CategoryState,
		Description: "Increments a numeric value on target entities",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
			{Name: "value", Type: "float64", Required: false, Default: 1.0},
			{Name: "targets", Type: "string|View", Required: false},
		},
		Func: effectIncrement,
	})

	// Decrement
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeDecrement),
		Category:    CategoryState,
		Description: "Decrements a numeric value on target entities",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
			{Name: "value", Type: "float64", Required: false, Default: 1.0},
			{Name: "targets", Type: "string|View", Required: false},
		},
		Func: effectDecrement,
	})

	// Transform
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeTransform),
		Category:    CategoryState,
		Description: "Applies a transform to target entities",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
			{Name: "value", Type: "Transform", Required: true},
			{Name: "targets", Type: "string|View", Required: false},
		},
		Func: effectTransform,
	})

	// SetFromView
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeSetFromView),
		Category:    CategoryState,
		Description: "Sets a field from a parameterized view result",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
			{Name: "valueExpression", Type: "ValueExpression", Required: true},
			{Name: "targets", Type: "string|View", Required: false},
		},
		Func: effectSetFromView,
	})

	// Spawn
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeSpawn),
		Category:    CategoryState,
		Description: "Creates a new entity with owner assignment",
		Inputs: []PortDefinition{
			{Name: "entity", Type: "string", Required: true},
			{Name: "fields", Type: "map[string]interface{}", Required: false},
			{Name: "owner", Type: "string", Required: false, Description: "Owner player ID (defaults to sender if not specified)"},
		},
		Func: effectSpawn,
	})

	// Destroy
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeDestroy),
		Category:    CategoryState,
		Description: "Destroys target entities",
		Inputs: []PortDefinition{
			{Name: "targets", Type: "string|View", Required: true},
		},
		Func: effectDestroy,
	})
}

// =============================================================================
// Event Effects
// =============================================================================

func registerEventEffects() {
	// Emit
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeEmit),
		Category:    CategoryEvent,
		Description: "Emits an event",
		Inputs: []PortDefinition{
			{Name: "event", Type: "string", Required: true},
			{Name: "payload", Type: "map[string]interface{}", Required: false},
			{Name: "to", Type: "string", Required: false},
		},
		Func: effectEmit,
	})
}

// =============================================================================
// Control Effects
// =============================================================================

func registerControlEffects() {
	// If
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeIf),
		Category:    CategoryControl,
		Description: "Conditional effect execution",
		Inputs: []PortDefinition{
			{Name: "condition", Type: "interface{}", Required: true},
			{Name: "then", Type: "Effect", Required: false},
			{Name: "else", Type: "Effect", Required: false},
		},
		Func: effectIf,
	})

	// Sequence
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeSequence),
		Category:    CategoryControl,
		Description: "Executes a sequence of effects",
		Inputs: []PortDefinition{
			{Name: "effects", Type: "[]Effect", Required: true},
		},
		Func: effectSequence,
	})

	// EnableRule
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeEnableRule),
		Category:    CategoryControl,
		Description: "Enables a rule by name",
		Inputs: []PortDefinition{
			{Name: "rule", Type: "string", Required: true},
		},
		Func: effectEnableRule,
	})

	// DisableRule
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeDisableRule),
		Category:    CategoryControl,
		Description: "Disables a rule by name",
		Inputs: []PortDefinition{
			{Name: "rule", Type: "string", Required: true},
		},
		Func: effectDisableRule,
	})

	// EnableTrigger
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeEnableTrigger),
		Category:    CategoryControl,
		Description: "Enables a trigger by rule name",
		Inputs: []PortDefinition{
			{Name: "rule", Type: "string", Required: true},
		},
		Func: effectEnableTrigger,
	})

	// DisableTrigger
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeDisableTrigger),
		Category:    CategoryControl,
		Description: "Disables a trigger by rule name",
		Inputs: []PortDefinition{
			{Name: "rule", Type: "string", Required: true},
		},
		Func: effectDisableTrigger,
	})

	// ResetTimer
	RegisterEffect(&EffectDefinition{
		Name:        string(ast.EffectTypeResetTimer),
		Category:    CategoryControl,
		Description: "Resets the timer for a rule",
		Inputs: []PortDefinition{
			{Name: "rule", Type: "string", Required: true},
		},
		Func: effectResetTimer,
	})
}

// =============================================================================
// Effect Implementations
// =============================================================================

func effectSet(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	targets, err := ee.resolveTargets(ctx, effect, ruleViews)
	if err != nil {
		return err
	}

	for i, target := range targets {
		entityCtx := ctx.WithEntity(target, i)

		value, err := entityCtx.Resolve(effect.Value)
		if err != nil {
			return fmt.Errorf("resolve value for entity %d: %w", i, err)
		}

		if err := entityCtx.SetPath(effect.Path, value); err != nil {
			return fmt.Errorf("set path for entity %d: %w", i, err)
		}
	}

	return nil
}

func effectIncrement(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	targets, err := ee.resolveTargets(ctx, effect, ruleViews)
	if err != nil {
		return err
	}

	for i, target := range targets {
		entityCtx := ctx.WithEntity(target, i)

		current, err := entityCtx.ResolvePath(effect.Path)
		if err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}

		increment, err := entityCtx.Resolve(effect.Value)
		if err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}

		currentNum, ok := toFloat64(current)
		if !ok {
			return fmt.Errorf("entity %d: cannot increment non-numeric value", i)
		}

		incrementNum, ok := toFloat64(increment)
		if !ok {
			return fmt.Errorf("entity %d: increment value must be numeric", i)
		}

		if err := entityCtx.SetPath(effect.Path, currentNum+incrementNum); err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}
	}

	return nil
}

func effectDecrement(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	targets, err := ee.resolveTargets(ctx, effect, ruleViews)
	if err != nil {
		return err
	}

	for i, target := range targets {
		entityCtx := ctx.WithEntity(target, i)

		current, err := entityCtx.ResolvePath(effect.Path)
		if err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}

		decrement, err := entityCtx.Resolve(effect.Value)
		if err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}

		currentNum, ok := toFloat64(current)
		if !ok {
			return fmt.Errorf("entity %d: cannot decrement non-numeric value", i)
		}

		decrementNum, ok := toFloat64(decrement)
		if !ok {
			return fmt.Errorf("entity %d: decrement value must be numeric", i)
		}

		if err := entityCtx.SetPath(effect.Path, currentNum-decrementNum); err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}
	}

	return nil
}

func effectTransform(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	targets, err := ee.resolveTargets(ctx, effect, ruleViews)
	if err != nil {
		return err
	}

	for i, target := range targets {
		entityCtx := ctx.WithEntity(target, i)

		transform, ok := effect.Value.(*ast.Transform)
		if !ok {
			if transformMap, ok := effect.Value.(map[string]interface{}); ok {
				result, err := entityCtx.Resolve(transformMap)
				if err != nil {
					return fmt.Errorf("entity %d: %w", i, err)
				}
				if err := entityCtx.SetPath(effect.Path, result); err != nil {
					return fmt.Errorf("entity %d: %w", i, err)
				}
				continue
			}
			return fmt.Errorf("entity %d: invalid transform", i)
		}

		result, err := ee.transformEval.Evaluate(entityCtx, transform)
		if err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}

		if err := entityCtx.SetPath(effect.Path, result); err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}
	}

	return nil
}

func effectSetFromView(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	if effect.ValueExpression == nil {
		return fmt.Errorf("valueExpression is required for SetFromView")
	}

	targets, err := ee.resolveTargets(ctx, effect, ruleViews)
	if err != nil {
		return err
	}

	for i, target := range targets {
		entityCtx := ctx.WithEntity(target, i)

		value, err := ee.evaluateValueExpression(entityCtx, effect.ValueExpression, ruleViews)
		if err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}

		if err := entityCtx.SetPath(effect.Path, value); err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}
	}

	return nil
}

func effectSpawn(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	fields := make(map[string]interface{})
	for k, v := range effect.Fields {
		resolved, err := ctx.Resolve(v)
		if err != nil {
			return fmt.Errorf("field %s: %w", k, err)
		}
		fields[k] = resolved
	}

	// If owner field is not explicitly set, try to set it from sender
	if ctx.PermissionChecker != nil && ctx.SenderID != "" {
		schema := ctx.PermissionChecker.schema
		if schema != nil {
			// Look up the entity type in the permission schema
			if typePerm, ok := schema.Types[effect.Entity]; ok && typePerm.OwnerField != "" {
				// Only set owner if not already specified in fields
				if _, hasOwner := fields[typePerm.OwnerField]; !hasOwner {
					fields[typePerm.OwnerField] = ctx.SenderID
				}
			}
		}
	}

	// TODO: Integrate with actual entity creation system
	// The entity type and initialized fields are ready for use
	_ = effect.Entity
	_ = fields

	return nil
}

func effectDestroy(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	// TODO: Integrate with actual entity destruction system
	_, err := ee.resolveTargets(ctx, effect, ruleViews)
	if err != nil {
		return err
	}

	return nil
}

func effectEmit(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	payload := make(map[string]interface{})
	for k, v := range effect.Payload {
		resolved, err := ctx.Resolve(v)
		if err != nil {
			return fmt.Errorf("payload %s: %w", k, err)
		}
		payload[k] = resolved
	}

	// TODO: Integrate with actual event emission system
	_ = effect.Event
	_ = effect.To
	_ = payload

	return nil
}

func effectIf(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	condition, err := ctx.Resolve(effect.Condition)
	if err != nil {
		return err
	}

	var condResult bool
	switch v := condition.(type) {
	case bool:
		condResult = v
	case *ast.Expression:
		condResult, err = evaluateExpression(ctx, v)
		if err != nil {
			return err
		}
	default:
		condResult = condition != nil && condition != false && condition != 0 && condition != ""
	}

	if condResult {
		if effect.Then != nil {
			return ee.Apply(ctx, effect.Then, ruleViews)
		}
	} else {
		if effect.Else != nil {
			return ee.Apply(ctx, effect.Else, ruleViews)
		}
	}

	return nil
}

func effectSequence(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	for _, subEffect := range effect.Effects {
		if err := ee.Apply(ctx, subEffect, ruleViews); err != nil {
			return err
		}
	}
	return nil
}

func effectEnableRule(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	if ee.ruleController == nil {
		return fmt.Errorf("rule controller not set")
	}
	if effect.Rule == "" {
		return fmt.Errorf("rule name is required for EnableRule effect")
	}
	if !ee.ruleController.EnableRule(effect.Rule) {
		return fmt.Errorf("rule not found: %s", effect.Rule)
	}
	return nil
}

func effectDisableRule(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	if ee.ruleController == nil {
		return fmt.Errorf("rule controller not set")
	}
	if effect.Rule == "" {
		return fmt.Errorf("rule name is required for DisableRule effect")
	}
	if !ee.ruleController.DisableRule(effect.Rule) {
		return fmt.Errorf("rule not found: %s", effect.Rule)
	}
	return nil
}

func effectEnableTrigger(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	if ee.ruleController == nil {
		return fmt.Errorf("rule controller not set")
	}
	if effect.Rule == "" {
		return fmt.Errorf("rule name is required for EnableTrigger effect")
	}
	if !ee.ruleController.EnableTrigger(effect.Rule) {
		return fmt.Errorf("trigger not found for rule: %s", effect.Rule)
	}
	return nil
}

func effectDisableTrigger(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	if ee.ruleController == nil {
		return fmt.Errorf("rule controller not set")
	}
	if effect.Rule == "" {
		return fmt.Errorf("rule name is required for DisableTrigger effect")
	}
	if !ee.ruleController.DisableTrigger(effect.Rule) {
		return fmt.Errorf("trigger not found for rule: %s", effect.Rule)
	}
	return nil
}

func effectResetTimer(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error {
	if ee.ruleController == nil {
		return fmt.Errorf("rule controller not set")
	}
	if effect.Rule == "" {
		return fmt.Errorf("rule name is required for ResetTimer effect")
	}
	ee.ruleController.ResetTimer(effect.Rule)
	return nil
}

// effectMatchesWhere checks if an entity matches a where clause (kept for compatibility)
func effectMatchesWhere(ctx *Context, entity interface{}, where *ast.WhereClause) (bool, error) {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return false, nil
	}

	field := val.FieldByName(where.Field)
	if !field.IsValid() || !field.CanInterface() {
		return false, nil
	}

	compareVal, err := ctx.Resolve(where.Value)
	if err != nil {
		return false, err
	}

	return compare(field.Interface(), compareVal, where.Op)
}
