package eval

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
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

// Evaluate evaluates a transform and returns the result
func (te *TransformEvaluator) Evaluate(ctx *Context, transform *ast.Transform) (interface{}, error) {
	if transform == nil {
		return nil, fmt.Errorf("transform is nil")
	}

	switch transform.Type {
	// Math transforms
	case ast.TransformTypeAdd:
		return te.evaluateAdd(ctx, transform)
	case ast.TransformTypeSubtract:
		return te.evaluateSubtract(ctx, transform)
	case ast.TransformTypeMultiply:
		return te.evaluateMultiply(ctx, transform)
	case ast.TransformTypeDivide:
		return te.evaluateDivide(ctx, transform)
	case ast.TransformTypeModulo:
		return te.evaluateModulo(ctx, transform)
	case ast.TransformTypeClamp:
		return te.evaluateClamp(ctx, transform)
	case ast.TransformTypeRound:
		return te.evaluateRound(ctx, transform)
	case ast.TransformTypeFloor:
		return te.evaluateFloor(ctx, transform)
	case ast.TransformTypeCeil:
		return te.evaluateCeil(ctx, transform)
	case ast.TransformTypeAbs:
		return te.evaluateAbs(ctx, transform)
	case ast.TransformTypeMin:
		return te.evaluateMin(ctx, transform)
	case ast.TransformTypeMax:
		return te.evaluateMax(ctx, transform)
	case ast.TransformTypeRandom:
		return te.evaluateRandom(ctx, transform)

	// GPS transforms
	case ast.TransformTypeMoveTowards:
		return te.evaluateMoveTowards(ctx, transform)
	case ast.TransformTypeGpsDistance:
		return te.evaluateGpsDistance(ctx, transform)
	case ast.TransformTypeGpsBearing:
		return te.evaluateGpsBearing(ctx, transform)
	case ast.TransformTypePointInRadius:
		return te.evaluatePointInRadius(ctx, transform)
	case ast.TransformTypePointInPolygon:
		return te.evaluatePointInPolygon(ctx, transform)

	// String transforms
	case ast.TransformTypeConcat:
		return te.evaluateConcat(ctx, transform)
	case ast.TransformTypeFormat:
		return te.evaluateFormat(ctx, transform)
	case ast.TransformTypeSubstring:
		return te.evaluateSubstring(ctx, transform)
	case ast.TransformTypeToUpper:
		return te.evaluateToUpper(ctx, transform)
	case ast.TransformTypeToLower:
		return te.evaluateToLower(ctx, transform)
	case ast.TransformTypeTrim:
		return te.evaluateTrim(ctx, transform)

	// Logic transforms
	case ast.TransformTypeIf:
		return te.evaluateIf(ctx, transform)
	case ast.TransformTypeCoalesce:
		return te.evaluateCoalesce(ctx, transform)
	case ast.TransformTypeNot:
		return te.evaluateNot(ctx, transform)

	// Time transforms
	case ast.TransformTypeNow:
		return te.evaluateNow(ctx, transform)
	case ast.TransformTypeTimeSince:
		return te.evaluateTimeSince(ctx, transform)
	case ast.TransformTypeTimeAdd:
		return te.evaluateTimeAdd(ctx, transform)

	// UUID transform
	case ast.TransformTypeUUID:
		return te.evaluateUUID(ctx, transform)

	default:
		return nil, fmt.Errorf("unknown transform type: %s", transform.Type)
	}
}

// Math transforms

func (te *TransformEvaluator) evaluateAdd(ctx *Context, t *ast.Transform) (interface{}, error) {
	left, err := te.resolveNumber(ctx, t.Left)
	if err != nil {
		return nil, err
	}
	right, err := te.resolveNumber(ctx, t.Right)
	if err != nil {
		return nil, err
	}
	return left + right, nil
}

func (te *TransformEvaluator) evaluateSubtract(ctx *Context, t *ast.Transform) (interface{}, error) {
	left, err := te.resolveNumber(ctx, t.Left)
	if err != nil {
		return nil, err
	}
	right, err := te.resolveNumber(ctx, t.Right)
	if err != nil {
		return nil, err
	}
	return left - right, nil
}

func (te *TransformEvaluator) evaluateMultiply(ctx *Context, t *ast.Transform) (interface{}, error) {
	left, err := te.resolveNumber(ctx, t.Left)
	if err != nil {
		return nil, err
	}
	right, err := te.resolveNumber(ctx, t.Right)
	if err != nil {
		return nil, err
	}
	return left * right, nil
}

func (te *TransformEvaluator) evaluateDivide(ctx *Context, t *ast.Transform) (interface{}, error) {
	left, err := te.resolveNumber(ctx, t.Left)
	if err != nil {
		return nil, err
	}
	right, err := te.resolveNumber(ctx, t.Right)
	if err != nil {
		return nil, err
	}
	if right == 0 {
		return nil, fmt.Errorf("division by zero")
	}
	return left / right, nil
}

func (te *TransformEvaluator) evaluateModulo(ctx *Context, t *ast.Transform) (interface{}, error) {
	left, err := te.resolveNumber(ctx, t.Left)
	if err != nil {
		return nil, err
	}
	right, err := te.resolveNumber(ctx, t.Right)
	if err != nil {
		return nil, err
	}
	if right == 0 {
		return nil, fmt.Errorf("modulo by zero")
	}
	return math.Mod(left, right), nil
}

func (te *TransformEvaluator) evaluateClamp(ctx *Context, t *ast.Transform) (interface{}, error) {
	value, err := te.resolveNumber(ctx, t.Value)
	if err != nil {
		return nil, err
	}
	minVal, err := te.resolveNumber(ctx, t.Min)
	if err != nil {
		return nil, err
	}
	maxVal, err := te.resolveNumber(ctx, t.Max)
	if err != nil {
		return nil, err
	}
	return math.Max(minVal, math.Min(maxVal, value)), nil
}

func (te *TransformEvaluator) evaluateRound(ctx *Context, t *ast.Transform) (interface{}, error) {
	value, err := te.resolveNumber(ctx, t.Value)
	if err != nil {
		return nil, err
	}
	return math.Round(value), nil
}

func (te *TransformEvaluator) evaluateFloor(ctx *Context, t *ast.Transform) (interface{}, error) {
	value, err := te.resolveNumber(ctx, t.Value)
	if err != nil {
		return nil, err
	}
	return math.Floor(value), nil
}

func (te *TransformEvaluator) evaluateCeil(ctx *Context, t *ast.Transform) (interface{}, error) {
	value, err := te.resolveNumber(ctx, t.Value)
	if err != nil {
		return nil, err
	}
	return math.Ceil(value), nil
}

func (te *TransformEvaluator) evaluateAbs(ctx *Context, t *ast.Transform) (interface{}, error) {
	value, err := te.resolveNumber(ctx, t.Value)
	if err != nil {
		return nil, err
	}
	return math.Abs(value), nil
}

func (te *TransformEvaluator) evaluateMin(ctx *Context, t *ast.Transform) (interface{}, error) {
	left, err := te.resolveNumber(ctx, t.Left)
	if err != nil {
		return nil, err
	}
	right, err := te.resolveNumber(ctx, t.Right)
	if err != nil {
		return nil, err
	}
	return math.Min(left, right), nil
}

func (te *TransformEvaluator) evaluateMax(ctx *Context, t *ast.Transform) (interface{}, error) {
	left, err := te.resolveNumber(ctx, t.Left)
	if err != nil {
		return nil, err
	}
	right, err := te.resolveNumber(ctx, t.Right)
	if err != nil {
		return nil, err
	}
	return math.Max(left, right), nil
}

func (te *TransformEvaluator) evaluateRandom(ctx *Context, t *ast.Transform) (interface{}, error) {
	minVal, err := te.resolveNumber(ctx, t.MinValue)
	if err != nil {
		minVal = 0
	}
	maxVal, err := te.resolveNumber(ctx, t.MaxValue)
	if err != nil {
		maxVal = 1
	}
	return minVal + te.rng.Float64()*(maxVal-minVal), nil
}

// GPS transforms

func (te *TransformEvaluator) evaluateMoveTowards(ctx *Context, t *ast.Transform) (interface{}, error) {
	currentVal, err := ctx.Resolve(t.Current)
	if err != nil {
		return nil, fmt.Errorf("current: %w", err)
	}
	current, err := toGeoPoint(currentVal)
	if err != nil {
		return nil, fmt.Errorf("current: %w", err)
	}

	targetVal, err := ctx.Resolve(t.Target)
	if err != nil {
		return nil, fmt.Errorf("target: %w", err)
	}
	target, err := toGeoPoint(targetVal)
	if err != nil {
		return nil, fmt.Errorf("target: %w", err)
	}

	// Calculate distance to move based on speed and delta time
	speedMps := normalizeSpeedToMetersPerMillisecond(t.Speed, t.Unit)
	distance := speedMps * float64(ctx.DeltaTime.Milliseconds())

	return moveTowards(current, target, distance), nil
}

func (te *TransformEvaluator) evaluateGpsDistance(ctx *Context, t *ast.Transform) (interface{}, error) {
	fromVal, err := ctx.Resolve(t.From)
	if err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}
	from, err := toGeoPoint(fromVal)
	if err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}

	toVal, err := ctx.Resolve(t.To)
	if err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}
	to, err := toGeoPoint(toVal)
	if err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}

	distance := haversineDistance(from, to)

	// Convert to requested unit
	switch t.Unit {
	case "km", "kilometers":
		return distance / 1000.0, nil
	case "miles", "mi":
		return distance / 1609.344, nil
	default:
		return distance, nil
	}
}

func (te *TransformEvaluator) evaluateGpsBearing(ctx *Context, t *ast.Transform) (interface{}, error) {
	fromVal, err := ctx.Resolve(t.From)
	if err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}
	from, err := toGeoPoint(fromVal)
	if err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}

	toVal, err := ctx.Resolve(t.To)
	if err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}
	to, err := toGeoPoint(toVal)
	if err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}

	return bearing(from, to), nil
}

func (te *TransformEvaluator) evaluatePointInRadius(ctx *Context, t *ast.Transform) (interface{}, error) {
	pointVal, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, fmt.Errorf("value: %w", err)
	}
	point, err := toGeoPoint(pointVal)
	if err != nil {
		return nil, fmt.Errorf("value: %w", err)
	}

	centerVal, err := ctx.Resolve(t.Center)
	if err != nil {
		return nil, fmt.Errorf("center: %w", err)
	}
	center, err := toGeoPoint(centerVal)
	if err != nil {
		return nil, fmt.Errorf("center: %w", err)
	}

	radius := convertDistance(t.Radius, t.Unit)
	return pointInCircle(point, center, radius), nil
}

func (te *TransformEvaluator) evaluatePointInPolygon(ctx *Context, t *ast.Transform) (interface{}, error) {
	pointVal, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, fmt.Errorf("value: %w", err)
	}
	point, err := toGeoPoint(pointVal)
	if err != nil {
		return nil, fmt.Errorf("value: %w", err)
	}

	return pointInPolygon(point, t.Polygon), nil
}

// String transforms

func (te *TransformEvaluator) evaluateConcat(ctx *Context, t *ast.Transform) (interface{}, error) {
	var builder strings.Builder
	for _, s := range t.Strings {
		val, err := ctx.Resolve(s)
		if err != nil {
			return nil, err
		}
		builder.WriteString(fmt.Sprintf("%v", val))
	}
	return builder.String(), nil
}

func (te *TransformEvaluator) evaluateFormat(ctx *Context, t *ast.Transform) (interface{}, error) {
	args := make([]interface{}, len(t.Args))
	for i, arg := range t.Args {
		val, err := ctx.Resolve(arg)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}
	return fmt.Sprintf(t.Format, args...), nil
}

func (te *TransformEvaluator) evaluateSubstring(ctx *Context, t *ast.Transform) (interface{}, error) {
	val, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, err
	}
	str := fmt.Sprintf("%v", val)

	start := t.Start
	if start < 0 {
		start = 0
	}
	if start >= len(str) {
		return "", nil
	}

	end := start + t.Length
	if t.Length <= 0 || end > len(str) {
		end = len(str)
	}

	return str[start:end], nil
}

func (te *TransformEvaluator) evaluateToUpper(ctx *Context, t *ast.Transform) (interface{}, error) {
	val, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, err
	}
	return strings.ToUpper(fmt.Sprintf("%v", val)), nil
}

func (te *TransformEvaluator) evaluateToLower(ctx *Context, t *ast.Transform) (interface{}, error) {
	val, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, err
	}
	return strings.ToLower(fmt.Sprintf("%v", val)), nil
}

func (te *TransformEvaluator) evaluateTrim(ctx *Context, t *ast.Transform) (interface{}, error) {
	val, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, err
	}
	return strings.TrimSpace(fmt.Sprintf("%v", val)), nil
}

// Logic transforms

func (te *TransformEvaluator) evaluateIf(ctx *Context, t *ast.Transform) (interface{}, error) {
	condVal, err := ctx.Resolve(t.Condition)
	if err != nil {
		return nil, err
	}

	var condResult bool
	switch v := condVal.(type) {
	case bool:
		condResult = v
	case *ast.Expression:
		condResult, err = evaluateExpression(ctx, v)
		if err != nil {
			return nil, err
		}
	default:
		// Truthy check
		condResult = condVal != nil && condVal != false && condVal != 0 && condVal != ""
	}

	if condResult {
		return ctx.Resolve(t.Then)
	}
	return ctx.Resolve(t.Else)
}

func (te *TransformEvaluator) evaluateCoalesce(ctx *Context, t *ast.Transform) (interface{}, error) {
	for _, v := range t.Values {
		val, err := ctx.Resolve(v)
		if err != nil {
			continue // Treat errors as nil
		}
		if val != nil {
			return val, nil
		}
	}
	return nil, nil
}

func (te *TransformEvaluator) evaluateNot(ctx *Context, t *ast.Transform) (interface{}, error) {
	val, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, err
	}

	switch v := val.(type) {
	case bool:
		return !v, nil
	default:
		// Truthy check
		truthy := val != nil && val != false && val != 0 && val != ""
		return !truthy, nil
	}
}

// Time transforms

func (te *TransformEvaluator) evaluateNow(ctx *Context, t *ast.Transform) (interface{}, error) {
	return time.Now().UnixMilli(), nil
}

func (te *TransformEvaluator) evaluateTimeSince(ctx *Context, t *ast.Transform) (interface{}, error) {
	sinceVal, err := ctx.Resolve(t.Since)
	if err != nil {
		return nil, err
	}

	var sinceMs int64
	switch v := sinceVal.(type) {
	case int64:
		sinceMs = v
	case float64:
		sinceMs = int64(v)
	case int:
		sinceMs = int64(v)
	default:
		return nil, fmt.Errorf("since must be a timestamp (milliseconds)")
	}

	return time.Now().UnixMilli() - sinceMs, nil
}

func (te *TransformEvaluator) evaluateTimeAdd(ctx *Context, t *ast.Transform) (interface{}, error) {
	val, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, err
	}

	var baseMs int64
	switch v := val.(type) {
	case int64:
		baseMs = v
	case float64:
		baseMs = int64(v)
	case int:
		baseMs = int64(v)
	default:
		return nil, fmt.Errorf("value must be a timestamp (milliseconds)")
	}

	return baseMs + int64(t.Duration), nil
}

// UUID transform

func (te *TransformEvaluator) evaluateUUID(ctx *Context, t *ast.Transform) (interface{}, error) {
	// Generate a simple UUID v4
	// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	uuid := make([]byte, 36)
	const hexChars = "0123456789abcdef"

	for i := 0; i < 36; i++ {
		switch i {
		case 8, 13, 18, 23:
			uuid[i] = '-'
		case 14:
			uuid[i] = '4'
		case 19:
			uuid[i] = hexChars[te.rng.Intn(4)+8] // 8, 9, a, or b
		default:
			uuid[i] = hexChars[te.rng.Intn(16)]
		}
	}

	return string(uuid), nil
}

// Helper functions

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
