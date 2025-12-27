package eval

import (
	"fmt"
	"reflect"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// ViewEvaluator evaluates views using a pipeline approach.
// Views are pure functions - no side effects allowed.
type ViewEvaluator struct{}

// NewViewEvaluator creates a new view evaluator
func NewViewEvaluator() *ViewEvaluator {
	return &ViewEvaluator{}
}

// Evaluate evaluates a view and returns the result.
// The result is either a list of entities or a scalar value (for aggregations).
func (ve *ViewEvaluator) Evaluate(ctx *Context, view *ast.View, viewParams map[string]interface{}) (interface{}, error) {
	// Get source entities from state
	entities, err := ve.getSourceEntities(ctx, view.Source)
	if err != nil {
		return nil, fmt.Errorf("source %s: %w", view.Source, err)
	}

	// Merge view params into context params
	if viewParams != nil {
		for k, v := range viewParams {
			ctx.Params[k] = v
		}
	}

	// Apply pipeline operations in order
	var result interface{} = entities
	for i, op := range view.Pipeline {
		result, err = ve.applyOperation(ctx, op, result)
		if err != nil {
			return nil, fmt.Errorf("pipeline step %d (%s): %w", i, op.Type, err)
		}
	}

	return result, nil
}

// getSourceEntities gets entities from the state by collection name
func (ve *ViewEvaluator) getSourceEntities(ctx *Context, source string) ([]interface{}, error) {
	stateVal := ctx.GetStateValue()
	if stateVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("state must be a struct")
	}

	field := stateVal.FieldByName(source)
	if !field.IsValid() {
		return nil, fmt.Errorf("source collection not found: %s", source)
	}

	// Convert to slice of interfaces
	if field.Kind() != reflect.Slice && field.Kind() != reflect.Array {
		return nil, fmt.Errorf("source must be a slice or array: %s", source)
	}

	entities := make([]interface{}, field.Len())
	for i := 0; i < field.Len(); i++ {
		entities[i] = field.Index(i).Interface()
	}

	return entities, nil
}

// applyOperation applies a single pipeline operation using the registry
func (ve *ViewEvaluator) applyOperation(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	// Look up operation in registry
	def, ok := GetViewOp(string(op.Type))
	if !ok {
		return nil, fmt.Errorf("unknown operation type: %s (not registered)", op.Type)
	}

	// Call the registered function
	return def.Func(ctx, op, input)
}

// Note: View operation implementations are now in viewops_builtin.go
// They are registered via the global registry and called through applyOperation
