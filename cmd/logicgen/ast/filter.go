package ast

import (
	"encoding/json"
	"fmt"
)

// Filter represents a state transformation for a specific viewer.
// It reuses View pipeline operations and adds transformation ops.
//
// A Filter is essentially a View that transforms data instead of just querying it.
// It uses the same WhereClause syntax as View for conditions.
type Filter struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"` // Pointer to distinguish unset from false
	Params      []FilterParam     `json:"params,omitempty"`
	Operations  []FilterOperation `json:"operations"`
}

// IsEnabled returns whether the filter is enabled (default true)
func (f *Filter) IsEnabled() bool {
	if f.Enabled == nil {
		return true
	}
	return *f.Enabled
}

// SetEnabled sets the enabled state
func (f *Filter) SetEnabled(enabled bool) {
	f.Enabled = &enabled
}

// FilterParam defines a filter parameter
type FilterParam struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"`              // "string", "int", "bool"
	Default interface{} `json:"default,omitempty"` // Default value
}

// FilterOperationType identifies the type of filter operation
type FilterOperationType string

const (
	// Array operations (reuse View semantics)
	FilterOpKeepWhere   FilterOperationType = "KeepWhere"   // View's Filter - keep matching
	FilterOpRemoveWhere FilterOperationType = "RemoveWhere" // Inverse of Filter - remove matching

	// Transformation operations (new - modify data)
	FilterOpHideFieldsWhere   FilterOperationType = "HideFieldsWhere"   // Zero out fields on matching items
	FilterOpReplaceFieldWhere FilterOperationType = "ReplaceFieldWhere" // Replace field value on matching items

	// Legacy aliases
	FilterOpFilterArray FilterOperationType = "FilterArray" // -> KeepWhere
)

// FilterOperation represents a single filter transformation.
// Uses WhereClause from View for conditions.
type FilterOperation struct {
	ID   string              `json:"id"`
	Type FilterOperationType `json:"type"`

	// Target array path: "$.Chat", "$.Players"
	Target string `json:"target"`

	// Where condition - uses View's WhereClause
	Where *WhereClause `json:"where,omitempty"`

	// For HideFieldsWhere - fields to set to zero value
	Fields []string `json:"fields,omitempty"`

	// For ReplaceFieldWhere - field to modify and new value
	Field string      `json:"field,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Filter
func (f *Filter) UnmarshalJSON(data []byte) error {
	type filterAlias Filter
	var alias filterAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return fmt.Errorf("invalid filter: %w", err)
	}
	*f = Filter(alias)

	if f.Name == "" {
		return fmt.Errorf("filter name is required")
	}

	return nil
}

// GetParam returns a parameter by name
func (f *Filter) GetParam(name string) *FilterParam {
	for i := range f.Params {
		if f.Params[i].Name == name {
			return &f.Params[i]
		}
	}
	return nil
}

// HasParam checks if a parameter exists
func (f *Filter) HasParam(name string) bool {
	return f.GetParam(name) != nil
}
