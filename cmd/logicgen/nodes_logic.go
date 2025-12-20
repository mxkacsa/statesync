package main

import "strings"

// ============================================================================
// Logic and Math Nodes - Registered via the extensibility system
// ============================================================================

func init() {
	registerLogicNodes()
	registerMathNodes()
	registerRandomNodes()
	registerTimeNodes()
}

// ============================================================================
// Logic Nodes
// ============================================================================

func registerLogicNodes() {
	// Compare - Compares two values
	MustRegisterNode(NodeDefinition{
		Type:        NodeCompare,
		Category:    "Logic",
		Description: "Compares two values using the specified operator",
		Inputs: []PortDefinition{
			{Name: "left", Type: "any", Required: true},
			{Name: "op", Type: "string", Required: true},
			{Name: "right", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool", Required: true},
		},
		Generator: generateCompareNode,
	})

	// And - Logical AND
	MustRegisterNode(NodeDefinition{
		Type:        NodeAnd,
		Category:    "Logic",
		Description: "Logical AND of two booleans",
		Inputs: []PortDefinition{
			{Name: "a", Type: "bool", Required: true},
			{Name: "b", Type: "bool", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool", Required: true},
		},
		Generator: func(ctx *GeneratorContext, node *Node) error {
			return generateLogicOpNode(ctx, node, "&&")
		},
	})

	// Or - Logical OR
	MustRegisterNode(NodeDefinition{
		Type:        NodeOr,
		Category:    "Logic",
		Description: "Logical OR of two booleans",
		Inputs: []PortDefinition{
			{Name: "a", Type: "bool", Required: true},
			{Name: "b", Type: "bool", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool", Required: true},
		},
		Generator: func(ctx *GeneratorContext, node *Node) error {
			return generateLogicOpNode(ctx, node, "||")
		},
	})

	// Not - Logical NOT
	MustRegisterNode(NodeDefinition{
		Type:        NodeNot,
		Category:    "Logic",
		Description: "Logical NOT of a boolean",
		Inputs: []PortDefinition{
			{Name: "value", Type: "bool", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool", Required: true},
		},
		Generator: generateNotNode,
	})

	// IsNull - Check if value is null
	MustRegisterNode(NodeDefinition{
		Type:        NodeIsNull,
		Category:    "Logic",
		Description: "Check if value is null/nil",
		Inputs: []PortDefinition{
			{Name: "value", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool", Required: true},
		},
		Generator: generateIsNullNode,
	})

	// IsEmpty - Check if array/string is empty
	MustRegisterNode(NodeDefinition{
		Type:        NodeIsEmpty,
		Category:    "Logic",
		Description: "Check if array or string is empty",
		Inputs: []PortDefinition{
			{Name: "value", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool", Required: true},
		},
		Generator: generateIsEmptyNode,
	})
}

// ============================================================================
// Math Nodes
// ============================================================================

func registerMathNodes() {
	// Basic math operations
	mathOps := []struct {
		nodeType NodeType
		op       string
		desc     string
	}{
		{NodeAdd, "+", "Adds two numbers"},
		{NodeSubtract, "-", "Subtracts two numbers"},
		{NodeMultiply, "*", "Multiplies two numbers"},
		{NodeDivide, "/", "Divides two numbers"},
		{NodeModulo, "%", "Modulo of two numbers"},
	}

	for _, m := range mathOps {
		op := m.op // capture for closure
		MustRegisterNode(NodeDefinition{
			Type:        m.nodeType,
			Category:    "Math",
			Description: m.desc,
			Inputs: []PortDefinition{
				{Name: "a", Type: "number", Required: true},
				{Name: "b", Type: "number", Required: true},
			},
			Outputs: []PortDefinition{
				{Name: "result", Type: "number", Required: true},
			},
			Generator: func(ctx *GeneratorContext, node *Node) error {
				return generateMathOpNode(ctx, node, op)
			},
		})
	}

	// Min/Max
	MustRegisterNode(NodeDefinition{
		Type:        NodeMin,
		Category:    "Math",
		Description: "Minimum of two numbers",
		Inputs: []PortDefinition{
			{Name: "a", Type: "number", Required: true},
			{Name: "b", Type: "number", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "number", Required: true},
		},
		Generator: func(ctx *GeneratorContext, node *Node) error {
			return generateMinMaxNode(ctx, node, "min")
		},
	})

	MustRegisterNode(NodeDefinition{
		Type:        NodeMax,
		Category:    "Math",
		Description: "Maximum of two numbers",
		Inputs: []PortDefinition{
			{Name: "a", Type: "number", Required: true},
			{Name: "b", Type: "number", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "number", Required: true},
		},
		Generator: func(ctx *GeneratorContext, node *Node) error {
			return generateMinMaxNode(ctx, node, "max")
		},
	})

	// Single-arg math functions
	singleArgFuncs := []struct {
		nodeType NodeType
		fn       string
		desc     string
	}{
		{NodeSqrt, "math.Sqrt", "Square root of a number"},
		{NodeAbs, "math.Abs", "Absolute value"},
		{NodeSin, "math.Sin", "Sine of an angle (radians)"},
		{NodeCos, "math.Cos", "Cosine of an angle (radians)"},
	}

	for _, f := range singleArgFuncs {
		fn := f.fn // capture for closure
		MustRegisterNode(NodeDefinition{
			Type:        f.nodeType,
			Category:    "Math",
			Description: f.desc,
			Inputs: []PortDefinition{
				{Name: "value", Type: "float64", Required: true},
			},
			Outputs: []PortDefinition{
				{Name: "result", Type: "float64", Required: true},
			},
			Generator: func(ctx *GeneratorContext, node *Node) error {
				return generateMathFuncNode(ctx, node, fn)
			},
		})
	}

	// Two-arg math functions
	MustRegisterNode(NodeDefinition{
		Type:        NodePow,
		Category:    "Math",
		Description: "Power of a number",
		Inputs: []PortDefinition{
			{Name: "base", Type: "float64", Required: true},
			{Name: "exp", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64", Required: true},
		},
		Generator: func(ctx *GeneratorContext, node *Node) error {
			return generateMathFunc2Node(ctx, node, "math.Pow")
		},
	})

	MustRegisterNode(NodeDefinition{
		Type:        NodeAtan2,
		Category:    "Math",
		Description: "Arc tangent of y/x (radians)",
		Inputs: []PortDefinition{
			{Name: "y", Type: "float64", Required: true},
			{Name: "x", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64", Required: true},
		},
		Generator: func(ctx *GeneratorContext, node *Node) error {
			return generateMathFunc2Node(ctx, node, "math.Atan2")
		},
	})
}

// ============================================================================
// Random Nodes
// ============================================================================

func registerRandomNodes() {
	// RandomInt
	MustRegisterNode(NodeDefinition{
		Type:        NodeRandomInt,
		Category:    "Random",
		Description: "Generate random integer in range [min, max]",
		Inputs: []PortDefinition{
			{Name: "min", Type: "int", Required: true},
			{Name: "max", Type: "int", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "value", Type: "int", Required: true},
		},
		Generator: generateRandomIntNode,
	})

	// RandomFloat
	MustRegisterNode(NodeDefinition{
		Type:        NodeRandomFloat,
		Category:    "Random",
		Description: "Generate random float in range [min, max)",
		Inputs: []PortDefinition{
			{Name: "min", Type: "float64", Required: true},
			{Name: "max", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "value", Type: "float64", Required: true},
		},
		Generator: generateRandomFloatNode,
	})

	// RandomBool
	MustRegisterNode(NodeDefinition{
		Type:        NodeRandomBool,
		Category:    "Random",
		Description: "Generate random boolean with given probability (0.0-1.0) of true",
		Inputs: []PortDefinition{
			{Name: "probability", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "value", Type: "bool", Required: true},
		},
		Generator: generateRandomBoolNode,
	})
}

// ============================================================================
// Time Nodes
// ============================================================================

func registerTimeNodes() {
	// GetCurrentTime
	MustRegisterNode(NodeDefinition{
		Type:        NodeGetCurrentTime,
		Category:    "Time",
		Description: "Get current Unix timestamp in seconds",
		Inputs:      []PortDefinition{},
		Outputs: []PortDefinition{
			{Name: "timestamp", Type: "int64", Required: true},
		},
		Generator: generateGetCurrentTimeNode,
	})

	// TimeSince
	MustRegisterNode(NodeDefinition{
		Type:        NodeTimeSince,
		Category:    "Time",
		Description: "Calculate seconds elapsed since a timestamp",
		Inputs: []PortDefinition{
			{Name: "startTime", Type: "int64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "seconds", Type: "int64", Required: true},
		},
		Generator: generateTimeSinceNode,
	})
}

// ============================================================================
// Generator Functions - Logic
// ============================================================================

func generateCompareNode(ctx *GeneratorContext, node *Node) error {
	leftSrc, err := ctx.ResolveInput(node, "left")
	if err != nil {
		return err
	}
	rightSrc, err := ctx.ResolveInput(node, "right")
	if err != nil {
		return err
	}
	opSrc, err := ctx.ResolveInput(node, "op")
	if err != nil {
		return err
	}

	op := strings.Trim(opSrc, `"`)
	outputVar := ctx.AllocateVariable(node, "result", "bool")

	ctx.WriteLine("%s := %s %s %s", outputVar, leftSrc, op, rightSrc)

	return nil
}

func generateLogicOpNode(ctx *GeneratorContext, node *Node, op string) error {
	aSrc, err := ctx.ResolveInput(node, "a")
	if err != nil {
		return err
	}
	bSrc, err := ctx.ResolveInput(node, "b")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "result", "bool")
	ctx.WriteLine("%s := %s %s %s", outputVar, aSrc, op, bSrc)
	return nil
}

func generateNotNode(ctx *GeneratorContext, node *Node) error {
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "result", "bool")
	ctx.WriteLine("%s := !%s", outputVar, valueSrc)
	return nil
}

func generateIsNullNode(ctx *GeneratorContext, node *Node) error {
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "result", "bool")
	ctx.WriteLine("%s := %s == nil", outputVar, valueSrc)
	return nil
}

func generateIsEmptyNode(ctx *GeneratorContext, node *Node) error {
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "result", "bool")
	ctx.WriteLine("%s := len(%s) == 0", outputVar, valueSrc)
	return nil
}

// ============================================================================
// Generator Functions - Math
// ============================================================================

func generateMathOpNode(ctx *GeneratorContext, node *Node, op string) error {
	aSrc, err := ctx.ResolveInput(node, "a")
	if err != nil {
		return err
	}
	bSrc, err := ctx.ResolveInput(node, "b")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "result", "number")

	// Add zero-check for division and modulo
	if op == "/" || op == "%" {
		ctx.WriteLine("// Safe division/modulo with zero check")
		ctx.WriteLine("var %s float64", outputVar)
		ctx.WriteLine("if %s != 0 {", bSrc)
		ctx.Indent()
		ctx.WriteLine("%s = float64(%s) %s float64(%s)", outputVar, aSrc, op, bSrc)
		ctx.Dedent()
		ctx.WriteLine("} else {")
		ctx.Indent()
		ctx.WriteLine("// Division by zero - result is 0")
		ctx.WriteLine("%s = 0", outputVar)
		ctx.Dedent()
		ctx.WriteLine("}")
	} else {
		ctx.WriteLine("%s := %s %s %s", outputVar, aSrc, op, bSrc)
	}

	return nil
}

func generateMinMaxNode(ctx *GeneratorContext, node *Node, fn string) error {
	aSrc, err := ctx.ResolveInput(node, "a")
	if err != nil {
		return err
	}
	bSrc, err := ctx.ResolveInput(node, "b")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "result", "number")
	ctx.WriteLine("%s := %s(%s, %s)", outputVar, fn, aSrc, bSrc)

	return nil
}

func generateMathFuncNode(ctx *GeneratorContext, node *Node, fn string) error {
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		// Try alternative input names
		valueSrc, err = ctx.ResolveInput(node, "angle")
		if err != nil {
			return err
		}
	}

	outputVar := ctx.AllocateVariable(node, "result", "float64")
	ctx.WriteLine("%s := %s(%s)", outputVar, fn, valueSrc)

	return nil
}

func generateMathFunc2Node(ctx *GeneratorContext, node *Node, fn string) error {
	var arg1, arg2 string
	var err error

	// Try base/exp for Pow
	arg1, err = ctx.ResolveInput(node, "base")
	if err != nil {
		// Try y/x for Atan2
		arg1, err = ctx.ResolveInput(node, "y")
		if err != nil {
			return err
		}
	}

	arg2, err = ctx.ResolveInput(node, "exp")
	if err != nil {
		arg2, err = ctx.ResolveInput(node, "x")
		if err != nil {
			return err
		}
	}

	outputVar := ctx.AllocateVariable(node, "result", "float64")
	ctx.WriteLine("%s := %s(%s, %s)", outputVar, fn, arg1, arg2)

	return nil
}

// ============================================================================
// Generator Functions - Random
// ============================================================================

func generateRandomIntNode(ctx *GeneratorContext, node *Node) error {
	minVal, err := ctx.ResolveInput(node, "min")
	if err != nil {
		return err
	}
	maxVal, err := ctx.ResolveInput(node, "max")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "result", "int")
	ctx.WriteLine("%s := rand.Intn(%s-%s+1) + %s", outputVar, maxVal, minVal, minVal)

	return nil
}

func generateRandomFloatNode(ctx *GeneratorContext, node *Node) error {
	minVal, err := ctx.ResolveInput(node, "min")
	if err != nil {
		return err
	}
	maxVal, err := ctx.ResolveInput(node, "max")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "result", "float64")
	ctx.WriteLine("%s := %s + rand.Float64()*(%s-%s)", outputVar, minVal, maxVal, minVal)

	return nil
}

func generateRandomBoolNode(ctx *GeneratorContext, node *Node) error {
	probSrc, err := ctx.ResolveInput(node, "probability")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "result", "bool")
	ctx.WriteLine("%s := rand.Float64() < %s", outputVar, probSrc)

	return nil
}

// ============================================================================
// Generator Functions - Time
// ============================================================================

func generateGetCurrentTimeNode(ctx *GeneratorContext, node *Node) error {
	outputVar := ctx.AllocateVariable(node, "timestamp", "int64")
	ctx.WriteLine("%s := time.Now().Unix()", outputVar)
	return nil
}

func generateTimeSinceNode(ctx *GeneratorContext, node *Node) error {
	startTime, err := ctx.ResolveInput(node, "startTime")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "seconds", "int64")
	ctx.WriteLine("%s := time.Now().Unix() - %s", outputVar, startTime)
	return nil
}
