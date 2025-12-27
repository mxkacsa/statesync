package eval

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

func init() {
	registerMathTransforms()
	registerGeoTransforms()
	registerStringTransforms()
	registerLogicTransforms()
	registerTimeTransforms()
	registerUtilTransforms()
}

// =============================================================================
// Math Transforms
// =============================================================================

func registerMathTransforms() {
	// Add
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeAdd),
		Category:    CategoryMath,
		Description: "Adds two numbers",
		Inputs: []PortDefinition{
			{Name: "left", Type: "float64", Required: true},
			{Name: "right", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformAdd,
	})

	// Subtract
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeSubtract),
		Category:    CategoryMath,
		Description: "Subtracts right from left",
		Inputs: []PortDefinition{
			{Name: "left", Type: "float64", Required: true},
			{Name: "right", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformSubtract,
	})

	// Multiply
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeMultiply),
		Category:    CategoryMath,
		Description: "Multiplies two numbers",
		Inputs: []PortDefinition{
			{Name: "left", Type: "float64", Required: true},
			{Name: "right", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformMultiply,
	})

	// Divide
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeDivide),
		Category:    CategoryMath,
		Description: "Divides left by right",
		Inputs: []PortDefinition{
			{Name: "left", Type: "float64", Required: true},
			{Name: "right", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformDivide,
	})

	// Modulo
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeModulo),
		Category:    CategoryMath,
		Description: "Calculates modulo",
		Inputs: []PortDefinition{
			{Name: "left", Type: "float64", Required: true},
			{Name: "right", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformModulo,
	})

	// Clamp
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeClamp),
		Category:    CategoryMath,
		Description: "Clamps a value between min and max",
		Inputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
			{Name: "min", Type: "float64", Required: true},
			{Name: "max", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformClamp,
	})

	// Round
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeRound),
		Category:    CategoryMath,
		Description: "Rounds to nearest integer",
		Inputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformRound,
	})

	// Floor
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeFloor),
		Category:    CategoryMath,
		Description: "Rounds down to integer",
		Inputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformFloor,
	})

	// Ceil
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeCeil),
		Category:    CategoryMath,
		Description: "Rounds up to integer",
		Inputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformCeil,
	})

	// Abs
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeAbs),
		Category:    CategoryMath,
		Description: "Returns absolute value",
		Inputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformAbs,
	})

	// Min
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeMin),
		Category:    CategoryMath,
		Description: "Returns smaller of two numbers",
		Inputs: []PortDefinition{
			{Name: "left", Type: "float64", Required: true},
			{Name: "right", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformMin,
	})

	// Max
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeMax),
		Category:    CategoryMath,
		Description: "Returns larger of two numbers",
		Inputs: []PortDefinition{
			{Name: "left", Type: "float64", Required: true},
			{Name: "right", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformMax,
	})

	// Random
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeRandom),
		Category:    CategoryMath,
		Description: "Generates random number between min and max",
		Inputs: []PortDefinition{
			{Name: "minValue", Type: "float64", Required: false, Default: 0.0},
			{Name: "maxValue", Type: "float64", Required: false, Default: 1.0},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformRandom,
	})
}

// =============================================================================
// Geo Transforms
// =============================================================================

func registerGeoTransforms() {
	// MoveTowards
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeMoveTowards),
		Category:    CategoryGeo,
		Description: "Moves a point towards a target by a distance",
		Inputs: []PortDefinition{
			{Name: "current", Type: "GeoPoint", Required: true},
			{Name: "target", Type: "GeoPoint", Required: true},
			{Name: "speed", Type: "float64", Required: true},
			{Name: "unit", Type: "string", Required: false, Default: "m/s"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "GeoPoint"},
		},
		Func: transformMoveTowards,
	})

	// GpsDistance
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeGpsDistance),
		Category:    CategoryGeo,
		Description: "Calculates distance between two GPS points",
		Inputs: []PortDefinition{
			{Name: "from", Type: "GeoPoint", Required: true},
			{Name: "to", Type: "GeoPoint", Required: true},
			{Name: "unit", Type: "string", Required: false, Default: "meters"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformGpsDistance,
	})

	// GpsBearing
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeGpsBearing),
		Category:    CategoryGeo,
		Description: "Calculates bearing between two GPS points",
		Inputs: []PortDefinition{
			{Name: "from", Type: "GeoPoint", Required: true},
			{Name: "to", Type: "GeoPoint", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: transformGpsBearing,
	})

	// PointInRadius
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypePointInRadius),
		Category:    CategoryGeo,
		Description: "Checks if point is within radius of center",
		Inputs: []PortDefinition{
			{Name: "value", Type: "GeoPoint", Required: true},
			{Name: "center", Type: "GeoPoint", Required: true},
			{Name: "radius", Type: "float64", Required: true},
			{Name: "unit", Type: "string", Required: false, Default: "meters"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool"},
		},
		Func: transformPointInRadius,
	})

	// PointInPolygon
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypePointInPolygon),
		Category:    CategoryGeo,
		Description: "Checks if point is inside polygon",
		Inputs: []PortDefinition{
			{Name: "value", Type: "GeoPoint", Required: true},
			{Name: "polygon", Type: "[]GeoPoint", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool"},
		},
		Func: transformPointInPolygon,
	})
}

// =============================================================================
// String Transforms
// =============================================================================

func registerStringTransforms() {
	// Concat
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeConcat),
		Category:    CategoryString,
		Description: "Concatenates strings",
		Inputs: []PortDefinition{
			{Name: "strings", Type: "[]interface{}", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string"},
		},
		Func: transformConcat,
	})

	// Format
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeFormat),
		Category:    CategoryString,
		Description: "Formats a string with arguments",
		Inputs: []PortDefinition{
			{Name: "format", Type: "string", Required: true},
			{Name: "args", Type: "[]interface{}", Required: false},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string"},
		},
		Func: transformFormat,
	})

	// Substring
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeSubstring),
		Category:    CategoryString,
		Description: "Extracts a substring",
		Inputs: []PortDefinition{
			{Name: "value", Type: "string", Required: true},
			{Name: "start", Type: "int", Required: true},
			{Name: "length", Type: "int", Required: false},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string"},
		},
		Func: transformSubstring,
	})

	// ToUpper
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeToUpper),
		Category:    CategoryString,
		Description: "Converts string to uppercase",
		Inputs: []PortDefinition{
			{Name: "value", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string"},
		},
		Func: transformToUpper,
	})

	// ToLower
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeToLower),
		Category:    CategoryString,
		Description: "Converts string to lowercase",
		Inputs: []PortDefinition{
			{Name: "value", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string"},
		},
		Func: transformToLower,
	})

	// Trim
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeTrim),
		Category:    CategoryString,
		Description: "Trims whitespace from string",
		Inputs: []PortDefinition{
			{Name: "value", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string"},
		},
		Func: transformTrim,
	})
}

// =============================================================================
// Logic Transforms
// =============================================================================

func registerLogicTransforms() {
	// If
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeIf),
		Category:    CategoryLogic,
		Description: "Conditional value selection",
		Inputs: []PortDefinition{
			{Name: "condition", Type: "interface{}", Required: true},
			{Name: "then", Type: "interface{}", Required: true},
			{Name: "else", Type: "interface{}", Required: false},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "interface{}"},
		},
		Func: transformIf,
	})

	// Coalesce
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeCoalesce),
		Category:    CategoryLogic,
		Description: "Returns first non-nil value",
		Inputs: []PortDefinition{
			{Name: "values", Type: "[]interface{}", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "interface{}"},
		},
		Func: transformCoalesce,
	})

	// Not
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeNot),
		Category:    CategoryLogic,
		Description: "Logical negation",
		Inputs: []PortDefinition{
			{Name: "value", Type: "interface{}", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool"},
		},
		Func: transformNot,
	})
}

// =============================================================================
// Time Transforms
// =============================================================================

func registerTimeTransforms() {
	// Now
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeNow),
		Category:    CategoryTime,
		Description: "Returns current timestamp in milliseconds",
		Inputs:      []PortDefinition{},
		Outputs: []PortDefinition{
			{Name: "result", Type: "int64"},
		},
		Func: transformNow,
	})

	// TimeSince
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeTimeSince),
		Category:    CategoryTime,
		Description: "Returns milliseconds elapsed since timestamp",
		Inputs: []PortDefinition{
			{Name: "since", Type: "int64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "int64"},
		},
		Func: transformTimeSince,
	})

	// TimeAdd
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeTimeAdd),
		Category:    CategoryTime,
		Description: "Adds duration to timestamp",
		Inputs: []PortDefinition{
			{Name: "value", Type: "int64", Required: true},
			{Name: "duration", Type: "int64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "int64"},
		},
		Func: transformTimeAdd,
	})
}

// =============================================================================
// Utility Transforms
// =============================================================================

func registerUtilTransforms() {
	// UUID
	RegisterTransform(&TransformDefinition{
		Name:        string(ast.TransformTypeUUID),
		Category:    CategoryLogic,
		Description: "Generates a UUID v4",
		Inputs:      []PortDefinition{},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string"},
		},
		Func: transformUUID,
	})
}

// =============================================================================
// Transform Implementations
// =============================================================================

// Math Transforms

func transformAdd(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformSubtract(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformMultiply(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformDivide(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformModulo(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformClamp(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformRound(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	value, err := te.resolveNumber(ctx, t.Value)
	if err != nil {
		return nil, err
	}
	return math.Round(value), nil
}

func transformFloor(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	value, err := te.resolveNumber(ctx, t.Value)
	if err != nil {
		return nil, err
	}
	return math.Floor(value), nil
}

func transformCeil(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	value, err := te.resolveNumber(ctx, t.Value)
	if err != nil {
		return nil, err
	}
	return math.Ceil(value), nil
}

func transformAbs(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	value, err := te.resolveNumber(ctx, t.Value)
	if err != nil {
		return nil, err
	}
	return math.Abs(value), nil
}

func transformMin(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformMax(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformRandom(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

// Geo Transforms

func transformMoveTowards(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

	speedMps := normalizeSpeedToMetersPerMillisecond(t.Speed, t.Unit)
	distance := speedMps * float64(ctx.DeltaTime.Milliseconds())

	return moveTowards(current, target, distance), nil
}

func transformGpsDistance(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

	switch t.Unit {
	case "km", "kilometers":
		return distance / 1000.0, nil
	case "miles", "mi":
		return distance / 1609.344, nil
	default:
		return distance, nil
	}
}

func transformGpsBearing(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformPointInRadius(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformPointInPolygon(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

// String Transforms

func transformConcat(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformFormat(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformSubstring(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformToUpper(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	val, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, err
	}
	return strings.ToUpper(fmt.Sprintf("%v", val)), nil
}

func transformToLower(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	val, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, err
	}
	return strings.ToLower(fmt.Sprintf("%v", val)), nil
}

func transformTrim(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	val, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, err
	}
	return strings.TrimSpace(fmt.Sprintf("%v", val)), nil
}

// Logic Transforms

func transformIf(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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
		condResult = condVal != nil && condVal != false && condVal != 0 && condVal != ""
	}

	if condResult {
		return ctx.Resolve(t.Then)
	}
	return ctx.Resolve(t.Else)
}

func transformCoalesce(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	for _, v := range t.Values {
		val, err := ctx.Resolve(v)
		if err != nil {
			continue
		}
		if val != nil {
			return val, nil
		}
	}
	return nil, nil
}

func transformNot(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	val, err := ctx.Resolve(t.Value)
	if err != nil {
		return nil, err
	}

	switch v := val.(type) {
	case bool:
		return !v, nil
	default:
		truthy := val != nil && val != false && val != 0 && val != ""
		return !truthy, nil
	}
}

// Time Transforms

func transformNow(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	return time.Now().UnixMilli(), nil
}

func transformTimeSince(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

func transformTimeAdd(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
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

// UUID Transform

func transformUUID(te *TransformEvaluator, ctx *Context, t *ast.Transform) (interface{}, error) {
	uuid := make([]byte, 36)
	const hexChars = "0123456789abcdef"

	for i := 0; i < 36; i++ {
		switch i {
		case 8, 13, 18, 23:
			uuid[i] = '-'
		case 14:
			uuid[i] = '4'
		case 19:
			uuid[i] = hexChars[te.rng.Intn(4)+8]
		default:
			uuid[i] = hexChars[te.rng.Intn(16)]
		}
	}

	return string(uuid), nil
}
