package main

import (
	"fmt"
	"sync"
)

// ============================================================================
// Node Registry - Extensibility API for Custom Nodes
// ============================================================================
//
// This file provides the public API for registering custom node types.
// External packages can use this to extend the logicgen node system
// with custom nodes for their specific use cases.
//
// Example usage in your framework:
//
//	import "github.com/your/project/logicgen"
//
//	func init() {
//	    logicgen.RegisterNode(logicgen.NodeDefinition{
//	        Type:        "MyCustomNode",
//	        Category:    "Custom",
//	        Description: "Does something custom",
//	        Inputs: []logicgen.PortDefinition{
//	            {Name: "value", Type: "string", Required: true},
//	        },
//	        Outputs: []logicgen.PortDefinition{
//	            {Name: "result", Type: "string"},
//	        },
//	        Generator: func(ctx *logicgen.GeneratorContext, node *logicgen.Node) error {
//	            value, err := ctx.ResolveInput(node, "value")
//	            if err != nil {
//	                return err
//	            }
//	            output := ctx.AllocateVariable(node, "result", "string")
//	            ctx.WriteLine("%s := processCustom(%s)", output, value)
//	            return nil
//	        },
//	    })
//	}

var (
	registryMu    sync.RWMutex
	coreNodes     = make(map[NodeType]*NodeDefinition) // Core nodes (registered by init() in nodes_*.go)
	customNodes   = make(map[NodeType]*NodeDefinition) // User-registered custom nodes
	customImports = make(map[string]bool)
)

// RegisterNode registers a custom node type that can be used in node graphs.
// This function is thread-safe and can be called from init() functions.
//
// The Generator function in the NodeDefinition will be called during code
// generation. If Generator is nil, the node will produce a TODO comment
// in the generated code.
func RegisterNode(def NodeDefinition) error {
	if def.Type == "" {
		return fmt.Errorf("node type cannot be empty")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	// Check if already registered in built-in nodes
	if _, exists := nodeDefinitions[def.Type]; exists {
		return fmt.Errorf("node type %s is already registered as built-in", def.Type)
	}

	// Check if already registered in core nodes
	if _, exists := coreNodes[def.Type]; exists {
		return fmt.Errorf("node type %s is already registered as core", def.Type)
	}

	// Check if already registered in custom nodes
	if _, exists := customNodes[def.Type]; exists {
		return fmt.Errorf("node type %s is already registered", def.Type)
	}

	customNodes[def.Type] = &def
	return nil
}

// RegisterCoreNode registers a core node type (used by internal init() functions).
// Core nodes are not cleared by ClearCustomNodes().
func RegisterCoreNode(def NodeDefinition) error {
	if def.Type == "" {
		return fmt.Errorf("node type cannot be empty")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	// Check all registries
	if _, exists := nodeDefinitions[def.Type]; exists {
		return fmt.Errorf("node type %s is already registered as built-in", def.Type)
	}
	if _, exists := coreNodes[def.Type]; exists {
		return fmt.Errorf("node type %s is already registered as core", def.Type)
	}
	if _, exists := customNodes[def.Type]; exists {
		return fmt.Errorf("node type %s is already registered", def.Type)
	}

	coreNodes[def.Type] = &def
	return nil
}

// MustRegisterCoreNode is like RegisterCoreNode but panics on error.
func MustRegisterCoreNode(def NodeDefinition) {
	if err := RegisterCoreNode(def); err != nil {
		panic(fmt.Sprintf("failed to register core node %s: %v", def.Type, err))
	}
}

// MustRegisterNode registers a core node and panics on error.
// Use this in init() functions for core nodes. Core nodes are not cleared by ClearCustomNodes().
func MustRegisterNode(def NodeDefinition) {
	if err := RegisterCoreNode(def); err != nil {
		panic(fmt.Sprintf("failed to register node %s: %v", def.Type, err))
	}
}

// RegisterNodeImport registers an import path that should be added to
// generated files when custom nodes require external packages.
func RegisterNodeImport(importPath string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	customImports[importPath] = true
}

// GetCustomNodeDefinition returns the definition for a custom or core node type.
// Returns nil if the node type is not registered.
func GetCustomNodeDefinition(nodeType NodeType) *NodeDefinition {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if def, ok := coreNodes[nodeType]; ok {
		return def
	}
	return customNodes[nodeType]
}

// GetAllNodeDefinitions returns all registered node definitions,
// including built-in, core, and custom nodes.
func GetAllNodeDefinitions() map[NodeType]*NodeDefinition {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make(map[NodeType]*NodeDefinition, len(nodeDefinitions)+len(coreNodes)+len(customNodes))
	for k, v := range nodeDefinitions {
		result[k] = v
	}
	for k, v := range coreNodes {
		result[k] = v
	}
	for k, v := range customNodes {
		result[k] = v
	}
	return result
}

// GetCustomImports returns all registered import paths for custom nodes.
func GetCustomImports() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	imports := make([]string, 0, len(customImports))
	for imp := range customImports {
		imports = append(imports, imp)
	}
	return imports
}

// IsNodeRegistered checks if a node type is registered (built-in, core, or custom).
func IsNodeRegistered(nodeType NodeType) bool {
	if _, exists := nodeDefinitions[nodeType]; exists {
		return true
	}
	registryMu.RLock()
	defer registryMu.RUnlock()
	if _, exists := coreNodes[nodeType]; exists {
		return true
	}
	_, exists := customNodes[nodeType]
	return exists
}

// ListNodesByCategory returns all node types grouped by category.
func ListNodesByCategory() map[string][]NodeType {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make(map[string][]NodeType)

	// Built-in nodes
	for _, def := range nodeDefinitions {
		result[def.Category] = append(result[def.Category], def.Type)
	}

	// Core nodes
	for _, def := range coreNodes {
		result[def.Category] = append(result[def.Category], def.Type)
	}

	// Custom nodes
	for _, def := range customNodes {
		result[def.Category] = append(result[def.Category], def.Type)
	}

	return result
}

// ClearCustomNodes removes all custom node registrations.
// This is primarily useful for testing.
func ClearCustomNodes() {
	registryMu.Lock()
	defer registryMu.Unlock()
	customNodes = make(map[NodeType]*NodeDefinition)
	customImports = make(map[string]bool)
}

// ============================================================================
// GeneratorContext - Context provided to custom node generators
// ============================================================================

// GeneratorContext provides access to code generation utilities for custom nodes.
// This is passed to the Generator function in NodeDefinition.
type GeneratorContext struct {
	gen *CodeGenerator
}

// ResolveInput resolves an input value for a node.
// Returns the Go expression that evaluates to the input value.
func (ctx *GeneratorContext) ResolveInput(node *Node, inputName string) (string, error) {
	return ctx.gen.resolveInput(node, inputName)
}

// ResolveInputOptional resolves an optional input, returning defaultValue if not specified.
func (ctx *GeneratorContext) ResolveInputOptional(node *Node, inputName, defaultValue string) string {
	val, err := ctx.gen.resolveInput(node, inputName)
	if err != nil || val == "" {
		return defaultValue
	}
	return val
}

// AllocateVariable allocates a unique variable name for a node output.
func (ctx *GeneratorContext) AllocateVariable(node *Node, outputName, typeName string) string {
	return ctx.gen.allocateVariable(node, outputName, typeName)
}

// WriteLine writes a line of code with proper indentation.
// Supports fmt.Sprintf-style formatting.
func (ctx *GeneratorContext) WriteLine(format string, args ...interface{}) {
	ctx.gen.writeLine(format, args...)
}

// WriteRaw writes raw code without indentation.
func (ctx *GeneratorContext) WriteRaw(code string) {
	ctx.gen.buf.WriteString(code)
}

// Indent increases the indentation level.
func (ctx *GeneratorContext) Indent() {
	ctx.gen.indent++
}

// Dedent decreases the indentation level.
func (ctx *GeneratorContext) Dedent() {
	ctx.gen.indent--
}

// GetSchema returns the current schema context, or nil if not available.
func (ctx *GeneratorContext) GetSchema() *SchemaContext {
	return ctx.gen.schema
}

// GetCurrentHandler returns the current event handler being generated.
func (ctx *GeneratorContext) GetCurrentHandler() *EventHandler {
	return ctx.gen.currentHandler
}

// IsDebugMode returns true if debug mode is enabled.
func (ctx *GeneratorContext) IsDebugMode() bool {
	return ctx.gen.debugMode
}

// AddImport adds an import to the generated file.
func (ctx *GeneratorContext) AddImport(importPath string) {
	ctx.gen.addImport(importPath)
}

// GetNodeOutput returns the variable name for a node's output.
// Returns empty string if the output doesn't exist.
func (ctx *GeneratorContext) GetNodeOutput(nodeID, outputName string) string {
	key := fmt.Sprintf("node:%s:%s", nodeID, outputName)
	if varName, ok := ctx.gen.variables[key]; ok {
		return varName
	}
	return ""
}

// GeneratePathAccess generates code to access a state path and returns the expression.
// Returns (code, expression, error) where code is any setup code needed,
// and expression is the Go expression to access the value.
func (ctx *GeneratorContext) GeneratePathAccess(pathStr string, forWrite bool) (string, string, error) {
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return "", "", err
	}
	return ctx.gen.generatePathAccess(parsedPath, forWrite)
}

// GeneratePathSet generates code to set a value at a state path.
func (ctx *GeneratorContext) GeneratePathSet(pathStr string, value string) error {
	parsedPath, err := ParsePath(pathStr)
	if err != nil {
		return err
	}
	return ctx.gen.generatePathSet(parsedPath, value)
}

// GetFieldAccessInfo returns schema information for a field.
func (ctx *GeneratorContext) GetFieldAccessInfo(typeName, fieldName string) *FieldAccessInfo {
	if ctx.gen.schema == nil {
		return nil
	}
	return ctx.gen.schema.GetFieldAccessInfo(typeName, fieldName)
}

// GetRootTypeName returns the name of the root state type.
func (ctx *GeneratorContext) GetRootTypeName() string {
	if ctx.gen.schema != nil && ctx.gen.schema.RootType != nil {
		return ctx.gen.schema.RootType.Name
	}
	return "GameState"
}

// GenerateBodyChain generates code for a chain of nodes (for control flow).
func (ctx *GeneratorContext) GenerateBodyChain(startNodeID string) error {
	return ctx.gen.generateBodyChain(startNodeID)
}

// GetFlowEdgesFrom returns all flow edges from a node.
func (ctx *GeneratorContext) GetFlowEdgesFrom(nodeID string) []FlowEdge {
	return ctx.gen.getFlowEdgesFrom(nodeID)
}

// FindNode finds a node by ID in the current handler.
func (ctx *GeneratorContext) FindNode(nodeID string) *Node {
	return ctx.gen.findNode(ctx.gen.currentHandler, nodeID)
}

// GetLoopItemVar returns the current loop item variable (for ForEach).
func (ctx *GeneratorContext) GetLoopItemVar() string {
	if len(ctx.gen.loopStack) == 0 {
		return ""
	}
	return ctx.gen.loopStack[len(ctx.gen.loopStack)-1].itemVar
}

// GetLoopIndexVar returns the current loop index variable (for ForEach).
func (ctx *GeneratorContext) GetLoopIndexVar() string {
	if len(ctx.gen.loopStack) == 0 {
		return ""
	}
	return ctx.gen.loopStack[len(ctx.gen.loopStack)-1].indexVar
}

// PushLoop pushes a loop context onto the stack.
func (ctx *GeneratorContext) PushLoop(nodeID, itemVar, indexVar, arrayExpr string) {
	ctx.gen.loopStack = append(ctx.gen.loopStack, loopContext{
		nodeID:    nodeID,
		itemVar:   itemVar,
		indexVar:  indexVar,
		arrayExpr: arrayExpr,
	})
}

// PopLoop pops a loop context from the stack.
func (ctx *GeneratorContext) PopLoop() {
	if len(ctx.gen.loopStack) > 0 {
		ctx.gen.loopStack = ctx.gen.loopStack[:len(ctx.gen.loopStack)-1]
	}
}

// NodeID returns a unique prefix for the node to avoid variable collisions.
func (ctx *GeneratorContext) NodePrefix(node *Node) string {
	return "_" + node.ID + "_"
}

// IsContextMode returns true if the generator is in context mode (for async nodes).
func (ctx *GeneratorContext) IsContextMode() bool {
	return ctx.gen.contextMode
}

// ResolveReference resolves a reference string (like "node:xyz:result" or "param:foo")
// and returns the Go expression that evaluates to the value.
func (ctx *GeneratorContext) ResolveReference(ref string) (string, error) {
	return ctx.gen.resolveReference(ref)
}

// ============================================================================
// Integration with CodeGenerator
// ============================================================================

// createGeneratorContext creates a GeneratorContext for custom node generation.
func (g *CodeGenerator) createGeneratorContext() *GeneratorContext {
	return &GeneratorContext{gen: g}
}

// tryCustomNodeGenerator attempts to generate code using a core or custom node's Generator.
// Returns true if a generator was found and executed, false otherwise.
func (g *CodeGenerator) tryCustomNodeGenerator(node *Node) (bool, error) {
	registryMu.RLock()
	def, exists := coreNodes[NodeType(node.Type)]
	if !exists {
		def, exists = customNodes[NodeType(node.Type)]
	}
	registryMu.RUnlock()

	if !exists {
		return false, nil
	}

	if def.Generator == nil {
		// Node without generator - produce TODO comment
		g.writeLine("// TODO: Node %s has no generator", node.Type)
		return true, nil
	}

	ctx := g.createGeneratorContext()
	return true, def.Generator(ctx, node)
}
