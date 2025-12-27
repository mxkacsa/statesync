package ast

import (
	"encoding/json"
	"fmt"
)

// Effect represents a declarative batch operation on entities.
// Effects are not per-entity loops - the engine handles iteration.
//
// Key principles:
// - Targets come from a View reference
// - `self` only exists at engine runtime (symbol bound by engine)
// - Operations are declared, not executed in the node editor
// - Engine decides execution order and optimization
type Effect struct {
	Type EffectType `json:"type"`

	// Targets specifies which entities to affect
	// Can be a view name (string) or inline view definition
	Targets interface{} `json:"targets,omitempty"`

	// Set/Increment/Decrement/Transform - path on each target entity
	// Uses `self.` prefix which engine binds at runtime
	Path  Path        `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"` // Literal, param ref, or expression

	// SetFromView - special effect for computed values per entity
	// value can reference other views with params bound to self
	ValueExpression *ValueExpression `json:"valueExpression,omitempty"`

	// Emit effect
	Event   string                 `json:"event,omitempty"`
	To      string                 `json:"to,omitempty"` // "all", "targets", "except:targets"
	Payload map[string]interface{} `json:"payload,omitempty"`

	// Spawn effect
	Entity string                 `json:"entity,omitempty"` // Entity type to create
	Fields map[string]interface{} `json:"fields,omitempty"` // Initial field values

	// Destroy effect - destroys target entities
	// Uses Targets field

	// Conditional effect (batch condition, not per-entity)
	Condition interface{} `json:"condition,omitempty"` // Expression evaluated once
	Then      *Effect     `json:"then,omitempty"`
	Else      *Effect     `json:"else,omitempty"`

	// Sequence effect
	Effects []*Effect `json:"effects,omitempty"`

	// Rule control effects
	Rule string `json:"rule,omitempty"` // Target rule name

	// Filter control effects
	Filter       string                 `json:"filter,omitempty"`       // Target filter name (for Enable/Disable)
	ViewerID     interface{}            `json:"viewerID,omitempty"`     // Who to add filter for
	FilterID     interface{}            `json:"filterID,omitempty"`     // Unique filter instance ID
	FilterName   string                 `json:"filterName,omitempty"`   // Filter type name
	FilterParams map[string]interface{} `json:"filterParams,omitempty"` // Filter parameters
}

// EffectType identifies the type of effect
type EffectType string

const (
	// Data mutation effects (batch operations on target entities)
	EffectTypeSet       EffectType = "Set"       // Set a field on all targets
	EffectTypeIncrement EffectType = "Increment" // Increment a field on all targets
	EffectTypeDecrement EffectType = "Decrement" // Decrement a field on all targets
	EffectTypeTransform EffectType = "Transform" // Apply transform to field on all targets

	// Batch computed value effect
	EffectTypeSetFromView EffectType = "SetFromView" // Set field from parameterized view per entity

	// Entity lifecycle effects
	EffectTypeSpawn   EffectType = "Spawn"   // Create new entity
	EffectTypeDestroy EffectType = "Destroy" // Destroy target entities

	// Event effects
	EffectTypeEmit EffectType = "Emit" // Emit event to targets

	// Control flow (batch, not per-entity)
	EffectTypeIf       EffectType = "If"       // Conditional effect (single evaluation)
	EffectTypeSequence EffectType = "Sequence" // Execute effects in order

	// Rule control effects
	EffectTypeEnableRule     EffectType = "EnableRule"     // Enable a rule by name
	EffectTypeDisableRule    EffectType = "DisableRule"    // Disable a rule by name
	EffectTypeEnableTrigger  EffectType = "EnableTrigger"  // Enable trigger of a rule
	EffectTypeDisableTrigger EffectType = "DisableTrigger" // Disable trigger of a rule
	EffectTypeResetTimer     EffectType = "ResetTimer"     // Reset timer for a rule

	// Filter control effects
	EffectTypeAddFilter     EffectType = "AddFilter"     // Add filter for a viewer
	EffectTypeRemoveFilter  EffectType = "RemoveFilter"  // Remove filter from a viewer
	EffectTypeEnableFilter  EffectType = "EnableFilter"  // Enable a filter by name
	EffectTypeDisableFilter EffectType = "DisableFilter" // Disable a filter by name
)

// ValueExpression defines a computed value that can reference views with self-bound params
type ValueExpression struct {
	// Type of expression
	Type string `json:"type"` // "viewResult", "distance", "field", "literal", "transform"

	// For viewResult type - call a view with params bound to current entity
	View       string                 `json:"view,omitempty"`       // View name to call
	ViewParams map[string]interface{} `json:"viewParams,omitempty"` // Params, can use "self.field"

	// For field access - get field from view result or self
	Field Path `json:"field,omitempty"` // Field to extract

	// For distance type
	From interface{} `json:"from,omitempty"` // Can use "self.position"
	To   interface{} `json:"to,omitempty"`   // Can use view result

	// For literal type
	Literal interface{} `json:"literal,omitempty"`

	// For transform type
	Transform *Transform `json:"transform,omitempty"`
}

// Modifies returns the state paths this effect modifies
func (e *Effect) Modifies() []Path {
	var paths []Path

	if e.Path != "" {
		paths = append(paths, e.Path)
	}

	// Collect from nested effects
	if e.Then != nil {
		paths = append(paths, e.Then.Modifies()...)
	}
	if e.Else != nil {
		paths = append(paths, e.Else.Modifies()...)
	}
	for _, sub := range e.Effects {
		paths = append(paths, sub.Modifies()...)
	}

	return paths
}

// DependsOn returns the state paths this effect depends on
func (e *Effect) DependsOn() []Path {
	var paths []Path

	collectPaths := func(v interface{}) {
		if v == nil {
			return
		}
		switch val := v.(type) {
		case string:
			if len(val) > 0 && val[0] == '$' {
				paths = append(paths, Path(val))
			}
		case Path:
			paths = append(paths, val)
		}
	}

	collectPaths(e.Value)
	collectPaths(e.Condition)

	// Collect from view expression
	if e.ValueExpression != nil {
		collectPaths(e.ValueExpression.From)
		collectPaths(e.ValueExpression.To)
		if e.ValueExpression.Field != "" {
			paths = append(paths, e.ValueExpression.Field)
		}
	}

	for _, v := range e.Payload {
		collectPaths(v)
	}
	for _, v := range e.Fields {
		collectPaths(v)
	}

	// Collect from nested effects
	if e.Then != nil {
		paths = append(paths, e.Then.DependsOn()...)
	}
	if e.Else != nil {
		paths = append(paths, e.Else.DependsOn()...)
	}
	for _, sub := range e.Effects {
		paths = append(paths, sub.DependsOn()...)
	}

	return paths
}

// GetTargetsView returns the targets as a view name if it's a string reference
func (e *Effect) GetTargetsView() (string, bool) {
	if str, ok := e.Targets.(string); ok {
		return str, true
	}
	return "", false
}

// HasTargets returns true if this effect has targets specified
func (e *Effect) HasTargets() bool {
	return e.Targets != nil
}

// IsBatchEffect returns true if this effect operates on multiple entities
func (e *Effect) IsBatchEffect() bool {
	switch e.Type {
	case EffectTypeSet, EffectTypeIncrement, EffectTypeDecrement,
		EffectTypeTransform, EffectTypeSetFromView, EffectTypeDestroy, EffectTypeEmit:
		return e.HasTargets()
	default:
		return false
	}
}

// UnmarshalJSON implements custom JSON unmarshaling for Effect
func (e *Effect) UnmarshalJSON(data []byte) error {
	type effectAlias Effect
	var alias effectAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return fmt.Errorf("invalid effect: %w", err)
	}
	*e = Effect(alias)

	// Validate effect type
	switch e.Type {
	case EffectTypeSet, EffectTypeIncrement, EffectTypeDecrement,
		EffectTypeTransform, EffectTypeSetFromView,
		EffectTypeSpawn, EffectTypeDestroy, EffectTypeEmit,
		EffectTypeIf, EffectTypeSequence,
		EffectTypeEnableRule, EffectTypeDisableRule,
		EffectTypeEnableTrigger, EffectTypeDisableTrigger, EffectTypeResetTimer,
		EffectTypeAddFilter, EffectTypeRemoveFilter,
		EffectTypeEnableFilter, EffectTypeDisableFilter:
		// Valid
	case "":
		// Default to Set if path and value are provided
		if e.Path != "" && e.Value != nil {
			e.Type = EffectTypeSet
		} else if e.ValueExpression != nil && e.Path != "" {
			e.Type = EffectTypeSetFromView
		} else {
			return fmt.Errorf("effect type is required")
		}
	default:
		return fmt.Errorf("unknown effect type: %s", e.Type)
	}

	return nil
}
