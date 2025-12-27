package eval

import (
	"fmt"
	"reflect"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// FilterEvaluator evaluates filter definitions
type FilterEvaluator struct{}

// NewFilterEvaluator creates a new filter evaluator
func NewFilterEvaluator() *FilterEvaluator {
	return &FilterEvaluator{}
}

// FilterContext holds the context for filter evaluation
type FilterContext struct {
	State  interface{}            // The state being filtered
	Params map[string]interface{} // Filter parameters (viewerTeam, etc.)
}

// NewFilterContext creates a new filter context
func NewFilterContext(state interface{}, params map[string]interface{}) *FilterContext {
	return &FilterContext{
		State:  state,
		Params: params,
	}
}

// Apply executes a filter and returns the filtered state
func (fe *FilterEvaluator) Apply(filter *ast.Filter, state interface{}, params map[string]interface{}) (interface{}, error) {
	ctx := NewFilterContext(state, params)

	for _, op := range filter.Operations {
		if err := fe.executeOperation(ctx, &op); err != nil {
			return nil, fmt.Errorf("filter %s: %w", filter.Name, err)
		}
	}

	return ctx.State, nil
}

// executeOperation executes a single filter operation
func (fe *FilterEvaluator) executeOperation(ctx *FilterContext, op *ast.FilterOperation) error {
	switch op.Type {
	case ast.FilterOpKeepWhere, ast.FilterOpFilterArray:
		return fe.keepWhere(ctx, op)
	case ast.FilterOpRemoveWhere:
		return fe.removeWhere(ctx, op)
	case ast.FilterOpHideFieldsWhere:
		return fe.hideFieldsWhere(ctx, op)
	case ast.FilterOpReplaceFieldWhere:
		return fe.replaceFieldWhere(ctx, op)
	default:
		return fmt.Errorf("unknown filter operation: %s", op.Type)
	}
}

// keepWhere keeps only array items matching the condition
func (fe *FilterEvaluator) keepWhere(ctx *FilterContext, op *ast.FilterOperation) error {
	arr, err := fe.getArrayValue(ctx, op.Target)
	if err != nil {
		return err
	}

	if !arr.IsValid() || op.Where == nil {
		return nil
	}

	filtered := fe.filterSlice(arr, op.Where, ctx, true)
	return fe.setArrayValue(ctx, op.Target, filtered)
}

// removeWhere removes array items matching the condition
func (fe *FilterEvaluator) removeWhere(ctx *FilterContext, op *ast.FilterOperation) error {
	arr, err := fe.getArrayValue(ctx, op.Target)
	if err != nil {
		return err
	}

	if !arr.IsValid() || op.Where == nil {
		return nil
	}

	filtered := fe.filterSlice(arr, op.Where, ctx, false)
	return fe.setArrayValue(ctx, op.Target, filtered)
}

// hideFieldsWhere sets fields to zero on items matching the condition
func (fe *FilterEvaluator) hideFieldsWhere(ctx *FilterContext, op *ast.FilterOperation) error {
	arr, err := fe.getArrayValue(ctx, op.Target)
	if err != nil {
		return err
	}

	if !arr.IsValid() {
		return nil
	}

	arrVal := arr
	if arrVal.Kind() == reflect.Ptr {
		arrVal = arrVal.Elem()
	}
	if arrVal.Kind() != reflect.Slice {
		return nil
	}

	for i := 0; i < arrVal.Len(); i++ {
		item := arrVal.Index(i)
		if op.Where == nil || fe.matchesCondition(item, op.Where, ctx) {
			fe.zeroFields(item, op.Fields)
		}
	}

	return nil
}

// replaceFieldWhere replaces a field value on items matching the condition
func (fe *FilterEvaluator) replaceFieldWhere(ctx *FilterContext, op *ast.FilterOperation) error {
	arr, err := fe.getArrayValue(ctx, op.Target)
	if err != nil {
		return err
	}

	if !arr.IsValid() {
		return nil
	}

	arrVal := arr
	if arrVal.Kind() == reflect.Ptr {
		arrVal = arrVal.Elem()
	}
	if arrVal.Kind() != reflect.Slice {
		return nil
	}

	newVal := fe.resolveValue(op.Value, ctx)

	for i := 0; i < arrVal.Len(); i++ {
		item := arrVal.Index(i)
		if op.Where == nil || fe.matchesCondition(item, op.Where, ctx) {
			fe.setField(item, op.Field, newVal)
		}
	}

	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// filterSlice filters a slice based on condition
func (fe *FilterEvaluator) filterSlice(arr reflect.Value, where *ast.WhereClause, ctx *FilterContext, keepMatching bool) reflect.Value {
	if arr.Kind() == reflect.Ptr {
		arr = arr.Elem()
	}
	if arr.Kind() != reflect.Slice {
		return arr
	}

	result := reflect.MakeSlice(arr.Type(), 0, arr.Len())

	for i := 0; i < arr.Len(); i++ {
		item := arr.Index(i)
		matches := fe.matchesCondition(item, where, ctx)
		if matches == keepMatching {
			result = reflect.Append(result, item)
		}
	}

	return result
}

// matchesCondition checks if an item matches the where clause
// Uses View's WhereClause - same logic as view filter
func (fe *FilterEvaluator) matchesCondition(item reflect.Value, cond *ast.WhereClause, ctx *FilterContext) bool {
	if cond == nil {
		return true
	}

	// Logical AND
	if len(cond.And) > 0 {
		for _, sub := range cond.And {
			if !fe.matchesCondition(item, sub, ctx) {
				return false
			}
		}
		return true
	}

	// Logical OR
	if len(cond.Or) > 0 {
		for _, sub := range cond.Or {
			if fe.matchesCondition(item, sub, ctx) {
				return true
			}
		}
		return false
	}

	// Logical NOT
	if cond.Not != nil {
		return !fe.matchesCondition(item, cond.Not, ctx)
	}

	// Simple field comparison
	fieldVal := fe.getFieldValue(item, cond.Field)
	expectedVal := fe.resolveValue(cond.Value, ctx)

	result, _ := compare(fieldVal, expectedVal, cond.Op)
	return result
}

// getFieldValue gets a field value from a struct
func (fe *FilterEvaluator) getFieldValue(item reflect.Value, fieldName string) interface{} {
	if item.Kind() == reflect.Ptr {
		item = item.Elem()
	}
	if item.Kind() != reflect.Struct {
		return nil
	}

	field := item.FieldByName(fieldName)
	if !field.IsValid() {
		return nil
	}

	return field.Interface()
}

// resolveValue resolves parameter references like "param:viewerTeam"
func (fe *FilterEvaluator) resolveValue(val interface{}, ctx *FilterContext) interface{} {
	if str, ok := val.(string); ok {
		// Check for param reference: "param:viewerTeam"
		if len(str) > 6 && str[:6] == "param:" {
			paramName := str[6:]
			if paramVal, ok := ctx.Params[paramName]; ok {
				return paramVal
			}
		}
	}
	return val
}

// zeroFields sets specified fields to their zero value
func (fe *FilterEvaluator) zeroFields(item reflect.Value, fields []string) {
	if item.Kind() == reflect.Ptr {
		item = item.Elem()
	}
	if item.Kind() != reflect.Struct || !item.CanAddr() {
		return
	}

	for _, fieldName := range fields {
		field := item.FieldByName(fieldName)
		if field.IsValid() && field.CanSet() {
			field.Set(reflect.Zero(field.Type()))
		}
	}
}

// setField sets a field to a new value
func (fe *FilterEvaluator) setField(item reflect.Value, fieldName string, value interface{}) {
	if item.Kind() == reflect.Ptr {
		item = item.Elem()
	}
	if item.Kind() != reflect.Struct || !item.CanAddr() {
		return
	}

	field := item.FieldByName(fieldName)
	if !field.IsValid() || !field.CanSet() {
		return
	}

	if value != nil {
		field.Set(reflect.ValueOf(value))
	} else {
		field.Set(reflect.Zero(field.Type()))
	}
}

// getArrayValue gets an array from state by path
func (fe *FilterEvaluator) getArrayValue(ctx *FilterContext, path string) (reflect.Value, error) {
	if len(path) < 3 || path[:2] != "$." {
		return reflect.Value{}, fmt.Errorf("invalid path: %s", path)
	}

	fieldName := path[2:]
	stateVal := reflect.ValueOf(ctx.State)
	if stateVal.Kind() == reflect.Ptr {
		stateVal = stateVal.Elem()
	}

	field := stateVal.FieldByName(fieldName)
	if !field.IsValid() {
		return reflect.Value{}, fmt.Errorf("field not found: %s", fieldName)
	}

	return field, nil
}

// setArrayValue sets an array in state by path
func (fe *FilterEvaluator) setArrayValue(ctx *FilterContext, path string, value reflect.Value) error {
	if len(path) < 3 || path[:2] != "$." {
		return fmt.Errorf("invalid path: %s", path)
	}

	fieldName := path[2:]
	stateVal := reflect.ValueOf(ctx.State)
	if stateVal.Kind() == reflect.Ptr {
		stateVal = stateVal.Elem()
	}

	field := stateVal.FieldByName(fieldName)
	if !field.IsValid() || !field.CanSet() {
		return fmt.Errorf("cannot set field: %s", fieldName)
	}

	field.Set(value)
	return nil
}
