package main

import (
	"strings"
	"testing"
)

// ============================================================================
// GPS Nodes Registry Tests
// ============================================================================

func TestGpsNodes_AreRegistered(t *testing.T) {
	// GPS nodes should be registered via init()
	gpsNodes := []NodeType{
		NodeGpsDistance,
		NodeGpsMoveToward,
		NodePointInCircle,
		NodePointInPolygon,
	}

	for _, nodeType := range gpsNodes {
		t.Run(string(nodeType), func(t *testing.T) {
			if !IsNodeRegistered(nodeType) {
				t.Errorf("Node %s should be registered", nodeType)
			}

			// Should be in custom nodes, not built-in
			def := GetCustomNodeDefinition(nodeType)
			if def == nil {
				t.Errorf("Node %s should be in custom nodes registry", nodeType)
			}

			if def.Generator == nil {
				t.Errorf("Node %s should have a generator function", nodeType)
			}
		})
	}
}

func TestGpsNodes_Category(t *testing.T) {
	byCategory := ListNodesByCategory()

	gpsNodes, ok := byCategory["GPS"]
	if !ok {
		t.Fatal("GPS category should exist")
	}

	expected := []NodeType{
		NodeGpsDistance,
		NodeGpsMoveToward,
		NodePointInCircle,
		NodePointInPolygon,
	}

	for _, nodeType := range expected {
		found := false
		for _, n := range gpsNodes {
			if n == nodeType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Node %s should be in GPS category", nodeType)
		}
	}
}

func TestGpsDistance_Generate(t *testing.T) {
	graph := &NodeGraph{
		Package: "testpkg",
		Handlers: []EventHandler{
			{
				Name:  "TestHandler",
				Event: "Test",
				Parameters: []Parameter{
					{Name: "lat1", Type: "float64"},
					{Name: "lng1", Type: "float64"},
					{Name: "lat2", Type: "float64"},
					{Name: "lng2", Type: "float64"},
				},
				Nodes: []Node{
					{
						ID:   "distance",
						Type: "GpsDistance",
						Inputs: map[string]interface{}{
							"lat1": "param:lat1",
							"lng1": "param:lng1",
							"lat2": "param:lat2",
							"lng2": "param:lng2",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "distance"},
				},
			},
		},
	}

	gen := NewCodeGenerator(graph, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should contain Haversine formula elements
	if !strings.Contains(codeStr, "Haversine") {
		t.Error("Generated code should mention Haversine")
	}
	if !strings.Contains(codeStr, "6371000") {
		t.Error("Generated code should contain Earth radius (6371000 meters)")
	}
	if !strings.Contains(codeStr, "math.Pi") {
		t.Error("Generated code should contain math.Pi for degree conversion")
	}
}

func TestPointInCircle_Generate(t *testing.T) {
	graph := &NodeGraph{
		Package: "testpkg",
		Handlers: []EventHandler{
			{
				Name:  "TestHandler",
				Event: "Test",
				Parameters: []Parameter{
					{Name: "playerLat", Type: "float64"},
					{Name: "playerLng", Type: "float64"},
				},
				Nodes: []Node{
					{
						ID:   "check",
						Type: "PointInCircle",
						Inputs: map[string]interface{}{
							"pointLat":     "param:playerLat",
							"pointLng":     "param:playerLng",
							"centerLat":    47.4979,
							"centerLng":    19.0402,
							"radiusMeters": 100.0,
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "check"},
				},
			},
		},
	}

	gen := NewCodeGenerator(graph, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should contain circle check logic
	if !strings.Contains(codeStr, "Haversine") {
		t.Error("Generated code should use Haversine for distance")
	}
	if !strings.Contains(codeStr, "<=") {
		t.Error("Generated code should compare distance to radius")
	}
}

func TestPointInPolygon_Generate(t *testing.T) {
	graph := &NodeGraph{
		Package: "testpkg",
		Handlers: []EventHandler{
			{
				Name:  "TestHandler",
				Event: "Test",
				Parameters: []Parameter{
					{Name: "lat", Type: "float64"},
					{Name: "lng", Type: "float64"},
					{Name: "polygon", Type: "[]GpsCoord"},
				},
				Nodes: []Node{
					{
						ID:   "check",
						Type: "PointInPolygon",
						Inputs: map[string]interface{}{
							"pointLat": "param:lat",
							"pointLng": "param:lng",
							"polygon":  "param:polygon",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "check"},
				},
			},
		},
	}

	gen := NewCodeGenerator(graph, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should contain ray casting algorithm
	if !strings.Contains(codeStr, "Ray casting") {
		t.Error("Generated code should mention ray casting algorithm")
	}
	if !strings.Contains(codeStr, "inside") {
		t.Error("Generated code should track inside state")
	}
}

func TestGpsMoveToward_Generate(t *testing.T) {
	graph := &NodeGraph{
		Package: "testpkg",
		Handlers: []EventHandler{
			{
				Name:  "TestHandler",
				Event: "Test",
				Parameters: []Parameter{
					{Name: "fromLat", Type: "float64"},
					{Name: "fromLng", Type: "float64"},
					{Name: "toLat", Type: "float64"},
					{Name: "toLng", Type: "float64"},
					{Name: "distance", Type: "float64"},
				},
				Nodes: []Node{
					{
						ID:   "move",
						Type: "GpsMoveToward",
						Inputs: map[string]interface{}{
							"fromLat":  "param:fromLat",
							"fromLng":  "param:fromLng",
							"toLat":    "param:toLat",
							"toLng":    "param:toLng",
							"distance": "param:distance",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "move"},
				},
			},
		},
	}

	gen := NewCodeGenerator(graph, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should contain bearing calculation
	if !strings.Contains(codeStr, "bearing") {
		t.Error("Generated code should calculate bearing")
	}
	// Should output new coordinates
	if !strings.Contains(codeStr, "newLat") {
		t.Error("Generated code should produce newLat output")
	}
	if !strings.Contains(codeStr, "newLng") {
		t.Error("Generated code should produce newLng output")
	}
}
