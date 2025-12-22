package ast

import (
	"encoding/json"
	"fmt"
)

// SelectorType identifies the type of selector
type SelectorType string

const (
	SelectorTypeAll      SelectorType = "All"
	SelectorTypeFilter   SelectorType = "Filter"
	SelectorTypeSingle   SelectorType = "Single"
	SelectorTypeRelated  SelectorType = "Related"
	SelectorTypeNearest  SelectorType = "Nearest"
	SelectorTypeFarthest SelectorType = "Farthest"
)

// Selector represents how to select entities for a rule
type Selector struct {
	Type   SelectorType `json:"type"`
	Entity string       `json:"entity"` // Entity type name (e.g., "Players", "Drones")

	// Filter fields
	Where *WhereClause `json:"where,omitempty"`

	// Single fields
	ID  Path `json:"id,omitempty"`  // Path to entity ID
	Key Path `json:"key,omitempty"` // Key field name (default: "ID")

	// Related fields
	Relation string `json:"relation,omitempty"` // Relation name
	From     Path   `json:"from,omitempty"`     // Source entity path

	// Nearest/Farthest fields
	Position    Path    `json:"position,omitempty"`    // GPS position field
	Origin      Path    `json:"origin,omitempty"`      // Origin position to measure from
	Limit       int     `json:"limit,omitempty"`       // Max entities to select
	MaxDistance float64 `json:"maxDistance,omitempty"` // Max distance in meters
	MinDistance float64 `json:"minDistance,omitempty"` // Min distance in meters
}

// DependsOn returns the state paths this selector depends on
func (s *Selector) DependsOn() []Path {
	var paths []Path

	// Entity collection path
	if s.Entity != "" {
		paths = append(paths, Path("$."+s.Entity))
	}

	// ID path for Single selector
	if s.ID != "" {
		paths = append(paths, s.ID)
	}

	// Related paths
	if s.From != "" {
		paths = append(paths, s.From)
	}

	// GPS paths
	if s.Position != "" {
		paths = append(paths, s.Position)
	}
	if s.Origin != "" {
		paths = append(paths, s.Origin)
	}

	return paths
}

// UnmarshalJSON implements custom JSON unmarshaling for Selector
func (s *Selector) UnmarshalJSON(data []byte) error {
	type selectorAlias Selector
	var alias selectorAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return fmt.Errorf("invalid selector: %w", err)
	}
	*s = Selector(alias)

	// Validate selector type
	switch s.Type {
	case SelectorTypeAll, SelectorTypeFilter, SelectorTypeSingle,
		SelectorTypeRelated, SelectorTypeNearest, SelectorTypeFarthest:
		// Valid
	default:
		return fmt.Errorf("unknown selector type: %s", s.Type)
	}

	return nil
}

// WhereClause can also be expressed as a map for convenience
func (w *WhereClause) UnmarshalJSON(data []byte) error {
	// Try standard format first
	type whereAlias WhereClause
	var alias whereAlias
	if err := json.Unmarshal(data, &alias); err == nil && alias.Field != "" {
		*w = WhereClause(alias)
		return nil
	}

	// Try map format: {"Status": "Moving"} or {"Score": {">": 100}}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("invalid where clause: %w", err)
	}

	for field, value := range m {
		w.Field = field
		// Check if value is an operator map
		if opMap, ok := value.(map[string]interface{}); ok {
			for op, v := range opMap {
				w.Op = op
				w.Value = v
				return nil
			}
		}
		// Simple equality
		w.Op = "=="
		w.Value = value
		return nil
	}

	return fmt.Errorf("empty where clause")
}
