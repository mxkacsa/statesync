package ast

import (
	"encoding/json"
	"fmt"
)

// EffectType identifies the type of effect
type EffectType string

const (
	EffectTypeSet            EffectType = "Set"            // Set a value
	EffectTypeIncrement      EffectType = "Increment"      // Increment a numeric value
	EffectTypeDecrement      EffectType = "Decrement"      // Decrement a numeric value
	EffectTypeAppend         EffectType = "Append"         // Append to array
	EffectTypeRemove         EffectType = "Remove"         // Remove from array
	EffectTypeClear          EffectType = "Clear"          // Clear array or map
	EffectTypeTransform      EffectType = "Transform"      // Apply transform and save result
	EffectTypeEmit           EffectType = "Emit"           // Emit an event
	EffectTypeSpawn          EffectType = "Spawn"          // Create new entity
	EffectTypeDestroy        EffectType = "Destroy"        // Destroy entity
	EffectTypeIf             EffectType = "If"             // Conditional effect
	EffectTypeSequence       EffectType = "Sequence"       // Execute effects in sequence
	EffectTypeEnableRule     EffectType = "EnableRule"     // Enable a rule by name
	EffectTypeDisableRule    EffectType = "DisableRule"    // Disable a rule by name
	EffectTypeEnableTrigger  EffectType = "EnableTrigger"  // Enable a trigger by rule name
	EffectTypeDisableTrigger EffectType = "DisableTrigger" // Disable a trigger by rule name
	EffectTypeResetTimer     EffectType = "ResetTimer"     // Reset a timer trigger
)

// Effect represents a mutation to apply to the state
type Effect struct {
	Type EffectType `json:"type"`

	// Set/Increment/Decrement fields
	Path  Path        `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"` // Can be literal, path, view reference, or transform

	// Transform effect
	Transform *Transform `json:"transform,omitempty"`

	// Append/Remove fields
	Item  interface{}  `json:"item,omitempty"`  // Item to append
	Index interface{}  `json:"index,omitempty"` // Index to remove at
	Where *WhereClause `json:"where,omitempty"` // Remove where condition

	// Emit fields
	Event   string                 `json:"event,omitempty"`
	To      string                 `json:"to,omitempty"` // "all", "sender", "except:sender", or player ID path
	Payload map[string]interface{} `json:"payload,omitempty"`

	// Spawn fields
	Entity string                 `json:"entity,omitempty"` // Entity type to create
	Fields map[string]interface{} `json:"fields,omitempty"` // Initial field values

	// Conditional effect
	Condition interface{} `json:"condition,omitempty"` // Expression or path
	Then      *Effect     `json:"then,omitempty"`
	Else      *Effect     `json:"else,omitempty"`

	// Sequence effect
	Effects []*Effect `json:"effects,omitempty"`

	// Rule control effects (EnableRule, DisableRule, ResetTimer)
	Rule string `json:"rule,omitempty"` // Target rule name
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
			if len(val) > 5 && val[:5] == "view:" {
				// View reference - doesn't count as state dependency
			}
		case Path:
			paths = append(paths, val)
		}
	}

	collectPaths(e.Value)
	collectPaths(e.Item)
	collectPaths(e.Index)
	collectPaths(e.Condition)

	if e.Transform != nil {
		paths = append(paths, e.Transform.DependsOn()...)
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
		EffectTypeAppend, EffectTypeRemove, EffectTypeClear,
		EffectTypeTransform, EffectTypeEmit, EffectTypeSpawn,
		EffectTypeDestroy, EffectTypeIf, EffectTypeSequence,
		EffectTypeEnableRule, EffectTypeDisableRule,
		EffectTypeEnableTrigger, EffectTypeDisableTrigger, EffectTypeResetTimer:
		// Valid
	case "":
		// Default to Set if path and value are provided
		if e.Path != "" && e.Value != nil {
			e.Type = EffectTypeSet
		} else if e.Transform != nil && e.Path != "" {
			e.Type = EffectTypeTransform
		} else {
			return fmt.Errorf("effect type is required")
		}
	default:
		return fmt.Errorf("unknown effect type: %s", e.Type)
	}

	return nil
}
