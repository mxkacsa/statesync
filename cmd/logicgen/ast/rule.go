package ast

// RuleSet represents a complete set of rules for a game
type RuleSet struct {
	Version string  `json:"version"`
	Package string  `json:"package"`
	Rules   []*Rule `json:"rules"`

	// Named views that can be referenced by rules
	Views map[string]*View `json:"views,omitempty"`
}

// Rule represents a single game logic rule.
// Rules follow the Trigger → Views → Effects pattern:
// - Trigger: When does this rule fire?
// - Views: What data do we need? (pure queries, no side effects)
// - Effects: What batch operations to perform? (engine handles iteration)
type Rule struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"` // Pointer to distinguish unset from false

	// Trigger defines when this rule fires
	Trigger *Trigger `json:"trigger"`

	// Views define named queries for this rule (pure, no side effects)
	// Views are computed before effects and can be referenced by effects
	Views map[string]*View `json:"views,omitempty"`

	// Effects define batch operations to perform
	// Effects reference views for their targets
	// Engine handles the per-entity iteration internally
	Effects []*Effect `json:"effects"`
}

// IsEnabled returns whether the rule is enabled (default true)
func (r *Rule) IsEnabled() bool {
	if r.Enabled == nil {
		return true
	}
	return *r.Enabled
}

// GetPriority returns the rule priority (default 0)
func (r *Rule) GetPriority() int {
	return r.Priority
}

// DependsOn returns all state paths this rule depends on for change detection
func (r *Rule) DependsOn() []Path {
	var paths []Path

	// Collect from trigger
	if r.Trigger != nil {
		paths = append(paths, r.Trigger.DependsOn()...)
	}

	// Collect from views
	for _, v := range r.Views {
		paths = append(paths, v.DependsOn()...)
	}

	// Collect from effects
	for _, e := range r.Effects {
		paths = append(paths, e.DependsOn()...)
	}

	return paths
}

// Modifies returns all state paths this rule modifies
func (r *Rule) Modifies() []Path {
	var paths []Path
	for _, e := range r.Effects {
		paths = append(paths, e.Modifies()...)
	}
	return paths
}

// GetView returns a view by name, checking both rule-level and effect-level views
func (r *Rule) GetView(name string) *View {
	if v, ok := r.Views[name]; ok {
		return v
	}
	return nil
}

// Validate checks if the rule is valid
func (r *Rule) Validate() error {
	if r.Name == "" {
		return &ValidationError{Field: "name", Message: "rule name is required"}
	}
	if r.Trigger == nil {
		return &ValidationError{Field: "trigger", Message: "trigger is required"}
	}
	if len(r.Effects) == 0 {
		return &ValidationError{Field: "effects", Message: "at least one effect is required"}
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
