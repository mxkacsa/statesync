package eval

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// toFloat64 converts a value to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}

// compare compares two values with an operator
func compare(left, right interface{}, op string) (bool, error) {
	// Handle nil
	if left == nil || right == nil {
		switch op {
		case "==":
			return left == right, nil
		case "!=":
			return left != right, nil
		default:
			return false, nil
		}
	}

	// Try numeric comparison
	leftNum, leftOk := toFloat64(left)
	rightNum, rightOk := toFloat64(right)
	if leftOk && rightOk {
		switch op {
		case "==":
			return leftNum == rightNum, nil
		case "!=":
			return leftNum != rightNum, nil
		case ">":
			return leftNum > rightNum, nil
		case ">=":
			return leftNum >= rightNum, nil
		case "<":
			return leftNum < rightNum, nil
		case "<=":
			return leftNum <= rightNum, nil
		}
	}

	// String comparison
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	switch op {
	case "==":
		return leftStr == rightStr, nil
	case "!=":
		return leftStr != rightStr, nil
	case ">":
		return leftStr > rightStr, nil
	case ">=":
		return leftStr >= rightStr, nil
	case "<":
		return leftStr < rightStr, nil
	case "<=":
		return leftStr <= rightStr, nil
	case "contains":
		return strings.Contains(leftStr, rightStr), nil
	case "in":
		return strings.Contains(rightStr, leftStr), nil
	default:
		return false, fmt.Errorf("unknown operator: %s", op)
	}
}

// toGeoPoint converts a value to GeoPoint
func toGeoPoint(v interface{}) (ast.GeoPoint, error) {
	switch val := v.(type) {
	case ast.GeoPoint:
		return val, nil
	case *ast.GeoPoint:
		if val == nil {
			return ast.GeoPoint{}, fmt.Errorf("nil GeoPoint")
		}
		return *val, nil
	case map[string]interface{}:
		lat, latOk := val["lat"].(float64)
		lon, lonOk := val["lon"].(float64)
		if !latOk || !lonOk {
			// Try Lat/Lon
			lat, latOk = val["Lat"].(float64)
			lon, lonOk = val["Lon"].(float64)
		}
		if latOk && lonOk {
			return ast.GeoPoint{Lat: lat, Lon: lon}, nil
		}
		return ast.GeoPoint{}, fmt.Errorf("invalid GeoPoint map")
	default:
		return ast.GeoPoint{}, fmt.Errorf("cannot convert %T to GeoPoint", v)
	}
}

// toEntitySlice converts input to a slice of entities
func toEntitySlice(input interface{}) ([]interface{}, bool) {
	if input == nil {
		return nil, true
	}

	// Already a slice of interfaces
	if slice, ok := input.([]interface{}); ok {
		return slice, true
	}

	// Try reflection
	val := reflect.ValueOf(input)
	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		result := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			result[i] = val.Index(i).Interface()
		}
		return result, true
	}

	return nil, false
}

// getFieldValue gets a field value from an entity
// Supports nested paths: "Cards.Value", "Player.Cards[*].Value"
func getFieldValue(entity interface{}, field string) (interface{}, error) {
	// Handle nested paths by splitting on "."
	parts := splitFieldPath(field)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty field path")
	}

	var current interface{} = entity
	for i, part := range parts {
		// Check if current is an array result from [*] - need to map over it
		if isSlice(current) && !strings.Contains(part, "[*]") {
			// Apply remaining path to each element
			remainingPath := strings.Join(parts[i:], ".")
			return mapOverSlice(current, remainingPath)
		}

		val, err := getSimpleFieldValue(current, part)
		if err != nil {
			return nil, err
		}
		current = val
	}

	return current, nil
}

// isSlice checks if value is a slice/array
func isSlice(v interface{}) bool {
	if v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
}

// mapOverSlice applies a field path to each element of a slice
func mapOverSlice(slice interface{}, path string) ([]interface{}, error) {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, fmt.Errorf("not a slice")
	}

	var results []interface{}
	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i).Interface()
		val, err := getFieldValue(elem, path)
		if err == nil {
			results = append(results, val)
		}
	}
	return results, nil
}

// splitFieldPath splits a field path on "." while respecting array syntax
func splitFieldPath(path string) []string {
	var parts []string
	var current string
	inBracket := false

	for i := 0; i < len(path); i++ {
		ch := path[i]
		if ch == '[' {
			inBracket = true
			current += string(ch)
		} else if ch == ']' {
			inBracket = false
			current += string(ch)
		} else if ch == '.' && !inBracket {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// getSimpleFieldValue handles a single field access (possibly with array wildcard)
func getSimpleFieldValue(entity interface{}, field string) (interface{}, error) {
	// Check for array wildcard: "Cards[*]" -> collect from all elements
	if idx := strings.Index(field, "[*]"); idx > 0 {
		arrayField := field[:idx]
		remainingPath := ""
		if len(field) > idx+3 {
			remainingPath = field[idx+3:]
			if len(remainingPath) > 0 && remainingPath[0] == '.' {
				remainingPath = remainingPath[1:]
			}
		}

		// Get the array
		arr, err := getDirectFieldValue(entity, arrayField)
		if err != nil {
			return nil, err
		}

		arrVal := reflect.ValueOf(arr)
		if arrVal.Kind() != reflect.Slice && arrVal.Kind() != reflect.Array {
			return nil, fmt.Errorf("field %s is not an array", arrayField)
		}

		// Collect values from all elements
		var results []interface{}
		for i := 0; i < arrVal.Len(); i++ {
			elem := arrVal.Index(i).Interface()
			if remainingPath != "" {
				val, err := getFieldValue(elem, remainingPath)
				if err == nil {
					results = append(results, val)
				}
			} else {
				results = append(results, elem)
			}
		}
		return results, nil
	}

	return getDirectFieldValue(entity, field)
}

// getDirectFieldValue gets a direct field from struct or map
func getDirectFieldValue(entity interface{}, field string) (interface{}, error) {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		f := val.FieldByName(field)
		if f.IsValid() && f.CanInterface() {
			return f.Interface(), nil
		}
	}

	if val.Kind() == reflect.Map {
		v := val.MapIndex(reflect.ValueOf(field))
		if v.IsValid() {
			return v.Interface(), nil
		}
	}

	return nil, fmt.Errorf("field not found: %s", field)
}
