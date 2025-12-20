package main

import "fmt"

// ============================================================================
// Async Nodes - Wait, timeout, and asynchronous operations
// ============================================================================

func init() {
	registerAsyncNodes()
}

// ============================================================================
// Async Nodes
// ============================================================================

func registerAsyncNodes() {
	// Wait - Pause execution for a duration
	MustRegisterNode(NodeDefinition{
		Type:        NodeWait,
		Category:    "Async",
		Description: "Pause execution for a duration",
		Inputs: []PortDefinition{
			{Name: "duration", Type: "int", Required: true},
			{Name: "unit", Type: "string", Required: false, Default: "ms"},
		},
		Outputs:   []PortDefinition{},
		Generator: generateWaitNode,
	})

	// WaitUntil - Wait until condition is true or timeout
	MustRegisterNode(NodeDefinition{
		Type:        NodeWaitUntil,
		Category:    "Async",
		Description: "Wait until condition is true or timeout",
		Inputs: []PortDefinition{
			{Name: "condition", Type: "bool", Required: true},
			{Name: "checkInterval", Type: "int", Required: false, Default: 100},
			{Name: "timeout", Type: "int", Required: false, Default: 30000},
		},
		Outputs: []PortDefinition{
			{Name: "timedOut", Type: "bool", Required: true},
		},
		Generator: generateWaitUntilNode,
	})

	// Timeout - Execute body nodes with a timeout
	MustRegisterNode(NodeDefinition{
		Type:        NodeTimeout,
		Category:    "Async",
		Description: "Execute body nodes with a timeout",
		Inputs: []PortDefinition{
			{Name: "duration", Type: "int", Required: true},
			{Name: "unit", Type: "string", Required: false, Default: "ms"},
		},
		Outputs: []PortDefinition{
			{Name: "timedOut", Type: "bool", Required: true},
		},
		Generator: generateTimeoutNode,
	})
}

// ============================================================================
// Generator Functions - Async
// ============================================================================

func generateWaitNode(ctx *GeneratorContext, node *Node) error {
	duration, err := ctx.ResolveInput(node, "duration")
	if err != nil {
		return err
	}

	// Get unit, default to "ms"
	unit := "ms"
	if unitVal, ok := node.Inputs["unit"]; ok {
		if unitMap, ok := unitVal.(map[string]interface{}); ok {
			if constVal, ok := unitMap["constant"]; ok {
				unit = fmt.Sprintf("%v", constVal)
			}
		} else if unitStr, ok := unitVal.(string); ok {
			if resolved, err := ctx.ResolveInput(node, "unit"); err == nil {
				unit = resolved
			} else {
				unit = unitStr
			}
		}
	}

	// Convert duration to time.Duration
	var durationExpr string
	switch unit {
	case "s":
		durationExpr = fmt.Sprintf("time.Duration(%s) * time.Second", duration)
	case "m":
		durationExpr = fmt.Sprintf("time.Duration(%s) * time.Minute", duration)
	case "ms":
		fallthrough
	default:
		durationExpr = fmt.Sprintf("time.Duration(%s) * time.Millisecond", duration)
	}

	ctx.WriteLine("select {")
	ctx.WriteLine("case <-time.After(%s):", durationExpr)
	ctx.Indent()
	ctx.WriteLine("// Wait completed")
	ctx.Dedent()
	ctx.WriteLine("case <-ctx.Done():")
	ctx.Indent()
	ctx.WriteLine("return ctx.Err()")
	ctx.Dedent()
	ctx.WriteLine("}")

	return nil
}

func generateWaitUntilNode(ctx *GeneratorContext, node *Node) error {
	condition, err := ctx.ResolveInput(node, "condition")
	if err != nil {
		return err
	}

	// Get check interval, default to 100ms
	checkInterval := "100"
	if intervalVal, ok := node.Inputs["checkInterval"]; ok {
		if intervalMap, ok := intervalVal.(map[string]interface{}); ok {
			if constVal, ok := intervalMap["constant"]; ok {
				checkInterval = fmt.Sprintf("%v", constVal)
			}
		}
	}

	// Get timeout, default to 30000ms (0 = no timeout)
	timeout := "30000"
	if timeoutVal, ok := node.Inputs["timeout"]; ok {
		if timeoutMap, ok := timeoutVal.(map[string]interface{}); ok {
			if constVal, ok := timeoutMap["constant"]; ok {
				timeout = fmt.Sprintf("%v", constVal)
			}
		}
	}

	timedOutVar := ctx.AllocateVariable(node, "timedOut", "bool")
	prefix := ctx.NodePrefix(node)

	ctx.WriteLine("%sdeadline := time.Now().Add(time.Duration(%s) * time.Millisecond)", prefix, timeout)
	ctx.WriteLine("%sticker := time.NewTicker(time.Duration(%s) * time.Millisecond)", prefix, checkInterval)
	ctx.WriteLine("defer %sticker.Stop()", prefix)
	ctx.WriteLine("%s := false", timedOutVar)
	ctx.WriteLine("")
	ctx.WriteLine("%sloop:", prefix)
	ctx.WriteLine("for {")
	ctx.Indent()
	ctx.WriteLine("if %s {", condition)
	ctx.Indent()
	ctx.WriteLine("break %sloop", prefix)
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.WriteLine("if %s > 0 && time.Now().After(%sdeadline) {", timeout, prefix)
	ctx.Indent()
	ctx.WriteLine("%s = true", timedOutVar)
	ctx.WriteLine("break %sloop", prefix)
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.WriteLine("select {")
	ctx.WriteLine("case <-%sticker.C:", prefix)
	ctx.Indent()
	ctx.WriteLine("// Check again")
	ctx.Dedent()
	ctx.WriteLine("case <-ctx.Done():")
	ctx.Indent()
	ctx.WriteLine("return ctx.Err()")
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.Dedent()
	ctx.WriteLine("}")

	return nil
}

func generateTimeoutNode(ctx *GeneratorContext, node *Node) error {
	duration, err := ctx.ResolveInput(node, "duration")
	if err != nil {
		return err
	}

	// Get unit, default to "ms"
	unit := "ms"
	if unitVal, ok := node.Inputs["unit"]; ok {
		if unitMap, ok := unitVal.(map[string]interface{}); ok {
			if constVal, ok := unitMap["constant"]; ok {
				unit = fmt.Sprintf("%v", constVal)
			}
		}
	}

	// Convert duration to time.Duration
	var durationExpr string
	switch unit {
	case "s":
		durationExpr = fmt.Sprintf("time.Duration(%s) * time.Second", duration)
	case "m":
		durationExpr = fmt.Sprintf("time.Duration(%s) * time.Minute", duration)
	case "ms":
		fallthrough
	default:
		durationExpr = fmt.Sprintf("time.Duration(%s) * time.Millisecond", duration)
	}

	timedOutVar := ctx.AllocateVariable(node, "timedOut", "bool")
	prefix := ctx.NodePrefix(node)

	ctx.WriteLine("%stimeoutCtx, %scancel := context.WithTimeout(ctx, %s)", prefix, prefix, durationExpr)
	ctx.WriteLine("defer %scancel()", prefix)
	ctx.WriteLine("%s := false", timedOutVar)
	ctx.WriteLine("_ = %stimeoutCtx // Use this context for nested operations", prefix)

	return nil
}
