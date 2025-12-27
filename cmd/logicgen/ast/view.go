package ast

import (
	"encoding/json"
	"fmt"
)

// View represents a pure, deterministic query over entities.
// Views are like SQL views or ECS queries - they compute derived state without side effects.
//
// Key principles:
// - Input: entity source (from state) + optional parameters
// - Output: list of entities or scalar value
// - No side effects (no SET, WAIT, EMIT, LOOP)
// - self only exists at engine runtime, not in view definition
// - Parameters are immutable and request-scoped
type View struct {
	Name   string `json:"name,omitempty"` // View name for referencing
	Source string `json:"source"`         // Entity collection name (e.g., "Players", "Drones")

	// Pipeline operations (applied in order)
	Pipeline []ViewOperation `json:"pipeline,omitempty"`

	// Parameters for parameterized views (immutable, request-scoped)
	Params map[string]ParamDef `json:"params,omitempty"`
}

// ParamDef defines a view parameter
type ParamDef struct {
	Type    string      `json:"type"`              // "number", "string", "vec2", "bool"
	Default interface{} `json:"default,omitempty"` // Default value
}

// ViewOperationType identifies the type of view operation
type ViewOperationType string

const (
	// Query operations (pure, no side effects)
	ViewOpFilter   ViewOperationType = "Filter"   // Filter entities by condition
	ViewOpMap      ViewOperationType = "Map"      // Transform/project fields
	ViewOpFlatMap  ViewOperationType = "FlatMap"  // Flatten nested arrays
	ViewOpOrderBy  ViewOperationType = "OrderBy"  // Sort entities
	ViewOpGroupBy  ViewOperationType = "GroupBy"  // Group entities by field
	ViewOpFirst    ViewOperationType = "First"    // Take first entity
	ViewOpLast     ViewOperationType = "Last"     // Take last entity
	ViewOpLimit    ViewOperationType = "Limit"    // Limit number of results
	ViewOpDistinct ViewOperationType = "Distinct" // Unique values

	// Aggregation operations (pure, return scalar)
	ViewOpMin   ViewOperationType = "Min"   // Minimum value
	ViewOpMax   ViewOperationType = "Max"   // Maximum value
	ViewOpSum   ViewOperationType = "Sum"   // Sum of values
	ViewOpCount ViewOperationType = "Count" // Count of entities
	ViewOpAvg   ViewOperationType = "Avg"   // Average value

	// Spatial operations (pure)
	ViewOpDistance ViewOperationType = "Distance" // Calculate distance
	ViewOpNearest  ViewOperationType = "Nearest"  // Find nearest entities
	ViewOpFarthest ViewOperationType = "Farthest" // Find farthest entities
)

// ViewOperation represents a single operation in the view pipeline
type ViewOperation struct {
	Type ViewOperationType `json:"type"`

	// Filter operation
	Where *WhereClause `json:"where,omitempty"` // Filter condition

	// Map operation - projection/transformation
	Fields map[string]interface{} `json:"fields,omitempty"` // Field projections

	// OrderBy operation
	By    Path   `json:"by,omitempty"`    // Field to sort by
	Order string `json:"order,omitempty"` // "asc" or "desc"

	// GroupBy operation
	GroupField string `json:"groupField,omitempty"` // Field to group by
	Aggregate  string `json:"aggregate,omitempty"`  // Aggregation function for grouped values

	// Limit operation
	Count int `json:"count,omitempty"` // Number of entities to take

	// Aggregation operations
	Field Path `json:"field,omitempty"` // Field for aggregation

	// Return mode for aggregations
	Return string `json:"return,omitempty"` // "value" or "entity"

	// Distance/Spatial operations
	From        interface{} `json:"from,omitempty"`        // Origin point (path or param reference)
	To          interface{} `json:"to,omitempty"`          // Target point (path or param reference)
	Position    Path        `json:"position,omitempty"`    // Entity position field
	Origin      interface{} `json:"origin,omitempty"`      // Origin for nearest/farthest
	MaxDistance float64     `json:"maxDistance,omitempty"` // Max distance in meters
	MinDistance float64     `json:"minDistance,omitempty"` // Min distance in meters
	Unit        string      `json:"unit,omitempty"`        // "meters" or "kilometers"
}

// WhereClause represents a filter condition
type WhereClause struct {
	Field string      `json:"field"`
	Op    string      `json:"op"` // "==", "!=", "<", ">", "<=", ">=", "contains", "in"
	Value interface{} `json:"value"`

	// Logical combinations
	And []*WhereClause `json:"and,omitempty"`
	Or  []*WhereClause `json:"or,omitempty"`
	Not *WhereClause   `json:"not,omitempty"`
}

// DependsOn returns the state paths this view depends on
func (v *View) DependsOn() []Path {
	var paths []Path

	// Source entity collection
	if v.Source != "" {
		paths = append(paths, Path("$."+v.Source))
	}

	// Collect from pipeline operations
	for _, op := range v.Pipeline {
		paths = append(paths, op.DependsOn()...)
	}

	return paths
}

// DependsOn returns paths that this operation depends on
func (op *ViewOperation) DependsOn() []Path {
	var paths []Path

	if op.Field != "" {
		paths = append(paths, op.Field)
	}
	if op.By != "" {
		paths = append(paths, op.By)
	}
	if op.Position != "" {
		paths = append(paths, op.Position)
	}

	// From/To/Origin can be paths
	addIfPath := func(v interface{}) {
		if str, ok := v.(string); ok && len(str) > 0 && str[0] == '$' {
			paths = append(paths, Path(str))
		}
	}
	addIfPath(op.From)
	addIfPath(op.To)
	addIfPath(op.Origin)

	// Fields map
	for _, val := range op.Fields {
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

	// Validate source is set
	if v.Source == "" {
		return fmt.Errorf("view source is required")
	}

	// Validate pipeline operations
	for i, op := range v.Pipeline {
		switch op.Type {
		case ViewOpFilter, ViewOpMap, ViewOpFlatMap, ViewOpOrderBy, ViewOpGroupBy,
			ViewOpFirst, ViewOpLast, ViewOpLimit, ViewOpDistinct,
			ViewOpMin, ViewOpMax, ViewOpSum, ViewOpCount, ViewOpAvg,
			ViewOpDistance, ViewOpNearest, ViewOpFarthest:
			// Valid
		case "":
			return fmt.Errorf("pipeline operation %d: type is required", i)
		default:
			return fmt.Errorf("pipeline operation %d: unknown type: %s", i, op.Type)
		}
	}

	return nil
}

// UnmarshalJSON implements custom JSON unmarshaling for WhereClause
func (w *WhereClause) UnmarshalJSON(data []byte) error {
	// Try standard format first
	type whereAlias WhereClause
	var alias whereAlias
	if err := json.Unmarshal(data, &alias); err == nil && (alias.Field != "" || alias.And != nil || alias.Or != nil || alias.Not != nil) {
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

// IsAggregation returns true if this view returns a scalar value
func (v *View) IsAggregation() bool {
	for _, op := range v.Pipeline {
		switch op.Type {
		case ViewOpMin, ViewOpMax, ViewOpSum, ViewOpCount, ViewOpAvg, ViewOpFirst, ViewOpLast:
			return true
		}
	}
	return false
}

// IsSpatial returns true if this view uses spatial operations
func (v *View) IsSpatial() bool {
	for _, op := range v.Pipeline {
		switch op.Type {
		case ViewOpDistance, ViewOpNearest, ViewOpFarthest:
			return true
		}
	}
	return false
}
