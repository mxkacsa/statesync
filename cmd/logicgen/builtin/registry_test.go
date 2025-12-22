package builtin

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if r.nodes == nil {
		t.Error("nodes map should be initialized")
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	def := &NodeDefinition{
		Name:        "Test.Node",
		Category:    "test",
		Description: "A test node",
		Inputs: []PortDefinition{
			{Name: "input", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "output", Type: "string"},
		},
		Func: func(args map[string]interface{}) (interface{}, error) {
			return args["input"], nil
		},
	}

	err := r.Register(def)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Try to register same node again
	err = r.Register(def)
	if err == nil {
		t.Error("expected error when registering duplicate node")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	def := &NodeDefinition{
		Name:     "Test.Get",
		Category: "test",
		Func:     func(args map[string]interface{}) (interface{}, error) { return nil, nil },
	}

	r.Register(def)

	// Get existing
	found, ok := r.Get("Test.Get")
	if !ok {
		t.Error("expected to find registered node")
	}
	if found.Name != "Test.Get" {
		t.Errorf("expected name 'Test.Get', got %s", found.Name)
	}

	// Get non-existing
	_, ok = r.Get("NonExistent")
	if ok {
		t.Error("expected not to find non-existent node")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	r.Register(&NodeDefinition{
		Name:     "Test.A",
		Category: "cat1",
		Func:     func(args map[string]interface{}) (interface{}, error) { return nil, nil },
	})
	r.Register(&NodeDefinition{
		Name:     "Test.B",
		Category: "cat2",
		Func:     func(args map[string]interface{}) (interface{}, error) { return nil, nil },
	})
	r.Register(&NodeDefinition{
		Name:     "Test.C",
		Category: "cat1",
		Func:     func(args map[string]interface{}) (interface{}, error) { return nil, nil },
	})

	all := r.List()
	if len(all) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(all))
	}
}

func TestRegistry_ListByCategory(t *testing.T) {
	r := NewRegistry()

	r.Register(&NodeDefinition{
		Name:     "Cat1.A",
		Category: "cat1",
		Func:     func(args map[string]interface{}) (interface{}, error) { return nil, nil },
	})
	r.Register(&NodeDefinition{
		Name:     "Cat2.A",
		Category: "cat2",
		Func:     func(args map[string]interface{}) (interface{}, error) { return nil, nil },
	})
	r.Register(&NodeDefinition{
		Name:     "Cat1.B",
		Category: "cat1",
		Func:     func(args map[string]interface{}) (interface{}, error) { return nil, nil },
	})

	cat1 := r.ListByCategory("cat1")
	if len(cat1) != 2 {
		t.Errorf("expected 2 nodes in cat1, got %d", len(cat1))
	}

	cat2 := r.ListByCategory("cat2")
	if len(cat2) != 1 {
		t.Errorf("expected 1 node in cat2, got %d", len(cat2))
	}

	cat3 := r.ListByCategory("cat3")
	if len(cat3) != 0 {
		t.Errorf("expected 0 nodes in cat3, got %d", len(cat3))
	}
}

func TestRegistry_Call(t *testing.T) {
	r := NewRegistry()

	r.Register(&NodeDefinition{
		Name:     "Test.Add",
		Category: "test",
		Inputs: []PortDefinition{
			{Name: "a", Type: "float64", Required: true},
			{Name: "b", Type: "float64", Required: true},
		},
		Func: func(args map[string]interface{}) (interface{}, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			return a + b, nil
		},
	})

	result, err := r.Call("Test.Add", map[string]interface{}{
		"a": 10.0,
		"b": 5.0,
	})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result != 15.0 {
		t.Errorf("expected 15.0, got %v", result)
	}
}

func TestRegistry_Call_UnknownNode(t *testing.T) {
	r := NewRegistry()

	_, err := r.Call("NonExistent", nil)
	if err == nil {
		t.Error("expected error for unknown node")
	}
}

func TestRegistry_Call_MissingRequiredInput(t *testing.T) {
	r := NewRegistry()

	r.Register(&NodeDefinition{
		Name:     "Test.Required",
		Category: "test",
		Inputs: []PortDefinition{
			{Name: "required_input", Type: "string", Required: true},
		},
		Func: func(args map[string]interface{}) (interface{}, error) {
			return args["required_input"], nil
		},
	})

	_, err := r.Call("Test.Required", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing required input")
	}
}

func TestRegistry_Call_WithDefault(t *testing.T) {
	r := NewRegistry()

	r.Register(&NodeDefinition{
		Name:     "Test.Default",
		Category: "test",
		Inputs: []PortDefinition{
			{Name: "value", Type: "string", Required: true, Default: "default_value"},
		},
		Func: func(args map[string]interface{}) (interface{}, error) {
			return args["value"], nil
		},
	})

	result, err := r.Call("Test.Default", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result != "default_value" {
		t.Errorf("expected 'default_value', got %v", result)
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Global registry should have built-in nodes from init()
	gpsNodes := ListByCategory(CategoryGPS)
	if len(gpsNodes) == 0 {
		t.Error("expected GPS nodes in global registry")
	}

	mathNodes := ListByCategory(CategoryMath)
	if len(mathNodes) == 0 {
		t.Error("expected Math nodes in global registry")
	}

	timeNodes := ListByCategory(CategoryTime)
	if len(timeNodes) == 0 {
		t.Error("expected Time nodes in global registry")
	}
}

func TestCategories(t *testing.T) {
	// Verify category constants
	categories := []string{
		CategoryGPS,
		CategoryMath,
		CategoryString,
		CategoryTime,
		CategoryLogic,
		CategoryArray,
	}

	for _, cat := range categories {
		if cat == "" {
			t.Error("category constant should not be empty")
		}
	}
}
