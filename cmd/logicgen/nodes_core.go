package main

import (
	"fmt"
	"strings"
)

// ============================================================================
// Core Nodes - State, Array, Map, Control Flow, Variables, Object, String, Batch
// ============================================================================

func init() {
	registerStateNodes()
	registerArrayNodes()
	registerMapNodes()
	registerControlFlowNodes()
	registerVariableNodes()
	registerObjectNodes()
	registerStringNodes()
	registerBatchNodes()
}

// ============================================================================
// State Nodes
// ============================================================================

func registerStateNodes() {
	// GetCurrentState - Gets the current state object
	MustRegisterNode(NodeDefinition{
		Type:        NodeGetCurrentState,
		Category:    "State",
		Description: "Gets the current state object",
		Inputs:      []PortDefinition{},
		Outputs: []PortDefinition{
			{Name: "state", Type: "*GameState", Required: true},
		},
		Generator: generateGetCurrentStateNode,
	})

	// GetField - Gets a field value from the state
	MustRegisterNode(NodeDefinition{
		Type:        NodeGetField,
		Category:    "State",
		Description: "Gets a field value from the state",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "value", Type: "any", Required: true},
		},
		Generator: generateGetFieldNode,
	})

	// SetField - Sets a field value in the state
	MustRegisterNode(NodeDefinition{
		Type:        NodeSetField,
		Category:    "State",
		Description: "Sets a field value in the state",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
			{Name: "value", Type: "any", Required: true},
		},
		Outputs:   []PortDefinition{},
		Generator: generateSetFieldNode,
	})

	// GetPlayer - Gets a player by ID
	MustRegisterNode(NodeDefinition{
		Type:        NodeGetPlayer,
		Category:    "State",
		Description: "Gets a player by ID",
		Inputs: []PortDefinition{
			{Name: "playerID", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "player", Type: "*Player", Required: true},
		},
		Generator: generateGetPlayerNode,
	})
}

// ============================================================================
// Array Nodes
// ============================================================================

func registerArrayNodes() {
	// AddToArray - Adds an element to an array
	MustRegisterNode(NodeDefinition{
		Type:        NodeAddToArray,
		Category:    "Array",
		Description: "Adds an element to an array",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: false},
			{Name: "path", Type: "string", Required: false},
			{Name: "element", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]any", Required: true},
		},
		Generator: generateAddToArrayNode,
	})

	// ArrayAppend - Appends an element to a state array by path
	MustRegisterNode(NodeDefinition{
		Type:        NodeArrayAppend,
		Category:    "Array",
		Description: "Appends an element to a state array by path",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
			{Name: "item", Type: "any", Required: true},
		},
		Outputs:   []PortDefinition{},
		Generator: generateArrayAppendNode,
	})

	// RemoveFromArray - Removes elements from an array
	MustRegisterNode(NodeDefinition{
		Type:        NodeRemoveFromArray,
		Category:    "Array",
		Description: "Removes elements from an array",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
			{Name: "index", Type: "int", Required: true},
		},
		Outputs:   []PortDefinition{},
		Generator: generateRemoveFromArrayNode,
	})

	// FilterArray - Filters an array based on a predicate
	MustRegisterNode(NodeDefinition{
		Type:        NodeFilterArray,
		Category:    "Array",
		Description: "Filters an array based on a predicate",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
			{Name: "field", Type: "string", Required: false},
			{Name: "op", Type: "string", Required: false},
			{Name: "value", Type: "any", Required: false},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]any", Required: true},
		},
		Generator: generateFilterArrayNode,
	})

	// FindInArray - Finds an element in an array
	MustRegisterNode(NodeDefinition{
		Type:        NodeFindInArray,
		Category:    "Array",
		Description: "Finds an element in an array",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
			{Name: "field", Type: "string", Required: false},
			{Name: "value", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "item", Type: "any", Required: true},
			{Name: "index", Type: "int", Required: true},
		},
		Generator: generateFindInArrayNode,
	})

	// ArrayLength - Returns the length of an array
	MustRegisterNode(NodeDefinition{
		Type:        NodeArrayLength,
		Category:    "Array",
		Description: "Returns the length of an array",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "length", Type: "int", Required: true},
		},
		Generator: generateArrayLengthNode,
	})

	// ArrayAt - Gets element at index with bounds checking
	MustRegisterNode(NodeDefinition{
		Type:        NodeArrayAt,
		Category:    "Array",
		Description: "Gets element at index with bounds checking",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
			{Name: "index", Type: "int", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "item", Type: "any", Required: true},
			{Name: "valid", Type: "bool", Required: false}, // true if index is within bounds
		},
		Generator: generateArrayAtNode,
	})

	// MapArray - Transforms array elements
	MustRegisterNode(NodeDefinition{
		Type:        NodeMapArray,
		Category:    "Array",
		Description: "Transforms array elements",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]any", Required: true},
		},
		Generator: generateMapArrayNode,
	})
}

// ============================================================================
// Map Nodes
// ============================================================================

func registerMapNodes() {
	// SetMapValue - Sets a value in a map
	MustRegisterNode(NodeDefinition{
		Type:        NodeSetMapValue,
		Category:    "Map",
		Description: "Sets a value in a map",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: false},
			{Name: "key", Type: "string", Required: true},
			{Name: "value", Type: "any", Required: true},
		},
		Outputs:   []PortDefinition{},
		Generator: generateSetMapValueNode,
	})

	// GetMapValue - Gets a value from a map
	MustRegisterNode(NodeDefinition{
		Type:        NodeGetMapValue,
		Category:    "Map",
		Description: "Gets a value from a map",
		Inputs: []PortDefinition{
			{Name: "map", Type: "map", Required: true},
			{Name: "key", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "value", Type: "any", Required: true},
			{Name: "exists", Type: "bool", Required: true},
		},
		Generator: generateGetMapValueNode,
	})

	// RemoveMapKey - Removes a key from a map
	MustRegisterNode(NodeDefinition{
		Type:        NodeRemoveMapKey,
		Category:    "Map",
		Description: "Removes a key from a map",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: false},
			{Name: "key", Type: "string", Required: true},
		},
		Outputs:   []PortDefinition{},
		Generator: generateRemoveMapKeyNode,
	})

	// HasMapKey - Checks if a key exists in a map
	MustRegisterNode(NodeDefinition{
		Type:        NodeHasMapKey,
		Category:    "Map",
		Description: "Checks if a key exists in a map",
		Inputs: []PortDefinition{
			{Name: "map", Type: "map", Required: true},
			{Name: "key", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "exists", Type: "bool", Required: true},
		},
		Generator: generateHasMapKeyNode,
	})
}

// ============================================================================
// Control Flow Nodes
// ============================================================================

func registerControlFlowNodes() {
	// If - Conditional branch
	MustRegisterNode(NodeDefinition{
		Type:        NodeIf,
		Category:    "Control Flow",
		Description: "Conditional branch",
		Inputs: []PortDefinition{
			{Name: "condition", Type: "bool", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "true", Type: "flow", Required: false},
			{Name: "false", Type: "flow", Required: false},
		},
		Generator: generateIfNode,
	})

	// ForEach - Iterates over array elements
	MustRegisterNode(NodeDefinition{
		Type:        NodeForEach,
		Category:    "Control Flow",
		Description: "Iterates over array elements",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "item", Type: "any", Required: true},
			{Name: "index", Type: "int", Required: true},
		},
		Generator: generateForEachNode,
	})

	// While - Loops while condition is true
	MustRegisterNode(NodeDefinition{
		Type:        NodeWhile,
		Category:    "Control Flow",
		Description: "Loops while condition is true",
		Inputs: []PortDefinition{
			{Name: "condition", Type: "bool", Required: true},
		},
		Outputs:   []PortDefinition{},
		Generator: generateWhileNode,
	})
}

// ============================================================================
// Variable Nodes
// ============================================================================

func registerVariableNodes() {
	// SetVariable - Sets a local variable
	MustRegisterNode(NodeDefinition{
		Type:        NodeSetVariable,
		Category:    "Variables",
		Description: "Sets a local variable",
		Inputs: []PortDefinition{
			{Name: "name", Type: "string", Required: true},
			{Name: "value", Type: "any", Required: true},
		},
		Outputs:   []PortDefinition{},
		Generator: generateSetVariableNode,
	})

	// GetVariable - Gets a local variable
	MustRegisterNode(NodeDefinition{
		Type:        NodeGetVariable,
		Category:    "Variables",
		Description: "Gets a local variable",
		Inputs: []PortDefinition{
			{Name: "name", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "value", Type: "any", Required: true},
		},
		Generator: generateGetVariableNode,
	})

	// Constant - A constant value
	MustRegisterNode(NodeDefinition{
		Type:        NodeConstant,
		Category:    "Variables",
		Description: "A constant value",
		Inputs: []PortDefinition{
			{Name: "value", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "value", Type: "any", Required: true},
		},
		Generator: generateConstantNode,
	})
}

// ============================================================================
// Object Nodes
// ============================================================================

func registerObjectNodes() {
	// CreateStruct - Create a new struct instance
	MustRegisterNode(NodeDefinition{
		Type:        NodeCreateStruct,
		Category:    "Object",
		Description: "Create a new struct instance with field values",
		Inputs: []PortDefinition{
			{Name: "type", Type: "string", Required: true},
			{Name: "fields", Type: "map", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "any", Required: true},
		},
		Generator: generateCreateStructNode,
	})

	// UpdateStruct - Update specific fields on a struct
	MustRegisterNode(NodeDefinition{
		Type:        NodeUpdateStruct,
		Category:    "Object",
		Description: "Update specific fields on a struct",
		Inputs: []PortDefinition{
			{Name: "object", Type: "any", Required: true},
			{Name: "fields", Type: "map", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "any", Required: true},
		},
		Generator: generateUpdateStructNode,
	})

	// GetStructField - Get a field value from a struct
	MustRegisterNode(NodeDefinition{
		Type:        NodeGetStructField,
		Category:    "Object",
		Description: "Get a field value from a struct",
		Inputs: []PortDefinition{
			{Name: "object", Type: "any", Required: true},
			{Name: "field", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "value", Type: "any", Required: true},
		},
		Generator: generateGetStructFieldNode,
	})
}

// ============================================================================
// String Nodes
// ============================================================================

func registerStringNodes() {
	// Concat - Concatenates strings
	MustRegisterNode(NodeDefinition{
		Type:        NodeConcat,
		Category:    "String",
		Description: "Concatenates strings",
		Inputs: []PortDefinition{
			{Name: "strings", Type: "[]string", Required: false},
			{Name: "a", Type: "string", Required: false},
			{Name: "b", Type: "string", Required: false},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string", Required: true},
		},
		Generator: generateConcatNode,
	})

	// Format - Formats a string using fmt.Sprintf
	MustRegisterNode(NodeDefinition{
		Type:        NodeFormat,
		Category:    "String",
		Description: "Formats a string using fmt.Sprintf",
		Inputs: []PortDefinition{
			{Name: "format", Type: "string", Required: true},
			{Name: "args", Type: "any", Required: false},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string", Required: true},
		},
		Generator: generateFormatNode,
	})
}

// ============================================================================
// Batch Nodes
// ============================================================================

func registerBatchNodes() {
	// ForEachWhere - Iterates over array elements that match a condition
	MustRegisterNode(NodeDefinition{
		Type:        NodeForEachWhere,
		Category:    "Batch",
		Description: "Iterates over array elements that match a condition",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
			{Name: "field", Type: "string", Required: true},
			{Name: "op", Type: "string", Required: true},
			{Name: "value", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "item", Type: "any", Required: true},
			{Name: "index", Type: "int", Required: true},
		},
		Generator: generateForEachWhereNode,
	})

	// UpdateWhere - Updates all array elements matching a condition
	MustRegisterNode(NodeDefinition{
		Type:        NodeUpdateWhere,
		Category:    "Batch",
		Description: "Updates all array elements matching a condition",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
			{Name: "whereField", Type: "string", Required: true},
			{Name: "whereOp", Type: "string", Required: true},
			{Name: "whereValue", Type: "any", Required: true},
			{Name: "setField", Type: "string", Required: false},
			{Name: "setValue", Type: "any", Required: false},
			{Name: "updates", Type: "map", Required: false},
		},
		Outputs: []PortDefinition{
			{Name: "count", Type: "int", Required: true},
		},
		Generator: generateUpdateWhereNode,
	})

	// FindWhere - Finds the first element matching a condition
	MustRegisterNode(NodeDefinition{
		Type:        NodeFindWhere,
		Category:    "Batch",
		Description: "Finds the first element matching a condition",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
			{Name: "field", Type: "string", Required: true},
			{Name: "op", Type: "string", Required: true},
			{Name: "value", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "item", Type: "any", Required: true},
			{Name: "index", Type: "int", Required: true},
			{Name: "found", Type: "bool", Required: true},
		},
		Generator: generateFindWhereNode,
	})

	// CountWhere - Counts elements matching a condition
	MustRegisterNode(NodeDefinition{
		Type:        NodeCountWhere,
		Category:    "Batch",
		Description: "Counts elements matching a condition",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
			{Name: "field", Type: "string", Required: true},
			{Name: "op", Type: "string", Required: true},
			{Name: "value", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "count", Type: "int", Required: true},
		},
		Generator: generateCountWhereNode,
	})
}

// ============================================================================
// Generator Functions - State
// ============================================================================

func generateGetCurrentStateNode(ctx *GeneratorContext, node *Node) error {
	outputVar := ctx.AllocateVariable(node, "state", "*GameState")
	ctx.WriteLine("%s := state", outputVar)
	return nil
}

func generateGetFieldNode(ctx *GeneratorContext, node *Node) error {
	pathSrc, err := ctx.ResolveInput(node, "path")
	if err != nil {
		return err
	}

	pathStr := strings.Trim(pathSrc, `"`)
	outputVar := ctx.AllocateVariable(node, "value", "any")

	code, resultExpr, err := ctx.GeneratePathAccess(pathStr, false)
	if err != nil {
		return err
	}

	if code != "" {
		ctx.WriteRaw(code)
	}
	ctx.WriteLine("%s := %s", outputVar, resultExpr)

	return nil
}

func generateSetFieldNode(ctx *GeneratorContext, node *Node) error {
	pathSrc, err := ctx.ResolveInput(node, "path")
	if err != nil {
		return err
	}
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		return err
	}

	pathStr := strings.Trim(pathSrc, `"`)
	return ctx.GeneratePathSet(pathStr, valueSrc)
}

func generateGetPlayerNode(ctx *GeneratorContext, node *Node) error {
	playerIDSrc, err := ctx.ResolveInput(node, "playerID")
	if err != nil {
		return err
	}

	// Get player type info from schema
	playerTypeName := "Player"
	keyField := "ID"

	schema := ctx.GetSchema()
	if schema != nil && schema.RootType != nil {
		if info := schema.GetFieldAccessInfo(schema.RootType.Name, "Players"); info != nil {
			if info.ParsedType.ElemType != "" {
				playerTypeName = info.ParsedType.ElemType
			}
			if info.FieldDef.Key != "" {
				keyField = info.FieldDef.Key
			}
		}
	}

	outputVar := ctx.AllocateVariable(node, "player", "*"+playerTypeName)

	ctx.WriteLine("var %s *%s", outputVar, playerTypeName)
	ctx.WriteLine("for i := range state.Players() {")
	ctx.Indent()
	ctx.WriteLine("p := state.PlayersAt(i)")
	ctx.WriteLine("if p.%s == %s {", keyField, playerIDSrc)
	ctx.Indent()
	ctx.WriteLine("%s = &p", outputVar)
	ctx.WriteLine("break")
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.WriteLine("if %s == nil {", outputVar)
	ctx.Indent()
	ctx.WriteLine(`return fmt.Errorf("player not found: %%v", %s)`, playerIDSrc)
	ctx.Dedent()
	ctx.WriteLine("}")

	return nil
}

// ============================================================================
// Generator Functions - Array
// ============================================================================

func generateAddToArrayNode(ctx *GeneratorContext, node *Node) error {
	pathSrc, pathErr := ctx.ResolveInput(node, "path")
	elementSrc, elemErr := ctx.ResolveInput(node, "element")
	if elemErr != nil {
		return elemErr
	}

	if pathErr != nil {
		// Fallback: array might be a reference
		arraySrc, err := ctx.ResolveInput(node, "array")
		if err != nil {
			return pathErr
		}
		outputVar := ctx.AllocateVariable(node, "result", "[]any")
		ctx.WriteLine("%s := append(%s, %s)", outputVar, arraySrc, elementSrc)
		return nil
	}

	pathStr := strings.Trim(pathSrc, `"`)
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return err
	}

	schema := ctx.GetSchema()
	if len(parsedPath.Segments) == 1 {
		seg := parsedPath.Segments[0]
		if schema != nil && schema.RootType != nil {
			info := schema.GetFieldAccessInfo(schema.RootType.Name, seg.FieldName)
			if info != nil {
				ctx.WriteLine("state.%s(%s)", info.AppendName, elementSrc)
				return nil
			}
		}
		ctx.WriteLine("state.Append%s(%s)", seg.FieldName, elementSrc)
	} else {
		parentPath := &ParsedPath{Segments: parsedPath.Segments[:len(parsedPath.Segments)-1]}
		lastSeg := parsedPath.Segments[len(parsedPath.Segments)-1]

		code, parentExpr, err := ctx.GeneratePathAccess(parentPath.RawPath, false)
		if err != nil {
			return err
		}
		if code != "" {
			ctx.WriteRaw(code)
		}
		ctx.WriteLine("%s.Append%s(%s)", parentExpr, lastSeg.FieldName, elementSrc)
	}

	return nil
}

func generateArrayAppendNode(ctx *GeneratorContext, node *Node) error {
	pathSrc, err := ctx.ResolveInput(node, "path")
	if err != nil {
		return err
	}
	itemSrc, err := ctx.ResolveInput(node, "item")
	if err != nil {
		return err
	}

	pathStr := strings.Trim(pathSrc, `"`)
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return err
	}

	schema := ctx.GetSchema()
	if len(parsedPath.Segments) == 1 {
		seg := parsedPath.Segments[0]
		if schema != nil && schema.RootType != nil {
			info := schema.GetFieldAccessInfo(schema.RootType.Name, seg.FieldName)
			if info != nil {
				ctx.WriteLine("state.%s(%s)", info.AppendName, itemSrc)
				return nil
			}
		}
		ctx.WriteLine("state.Append%s(%s)", seg.FieldName, itemSrc)
	} else {
		parentPath := &ParsedPath{Segments: parsedPath.Segments[:len(parsedPath.Segments)-1]}
		lastSeg := parsedPath.Segments[len(parsedPath.Segments)-1]

		code, parentExpr, err := ctx.GeneratePathAccess(parentPath.RawPath, false)
		if err != nil {
			return err
		}
		if code != "" {
			ctx.WriteRaw(code)
		}
		ctx.WriteLine("%s.Append%s(%s)", parentExpr, lastSeg.FieldName, itemSrc)
	}

	return nil
}

func generateRemoveFromArrayNode(ctx *GeneratorContext, node *Node) error {
	pathSrc, err := ctx.ResolveInput(node, "path")
	if err != nil {
		return err
	}
	indexSrc, err := ctx.ResolveInput(node, "index")
	if err != nil {
		return err
	}

	pathStr := strings.Trim(pathSrc, `"`)
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return err
	}

	schema := ctx.GetSchema()
	if len(parsedPath.Segments) == 1 {
		seg := parsedPath.Segments[0]
		if schema != nil && schema.RootType != nil {
			info := schema.GetFieldAccessInfo(schema.RootType.Name, seg.FieldName)
			if info != nil {
				ctx.WriteLine("state.%s(%s)", info.RemoveAtName, indexSrc)
				return nil
			}
		}
		ctx.WriteLine("state.Remove%sAt(%s)", seg.FieldName, indexSrc)
	}

	return nil
}

func generateFilterArrayNode(ctx *GeneratorContext, node *Node) error {
	arraySrc, err := ctx.ResolveInput(node, "array")
	if err != nil {
		return err
	}

	fieldName := ctx.ResolveInputOptional(node, "field", "")
	op := ctx.ResolveInputOptional(node, "op", "")
	valueSrc := ctx.ResolveInputOptional(node, "value", "")

	outputVar := ctx.AllocateVariable(node, "result", "[]any")

	ctx.WriteLine("var %s []any", outputVar)
	ctx.WriteLine("for _, _item := range %s {", arraySrc)
	ctx.Indent()

	if fieldName != "" && op != "" && valueSrc != "" {
		field := strings.Trim(fieldName, `"`)
		operator := strings.Trim(op, `"`)
		ctx.WriteLine("if _item.%s %s %s {", field, operator, valueSrc)
	} else {
		ctx.WriteLine("if true { // TODO: add filter condition")
	}

	ctx.Indent()
	ctx.WriteLine("%s = append(%s, _item)", outputVar, outputVar)
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.Dedent()
	ctx.WriteLine("}")

	return nil
}

func generateFindInArrayNode(ctx *GeneratorContext, node *Node) error {
	arraySrc, err := ctx.ResolveInput(node, "array")
	if err != nil {
		return err
	}

	fieldName := ctx.ResolveInputOptional(node, "field", "")
	valueSrc := ctx.ResolveInputOptional(node, "value", "")

	outputVar := ctx.AllocateVariable(node, "item", "any")
	indexVar := ctx.AllocateVariable(node, "index", "int")

	ctx.WriteLine("%s := -1", indexVar)
	ctx.WriteLine("var %s any", outputVar)
	ctx.WriteLine("for _i, _item := range %s {", arraySrc)
	ctx.Indent()

	if fieldName != "" && valueSrc != "" {
		field := strings.Trim(fieldName, `"`)
		ctx.WriteLine("if _item.%s == %s {", field, valueSrc)
	} else {
		ctx.WriteLine("if _item == %s {", valueSrc)
	}

	ctx.Indent()
	ctx.WriteLine("%s = _i", indexVar)
	ctx.WriteLine("%s = _item", outputVar)
	ctx.WriteLine("break")
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.Dedent()
	ctx.WriteLine("}")

	return nil
}

func generateArrayLengthNode(ctx *GeneratorContext, node *Node) error {
	arraySrc, err := ctx.ResolveInput(node, "array")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "length", "int")
	ctx.WriteLine("%s := len(%s)", outputVar, arraySrc)
	return nil
}

func generateArrayAtNode(ctx *GeneratorContext, node *Node) error {
	arraySrc, err := ctx.ResolveInput(node, "array")
	if err != nil {
		return err
	}
	indexSrc, err := ctx.ResolveInput(node, "index")
	if err != nil {
		return err
	}

	// Generate bounds-checked access
	validVar := ctx.AllocateVariable(node, "valid", "bool")
	ctx.WriteLine("%s := %s >= 0 && %s < len(%s)", validVar, indexSrc, indexSrc, arraySrc)

	outputVar := ctx.AllocateVariable(node, "item", "any")
	ctx.WriteLine("var %s interface{}", outputVar)
	ctx.WriteLine("if %s {", validVar)
	ctx.Indent()
	ctx.WriteLine("%s = %s[%s]", outputVar, arraySrc, indexSrc)
	ctx.Dedent()
	ctx.WriteLine("}")

	return nil
}

func generateMapArrayNode(ctx *GeneratorContext, node *Node) error {
	arraySrc, err := ctx.ResolveInput(node, "array")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "result", "[]any")
	ctx.WriteLine("%s := make([]any, len(%s))", outputVar, arraySrc)
	ctx.WriteLine("for _i, _item := range %s {", arraySrc)
	ctx.Indent()
	ctx.WriteLine("_ = _item")
	ctx.WriteLine("%s[_i] = _item // TODO: transform", outputVar)
	ctx.Dedent()
	ctx.WriteLine("}")
	return nil
}

// ============================================================================
// Generator Functions - Map
// ============================================================================

func generateSetMapValueNode(ctx *GeneratorContext, node *Node) error {
	pathSrc := ctx.ResolveInputOptional(node, "path", "")
	keySrc, err := ctx.ResolveInput(node, "key")
	if err != nil {
		return err
	}
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		return err
	}

	if pathSrc != "" {
		pathStr := strings.Trim(pathSrc, `"`)
		parsedPath, _ := ParsePath(pathStr)

		schema := ctx.GetSchema()
		if len(parsedPath.Segments) == 1 && schema != nil && schema.RootType != nil {
			seg := parsedPath.Segments[0]
			info := schema.GetFieldAccessInfo(schema.RootType.Name, seg.FieldName)
			if info != nil {
				ctx.WriteLine("state.%s(%s, %s)", info.SetKeyName, keySrc, valueSrc)
				return nil
			}
		}
	}

	ctx.WriteLine("// TODO: SetMapValue for path %s", pathSrc)
	return nil
}

func generateGetMapValueNode(ctx *GeneratorContext, node *Node) error {
	mapSrc, err := ctx.ResolveInput(node, "map")
	if err != nil {
		return err
	}
	keySrc, err := ctx.ResolveInput(node, "key")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "value", "any")
	existsVar := ctx.AllocateVariable(node, "exists", "bool")
	ctx.WriteLine("%s, %s := %s[%s]", outputVar, existsVar, mapSrc, keySrc)
	ctx.WriteLine("_ = %s", existsVar)
	return nil
}

func generateRemoveMapKeyNode(ctx *GeneratorContext, node *Node) error {
	pathSrc := ctx.ResolveInputOptional(node, "path", "")
	keySrc, err := ctx.ResolveInput(node, "key")
	if err != nil {
		return err
	}

	if pathSrc != "" {
		pathStr := strings.Trim(pathSrc, `"`)
		parsedPath, _ := ParsePath(pathStr)

		schema := ctx.GetSchema()
		if len(parsedPath.Segments) == 1 && schema != nil && schema.RootType != nil {
			seg := parsedPath.Segments[0]
			info := schema.GetFieldAccessInfo(schema.RootType.Name, seg.FieldName)
			if info != nil {
				ctx.WriteLine("state.%s(%s)", info.DeleteKeyName, keySrc)
				return nil
			}
		}
	}

	ctx.WriteLine("// TODO: RemoveMapKey for path %s", pathSrc)
	return nil
}

func generateHasMapKeyNode(ctx *GeneratorContext, node *Node) error {
	mapSrc, err := ctx.ResolveInput(node, "map")
	if err != nil {
		return err
	}
	keySrc, err := ctx.ResolveInput(node, "key")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "exists", "bool")
	ctx.WriteLine("_, %s := %s[%s]", outputVar, mapSrc, keySrc)
	return nil
}

// ============================================================================
// Generator Functions - Control Flow
// ============================================================================

func generateIfNode(ctx *GeneratorContext, node *Node) error {
	conditionSrc, err := ctx.ResolveInput(node, "condition")
	if err != nil {
		return err
	}

	ctx.WriteLine("if %s {", conditionSrc)
	ctx.Indent()

	// Find and generate true branch
	flowEdges := ctx.GetFlowEdgesFrom(node.ID)
	var trueBranch, falseBranch string
	for _, edge := range flowEdges {
		if edge.Label == "true" {
			trueBranch = edge.To
		} else if edge.Label == "false" {
			falseBranch = edge.To
		}
	}

	if trueBranch != "" {
		if err := ctx.GenerateBodyChain(trueBranch); err != nil {
			return err
		}
	}

	ctx.Dedent()

	if falseBranch != "" {
		ctx.WriteLine("} else {")
		ctx.Indent()
		if err := ctx.GenerateBodyChain(falseBranch); err != nil {
			return err
		}
		ctx.Dedent()
	}

	ctx.WriteLine("}")
	return nil
}

func generateForEachNode(ctx *GeneratorContext, node *Node) error {
	arraySrc, err := ctx.ResolveInput(node, "array")
	if err != nil {
		return err
	}

	itemVar := ctx.AllocateVariable(node, "item", "any")
	indexVar := ctx.AllocateVariable(node, "index", "int")

	ctx.PushLoop(node.ID, itemVar, indexVar, arraySrc)

	ctx.WriteLine("for %s, %s := range %s {", indexVar, itemVar, arraySrc)
	ctx.Indent()

	// Find and generate body nodes
	flowEdges := ctx.GetFlowEdgesFrom(node.ID)
	for _, edge := range flowEdges {
		if edge.Label == "body" {
			bodyNode := ctx.FindNode(edge.To)
			if bodyNode != nil {
				ctx.GenerateBodyChain(edge.To)
			}
			break
		}
	}

	ctx.Dedent()
	ctx.WriteLine("}")

	ctx.PopLoop()

	return nil
}

func generateWhileNode(ctx *GeneratorContext, node *Node) error {
	conditionSrc, err := ctx.ResolveInput(node, "condition")
	if err != nil {
		return err
	}

	ctx.WriteLine("for %s {", conditionSrc)
	ctx.Indent()

	// Find and generate body nodes
	flowEdges := ctx.GetFlowEdgesFrom(node.ID)
	for _, edge := range flowEdges {
		if edge.Label == "body" {
			ctx.GenerateBodyChain(edge.To)
			break
		}
	}

	ctx.Dedent()
	ctx.WriteLine("}")
	return nil
}

// ============================================================================
// Generator Functions - Variables
// ============================================================================

func generateSetVariableNode(ctx *GeneratorContext, node *Node) error {
	nameSrc, err := ctx.ResolveInput(node, "name")
	if err != nil {
		return err
	}
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		return err
	}

	varName := strings.Trim(nameSrc, `"`)
	ctx.WriteLine("%s := %s", varName, valueSrc)
	return nil
}

func generateGetVariableNode(ctx *GeneratorContext, node *Node) error {
	nameSrc, err := ctx.ResolveInput(node, "name")
	if err != nil {
		return err
	}

	varName := strings.Trim(nameSrc, `"`)
	outputVar := ctx.AllocateVariable(node, "value", "any")
	ctx.WriteLine("%s := %s", outputVar, varName)
	return nil
}

func generateConstantNode(ctx *GeneratorContext, node *Node) error {
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "value", "any")
	ctx.WriteLine("%s := %s", outputVar, valueSrc)
	return nil
}

// ============================================================================
// Generator Functions - Object
// ============================================================================

func generateCreateStructNode(ctx *GeneratorContext, node *Node) error {
	typeSrc, err := ctx.ResolveInput(node, "type")
	if err != nil {
		return err
	}

	typeName := strings.Trim(typeSrc, `"`)
	outputVar := ctx.AllocateVariable(node, "result", typeName)

	fieldsInput, hasFields := node.Inputs["fields"]

	ctx.WriteLine("%s := %s{", outputVar, typeName)
	ctx.Indent()

	if hasFields {
		if fieldsMap, ok := fieldsInput.(map[string]interface{}); ok {
			for fieldName, fieldValue := range fieldsMap {
				valueSrc, err := resolveInputValue(ctx, fieldValue)
				if err != nil {
					ctx.WriteLine("%s: %v,", fieldName, fieldValue)
				} else {
					ctx.WriteLine("%s: %s,", fieldName, valueSrc)
				}
			}
		}
	}

	ctx.Dedent()
	ctx.WriteLine("}")

	return nil
}

func generateUpdateStructNode(ctx *GeneratorContext, node *Node) error {
	objectSrc, err := ctx.ResolveInput(node, "object")
	if err != nil {
		return err
	}

	// Store reference for other nodes
	if nodeOutputs := ctx.GetNodeOutput(node.ID, "result"); nodeOutputs == "" {
		ctx.AllocateVariable(node, "result", "any")
	}

	fieldsInput, hasFields := node.Inputs["fields"]
	if hasFields {
		if fieldsMap, ok := fieldsInput.(map[string]interface{}); ok {
			for fieldName, fieldValue := range fieldsMap {
				valueSrc, err := resolveInputValue(ctx, fieldValue)
				if err != nil {
					ctx.WriteLine("%s.%s = %v", objectSrc, fieldName, fieldValue)
				} else {
					ctx.WriteLine("%s.%s = %s", objectSrc, fieldName, valueSrc)
				}
			}
		}
	}

	return nil
}

func generateGetStructFieldNode(ctx *GeneratorContext, node *Node) error {
	objectSrc, err := ctx.ResolveInput(node, "object")
	if err != nil {
		return err
	}
	fieldSrc, err := ctx.ResolveInput(node, "field")
	if err != nil {
		return err
	}

	fieldName := strings.Trim(fieldSrc, `"`)
	outputVar := ctx.AllocateVariable(node, "value", "any")

	ctx.WriteLine("%s := %s.%s", outputVar, objectSrc, fieldName)

	return nil
}

// ============================================================================
// Generator Functions - String
// ============================================================================

func generateConcatNode(ctx *GeneratorContext, node *Node) error {
	stringsSrc, err := ctx.ResolveInput(node, "strings")
	if err != nil {
		// Try a, b format
		aSrc, aErr := ctx.ResolveInput(node, "a")
		if aErr != nil {
			return err
		}
		bSrc, bErr := ctx.ResolveInput(node, "b")
		if bErr != nil {
			return bErr
		}
		outputVar := ctx.AllocateVariable(node, "result", "string")
		ctx.WriteLine("%s := %s + %s", outputVar, aSrc, bSrc)
		return nil
	}

	outputVar := ctx.AllocateVariable(node, "result", "string")
	ctx.WriteLine("%s := strings.Join(%s, \"\")", outputVar, stringsSrc)
	return nil
}

func generateFormatNode(ctx *GeneratorContext, node *Node) error {
	formatSrc, err := ctx.ResolveInput(node, "format")
	if err != nil {
		return err
	}
	argsSrc := ctx.ResolveInputOptional(node, "args", "")

	outputVar := ctx.AllocateVariable(node, "result", "string")
	if argsSrc != "" {
		ctx.WriteLine("%s := fmt.Sprintf(%s, %s...)", outputVar, formatSrc, argsSrc)
	} else {
		ctx.WriteLine("%s := fmt.Sprintf(%s)", outputVar, formatSrc)
	}
	return nil
}

// ============================================================================
// Generator Functions - Batch
// ============================================================================

func generateForEachWhereNode(ctx *GeneratorContext, node *Node) error {
	arraySrc, err := ctx.ResolveInput(node, "array")
	if err != nil {
		return err
	}
	fieldName, err := ctx.ResolveInput(node, "field")
	if err != nil {
		return err
	}
	op, err := ctx.ResolveInput(node, "op")
	if err != nil {
		return err
	}
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		return err
	}

	field := strings.Trim(fieldName, `"`)
	operator := strings.Trim(op, `"`)

	itemVar := ctx.AllocateVariable(node, "item", "any")
	indexVar := ctx.AllocateVariable(node, "index", "int")

	isStateArray := strings.HasPrefix(arraySrc, "state.")

	ctx.PushLoop(node.ID, itemVar, indexVar, arraySrc)

	if isStateArray {
		arrayField := strings.TrimSuffix(strings.TrimPrefix(arraySrc, "state."), "()")
		ctx.WriteLine("for %s := 0; %s < state.%sLen(); %s++ {", indexVar, indexVar, arrayField, indexVar)
		ctx.Indent()
		ctx.WriteLine("%s := state.%sAt(%s)", itemVar, arrayField, indexVar)
	} else {
		ctx.WriteLine("for %s, %s := range %s {", indexVar, itemVar, arraySrc)
		ctx.Indent()
	}

	ctx.WriteLine("if %s.%s %s %s {", itemVar, field, operator, valueSrc)
	ctx.Indent()

	// Find and generate body
	flowEdges := ctx.GetFlowEdgesFrom(node.ID)
	for _, edge := range flowEdges {
		if edge.Label == "body" {
			ctx.GenerateBodyChain(edge.To)
			break
		}
	}

	// For state arrays, update the item
	if isStateArray {
		arrayField := strings.TrimSuffix(strings.TrimPrefix(arraySrc, "state."), "()")
		ctx.WriteLine("state.Update%sAt(%s, %s)", arrayField, indexVar, itemVar)
	}

	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.Dedent()
	ctx.WriteLine("}")

	ctx.PopLoop()

	return nil
}

func generateUpdateWhereNode(ctx *GeneratorContext, node *Node) error {
	pathSrc, err := ctx.ResolveInput(node, "path")
	if err != nil {
		return err
	}
	whereField, err := ctx.ResolveInput(node, "whereField")
	if err != nil {
		return err
	}
	whereOp, err := ctx.ResolveInput(node, "whereOp")
	if err != nil {
		return err
	}
	whereValue, err := ctx.ResolveInput(node, "whereValue")
	if err != nil {
		return err
	}

	updatesMap, hasUpdates := node.Inputs["updates"]
	var singleFieldName, singleFieldValue string
	if !hasUpdates {
		setField, err := ctx.ResolveInput(node, "setField")
		if err != nil {
			return err
		}
		setValue, err := ctx.ResolveInput(node, "setValue")
		if err != nil {
			return err
		}
		singleFieldName = strings.Trim(setField, `"`)
		singleFieldValue = setValue
	}

	pathStr := strings.Trim(pathSrc, `"`)
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return err
	}

	wField := strings.Trim(whereField, `"`)
	wOp := strings.Trim(whereOp, `"`)

	countVar := ctx.AllocateVariable(node, "count", "int")
	ctx.WriteLine("%s := 0", countVar)

	generateUpdates := func() {
		if hasUpdates {
			if um, ok := updatesMap.(map[string]interface{}); ok {
				for fieldName, fieldValue := range um {
					valueSrc, err := resolveInputValue(ctx, fieldValue)
					if err != nil {
						ctx.WriteLine("_item.%s = %v", fieldName, fieldValue)
					} else {
						ctx.WriteLine("_item.%s = %s", fieldName, valueSrc)
					}
				}
			}
		} else {
			ctx.WriteLine("_item.%s = %s", singleFieldName, singleFieldValue)
		}
	}

	schema := ctx.GetSchema()
	if len(parsedPath.Segments) == 1 {
		seg := parsedPath.Segments[0]
		var info *FieldAccessInfo
		if schema != nil && schema.RootType != nil {
			info = schema.GetFieldAccessInfo(schema.RootType.Name, seg.FieldName)
		}

		if info != nil {
			ctx.WriteLine("for _i := 0; _i < state.%s(); _i++ {", info.LenName)
			ctx.Indent()
			ctx.WriteLine("_item := state.%s(_i)", info.AtName)
			ctx.WriteLine("if _item.%s %s %s {", wField, wOp, whereValue)
			ctx.Indent()
			generateUpdates()
			ctx.WriteLine("state.%s(_i, _item)", info.UpdateAtName)
			ctx.WriteLine("%s++", countVar)
			ctx.Dedent()
			ctx.WriteLine("}")
			ctx.Dedent()
			ctx.WriteLine("}")
		} else {
			ctx.WriteLine("for _i := 0; _i < state.%sLen(); _i++ {", seg.FieldName)
			ctx.Indent()
			ctx.WriteLine("_item := state.%sAt(_i)", seg.FieldName)
			ctx.WriteLine("if _item.%s %s %s {", wField, wOp, whereValue)
			ctx.Indent()
			generateUpdates()
			ctx.WriteLine("state.Update%sAt(_i, _item)", seg.FieldName)
			ctx.WriteLine("%s++", countVar)
			ctx.Dedent()
			ctx.WriteLine("}")
			ctx.Dedent()
			ctx.WriteLine("}")
		}
	} else {
		parentPath := &ParsedPath{Segments: parsedPath.Segments[:len(parsedPath.Segments)-1]}
		lastSeg := parsedPath.Segments[len(parsedPath.Segments)-1]

		code, parentExpr, err := ctx.GeneratePathAccess(parentPath.RawPath, false)
		if err != nil {
			return err
		}
		if code != "" {
			ctx.WriteRaw(code)
		}

		ctx.WriteLine("for _i := 0; _i < %s.%sLen(); _i++ {", parentExpr, lastSeg.FieldName)
		ctx.Indent()
		ctx.WriteLine("_item := %s.%sAt(_i)", parentExpr, lastSeg.FieldName)
		ctx.WriteLine("if _item.%s %s %s {", wField, wOp, whereValue)
		ctx.Indent()
		generateUpdates()
		ctx.WriteLine("%s.Update%sAt(_i, _item)", parentExpr, lastSeg.FieldName)
		ctx.WriteLine("%s++", countVar)
		ctx.Dedent()
		ctx.WriteLine("}")
		ctx.Dedent()
		ctx.WriteLine("}")
	}

	return nil
}

func generateFindWhereNode(ctx *GeneratorContext, node *Node) error {
	arraySrc, err := ctx.ResolveInput(node, "array")
	if err != nil {
		return err
	}
	fieldName, err := ctx.ResolveInput(node, "field")
	if err != nil {
		return err
	}
	op, err := ctx.ResolveInput(node, "op")
	if err != nil {
		return err
	}
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		return err
	}

	field := strings.Trim(fieldName, `"`)
	operator := strings.Trim(op, `"`)

	itemVar := ctx.AllocateVariable(node, "item", "any")
	indexVar := ctx.AllocateVariable(node, "index", "int")
	foundVar := ctx.AllocateVariable(node, "found", "bool")

	ctx.WriteLine("%s := -1", indexVar)
	ctx.WriteLine("var %s any", itemVar)
	ctx.WriteLine("%s := false", foundVar)
	ctx.WriteLine("for _i, _item := range %s {", arraySrc)
	ctx.Indent()
	ctx.WriteLine("if _item.%s %s %s {", field, operator, valueSrc)
	ctx.Indent()
	ctx.WriteLine("%s = _i", indexVar)
	ctx.WriteLine("%s = _item", itemVar)
	ctx.WriteLine("%s = true", foundVar)
	ctx.WriteLine("break")
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.Dedent()
	ctx.WriteLine("}")

	return nil
}

func generateCountWhereNode(ctx *GeneratorContext, node *Node) error {
	arraySrc, err := ctx.ResolveInput(node, "array")
	if err != nil {
		return err
	}
	fieldName, err := ctx.ResolveInput(node, "field")
	if err != nil {
		return err
	}
	op, err := ctx.ResolveInput(node, "op")
	if err != nil {
		return err
	}
	valueSrc, err := ctx.ResolveInput(node, "value")
	if err != nil {
		return err
	}

	field := strings.Trim(fieldName, `"`)
	operator := strings.Trim(op, `"`)

	countVar := ctx.AllocateVariable(node, "count", "int")

	ctx.WriteLine("%s := 0", countVar)
	ctx.WriteLine("for _, _item := range %s {", arraySrc)
	ctx.Indent()
	ctx.WriteLine("if _item.%s %s %s {", field, operator, valueSrc)
	ctx.Indent()
	ctx.WriteLine("%s++", countVar)
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.Dedent()
	ctx.WriteLine("}")

	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// resolveInputValue resolves an input value directly (for struct fields)
func resolveInputValue(ctx *GeneratorContext, v interface{}) (string, error) {
	source, err := ParseValueSource(v)
	if err != nil {
		return "", err
	}

	switch source.Type {
	case "constant":
		return formatConstantValue(source.Value), nil
	case "ref":
		// For refs, we need to resolve the reference
		refStr, ok := source.Value.(string)
		if !ok {
			return "", fmt.Errorf("invalid ref value: %v", source.Value)
		}
		return ctx.ResolveReference(refStr)
	default:
		return "", fmt.Errorf("unknown source type: %s", source.Type)
	}
}

func formatConstantValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return `"` + v + `"`
	case int:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case uint:
		return fmt.Sprintf("%d", v)
	case uint32:
		return fmt.Sprintf("%d", v)
	case uint64:
		return fmt.Sprintf("%d", v)
	case float32:
		return fmt.Sprintf("%v", v)
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}
