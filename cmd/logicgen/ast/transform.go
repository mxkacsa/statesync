package ast

import (
	"encoding/json"
	"fmt"
)

// TransformType identifies the type of transform
type TransformType string

const (
	// Math transforms
	TransformTypeAdd      TransformType = "Add"
	TransformTypeSubtract TransformType = "Subtract"
	TransformTypeMultiply TransformType = "Multiply"
	TransformTypeDivide   TransformType = "Divide"
	TransformTypeModulo   TransformType = "Modulo"
	TransformTypeClamp    TransformType = "Clamp"
	TransformTypeRound    TransformType = "Round"
	TransformTypeFloor    TransformType = "Floor"
	TransformTypeCeil     TransformType = "Ceil"
	TransformTypeAbs      TransformType = "Abs"
	TransformTypeMin      TransformType = "Min"
	TransformTypeMax      TransformType = "Max"
	TransformTypeRandom   TransformType = "Random"

	// GPS transforms
	TransformTypeMoveTowards    TransformType = "MoveTowards"
	TransformTypeGpsDistance    TransformType = "GpsDistance"
	TransformTypeGpsBearing     TransformType = "GpsBearing"
	TransformTypePointInRadius  TransformType = "PointInRadius"
	TransformTypePointInPolygon TransformType = "PointInPolygon"

	// String transforms
	TransformTypeConcat    TransformType = "Concat"
	TransformTypeFormat    TransformType = "Format"
	TransformTypeSubstring TransformType = "Substring"
	TransformTypeToUpper   TransformType = "ToUpper"
	TransformTypeToLower   TransformType = "ToLower"
	TransformTypeTrim      TransformType = "Trim"

	// Logic transforms
	TransformTypeIf       TransformType = "If"
	TransformTypeCoalesce TransformType = "Coalesce"
	TransformTypeNot      TransformType = "Not"

	// Time transforms
	TransformTypeNow       TransformType = "Now"
	TransformTypeTimeSince TransformType = "TimeSince"
	TransformTypeTimeAdd   TransformType = "TimeAdd"

	// UUID transform
	TransformTypeUUID TransformType = "UUID"
)

// Transform represents a pure transformation of values
type Transform struct {
	Type TransformType `json:"type"`

	// Math operation arguments
	Left  interface{} `json:"left,omitempty"`  // Can be path, value, or nested transform
	Right interface{} `json:"right,omitempty"` // Can be path, value, or nested transform
	Value interface{} `json:"value,omitempty"` // Single value input

	// Clamp arguments
	Min interface{} `json:"min,omitempty"`
	Max interface{} `json:"max,omitempty"`

	// GPS arguments
	Current interface{} `json:"current,omitempty"` // Current position (path or GeoPoint)
	Target  interface{} `json:"target,omitempty"`  // Target position (path or GeoPoint)
	Speed   float64     `json:"speed,omitempty"`   // Speed value
	Unit    string      `json:"unit,omitempty"`    // Unit: "km/h", "m/s", "meters", etc.
	From    interface{} `json:"from,omitempty"`    // From position
	To      interface{} `json:"to,omitempty"`      // To position
	Center  interface{} `json:"center,omitempty"`  // Circle center
	Radius  float64     `json:"radius,omitempty"`  // Circle radius
	Polygon []GeoPoint  `json:"polygon,omitempty"` // Polygon points

	// String arguments
	Strings []interface{} `json:"strings,omitempty"` // For Concat
	Format  string        `json:"format,omitempty"`  // For Format
	Args    []interface{} `json:"args,omitempty"`    // For Format
	Start   int           `json:"start,omitempty"`   // For Substring
	Length  int           `json:"length,omitempty"`  // For Substring

	// Conditional arguments
	Condition interface{}   `json:"condition,omitempty"` // Can be expression or path
	Then      interface{}   `json:"then,omitempty"`
	Else      interface{}   `json:"else,omitempty"`
	Values    []interface{} `json:"values,omitempty"` // For Coalesce

	// Random arguments
	MinValue interface{} `json:"minValue,omitempty"`
	MaxValue interface{} `json:"maxValue,omitempty"`

	// Time arguments
	Duration int         `json:"duration,omitempty"` // For TimeAdd (milliseconds)
	Since    interface{} `json:"since,omitempty"`    // For TimeSince
}

// DependsOn returns the state paths this transform depends on
func (t *Transform) DependsOn() []Path {
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
		case *Transform:
			paths = append(paths, val.DependsOn()...)
		case map[string]interface{}:
			// Could be a nested transform
			if typeStr, ok := val["type"].(string); ok && typeStr != "" {
				data, _ := json.Marshal(val)
				var nested Transform
				if json.Unmarshal(data, &nested) == nil {
					paths = append(paths, nested.DependsOn()...)
				}
			}
		}
	}

	collectPaths(t.Left)
	collectPaths(t.Right)
	collectPaths(t.Value)
	collectPaths(t.Current)
	collectPaths(t.Target)
	collectPaths(t.From)
	collectPaths(t.To)
	collectPaths(t.Center)
	collectPaths(t.Min)
	collectPaths(t.Max)
	collectPaths(t.Condition)
	collectPaths(t.Then)
	collectPaths(t.Else)
	collectPaths(t.Since)

	for _, s := range t.Strings {
		collectPaths(s)
	}
	for _, a := range t.Args {
		collectPaths(a)
	}
	for _, v := range t.Values {
		collectPaths(v)
	}

	return paths
}

// UnmarshalJSON implements custom JSON unmarshaling for Transform
func (t *Transform) UnmarshalJSON(data []byte) error {
	type transformAlias Transform
	var alias transformAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return fmt.Errorf("invalid transform: %w", err)
	}
	*t = Transform(alias)

	// Validate type if set
	if t.Type == "" {
		return fmt.Errorf("transform type is required")
	}

	return nil
}
