package main

import (
	"strings"
	"testing"
)

// ============================================================================
// RegisterNode Tests
// ============================================================================

func TestRegisterNode(t *testing.T) {
	// Clear any existing custom nodes
	ClearCustomNodes()
	defer ClearCustomNodes()

	def := NodeDefinition{
		Type:        "TestNode",
		Category:    "Test",
		Description: "A test node",
		Inputs: []PortDefinition{
			{Name: "value", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string"},
		},
	}

	err := RegisterNode(def)
	if err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Verify registration
	if !IsNodeRegistered("TestNode") {
		t.Error("TestNode should be registered")
	}

	retrieved := GetCustomNodeDefinition("TestNode")
	if retrieved == nil {
		t.Fatal("GetCustomNodeDefinition returned nil")
	}
	if retrieved.Type != "TestNode" {
		t.Errorf("Type = %v, want TestNode", retrieved.Type)
	}
	if retrieved.Category != "Test" {
		t.Errorf("Category = %v, want Test", retrieved.Category)
	}
}

func TestRegisterNode_EmptyType(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	def := NodeDefinition{
		Type:     "",
		Category: "Test",
	}

	err := RegisterNode(def)
	if err == nil {
		t.Error("RegisterNode should fail for empty type")
	}
}

func TestRegisterNode_DuplicateCustom(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	def := NodeDefinition{
		Type:     "DuplicateNode",
		Category: "Test",
	}

	err := RegisterNode(def)
	if err != nil {
		t.Fatalf("First RegisterNode failed: %v", err)
	}

	err = RegisterNode(def)
	if err == nil {
		t.Error("RegisterNode should fail for duplicate registration")
	}
}

func TestRegisterNode_BuiltInCollision(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	// Try to register a node with a built-in type name
	def := NodeDefinition{
		Type:     NodeCompare, // This is a built-in
		Category: "Test",
	}

	err := RegisterNode(def)
	if err == nil {
		t.Error("RegisterNode should fail when colliding with built-in")
	}
}

func TestMustRegisterNode(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	def := NodeDefinition{
		Type:     "MustTestNode",
		Category: "Test",
	}

	// Should not panic
	MustRegisterNode(def)

	if !IsNodeRegistered("MustTestNode") {
		t.Error("MustTestNode should be registered")
	}
}

func TestMustRegisterNode_Panic(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	def := NodeDefinition{
		Type: "", // Empty type should cause error
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRegisterNode should panic on error")
		}
	}()

	MustRegisterNode(def)
}

// ============================================================================
// Custom Import Tests
// ============================================================================

func TestRegisterNodeImport(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	RegisterNodeImport("github.com/custom/package")

	imports := GetCustomImports()
	found := false
	for _, imp := range imports {
		if imp == "github.com/custom/package" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Custom import not found")
	}
}

// ============================================================================
// GetAllNodeDefinitions Tests
// ============================================================================

func TestGetAllNodeDefinitions(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	// Register a custom node
	RegisterNode(NodeDefinition{
		Type:     "CustomTestNode",
		Category: "Custom",
	})

	all := GetAllNodeDefinitions()

	// Should include built-in nodes
	if _, ok := all[NodeCompare]; !ok {
		t.Error("Built-in Compare node should be in all definitions")
	}

	// Should include custom nodes
	if _, ok := all["CustomTestNode"]; !ok {
		t.Error("Custom node should be in all definitions")
	}
}

// ============================================================================
// ListNodesByCategory Tests
// ============================================================================

func TestListNodesByCategory(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	// Register a custom node
	RegisterNode(NodeDefinition{
		Type:     "MyGameNode",
		Category: "GameSpecific",
	})

	byCategory := ListNodesByCategory()

	// Check built-in category
	if _, ok := byCategory["Logic"]; !ok {
		t.Error("Logic category should exist")
	}

	// Check custom category
	if nodes, ok := byCategory["GameSpecific"]; !ok {
		t.Error("GameSpecific category should exist")
	} else {
		found := false
		for _, n := range nodes {
			if n == "MyGameNode" {
				found = true
				break
			}
		}
		if !found {
			t.Error("MyGameNode should be in GameSpecific category")
		}
	}
}

// ============================================================================
// GeneratorContext Tests
// ============================================================================

func TestGeneratorContext_CustomNodeWithGenerator(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	// Register a custom node with a generator
	generatorCalled := false
	RegisterNode(NodeDefinition{
		Type:        "HelloWorldNode",
		Category:    "Custom",
		Description: "Outputs Hello World",
		Inputs: []PortDefinition{
			{Name: "name", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "greeting", Type: "string"},
		},
		Generator: func(ctx *GeneratorContext, node *Node) error {
			generatorCalled = true
			name, err := ctx.ResolveInput(node, "name")
			if err != nil {
				return err
			}
			output := ctx.AllocateVariable(node, "greeting", "string")
			ctx.WriteLine("%s := \"Hello, \" + %s", output, name)
			return nil
		},
	})

	// Create a node graph with the custom node
	graph := &NodeGraph{
		Package: "testpkg",
		Handlers: []EventHandler{
			{
				Name:  "TestHandler",
				Event: "Test",
				Nodes: []Node{
					{
						ID:   "hello",
						Type: "HelloWorldNode",
						Inputs: map[string]interface{}{
							"name": "param:userName",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "hello"},
				},
				Parameters: []Parameter{
					{Name: "userName", Type: "string"},
				},
			},
		},
	}

	gen := NewCodeGenerator(graph, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !generatorCalled {
		t.Error("Custom node generator was not called")
	}

	// Check that the generated code contains the custom node output
	codeStr := string(code)
	if !strings.Contains(codeStr, "Hello, ") {
		t.Error("Generated code should contain custom node output")
	}
}

func TestGeneratorContext_CustomNodeWithImport(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	// Register a custom node that adds an import
	RegisterNode(NodeDefinition{
		Type:     "UUIDNode",
		Category: "Custom",
		Outputs: []PortDefinition{
			{Name: "uuid", Type: "string"},
		},
		Generator: func(ctx *GeneratorContext, node *Node) error {
			ctx.AddImport("github.com/google/uuid")
			output := ctx.AllocateVariable(node, "uuid", "string")
			ctx.WriteLine("%s := uuid.New().String()", output)
			return nil
		},
	})

	graph := &NodeGraph{
		Package: "testpkg",
		Handlers: []EventHandler{
			{
				Name:  "TestHandler",
				Event: "Test",
				Nodes: []Node{
					{
						ID:   "uuid",
						Type: "UUIDNode",
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "uuid"},
				},
			},
		},
	}

	gen := NewCodeGenerator(graph, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "github.com/google/uuid") {
		t.Error("Generated code should include custom import")
	}
}

func TestGeneratorContext_ResolveInputOptional(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	var resolvedValue string
	RegisterNode(NodeDefinition{
		Type:     "OptionalInputNode",
		Category: "Custom",
		Inputs: []PortDefinition{
			{Name: "optional", Type: "string", Required: false, Default: "default"},
		},
		Generator: func(ctx *GeneratorContext, node *Node) error {
			resolvedValue = ctx.ResolveInputOptional(node, "optional", `"defaultValue"`)
			ctx.WriteLine("// resolved: %s", resolvedValue)
			return nil
		},
	})

	graph := &NodeGraph{
		Package: "testpkg",
		Handlers: []EventHandler{
			{
				Name:  "TestHandler",
				Event: "Test",
				Nodes: []Node{
					{
						ID:     "opt",
						Type:   "OptionalInputNode",
						Inputs: map[string]interface{}{}, // No inputs provided
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "opt"},
				},
			},
		},
	}

	gen := NewCodeGenerator(graph, nil)
	_, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resolvedValue != `"defaultValue"` {
		t.Errorf("ResolveInputOptional returned %v, want \"defaultValue\"", resolvedValue)
	}
}

func TestGeneratorContext_IndentDedent(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	RegisterNode(NodeDefinition{
		Type:     "IndentTestNode",
		Category: "Custom",
		Generator: func(ctx *GeneratorContext, node *Node) error {
			ctx.WriteLine("if true {")
			ctx.Indent()
			ctx.WriteLine("// indented")
			ctx.Dedent()
			ctx.WriteLine("}")
			return nil
		},
	})

	graph := &NodeGraph{
		Package: "testpkg",
		Handlers: []EventHandler{
			{
				Name:  "TestHandler",
				Event: "Test",
				Nodes: []Node{
					{
						ID:   "indent",
						Type: "IndentTestNode",
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "indent"},
				},
			},
		},
	}

	gen := NewCodeGenerator(graph, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "if true {") {
		t.Error("Generated code should contain if statement")
	}
}

func TestGeneratorContext_DebugMode(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	var wasDebugMode bool
	RegisterNode(NodeDefinition{
		Type:     "DebugCheckNode",
		Category: "Custom",
		Generator: func(ctx *GeneratorContext, node *Node) error {
			wasDebugMode = ctx.IsDebugMode()
			ctx.WriteLine("// debug mode: %v", wasDebugMode)
			return nil
		},
	})

	graph := &NodeGraph{
		Package: "testpkg",
		Handlers: []EventHandler{
			{
				Name:  "TestHandler",
				Event: "Test",
				Nodes: []Node{
					{
						ID:   "dbg",
						Type: "DebugCheckNode",
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "dbg"},
				},
			},
		},
	}

	gen := NewCodeGeneratorWithDebug(graph, nil, true)
	_, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !wasDebugMode {
		t.Error("IsDebugMode should return true when debug mode is enabled")
	}
}

// ============================================================================
// IsNodeRegistered Tests
// ============================================================================

func TestIsNodeRegistered(t *testing.T) {
	ClearCustomNodes()
	defer ClearCustomNodes()

	// Built-in should be registered
	if !IsNodeRegistered(NodeCompare) {
		t.Error("NodeCompare should be registered")
	}

	// Unknown should not be registered
	if IsNodeRegistered("UnknownNode") {
		t.Error("UnknownNode should not be registered")
	}

	// Register custom and check
	RegisterNode(NodeDefinition{
		Type:     "NewCustomNode",
		Category: "Custom",
	})
	if !IsNodeRegistered("NewCustomNode") {
		t.Error("NewCustomNode should be registered after RegisterNode")
	}
}
