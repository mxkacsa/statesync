package builtin

import (
	"fmt"
	"math"
)

const (
	earthRadiusMeters = 6371000.0
	degreesToRadians  = math.Pi / 180.0
	radiansToDegrees  = 180.0 / math.Pi
)

// GeoPoint represents a GPS coordinate
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

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

// GPS helper functions

func haversineDistance(from, to GeoPoint) float64 {
	lat1 := from.Lat * degreesToRadians
	lat2 := to.Lat * degreesToRadians
	deltaLat := (to.Lat - from.Lat) * degreesToRadians
	deltaLon := (to.Lon - from.Lon) * degreesToRadians

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

func bearing(from, to GeoPoint) float64 {
	lat1 := from.Lat * degreesToRadians
	lat2 := to.Lat * degreesToRadians
	deltaLon := (to.Lon - from.Lon) * degreesToRadians

	y := math.Sin(deltaLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(deltaLon)

	brng := math.Atan2(y, x) * radiansToDegrees
	return math.Mod(brng+360, 360)
}

func moveTowards(current, target GeoPoint, distance float64) GeoPoint {
	totalDist := haversineDistance(current, target)
	if totalDist <= 0 || distance <= 0 {
		return current
	}
	if distance >= totalDist {
		return target
	}

	brng := bearing(current, target) * degreesToRadians
	lat1 := current.Lat * degreesToRadians
	lon1 := current.Lon * degreesToRadians
	angularDist := distance / earthRadiusMeters

	lat2 := math.Asin(math.Sin(lat1)*math.Cos(angularDist) +
		math.Cos(lat1)*math.Sin(angularDist)*math.Cos(brng))

	lon2 := lon1 + math.Atan2(
		math.Sin(brng)*math.Sin(angularDist)*math.Cos(lat1),
		math.Cos(angularDist)-math.Sin(lat1)*math.Sin(lat2),
	)

	return GeoPoint{
		Lat: lat2 * radiansToDegrees,
		Lon: lon2 * radiansToDegrees,
	}
}

func pointInPolygon(point GeoPoint, polygon []GeoPoint) bool {
	if len(polygon) < 3 {
		return false
	}

	n := len(polygon)
	inside := false

	j := n - 1
	for i := 0; i < n; i++ {
		if ((polygon[i].Lat > point.Lat) != (polygon[j].Lat > point.Lat)) &&
			(point.Lon < (polygon[j].Lon-polygon[i].Lon)*(point.Lat-polygon[i].Lat)/(polygon[j].Lat-polygon[i].Lat)+polygon[i].Lon) {
			inside = !inside
		}
		j = i
	}

	return inside
}

// Conversion helpers

func toGeoPoint(v interface{}) (GeoPoint, error) {
	switch val := v.(type) {
	case GeoPoint:
		return val, nil
	case *GeoPoint:
		if val == nil {
			return GeoPoint{}, fmt.Errorf("nil GeoPoint")
		}
		return *val, nil
	case map[string]interface{}:
		lat, latOk := val["lat"].(float64)
		lon, lonOk := val["lon"].(float64)
		if !latOk || !lonOk {
			lat, latOk = val["Lat"].(float64)
			lon, lonOk = val["Lon"].(float64)
		}
		if latOk && lonOk {
			return GeoPoint{Lat: lat, Lon: lon}, nil
		}
		return GeoPoint{}, fmt.Errorf("invalid GeoPoint map")
	default:
		return GeoPoint{}, fmt.Errorf("cannot convert %T to GeoPoint", v)
	}
}

func toGeoPointSlice(v interface{}) ([]GeoPoint, error) {
	switch val := v.(type) {
	case []GeoPoint:
		return val, nil
	case []interface{}:
		result := make([]GeoPoint, len(val))
		for i, item := range val {
			pt, err := toGeoPoint(item)
			if err != nil {
				return nil, fmt.Errorf("item %d: %w", i, err)
			}
			result[i] = pt
		}
		return result, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to []GeoPoint", v)
	}
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
