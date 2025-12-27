// Package builtin provides built-in nodes and functions for the LogicGen v2 rule engine.
package builtin

import (
	"fmt"
	"sync"
)

// NodeFunc is a function that can be called by the rule engine
type NodeFunc func(args map[string]interface{}) (interface{}, error)

// NodeDefinition defines a built-in node
type NodeDefinition struct {
	Name        string
	Category    string
	Description string
	Inputs      []PortDefinition
	Outputs     []PortDefinition
	Func        NodeFunc
}

// PortDefinition defines an input or output port
type PortDefinition struct {
	Name        string
	Type        string
	Required    bool
	Default     interface{}
	Description string
}

// Registry holds all built-in node definitions
type Registry struct {
	mu    sync.RWMutex
	nodes map[string]*NodeDefinition
}

// globalRegistry is the default global registry
var globalRegistry = NewRegistry()

// NewRegistry creates a new registry
func NewRegistry() *Registry {
	r := &Registry{
		nodes: make(map[string]*NodeDefinition),
	}
	return r
}

// Register registers a new node definition
func (r *Registry) Register(def *NodeDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[def.Name]; exists {
		return fmt.Errorf("node already registered: %s", def.Name)
	}

	r.nodes[def.Name] = def
	return nil
}

// Get returns a node definition by name
func (r *Registry) Get(name string) (*NodeDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.nodes[name]
	return def, ok
}

// List returns all registered node definitions
func (r *Registry) List() []*NodeDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]*NodeDefinition, 0, len(r.nodes))
	for _, def := range r.nodes {
		defs = append(defs, def)
	}
	return defs
}

// ListByCategory returns all nodes in a category
func (r *Registry) ListByCategory(category string) []*NodeDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var defs []*NodeDefinition
	for _, def := range r.nodes {
		if def.Category == category {
			defs = append(defs, def)
		}
	}
	return defs
}

// Call calls a node function with the given arguments
func (r *Registry) Call(name string, args map[string]interface{}) (interface{}, error) {
	def, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown node: %s", name)
	}

	// Validate required inputs
	for _, input := range def.Inputs {
		if input.Required {
			if _, ok := args[input.Name]; !ok {
				if input.Default != nil {
					args[input.Name] = input.Default
				} else {
					return nil, fmt.Errorf("missing required input: %s", input.Name)
				}
			}
		}
	}

	return def.Func(args)
}

// Global registry functions

// Register registers a node to the global registry
func Register(def *NodeDefinition) error {
	return globalRegistry.Register(def)
}

// Get gets a node from the global registry
func Get(name string) (*NodeDefinition, bool) {
	return globalRegistry.Get(name)
}

// List lists all nodes in the global registry
func List() []*NodeDefinition {
	return globalRegistry.List()
}

// ListByCategory lists all nodes in a category from the global registry
func ListByCategory(category string) []*NodeDefinition {
	return globalRegistry.ListByCategory(category)
}

// Call calls a node from the global registry
func Call(name string, args map[string]interface{}) (interface{}, error) {
	return globalRegistry.Call(name, args)
}

// Categories for built-in nodes
const (
	CategoryGPS    = "gps"
	CategoryMath   = "math"
	CategoryString = "string"
	CategoryTime   = "time"
	CategoryLogic  = "logic"
	CategoryArray  = "array"
)

// =============================================================================
// Editor API - JSON-serializable types for the visual editor
// =============================================================================

// PortSchema is a JSON-serializable port definition
type PortSchema struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

// NodeSchema is a JSON-serializable node definition for the editor
type NodeSchema struct {
	Name        string       `json:"name"`
	Category    string       `json:"category"`
	Description string       `json:"description,omitempty"`
	Inputs      []PortSchema `json:"inputs"`
	Outputs     []PortSchema `json:"outputs"`
}

// CategorySchema groups nodes by category
type CategorySchema struct {
	Name  string       `json:"name"`
	Nodes []NodeSchema `json:"nodes"`
}

// EditorSchema is the complete schema for the visual editor
type EditorSchema struct {
	Version    string           `json:"version"`
	Categories []CategorySchema `json:"categories"`
}

// ToSchema converts a NodeDefinition to a JSON-serializable NodeSchema
func (def *NodeDefinition) ToSchema() NodeSchema {
	inputs := make([]PortSchema, len(def.Inputs))
	for i, p := range def.Inputs {
		inputs[i] = PortSchema{
			Name:        p.Name,
			Type:        p.Type,
			Required:    p.Required,
			Default:     p.Default,
			Description: p.Description,
		}
	}

	outputs := make([]PortSchema, len(def.Outputs))
	for i, p := range def.Outputs {
		outputs[i] = PortSchema{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
		}
	}

	return NodeSchema{
		Name:        def.Name,
		Category:    def.Category,
		Description: def.Description,
		Inputs:      inputs,
		Outputs:     outputs,
	}
}

// Categories returns all unique categories in the registry
func (r *Registry) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	var categories []string
	for _, def := range r.nodes {
		if !seen[def.Category] {
			seen[def.Category] = true
			categories = append(categories, def.Category)
		}
	}
	return categories
}

// GetEditorSchema returns the complete schema for the visual editor
func (r *Registry) GetEditorSchema() EditorSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Group nodes by category
	categoryMap := make(map[string][]NodeSchema)
	for _, def := range r.nodes {
		categoryMap[def.Category] = append(categoryMap[def.Category], def.ToSchema())
	}

	// Build category list
	var categories []CategorySchema
	for name, nodes := range categoryMap {
		categories = append(categories, CategorySchema{
			Name:  name,
			Nodes: nodes,
		})
	}

	return EditorSchema{
		Version:    "1.0",
		Categories: categories,
	}
}

// Global registry functions for editor

// Categories returns all categories from the global registry
func Categories() []string {
	return globalRegistry.Categories()
}

// GetEditorSchema returns the editor schema from the global registry
func GetEditorSchema() EditorSchema {
	return globalRegistry.GetEditorSchema()
}
