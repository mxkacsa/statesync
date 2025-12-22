package ast

import (
	"encoding/json"
	"fmt"
)

// TriggerType identifies the type of trigger
type TriggerType string

const (
	TriggerTypeOnTick    TriggerType = "OnTick"
	TriggerTypeOnEvent   TriggerType = "OnEvent"
	TriggerTypeOnChange  TriggerType = "OnChange"
	TriggerTypeDistance  TriggerType = "Distance"
	TriggerTypeTimer     TriggerType = "Timer"
	TriggerTypeCondition TriggerType = "Condition"
	TriggerTypeCron      TriggerType = "Cron"     // Cron expression based trigger
	TriggerTypeWait      TriggerType = "Wait"     // One-shot delay trigger
	TriggerTypeSchedule  TriggerType = "Schedule" // Scheduled time trigger (e.g., every day at 14:00)
)

// Trigger represents when a rule should fire
type Trigger struct {
	Type    TriggerType `json:"type"`
	Enabled *bool       `json:"enabled,omitempty"` // Default enabled state (nil = enabled)

	// OnTick fields
	Interval int `json:"interval,omitempty"` // ms, 0 = every tick

	// OnEvent fields
	Event  string   `json:"event,omitempty"`
	Params []string `json:"params,omitempty"`

	// OnChange fields
	Watch []Path `json:"watch,omitempty"`

	// Distance fields
	From     Path    `json:"from,omitempty"`
	To       Path    `json:"to,omitempty"`
	Operator string  `json:"operator,omitempty"` // "<=", "<", ">=", ">", "=="
	Value    float64 `json:"value,omitempty"`    // distance value
	Unit     string  `json:"unit,omitempty"`     // "meters", "kilometers"

	// Timer fields
	Duration   int  `json:"duration,omitempty"` // ms
	Repeat     bool `json:"repeat,omitempty"`
	StartDelay int  `json:"startDelay,omitempty"` // ms

	// Cron fields
	Cron string `json:"cron,omitempty"` // Cron expression (e.g., "*/5 * * * *" for every 5 minutes)

	// Schedule fields
	At       string `json:"at,omitempty"`       // Time of day (e.g., "14:30")
	Every    string `json:"every,omitempty"`    // Interval (e.g., "5m", "1h", "24h")
	Weekdays []int  `json:"weekdays,omitempty"` // 0=Sunday, 1=Monday, etc.

	// Condition fields
	Condition *Expression `json:"condition,omitempty"`

	// Rule name for stable timer keys (set by engine)
	RuleName string `json:"-"`
}

// IsEnabled returns whether the trigger is enabled (default true)
func (t *Trigger) IsEnabled() bool {
	if t.Enabled == nil {
		return true
	}
	return *t.Enabled
}

// SetEnabled sets the enabled state of the trigger
func (t *Trigger) SetEnabled(enabled bool) {
	t.Enabled = &enabled
}

// DependsOn returns the state paths this trigger depends on
func (t *Trigger) DependsOn() []Path {
	var paths []Path

	switch t.Type {
	case TriggerTypeOnChange:
		paths = append(paths, t.Watch...)
	case TriggerTypeDistance:
		if t.From != "" {
			paths = append(paths, t.From)
		}
		if t.To != "" {
			paths = append(paths, t.To)
		}
	case TriggerTypeCondition:
		paths = append(paths, extractPathsFromExpression(t.Condition)...)
	}

	return paths
}

// UnmarshalJSON implements custom JSON unmarshaling for Trigger
func (t *Trigger) UnmarshalJSON(data []byte) error {
	type triggerAlias Trigger
	var alias triggerAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return fmt.Errorf("invalid trigger: %w", err)
	}
	*t = Trigger(alias)

	// Validate trigger type
	switch t.Type {
	case TriggerTypeOnTick, TriggerTypeOnEvent, TriggerTypeOnChange,
		TriggerTypeDistance, TriggerTypeTimer, TriggerTypeCondition,
		TriggerTypeCron, TriggerTypeWait, TriggerTypeSchedule:
		// Valid
	default:
		return fmt.Errorf("unknown trigger type: %s", t.Type)
	}

	return nil
}

// extractPathsFromExpression extracts all paths from an expression
func extractPathsFromExpression(e *Expression) []Path {
	if e == nil {
		return nil
	}

	var paths []Path

	// Check Left
	if str, ok := e.Left.(string); ok && len(str) > 0 && str[0] == '$' {
		paths = append(paths, Path(str))
	}

	// Check Right
	if str, ok := e.Right.(string); ok && len(str) > 0 && str[0] == '$' {
		paths = append(paths, Path(str))
	}

	// Check And/Or clauses
	for _, sub := range e.And {
		paths = append(paths, extractPathsFromExpression(&sub)...)
	}
	for _, sub := range e.Or {
		paths = append(paths, extractPathsFromExpression(&sub)...)
	}
	if e.Not != nil {
		paths = append(paths, extractPathsFromExpression(e.Not)...)
	}

	return paths
}
