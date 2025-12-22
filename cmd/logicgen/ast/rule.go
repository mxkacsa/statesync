package ast

// RuleSet represents a complete set of rules for a game
type RuleSet struct {
	Version string   `json:"version"`
	Package string   `json:"package"`
	Imports []string `json:"imports,omitempty"`
	Rules   []*Rule  `json:"rules"`
}

// Rule represents a single game logic rule
type Rule struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Priority    int              `json:"priority,omitempty"`
	Enabled     *bool            `json:"enabled,omitempty"` // Pointer to distinguish unset from false
	Trigger     *Trigger         `json:"trigger"`
	Selector    *Selector        `json:"selector,omitempty"`
	Views       map[string]*View `json:"views,omitempty"`
	Effects     []*Effect        `json:"effects"`
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

	// Collect from selector
	if r.Selector != nil {
		paths = append(paths, r.Selector.DependsOn()...)
	}

	// Collect from views
	for _, v := range r.Views {
		paths = append(paths, v.DependsOn()...)
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
