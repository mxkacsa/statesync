package builtin

import (
	"math"
	"testing"
)

func TestGPSDistance_Meters(t *testing.T) {
	// Budapest to Vienna is approximately 215km
	result, err := Call("GPS.Distance", map[string]interface{}{
		"from": GeoPoint{Lat: 47.4979, Lon: 19.0402}, // Budapest
		"to":   GeoPoint{Lat: 48.2082, Lon: 16.3738}, // Vienna
		"unit": "meters",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distance := result.(float64)
	// Should be between 200km and 250km
	if distance < 200000 || distance > 250000 {
		t.Errorf("expected distance ~215000 meters, got %f", distance)
	}
}

func TestGPSDistance_Kilometers(t *testing.T) {
	result, err := Call("GPS.Distance", map[string]interface{}{
		"from": GeoPoint{Lat: 47.4979, Lon: 19.0402},
		"to":   GeoPoint{Lat: 48.2082, Lon: 16.3738},
		"unit": "km",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distance := result.(float64)
	if distance < 200 || distance > 250 {
		t.Errorf("expected distance ~215 km, got %f", distance)
	}
}

func TestGPSDistance_Miles(t *testing.T) {
	result, err := Call("GPS.Distance", map[string]interface{}{
		"from": GeoPoint{Lat: 47.4979, Lon: 19.0402},
		"to":   GeoPoint{Lat: 48.2082, Lon: 16.3738},
		"unit": "miles",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distance := result.(float64)
	// 215km ≈ 133 miles
	if distance < 120 || distance > 150 {
		t.Errorf("expected distance ~133 miles, got %f", distance)
	}
}

func TestGPSDistance_SamePoint(t *testing.T) {
	point := GeoPoint{Lat: 47.4979, Lon: 19.0402}
	result, err := Call("GPS.Distance", map[string]interface{}{
		"from": point,
		"to":   point,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distance := result.(float64)
	if distance != 0 {
		t.Errorf("expected distance 0, got %f", distance)
	}
}

func TestGPSDistance_MapInput(t *testing.T) {
	result, err := Call("GPS.Distance", map[string]interface{}{
		"from": map[string]interface{}{"lat": 47.4979, "lon": 19.0402},
		"to":   map[string]interface{}{"lat": 48.2082, "lon": 16.3738},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	distance := result.(float64)
	if distance < 200000 || distance > 250000 {
		t.Errorf("expected distance ~215000 meters, got %f", distance)
	}
}

func TestGPSDistance_InvalidFrom(t *testing.T) {
	_, err := Call("GPS.Distance", map[string]interface{}{
		"from": "invalid",
		"to":   GeoPoint{Lat: 48.2082, Lon: 16.3738},
	})

	if err == nil {
		t.Error("expected error for invalid from point")
	}
}

func TestGPSBearing(t *testing.T) {
	// Due north
	result, err := Call("GPS.Bearing", map[string]interface{}{
		"from": GeoPoint{Lat: 0, Lon: 0},
		"to":   GeoPoint{Lat: 1, Lon: 0},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bearing := result.(float64)
	// Should be approximately 0 degrees (north)
	if bearing > 1 && bearing < 359 {
		t.Errorf("expected bearing ~0 degrees, got %f", bearing)
	}
}

func TestGPSBearing_East(t *testing.T) {
	result, err := Call("GPS.Bearing", map[string]interface{}{
		"from": GeoPoint{Lat: 0, Lon: 0},
		"to":   GeoPoint{Lat: 0, Lon: 1},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bearing := result.(float64)
	// Should be approximately 90 degrees (east)
	if math.Abs(bearing-90) > 1 {
		t.Errorf("expected bearing ~90 degrees, got %f", bearing)
	}
}

func TestGPSBearing_West(t *testing.T) {
	result, err := Call("GPS.Bearing", map[string]interface{}{
		"from": GeoPoint{Lat: 0, Lon: 0},
		"to":   GeoPoint{Lat: 0, Lon: -1},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bearing := result.(float64)
	// Should be approximately 270 degrees (west)
	if math.Abs(bearing-270) > 1 {
		t.Errorf("expected bearing ~270 degrees, got %f", bearing)
	}
}

func TestGPSMoveTowards(t *testing.T) {
	result, err := Call("GPS.MoveTowards", map[string]interface{}{
		"current":  GeoPoint{Lat: 47.0, Lon: 19.0},
		"target":   GeoPoint{Lat: 48.0, Lon: 19.0}, // Due north
		"distance": 1000.0,                         // 1km
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newPos := result.(GeoPoint)
	// Should move north
	if newPos.Lat <= 47.0 {
		t.Error("expected latitude to increase (move north)")
	}
	// Longitude should stay approximately the same
	if math.Abs(newPos.Lon-19.0) > 0.01 {
		t.Errorf("expected longitude ~19.0, got %f", newPos.Lon)
	}
}

func TestGPSMoveTowards_ExceedsTarget(t *testing.T) {
	result, err := Call("GPS.MoveTowards", map[string]interface{}{
		"current":  GeoPoint{Lat: 47.0, Lon: 19.0},
		"target":   GeoPoint{Lat: 47.001, Lon: 19.0}, // Very close
		"distance": 1000000.0,                        // 1000km - way more than needed
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newPos := result.(GeoPoint)
	// Should stop at target
	if math.Abs(newPos.Lat-47.001) > 0.0001 {
		t.Errorf("expected to reach target lat 47.001, got %f", newPos.Lat)
	}
}

func TestGPSMoveTowards_ZeroDistance(t *testing.T) {
	current := GeoPoint{Lat: 47.0, Lon: 19.0}
	result, err := Call("GPS.MoveTowards", map[string]interface{}{
		"current":  current,
		"target":   GeoPoint{Lat: 48.0, Lon: 19.0},
		"distance": 0.0,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newPos := result.(GeoPoint)
	if newPos.Lat != current.Lat || newPos.Lon != current.Lon {
		t.Error("expected to stay at current position with zero distance")
	}
}

func TestGPSPointInRadius_Inside(t *testing.T) {
	result, err := Call("GPS.PointInRadius", map[string]interface{}{
		"point":  GeoPoint{Lat: 47.498, Lon: 19.041},
		"center": GeoPoint{Lat: 47.4979, Lon: 19.0402},
		"radius": 1000.0, // 1km
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inside := result.(bool)
	if !inside {
		t.Error("expected point to be inside radius")
	}
}

func TestGPSPointInRadius_Outside(t *testing.T) {
	result, err := Call("GPS.PointInRadius", map[string]interface{}{
		"point":  GeoPoint{Lat: 48.0, Lon: 19.0}, // Far away
		"center": GeoPoint{Lat: 47.0, Lon: 19.0},
		"radius": 1000.0, // 1km
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inside := result.(bool)
	if inside {
		t.Error("expected point to be outside radius")
	}
}

func TestGPSPointInRadius_OnBoundary(t *testing.T) {
	center := GeoPoint{Lat: 0, Lon: 0}
	// Find a point roughly 1km north
	result, err := Call("GPS.MoveTowards", map[string]interface{}{
		"current":  center,
		"target":   GeoPoint{Lat: 1, Lon: 0},
		"distance": 1000.0,
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	boundary := result.(GeoPoint)

	// Point exactly on boundary should be inside (<=)
	result2, err := Call("GPS.PointInRadius", map[string]interface{}{
		"point":  boundary,
		"center": center,
		"radius": 1000.0,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inside := result2.(bool)
	if !inside {
		t.Error("expected point on boundary to be considered inside")
	}
}

func TestGPSPointInPolygon_Inside(t *testing.T) {
	// Simple square polygon
	polygon := []GeoPoint{
		{Lat: 0, Lon: 0},
		{Lat: 0, Lon: 10},
		{Lat: 10, Lon: 10},
		{Lat: 10, Lon: 0},
	}

	result, err := Call("GPS.PointInPolygon", map[string]interface{}{
		"point":   GeoPoint{Lat: 5, Lon: 5}, // Center
		"polygon": polygon,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inside := result.(bool)
	if !inside {
		t.Error("expected point to be inside polygon")
	}
}

func TestGPSPointInPolygon_Outside(t *testing.T) {
	polygon := []GeoPoint{
		{Lat: 0, Lon: 0},
		{Lat: 0, Lon: 10},
		{Lat: 10, Lon: 10},
		{Lat: 10, Lon: 0},
	}

	result, err := Call("GPS.PointInPolygon", map[string]interface{}{
		"point":   GeoPoint{Lat: 20, Lon: 20}, // Outside
		"polygon": polygon,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inside := result.(bool)
	if inside {
		t.Error("expected point to be outside polygon")
	}
}

func TestGPSPointInPolygon_TooFewVertices(t *testing.T) {
	// Less than 3 vertices - not a valid polygon
	polygon := []GeoPoint{
		{Lat: 0, Lon: 0},
		{Lat: 10, Lon: 10},
	}

	result, err := Call("GPS.PointInPolygon", map[string]interface{}{
		"point":   GeoPoint{Lat: 5, Lon: 5},
		"polygon": polygon,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inside := result.(bool)
	if inside {
		t.Error("expected false for invalid polygon")
	}
}

func TestGPSPointInPolygon_ComplexShape(t *testing.T) {
	// L-shaped polygon
	polygon := []GeoPoint{
		{Lat: 0, Lon: 0},
		{Lat: 0, Lon: 5},
		{Lat: 5, Lon: 5},
		{Lat: 5, Lon: 10},
		{Lat: 10, Lon: 10},
		{Lat: 10, Lon: 0},
	}

	// Point in the L's foot
	result, err := Call("GPS.PointInPolygon", map[string]interface{}{
		"point":   GeoPoint{Lat: 2, Lon: 2},
		"polygon": polygon,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.(bool) {
		t.Error("expected point to be inside L-shaped polygon")
	}

	// Point in the empty corner of the L
	result2, err := Call("GPS.PointInPolygon", map[string]interface{}{
		"point":   GeoPoint{Lat: 2, Lon: 8},
		"polygon": polygon,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result2.(bool) {
		t.Error("expected point to be outside L-shaped polygon's empty area")
	}
}

func TestGPSInterpolate(t *testing.T) {
	from := GeoPoint{Lat: 0, Lon: 0}
	to := GeoPoint{Lat: 10, Lon: 10}

	tests := []struct {
		name  string
		ratio float64
		want  GeoPoint
	}{
		{"start", 0.0, from},
		{"end", 1.0, to},
		{"middle", 0.5, GeoPoint{Lat: 5, Lon: 5}},
		{"quarter", 0.25, GeoPoint{Lat: 2.5, Lon: 2.5}},
		{"below zero", -0.5, from},
		{"above one", 1.5, to},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Call("GPS.Interpolate", map[string]interface{}{
				"from":  from,
				"to":    to,
				"ratio": tt.ratio,
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			pos := result.(GeoPoint)
			if math.Abs(pos.Lat-tt.want.Lat) > 0.0001 || math.Abs(pos.Lon-tt.want.Lon) > 0.0001 {
				t.Errorf("expected (%f, %f), got (%f, %f)", tt.want.Lat, tt.want.Lon, pos.Lat, pos.Lon)
			}
		})
	}
}

func TestGeoPoint_Conversions(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  GeoPoint
		err   bool
	}{
		{
			name:  "GeoPoint direct",
			input: GeoPoint{Lat: 47.5, Lon: 19.0},
			want:  GeoPoint{Lat: 47.5, Lon: 19.0},
		},
		{
			name:  "GeoPoint pointer",
			input: &GeoPoint{Lat: 47.5, Lon: 19.0},
			want:  GeoPoint{Lat: 47.5, Lon: 19.0},
		},
		{
			name:  "map lowercase",
			input: map[string]interface{}{"lat": 47.5, "lon": 19.0},
			want:  GeoPoint{Lat: 47.5, Lon: 19.0},
		},
		{
			name:  "map titlecase",
			input: map[string]interface{}{"Lat": 47.5, "Lon": 19.0},
			want:  GeoPoint{Lat: 47.5, Lon: 19.0},
		},
		{
			name:  "nil pointer",
			input: (*GeoPoint)(nil),
			err:   true,
		},
		{
			name:  "invalid type",
			input: "invalid",
			err:   true,
		},
		{
			name:  "invalid map",
			input: map[string]interface{}{"x": 47.5, "y": 19.0},
			err:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toGeoPoint(tt.input)
			if tt.err {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.want {
				t.Errorf("expected %v, got %v", tt.want, result)
			}
		})
	}
}

func TestGeoPointSlice_Conversion(t *testing.T) {
	// Direct slice
	input := []GeoPoint{
		{Lat: 0, Lon: 0},
		{Lat: 1, Lon: 1},
	}

	result, err := toGeoPointSlice(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 points, got %d", len(result))
	}

	// Interface slice
	input2 := []interface{}{
		GeoPoint{Lat: 0, Lon: 0},
		map[string]interface{}{"lat": 1.0, "lon": 1.0},
	}

	result2, err := toGeoPointSlice(input2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result2) != 2 {
		t.Errorf("expected 2 points, got %d", len(result2))
	}

	// Invalid item
	input3 := []interface{}{
		GeoPoint{Lat: 0, Lon: 0},
		"invalid",
	}

	_, err = toGeoPointSlice(input3)
	if err == nil {
		t.Error("expected error for invalid item")
	}

	// Invalid type
	_, err = toGeoPointSlice("invalid")
	if err == nil {
		t.Error("expected error for invalid type")
	}
}
