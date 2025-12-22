package eval

import (
	"fmt"
	"reflect"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// EffectEvaluator applies effects to state
type EffectEvaluator struct {
	transformEval  *TransformEvaluator
	ruleController RuleController
}

// NewEffectEvaluator creates a new effect evaluator
func NewEffectEvaluator() *EffectEvaluator {
	return &EffectEvaluator{
		transformEval: NewTransformEvaluator(),
	}
}

// SetRuleController sets the rule controller for enabling/disabling rules
func (ee *EffectEvaluator) SetRuleController(rc RuleController) {
	ee.ruleController = rc
}

// Apply applies an effect to the state
func (ee *EffectEvaluator) Apply(ctx *Context, effect *ast.Effect) error {
	switch effect.Type {
	case ast.EffectTypeSet:
		return ee.applySet(ctx, effect)
	case ast.EffectTypeIncrement:
		return ee.applyIncrement(ctx, effect)
	case ast.EffectTypeDecrement:
		return ee.applyDecrement(ctx, effect)
	case ast.EffectTypeAppend:
		return ee.applyAppend(ctx, effect)
	case ast.EffectTypeRemove:
		return ee.applyRemove(ctx, effect)
	case ast.EffectTypeClear:
		return ee.applyClear(ctx, effect)
	case ast.EffectTypeTransform:
		return ee.applyTransform(ctx, effect)
	case ast.EffectTypeEmit:
		return ee.applyEmit(ctx, effect)
	case ast.EffectTypeSpawn:
		return ee.applySpawn(ctx, effect)
	case ast.EffectTypeDestroy:
		return ee.applyDestroy(ctx, effect)
	case ast.EffectTypeIf:
		return ee.applyIf(ctx, effect)
	case ast.EffectTypeSequence:
		return ee.applySequence(ctx, effect)
	case ast.EffectTypeEnableRule:
		return ee.applyEnableRule(ctx, effect)
	case ast.EffectTypeDisableRule:
		return ee.applyDisableRule(ctx, effect)
	case ast.EffectTypeEnableTrigger:
		return ee.applyEnableTrigger(ctx, effect)
	case ast.EffectTypeDisableTrigger:
		return ee.applyDisableTrigger(ctx, effect)
	case ast.EffectTypeResetTimer:
		return ee.applyResetTimer(ctx, effect)
	default:
		return fmt.Errorf("unknown effect type: %s", effect.Type)
	}
}

// applySet sets a value at a path
func (ee *EffectEvaluator) applySet(ctx *Context, effect *ast.Effect) error {
	value, err := ctx.Resolve(effect.Value)
	if err != nil {
		return fmt.Errorf("resolve value: %w", err)
	}

	return ctx.SetPath(effect.Path, value)
}

// applyIncrement increments a numeric value
func (ee *EffectEvaluator) applyIncrement(ctx *Context, effect *ast.Effect) error {
	current, err := ctx.ResolvePath(effect.Path)
	if err != nil {
		return err
	}

	increment, err := ctx.Resolve(effect.Value)
	if err != nil {
		return err
	}

	currentNum, ok := toFloat64(current)
	if !ok {
		return fmt.Errorf("cannot increment non-numeric value")
	}

	incrementNum, ok := toFloat64(increment)
	if !ok {
		return fmt.Errorf("increment value must be numeric")
	}

	return ctx.SetPath(effect.Path, currentNum+incrementNum)
}

// applyDecrement decrements a numeric value
func (ee *EffectEvaluator) applyDecrement(ctx *Context, effect *ast.Effect) error {
	current, err := ctx.ResolvePath(effect.Path)
	if err != nil {
		return err
	}

	decrement, err := ctx.Resolve(effect.Value)
	if err != nil {
		return err
	}

	currentNum, ok := toFloat64(current)
	if !ok {
		return fmt.Errorf("cannot decrement non-numeric value")
	}

	decrementNum, ok := toFloat64(decrement)
	if !ok {
		return fmt.Errorf("decrement value must be numeric")
	}

	return ctx.SetPath(effect.Path, currentNum-decrementNum)
}

// applyAppend appends an item to an array
func (ee *EffectEvaluator) applyAppend(ctx *Context, effect *ast.Effect) error {
	current, err := ctx.ResolvePath(effect.Path)
	if err != nil {
		return err
	}

	item, err := ctx.Resolve(effect.Item)
	if err != nil {
		return err
	}

	// Get current slice and append
	currentVal := reflect.ValueOf(current)
	if currentVal.Kind() != reflect.Slice {
		return fmt.Errorf("cannot append to non-slice")
	}

	itemVal := reflect.ValueOf(item)
	if !itemVal.Type().AssignableTo(currentVal.Type().Elem()) {
		if itemVal.Type().ConvertibleTo(currentVal.Type().Elem()) {
			itemVal = itemVal.Convert(currentVal.Type().Elem())
		} else {
			return fmt.Errorf("item type mismatch")
		}
	}

	newSlice := reflect.Append(currentVal, itemVal)
	return ctx.SetPath(effect.Path, newSlice.Interface())
}

// applyRemove removes an item from an array
func (ee *EffectEvaluator) applyRemove(ctx *Context, effect *ast.Effect) error {
	current, err := ctx.ResolvePath(effect.Path)
	if err != nil {
		return err
	}

	currentVal := reflect.ValueOf(current)
	if currentVal.Kind() != reflect.Slice {
		return fmt.Errorf("cannot remove from non-slice")
	}

	// Remove by index
	if effect.Index != nil {
		index, err := ctx.Resolve(effect.Index)
		if err != nil {
			return err
		}
		idx, ok := toInt(index)
		if !ok {
			return fmt.Errorf("index must be integer")
		}
		if idx < 0 || idx >= currentVal.Len() {
			return fmt.Errorf("index out of bounds")
		}

		newSlice := reflect.AppendSlice(
			currentVal.Slice(0, idx),
			currentVal.Slice(idx+1, currentVal.Len()),
		)
		return ctx.SetPath(effect.Path, newSlice.Interface())
	}

	// Remove by condition
	if effect.Where != nil {
		var keepIndices []int
		for i := 0; i < currentVal.Len(); i++ {
			elem := currentVal.Index(i).Interface()
			matches, err := matchesWhere(ctx, elem, effect.Where)
			if err != nil {
				return err
			}
			if !matches {
				keepIndices = append(keepIndices, i)
			}
		}

		newSlice := reflect.MakeSlice(currentVal.Type(), len(keepIndices), len(keepIndices))
		for i, idx := range keepIndices {
			newSlice.Index(i).Set(currentVal.Index(idx))
		}
		return ctx.SetPath(effect.Path, newSlice.Interface())
	}

	return nil
}

// applyClear clears an array or map
func (ee *EffectEvaluator) applyClear(ctx *Context, effect *ast.Effect) error {
	current, err := ctx.ResolvePath(effect.Path)
	if err != nil {
		return err
	}

	currentVal := reflect.ValueOf(current)
	switch currentVal.Kind() {
	case reflect.Slice:
		newSlice := reflect.MakeSlice(currentVal.Type(), 0, 0)
		return ctx.SetPath(effect.Path, newSlice.Interface())
	case reflect.Map:
		newMap := reflect.MakeMap(currentVal.Type())
		return ctx.SetPath(effect.Path, newMap.Interface())
	default:
		return fmt.Errorf("cannot clear %s", currentVal.Kind())
	}
}

// applyTransform applies a transform and saves the result
func (ee *EffectEvaluator) applyTransform(ctx *Context, effect *ast.Effect) error {
	if effect.Transform == nil {
		return fmt.Errorf("transform is required")
	}

	result, err := ee.transformEval.Evaluate(ctx, effect.Transform)
	if err != nil {
		return fmt.Errorf("transform: %w", err)
	}

	return ctx.SetPath(effect.Path, result)
}

// applyEmit emits an event (placeholder - needs event system integration)
func (ee *EffectEvaluator) applyEmit(ctx *Context, effect *ast.Effect) error {
	// Build payload
	payload := make(map[string]interface{})
	for k, v := range effect.Payload {
		resolved, err := ctx.Resolve(v)
		if err != nil {
			return fmt.Errorf("payload %s: %w", k, err)
		}
		payload[k] = resolved
	}

	// TODO: Integrate with actual event emission system
	// For now, just log or store the event
	_ = payload
	_ = effect.Event
	_ = effect.To

	return nil
}

// applySpawn creates a new entity (placeholder - needs entity creation integration)
func (ee *EffectEvaluator) applySpawn(ctx *Context, effect *ast.Effect) error {
	// Build entity fields
	fields := make(map[string]interface{})
	for k, v := range effect.Fields {
		resolved, err := ctx.Resolve(v)
		if err != nil {
			return fmt.Errorf("field %s: %w", k, err)
		}
		fields[k] = resolved
	}

	// TODO: Integrate with actual entity creation
	_ = effect.Entity
	_ = fields

	return nil
}

// applyDestroy destroys entities (placeholder)
func (ee *EffectEvaluator) applyDestroy(ctx *Context, effect *ast.Effect) error {
	// TODO: Integrate with actual entity destruction
	return nil
}

// applyIf applies a conditional effect
func (ee *EffectEvaluator) applyIf(ctx *Context, effect *ast.Effect) error {
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
		// Truthy check
		condResult = condition != nil && condition != false && condition != 0 && condition != ""
	}

	if condResult {
		if effect.Then != nil {
			return ee.Apply(ctx, effect.Then)
		}
	} else {
		if effect.Else != nil {
			return ee.Apply(ctx, effect.Else)
		}
	}

	return nil
}

// applySequence applies a sequence of effects
func (ee *EffectEvaluator) applySequence(ctx *Context, effect *ast.Effect) error {
	for _, subEffect := range effect.Effects {
		if err := ee.Apply(ctx, subEffect); err != nil {
			return err
		}
	}
	return nil
}

// matchesWhere checks if an entity matches a where clause
func matchesWhere(ctx *Context, entity interface{}, where *ast.WhereClause) (bool, error) {
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

// toInt converts a value to int
func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	default:
		return 0, false
	}
}

// applyEnableRule enables a rule by name
func (ee *EffectEvaluator) applyEnableRule(ctx *Context, effect *ast.Effect) error {
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

// applyDisableRule disables a rule by name
func (ee *EffectEvaluator) applyDisableRule(ctx *Context, effect *ast.Effect) error {
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

// applyEnableTrigger enables a trigger by rule name
func (ee *EffectEvaluator) applyEnableTrigger(ctx *Context, effect *ast.Effect) error {
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

// applyDisableTrigger disables a trigger by rule name
func (ee *EffectEvaluator) applyDisableTrigger(ctx *Context, effect *ast.Effect) error {
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

// applyResetTimer resets the timer for a rule
func (ee *EffectEvaluator) applyResetTimer(ctx *Context, effect *ast.Effect) error {
	if ee.ruleController == nil {
		return fmt.Errorf("rule controller not set")
	}
	if effect.Rule == "" {
		return fmt.Errorf("rule name is required for ResetTimer effect")
	}
	ee.ruleController.ResetTimer(effect.Rule)
	return nil
}
