package eval

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// TransformEvaluator evaluates transforms to produce values
type TransformEvaluator struct {
	rng *rand.Rand
}

// NewTransformEvaluator creates a new transform evaluator
func NewTransformEvaluator() *TransformEvaluator {
	return &TransformEvaluator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Evaluate evaluates a transform using the registry and returns the result
func (te *TransformEvaluator) Evaluate(ctx *Context, transform *ast.Transform) (interface{}, error) {
	if transform == nil {
		return nil, fmt.Errorf("transform is nil")
	}

	// Look up transform handler in registry
	def, ok := GetTransform(string(transform.Type))
	if !ok {
		return nil, fmt.Errorf("unknown transform type: %s (not registered)", transform.Type)
	}

	// Call the registered function
	return def.Func(te, ctx, transform)
}

// Note: Transform implementations are now in transforms_builtin.go
// They are registered via the global registry and called through Evaluate

// resolveNumber is a helper method used by transform implementations
func (te *TransformEvaluator) resolveNumber(ctx *Context, v interface{}) (float64, error) {
	if v == nil {
		return 0, nil
	}

	resolved, err := ctx.Resolve(v)
	if err != nil {
		return 0, err
	}

	num, ok := toFloat64(resolved)
	if !ok {
		return 0, fmt.Errorf("cannot convert %T to number", resolved)
	}

	return num, nil
}
