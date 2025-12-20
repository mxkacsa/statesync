package main

// ============================================================================
// Code Generator - Design Notes and Limitations
// ============================================================================
//
// This generator creates Go code from a visual node graph (similar to Unreal Blueprints).
// The generated code integrates with the statesync package for state tracking and sync.
//
// THREAD SAFETY:
// The generated handlers are thread-safe when used with TrackedSession/TrackedState.
// All state modifications go through session.State().Update() which uses internal locks.
// There is no need for additional locking in generated code.
//
// PATH ACCESS:
// - Paths like "Players[playerID:ID].Score" are parsed at CODE GENERATION time, not runtime
// - The generated code uses direct field accessors (e.g., state.PlayersAt(idx).Score())
// - Key lookups (e.g., [playerID:ID]) include automatic not-found error handling
//
// FILTERS:
// - Filters use ShallowClone() for performance - nested slices/maps are shared
// - Filters should only hide/mask data, not deeply modify nested structures
// - For deep modifications, implement DeepClone in the schema or use immutable patterns
//
// TYPE INFERENCE:
// - Type names are inferred from the schema where possible
// - Falls back to defaults (e.g., "Player") when schema info is unavailable
// - The schemagen tool should be used to generate matching type definitions
//
// WAIT NODES:
// - Wait/WaitUntil nodes use context.Context for cancellation
// - Wait state is in-memory only - does not survive server restart
// - For long-duration waits (minutes+), use external scheduling instead
//
// EXTENSIBILITY (Custom Nodes):
// External packages can register custom node types using the registry API.
// See registry.go for the full API. Example usage in your framework:
//
//     import "github.com/mxkacsa/statesync/cmd/logicgen"
//
//     func init() {
//         logicgen.MustRegisterNode(logicgen.NodeDefinition{
//             Type:        "MyCustomNode",
//             Category:    "Custom",
//             Description: "Does something custom",
//             Inputs: []logicgen.PortDefinition{
//                 {Name: "value", Type: "string", Required: true},
//             },
//             Outputs: []logicgen.PortDefinition{
//                 {Name: "result", Type: "string"},
//             },
//             Generator: func(ctx *logicgen.GeneratorContext, node *logicgen.Node) error {
//                 value, _ := ctx.ResolveInput(node, "value")
//                 output := ctx.AllocateVariable(node, "result", "string")
//                 ctx.WriteLine("%s := process(%s)", output, value)
//                 return nil
//             },
//         })
//         // Optional: register imports needed by your custom nodes
//         logicgen.RegisterNodeImport("github.com/your/package")
//     }
//
// GeneratorContext methods available for custom node generators:
// - ResolveInput(node, inputName) - Get Go expression for input value
// - ResolveInputOptional(node, name, default) - Get input or default
// - AllocateVariable(node, outputName, type) - Allocate output variable
// - WriteLine(format, args...) - Write a line with proper indentation
// - WriteRaw(code) - Write raw code without indentation
// - Indent() / Dedent() - Manage indentation level
// - AddImport(path) - Add an import to the generated file
// - GetSchema() - Access schema information
// - GetCurrentHandler() - Access current handler being generated
// - IsDebugMode() - Check if debug mode is enabled
//
// ============================================================================

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"
)

// CodeGenerator generates Go code from a node graph
type CodeGenerator struct {
	buf            *bytes.Buffer
	indent         int
	nodeGraph      *NodeGraph
	schema         *SchemaContext
	currentHandler *EventHandler
	variables      map[string]string            // variable name -> type
	nodeOutputs    map[string]map[string]string // nodeID -> output name -> variable name
	loopStack      []loopContext                // Stack for nested loops
	generatedNodes map[string]bool              // Track which nodes have been generated (for control flow)
	debugMode      bool                         // Generate debug hooks
	contextMode    bool                         // Generate with context support (for Wait nodes)
	hasWaitNodes   bool                         // Track if current handler has wait nodes
	extraImports   map[string]bool              // Extra imports added by custom nodes
}

// loopContext tracks loop variables for ForEach/While
type loopContext struct {
	nodeID    string
	itemVar   string
	indexVar  string
	arrayExpr string
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(graph *NodeGraph, schema *SchemaContext) *CodeGenerator {
	return &CodeGenerator{
		buf:          &bytes.Buffer{},
		nodeGraph:    graph,
		schema:       schema,
		variables:    make(map[string]string),
		nodeOutputs:  make(map[string]map[string]string),
		loopStack:    []loopContext{},
		debugMode:    false,
		extraImports: make(map[string]bool),
	}
}

// NewCodeGeneratorWithDebug creates a new code generator with debug mode
func NewCodeGeneratorWithDebug(graph *NodeGraph, schema *SchemaContext, debug bool) *CodeGenerator {
	g := &CodeGenerator{
		buf:          &bytes.Buffer{},
		nodeGraph:    graph,
		schema:       schema,
		variables:    make(map[string]string),
		nodeOutputs:  make(map[string]map[string]string),
		loopStack:    []loopContext{},
		debugMode:    debug,
		contextMode:  false,
		extraImports: make(map[string]bool),
	}
	// Auto-detect if we need context mode
	g.contextMode = g.detectContextMode()
	return g
}

// NewCodeGeneratorFull creates a new code generator with all options
func NewCodeGeneratorFull(graph *NodeGraph, schema *SchemaContext, debug, context bool) *CodeGenerator {
	return &CodeGenerator{
		buf:          &bytes.Buffer{},
		nodeGraph:    graph,
		schema:       schema,
		variables:    make(map[string]string),
		nodeOutputs:  make(map[string]map[string]string),
		loopStack:    []loopContext{},
		debugMode:    debug,
		contextMode:  context,
		extraImports: make(map[string]bool),
	}
}

// detectContextMode checks if any handler uses Wait nodes
func (g *CodeGenerator) detectContextMode() bool {
	for _, handler := range g.nodeGraph.Handlers {
		for _, node := range handler.Nodes {
			switch NodeType(node.Type) {
			case NodeWait, NodeWaitUntil, NodeTimeout:
				return true
			}
		}
	}
	return false
}

// handlerHasWaitNodes checks if a specific handler uses Wait nodes
func (g *CodeGenerator) handlerHasWaitNodes(handler *EventHandler) bool {
	for _, node := range handler.Nodes {
		switch NodeType(node.Type) {
		case NodeWait, NodeWaitUntil, NodeTimeout:
			return true
		}
	}
	return false
}

// addImport adds an import path to be included in the generated file.
// This is used by custom node generators to add required imports.
func (g *CodeGenerator) addImport(importPath string) {
	if g.extraImports == nil {
		g.extraImports = make(map[string]bool)
	}
	g.extraImports[importPath] = true
}

// Generate generates Go code from the node graph
func (g *CodeGenerator) Generate() ([]byte, error) {
	// Phase 1: Generate all handlers and filters first to collect extraImports
	handlerBuf := g.buf
	g.buf = &bytes.Buffer{}

	// Generate allowedPlayers maps for handlers that need them
	for i := range g.nodeGraph.Handlers {
		handler := &g.nodeGraph.Handlers[i]
		if handler.Permissions != nil && len(handler.Permissions.AllowedPlayers) > 0 {
			g.writeLine("var _%s_allowedPlayers = map[string]bool{", handler.Name)
			g.indent++
			for _, pid := range handler.Permissions.AllowedPlayers {
				g.writeLine(`"%s": true,`, pid)
			}
			g.indent--
			g.writeLine("}")
			g.writeLine("")
		}
	}

	// Generate filter factories and registry if filters are defined
	if len(g.nodeGraph.Filters) > 0 {
		if err := g.generateFilters(); err != nil {
			return nil, fmt.Errorf("failed to generate filters: %w", err)
		}
	}

	// Generate reusable functions if defined
	if len(g.nodeGraph.Functions) > 0 {
		if err := g.generateFunctions(); err != nil {
			return nil, fmt.Errorf("failed to generate functions: %w", err)
		}
	}

	// Generate each event handler
	for i := range g.nodeGraph.Handlers {
		if err := g.generateHandler(&g.nodeGraph.Handlers[i]); err != nil {
			return nil, fmt.Errorf("failed to generate handler %s: %w", g.nodeGraph.Handlers[i].Name, err)
		}
		g.writeLine("")
	}

	// Save handler code and switch back to main buffer
	handlerCode := g.buf.Bytes()
	g.buf = handlerBuf
	g.buf.Reset()

	// Phase 2: Generate header with correct imports (now we know all extraImports)

	// Build tag for debug mode
	if g.debugMode {
		g.writeLine("//go:build debug")
		g.writeLine("// +build debug")
		g.writeLine("")
	}

	// Package declaration
	g.writeLine("package %s", g.nodeGraph.Package)
	g.writeLine("")

	// Imports
	g.writeLine("import (")
	g.indent++
	if g.contextMode {
		g.writeLine(`"context"`)
	}
	g.writeLine(`"errors"`)
	g.writeLine(`"fmt"`)
	g.writeLine(`"math"`)
	g.writeLine(`"math/rand"`)
	if len(g.nodeGraph.Filters) > 0 {
		g.writeLine(`"sync"`)
	}
	g.writeLine(`"time"`)
	g.writeLine("")
	g.writeLine(`"github.com/mxkacsa/statesync"`)
	if g.debugMode {
		g.writeLine(`"github.com/mxkacsa/statesync/debug"`)
	}
	for _, imp := range g.nodeGraph.Imports {
		g.writeLine(`"%s"`, imp)
	}
	// Add custom node imports (globally registered)
	for _, imp := range GetCustomImports() {
		g.writeLine(`"%s"`, imp)
	}
	// Add per-generator extra imports (added during generation)
	for imp := range g.extraImports {
		g.writeLine(`"%s"`, imp)
	}
	g.indent--
	g.writeLine(")")
	g.writeLine("")

	// Suppress unused import error
	g.writeLine("var _ = fmt.Sprintf")
	g.writeLine("var _ = math.Sqrt")
	g.writeLine("var _ = rand.Intn")
	g.writeLine("var _ = time.Now")
	g.writeLine("")

	// Permission error types
	g.writeLine("// Permission errors")
	g.writeLine("var (")
	g.indent++
	g.writeLine(`ErrNotHost    = errors.New("only host can perform this action")`)
	g.writeLine(`ErrNotAllowed = errors.New("player not allowed to perform this action")`)
	g.indent--
	g.writeLine(")")
	g.writeLine("")

	// Append handler code
	g.buf.Write(handlerCode)

	// Format the generated code
	formatted, err := format.Source(g.buf.Bytes())
	if err != nil {
		// Return unformatted code with error for debugging
		return g.buf.Bytes(), fmt.Errorf("failed to format code: %w", err)
	}

	return formatted, nil
}

// generateHandler generates code for a single event handler
func (g *CodeGenerator) generateHandler(handler *EventHandler) error {
	g.currentHandler = handler
	g.variables = make(map[string]string)
	g.nodeOutputs = make(map[string]map[string]string)
	g.loopStack = []loopContext{}
	g.generatedNodes = make(map[string]bool)
	g.hasWaitNodes = g.handlerHasWaitNodes(handler)

	// Determine root type name from schema
	rootTypeName := "GameState"
	if g.schema != nil && g.schema.RootType != nil {
		rootTypeName = g.schema.RootType.Name
	}

	// Function signature
	g.writeLine("// %s handles the %s event", handler.Name, handler.Event)

	// First parameter: context if handler has wait nodes
	if g.hasWaitNodes {
		g.write("func %s(ctx context.Context, session *statesync.TrackedSession[*%s, any, string]", handler.Name, rootTypeName)
	} else {
		g.write("func %s(session *statesync.TrackedSession[*%s, any, string]", handler.Name, rootTypeName)
	}

	// Always add senderID for permission checking
	g.write(", senderID string")
	g.variables["senderID"] = "string"

	// Add parameters
	for _, param := range handler.Parameters {
		g.write(", %s %s", param.Name, param.Type)
		g.variables[param.Name] = param.Type
	}

	// Add debug hook parameter in debug mode
	if g.debugMode {
		g.write(", dbg debug.DebugHook")
	}

	g.write(") error {\n")
	g.indent++

	// Debug: event start
	if g.debugMode {
		g.writeLine("_startTime := time.Now()")
		g.writeLine("if dbg != nil {")
		g.indent++
		// Build params map
		if len(handler.Parameters) > 0 {
			g.writeLine("_params := map[string]any{")
			g.indent++
			for _, param := range handler.Parameters {
				g.writeLine(`"%s": %s,`, param.Name, param.Name)
			}
			g.indent--
			g.writeLine("}")
			g.writeLine(`dbg.OnEventStart(session.ID(), "%s", _params)`, handler.Name)
		} else {
			g.writeLine(`dbg.OnEventStart(session.ID(), "%s", nil)`, handler.Name)
		}
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// Permission checks
	if handler.Permissions != nil {
		g.generatePermissionChecks(handler)
	}

	// Get state reference
	g.writeLine("state := session.State().Get()")
	g.writeLine("_ = state // may be unused")
	g.writeLine("_ = senderID // may be unused")
	if g.hasWaitNodes {
		g.writeLine("_ = ctx // may be unused")
	}
	g.writeLine("")

	// Build execution order from flow
	executionOrder, err := g.buildExecutionOrder(handler)
	if err != nil {
		return err
	}

	// Generate code for each node in order
	for _, nodeID := range executionOrder {
		// Skip already generated nodes (e.g., nodes inside control flow branches)
		if g.generatedNodes[nodeID] {
			continue
		}

		node := g.findNode(handler, nodeID)
		if node == nil {
			continue // Skip flow markers like "start"
		}

		if err := g.generateNode(node); err != nil {
			return fmt.Errorf("failed to generate node %s: %w", node.ID, err)
		}
	}

	// Debug: event end
	if g.debugMode {
		g.writeLine("if dbg != nil {")
		g.indent++
		g.writeLine("_duration := float64(time.Since(_startTime).Microseconds()) / 1000.0")
		g.writeLine(`dbg.OnEventEnd(session.ID(), "%s", _duration, nil)`, handler.Name)
		g.indent--
		g.writeLine("}")
	}

	// Return
	g.writeLine("return nil")
	g.indent--
	g.writeLine("}")

	return nil
}

// buildExecutionOrder creates a topologically sorted list of node IDs
func (g *CodeGenerator) buildExecutionOrder(handler *EventHandler) ([]string, error) {
	// Build adjacency list
	adj := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, edge := range handler.Flow {
		adj[edge.From] = append(adj[edge.From], edge.To)
		inDegree[edge.To]++
		if _, exists := inDegree[edge.From]; !exists {
			inDegree[edge.From] = 0
		}
	}

	// Find start nodes (in-degree 0)
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	// Topological sort
	result := []string{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, next := range adj[current] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	// Check for cycles
	if len(result) != len(inDegree) {
		return nil, fmt.Errorf("cycle detected in node graph")
	}

	return result, nil
}

// findNode finds a node by ID
func (g *CodeGenerator) findNode(handler *EventHandler, nodeID string) *Node {
	for i := range handler.Nodes {
		if handler.Nodes[i].ID == nodeID {
			return &handler.Nodes[i]
		}
	}
	return nil
}

// generateNode generates code for a single node
func (g *CodeGenerator) generateNode(node *Node) error {
	// Mark this node as generated to prevent duplicate generation
	g.generatedNodes[node.ID] = true

	// Debug: node start
	if g.debugMode {
		g.writeLine("// Node: %s (%s)", node.ID, node.Type)
		g.writeLine("if dbg != nil {")
		g.indent++
		g.writeLine(`dbg.OnNodeStart(session.ID(), "%s", "%s", "%s", nil)`, g.currentHandler.Name, node.ID, node.Type)
		g.indent--
		g.writeLine("}")
	} else {
		g.writeLine("// Node: %s (%s)", node.ID, node.Type)
	}

	// All nodes are registered via the registry system (see nodes_*.go files)
	if handled, err := g.tryCustomNodeGenerator(node); handled {
		return err
	}

	// Unknown node type
	g.writeLine("// TODO: Implement node type: %s", node.Type)
	return nil
}

// ============================================================================
// State Access Nodes
// ============================================================================

// generateGetPlayer generates code for GetPlayer node
// Uses schema information to determine the Player type name and key field
func (g *CodeGenerator) generateGetPlayer(node *Node) error {
	playerIDSrc, err := g.resolveInput(node, "playerID")
	if err != nil {
		return err
	}

	// Get player type info from schema
	playerTypeName := "Player" // Default fallback
	keyField := "ID"           // Default key field

	if g.schema != nil && g.schema.RootType != nil {
		if info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, "Players"); info != nil {
			// Get element type from the array field
			if info.ParsedType.ElemType != "" {
				playerTypeName = info.ParsedType.ElemType
			}
			if info.FieldDef.Key != "" {
				keyField = info.FieldDef.Key
			}
		}
	}

	outputVar := g.allocateVariable(node, "player", "*"+playerTypeName)

	g.writeLine("var %s *%s", outputVar, playerTypeName)
	g.writeLine("for i := range state.Players() {")
	g.indent++
	g.writeLine("p := state.PlayersAt(i)")
	g.writeLine("if p.%s == %s {", keyField, playerIDSrc)
	g.indent++
	g.writeLine("%s = &p", outputVar)
	g.writeLine("break")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("if %s == nil {", outputVar)
	g.indent++
	g.writeLine(`return fmt.Errorf("player not found: %%v", %s)`, playerIDSrc)
	g.indent--
	g.writeLine("}")

	return nil
}

// generateGetField generates code for GetField node (schema-aware)
func (g *CodeGenerator) generateGetField(node *Node) error {
	pathSrc, err := g.resolveInput(node, "path")
	if err != nil {
		return err
	}

	// Parse the path
	pathStr := strings.Trim(pathSrc, `"`)
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return fmt.Errorf("invalid path %s: %w", pathStr, err)
	}

	outputVar := g.allocateVariable(node, "value", "any")

	// Generate code to traverse the path
	code, resultExpr, err := g.generatePathAccess(parsedPath, false)
	if err != nil {
		return err
	}

	if code != "" {
		g.buf.WriteString(code)
	}
	g.writeLine("%s := %s", outputVar, resultExpr)

	return nil
}

// generateSetField generates code for SetField node (schema-aware)
func (g *CodeGenerator) generateSetField(node *Node) error {
	pathSrc, err := g.resolveInput(node, "path")
	if err != nil {
		return err
	}
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	// Parse the path
	pathStr := strings.Trim(pathSrc, `"`)
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return fmt.Errorf("invalid path %s: %w", pathStr, err)
	}

	// Generate the setter code
	return g.generatePathSet(parsedPath, valueSrc)
}

// generateGetCurrentState generates code for GetCurrentState node
func (g *CodeGenerator) generateGetCurrentState(node *Node) error {
	outputVar := g.allocateVariable(node, "state", "*GameState")
	g.writeLine("%s := state", outputVar)
	return nil
}

// generatePathAccess generates code to access a path and returns the expression
func (g *CodeGenerator) generatePathAccess(path *ParsedPath, forWrite bool) (string, string, error) {
	if len(path.Segments) == 0 {
		return "", "state", nil
	}

	var codeBuf bytes.Buffer
	currentExpr := "state"
	currentTypeName := ""
	if g.schema != nil && g.schema.RootType != nil {
		currentTypeName = g.schema.RootType.Name
	}

	for i, seg := range path.Segments {
		isLast := i == len(path.Segments)-1

		// Get field access info from schema
		var info *FieldAccessInfo
		if g.schema != nil && currentTypeName != "" {
			info = g.schema.GetFieldAccessInfo(currentTypeName, seg.FieldName)
		}

		switch seg.IndexType {
		case "": // Simple field access
			if info != nil {
				currentExpr = fmt.Sprintf("%s.%s()", currentExpr, info.GetterName)
				// Update current type for next iteration
				if !info.ParsedType.IsArray && !info.ParsedType.IsMap {
					currentTypeName = info.ParsedType.BaseType
				}
			} else {
				currentExpr = fmt.Sprintf("%s.%s()", currentExpr, seg.FieldName)
			}

		case "literal": // Array index with literal number
			idx := seg.IndexValue.(int)
			if info != nil {
				currentExpr = fmt.Sprintf("%s.%s(%d)", currentExpr, info.AtName, idx)
				currentTypeName = info.ParsedType.ElemType
			} else {
				currentExpr = fmt.Sprintf("%s.%sAt(%d)", currentExpr, seg.FieldName, idx)
			}

		case "variable": // Array index with variable
			varName := seg.IndexValue.(string)
			if info != nil {
				currentExpr = fmt.Sprintf("%s.%s(%s)", currentExpr, info.AtName, varName)
				currentTypeName = info.ParsedType.ElemType
			} else {
				currentExpr = fmt.Sprintf("%s.%sAt(%s)", currentExpr, seg.FieldName, varName)
			}

		case "key_lookup": // Find element by key field
			varName := seg.IndexValue.(string)
			keyField := seg.KeyField

			// Generate a lookup loop with proper not-found handling
			tmpIdx := fmt.Sprintf("_idx_%s_%d", seg.FieldName, i)
			tmpFound := fmt.Sprintf("_found_%s_%d", seg.FieldName, i)

			codeBuf.WriteString(g.indentStr())
			codeBuf.WriteString(fmt.Sprintf("%s := -1\n", tmpIdx))
			codeBuf.WriteString(g.indentStr())
			codeBuf.WriteString(fmt.Sprintf("for _i := 0; _i < %s.%sLen(); _i++ {\n", currentExpr, seg.FieldName))
			codeBuf.WriteString(g.indentStr())
			codeBuf.WriteString(fmt.Sprintf("\tif %s.%sAt(_i).%s == %s {\n", currentExpr, seg.FieldName, keyField, varName))
			codeBuf.WriteString(g.indentStr())
			codeBuf.WriteString(fmt.Sprintf("\t\t%s = _i\n", tmpIdx))
			codeBuf.WriteString(g.indentStr())
			codeBuf.WriteString("\t\tbreak\n")
			codeBuf.WriteString(g.indentStr())
			codeBuf.WriteString("\t}\n")
			codeBuf.WriteString(g.indentStr())
			codeBuf.WriteString("}\n")

			// Add not-found check to prevent accessing invalid index
			codeBuf.WriteString(g.indentStr())
			codeBuf.WriteString(fmt.Sprintf("if %s < 0 {\n", tmpIdx))
			codeBuf.WriteString(g.indentStr())
			codeBuf.WriteString(fmt.Sprintf("\treturn fmt.Errorf(\"%s not found in %s: %%v\", %s)\n", keyField, seg.FieldName, varName))
			codeBuf.WriteString(g.indentStr())
			codeBuf.WriteString("}\n")

			if !isLast || !forWrite {
				codeBuf.WriteString(g.indentStr())
				codeBuf.WriteString(fmt.Sprintf("%s := %s.%sAt(%s)\n", tmpFound, currentExpr, seg.FieldName, tmpIdx))
				currentExpr = tmpFound
			} else {
				// For write, we'll use the index
				currentExpr = fmt.Sprintf("%s.%sAt(%s)", currentExpr, seg.FieldName, tmpIdx)
			}

			if info != nil {
				currentTypeName = info.ParsedType.ElemType
			}
		}
	}

	return codeBuf.String(), currentExpr, nil
}

// generatePathSet generates code to set a value at a path
func (g *CodeGenerator) generatePathSet(path *ParsedPath, value string) error {
	if len(path.Segments) == 0 {
		return fmt.Errorf("cannot set root state")
	}

	// For single segment, simple setter
	if len(path.Segments) == 1 {
		seg := path.Segments[0]
		info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, seg.FieldName)

		switch seg.IndexType {
		case "": // Simple field
			if info != nil {
				g.writeLine("state.%s(%s)", info.SetterName, value)
			} else {
				g.writeLine("state.Set%s(%s)", seg.FieldName, value)
			}

		case "literal": // Array index
			idx := seg.IndexValue.(int)
			if info != nil {
				g.writeLine("state.%s(%d, %s)", info.UpdateAtName, idx, value)
			} else {
				g.writeLine("state.Update%sAt(%d, %s)", seg.FieldName, idx, value)
			}

		case "variable": // Variable index
			varName := seg.IndexValue.(string)
			if info != nil {
				g.writeLine("state.%s(%s, %s)", info.UpdateAtName, varName, value)
			} else {
				g.writeLine("state.Update%sAt(%s, %s)", seg.FieldName, varName, value)
			}

		case "key_lookup":
			varName := seg.IndexValue.(string)
			keyField := seg.KeyField
			g.writeLine("for _i := 0; _i < state.%sLen(); _i++ {", seg.FieldName)
			g.indent++
			g.writeLine("if state.%sAt(_i).%s == %s {", seg.FieldName, keyField, varName)
			g.indent++
			g.writeLine("state.Update%sAt(_i, %s)", seg.FieldName, value)
			g.writeLine("break")
			g.indent--
			g.writeLine("}")
			g.indent--
			g.writeLine("}")
		}
		return nil
	}

	// Multi-segment path: need to traverse and then set
	parentPath := &ParsedPath{
		Segments: path.Segments[:len(path.Segments)-1],
		RawPath:  path.RawPath,
	}
	lastSeg := path.Segments[len(path.Segments)-1]

	// Get the parent object
	code, parentExpr, err := g.generatePathAccess(parentPath, true)
	if err != nil {
		return err
	}

	if code != "" {
		g.buf.WriteString(code)
	}

	// Determine the parent type to get field info
	parentTypeName := g.getTypeAtPath(parentPath)

	var info *FieldAccessInfo
	if g.schema != nil && parentTypeName != "" {
		info = g.schema.GetFieldAccessInfo(parentTypeName, lastSeg.FieldName)
	}

	switch lastSeg.IndexType {
	case "": // Simple field set
		if info != nil {
			g.writeLine("%s.%s(%s)", parentExpr, info.SetterName, value)
		} else {
			g.writeLine("%s.Set%s(%s)", parentExpr, lastSeg.FieldName, value)
		}

	case "literal":
		idx := lastSeg.IndexValue.(int)
		if info != nil {
			g.writeLine("%s.%s(%d, %s)", parentExpr, info.UpdateAtName, idx, value)
		} else {
			g.writeLine("%s.Update%sAt(%d, %s)", parentExpr, lastSeg.FieldName, idx, value)
		}

	case "variable":
		varName := lastSeg.IndexValue.(string)
		if info != nil {
			g.writeLine("%s.%s(%s, %s)", parentExpr, info.UpdateAtName, varName, value)
		} else {
			g.writeLine("%s.Update%sAt(%s, %s)", parentExpr, lastSeg.FieldName, varName, value)
		}

	case "key_lookup":
		varName := lastSeg.IndexValue.(string)
		keyField := lastSeg.KeyField
		g.writeLine("for _i := 0; _i < %s.%sLen(); _i++ {", parentExpr, lastSeg.FieldName)
		g.indent++
		g.writeLine("if %s.%sAt(_i).%s == %s {", parentExpr, lastSeg.FieldName, keyField, varName)
		g.indent++
		g.writeLine("%s.Update%sAt(_i, %s)", parentExpr, lastSeg.FieldName, value)
		g.writeLine("break")
		g.indent--
		g.writeLine("}")
		g.indent--
		g.writeLine("}")
	}

	return nil
}

// getTypeAtPath returns the type name at a given path
func (g *CodeGenerator) getTypeAtPath(path *ParsedPath) string {
	if g.schema == nil || g.schema.RootType == nil {
		return ""
	}

	currentType := g.schema.RootType.Name

	for _, seg := range path.Segments {
		info := g.schema.GetFieldAccessInfo(currentType, seg.FieldName)
		if info == nil {
			return ""
		}

		if info.ParsedType.IsArray || info.ParsedType.IsMap {
			currentType = info.ParsedType.ElemType
		} else {
			currentType = info.ParsedType.BaseType
		}
	}

	return currentType
}

// ============================================================================
// Array Operations
// ============================================================================

func (g *CodeGenerator) generateAddToArray(node *Node) error {
	pathSrc, err := g.resolveInput(node, "path")
	if err != nil {
		// Fallback: array might be a reference
		arraySrc, err2 := g.resolveInput(node, "array")
		if err2 != nil {
			return err
		}
		elementSrc, err := g.resolveInput(node, "element")
		if err != nil {
			return err
		}
		outputVar := g.allocateVariable(node, "result", "[]any")
		g.writeLine("%s := append(%s, %s)", outputVar, arraySrc, elementSrc)
		return nil
	}

	elementSrc, err := g.resolveInput(node, "element")
	if err != nil {
		return err
	}

	// Parse path and use schema-aware append
	pathStr := strings.Trim(pathSrc, `"`)
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return err
	}

	if len(parsedPath.Segments) == 1 {
		seg := parsedPath.Segments[0]
		info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, seg.FieldName)
		if info != nil {
			g.writeLine("state.%s(%s)", info.AppendName, elementSrc)
		} else {
			g.writeLine("state.Append%s(%s)", seg.FieldName, elementSrc)
		}
	} else {
		// Need to get parent and append
		parentPath := &ParsedPath{Segments: parsedPath.Segments[:len(parsedPath.Segments)-1]}
		lastSeg := parsedPath.Segments[len(parsedPath.Segments)-1]

		code, parentExpr, err := g.generatePathAccess(parentPath, false)
		if err != nil {
			return err
		}
		if code != "" {
			g.buf.WriteString(code)
		}

		g.writeLine("%s.Append%s(%s)", parentExpr, lastSeg.FieldName, elementSrc)
	}

	return nil
}

// generateArrayAppend generates code for ArrayAppend node (appends to state array by path)
func (g *CodeGenerator) generateArrayAppend(node *Node) error {
	pathSrc, err := g.resolveInput(node, "path")
	if err != nil {
		return err
	}
	itemSrc, err := g.resolveInput(node, "item")
	if err != nil {
		return err
	}

	// Parse path and use schema-aware append
	pathStr := strings.Trim(pathSrc, `"`)
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return err
	}

	if len(parsedPath.Segments) == 1 {
		seg := parsedPath.Segments[0]
		info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, seg.FieldName)
		if info != nil {
			g.writeLine("state.%s(%s)", info.AppendName, itemSrc)
		} else {
			g.writeLine("state.Append%s(%s)", seg.FieldName, itemSrc)
		}
	} else {
		// Need to get parent and append
		parentPath := &ParsedPath{Segments: parsedPath.Segments[:len(parsedPath.Segments)-1]}
		lastSeg := parsedPath.Segments[len(parsedPath.Segments)-1]

		code, parentExpr, err := g.generatePathAccess(parentPath, false)
		if err != nil {
			return err
		}
		if code != "" {
			g.buf.WriteString(code)
		}

		g.writeLine("%s.Append%s(%s)", parentExpr, lastSeg.FieldName, itemSrc)
	}

	return nil
}

func (g *CodeGenerator) generateRemoveFromArray(node *Node) error {
	pathSrc, err := g.resolveInput(node, "path")
	if err != nil {
		return err
	}
	indexSrc, err := g.resolveInput(node, "index")
	if err != nil {
		return err
	}

	pathStr := strings.Trim(pathSrc, `"`)
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return err
	}

	if len(parsedPath.Segments) == 1 {
		seg := parsedPath.Segments[0]
		info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, seg.FieldName)
		if info != nil {
			g.writeLine("state.%s(%s)", info.RemoveAtName, indexSrc)
		} else {
			g.writeLine("state.Remove%sAt(%s)", seg.FieldName, indexSrc)
		}
	}

	return nil
}

func (g *CodeGenerator) generateFilterArray(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
		return err
	}

	fieldName, _ := g.resolveInputOptional(node, "field")
	op, _ := g.resolveInputOptional(node, "op")
	valueSrc, _ := g.resolveInputOptional(node, "value")

	outputVar := g.allocateVariable(node, "result", "[]any")
	itemType := g.inferArrayItemType(arraySrc)

	g.writeLine("var %s []%s", outputVar, itemType)
	g.writeLine("for _, _item := range %s {", arraySrc)
	g.indent++

	if fieldName != "" && op != "" && valueSrc != "" {
		field := strings.Trim(fieldName, `"`)
		operator := strings.Trim(op, `"`)
		g.writeLine("if _item.%s %s %s {", field, operator, valueSrc)
	} else {
		g.writeLine("if true { // TODO: add filter condition")
	}

	g.indent++
	g.writeLine("%s = append(%s, _item)", outputVar, outputVar)
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")

	return nil
}

func (g *CodeGenerator) generateFindInArray(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
		return err
	}

	fieldName, _ := g.resolveInputOptional(node, "field")
	valueSrc, _ := g.resolveInputOptional(node, "value")

	outputVar := g.allocateVariable(node, "item", "any")
	indexVar := g.allocateVariable(node, "index", "int")

	g.writeLine("%s := -1", indexVar)
	g.writeLine("var %s any", outputVar)
	g.writeLine("for _i, _item := range %s {", arraySrc)
	g.indent++

	if fieldName != "" && valueSrc != "" {
		field := strings.Trim(fieldName, `"`)
		g.writeLine("if _item.%s == %s {", field, valueSrc)
	} else {
		g.writeLine("if _item == %s {", valueSrc)
	}

	g.indent++
	g.writeLine("%s = _i", indexVar)
	g.writeLine("%s = _item", outputVar)
	g.writeLine("break")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")

	return nil
}

func (g *CodeGenerator) generateArrayLength(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "length", "int")
	g.writeLine("%s := len(%s)", outputVar, arraySrc)
	return nil
}

func (g *CodeGenerator) generateArrayAt(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
		return err
	}
	indexSrc, err := g.resolveInput(node, "index")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "item", "any")
	g.writeLine("%s := %s[%s]", outputVar, arraySrc, indexSrc)
	return nil
}

func (g *CodeGenerator) generateMapArray(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "result", "[]any")
	g.writeLine("%s := make([]any, len(%s))", outputVar, arraySrc)
	g.writeLine("for _i, _item := range %s {", arraySrc)
	g.indent++
	g.writeLine("_ = _item")
	g.writeLine("%s[_i] = _item // TODO: transform", outputVar)
	g.indent--
	g.writeLine("}")
	return nil
}

// ============================================================================
// Map Operations
// ============================================================================

func (g *CodeGenerator) generateSetMapValue(node *Node) error {
	pathSrc, _ := g.resolveInputOptional(node, "path")
	keySrc, err := g.resolveInput(node, "key")
	if err != nil {
		return err
	}
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	if pathSrc != "" {
		pathStr := strings.Trim(pathSrc, `"`)
		parsedPath, _ := ParsePath(pathStr)

		if len(parsedPath.Segments) == 1 {
			seg := parsedPath.Segments[0]
			info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, seg.FieldName)
			if info != nil {
				g.writeLine("state.%s(%s, %s)", info.SetKeyName, keySrc, valueSrc)
				return nil
			}
		}
	}

	g.writeLine("// TODO: SetMapValue for path %s", pathSrc)
	return nil
}

func (g *CodeGenerator) generateGetMapValue(node *Node) error {
	mapSrc, err := g.resolveInput(node, "map")
	if err != nil {
		return err
	}
	keySrc, err := g.resolveInput(node, "key")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "value", "any")
	existsVar := g.allocateVariable(node, "exists", "bool")
	g.writeLine("%s, %s := %s[%s]", outputVar, existsVar, mapSrc, keySrc)
	g.writeLine("_ = %s", existsVar)
	return nil
}

func (g *CodeGenerator) generateRemoveMapKey(node *Node) error {
	pathSrc, _ := g.resolveInputOptional(node, "path")
	keySrc, err := g.resolveInput(node, "key")
	if err != nil {
		return err
	}

	if pathSrc != "" {
		pathStr := strings.Trim(pathSrc, `"`)
		parsedPath, _ := ParsePath(pathStr)

		if len(parsedPath.Segments) == 1 {
			seg := parsedPath.Segments[0]
			info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, seg.FieldName)
			if info != nil {
				g.writeLine("state.%s(%s)", info.DeleteKeyName, keySrc)
				return nil
			}
		}
	}

	g.writeLine("// TODO: RemoveMapKey for path %s", pathSrc)
	return nil
}

func (g *CodeGenerator) generateHasMapKey(node *Node) error {
	mapSrc, err := g.resolveInput(node, "map")
	if err != nil {
		return err
	}
	keySrc, err := g.resolveInput(node, "key")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "exists", "bool")
	g.writeLine("_, %s := %s[%s]", outputVar, mapSrc, keySrc)
	return nil
}

// ============================================================================
// Control Flow
// ============================================================================

func (g *CodeGenerator) generateIf(node *Node) error {
	conditionSrc, err := g.resolveInput(node, "condition")
	if err != nil {
		return err
	}

	g.writeLine("if %s {", conditionSrc)
	g.indent++

	// Find and generate true branch - follow the entire chain
	trueBranch := g.findFlowTargets(node.ID, "true")
	if len(trueBranch) > 0 {
		if err := g.generateBodyChain(trueBranch[0]); err != nil {
			return err
		}
	}

	g.indent--

	// Check for false branch
	falseBranch := g.findFlowTargets(node.ID, "false")
	if len(falseBranch) > 0 {
		g.writeLine("} else {")
		g.indent++
		if err := g.generateBodyChain(falseBranch[0]); err != nil {
			return err
		}
		g.indent--
	}

	g.writeLine("}")
	return nil
}

func (g *CodeGenerator) generateForEach(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
		return err
	}

	itemVar := g.allocateVariable(node, "item", "any")
	indexVar := g.allocateVariable(node, "index", "int")

	// Push loop context
	g.loopStack = append(g.loopStack, loopContext{
		nodeID:    node.ID,
		itemVar:   itemVar,
		indexVar:  indexVar,
		arrayExpr: arraySrc,
	})

	g.writeLine("for %s, %s := range %s {", indexVar, itemVar, arraySrc)
	g.indent++

	// Find and generate body nodes
	bodyNodes := g.findFlowTargets(node.ID, "body")
	for _, targetID := range bodyNodes {
		if g.generatedNodes[targetID] {
			continue
		}
		targetNode := g.findNode(g.currentHandler, targetID)
		if targetNode != nil {
			if err := g.generateNode(targetNode); err != nil {
				return err
			}
		}
	}

	g.indent--
	g.writeLine("}")

	// Pop loop context
	g.loopStack = g.loopStack[:len(g.loopStack)-1]

	return nil
}

func (g *CodeGenerator) generateWhile(node *Node) error {
	conditionSrc, err := g.resolveInput(node, "condition")
	if err != nil {
		return err
	}

	g.writeLine("for %s {", conditionSrc)
	g.indent++

	bodyNodes := g.findFlowTargets(node.ID, "body")
	for _, targetID := range bodyNodes {
		if g.generatedNodes[targetID] {
			continue
		}
		targetNode := g.findNode(g.currentHandler, targetID)
		if targetNode != nil {
			if err := g.generateNode(targetNode); err != nil {
				return err
			}
		}
	}

	g.indent--
	g.writeLine("}")
	return nil
}

// findFlowTargets finds nodes connected via a specific flow label
func (g *CodeGenerator) findFlowTargets(fromID, label string) []string {
	var targets []string
	for _, edge := range g.currentHandler.Flow {
		if edge.From == fromID && edge.Label == label {
			targets = append(targets, edge.To)
		}
	}
	return targets
}

// getFlowEdgesFrom returns all flow edges originating from a node
func (g *CodeGenerator) getFlowEdgesFrom(nodeID string) []FlowEdge {
	var edges []FlowEdge
	for _, edge := range g.currentHandler.Flow {
		if edge.From == nodeID {
			edges = append(edges, edge)
		}
	}
	return edges
}

// ============================================================================
// Batch Operations
// ============================================================================

// generateForEachWhere generates: for each item in array where condition { body }
// Example: game.players.where(team == currentTeam).forEach(...)
func (g *CodeGenerator) generateForEachWhere(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
		return err
	}
	fieldName, err := g.resolveInput(node, "field")
	if err != nil {
		return err
	}
	op, err := g.resolveInput(node, "op")
	if err != nil {
		return err
	}
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	field := strings.Trim(fieldName, `"`)
	operator := strings.Trim(op, `"`)

	itemVar := g.allocateVariable(node, "item", "any")
	indexVar := g.allocateVariable(node, "index", "int")

	// Determine if this is a state array (for proper update syntax)
	isStateArray := strings.HasPrefix(arraySrc, "state.")

	// Push loop context
	g.loopStack = append(g.loopStack, loopContext{
		nodeID:    node.ID,
		itemVar:   itemVar,
		indexVar:  indexVar,
		arrayExpr: arraySrc,
	})

	// Use index-based iteration for state arrays to allow updates
	if isStateArray {
		// Extract field name from state.FieldName() pattern
		arrayField := strings.TrimSuffix(strings.TrimPrefix(arraySrc, "state."), "()")
		g.writeLine("for %s := 0; %s < state.%sLen(); %s++ {", indexVar, indexVar, arrayField, indexVar)
		g.indent++
		g.writeLine("%s := state.%sAt(%s)", itemVar, arrayField, indexVar)
	} else {
		g.writeLine("for %s, %s := range %s {", indexVar, itemVar, arraySrc)
		g.indent++
	}

	g.writeLine("if %s.%s %s %s {", itemVar, field, operator, valueSrc)
	g.indent++

	// Find first body node and generate the entire body flow chain
	bodyNodes := g.findFlowTargets(node.ID, "body")
	if len(bodyNodes) > 0 {
		if err := g.generateBodyChain(bodyNodes[0]); err != nil {
			return err
		}
	}

	// For state arrays, update the item at the end of the body
	if isStateArray {
		arrayField := strings.TrimSuffix(strings.TrimPrefix(arraySrc, "state."), "()")
		g.writeLine("state.Update%sAt(%s, %s)", arrayField, indexVar, itemVar)
	}

	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")

	// Pop loop context
	g.loopStack = g.loopStack[:len(g.loopStack)-1]

	return nil
}

// generateBodyChain generates nodes following the flow chain until it hits an endpoint
func (g *CodeGenerator) generateBodyChain(startNodeID string) error {
	currentID := startNodeID
	for currentID != "" && currentID != "end" && !strings.HasPrefix(currentID, "end_") {
		if g.generatedNodes[currentID] {
			break
		}
		targetNode := g.findNode(g.currentHandler, currentID)
		if targetNode == nil {
			break
		}
		if err := g.generateNode(targetNode); err != nil {
			return err
		}

		// Find next node in the chain (non-labeled edges or "body" labeled)
		nextID := ""
		for _, edge := range g.currentHandler.Flow {
			if edge.From == currentID {
				// Skip labeled edges that are control flow (true/false/done) - these are handled by If/ForEach
				if edge.Label == "true" || edge.Label == "false" || edge.Label == "done" {
					continue
				}
				// Follow body edges or unlabeled edges
				if edge.Label == "body" || edge.Label == "" {
					nextID = edge.To
					break
				}
			}
		}
		currentID = nextID
	}
	return nil
}

// generateUpdateWhere generates: update all items in array where condition, set field = value
// Example: game.players.where(team == currentTeam).set(targetX, targetY)
func (g *CodeGenerator) generateUpdateWhere(node *Node) error {
	pathSrc, err := g.resolveInput(node, "path")
	if err != nil {
		return err
	}
	whereField, err := g.resolveInput(node, "whereField")
	if err != nil {
		return err
	}
	whereOp, err := g.resolveInput(node, "whereOp")
	if err != nil {
		return err
	}
	whereValue, err := g.resolveInput(node, "whereValue")
	if err != nil {
		return err
	}

	// Check for "updates" map (multiple field updates) or single setField/setValue
	updatesMap, hasUpdates := node.Inputs["updates"]
	var singleFieldName, singleFieldValue string
	if !hasUpdates {
		setField, err := g.resolveInput(node, "setField")
		if err != nil {
			return err
		}
		setValue, err := g.resolveInput(node, "setValue")
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

	countVar := g.allocateVariable(node, "count", "int")
	g.writeLine("%s := 0", countVar)

	// Helper to generate the field updates inside the if block
	generateUpdates := func() error {
		if hasUpdates {
			if um, ok := updatesMap.(map[string]interface{}); ok {
				for fieldName, fieldValue := range um {
					valueSrc, err := g.resolveInputValue(fieldValue)
					if err != nil {
						g.writeLine("_item.%s = %v", fieldName, fieldValue)
					} else {
						g.writeLine("_item.%s = %s", fieldName, valueSrc)
					}
				}
			}
		} else {
			g.writeLine("_item.%s = %s", singleFieldName, singleFieldValue)
		}
		return nil
	}

	// Get the array expression
	if len(parsedPath.Segments) == 1 {
		seg := parsedPath.Segments[0]
		info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, seg.FieldName)

		if info != nil {
			g.writeLine("for _i := 0; _i < state.%s(); _i++ {", info.LenName)
			g.indent++
			g.writeLine("_item := state.%s(_i)", info.AtName)
			g.writeLine("if _item.%s %s %s {", wField, wOp, whereValue)
			g.indent++
			generateUpdates()
			g.writeLine("state.%s(_i, _item)", info.UpdateAtName)
			g.writeLine("%s++", countVar)
			g.indent--
			g.writeLine("}")
			g.indent--
			g.writeLine("}")
		} else {
			g.writeLine("for _i := 0; _i < state.%sLen(); _i++ {", seg.FieldName)
			g.indent++
			g.writeLine("_item := state.%sAt(_i)", seg.FieldName)
			g.writeLine("if _item.%s %s %s {", wField, wOp, whereValue)
			g.indent++
			generateUpdates()
			g.writeLine("state.Update%sAt(_i, _item)", seg.FieldName)
			g.writeLine("%s++", countVar)
			g.indent--
			g.writeLine("}")
			g.indent--
			g.writeLine("}")
		}
	} else {
		// Nested path - get parent first
		parentPath := &ParsedPath{Segments: parsedPath.Segments[:len(parsedPath.Segments)-1]}
		lastSeg := parsedPath.Segments[len(parsedPath.Segments)-1]

		code, parentExpr, err := g.generatePathAccess(parentPath, false)
		if err != nil {
			return err
		}
		if code != "" {
			g.buf.WriteString(code)
		}

		g.writeLine("for _i := 0; _i < %s.%sLen(); _i++ {", parentExpr, lastSeg.FieldName)
		g.indent++
		g.writeLine("_item := %s.%sAt(_i)", parentExpr, lastSeg.FieldName)
		g.writeLine("if _item.%s %s %s {", wField, wOp, whereValue)
		g.indent++
		generateUpdates()
		g.writeLine("%s.Update%sAt(_i, _item)", parentExpr, lastSeg.FieldName)
		g.writeLine("%s++", countVar)
		g.indent--
		g.writeLine("}")
		g.indent--
		g.writeLine("}")
	}

	return nil
}

// generateFindWhere generates: find first item where condition
func (g *CodeGenerator) generateFindWhere(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
		return err
	}
	fieldName, err := g.resolveInput(node, "field")
	if err != nil {
		return err
	}
	op, err := g.resolveInput(node, "op")
	if err != nil {
		return err
	}
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	field := strings.Trim(fieldName, `"`)
	operator := strings.Trim(op, `"`)

	itemVar := g.allocateVariable(node, "item", "any")
	indexVar := g.allocateVariable(node, "index", "int")
	foundVar := g.allocateVariable(node, "found", "bool")

	g.writeLine("%s := -1", indexVar)
	g.writeLine("var %s any", itemVar)
	g.writeLine("%s := false", foundVar)
	g.writeLine("for _i, _item := range %s {", arraySrc)
	g.indent++
	g.writeLine("if _item.%s %s %s {", field, operator, valueSrc)
	g.indent++
	g.writeLine("%s = _i", indexVar)
	g.writeLine("%s = _item", itemVar)
	g.writeLine("%s = true", foundVar)
	g.writeLine("break")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")

	return nil
}

// generateCountWhere generates: count items where condition
func (g *CodeGenerator) generateCountWhere(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
		return err
	}
	fieldName, err := g.resolveInput(node, "field")
	if err != nil {
		return err
	}
	op, err := g.resolveInput(node, "op")
	if err != nil {
		return err
	}
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	field := strings.Trim(fieldName, `"`)
	operator := strings.Trim(op, `"`)

	countVar := g.allocateVariable(node, "count", "int")

	g.writeLine("%s := 0", countVar)
	g.writeLine("for _, _item := range %s {", arraySrc)
	g.indent++
	g.writeLine("if _item.%s %s %s {", field, operator, valueSrc)
	g.indent++
	g.writeLine("%s++", countVar)
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")

	return nil
}

// ============================================================================
// Logic Operations
// ============================================================================

func (g *CodeGenerator) generateCompare(node *Node) error {
	leftSrc, err := g.resolveInput(node, "left")
	if err != nil {
		return err
	}
	rightSrc, err := g.resolveInput(node, "right")
	if err != nil {
		return err
	}
	opSrc, err := g.resolveInput(node, "op")
	if err != nil {
		return err
	}

	op := strings.Trim(opSrc, `"`)
	outputVar := g.allocateVariable(node, "result", "bool")

	g.writeLine("%s := %s %s %s", outputVar, leftSrc, op, rightSrc)

	return nil
}

func (g *CodeGenerator) generateLogicOp(node *Node, op string) error {
	aSrc, err := g.resolveInput(node, "a")
	if err != nil {
		return err
	}
	bSrc, err := g.resolveInput(node, "b")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "result", "bool")
	g.writeLine("%s := %s %s %s", outputVar, aSrc, op, bSrc)
	return nil
}

func (g *CodeGenerator) generateNot(node *Node) error {
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "result", "bool")
	g.writeLine("%s := !%s", outputVar, valueSrc)
	return nil
}

func (g *CodeGenerator) generateIsNull(node *Node) error {
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "result", "bool")
	g.writeLine("%s := %s == nil", outputVar, valueSrc)
	return nil
}

func (g *CodeGenerator) generateIsEmpty(node *Node) error {
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "result", "bool")
	g.writeLine("%s := len(%s) == 0", outputVar, valueSrc)
	return nil
}

// ============================================================================
// Math Operations
// ============================================================================

func (g *CodeGenerator) generateMathOp(node *Node, op string) error {
	aSrc, err := g.resolveInput(node, "a")
	if err != nil {
		return err
	}
	bSrc, err := g.resolveInput(node, "b")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "result", "number")
	g.writeLine("%s := %s %s %s", outputVar, aSrc, op, bSrc)

	return nil
}

func (g *CodeGenerator) generateMinMax(node *Node, fn string) error {
	aSrc, err := g.resolveInput(node, "a")
	if err != nil {
		return err
	}
	bSrc, err := g.resolveInput(node, "b")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "result", "number")
	g.writeLine("%s := %s(%s, %s)", outputVar, fn, aSrc, bSrc)
	return nil
}

// generateMathFunc generates a single-argument math function call (Sqrt, Abs, Sin, Cos)
func (g *CodeGenerator) generateMathFunc(node *Node, fn string) error {
	var valueSrc string
	var err error

	// Try different input names
	if valueSrc, err = g.resolveInput(node, "value"); err != nil {
		if valueSrc, err = g.resolveInput(node, "angle"); err != nil {
			return err
		}
	}

	outputVar := g.allocateVariable(node, "result", "float64")
	g.writeLine("%s := %s(%s)", outputVar, fn, valueSrc)
	return nil
}

// generateMathFunc2 generates a two-argument math function call (Pow, Atan2)
func (g *CodeGenerator) generateMathFunc2(node *Node, fn string) error {
	var aSrc, bSrc string
	var err error

	// Try different input name combinations
	if fn == "math.Pow" {
		aSrc, err = g.resolveInput(node, "base")
		if err != nil {
			return err
		}
		bSrc, err = g.resolveInput(node, "exp")
		if err != nil {
			return err
		}
	} else if fn == "math.Atan2" {
		aSrc, err = g.resolveInput(node, "y")
		if err != nil {
			return err
		}
		bSrc, err = g.resolveInput(node, "x")
		if err != nil {
			return err
		}
	} else {
		aSrc, err = g.resolveInput(node, "a")
		if err != nil {
			return err
		}
		bSrc, err = g.resolveInput(node, "b")
		if err != nil {
			return err
		}
	}

	outputVar := g.allocateVariable(node, "result", "float64")
	g.writeLine("%s := %s(%s, %s)", outputVar, fn, aSrc, bSrc)
	return nil
}

// ============================================================================
// GPS/Geometry Operations
// ============================================================================

// GPS/Geometry operations are registered via init() in nodes_gps.go
// This demonstrates the extensibility system - external packages can
// register custom nodes the same way.

// ============================================================================
// Random Operations
// ============================================================================

// generateRandomInt generates a random integer in range [min, max]
func (g *CodeGenerator) generateRandomInt(node *Node) error {
	minVal, err := g.resolveInput(node, "min")
	if err != nil {
		return err
	}
	maxVal, err := g.resolveInput(node, "max")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "result", "int")
	g.writeLine("%s := rand.Intn(%s-%s+1) + %s", outputVar, maxVal, minVal, minVal)

	return nil
}

// generateRandomFloat generates a random float in range [min, max)
func (g *CodeGenerator) generateRandomFloat(node *Node) error {
	minVal, err := g.resolveInput(node, "min")
	if err != nil {
		return err
	}
	maxVal, err := g.resolveInput(node, "max")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "result", "float64")
	g.writeLine("%s := rand.Float64()*(%s-%s) + %s", outputVar, maxVal, minVal, minVal)

	return nil
}

// generateRandomBool generates a random boolean with given probability (0.0 to 1.0)
func (g *CodeGenerator) generateRandomBool(node *Node) error {
	probability, err := g.resolveInput(node, "probability")
	if err != nil {
		// Default to 0.5 if not specified
		probability = "0.5"
	}

	outputVar := g.allocateVariable(node, "result", "bool")
	g.writeLine("%s := rand.Float64() < %s", outputVar, probability)

	return nil
}

// ============================================================================
// Time Operations
// ============================================================================

// generateGetCurrentTime gets the current Unix timestamp in seconds
func (g *CodeGenerator) generateGetCurrentTime(node *Node) error {
	outputVar := g.allocateVariable(node, "timestamp", "int64")
	g.writeLine("%s := time.Now().Unix()", outputVar)
	return nil
}

// generateTimeSince calculates the time elapsed since a given timestamp (in seconds)
func (g *CodeGenerator) generateTimeSince(node *Node) error {
	startTime, err := g.resolveInput(node, "startTime")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "elapsed", "int64")
	g.writeLine("%s := time.Now().Unix() - %s", outputVar, startTime)

	return nil
}

// ============================================================================
// Object Creation Operations
// ============================================================================

// generateCreateStruct creates a new struct instance
func (g *CodeGenerator) generateCreateStruct(node *Node) error {
	typeSrc, err := g.resolveInput(node, "type")
	if err != nil {
		return err
	}

	typeName := strings.Trim(typeSrc, `"`)
	outputVar := g.allocateVariable(node, "result", typeName)

	// Get the fields input
	fieldsInput, hasFields := node.Inputs["fields"]

	g.writeLine("%s := %s{", outputVar, typeName)
	g.indent++

	if hasFields {
		if fieldsMap, ok := fieldsInput.(map[string]interface{}); ok {
			for fieldName, fieldValue := range fieldsMap {
				valueSrc, err := g.resolveInputValue(fieldValue)
				if err != nil {
					g.writeLine("%s: %v,", fieldName, fieldValue)
				} else {
					g.writeLine("%s: %s,", fieldName, valueSrc)
				}
			}
		}
	}

	g.indent--
	g.writeLine("}")

	return nil
}

// generateUpdateStruct updates fields on an existing struct
func (g *CodeGenerator) generateUpdateStruct(node *Node) error {
	objectSrc, err := g.resolveInput(node, "object")
	if err != nil {
		return err
	}

	// Modify the object in-place (no new variable needed)
	// Also store a reference for other nodes that might need it
	g.allocateVariableWithValue(node, "result", "any", objectSrc)

	// Get the fields input and update the object directly
	fieldsInput, hasFields := node.Inputs["fields"]
	if hasFields {
		if fieldsMap, ok := fieldsInput.(map[string]interface{}); ok {
			for fieldName, fieldValue := range fieldsMap {
				valueSrc, err := g.resolveInputValue(fieldValue)
				if err != nil {
					g.writeLine("%s.%s = %v", objectSrc, fieldName, fieldValue)
				} else {
					g.writeLine("%s.%s = %s", objectSrc, fieldName, valueSrc)
				}
			}
		}
	}

	return nil
}

// generateGetStructField gets a field value from a struct
func (g *CodeGenerator) generateGetStructField(node *Node) error {
	objectSrc, err := g.resolveInput(node, "object")
	if err != nil {
		return err
	}
	fieldSrc, err := g.resolveInput(node, "field")
	if err != nil {
		return err
	}

	fieldName := strings.Trim(fieldSrc, `"`)
	outputVar := g.allocateVariable(node, "value", "any")

	g.writeLine("%s := %s.%s", outputVar, objectSrc, fieldName)

	return nil
}

// resolveInputValue resolves an input value directly (for struct fields)
func (g *CodeGenerator) resolveInputValue(v interface{}) (string, error) {
	source, err := ParseValueSource(v)
	if err != nil {
		return "", err
	}

	switch source.Type {
	case "constant":
		return g.formatConstant(source.Value), nil
	case "ref":
		ref := source.Value.(string)
		parts := strings.SplitN(ref, ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid reference: %s", ref)
		}

		switch parts[0] {
		case "param":
			return parts[1], nil
		case "node":
			nodeParts := strings.SplitN(parts[1], ":", 2)
			if len(nodeParts) != 2 {
				return "", fmt.Errorf("invalid node reference: %s", ref)
			}
			nodeID := nodeParts[0]
			outputName := nodeParts[1]

			if outputs, ok := g.nodeOutputs[nodeID]; ok {
				if varName, ok := outputs[outputName]; ok {
					return varName, nil
				}
			}
			return "", fmt.Errorf("node output not found: %s", ref)
		case "variable":
			return parts[1], nil
		default:
			return "", fmt.Errorf("unknown reference type: %s", parts[0])
		}
	default:
		return "", fmt.Errorf("unknown source type: %s", source.Type)
	}
}

// ============================================================================
// String Operations
// ============================================================================

func (g *CodeGenerator) generateConcat(node *Node) error {
	stringsSrc, err := g.resolveInput(node, "strings")
	if err != nil {
		// Try a, b format
		aSrc, err := g.resolveInput(node, "a")
		if err != nil {
			return err
		}
		bSrc, err := g.resolveInput(node, "b")
		if err != nil {
			return err
		}
		outputVar := g.allocateVariable(node, "result", "string")
		g.writeLine("%s := %s + %s", outputVar, aSrc, bSrc)
		return nil
	}

	outputVar := g.allocateVariable(node, "result", "string")
	g.writeLine("%s := strings.Join(%s, \"\")", outputVar, stringsSrc)
	return nil
}

func (g *CodeGenerator) generateFormat(node *Node) error {
	formatSrc, err := g.resolveInput(node, "format")
	if err != nil {
		return err
	}
	argsSrc, _ := g.resolveInputOptional(node, "args")

	outputVar := g.allocateVariable(node, "result", "string")
	if argsSrc != "" {
		g.writeLine("%s := fmt.Sprintf(%s, %s...)", outputVar, formatSrc, argsSrc)
	} else {
		g.writeLine("%s := fmt.Sprintf(%s)", outputVar, formatSrc)
	}
	return nil
}

// ============================================================================
// Variable Operations
// ============================================================================

func (g *CodeGenerator) generateSetVariable(node *Node) error {
	nameSrc, err := g.resolveInput(node, "name")
	if err != nil {
		return err
	}
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	varName := strings.Trim(nameSrc, `"`)
	g.writeLine("%s := %s", varName, valueSrc)
	g.variables[varName] = "any"
	return nil
}

func (g *CodeGenerator) generateGetVariable(node *Node) error {
	nameSrc, err := g.resolveInput(node, "name")
	if err != nil {
		return err
	}

	varName := strings.Trim(nameSrc, `"`)
	outputVar := g.allocateVariable(node, "value", "any")
	g.writeLine("%s := %s", outputVar, varName)
	return nil
}

func (g *CodeGenerator) generateConstant(node *Node) error {
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "value", "any")
	g.writeLine("%s := %s", outputVar, valueSrc)
	return nil
}

// ============================================================================
// Event Operations
// ============================================================================

func (g *CodeGenerator) generateEmitToAll(node *Node) error {
	eventTypeSrc, err := g.resolveInput(node, "eventType")
	if err != nil {
		return err
	}

	if _, hasPayload := node.Inputs["payload"]; hasPayload {
		payloadSrc, _ := g.resolveInput(node, "payload")
		g.writeLine("session.Emit(%s, %s)", eventTypeSrc, payloadSrc)
	} else {
		g.writeLine("session.Emit(%s, nil)", eventTypeSrc)
	}

	return nil
}

func (g *CodeGenerator) generateEmitToPlayer(node *Node) error {
	playerIDSrc, err := g.resolveInput(node, "playerID")
	if err != nil {
		return err
	}
	eventTypeSrc, err := g.resolveInput(node, "eventType")
	if err != nil {
		return err
	}

	if _, hasPayload := node.Inputs["payload"]; hasPayload {
		payloadSrc, _ := g.resolveInput(node, "payload")
		g.writeLine("session.EmitTo(%s, %s, %s)", playerIDSrc, eventTypeSrc, payloadSrc)
	} else {
		g.writeLine("session.EmitTo(%s, %s, nil)", playerIDSrc, eventTypeSrc)
	}

	return nil
}

func (g *CodeGenerator) generateEmitToMany(node *Node) error {
	playerIDsSrc, err := g.resolveInput(node, "playerIDs")
	if err != nil {
		return err
	}
	eventTypeSrc, err := g.resolveInput(node, "eventType")
	if err != nil {
		return err
	}

	if _, hasPayload := node.Inputs["payload"]; hasPayload {
		payloadSrc, _ := g.resolveInput(node, "payload")
		g.writeLine("session.EmitToMany(%s, %s, %s)", playerIDsSrc, eventTypeSrc, payloadSrc)
	} else {
		g.writeLine("session.EmitToMany(%s, %s, nil)", playerIDsSrc, eventTypeSrc)
	}

	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// resolveInput resolves an input value to Go code
func (g *CodeGenerator) resolveInput(node *Node, inputName string) (string, error) {
	input, ok := node.Inputs[inputName]
	if !ok {
		return "", fmt.Errorf("missing required input: %s", inputName)
	}

	source, err := ParseValueSource(input)
	if err != nil {
		return "", err
	}

	switch source.Type {
	case "constant":
		return g.formatConstant(source.Value), nil
	case "ref":
		ref := source.Value.(string)
		parts := strings.SplitN(ref, ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid reference: %s", ref)
		}

		switch parts[0] {
		case "param":
			return parts[1], nil
		case "node":
			nodeParts := strings.SplitN(parts[1], ":", 2)
			if len(nodeParts) != 2 {
				return "", fmt.Errorf("invalid node reference: %s", ref)
			}
			nodeID := nodeParts[0]
			outputName := nodeParts[1]

			if outputs, ok := g.nodeOutputs[nodeID]; ok {
				if varName, ok := outputs[outputName]; ok {
					return varName, nil
				}
			}
			return "", fmt.Errorf("node output not found: %s", ref)
		case "variable":
			return parts[1], nil
		case "state":
			// Direct state field access
			fieldName := parts[1]
			if g.schema != nil && g.schema.RootType != nil {
				info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, fieldName)
				if info != nil {
					return fmt.Sprintf("state.%s()", info.GetterName), nil
				}
			}
			return fmt.Sprintf("state.%s()", fieldName), nil
		case "loop":
			// Reference to current loop item/index
			if len(g.loopStack) > 0 {
				current := g.loopStack[len(g.loopStack)-1]
				switch parts[1] {
				case "item":
					return current.itemVar, nil
				case "index":
					return current.indexVar, nil
				}
			}
			return "", fmt.Errorf("no active loop for reference: %s", ref)
		default:
			return "", fmt.Errorf("unknown reference type: %s", parts[0])
		}
	default:
		return "", fmt.Errorf("unknown source type: %s", source.Type)
	}
}

// resolveInputOptional resolves an optional input
func (g *CodeGenerator) resolveInputOptional(node *Node, inputName string) (string, error) {
	if _, ok := node.Inputs[inputName]; !ok {
		return "", nil
	}
	return g.resolveInput(node, inputName)
}

// resolveReference resolves a reference string (like "node:xyz:result" or "param:foo")
// and returns the Go expression that evaluates to the value.
func (g *CodeGenerator) resolveReference(ref string) (string, error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid reference: %s", ref)
	}

	switch parts[0] {
	case "param":
		return parts[1], nil
	case "node":
		nodeParts := strings.SplitN(parts[1], ":", 2)
		if len(nodeParts) != 2 {
			return "", fmt.Errorf("invalid node reference: %s", ref)
		}
		nodeID := nodeParts[0]
		outputName := nodeParts[1]

		if outputs, ok := g.nodeOutputs[nodeID]; ok {
			if varName, ok := outputs[outputName]; ok {
				return varName, nil
			}
		}
		return "", fmt.Errorf("node output not found: %s", ref)
	case "variable":
		return parts[1], nil
	case "state":
		// Direct state field access
		fieldName := parts[1]
		if g.schema != nil && g.schema.RootType != nil {
			info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, fieldName)
			if info != nil {
				return fmt.Sprintf("state.%s()", info.GetterName), nil
			}
		}
		return fmt.Sprintf("state.%s()", fieldName), nil
	case "loop":
		// Reference to current loop item/index
		if len(g.loopStack) > 0 {
			current := g.loopStack[len(g.loopStack)-1]
			switch parts[1] {
			case "item":
				return current.itemVar, nil
			case "index":
				return current.indexVar, nil
			}
		}
		return "", fmt.Errorf("no active loop for reference: %s", ref)
	default:
		return "", fmt.Errorf("unknown reference type: %s", parts[0])
	}
}

// formatConstant formats a constant value for Go code
func (g *CodeGenerator) formatConstant(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, v)
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// allocateVariable allocates a new variable for a node output
func (g *CodeGenerator) allocateVariable(node *Node, outputName string, typ string) string {
	varName := fmt.Sprintf("%s_%s", node.ID, outputName)
	g.variables[varName] = typ

	if g.nodeOutputs[node.ID] == nil {
		g.nodeOutputs[node.ID] = make(map[string]string)
	}
	g.nodeOutputs[node.ID][outputName] = varName

	return varName
}

// allocateVariableWithValue stores a reference to an existing variable as a node output
// This is used when we want other nodes to reference the same variable (e.g., for in-place updates)
func (g *CodeGenerator) allocateVariableWithValue(node *Node, outputName string, typ string, value string) {
	g.variables[value] = typ

	if g.nodeOutputs[node.ID] == nil {
		g.nodeOutputs[node.ID] = make(map[string]string)
	}
	g.nodeOutputs[node.ID][outputName] = value
}

// inferArrayItemType tries to infer the item type of an array expression
func (g *CodeGenerator) inferArrayItemType(arrayExpr string) string {
	// Try to match state.SomeField() pattern
	if strings.HasPrefix(arrayExpr, "state.") && strings.HasSuffix(arrayExpr, "()") {
		fieldName := arrayExpr[6 : len(arrayExpr)-2]
		if g.schema != nil && g.schema.RootType != nil {
			info := g.schema.GetFieldAccessInfo(g.schema.RootType.Name, fieldName)
			if info != nil && info.ParsedType.IsArray {
				return info.ParsedType.ElemType
			}
		}
	}
	return "any"
}

// writeLine writes a formatted line with proper indentation
func (g *CodeGenerator) writeLine(format string, args ...interface{}) {
	g.writeIndent()
	fmt.Fprintf(g.buf, format, args...)
	g.buf.WriteString("\n")
}

// write writes formatted text without newline
func (g *CodeGenerator) write(format string, args ...interface{}) {
	fmt.Fprintf(g.buf, format, args...)
}

// writeIndent writes the current indentation
func (g *CodeGenerator) writeIndent() {
	for i := 0; i < g.indent; i++ {
		g.buf.WriteString("\t")
	}
}

// indentStr returns the current indentation as a string
func (g *CodeGenerator) indentStr() string {
	var buf bytes.Buffer
	for i := 0; i < g.indent; i++ {
		buf.WriteString("\t")
	}
	return buf.String()
}

// ============================================================================
// Validator
// ============================================================================

// Validator validates a node graph before code generation
type Validator struct {
	nodeGraph   *NodeGraph
	errors      []error
	warnings    []error
	nodeOutputs map[string]map[string]string // nodeID -> outputName -> type
	parameters  map[string]string            // paramName -> type
}

// NewValidator creates a new validator
func NewValidator(graph *NodeGraph) *Validator {
	return &Validator{
		nodeGraph:   graph,
		errors:      make([]error, 0),
		warnings:    make([]error, 0),
		nodeOutputs: make(map[string]map[string]string),
		parameters:  make(map[string]string),
	}
}

// Validate validates the node graph
func (v *Validator) Validate() error {
	// Check for circular references between functions
	if err := v.validateNoCircularFunctions(); err != nil {
		v.addError(err)
	}

	// Check for circular references between filters
	if err := v.validateNoCircularFilters(); err != nil {
		v.addError(err)
	}

	// Validate handlers
	for i := range v.nodeGraph.Handlers {
		v.validateHandler(&v.nodeGraph.Handlers[i])
	}

	// Validate functions
	for i := range v.nodeGraph.Functions {
		v.validateFunctionDef(&v.nodeGraph.Functions[i])
	}

	// Validate filters
	for i := range v.nodeGraph.Filters {
		v.validateFilter(&v.nodeGraph.Filters[i])
	}

	if len(v.errors) > 0 {
		return fmt.Errorf("validation failed with %d errors: %v", len(v.errors), v.errors[0])
	}
	return nil
}

// GetWarnings returns validation warnings (non-fatal issues)
func (v *Validator) GetWarnings() []error {
	return v.warnings
}

// ============================================================================
// Circular Reference Detection
// ============================================================================

// validateNoCircularFunctions checks for circular function calls
func (v *Validator) validateNoCircularFunctions() error {
	// Build function call graph
	callGraph := make(map[string][]string)
	funcNames := make(map[string]bool)

	for _, fn := range v.nodeGraph.Functions {
		funcNames[fn.Name] = true
		callGraph[fn.Name] = []string{}

		for _, node := range fn.Nodes {
			if node.Type == string(NodeCallFunction) {
				if funcInput, ok := node.Inputs["function"]; ok {
					calledFunc := extractFunctionName(funcInput)
					if calledFunc != "" {
						callGraph[fn.Name] = append(callGraph[fn.Name], calledFunc)
					}
				}
			}
		}
	}

	// Detect cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var detectCycle func(node string, path []string) error
	detectCycle = func(node string, path []string) error {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range callGraph[node] {
			if !funcNames[neighbor] {
				continue // External function, skip
			}
			if !visited[neighbor] {
				if err := detectCycle(neighbor, path); err != nil {
					return err
				}
			} else if recStack[neighbor] {
				// Found cycle
				cycleStart := -1
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}
				cyclePath := append(path[cycleStart:], neighbor)
				return fmt.Errorf("circular function call detected: %s", strings.Join(cyclePath, " -> "))
			}
		}

		recStack[node] = false
		return nil
	}

	for fn := range funcNames {
		if !visited[fn] {
			if err := detectCycle(fn, []string{}); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateNoCircularFilters checks for circular filter references
func (v *Validator) validateNoCircularFilters() error {
	// Build filter dependency graph (filters can add other filters)
	depGraph := make(map[string][]string)
	filterNames := make(map[string]bool)

	for _, filter := range v.nodeGraph.Filters {
		filterNames[filter.Name] = true
		depGraph[filter.Name] = []string{}

		for _, node := range filter.Nodes {
			if node.Type == string(NodeAddFilter) {
				if filterInput, ok := node.Inputs["filterName"]; ok {
					addedFilter := extractConstantString(filterInput)
					if addedFilter != "" && addedFilter != filter.Name {
						depGraph[filter.Name] = append(depGraph[filter.Name], addedFilter)
					}
				}
			}
		}
	}

	// Detect cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var detectCycle func(node string, path []string) error
	detectCycle = func(node string, path []string) error {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range depGraph[node] {
			if !filterNames[neighbor] {
				continue
			}
			if !visited[neighbor] {
				if err := detectCycle(neighbor, path); err != nil {
					return err
				}
			} else if recStack[neighbor] {
				cycleStart := -1
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}
				cyclePath := append(path[cycleStart:], neighbor)
				return fmt.Errorf("circular filter reference detected: %s", strings.Join(cyclePath, " -> "))
			}
		}

		recStack[node] = false
		return nil
	}

	for filter := range filterNames {
		if !visited[filter] {
			if err := detectCycle(filter, []string{}); err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper to extract function name from input
func extractFunctionName(input interface{}) string {
	if str, ok := input.(string); ok {
		return strings.Trim(str, `"`)
	}
	if m, ok := input.(map[string]interface{}); ok {
		if c, ok := m["constant"].(string); ok {
			return c
		}
	}
	return ""
}

// Helper to extract constant string from input
func extractConstantString(input interface{}) string {
	if m, ok := input.(map[string]interface{}); ok {
		if c, ok := m["constant"].(string); ok {
			return c
		}
	}
	if str, ok := input.(string); ok && !strings.Contains(str, ":") {
		return str
	}
	return ""
}

// validateFunctionDef validates a function definition
func (v *Validator) validateFunctionDef(fn *FunctionDefinition) {
	nodeIDs := make(map[string]bool)
	hasReturnNode := false

	for _, node := range fn.Nodes {
		if nodeIDs[node.ID] {
			v.addError(fmt.Errorf("function %s: duplicate node ID: %s", fn.Name, node.ID))
		}
		nodeIDs[node.ID] = true

		def, err := GetNodeDefinition(node.Type)
		if err != nil {
			v.addError(fmt.Errorf("function %s, node %s: %w", fn.Name, node.ID, err))
		} else {
			v.validateRequiredInputs(fn.Name, &node, def)
		}

		// Validate Return nodes
		if node.Type == string(NodeReturn) {
			hasReturnNode = true
			_, hasValue := node.Inputs["value"]

			// If function has a return type, Return node must have a value
			if fn.ReturnType != "" && !hasValue {
				v.addError(fmt.Errorf("function %s, node %s: Return node must have 'value' input for function with return type '%s'",
					fn.Name, node.ID, fn.ReturnType))
			}
		}
	}

	// If function has a return type but no Return node, warn
	if fn.ReturnType != "" && !hasReturnNode {
		v.addWarning(fmt.Errorf("function %s: has return type '%s' but no Return node found",
			fn.Name, fn.ReturnType))
	}

	for _, edge := range fn.Flow {
		if edge.From != "start" && !nodeIDs[edge.From] {
			v.addError(fmt.Errorf("function %s: flow edge references unknown node: %s", fn.Name, edge.From))
		}
		if edge.To != "end" && !nodeIDs[edge.To] {
			v.addError(fmt.Errorf("function %s: flow edge references unknown node: %s", fn.Name, edge.To))
		}
	}
}

// validateFilter validates a filter definition
func (v *Validator) validateFilter(filter *FilterDefinition) {
	nodeIDs := make(map[string]bool)
	for _, node := range filter.Nodes {
		if nodeIDs[node.ID] {
			v.addError(fmt.Errorf("filter %s: duplicate node ID: %s", filter.Name, node.ID))
		}
		nodeIDs[node.ID] = true

		def, err := GetNodeDefinition(node.Type)
		if err != nil {
			v.addError(fmt.Errorf("filter %s, node %s: %w", filter.Name, node.ID, err))
		} else {
			v.validateRequiredInputs(filter.Name, &node, def)
		}
	}

	for _, edge := range filter.Flow {
		if edge.From != "start" && !nodeIDs[edge.From] {
			v.addError(fmt.Errorf("filter %s: flow edge references unknown node: %s", filter.Name, edge.From))
		}
		if edge.To != "end" && !nodeIDs[edge.To] {
			v.addError(fmt.Errorf("filter %s: flow edge references unknown node: %s", filter.Name, edge.To))
		}
	}
}

// validateHandler validates a single handler
func (v *Validator) validateHandler(handler *EventHandler) {
	// Check for duplicate node IDs
	nodeIDs := make(map[string]bool)
	for _, node := range handler.Nodes {
		if nodeIDs[node.ID] {
			v.addError(fmt.Errorf("handler %s: duplicate node ID: %s", handler.Name, node.ID))
		}
		nodeIDs[node.ID] = true

		// Validate node type and required inputs
		def, err := GetNodeDefinition(node.Type)
		if err != nil {
			v.addError(fmt.Errorf("handler %s, node %s: %w", handler.Name, node.ID, err))
		} else {
			// Check required inputs
			v.validateRequiredInputs(handler.Name, &node, def)
		}
	}

	// Validate flow edges
	for _, edge := range handler.Flow {
		if edge.From != "start" && !nodeIDs[edge.From] {
			v.addError(fmt.Errorf("handler %s: flow edge references unknown node: %s", handler.Name, edge.From))
		}
		if edge.To != "end" && !nodeIDs[edge.To] {
			v.addError(fmt.Errorf("handler %s: flow edge references unknown node: %s", handler.Name, edge.To))
		}
	}

	// Dead code detection - find unreachable nodes
	v.detectDeadCode(handler.Name, handler.Nodes, handler.Flow)

	// Unused output detection
	v.detectUnusedOutputs(handler.Name, handler.Nodes)
}

// validateRequiredInputs checks that all required inputs are provided for a node
func (v *Validator) validateRequiredInputs(handlerName string, node *Node, def *NodeDefinition) {
	for _, port := range def.Inputs {
		if !port.Required {
			continue
		}

		input, exists := node.Inputs[port.Name]
		if !exists {
			v.addError(fmt.Errorf("handler %s, node %s (%s): missing required input '%s'",
				handlerName, node.ID, node.Type, port.Name))
			continue
		}

		// Check if input is empty
		if input == nil {
			v.addError(fmt.Errorf("handler %s, node %s (%s): required input '%s' is nil",
				handlerName, node.ID, node.Type, port.Name))
			continue
		}

		// Check if string input is empty
		if str, ok := input.(string); ok && str == "" {
			v.addError(fmt.Errorf("handler %s, node %s (%s): required input '%s' is empty",
				handlerName, node.ID, node.Type, port.Name))
			continue
		}

		// Type checking for constant values
		v.validateInputType(handlerName, node, port.Name, port.Type, input)
	}
}

// validateInputType checks if the input value is compatible with the expected type
func (v *Validator) validateInputType(handlerName string, node *Node, inputName, expectedType string, input interface{}) {
	// Skip type checking for "any" type
	if expectedType == "any" || expectedType == "" {
		return
	}

	// Check constant values
	if m, ok := input.(map[string]interface{}); ok {
		if constant, hasConstant := m["constant"]; hasConstant {
			actualType := getValueType(constant)
			if !isTypeCompatible(expectedType, actualType) {
				v.addWarning(fmt.Errorf("%s, node %s (%s): input '%s' type mismatch: expected %s, got %s",
					handlerName, node.ID, node.Type, inputName, expectedType, actualType))
			}
		}
	}

	// Check for potential division by zero
	if node.Type == string(NodeDivide) && inputName == "b" {
		if m, ok := input.(map[string]interface{}); ok {
			if constant, hasConstant := m["constant"]; hasConstant {
				if isZeroValue(constant) {
					v.addError(fmt.Errorf("%s, node %s (%s): division by zero - input 'b' is constant 0",
						handlerName, node.ID, node.Type))
				}
			}
		}
	}

	// Check for negative array index
	if node.Type == string(NodeArrayAt) && inputName == "index" {
		if m, ok := input.(map[string]interface{}); ok {
			if constant, hasConstant := m["constant"]; hasConstant {
				if idx, ok := toInt(constant); ok && idx < 0 {
					v.addError(fmt.Errorf("%s, node %s (%s): negative array index: %d",
						handlerName, node.ID, node.Type, idx))
				}
			}
		}
	}
}

// getValueType returns the type name for a value
func getValueType(v interface{}) string {
	switch v.(type) {
	case int, int32, int64, float64, float32:
		return "number"
	case string:
		return "string"
	case bool:
		return "bool"
	case nil:
		return "nil"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// isTypeCompatible checks if actual type is compatible with expected type
func isTypeCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	// Number types are interchangeable
	if expected == "number" && (actual == "int" || actual == "float" || actual == "number") {
		return true
	}
	if (expected == "int" || expected == "float" || expected == "float64" || expected == "int64") && actual == "number" {
		return true
	}
	return false
}

// isZeroValue checks if a value is zero
func isZeroValue(v interface{}) bool {
	switch val := v.(type) {
	case int:
		return val == 0
	case int32:
		return val == 0
	case int64:
		return val == 0
	case float32:
		return val == 0
	case float64:
		return val == 0
	}
	return false
}

// toInt converts a value to int if possible
func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	}
	return 0, false
}

// addError adds a validation error
func (v *Validator) addError(err error) {
	v.errors = append(v.errors, err)
}

// addWarning adds a validation warning (non-fatal)
func (v *Validator) addWarning(err error) {
	v.warnings = append(v.warnings, err)
}

// ============================================================================
// Dead Code Detection
// ============================================================================

// detectDeadCode finds nodes that are not reachable from the start node
func (v *Validator) detectDeadCode(handlerName string, nodes []Node, flow []FlowEdge) {
	if len(nodes) == 0 {
		return
	}

	// Build adjacency list from flow edges
	reachable := make(map[string]bool)

	// Find all nodes reachable from start using BFS
	queue := []string{}
	for _, edge := range flow {
		if edge.From == "start" {
			queue = append(queue, edge.To)
			reachable[edge.To] = true
		}
	}

	// Build adjacency map
	adj := make(map[string][]string)
	for _, edge := range flow {
		if edge.From != "start" && edge.To != "end" {
			adj[edge.From] = append(adj[edge.From], edge.To)
		}
	}

	// BFS to find all reachable nodes
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, next := range adj[current] {
			if !reachable[next] {
				reachable[next] = true
				queue = append(queue, next)
			}
		}
	}

	// Check for unreachable nodes
	for _, node := range nodes {
		if !reachable[node.ID] {
			v.addWarning(fmt.Errorf("%s: unreachable node '%s' (%s) - dead code",
				handlerName, node.ID, node.Type))
		}
	}
}

// detectUnusedOutputs finds node outputs that are never used by other nodes
func (v *Validator) detectUnusedOutputs(handlerName string, nodes []Node) {
	if len(nodes) == 0 {
		return
	}

	// Collect all node outputs
	nodeOutputs := make(map[string][]string) // nodeID -> output names
	for _, node := range nodes {
		def, err := GetNodeDefinition(node.Type)
		if err != nil {
			continue
		}
		for _, output := range def.Outputs {
			nodeOutputs[node.ID] = append(nodeOutputs[node.ID], output.Name)
		}
	}

	// Find all used outputs
	usedOutputs := make(map[string]bool) // "nodeID:outputName" -> used

	for _, node := range nodes {
		for _, input := range node.Inputs {
			refs := extractNodeReferences(input)
			for _, ref := range refs {
				usedOutputs[ref] = true
			}
		}
	}

	// Report unused outputs (as warnings)
	for _, node := range nodes {
		for _, outputName := range nodeOutputs[node.ID] {
			key := fmt.Sprintf("%s:%s", node.ID, outputName)
			if !usedOutputs[key] {
				// Skip warning for control flow nodes - their outputs are often implicit
				if node.Type == string(NodeIf) || node.Type == string(NodeForEach) ||
					node.Type == string(NodeWhile) || node.Type == string(NodeReturn) {
					continue
				}
				// Skip for event/session nodes that don't typically have consumed outputs
				if node.Type == string(NodeEmitToAll) || node.Type == string(NodeEmitToPlayer) ||
					node.Type == string(NodeKickPlayer) || node.Type == string(NodeSetField) ||
					node.Type == string(NodeAddFilter) || node.Type == string(NodeRemoveFilter) ||
					node.Type == string(NodeAddEffect) || node.Type == string(NodeRemoveEffect) {
					continue
				}
				v.addWarning(fmt.Errorf("%s: node '%s' output '%s' is never used",
					handlerName, node.ID, outputName))
			}
		}
	}
}

// extractNodeReferences extracts "nodeID:outputName" references from an input value
func extractNodeReferences(input interface{}) []string {
	var refs []string

	switch val := input.(type) {
	case string:
		// Check for node:nodeID:output format
		if strings.HasPrefix(val, "node:") {
			parts := strings.SplitN(val, ":", 3)
			if len(parts) == 3 {
				refs = append(refs, parts[1]+":"+parts[2])
			}
		}
	case map[string]interface{}:
		// Check source field
		if source, ok := val["source"].(string); ok {
			if strings.HasPrefix(source, "node:") {
				parts := strings.SplitN(source, ":", 3)
				if len(parts) == 3 {
					refs = append(refs, parts[1]+":"+parts[2])
				}
			}
		}
		// Recurse into nested maps (for args, params, etc.)
		for _, v := range val {
			refs = append(refs, extractNodeReferences(v)...)
		}
	}

	return refs
}

// TopologicalSort performs topological sort on nodes
func TopologicalSort(nodes []Node, edges []FlowEdge) ([]string, error) {
	adj := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize
	for _, node := range nodes {
		inDegree[node.ID] = 0
	}

	// Build adjacency list
	for _, edge := range edges {
		if edge.From == "start" || edge.To == "end" {
			continue
		}
		adj[edge.From] = append(adj[edge.From], edge.To)
		inDegree[edge.To]++
	}

	// Find start nodes
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	// Sort
	result := []string{}
	for len(queue) > 0 {
		sort.Strings(queue) // For deterministic output
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, next := range adj[current] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	// Check for cycles
	if len(result) != len(nodes) {
		return nil, fmt.Errorf("cycle detected in node graph")
	}

	return result, nil
}

// ============================================================================
// Wait/Async Nodes
// ============================================================================

// generateWait generates code for Wait node (pauses execution for duration)
func (g *CodeGenerator) generateWait(node *Node) error {
	duration, err := g.resolveInput(node, "duration")
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
			// Try to resolve as reference
			if resolved, err := g.resolveInput(node, "unit"); err == nil {
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

	g.writeLine("select {")
	g.writeLine("case <-time.After(%s):", durationExpr)
	g.indent++
	g.writeLine("// Wait completed")
	g.indent--
	g.writeLine("case <-ctx.Done():")
	g.indent++
	g.writeLine("return ctx.Err()")
	g.indent--
	g.writeLine("}")

	return nil
}

// generateWaitUntil generates code for WaitUntil node (waits until condition is true)
func (g *CodeGenerator) generateWaitUntil(node *Node) error {
	condition, err := g.resolveInput(node, "condition")
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

	timedOutVar := g.allocateVariable(node, "timedOut", "bool")
	prefix := "_" + node.ID + "_"

	g.writeLine("%sdeadline := time.Now().Add(time.Duration(%s) * time.Millisecond)", prefix, timeout)
	g.writeLine("%sticker := time.NewTicker(time.Duration(%s) * time.Millisecond)", prefix, checkInterval)
	g.writeLine("defer %sticker.Stop()", prefix)
	g.writeLine("%s := false", timedOutVar)
	g.writeLine("")
	g.writeLine("%sloop:", prefix)
	g.writeLine("for {")
	g.indent++
	g.writeLine("if %s {", condition)
	g.indent++
	g.writeLine("break %sloop", prefix)
	g.indent--
	g.writeLine("}")
	g.writeLine("if %s > 0 && time.Now().After(%sdeadline) {", timeout, prefix)
	g.indent++
	g.writeLine("%s = true", timedOutVar)
	g.writeLine("break %sloop", prefix)
	g.indent--
	g.writeLine("}")
	g.writeLine("select {")
	g.writeLine("case <-%sticker.C:", prefix)
	g.indent++
	g.writeLine("// Check again")
	g.indent--
	g.writeLine("case <-ctx.Done():")
	g.indent++
	g.writeLine("return ctx.Err()")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")

	return nil
}

// generateTimeout generates code for Timeout node (executes body with timeout)
func (g *CodeGenerator) generateTimeout(node *Node) error {
	duration, err := g.resolveInput(node, "duration")
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

	timedOutVar := g.allocateVariable(node, "timedOut", "bool")
	prefix := "_" + node.ID + "_"

	g.writeLine("%stimeoutCtx, %scancel := context.WithTimeout(ctx, %s)", prefix, prefix, durationExpr)
	g.writeLine("defer %scancel()", prefix)
	g.writeLine("%s := false", timedOutVar)
	g.writeLine("_ = %stimeoutCtx // Use this context for nested operations", prefix)

	return nil
}

// ============================================================================
// Permission Checks
// ============================================================================

// generatePermissionChecks generates permission validation code at handler start
func (g *CodeGenerator) generatePermissionChecks(handler *EventHandler) {
	perm := handler.Permissions
	if perm == nil {
		return
	}

	if perm.HostOnly {
		g.writeLine("// Permission: hostOnly")
		g.writeLine("if !session.IsHost(senderID) {")
		g.indent++
		g.writeLine("return ErrNotHost")
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	if perm.PlayerParam != "" {
		g.writeLine("// Permission: playerParam must match sender")
		g.writeLine("if %s != senderID {", perm.PlayerParam)
		g.indent++
		g.writeLine("return ErrNotAllowed")
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	if len(perm.AllowedPlayers) > 0 {
		g.writeLine("// Permission: allowedPlayers")
		g.writeLine("if !_%s_allowedPlayers[senderID] {", handler.Name)
		g.indent++
		g.writeLine("return ErrNotAllowed")
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}
}

// ============================================================================
// Session Management Nodes
// ============================================================================

// generateKickPlayer generates code for KickPlayer node
func (g *CodeGenerator) generateKickPlayer(node *Node) error {
	playerID, err := g.resolveInput(node, "playerID")
	if err != nil {
		return err
	}

	reason := `""`
	if reasonVal, ok := node.Inputs["reason"]; ok {
		if resolved, err := g.resolveInput(node, "reason"); err == nil {
			reason = resolved
		} else if reasonMap, ok := reasonVal.(map[string]interface{}); ok {
			if constVal, ok := reasonMap["constant"]; ok {
				reason = fmt.Sprintf(`"%v"`, constVal)
			}
		}
	}

	kickedVar := g.allocateVariable(node, "kicked", "bool")
	g.writeLine("%s := session.Kick(%s, %s)", kickedVar, playerID, reason)

	return nil
}

// generateGetHostPlayer generates code for GetHostPlayer node
func (g *CodeGenerator) generateGetHostPlayer(node *Node) error {
	hostVar := g.allocateVariable(node, "hostPlayerID", "string")
	g.writeLine("%s := session.HostPlayerID()", hostVar)
	return nil
}

// generateIsHost generates code for IsHost node
func (g *CodeGenerator) generateIsHost(node *Node) error {
	playerID, err := g.resolveInput(node, "playerID")
	if err != nil {
		return err
	}

	isHostVar := g.allocateVariable(node, "isHost", "bool")
	g.writeLine("%s := session.IsHost(%s)", isHostVar, playerID)

	return nil
}

// ============================================================================
// Filter Nodes
// ============================================================================

// generateAddFilter generates code for AddFilter node
func (g *CodeGenerator) generateAddFilter(node *Node) error {
	viewerID, err := g.resolveInput(node, "viewerID")
	if err != nil {
		return err
	}

	filterID, err := g.resolveInput(node, "filterID")
	if err != nil {
		return err
	}

	filterNameRaw, err := g.resolveInput(node, "filterName")
	if err != nil {
		return err
	}

	// filterName should be used as a function identifier, not a string
	// Strip quotes if it's a string constant
	filterName := strings.Trim(filterNameRaw, `"`)

	// Check if params input exists
	paramsInput := node.Inputs["params"]

	// Generate the filter creation based on filterName
	// The filterName should be a constant that matches a filter definition
	filterVar := fmt.Sprintf("%s_filter", node.ID)

	if paramsInput != nil {
		// Resolve params - they should be passed to the filter factory
		paramsMap, ok := paramsInput.(map[string]interface{})
		if ok && len(paramsMap) > 0 {
			// Build params list
			var paramValues []string
			for _, paramValue := range paramsMap {
				resolved, err := g.resolveInputValue(paramValue)
				if err != nil {
					return fmt.Errorf("failed to resolve filter param: %w", err)
				}
				paramValues = append(paramValues, resolved)
			}
			// Filter factory with params: FilterName(param1, param2, ...)
			g.writeLine("%s := %s(%s)", filterVar, filterName, strings.Join(paramValues, ", "))
		} else {
			// Filter factory without params
			g.writeLine("%s := %s()", filterVar, filterName)
		}
	} else {
		// Filter factory without params
		g.writeLine("%s := %s()", filterVar, filterName)
	}

	// Add to registry and update session filter
	g.writeLine("filterRegistry.Add(%s, %s, %s)", viewerID, filterID, filterVar)
	g.writeLine("session.SetFilter(%s, filterRegistry.GetComposed(%s))", viewerID, viewerID)

	return nil
}

// generateRemoveFilter generates code for RemoveFilter node
func (g *CodeGenerator) generateRemoveFilter(node *Node) error {
	viewerID, err := g.resolveInput(node, "viewerID")
	if err != nil {
		return err
	}

	filterID, err := g.resolveInput(node, "filterID")
	if err != nil {
		return err
	}

	removedVar := g.allocateVariable(node, "removed", "bool")
	g.writeLine("%s := filterRegistry.Remove(%s, %s)", removedVar, viewerID, filterID)
	g.writeLine("session.SetFilter(%s, filterRegistry.GetComposed(%s))", viewerID, viewerID)

	return nil
}

// generateHasFilter generates code for HasFilter node
func (g *CodeGenerator) generateHasFilter(node *Node) error {
	viewerID, err := g.resolveInput(node, "viewerID")
	if err != nil {
		return err
	}

	filterID, err := g.resolveInput(node, "filterID")
	if err != nil {
		return err
	}

	existsVar := g.allocateVariable(node, "exists", "bool")
	g.writeLine("%s := filterRegistry.Has(%s, %s)", existsVar, viewerID, filterID)

	return nil
}

// ============================================================================
// Filter Generation
// ============================================================================

// generateFilters generates filter factory functions and the FilterRegistry type
func (g *CodeGenerator) generateFilters() error {
	// Determine root type name from schema
	rootTypeName := "GameState"
	if g.schema != nil && g.schema.RootType != nil {
		rootTypeName = g.schema.RootType.Name
	}

	g.writeLine("// ============================================================================")
	g.writeLine("// Filter Factories")
	g.writeLine("// ============================================================================")
	g.writeLine("")

	// Generate filter factory for each filter definition
	for i := range g.nodeGraph.Filters {
		filter := &g.nodeGraph.Filters[i]
		if err := g.generateFilterFactory(filter, rootTypeName); err != nil {
			return fmt.Errorf("failed to generate filter %s: %w", filter.Name, err)
		}
		g.writeLine("")
	}

	// Generate FilterRegistry type
	g.writeLine("// ============================================================================")
	g.writeLine("// Filter Registry")
	g.writeLine("// ============================================================================")
	g.writeLine("")
	g.writeLine("// FilterRegistry manages active filters per viewer")
	g.writeLine("type FilterRegistry struct {")
	g.indent++
	g.writeLine("mu      sync.RWMutex")
	g.writeLine("filters map[string]map[string]statesync.FilterFunc[*%s]", rootTypeName)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// NewFilterRegistry
	g.writeLine("// NewFilterRegistry creates a new filter registry")
	g.writeLine("func NewFilterRegistry() *FilterRegistry {")
	g.indent++
	g.writeLine("return &FilterRegistry{")
	g.indent++
	g.writeLine("filters: make(map[string]map[string]statesync.FilterFunc[*%s]),", rootTypeName)
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Add method
	g.writeLine("// Add adds a filter instance for a viewer")
	g.writeLine("func (r *FilterRegistry) Add(viewerID, filterID string, filter statesync.FilterFunc[*%s]) {", rootTypeName)
	g.indent++
	g.writeLine("r.mu.Lock()")
	g.writeLine("defer r.mu.Unlock()")
	g.writeLine("if r.filters[viewerID] == nil {")
	g.indent++
	g.writeLine("r.filters[viewerID] = make(map[string]statesync.FilterFunc[*%s])", rootTypeName)
	g.indent--
	g.writeLine("}")
	g.writeLine("r.filters[viewerID][filterID] = filter")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Remove method
	g.writeLine("// Remove removes a filter instance")
	g.writeLine("func (r *FilterRegistry) Remove(viewerID, filterID string) bool {")
	g.indent++
	g.writeLine("r.mu.Lock()")
	g.writeLine("defer r.mu.Unlock()")
	g.writeLine("if r.filters[viewerID] == nil {")
	g.indent++
	g.writeLine("return false")
	g.indent--
	g.writeLine("}")
	g.writeLine("_, ok := r.filters[viewerID][filterID]")
	g.writeLine("delete(r.filters[viewerID], filterID)")
	g.writeLine("return ok")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Has method
	g.writeLine("// Has checks if a filter exists")
	g.writeLine("func (r *FilterRegistry) Has(viewerID, filterID string) bool {")
	g.indent++
	g.writeLine("r.mu.RLock()")
	g.writeLine("defer r.mu.RUnlock()")
	g.writeLine("if r.filters[viewerID] == nil {")
	g.indent++
	g.writeLine("return false")
	g.indent--
	g.writeLine("}")
	g.writeLine("_, ok := r.filters[viewerID][filterID]")
	g.writeLine("return ok")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// GetComposed method
	g.writeLine("// GetComposed returns a composed filter for a viewer")
	g.writeLine("func (r *FilterRegistry) GetComposed(viewerID string) statesync.FilterFunc[*%s] {", rootTypeName)
	g.indent++
	g.writeLine("r.mu.RLock()")
	g.writeLine("filters := r.filters[viewerID]")
	g.writeLine("if len(filters) == 0 {")
	g.indent++
	g.writeLine("r.mu.RUnlock()")
	g.writeLine("return nil")
	g.indent--
	g.writeLine("}")
	g.writeLine("fns := make([]statesync.FilterFunc[*%s], 0, len(filters))", rootTypeName)
	g.writeLine("for _, f := range filters {")
	g.indent++
	g.writeLine("fns = append(fns, f)")
	g.indent--
	g.writeLine("}")
	g.writeLine("r.mu.RUnlock()")
	g.writeLine("")
	g.writeLine("return func(state *%s) *%s {", rootTypeName, rootTypeName)
	g.indent++
	g.writeLine("for _, fn := range fns {")
	g.indent++
	g.writeLine("state = fn(state)")
	g.indent--
	g.writeLine("}")
	g.writeLine("return state")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Global filterRegistry variable
	g.writeLine("// Global filter registry - initialize with NewFilterRegistry()")
	g.writeLine("var filterRegistry *FilterRegistry")
	g.writeLine("")

	return nil
}

// generateFilterFactory generates a single filter factory function
func (g *CodeGenerator) generateFilterFactory(filter *FilterDefinition, rootTypeName string) error {
	// Generate function signature
	g.writeLine("// %s creates a filter: %s", filter.Name, filter.Description)
	g.write("func %s(", filter.Name)

	// Add parameters
	for i, param := range filter.Parameters {
		if i > 0 {
			g.write(", ")
		}
		g.write("%s %s", param.Name, param.Type)
	}

	g.write(") statesync.FilterFunc[*%s] {\n", rootTypeName)
	g.indent++

	g.writeLine("return func(state *%s) *%s {", rootTypeName, rootTypeName)
	g.indent++

	// Clone state at the start
	// NOTE: ShallowClone creates a copy of the top-level struct but shares nested slices/maps.
	// This is intentional for performance - filters should only hide/mask data, not deeply modify it.
	// If deep modifications are needed, implement DeepClone in the schema or use immutable patterns.
	g.writeLine("// Clone state for safe modification (shallow - nested structures are shared)")
	g.writeLine("filtered := state.ShallowClone()")
	g.writeLine("")

	// Set up variables for filter generation
	g.variables = make(map[string]string)
	g.nodeOutputs = make(map[string]map[string]string)
	g.loopStack = []loopContext{}
	g.generatedNodes = make(map[string]bool)

	// Add parameters to variables (use param: prefix for resolution)
	for _, param := range filter.Parameters {
		g.variables["param:"+param.Name] = param.Type
		g.variables[param.Name] = param.Type
	}

	// Add state to variables (filters work on cloned state)
	g.variables["filtered"] = "*" + rootTypeName
	g.variables["state"] = "*" + rootTypeName

	// Generate nodes following flow
	if len(filter.Flow) > 0 {
		// Find start node
		startNodeID := ""
		for _, edge := range filter.Flow {
			if edge.From == "start" {
				startNodeID = edge.To
				break
			}
		}

		if startNodeID != "" {
			// Create a temporary handler-like structure to use existing generation logic
			tempHandler := &EventHandler{
				Name:       filter.Name,
				Parameters: filter.Parameters,
				Nodes:      filter.Nodes,
				Flow:       filter.Flow,
			}
			g.currentHandler = tempHandler

			if err := g.generateBodyChain(startNodeID); err != nil {
				return fmt.Errorf("failed to generate filter body: %w", err)
			}
		}
	}

	g.writeLine("")
	g.writeLine("return filtered")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")

	return nil
}

// ============================================================================
// Function Generation - Reusable Node-Based Functions
// ============================================================================

// generateFunctions generates Go functions from FunctionDefinition nodes
func (g *CodeGenerator) generateFunctions() error {
	// Determine root type name from schema
	rootTypeName := "GameState"
	if g.schema != nil && g.schema.RootType != nil {
		rootTypeName = g.schema.RootType.Name
	}

	g.writeLine("// ============================================================================")
	g.writeLine("// Reusable Functions")
	g.writeLine("// ============================================================================")
	g.writeLine("")

	// Generate each function
	for i := range g.nodeGraph.Functions {
		fn := &g.nodeGraph.Functions[i]
		if err := g.generateFunction(fn, rootTypeName); err != nil {
			return fmt.Errorf("failed to generate function %s: %w", fn.Name, err)
		}
		g.writeLine("")
	}

	return nil
}

// generateFunction generates a single Go function from FunctionDefinition
func (g *CodeGenerator) generateFunction(fn *FunctionDefinition, rootTypeName string) error {
	// Build parameter list
	var params []string
	params = append(params, fmt.Sprintf("state *%s", rootTypeName))
	params = append(params, "session *statesync.TrackedSession[*"+rootTypeName+"]")

	for _, p := range fn.Parameters {
		params = append(params, fmt.Sprintf("%s %s", p.Name, goType(p.Type)))
	}

	// Determine return type
	returnType := ""
	if fn.ReturnType != "" {
		returnType = " " + goType(fn.ReturnType)
	}

	// Write function signature
	if fn.Description != "" {
		g.writeLine("// %s - %s", fn.Name, fn.Description)
	} else {
		g.writeLine("// %s is a reusable function defined in the node graph", fn.Name)
	}
	g.writeLine("func %s(%s)%s {", fn.Name, strings.Join(params, ", "), returnType)
	g.indent++

	// Set up generation context
	g.variables = make(map[string]string)
	g.nodeOutputs = make(map[string]map[string]string)
	g.loopStack = []loopContext{}
	g.generatedNodes = make(map[string]bool)

	// Add parameters to variables (use param: prefix for resolution)
	for _, param := range fn.Parameters {
		g.variables["param:"+param.Name] = param.Type
		g.variables[param.Name] = param.Type
	}

	// Generate nodes following flow
	if len(fn.Flow) > 0 {
		// Find start node
		startNodeID := ""
		for _, edge := range fn.Flow {
			if edge.From == "start" {
				startNodeID = edge.To
				break
			}
		}

		if startNodeID != "" {
			// Create a temporary handler-like structure to use existing generation logic
			tempHandler := &EventHandler{
				Name:       fn.Name,
				Parameters: fn.Parameters,
				Nodes:      fn.Nodes,
				Flow:       fn.Flow,
			}
			g.currentHandler = tempHandler

			if err := g.generateBodyChain(startNodeID); err != nil {
				return fmt.Errorf("failed to generate function body: %w", err)
			}
		}
	}

	// If no return was generated and we have a return type, add a default return
	if fn.ReturnType != "" {
		// Check if the last generated line was a return
		// If not, add a zero-value return
		g.writeLine("// Default return if no explicit Return node")
		g.writeLine("return %s", getZeroValue(fn.ReturnType))
	}

	g.indent--
	g.writeLine("}")

	return nil
}

// goType converts a type name to Go type
func goType(typeName string) string {
	switch typeName {
	case "int":
		return "int"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "float", "float64":
		return "float64"
	case "float32":
		return "float32"
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "any":
		return "interface{}"
	default:
		return typeName
	}
}
