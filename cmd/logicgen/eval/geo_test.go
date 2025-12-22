package eval

import (
	"math"
	"testing"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name     string
		from     ast.GeoPoint
		to       ast.GeoPoint
		expected float64 // in meters
		delta    float64 // allowed deviation in meters
	}{
		{
			name:     "same point",
			from:     ast.GeoPoint{Lat: 47.5, Lon: 19.0},
			to:       ast.GeoPoint{Lat: 47.5, Lon: 19.0},
			expected: 0,
			delta:    0.1,
		},
		{
			name:     "Budapest to Vienna approx",
			from:     ast.GeoPoint{Lat: 47.4979, Lon: 19.0402}, // Budapest
			to:       ast.GeoPoint{Lat: 48.2082, Lon: 16.3738}, // Vienna
			expected: 214000,                                   // ~214 km
			delta:    5000,                                     // Allow 5km deviation
		},
		{
			name:     "short distance",
			from:     ast.GeoPoint{Lat: 47.5, Lon: 19.0},
			to:       ast.GeoPoint{Lat: 47.501, Lon: 19.0}, // ~111m north
			expected: 111,
			delta:    5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := haversineDistance(tt.from, tt.to)
			if math.Abs(result-tt.expected) > tt.delta {
				t.Errorf("expected ~%v meters, got %v meters (delta: %v)", tt.expected, result, tt.delta)
			}
		})
	}
}

func TestBearing(t *testing.T) {
	tests := []struct {
		name     string
		from     ast.GeoPoint
		to       ast.GeoPoint
		expected float64 // in degrees
		delta    float64
	}{
		{
			name:     "due north",
			from:     ast.GeoPoint{Lat: 0, Lon: 0},
			to:       ast.GeoPoint{Lat: 1, Lon: 0},
			expected: 0, // North is 0 degrees
			delta:    0.1,
		},
		{
			name:     "due east",
			from:     ast.GeoPoint{Lat: 0, Lon: 0},
			to:       ast.GeoPoint{Lat: 0, Lon: 1},
			expected: 90,
			delta:    0.1,
		},
		{
			name:     "due south",
			from:     ast.GeoPoint{Lat: 1, Lon: 0},
			to:       ast.GeoPoint{Lat: 0, Lon: 0},
			expected: 180,
			delta:    0.1,
		},
		{
			name:     "due west",
			from:     ast.GeoPoint{Lat: 0, Lon: 1},
			to:       ast.GeoPoint{Lat: 0, Lon: 0},
			expected: 270,
			delta:    0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bearing(tt.from, tt.to)
			if math.Abs(result-tt.expected) > tt.delta {
				t.Errorf("expected ~%v degrees, got %v degrees", tt.expected, result)
			}
		})
	}
}

func TestMoveTowards(t *testing.T) {
	tests := []struct {
		name     string
		current  ast.GeoPoint
		target   ast.GeoPoint
		distance float64
		check    func(result ast.GeoPoint) bool
	}{
		{
			name:     "zero distance",
			current:  ast.GeoPoint{Lat: 47.5, Lon: 19.0},
			target:   ast.GeoPoint{Lat: 47.6, Lon: 19.1},
			distance: 0,
			check: func(result ast.GeoPoint) bool {
				return result.Lat == 47.5 && result.Lon == 19.0
			},
		},
		{
			name:     "distance exceeds target",
			current:  ast.GeoPoint{Lat: 47.5, Lon: 19.0},
			target:   ast.GeoPoint{Lat: 47.501, Lon: 19.0},
			distance: 1000000, // way more than needed
			check: func(result ast.GeoPoint) bool {
				// Should clamp to target
				return math.Abs(result.Lat-47.501) < 0.0001 && math.Abs(result.Lon-19.0) < 0.0001
			},
		},
		{
			name:     "move halfway",
			current:  ast.GeoPoint{Lat: 0, Lon: 0},
			target:   ast.GeoPoint{Lat: 0, Lon: 1},
			distance: haversineDistance(ast.GeoPoint{Lat: 0, Lon: 0}, ast.GeoPoint{Lat: 0, Lon: 1}) / 2,
			check: func(result ast.GeoPoint) bool {
				// Should be approximately halfway
				return math.Abs(result.Lon-0.5) < 0.05 && math.Abs(result.Lat) < 0.05
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := moveTowards(tt.current, tt.target, tt.distance)
			if !tt.check(result) {
				t.Errorf("check failed for result: %+v", result)
			}
		})
	}
}

func TestPointInCircle(t *testing.T) {
	center := ast.GeoPoint{Lat: 47.5, Lon: 19.0}
	radius := 1000.0 // 1km

	tests := []struct {
		name     string
		point    ast.GeoPoint
		expected bool
	}{
		{
			name:     "point at center",
			point:    center,
			expected: true,
		},
		{
			name:     "point inside",
			point:    ast.GeoPoint{Lat: 47.504, Lon: 19.0}, // ~445m north
			expected: true,
		},
		{
			name:     "point outside",
			point:    ast.GeoPoint{Lat: 47.6, Lon: 19.0}, // ~11km away
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pointInCircle(tt.point, center, radius)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPointInPolygon(t *testing.T) {
	// Define a simple square polygon
	polygon := []ast.GeoPoint{
		{Lat: 0, Lon: 0},
		{Lat: 0, Lon: 1},
		{Lat: 1, Lon: 1},
		{Lat: 1, Lon: 0},
	}

	tests := []struct {
		name     string
		point    ast.GeoPoint
		expected bool
	}{
		{
			name:     "point inside",
			point:    ast.GeoPoint{Lat: 0.5, Lon: 0.5},
			expected: true,
		},
		{
			name:     "point outside",
			point:    ast.GeoPoint{Lat: 2, Lon: 2},
			expected: false,
		},
		{
			name:     "point on edge",
			point:    ast.GeoPoint{Lat: 0, Lon: 0.5},
			expected: true, // Ray casting may include edge points
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pointInPolygon(tt.point, polygon)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNormalizeSpeedToMetersPerMillisecond(t *testing.T) {
	tests := []struct {
		name     string
		speed    float64
		unit     string
		expected float64
	}{
		{
			name:     "m/s to m/ms",
			speed:    1,
			unit:     "m/s",
			expected: 0.001, // 1 m/s = 0.001 m/ms
		},
		{
			name:     "km/h to m/ms",
			speed:    3.6,
			unit:     "km/h",
			expected: 0.001, // 3.6 km/h = 1 m/s = 0.001 m/ms
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSpeedToMetersPerMillisecond(tt.speed, tt.unit)
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
