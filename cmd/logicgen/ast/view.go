package ast

import (
	"encoding/json"
	"fmt"
)

// ViewType identifies the type of view
type ViewType string

const (
	ViewTypeField    ViewType = "Field"    // Simple field access
	ViewTypeMax      ViewType = "Max"      // Maximum value
	ViewTypeMin      ViewType = "Min"      // Minimum value
	ViewTypeSum      ViewType = "Sum"      // Sum of values
	ViewTypeCount    ViewType = "Count"    // Count of entities
	ViewTypeAvg      ViewType = "Avg"      // Average value
	ViewTypeFirst    ViewType = "First"    // First entity
	ViewTypeLast     ViewType = "Last"     // Last entity
	ViewTypeGroupBy  ViewType = "GroupBy"  // Group by field
	ViewTypeDistinct ViewType = "Distinct" // Distinct values
	ViewTypeMap      ViewType = "Map"      // Transform each entity
	ViewTypeReduce   ViewType = "Reduce"   // Reduce to single value
	ViewTypeSort     ViewType = "Sort"     // Sort entities
	ViewTypeDistance ViewType = "Distance" // GPS distance calculation
)

// View represents a computed/derived value from selected entities
type View struct {
	Type ViewType `json:"type"`

	// Field access
	Field Path `json:"field,omitempty"`

	// Aggregate options
	Return string `json:"return,omitempty"` // "value" or "entity"

	// GroupBy options
	GroupField string `json:"groupField,omitempty"`
	Aggregate  *View  `json:"aggregate,omitempty"`

	// Map/Transform options
	Transform map[string]interface{} `json:"transform,omitempty"`

	// Reduce options
	Initial     interface{} `json:"initial,omitempty"`
	Reducer     string      `json:"reducer,omitempty"` // Expression string
	Accumulator string      `json:"accumulator,omitempty"`

	// Sort options
	By    Path   `json:"by,omitempty"`
	Order string `json:"order,omitempty"` // "asc" or "desc"
	Limit int    `json:"limit,omitempty"`

	// Distance options
	From Path   `json:"from,omitempty"`
	To   Path   `json:"to,omitempty"`
	Unit string `json:"unit,omitempty"` // "meters", "kilometers"

	// Filter for Count
	Where *WhereClause `json:"where,omitempty"`
}

// DependsOn returns the state paths this view depends on
func (v *View) DependsOn() []Path {
	var paths []Path

	if v.Field != "" {
		paths = append(paths, v.Field)
	}
	if v.By != "" {
		paths = append(paths, v.By)
	}
	if v.From != "" {
		paths = append(paths, v.From)
	}
	if v.To != "" {
		paths = append(paths, v.To)
	}
	if v.Aggregate != nil {
		paths = append(paths, v.Aggregate.DependsOn()...)
	}

	// Extract paths from transform map
	for _, val := range v.Transform {
		if str, ok := val.(string); ok && len(str) > 0 && str[0] == '$' {
			paths = append(paths, Path(str))
		}
	}

	return paths
}

// UnmarshalJSON implements custom JSON unmarshaling for View
func (v *View) UnmarshalJSON(data []byte) error {
	type viewAlias View
	var alias viewAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return fmt.Errorf("invalid view: %w", err)
	}
	*v = View(alias)

	// Validate view type
	switch v.Type {
	case ViewTypeField, ViewTypeMax, ViewTypeMin, ViewTypeSum, ViewTypeCount,
		ViewTypeAvg, ViewTypeFirst, ViewTypeLast, ViewTypeGroupBy,
		ViewTypeDistinct, ViewTypeMap, ViewTypeReduce, ViewTypeSort, ViewTypeDistance:
		// Valid
	case "":
		// Default to Field if type is empty but field is set
		if v.Field != "" {
			v.Type = ViewTypeField
		} else {
			return fmt.Errorf("view type is required")
		}
	default:
		return fmt.Errorf("unknown view type: %s", v.Type)
	}

	return nil
}
