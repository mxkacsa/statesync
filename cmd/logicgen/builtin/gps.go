package builtin

import (
	"fmt"

	"github.com/mxkacsa/statesync/cmd/logicgen/internal/geo"
)

// GeoPoint represents a GPS coordinate (alias for geo.Point for backward compatibility)
type GeoPoint = geo.Point

func init() {
	// Register GPS nodes
	registerGPSNodes()
}

func registerGPSNodes() {
	// GPS.Distance - Calculate distance between two points
	Register(&NodeDefinition{
		Name:        "GPS.Distance",
		Category:    CategoryGPS,
		Description: "Calculates the distance between two GPS coordinates using the Haversine formula",
		Inputs: []PortDefinition{
			{Name: "from", Type: "GeoPoint", Required: true, Description: "Starting point"},
			{Name: "to", Type: "GeoPoint", Required: true, Description: "Ending point"},
			{Name: "unit", Type: "string", Required: false, Default: "meters", Description: "Output unit (meters, km, miles)"},
		},
		Outputs: []PortDefinition{
			{Name: "distance", Type: "float64", Description: "Distance in specified unit"},
		},
		Func: gpsDistance,
	})

	// GPS.Bearing - Calculate bearing between two points
	Register(&NodeDefinition{
		Name:        "GPS.Bearing",
		Category:    CategoryGPS,
		Description: "Calculates the initial bearing from one point to another",
		Inputs: []PortDefinition{
			{Name: "from", Type: "GeoPoint", Required: true, Description: "Starting point"},
			{Name: "to", Type: "GeoPoint", Required: true, Description: "Target point"},
		},
		Outputs: []PortDefinition{
			{Name: "bearing", Type: "float64", Description: "Bearing in degrees (0-360)"},
		},
		Func: gpsBearing,
	})

	// GPS.MoveTowards - Move a point towards another by a distance
	Register(&NodeDefinition{
		Name:        "GPS.MoveTowards",
		Category:    CategoryGPS,
		Description: "Moves a point towards a target by a specified distance",
		Inputs: []PortDefinition{
			{Name: "current", Type: "GeoPoint", Required: true, Description: "Current position"},
			{Name: "target", Type: "GeoPoint", Required: true, Description: "Target position"},
			{Name: "distance", Type: "float64", Required: true, Description: "Distance to move in meters"},
		},
		Outputs: []PortDefinition{
			{Name: "position", Type: "GeoPoint", Description: "New position"},
		},
		Func: gpsMoveTowards,
	})

	// GPS.PointInRadius - Check if point is within radius of center
	Register(&NodeDefinition{
		Name:        "GPS.PointInRadius",
		Category:    CategoryGPS,
		Description: "Checks if a point is within a radius of a center point",
		Inputs: []PortDefinition{
			{Name: "point", Type: "GeoPoint", Required: true, Description: "Point to check"},
			{Name: "center", Type: "GeoPoint", Required: true, Description: "Center of circle"},
			{Name: "radius", Type: "float64", Required: true, Description: "Radius in meters"},
		},
		Outputs: []PortDefinition{
			{Name: "inside", Type: "bool", Description: "True if point is inside radius"},
		},
		Func: gpsPointInRadius,
	})

	// GPS.PointInPolygon - Check if point is inside polygon
	Register(&NodeDefinition{
		Name:        "GPS.PointInPolygon",
		Category:    CategoryGPS,
		Description: "Checks if a point is inside a polygon",
		Inputs: []PortDefinition{
			{Name: "point", Type: "GeoPoint", Required: true, Description: "Point to check"},
			{Name: "polygon", Type: "[]GeoPoint", Required: true, Description: "Polygon vertices"},
		},
		Outputs: []PortDefinition{
			{Name: "inside", Type: "bool", Description: "True if point is inside polygon"},
		},
		Func: gpsPointInPolygon,
	})

	// GPS.Interpolate - Interpolate between two points
	Register(&NodeDefinition{
		Name:        "GPS.Interpolate",
		Category:    CategoryGPS,
		Description: "Interpolates a position between two points",
		Inputs: []PortDefinition{
			{Name: "from", Type: "GeoPoint", Required: true, Description: "Starting point"},
			{Name: "to", Type: "GeoPoint", Required: true, Description: "Ending point"},
			{Name: "ratio", Type: "float64", Required: true, Description: "Interpolation ratio (0.0 to 1.0)"},
		},
		Outputs: []PortDefinition{
			{Name: "position", Type: "GeoPoint", Description: "Interpolated position"},
		},
		Func: gpsInterpolate,
	})
}

// GPS node implementations

func gpsDistance(args map[string]interface{}) (interface{}, error) {
	from, err := toGeoPoint(args["from"])
	if err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}
	to, err := toGeoPoint(args["to"])
	if err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}

	distance := haversineDistance(from, to)

	// Convert unit
	unit, _ := args["unit"].(string)
	switch unit {
	case "km", "kilometers":
		distance /= 1000.0
	case "miles", "mi":
		distance /= 1609.344
	}

	return distance, nil
}

func gpsBearing(args map[string]interface{}) (interface{}, error) {
	from, err := toGeoPoint(args["from"])
	if err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}
	to, err := toGeoPoint(args["to"])
	if err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}

	return bearing(from, to), nil
}

func gpsMoveTowards(args map[string]interface{}) (interface{}, error) {
	current, err := toGeoPoint(args["current"])
	if err != nil {
		return nil, fmt.Errorf("current: %w", err)
	}
	target, err := toGeoPoint(args["target"])
	if err != nil {
		return nil, fmt.Errorf("target: %w", err)
	}
	distance, ok := toFloat64(args["distance"])
	if !ok {
		return nil, fmt.Errorf("distance must be a number")
	}

	return moveTowards(current, target, distance), nil
}

func gpsPointInRadius(args map[string]interface{}) (interface{}, error) {
	point, err := toGeoPoint(args["point"])
	if err != nil {
		return nil, fmt.Errorf("point: %w", err)
	}
	center, err := toGeoPoint(args["center"])
	if err != nil {
		return nil, fmt.Errorf("center: %w", err)
	}
	radius, ok := toFloat64(args["radius"])
	if !ok {
		return nil, fmt.Errorf("radius must be a number")
	}

	return haversineDistance(point, center) <= radius, nil
}

func gpsPointInPolygon(args map[string]interface{}) (interface{}, error) {
	point, err := toGeoPoint(args["point"])
	if err != nil {
		return nil, fmt.Errorf("point: %w", err)
	}

	polygonArg := args["polygon"]
	polygon, err := toGeoPointSlice(polygonArg)
	if err != nil {
		return nil, fmt.Errorf("polygon: %w", err)
	}

	return pointInPolygon(point, polygon), nil
}

func gpsInterpolate(args map[string]interface{}) (interface{}, error) {
	from, err := toGeoPoint(args["from"])
	if err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}
	to, err := toGeoPoint(args["to"])
	if err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}
	ratio, ok := toFloat64(args["ratio"])
	if !ok {
		return nil, fmt.Errorf("ratio must be a number")
	}

	if ratio <= 0 {
		return from, nil
	}
	if ratio >= 1 {
		return to, nil
	}

	return GeoPoint{
		Lat: from.Lat + (to.Lat-from.Lat)*ratio,
		Lon: from.Lon + (to.Lon-from.Lon)*ratio,
	}, nil
}

// GPS helper functions - delegating to shared geo package

func haversineDistance(from, to GeoPoint) float64 {
	return geo.HaversineDistance(from, to)
}

func bearing(from, to GeoPoint) float64 {
	return geo.Bearing(from, to)
}

func moveTowards(current, target GeoPoint, distance float64) GeoPoint {
	return geo.MoveTowards(current, target, distance)
}

func pointInPolygon(point GeoPoint, polygon []GeoPoint) bool {
	return geo.PointInPolygon(point, polygon)
}

// Conversion helpers - delegating to shared geo package

func toGeoPoint(v interface{}) (GeoPoint, error) {
	return geo.ToPoint(v)
}

func toGeoPointSlice(v interface{}) ([]GeoPoint, error) {
	return geo.ToPointSlice(v)
}

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
