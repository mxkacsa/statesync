package builtin

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func init() {
	registerMathNodes()
}

func registerMathNodes() {
	// Math.Add
	Register(&NodeDefinition{
		Name:        "Math.Add",
		Category:    CategoryMath,
		Description: "Adds two numbers",
		Inputs: []PortDefinition{
			{Name: "a", Type: "float64", Required: true},
			{Name: "b", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathAdd,
	})

	// Math.Subtract
	Register(&NodeDefinition{
		Name:        "Math.Subtract",
		Category:    CategoryMath,
		Description: "Subtracts b from a",
		Inputs: []PortDefinition{
			{Name: "a", Type: "float64", Required: true},
			{Name: "b", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathSubtract,
	})

	// Math.Multiply
	Register(&NodeDefinition{
		Name:        "Math.Multiply",
		Category:    CategoryMath,
		Description: "Multiplies two numbers",
		Inputs: []PortDefinition{
			{Name: "a", Type: "float64", Required: true},
			{Name: "b", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathMultiply,
	})

	// Math.Divide
	Register(&NodeDefinition{
		Name:        "Math.Divide",
		Category:    CategoryMath,
		Description: "Divides a by b",
		Inputs: []PortDefinition{
			{Name: "a", Type: "float64", Required: true},
			{Name: "b", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathDivide,
	})

	// Math.Modulo
	Register(&NodeDefinition{
		Name:        "Math.Modulo",
		Category:    CategoryMath,
		Description: "Calculates a mod b",
		Inputs: []PortDefinition{
			{Name: "a", Type: "float64", Required: true},
			{Name: "b", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathModulo,
	})

	// Math.Clamp
	Register(&NodeDefinition{
		Name:        "Math.Clamp",
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
		Func: mathClamp,
	})

	// Math.Min
	Register(&NodeDefinition{
		Name:        "Math.Min",
		Category:    CategoryMath,
		Description: "Returns the smaller of two numbers",
		Inputs: []PortDefinition{
			{Name: "a", Type: "float64", Required: true},
			{Name: "b", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathMin,
	})

	// Math.Max
	Register(&NodeDefinition{
		Name:        "Math.Max",
		Category:    CategoryMath,
		Description: "Returns the larger of two numbers",
		Inputs: []PortDefinition{
			{Name: "a", Type: "float64", Required: true},
			{Name: "b", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathMax,
	})

	// Math.Abs
	Register(&NodeDefinition{
		Name:        "Math.Abs",
		Category:    CategoryMath,
		Description: "Returns the absolute value",
		Inputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathAbs,
	})

	// Math.Round
	Register(&NodeDefinition{
		Name:        "Math.Round",
		Category:    CategoryMath,
		Description: "Rounds to nearest integer",
		Inputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathRound,
	})

	// Math.Floor
	Register(&NodeDefinition{
		Name:        "Math.Floor",
		Category:    CategoryMath,
		Description: "Rounds down to integer",
		Inputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathFloor,
	})

	// Math.Ceil
	Register(&NodeDefinition{
		Name:        "Math.Ceil",
		Category:    CategoryMath,
		Description: "Rounds up to integer",
		Inputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathCeil,
	})

	// Math.Sqrt
	Register(&NodeDefinition{
		Name:        "Math.Sqrt",
		Category:    CategoryMath,
		Description: "Calculates square root",
		Inputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathSqrt,
	})

	// Math.Pow
	Register(&NodeDefinition{
		Name:        "Math.Pow",
		Category:    CategoryMath,
		Description: "Calculates a^b",
		Inputs: []PortDefinition{
			{Name: "base", Type: "float64", Required: true},
			{Name: "exponent", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathPow,
	})

	// Math.Random
	Register(&NodeDefinition{
		Name:        "Math.Random",
		Category:    CategoryMath,
		Description: "Generates a random number between min and max",
		Inputs: []PortDefinition{
			{Name: "min", Type: "float64", Required: false, Default: 0.0},
			{Name: "max", Type: "float64", Required: false, Default: 1.0},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathRandom,
	})

	// Math.Lerp
	Register(&NodeDefinition{
		Name:        "Math.Lerp",
		Category:    CategoryMath,
		Description: "Linearly interpolates between a and b by t",
		Inputs: []PortDefinition{
			{Name: "a", Type: "float64", Required: true},
			{Name: "b", Type: "float64", Required: true},
			{Name: "t", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: mathLerp,
	})
}

// Math node implementations

func mathAdd(args map[string]interface{}) (interface{}, error) {
	a, ok := toFloat64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be a number")
	}
	b, ok := toFloat64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be a number")
	}
	return a + b, nil
}

func mathSubtract(args map[string]interface{}) (interface{}, error) {
	a, ok := toFloat64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be a number")
	}
	b, ok := toFloat64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be a number")
	}
	return a - b, nil
}

func mathMultiply(args map[string]interface{}) (interface{}, error) {
	a, ok := toFloat64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be a number")
	}
	b, ok := toFloat64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be a number")
	}
	return a * b, nil
}

func mathDivide(args map[string]interface{}) (interface{}, error) {
	a, ok := toFloat64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be a number")
	}
	b, ok := toFloat64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be a number")
	}
	if b == 0 {
		return nil, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

func mathModulo(args map[string]interface{}) (interface{}, error) {
	a, ok := toFloat64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be a number")
	}
	b, ok := toFloat64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be a number")
	}
	if b == 0 {
		return nil, fmt.Errorf("modulo by zero")
	}
	return math.Mod(a, b), nil
}

func mathClamp(args map[string]interface{}) (interface{}, error) {
	value, ok := toFloat64(args["value"])
	if !ok {
		return nil, fmt.Errorf("value must be a number")
	}
	min, ok := toFloat64(args["min"])
	if !ok {
		return nil, fmt.Errorf("min must be a number")
	}
	max, ok := toFloat64(args["max"])
	if !ok {
		return nil, fmt.Errorf("max must be a number")
	}
	return math.Max(min, math.Min(max, value)), nil
}

func mathMin(args map[string]interface{}) (interface{}, error) {
	a, ok := toFloat64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be a number")
	}
	b, ok := toFloat64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be a number")
	}
	return math.Min(a, b), nil
}

func mathMax(args map[string]interface{}) (interface{}, error) {
	a, ok := toFloat64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be a number")
	}
	b, ok := toFloat64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be a number")
	}
	return math.Max(a, b), nil
}

func mathAbs(args map[string]interface{}) (interface{}, error) {
	value, ok := toFloat64(args["value"])
	if !ok {
		return nil, fmt.Errorf("value must be a number")
	}
	return math.Abs(value), nil
}

func mathRound(args map[string]interface{}) (interface{}, error) {
	value, ok := toFloat64(args["value"])
	if !ok {
		return nil, fmt.Errorf("value must be a number")
	}
	return math.Round(value), nil
}

func mathFloor(args map[string]interface{}) (interface{}, error) {
	value, ok := toFloat64(args["value"])
	if !ok {
		return nil, fmt.Errorf("value must be a number")
	}
	return math.Floor(value), nil
}

func mathCeil(args map[string]interface{}) (interface{}, error) {
	value, ok := toFloat64(args["value"])
	if !ok {
		return nil, fmt.Errorf("value must be a number")
	}
	return math.Ceil(value), nil
}

func mathSqrt(args map[string]interface{}) (interface{}, error) {
	value, ok := toFloat64(args["value"])
	if !ok {
		return nil, fmt.Errorf("value must be a number")
	}
	if value < 0 {
		return nil, fmt.Errorf("cannot take square root of negative number")
	}
	return math.Sqrt(value), nil
}

func mathPow(args map[string]interface{}) (interface{}, error) {
	base, ok := toFloat64(args["base"])
	if !ok {
		return nil, fmt.Errorf("base must be a number")
	}
	exponent, ok := toFloat64(args["exponent"])
	if !ok {
		return nil, fmt.Errorf("exponent must be a number")
	}
	return math.Pow(base, exponent), nil
}

func mathRandom(args map[string]interface{}) (interface{}, error) {
	min, ok := toFloat64(args["min"])
	if !ok {
		min = 0
	}
	max, ok := toFloat64(args["max"])
	if !ok {
		max = 1
	}
	return min + rng.Float64()*(max-min), nil
}

func mathLerp(args map[string]interface{}) (interface{}, error) {
	a, ok := toFloat64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be a number")
	}
	b, ok := toFloat64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be a number")
	}
	t, ok := toFloat64(args["t"])
	if !ok {
		return nil, fmt.Errorf("t must be a number")
	}
	return a + (b-a)*t, nil
}
