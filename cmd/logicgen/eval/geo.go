package eval

import (
	"math"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

const (
	// Earth's radius in meters
	earthRadiusMeters = 6371000.0
	// Conversion factors
	metersPerKilometer = 1000.0
	degreesToRadians   = math.Pi / 180.0
	radiansToDegrees   = 180.0 / math.Pi
)

// haversineDistance calculates the distance between two GPS points in meters
// using the Haversine formula for great-circle distance
func haversineDistance(from, to ast.GeoPoint) float64 {
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

// bearing calculates the initial bearing from point 'from' to point 'to' in degrees
func bearing(from, to ast.GeoPoint) float64 {
	lat1 := from.Lat * degreesToRadians
	lat2 := to.Lat * degreesToRadians
	deltaLon := (to.Lon - from.Lon) * degreesToRadians

	y := math.Sin(deltaLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(deltaLon)

	brng := math.Atan2(y, x) * radiansToDegrees
	// Normalize to 0-360
	return math.Mod(brng+360, 360)
}

// moveTowards calculates a new position moving from 'current' towards 'target'
// by 'distance' meters
func moveTowards(current, target ast.GeoPoint, distance float64) ast.GeoPoint {
	// Calculate total distance to target
	totalDist := haversineDistance(current, target)
	if totalDist <= 0 || distance <= 0 {
		return current
	}

	// If distance is greater than remaining distance, clamp to target
	if distance >= totalDist {
		return target
	}

	// Calculate bearing
	brng := bearing(current, target) * degreesToRadians

	// Calculate new position using spherical geometry
	lat1 := current.Lat * degreesToRadians
	lon1 := current.Lon * degreesToRadians
	angularDist := distance / earthRadiusMeters

	lat2 := math.Asin(math.Sin(lat1)*math.Cos(angularDist) +
		math.Cos(lat1)*math.Sin(angularDist)*math.Cos(brng))

	lon2 := lon1 + math.Atan2(
		math.Sin(brng)*math.Sin(angularDist)*math.Cos(lat1),
		math.Cos(angularDist)-math.Sin(lat1)*math.Sin(lat2),
	)

	return ast.GeoPoint{
		Lat: lat2 * radiansToDegrees,
		Lon: lon2 * radiansToDegrees,
	}
}

// pointInCircle checks if a point is within a circle
func pointInCircle(point, center ast.GeoPoint, radiusMeters float64) bool {
	return haversineDistance(point, center) <= radiusMeters
}

// pointInPolygon checks if a point is inside a polygon using ray casting algorithm
// Note: This uses simple cartesian approximation, suitable for small areas
func pointInPolygon(point ast.GeoPoint, polygon []ast.GeoPoint) bool {
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

// interpolatePosition linearly interpolates between two GPS points by ratio (0.0 to 1.0)
func interpolatePosition(from, to ast.GeoPoint, ratio float64) ast.GeoPoint {
	if ratio <= 0 {
		return from
	}
	if ratio >= 1 {
		return to
	}

	return ast.GeoPoint{
		Lat: from.Lat + (to.Lat-from.Lat)*ratio,
		Lon: from.Lon + (to.Lon-from.Lon)*ratio,
	}
}

// normalizeSpeedToMetersPerMillisecond converts various speed units to m/ms
func normalizeSpeedToMetersPerMillisecond(speed float64, unit string) float64 {
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

// convertDistance converts distance to meters
func convertDistance(distance float64, unit string) float64 {
	switch unit {
	case "km", "kilometers":
		return distance * metersPerKilometer
	case "miles", "mi":
		return distance * 1609.344
	case "feet", "ft":
		return distance * 0.3048
	default:
		// Default to meters
		return distance
	}
}
