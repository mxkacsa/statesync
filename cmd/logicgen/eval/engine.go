package eval

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// RuleController provides methods to control rules at runtime
type RuleController interface {
	EnableRule(name string) bool
	DisableRule(name string) bool
	EnableTrigger(ruleName string) bool
	DisableTrigger(ruleName string) bool
	ResetTimer(ruleName string)
}

// Engine evaluates and executes rules.
// The engine follows the new architecture:
// - Views are pure queries (no side effects)
// - Effects are batch operations (engine handles iteration)
// - No per-entity loops in rule definition
type Engine struct {
	rules    []*ast.Rule
	state    interface{}
	tick     uint64
	lastTick time.Time
	tickRate time.Duration

	// Global views that can be referenced by any rule
	globalViews map[string]*ast.View

	// Evaluators
	triggerEval *TriggerEvaluator
	viewEval    *ViewEvaluator
	effectEval  *EffectEvaluator
}

// NewEngine creates a new rule engine
func NewEngine(state interface{}, rules []*ast.Rule) *Engine {
	e := &Engine{
		rules:       rules,
		state:       state,
		tickRate:    100 * time.Millisecond, // Default 100ms tick
		lastTick:    time.Now(),
		globalViews: make(map[string]*ast.View),
	}

	e.triggerEval = NewTriggerEvaluator()
	e.viewEval = NewViewEvaluator()
	e.effectEval = NewEffectEvaluator()
	e.effectEval.SetRuleController(e) // Allow effects to enable/disable rules

	// Set rule names on triggers for stable timer keys
	for _, rule := range e.rules {
		if rule.Trigger != nil {
			rule.Trigger.RuleName = rule.Name
		}
	}

	// Sort rules by priority (descending - higher priority first)
	sort.Slice(e.rules, func(i, j int) bool {
		return e.rules[i].GetPriority() > e.rules[j].GetPriority()
	})

	return e
}

// RegisterGlobalView registers a view that can be referenced by any rule
func (e *Engine) RegisterGlobalView(name string, view *ast.View) {
	e.globalViews[name] = view
}

// EnableRule enables a rule by name, returns true if found
func (e *Engine) EnableRule(name string) bool {
	rule := e.GetRule(name)
	if rule == nil {
		return false
	}
	enabled := true
	rule.Enabled = &enabled
	return true
}

// DisableRule disables a rule by name, returns true if found
func (e *Engine) DisableRule(name string) bool {
	rule := e.GetRule(name)
	if rule == nil {
		return false
	}
	enabled := false
	rule.Enabled = &enabled
	return true
}

// ResetTimer resets the timer for a rule
func (e *Engine) ResetTimer(ruleName string) {
	e.triggerEval.ResetTimer(ruleName)
}

// EnableTrigger enables the trigger of a rule by name, returns true if found
func (e *Engine) EnableTrigger(ruleName string) bool {
	rule := e.GetRule(ruleName)
	if rule == nil || rule.Trigger == nil {
		return false
	}
	rule.Trigger.SetEnabled(true)
	return true
}

// DisableTrigger disables the trigger of a rule by name, returns true if found
func (e *Engine) DisableTrigger(ruleName string) bool {
	rule := e.GetRule(ruleName)
	if rule == nil || rule.Trigger == nil {
		return false
	}
	rule.Trigger.SetEnabled(false)
	return true
}

// SetTickRate sets the tick rate
func (e *Engine) SetTickRate(rate time.Duration) {
	e.tickRate = rate
}

// Tick processes one game tick
func (e *Engine) Tick(ctx context.Context) error {
	now := time.Now()
	dt := now.Sub(e.lastTick)
	e.lastTick = now
	e.tick++

	evalCtx := NewContext(e.state, dt, e.tick)

	return e.processTick(ctx, evalCtx)
}

// TickWithDelta processes one tick with explicit delta time (for deterministic replay)
func (e *Engine) TickWithDelta(ctx context.Context, dt time.Duration) error {
	e.tick++
	evalCtx := NewContext(e.state, dt, e.tick)
	return e.processTick(ctx, evalCtx)
}

// HandleEvent processes an incoming event
func (e *Engine) HandleEvent(ctx context.Context, event *ast.Event) error {
	dt := time.Since(e.lastTick)
	evalCtx := NewContext(e.state, dt, e.tick).WithEvent(event)

	// Process event-triggered rules
	for _, rule := range e.rules {
		if !rule.IsEnabled() {
			continue
		}

		// Check if this rule triggers on this event
		if rule.Trigger == nil || rule.Trigger.Type != ast.TriggerTypeOnEvent {
			continue
		}
		if rule.Trigger.Event != event.Name {
			continue
		}

		// Execute the rule
		if err := e.executeRule(ctx, rule, evalCtx); err != nil {
			return fmt.Errorf("rule %s failed: %w", rule.Name, err)
		}
	}

	return nil
}

// processTick processes all tick-based rules
func (e *Engine) processTick(ctx context.Context, evalCtx *Context) error {
	for _, rule := range e.rules {
		// Handle nil context gracefully
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		if !rule.IsEnabled() {
			continue
		}

		// Evaluate trigger
		shouldFire, err := e.triggerEval.Evaluate(evalCtx, rule.Trigger)
		if err != nil {
			return fmt.Errorf("rule %s trigger error: %w", rule.Name, err)
		}
		if !shouldFire {
			continue
		}

		// Execute the rule
		if err := e.executeRule(ctx, rule, evalCtx); err != nil {
			return fmt.Errorf("rule %s failed: %w", rule.Name, err)
		}
	}

	return nil
}

// executeRule executes a single rule.
// The new architecture:
// 1. Evaluate all views (pure queries, stored in context)
// 2. Apply all effects (batch operations, engine handles iteration)
func (e *Engine) executeRule(ctx context.Context, rule *ast.Rule, evalCtx *Context) error {
	// Clear views from previous executions
	evalCtx.Views = make(map[string]interface{})

	// Merge global views with rule views
	allViews := make(map[string]*ast.View)
	for name, view := range e.globalViews {
		allViews[name] = view
	}
	for name, view := range rule.Views {
		allViews[name] = view
	}

	// Step 1: Evaluate all named views (pure queries, no side effects)
	// Views are computed once and cached in context for use by effects
	for name, view := range allViews {
		result, err := e.viewEval.Evaluate(evalCtx, view, nil)
		if err != nil {
			return fmt.Errorf("view %s error: %w", name, err)
		}
		evalCtx.Views[name] = result
	}

	// Step 2: Apply effects (batch operations)
	// Effects reference views for their targets
	// The engine handles per-entity iteration inside each effect
	for _, effect := range rule.Effects {
		if err := e.effectEval.Apply(evalCtx, effect, allViews); err != nil {
			return fmt.Errorf("effect error: %w", err)
		}
	}

	return nil
}

// GetState returns the current state
func (e *Engine) GetState() interface{} {
	return e.state
}

// GetTick returns the current tick number
func (e *Engine) GetTick() uint64 {
	return e.tick
}

// AddRule adds a rule to the engine
func (e *Engine) AddRule(rule *ast.Rule) {
	if rule.Trigger != nil {
		rule.Trigger.RuleName = rule.Name
	}
	e.rules = append(e.rules, rule)
	// Re-sort rules
	sort.Slice(e.rules, func(i, j int) bool {
		return e.rules[i].GetPriority() > e.rules[j].GetPriority()
	})
}

// RemoveRule removes a rule by name
func (e *Engine) RemoveRule(name string) bool {
	for i, rule := range e.rules {
		if rule.Name == name {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return true
		}
	}
	return false
}

// GetRule returns a rule by name
func (e *Engine) GetRule(name string) *ast.Rule {
	for _, rule := range e.rules {
		if rule.Name == name {
			return rule
		}
	}
	return nil
}
