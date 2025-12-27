// Package ast defines the Abstract Syntax Tree for LogicGen v2 rules.
package ast

import (
	"encoding/json"
	"fmt"
)

// Path represents a path expression to access state values.
// Examples: "$.Players[0].Score", "param:playerID", "view:maxScore", "self.position"
type Path string

// PathType indicates what kind of path this is
type PathType string

const (
	PathTypeState   PathType = "state"   // $.Field or state:$.Field
	PathTypeParam   PathType = "param"   // param:name
	PathTypeView    PathType = "view"    // view:name
	PathTypeConst   PathType = "const"   // const:value
	PathTypeSelf    PathType = "self"    // self.field (entity context, resolved at runtime)
	PathTypeCurrent PathType = "current" // $ (current entity in selector context) - deprecated
)

// ParsedPath represents a parsed path expression
type ParsedPath struct {
	Type     PathType
	Raw      string
	Segments []PathSegment
}

// PathSegment represents one part of a path
type PathSegment struct {
	Field      string      // Field name
	Index      interface{} // Array index (int) or map key (string) or "*" for wildcard
	IsWildcard bool
}

// Value represents any value in the system
type Value interface{}

// Entity represents a game entity (Player, Drone, etc.)
type Entity interface{}

// Expression represents a boolean or value expression
type Expression struct {
	// For simple comparisons
	Left  interface{} `json:"left,omitempty"`
	Op    string      `json:"op,omitempty"`
	Right interface{} `json:"right,omitempty"`

	// For logical operations (and, or, not)
	And []Expression `json:"and,omitempty"`
	Or  []Expression `json:"or,omitempty"`
	Not *Expression  `json:"not,omitempty"`
}

// Operator constants
const (
	OpEqual        = "=="
	OpNotEqual     = "!="
	OpGreater      = ">"
	OpGreaterEqual = ">="
	OpLess         = "<"
	OpLessEqual    = "<="
	OpContains     = "contains"
	OpIn           = "in"
)

// UnmarshalJSON implements custom JSON unmarshaling for Expression
func (e *Expression) UnmarshalJSON(data []byte) error {
	// Try as object first
	type exprAlias Expression
	var alias exprAlias
	if err := json.Unmarshal(data, &alias); err == nil {
		*e = Expression(alias)
		return nil
	}

	// Try as simple value (string path)
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		e.Left = str
		return nil
	}

	return fmt.Errorf("cannot unmarshal expression: %s", string(data))
}

// GeoPoint represents a GPS coordinate
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Event represents an incoming event
type Event struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params,omitempty"`
	Sender string                 `json:"sender,omitempty"`
}

// Vec2 represents a 2D vector (for positions)
type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
