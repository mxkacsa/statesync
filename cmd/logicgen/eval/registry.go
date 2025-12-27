package eval

import (
	"fmt"
	"sync"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// PortDefinition defines an input or output port for an operation
type PortDefinition struct {
	Name        string
	Type        string
	Required    bool
	Default     interface{}
	Description string
}

// Category constants for operations
const (
	CategoryCollection  = "collection"
	CategoryAggregation = "aggregation"
	CategoryGeo         = "geo"
	CategoryMath        = "math"
	CategoryString      = "string"
	CategoryTime        = "time"
	CategoryLogic       = "logic"
	CategoryState       = "state"
	CategoryEvent       = "event"
	CategoryControl     = "control"
)

// =============================================================================
// View Operation Registry
// =============================================================================

// ViewOpFunc is the function signature for view operations
type ViewOpFunc func(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error)

// ViewOpDefinition defines a view operation
type ViewOpDefinition struct {
	Name        string
	Category    string
	Description string
	Inputs      []PortDefinition
	Outputs     []PortDefinition
	Func        ViewOpFunc
}

// ViewOpRegistry holds all registered view operations
type ViewOpRegistry struct {
	mu  sync.RWMutex
	ops map[string]*ViewOpDefinition
}

// newViewOpRegistry creates a new view operation registry
func newViewOpRegistry() *ViewOpRegistry {
	return &ViewOpRegistry{
		ops: make(map[string]*ViewOpDefinition),
	}
}

// Register registers a view operation
func (r *ViewOpRegistry) Register(def *ViewOpDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.ops[def.Name]; exists {
		return fmt.Errorf("view operation already registered: %s", def.Name)
	}
	r.ops[def.Name] = def
	return nil
}

// Get returns a view operation by name
func (r *ViewOpRegistry) Get(name string) (*ViewOpDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.ops[name]
	return def, ok
}

// List returns all registered view operations
func (r *ViewOpRegistry) List() []*ViewOpDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]*ViewOpDefinition, 0, len(r.ops))
	for _, def := range r.ops {
		defs = append(defs, def)
	}
	return defs
}

// ListByCategory returns operations in a category
func (r *ViewOpRegistry) ListByCategory(category string) []*ViewOpDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var defs []*ViewOpDefinition
	for _, def := range r.ops {
		if def.Category == category {
			defs = append(defs, def)
		}
	}
	return defs
}

// =============================================================================
// Effect Handler Registry
// =============================================================================

// EffectFunc is the function signature for effect handlers
type EffectFunc func(ee *EffectEvaluator, ctx *Context, effect *ast.Effect, ruleViews map[string]*ast.View) error

// EffectDefinition defines an effect handler
type EffectDefinition struct {
	Name        string
	Category    string
	Description string
	Inputs      []PortDefinition
	Func        EffectFunc
}

// EffectRegistry holds all registered effect handlers
type EffectRegistry struct {
	mu      sync.RWMutex
	effects map[string]*EffectDefinition
}

// newEffectRegistry creates a new effect registry
func newEffectRegistry() *EffectRegistry {
	return &EffectRegistry{
		effects: make(map[string]*EffectDefinition),
	}
}

// Register registers an effect handler
func (r *EffectRegistry) Register(def *EffectDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.effects[def.Name]; exists {
		return fmt.Errorf("effect already registered: %s", def.Name)
	}
	r.effects[def.Name] = def
	return nil
}

// Get returns an effect handler by name
func (r *EffectRegistry) Get(name string) (*EffectDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.effects[name]
	return def, ok
}

// List returns all registered effects
func (r *EffectRegistry) List() []*EffectDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]*EffectDefinition, 0, len(r.effects))
	for _, def := range r.effects {
		defs = append(defs, def)
	}
	return defs
}

// =============================================================================
// Transform Handler Registry
// =============================================================================

// TransformFunc is the function signature for transform handlers
type TransformFunc func(te *TransformEvaluator, ctx *Context, transform *ast.Transform) (interface{}, error)

// TransformDefinition defines a transform handler
type TransformDefinition struct {
	Name        string
	Category    string
	Description string
	Inputs      []PortDefinition
	Outputs     []PortDefinition
	Func        TransformFunc
}

// TransformRegistry holds all registered transform handlers
type TransformRegistry struct {
	mu         sync.RWMutex
	transforms map[string]*TransformDefinition
}

// newTransformRegistry creates a new transform registry
func newTransformRegistry() *TransformRegistry {
	return &TransformRegistry{
		transforms: make(map[string]*TransformDefinition),
	}
}

// Register registers a transform handler
func (r *TransformRegistry) Register(def *TransformDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.transforms[def.Name]; exists {
		return fmt.Errorf("transform already registered: %s", def.Name)
	}
	r.transforms[def.Name] = def
	return nil
}

// Get returns a transform handler by name
func (r *TransformRegistry) Get(name string) (*TransformDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.transforms[name]
	return def, ok
}

// List returns all registered transforms
func (r *TransformRegistry) List() []*TransformDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]*TransformDefinition, 0, len(r.transforms))
	for _, def := range r.transforms {
		defs = append(defs, def)
	}
	return defs
}

// ListByCategory returns transforms in a category
func (r *TransformRegistry) ListByCategory(category string) []*TransformDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var defs []*TransformDefinition
	for _, def := range r.transforms {
		if def.Category == category {
			defs = append(defs, def)
		}
	}
	return defs
}

// =============================================================================
// Global Registries
// =============================================================================

var (
	globalViewOps    = newViewOpRegistry()
	globalEffects    = newEffectRegistry()
	globalTransforms = newTransformRegistry()
)

// RegisterViewOp registers a view operation to the global registry
func RegisterViewOp(def *ViewOpDefinition) error {
	return globalViewOps.Register(def)
}

// MustRegisterViewOp registers a view operation and panics on error.
// Use this in init() functions where errors should be fatal.
func MustRegisterViewOp(def *ViewOpDefinition) {
	if err := globalViewOps.Register(def); err != nil {
		panic(fmt.Sprintf("failed to register view operation %s: %v", def.Name, err))
	}
}

// GetViewOp gets a view operation from the global registry
func GetViewOp(name string) (*ViewOpDefinition, bool) {
	return globalViewOps.Get(name)
}

// ListViewOps lists all view operations
func ListViewOps() []*ViewOpDefinition {
	return globalViewOps.List()
}

// ListViewOpsByCategory lists view operations by category
func ListViewOpsByCategory(category string) []*ViewOpDefinition {
	return globalViewOps.ListByCategory(category)
}

// RegisterEffect registers an effect handler to the global registry
func RegisterEffect(def *EffectDefinition) error {
	return globalEffects.Register(def)
}

// MustRegisterEffect registers an effect handler and panics on error.
// Use this in init() functions where errors should be fatal.
func MustRegisterEffect(def *EffectDefinition) {
	if err := globalEffects.Register(def); err != nil {
		panic(fmt.Sprintf("failed to register effect %s: %v", def.Name, err))
	}
}

// GetEffect gets an effect handler from the global registry
func GetEffect(name string) (*EffectDefinition, bool) {
	return globalEffects.Get(name)
}

// ListEffects lists all effect handlers
func ListEffects() []*EffectDefinition {
	return globalEffects.List()
}

// RegisterTransform registers a transform handler to the global registry
func RegisterTransform(def *TransformDefinition) error {
	return globalTransforms.Register(def)
}

// MustRegisterTransform registers a transform handler and panics on error.
// Use this in init() functions where errors should be fatal.
func MustRegisterTransform(def *TransformDefinition) {
	if err := globalTransforms.Register(def); err != nil {
		panic(fmt.Sprintf("failed to register transform %s: %v", def.Name, err))
	}
}

// GetTransform gets a transform handler from the global registry
func GetTransform(name string) (*TransformDefinition, bool) {
	return globalTransforms.Get(name)
}

// ListTransforms lists all transform handlers
func ListTransforms() []*TransformDefinition {
	return globalTransforms.List()
}

// ListTransformsByCategory lists transform handlers by category
func ListTransformsByCategory(category string) []*TransformDefinition {
	return globalTransforms.ListByCategory(category)
}
