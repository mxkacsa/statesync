// Package geo provides shared GPS/geographic calculations for LogicGen.
// This package is used by both the evaluator (eval) and builtin nodes.
package geo

import (
	"fmt"
	"math"
)

const (
	// EarthRadiusMeters is the mean radius of Earth in meters
	EarthRadiusMeters = 6371000.0
	// MetersPerKilometer conversion factor
	MetersPerKilometer = 1000.0
	// DegreesToRadians conversion factor
	DegreesToRadians = math.Pi / 180.0
	// RadiansToDegrees conversion factor
	RadiansToDegrees = 180.0 / math.Pi
)

// Point represents a GPS coordinate (latitude/longitude)
type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// HaversineDistance calculates the distance between two GPS points in meters
// using the Haversine formula for great-circle distance.
func HaversineDistance(from, to Point) float64 {
	lat1 := from.Lat * DegreesToRadians
	lat2 := to.Lat * DegreesToRadians
	deltaLat := (to.Lat - from.Lat) * DegreesToRadians
	deltaLon := (to.Lon - from.Lon) * DegreesToRadians

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusMeters * c
}

// Bearing calculates the initial bearing from point 'from' to point 'to' in degrees (0-360)
func Bearing(from, to Point) float64 {
	lat1 := from.Lat * DegreesToRadians
	lat2 := to.Lat * DegreesToRadians
	deltaLon := (to.Lon - from.Lon) * DegreesToRadians

	y := math.Sin(deltaLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(deltaLon)

	brng := math.Atan2(y, x) * RadiansToDegrees
	// Normalize to 0-360
	return math.Mod(brng+360, 360)
}

// MoveTowards calculates a new position moving from 'current' towards 'target'
// by 'distance' meters. If distance exceeds the remaining distance, returns target.
func MoveTowards(current, target Point, distance float64) Point {
	totalDist := HaversineDistance(current, target)
	if totalDist <= 0 || distance <= 0 {
		return current
	}
	if distance >= totalDist {
		return target
	}

	brng := Bearing(current, target) * DegreesToRadians
	lat1 := current.Lat * DegreesToRadians
	lon1 := current.Lon * DegreesToRadians
	angularDist := distance / EarthRadiusMeters

	lat2 := math.Asin(math.Sin(lat1)*math.Cos(angularDist) +
		math.Cos(lat1)*math.Sin(angularDist)*math.Cos(brng))

	lon2 := lon1 + math.Atan2(
		math.Sin(brng)*math.Sin(angularDist)*math.Cos(lat1),
		math.Cos(angularDist)-math.Sin(lat1)*math.Sin(lat2),
	)

	return Point{
		Lat: lat2 * RadiansToDegrees,
		Lon: lon2 * RadiansToDegrees,
	}
}

// PointInCircle checks if a point is within a circle defined by center and radius (meters)
func PointInCircle(point, center Point, radiusMeters float64) bool {
	return HaversineDistance(point, center) <= radiusMeters
}

// PointInPolygon checks if a point is inside a polygon using ray casting algorithm.
// Note: This uses simple cartesian approximation, suitable for small areas.
func PointInPolygon(point Point, polygon []Point) bool {
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

// Interpolate linearly interpolates between two GPS points by ratio (0.0 to 1.0).
// Handles longitude wrap-around at +/-180 degrees.
func Interpolate(from, to Point, ratio float64) Point {
	if ratio <= 0 {
		return from
	}
	if ratio >= 1 {
		return to
	}

	// Handle longitude wrap-around at +/-180°
	deltaLon := to.Lon - from.Lon
	if deltaLon > 180 {
		deltaLon -= 360
	} else if deltaLon < -180 {
		deltaLon += 360
	}

	lon := from.Lon + deltaLon*ratio
	// Normalize longitude to [-180, 180]
	if lon > 180 {
		lon -= 360
	} else if lon < -180 {
		lon += 360
	}

	return Point{
		Lat: from.Lat + (to.Lat-from.Lat)*ratio,
		Lon: lon,
	}
}

// NormalizeSpeedToMetersPerMillisecond converts various speed units to m/ms
func NormalizeSpeedToMetersPerMillisecond(speed float64, unit string) float64 {
	switch unit {
	case "m/s", "mps", "meters_per_second":
		return speed / 1000.0 // m/s -> m/ms
	case "km/h", "kmh", "kilometers_per_hour":
		return (speed * 1000.0) / (3600.0 * 1000.0) // km/h -> m/ms
	case "mph", "miles_per_hour":
		return (speed * 1609.344) / (3600.0 * 1000.0) // mph -> m/ms
	default:
		// Default to m/s
		return speed / 1000.0
	}
}

// ConvertDistance converts distance to meters from various units
func ConvertDistance(distance float64, unit string) float64 {
	switch unit {
	case "km", "kilometers":
		return distance * MetersPerKilometer
	case "miles", "mi":
		return distance * 1609.344
	case "feet", "ft":
		return distance * 0.3048
	default:
		// Default to meters
		return distance
	}
}

// ToPoint converts various types to a Point
func ToPoint(v interface{}) (Point, error) {
	switch val := v.(type) {
	case Point:
		return val, nil
	case *Point:
		if val == nil {
			return Point{}, fmt.Errorf("nil Point")
		}
		return *val, nil
	case map[string]interface{}:
		lat, latOk := val["lat"].(float64)
		lon, lonOk := val["lon"].(float64)
		if !latOk || !lonOk {
			// Try capitalized versions
			lat, latOk = val["Lat"].(float64)
			lon, lonOk = val["Lon"].(float64)
		}
		if latOk && lonOk {
			return Point{Lat: lat, Lon: lon}, nil
		}
		return Point{}, fmt.Errorf("invalid Point map: missing lat/lon")
	default:
		return Point{}, fmt.Errorf("cannot convert %T to Point", v)
	}
}

// ToPointSlice converts a slice to []Point
func ToPointSlice(v interface{}) ([]Point, error) {
	switch val := v.(type) {
	case []Point:
		return val, nil
	case []interface{}:
		result := make([]Point, len(val))
		for i, item := range val {
			pt, err := ToPoint(item)
			if err != nil {
				return nil, fmt.Errorf("item %d: %w", i, err)
			}
			result[i] = pt
		}
		return result, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to []Point", v)
	}
}
