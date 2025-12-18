package main

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"
)

// CodeGenerator generates Go code from a node graph
type CodeGenerator struct {
	buf           *bytes.Buffer
	indent        int
	nodeGraph     *NodeGraph
	currentHandler *EventHandler
	variables     map[string]string // variable name -> type
	nodeOutputs   map[string]map[string]string // nodeID -> output name -> variable name
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(graph *NodeGraph) *CodeGenerator {
	return &CodeGenerator{
		buf:         &bytes.Buffer{},
		nodeGraph:   graph,
		variables:   make(map[string]string),
		nodeOutputs: make(map[string]map[string]string),
	}
}

// Generate generates Go code from the node graph
func (g *CodeGenerator) Generate() ([]byte, error) {
	// Package declaration
	g.writeLine("package %s", g.nodeGraph.Package)
	g.writeLine("")

	// Imports
	g.writeLine("import (")
	g.indent++
	g.writeLine(`"fmt"`)
	g.writeLine(`"github.com/mxkacsa/statesync"`)
	for _, imp := range g.nodeGraph.Imports {
		g.writeLine(`"%s"`, imp)
	}
	g.indent--
	g.writeLine(")")
	g.writeLine("")

	// Generate each event handler
	for i := range g.nodeGraph.Handlers {
		if err := g.generateHandler(&g.nodeGraph.Handlers[i]); err != nil {
			return nil, fmt.Errorf("failed to generate handler %s: %w", g.nodeGraph.Handlers[i].Name, err)
		}
		g.writeLine("")
	}

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

	// Function signature
	g.writeLine("// %s handles the %s event", handler.Name, handler.Event)
	g.write("func %s(session *statesync.TrackedSession[*GameState, any, string]", handler.Name)

	// Add parameters
	for _, param := range handler.Parameters {
		g.write(", %s %s", param.Name, param.Type)
		g.variables[param.Name] = param.Type
	}
	g.write(") error {\n")
	g.indent++

	// Build execution order from flow
	executionOrder, err := g.buildExecutionOrder(handler)
	if err != nil {
		return err
	}

	// Generate code for each node in order
	for _, nodeID := range executionOrder {
		node := g.findNode(handler, nodeID)
		if node == nil {
			continue // Skip flow markers like "start"
		}

		if err := g.generateNode(node); err != nil {
			return fmt.Errorf("failed to generate node %s: %w", node.ID, err)
		}
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
	g.writeLine("// Node: %s (%s)", node.ID, node.Type)

	switch NodeType(node.Type) {
	case NodeGetPlayer:
		return g.generateGetPlayer(node)
	case NodeSetField:
		return g.generateSetField(node)
	case NodeAddToArray:
		return g.generateAddToArray(node)
	case NodeRemoveFromArray:
		return g.generateRemoveFromArray(node)
	case NodeCompare:
		return g.generateCompare(node)
	case NodeEmitToAll:
		return g.generateEmitToAll(node)
	case NodeAdd:
		return g.generateMathOp(node, "+")
	case NodeSubtract:
		return g.generateMathOp(node, "-")
	case NodeMultiply:
		return g.generateMathOp(node, "*")
	case NodeDivide:
		return g.generateMathOp(node, "/")
	default:
		return fmt.Errorf("unsupported node type: %s", node.Type)
	}
}

// generateGetPlayer generates code for GetPlayer node
func (g *CodeGenerator) generateGetPlayer(node *Node) error {
	playerIDSrc, err := g.resolveInput(node, "playerID")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "player", "*Player")

	g.writeLine("state := session.State().Get()")
	g.writeLine("var %s *Player", outputVar)
	g.writeLine("for i := range state.Players {")
	g.indent++
	g.writeLine("if state.Players[i].ID == %s {", playerIDSrc)
	g.indent++
	g.writeLine("%s = &state.Players[i]", outputVar)
	g.writeLine("break")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("if %s == nil {", outputVar)
	g.indent++
	g.writeLine(`return fmt.Errorf("player not found: %%s", %s)`, playerIDSrc)
	g.indent--
	g.writeLine("}")

	return nil
}

// generateSetField generates code for SetField node
func (g *CodeGenerator) generateSetField(node *Node) error {
	pathSrc, err := g.resolveInput(node, "path")
	if err != nil {
		return err
	}
	valueSrc, err := g.resolveInput(node, "value")
	if err != nil {
		return err
	}

	// Parse the path (e.g., "player.Hand" or "GameState.Round")
	path := strings.Trim(pathSrc, `"`)

	g.writeLine("session.State().Update(func(s **GameState) {")
	g.indent++
	g.writeLine("// TODO: Set %s = %s", path, valueSrc)
	g.writeLine("// Implementation depends on the actual path structure")
	g.indent--
	g.writeLine("})")

	return nil
}

// generateAddToArray generates code for AddToArray node
func (g *CodeGenerator) generateAddToArray(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
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

// generateRemoveFromArray generates code for RemoveFromArray node
func (g *CodeGenerator) generateRemoveFromArray(node *Node) error {
	arraySrc, err := g.resolveInput(node, "array")
	if err != nil {
		return err
	}

	outputVar := g.allocateVariable(node, "result", "[]any")

	// Check if filtering by predicate or index
	if _, hasIndex := node.Inputs["index"]; hasIndex {
		indexSrc, err := g.resolveInput(node, "index")
		if err != nil {
			return err
		}
		g.writeLine("%s := append(%s[:%s], %s[%s+1:]...)", outputVar, arraySrc, indexSrc, arraySrc, indexSrc)
	} else {
		// Predicate-based removal (filter out elements)
		g.writeLine("%s := make([]any, 0, len(%s))", outputVar, arraySrc)
		g.writeLine("for _, item := range %s {", arraySrc)
		g.indent++
		g.writeLine("// Filter condition here")
		g.writeLine("%s = append(%s, item)", outputVar, outputVar)
		g.indent--
		g.writeLine("}")
	}

	return nil
}

// generateCompare generates code for Compare node
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

// generateEmitToAll generates code for EmitToAll node
func (g *CodeGenerator) generateEmitToAll(node *Node) error {
	eventTypeSrc, err := g.resolveInput(node, "eventType")
	if err != nil {
		return err
	}

	// Handle payload if present
	if _, hasPayload := node.Inputs["payload"]; hasPayload {
		g.writeLine("enc := statesync.NewEventPayloadEncoder()")
		// TODO: Encode payload fields
		g.writeLine("session.Emit(%s, enc.Bytes())", eventTypeSrc)
	} else {
		g.writeLine("session.Emit(%s, nil)", eventTypeSrc)
	}

	return nil
}

// generateMathOp generates code for math operations
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
		// Parse reference: "param:name", "node:id:output", "variable:name"
		parts := strings.SplitN(ref, ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid reference: %s", ref)
		}

		switch parts[0] {
		case "param":
			return parts[1], nil
		case "node":
			// Format: "node:nodeID:outputName"
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

// formatConstant formats a constant value for Go code
func (g *CodeGenerator) formatConstant(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, v)
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
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

// Validator validates a node graph before code generation
type Validator struct {
	nodeGraph *NodeGraph
	errors    []error
}

// NewValidator creates a new validator
func NewValidator(graph *NodeGraph) *Validator {
	return &Validator{
		nodeGraph: graph,
		errors:    make([]error, 0),
	}
}

// Validate validates the node graph
func (v *Validator) Validate() error {
	for i := range v.nodeGraph.Handlers {
		v.validateHandler(&v.nodeGraph.Handlers[i])
	}

	if len(v.errors) > 0 {
		return fmt.Errorf("validation failed with %d errors: %v", len(v.errors), v.errors[0])
	}
	return nil
}

// validateHandler validates a single handler
func (v *Validator) validateHandler(handler *EventHandler) {
	// Check for duplicate node IDs
	nodeIDs := make(map[string]bool)
	for _, node := range handler.Nodes {
		if nodeIDs[node.ID] {
			v.addError(fmt.Errorf("duplicate node ID: %s", node.ID))
		}
		nodeIDs[node.ID] = true

		// Validate node type
		if _, err := GetNodeDefinition(node.Type); err != nil {
			v.addError(err)
		}
	}

	// Validate flow edges
	for _, edge := range handler.Flow {
		if edge.From != "start" && !nodeIDs[edge.From] {
			v.addError(fmt.Errorf("flow edge references unknown node: %s", edge.From))
		}
		if edge.To != "end" && !nodeIDs[edge.To] {
			v.addError(fmt.Errorf("flow edge references unknown node: %s", edge.To))
		}
	}
}

// addError adds a validation error
func (v *Validator) addError(err error) {
	v.errors = append(v.errors, err)
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
