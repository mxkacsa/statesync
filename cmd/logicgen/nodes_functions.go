package main

import (
	"fmt"
	"strings"
)

// ============================================================================
// Function Nodes - Reusable node-based functions
// ============================================================================
//
// Functions allow creating reusable logic that can be called from multiple handlers.
// This is similar to the effect pattern but for general-purpose logic.
//
// Example function definition:
//
//	{
//	    "name": "CalculateDamage",
//	    "parameters": [
//	        {"name": "attackPower", "type": "int"},
//	        {"name": "defense", "type": "int"}
//	    ],
//	    "returnType": "int",
//	    "nodes": [
//	        {"id": "subtract", "type": "Subtract", "inputs": {"left": "param:attackPower", "right": "param:defense"}},
//	        {"id": "clamp", "type": "Max", "inputs": {"left": "node:subtract:result", "right": {"constant": 0}}},
//	        {"id": "return", "type": "Return", "inputs": {"value": "node:clamp:result"}}
//	    ],
//	    "flow": [
//	        {"from": "start", "to": "subtract"},
//	        {"from": "subtract", "to": "clamp"},
//	        {"from": "clamp", "to": "return"}
//	    ]
//	}
//
// Called from a handler:
//
//	{
//	    "id": "calcDmg",
//	    "type": "CallFunction",
//	    "inputs": {
//	        "function": {"constant": "CalculateDamage"},
//	        "args": {
//	            "attackPower": "node:getAttacker:power",
//	            "defense": "node:getDefender:defense"
//	        }
//	    }
//	}
//
// ============================================================================

func init() {
	registerFunctionNodes()
}

func registerFunctionNodes() {
	// CallFunction - Calls a defined function with arguments
	MustRegisterNode(NodeDefinition{
		Type:        NodeCallFunction,
		Category:    "Functions",
		Description: "Calls a reusable function defined in the node graph",
		Inputs: []PortDefinition{
			{Name: "function", Type: "string", Required: true},
			{Name: "args", Type: "map", Required: false},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "any", Required: false},
		},
		Generator: generateCallFunctionNode,
	})

	// Return - Returns a value from a function
	MustRegisterNode(NodeDefinition{
		Type:        NodeReturn,
		Category:    "Functions",
		Description: "Returns a value from a function",
		Inputs: []PortDefinition{
			{Name: "value", Type: "any", Required: false},
		},
		Outputs:   []PortDefinition{},
		Generator: generateReturnNode,
	})
}

// ============================================================================
// Generator Functions - Functions
// ============================================================================

func generateCallFunctionNode(ctx *GeneratorContext, node *Node) error {
	funcNameRaw, err := ctx.ResolveInput(node, "function")
	if err != nil {
		return err
	}

	// Function name should be used as a function identifier
	funcName := strings.Trim(funcNameRaw, `"`)

	// Get function definition to understand parameters and return type
	funcDef := ctx.findFunctionDefinition(funcName)
	if funcDef == nil {
		return fmt.Errorf("function %s not found in node graph", funcName)
	}

	// Build argument list
	argsInput := node.Inputs["args"]
	var argValues []string

	if argsInput != nil {
		argsMap, ok := argsInput.(map[string]interface{})
		if ok {
			// Resolve arguments in parameter order
			for _, param := range funcDef.Parameters {
				if argValue, hasArg := argsMap[param.Name]; hasArg {
					resolved, err := resolveInputValue(ctx, argValue)
					if err != nil {
						return fmt.Errorf("failed to resolve arg %s: %w", param.Name, err)
					}
					argValues = append(argValues, resolved)
				} else {
					// Use zero value for missing optional parameters
					argValues = append(argValues, getZeroValue(param.Type))
				}
			}
		}
	}

	// Generate function call
	argsStr := strings.Join(argValues, ", ")

	if funcDef.ReturnType != "" {
		resultVar := ctx.AllocateVariable(node, "result", funcDef.ReturnType)
		ctx.WriteLine("%s := %s(state, session, %s)", resultVar, funcName, argsStr)
	} else {
		ctx.WriteLine("%s(state, session, %s)", funcName, argsStr)
	}

	return nil
}

func generateReturnNode(ctx *GeneratorContext, node *Node) error {
	// Check if we have a value to return
	if _, hasValue := node.Inputs["value"]; hasValue {
		value, err := ctx.ResolveInput(node, "value")
		if err != nil {
			return err
		}
		ctx.WriteLine("return %s", value)
	} else {
		// Void return
		ctx.WriteLine("return")
	}

	return nil
}

// Helper to find function definition in the node graph
func (ctx *GeneratorContext) findFunctionDefinition(name string) *FunctionDefinition {
	if ctx.gen.nodeGraph == nil {
		return nil
	}
	for i := range ctx.gen.nodeGraph.Functions {
		if ctx.gen.nodeGraph.Functions[i].Name == name {
			return &ctx.gen.nodeGraph.Functions[i]
		}
	}
	return nil
}

// Helper to get zero value for a type
func getZeroValue(typeName string) string {
	switch typeName {
	case "int", "int32", "int64", "float32", "float64":
		return "0"
	case "string":
		return `""`
	case "bool":
		return "false"
	default:
		return "nil"
	}
}
