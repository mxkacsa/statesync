package eval

import (
	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
	"github.com/mxkacsa/statesync/cmd/logicgen/internal/geo"
)

// Wrapper functions that convert between ast.GeoPoint and geo.Point
// This maintains backward compatibility while using the shared geo package

// haversineDistance calculates the distance between two GPS points in meters
func haversineDistance(from, to ast.GeoPoint) float64 {
	return geo.HaversineDistance(
		geo.Point{Lat: from.Lat, Lon: from.Lon},
		geo.Point{Lat: to.Lat, Lon: to.Lon},
	)
}

// bearing calculates the initial bearing from point 'from' to point 'to' in degrees
func bearing(from, to ast.GeoPoint) float64 {
	return geo.Bearing(
		geo.Point{Lat: from.Lat, Lon: from.Lon},
		geo.Point{Lat: to.Lat, Lon: to.Lon},
	)
}

// moveTowards calculates a new position moving from 'current' towards 'target'
// by 'distance' meters
func moveTowards(current, target ast.GeoPoint, distance float64) ast.GeoPoint {
	result := geo.MoveTowards(
		geo.Point{Lat: current.Lat, Lon: current.Lon},
		geo.Point{Lat: target.Lat, Lon: target.Lon},
		distance,
	)
	return ast.GeoPoint{Lat: result.Lat, Lon: result.Lon}
}

// pointInCircle checks if a point is within a circle
func pointInCircle(point, center ast.GeoPoint, radiusMeters float64) bool {
	return geo.PointInCircle(
		geo.Point{Lat: point.Lat, Lon: point.Lon},
		geo.Point{Lat: center.Lat, Lon: center.Lon},
		radiusMeters,
	)
}

// pointInPolygon checks if a point is inside a polygon using ray casting algorithm
func pointInPolygon(point ast.GeoPoint, polygon []ast.GeoPoint) bool {
	geoPolygon := make([]geo.Point, len(polygon))
	for i, p := range polygon {
		geoPolygon[i] = geo.Point{Lat: p.Lat, Lon: p.Lon}
	}
	return geo.PointInPolygon(
		geo.Point{Lat: point.Lat, Lon: point.Lon},
		geoPolygon,
	)
}

// interpolatePosition linearly interpolates between two GPS points by ratio (0.0 to 1.0)
func interpolatePosition(from, to ast.GeoPoint, ratio float64) ast.GeoPoint {
	result := geo.Interpolate(
		geo.Point{Lat: from.Lat, Lon: from.Lon},
		geo.Point{Lat: to.Lat, Lon: to.Lon},
		ratio,
	)
	return ast.GeoPoint{Lat: result.Lat, Lon: result.Lon}
}

// normalizeSpeedToMetersPerMillisecond converts various speed units to m/ms
func normalizeSpeedToMetersPerMillisecond(speed float64, unit string) float64 {
	return geo.NormalizeSpeedToMetersPerMillisecond(speed, unit)
}

// convertDistance converts distance to meters
func convertDistance(distance float64, unit string) float64 {
	return geo.ConvertDistance(distance, unit)
}
