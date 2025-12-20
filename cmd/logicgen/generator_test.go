package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// CodeGenerator Creation Tests
// ============================================================================

func TestNewCodeGenerator(t *testing.T) {
	ng := &NodeGraph{
		Package:  "test",
		Handlers: []EventHandler{},
	}

	gen := NewCodeGenerator(ng, nil)
	if gen == nil {
		t.Fatal("NewCodeGenerator returned nil")
	}
	if gen.nodeGraph != ng {
		t.Error("nodeGraph not set correctly")
	}
}

func TestNewCodeGenerator_WithSchema(t *testing.T) {
	ng := &NodeGraph{
		Package:  "test",
		Handlers: []EventHandler{},
	}
	schema := &SchemaContext{
		Schema: &SchemaFile{},
		TypeIndex: map[string]*TypeDef{
			"GameState": {Name: "GameState", ID: 1},
		},
		RootType: &TypeDef{Name: "GameState", ID: 1},
	}

	gen := NewCodeGenerator(ng, schema)
	if gen.schema != schema {
		t.Error("schema not set correctly")
	}
}

// ============================================================================
// Code Generation Basic Tests
// ============================================================================

func TestGenerate_EmptyHandlers(t *testing.T) {
	ng := &NodeGraph{
		Package:  "test",
		Handlers: []EventHandler{},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(string(code), "package test") {
		t.Error("Generated code should contain package declaration")
	}
	if !strings.Contains(string(code), "import (") {
		t.Error("Generated code should contain imports")
	}
}

func TestGenerate_SimpleHandler(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:       "OnTest",
				Event:      "Test",
				Parameters: []Parameter{},
				Nodes:      []Node{},
				Flow:       []FlowEdge{},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "func OnTest(") {
		t.Error("Generated code should contain OnTest function")
	}
	if !strings.Contains(codeStr, "senderID string") {
		t.Error("Handler should have senderID parameter")
	}
}

func TestGenerate_HandlerWithParameters(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnPlayerAction",
				Event: "PlayerAction",
				Parameters: []Parameter{
					{Name: "playerID", Type: "string"},
					{Name: "x", Type: "float64"},
					{Name: "y", Type: "float64"},
				},
				Nodes: []Node{},
				Flow:  []FlowEdge{},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "playerID string") {
		t.Error("Handler should have playerID parameter")
	}
	if !strings.Contains(codeStr, "x float64") {
		t.Error("Handler should have x parameter")
	}
	if !strings.Contains(codeStr, "y float64") {
		t.Error("Handler should have y parameter")
	}
}

// ============================================================================
// Permission Generation Tests
// ============================================================================

func TestGenerate_HostOnlyPermission(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnStartGame",
				Event: "StartGame",
				Permissions: &EventPermissions{
					HostOnly: true,
				},
				Parameters: []Parameter{},
				Nodes:      []Node{},
				Flow:       []FlowEdge{},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "session.IsHost(senderID)") {
		t.Error("Should check IsHost for hostOnly permission")
	}
	if !strings.Contains(codeStr, "ErrNotHost") {
		t.Error("Should return ErrNotHost")
	}
}

func TestGenerate_PlayerParamPermission(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnPlayerMove",
				Event: "PlayerMove",
				Permissions: &EventPermissions{
					PlayerParam: "playerID",
				},
				Parameters: []Parameter{
					{Name: "playerID", Type: "string"},
				},
				Nodes: []Node{},
				Flow:  []FlowEdge{},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "playerID != senderID") {
		t.Error("Should check playerID != senderID")
	}
	if !strings.Contains(codeStr, "ErrNotAllowed") {
		t.Error("Should return ErrNotAllowed")
	}
}

func TestGenerate_AllowedPlayersPermission(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnAdminAction",
				Event: "AdminAction",
				Permissions: &EventPermissions{
					AllowedPlayers: []string{"admin1", "admin2"},
				},
				Parameters: []Parameter{},
				Nodes:      []Node{},
				Flow:       []FlowEdge{},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "_OnAdminAction_allowedPlayers") {
		t.Error("Should generate allowedPlayers map")
	}
	if !strings.Contains(codeStr, `"admin1": true`) {
		t.Error("Should include admin1 in allowedPlayers map")
	}
	if !strings.Contains(codeStr, `"admin2": true`) {
		t.Error("Should include admin2 in allowedPlayers map")
	}
}

// ============================================================================
// Node Generation Tests
// ============================================================================

func TestGenerate_CompareNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Parameters: []Parameter{
					{Name: "score", Type: "int"},
				},
				Nodes: []Node{
					{
						ID:   "checkScore",
						Type: "Compare",
						Inputs: map[string]interface{}{
							"left":  "param:score",
							"op":    map[string]interface{}{"constant": ">"},
							"right": map[string]interface{}{"constant": 100},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "checkScore"},
					{From: "checkScore", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "checkScore_result") {
		t.Error("Should generate result variable for Compare node")
	}
	if !strings.Contains(codeStr, "score > 100") {
		t.Error("Should generate comparison: score > 100")
	}
}

func TestGenerate_IfNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:       "OnTest",
				Event:      "Test",
				Parameters: []Parameter{},
				Nodes: []Node{
					{
						ID:   "checkCond",
						Type: "Compare",
						Inputs: map[string]interface{}{
							"left":  map[string]interface{}{"constant": 1},
							"op":    map[string]interface{}{"constant": "=="},
							"right": map[string]interface{}{"constant": 1},
						},
					},
					{
						ID:   "ifNode",
						Type: "If",
						Inputs: map[string]interface{}{
							"condition": "node:checkCond:result",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "checkCond"},
					{From: "checkCond", To: "ifNode"},
					{From: "ifNode", To: "end", Label: "true"},
					{From: "ifNode", To: "end", Label: "false"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "if checkCond_result") {
		t.Error("Should generate if statement")
	}
}

func TestGenerate_EmitToAllNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:       "OnTest",
				Event:      "Test",
				Parameters: []Parameter{},
				Nodes: []Node{
					{
						ID:   "emit",
						Type: "EmitToAll",
						Inputs: map[string]interface{}{
							"eventType": map[string]interface{}{"constant": "TestEvent"},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "emit"},
					{From: "emit", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, `session.Emit("TestEvent"`) {
		t.Error("Should generate session.Emit call")
	}
}

func TestGenerate_EmitToPlayerNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Parameters: []Parameter{
					{Name: "targetID", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "emit",
						Type: "EmitToPlayer",
						Inputs: map[string]interface{}{
							"playerID":  "param:targetID",
							"eventType": map[string]interface{}{"constant": "PrivateEvent"},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "emit"},
					{From: "emit", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "session.EmitTo(targetID") {
		t.Error("Should generate session.EmitTo call")
	}
}

// ============================================================================
// Session Node Generation Tests
// ============================================================================

func TestGenerate_KickPlayerNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnKick",
				Event: "Kick",
				Parameters: []Parameter{
					{Name: "targetID", Type: "string"},
					{Name: "reason", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "kick",
						Type: "KickPlayer",
						Inputs: map[string]interface{}{
							"playerID": "param:targetID",
							"reason":   "param:reason",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "kick"},
					{From: "kick", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "session.Kick(targetID, reason)") {
		t.Error("Should generate session.Kick call")
	}
}

func TestGenerate_GetHostPlayerNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:       "OnTest",
				Event:      "Test",
				Parameters: []Parameter{},
				Nodes: []Node{
					{
						ID:     "getHost",
						Type:   "GetHostPlayer",
						Inputs: map[string]interface{}{},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "getHost"},
					{From: "getHost", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "session.HostPlayerID()") {
		t.Error("Should generate session.HostPlayerID() call")
	}
}

func TestGenerate_IsHostNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Parameters: []Parameter{
					{Name: "playerID", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "checkHost",
						Type: "IsHost",
						Inputs: map[string]interface{}{
							"playerID": "param:playerID",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "checkHost"},
					{From: "checkHost", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "session.IsHost(playerID)") {
		t.Error("Should generate session.IsHost() call")
	}
}

// ============================================================================
// Filter Generation Tests
// ============================================================================

func TestGenerate_FilterRegistry(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Filters: []FilterDefinition{
			{
				Name:        "TestFilter",
				Description: "A test filter",
				Parameters: []Parameter{
					{Name: "param1", Type: "string"},
				},
				Nodes: []Node{},
				Flow:  []FlowEdge{},
			},
		},
		Handlers: []EventHandler{},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Check filter factory
	if !strings.Contains(codeStr, "func TestFilter(param1 string)") {
		t.Error("Should generate filter factory function")
	}

	// Check FilterRegistry type
	if !strings.Contains(codeStr, "type FilterRegistry struct") {
		t.Error("Should generate FilterRegistry type")
	}

	// Check registry methods
	if !strings.Contains(codeStr, "func (r *FilterRegistry) Add(") {
		t.Error("Should generate Add method")
	}
	if !strings.Contains(codeStr, "func (r *FilterRegistry) Remove(") {
		t.Error("Should generate Remove method")
	}
	if !strings.Contains(codeStr, "func (r *FilterRegistry) Has(") {
		t.Error("Should generate Has method")
	}
	if !strings.Contains(codeStr, "func (r *FilterRegistry) GetComposed(") {
		t.Error("Should generate GetComposed method")
	}

	// Check sync import
	if !strings.Contains(codeStr, `"sync"`) {
		t.Error("Should import sync package for FilterRegistry")
	}
}

func TestGenerate_AddFilterNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Filters: []FilterDefinition{
			{
				Name:       "HidePlayer",
				Parameters: []Parameter{{Name: "playerID", Type: "string"}},
			},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Parameters: []Parameter{
					{Name: "viewerID", Type: "string"},
					{Name: "targetID", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "addFilter",
						Type: "AddFilter",
						Inputs: map[string]interface{}{
							"viewerID":   "param:viewerID",
							"filterID":   "param:targetID",
							"filterName": map[string]interface{}{"constant": "HidePlayer"},
							"params": map[string]interface{}{
								"playerID": "param:targetID",
							},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "addFilter"},
					{From: "addFilter", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "HidePlayer(") {
		t.Error("Should call filter factory")
	}
	if !strings.Contains(codeStr, "filterRegistry.Add(") {
		t.Error("Should call filterRegistry.Add")
	}
}

func TestGenerate_RemoveFilterNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Filters: []FilterDefinition{
			{Name: "TestFilter"},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Parameters: []Parameter{
					{Name: "viewerID", Type: "string"},
					{Name: "filterID", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "removeFilter",
						Type: "RemoveFilter",
						Inputs: map[string]interface{}{
							"viewerID": "param:viewerID",
							"filterID": "param:filterID",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "removeFilter"},
					{From: "removeFilter", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "filterRegistry.Remove(") {
		t.Error("Should call filterRegistry.Remove")
	}
}

func TestGenerate_HasFilterNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Filters: []FilterDefinition{
			{Name: "TestFilter"},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Parameters: []Parameter{
					{Name: "viewerID", Type: "string"},
					{Name: "filterID", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "hasFilter",
						Type: "HasFilter",
						Inputs: map[string]interface{}{
							"viewerID": "param:viewerID",
							"filterID": "param:filterID",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "hasFilter"},
					{From: "hasFilter", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "filterRegistry.Has(") {
		t.Error("Should call filterRegistry.Has")
	}
}

func TestGenerate_FilterWithNodeBody(t *testing.T) {
	// Test that filter body nodes are actually generated
	ng := &NodeGraph{
		Package: "test",
		Filters: []FilterDefinition{
			{
				Name:        "HideEnemyLocations",
				Description: "Hides location data for enemy players",
				Parameters: []Parameter{
					{Name: "viewerTeam", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "checkTeam",
						Type: "Compare",
						Inputs: map[string]interface{}{
							"left":  "param:viewerTeam",
							"right": map[string]interface{}{"constant": "red"},
							"op":    map[string]interface{}{"constant": "=="},
						},
					},
					{
						ID:   "ifRed",
						Type: "If",
						Inputs: map[string]interface{}{
							"condition": "node:checkTeam:result",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "checkTeam"},
					{From: "checkTeam", To: "ifRed"},
					{From: "ifRed", To: "end"},
				},
			},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Nodes: []Node{},
				Flow:  []FlowEdge{},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Check that filter factory is generated
	if !strings.Contains(codeStr, "func HideEnemyLocations(") {
		t.Error("Should generate filter factory function")
	}

	// Check that filter body contains actual node logic (not just TODO)
	if strings.Contains(codeStr, "// TODO: Filter node logic") {
		t.Error("Filter body should contain actual node logic, not TODO placeholder")
	}

	// Check that Compare node logic is generated
	if !strings.Contains(codeStr, "viewerTeam ==") {
		t.Error("Should generate Compare node logic in filter body")
	}

	// Check that ShallowClone is called
	if !strings.Contains(codeStr, "ShallowClone()") {
		t.Error("Should call ShallowClone in filter")
	}

	// Check return statement
	if !strings.Contains(codeStr, "return filtered") {
		t.Error("Should return filtered state")
	}
}

// ============================================================================
// Math Node Generation Tests
// ============================================================================

func TestGenerate_MathNodes(t *testing.T) {
	tests := []struct {
		nodeType string
		operator string
	}{
		{"Add", "+"},
		{"Subtract", "-"},
		{"Multiply", "*"},
		{"Divide", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			ng := &NodeGraph{
				Package: "test",
				Handlers: []EventHandler{
					{
						Name:       "OnTest",
						Event:      "Test",
						Parameters: []Parameter{},
						Nodes: []Node{
							{
								ID:   "math",
								Type: tt.nodeType,
								Inputs: map[string]interface{}{
									"a": map[string]interface{}{"constant": 10},
									"b": map[string]interface{}{"constant": 5},
								},
							},
						},
						Flow: []FlowEdge{
							{From: "start", To: "math"},
							{From: "math", To: "end"},
						},
					},
				},
			}

			gen := NewCodeGenerator(ng, nil)
			code, err := gen.Generate()
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			codeStr := string(code)
			if !strings.Contains(codeStr, tt.operator) {
				t.Errorf("Should contain operator %s", tt.operator)
			}
		})
	}
}

// ============================================================================
// Debug Mode Tests
// ============================================================================

func TestGenerate_DebugMode(t *testing.T) {
	ng := &NodeGraph{
		Package:  "test",
		Handlers: []EventHandler{},
	}

	// Use NewCodeGeneratorWithDebug instead of SetDebugMode
	gen := NewCodeGeneratorWithDebug(ng, nil, true)

	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "//go:build debug") {
		t.Error("Should have debug build tag")
	}
}

// ============================================================================
// Integration Tests with Real Files
// ============================================================================

func TestGenerate_ChaseGameLogic(t *testing.T) {
	logicPath := filepath.Join("testdata", "chase_game_logic.json")
	schemaPath := filepath.Join("testdata", "chase_game.schema.json")

	if _, err := os.Stat(logicPath); os.IsNotExist(err) {
		t.Skip("Chase game logic file not found")
	}

	// Load schema
	schema, err := LoadSchema(schemaPath)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	// Load logic
	data, err := os.ReadFile(logicPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var ng NodeGraph
	if err := json.Unmarshal(data, &ng); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Generate
	gen := NewCodeGenerator(&ng, schema)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Verify handlers are generated
	if !strings.Contains(codeStr, "func OnGameTick(") {
		t.Error("Should generate OnGameTick handler")
	}

	// Verify package
	if !strings.Contains(codeStr, "package chase") {
		t.Error("Should have correct package name")
	}
}

func TestGenerate_PermissionsTestLogic(t *testing.T) {
	logicPath := filepath.Join("testdata", "permissions_test_logic.json")
	schemaPath := filepath.Join("testdata", "permissions_test.schema.json")

	if _, err := os.Stat(logicPath); os.IsNotExist(err) {
		t.Skip("Permissions test logic file not found")
	}

	// Load schema
	schema, err := LoadSchema(schemaPath)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	// Load logic
	data, err := os.ReadFile(logicPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var ng NodeGraph
	if err := json.Unmarshal(data, &ng); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Generate
	gen := NewCodeGenerator(&ng, schema)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Verify hostOnly permission
	if !strings.Contains(codeStr, "session.IsHost(senderID)") {
		t.Error("Should generate hostOnly check")
	}
}

func TestGenerate_FilterTestLogic(t *testing.T) {
	logicPath := filepath.Join("testdata", "filter_test_logic.json")
	schemaPath := filepath.Join("testdata", "permissions_test.schema.json")

	if _, err := os.Stat(logicPath); os.IsNotExist(err) {
		t.Skip("Filter test logic file not found")
	}

	// Load schema
	schema, err := LoadSchema(schemaPath)
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	// Load logic
	data, err := os.ReadFile(logicPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var ng NodeGraph
	if err := json.Unmarshal(data, &ng); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Generate
	gen := NewCodeGenerator(&ng, schema)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Verify filter generation
	if !strings.Contains(codeStr, "func HideEnemyLocations(") {
		t.Error("Should generate filter factory")
	}
	if !strings.Contains(codeStr, "type FilterRegistry struct") {
		t.Error("Should generate FilterRegistry")
	}
}

// ============================================================================
// Context Mode Tests (Wait Nodes)
// ============================================================================

func TestGenerate_WaitNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:       "OnTest",
				Event:      "Test",
				Parameters: []Parameter{},
				Nodes: []Node{
					{
						ID:   "wait",
						Type: "Wait",
						Inputs: map[string]interface{}{
							"duration": map[string]interface{}{"constant": 1000},
							"unit":     map[string]interface{}{"constant": "ms"},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "wait"},
					{From: "wait", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Handler should have context parameter
	if !strings.Contains(codeStr, "ctx context.Context") {
		t.Error("Handler with Wait node should have context parameter")
	}

	// Should use select with context
	if !strings.Contains(codeStr, "select {") {
		t.Error("Wait should use select statement")
	}
	if !strings.Contains(codeStr, "ctx.Done()") {
		t.Error("Wait should check ctx.Done()")
	}
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestGenerate_UnknownNodeType(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:       "OnTest",
				Event:      "Test",
				Parameters: []Parameter{},
				Nodes: []Node{
					{
						ID:     "unknown",
						Type:   "UnknownNodeType",
						Inputs: map[string]interface{}{},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "unknown"},
					{From: "unknown", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	// Unknown nodes should have TODO comment
	if !strings.Contains(codeStr, "TODO") {
		t.Error("Unknown node type should generate TODO comment")
	}
}

// ============================================================================
// Path Access Safety Tests
// ============================================================================

func TestGeneratePathAccess_KeyLookupWithNotFoundCheck(t *testing.T) {
	// Test that key_lookup paths generate proper not-found error handling
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnGetPlayer",
				Event: "GetPlayer",
				Parameters: []Parameter{
					{Name: "playerID", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "getPlayer",
						Type: "GetPlayer",
						Inputs: map[string]interface{}{
							"playerID": "param:playerID",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "getPlayer"},
					{From: "getPlayer", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Should have player not found error handling
	if !strings.Contains(codeStr, "player not found") {
		t.Error("GetPlayer should generate 'player not found' error check")
	}

	// Should check if player is nil
	if !strings.Contains(codeStr, "if") && !strings.Contains(codeStr, "nil") {
		t.Error("GetPlayer should check for nil player")
	}
}

func TestGenerateGetPlayer_DynamicTypeName(t *testing.T) {
	// Create a schema with a custom player type name
	schema := &SchemaContext{
		Schema: &SchemaFile{},
		RootType: &TypeDef{
			Name: "GameState",
			Fields: []*FieldDef{
				{
					Name: "Players",
					Type: "[]GamePlayer", // Custom type name, not just "Player"
					Key:  "ID",
				},
			},
		},
		TypeIndex: map[string]*TypeDef{
			"GameState": {
				Name: "GameState",
				Fields: []*FieldDef{
					{Name: "Players", Type: "[]GamePlayer", Key: "ID"},
				},
			},
			"GamePlayer": {
				Name: "GamePlayer",
				Fields: []*FieldDef{
					{Name: "ID", Type: "string"},
				},
			},
		},
	}

	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnGetPlayer",
				Event: "GetPlayer",
				Parameters: []Parameter{
					{Name: "playerID", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "getPlayer",
						Type: "GetPlayer",
						Inputs: map[string]interface{}{
							"playerID": "param:playerID",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "getPlayer"},
					{From: "getPlayer", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, schema)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Should use the custom type name from schema
	if !strings.Contains(codeStr, "*GamePlayer") {
		t.Error("GetPlayer should use type name from schema (GamePlayer), got code without *GamePlayer")
	}
}

// ============================================================================
// Effect Node Generation Tests
// ============================================================================

func TestGenerate_AddEffectNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnCollectEffect",
				Event: "CollectEffect",
				Parameters: []Parameter{
					{Name: "playerID", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "addEffect",
						Type: "AddEffect",
						Inputs: map[string]interface{}{
							"effectId":   map[string]interface{}{"constant": "noise-signal-1"},
							"effectName": map[string]interface{}{"constant": "NoiseSignalEffect"},
							"activator":  "param:playerID",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "addEffect"},
					{From: "addEffect", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "NoiseSignalEffect(") {
		t.Error("Should call effect factory function")
	}
	if !strings.Contains(codeStr, "session.AddEffect(") {
		t.Error("Should call session.AddEffect")
	}
}

func TestGenerate_AddEffectNodeWithParams(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnCollectEffect",
				Event: "CollectEffect",
				Parameters: []Parameter{
					{Name: "playerID", Type: "string"},
					{Name: "duration", Type: "int"},
				},
				Nodes: []Node{
					{
						ID:   "addEffect",
						Type: "AddEffect",
						Inputs: map[string]interface{}{
							"effectId":   map[string]interface{}{"constant": "timed-effect"},
							"effectName": map[string]interface{}{"constant": "TimedEffect"},
							"activator":  "param:playerID",
							"params": map[string]interface{}{
								"duration": "param:duration",
							},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "addEffect"},
					{From: "addEffect", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "TimedEffect(") {
		t.Error("Should call effect factory function with params")
	}
	if !strings.Contains(codeStr, "session.AddEffect(") {
		t.Error("Should call session.AddEffect")
	}
}

func TestGenerate_RemoveEffectNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnClearEffect",
				Event: "ClearEffect",
				Parameters: []Parameter{
					{Name: "effectID", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "removeEffect",
						Type: "RemoveEffect",
						Inputs: map[string]interface{}{
							"effectId": "param:effectID",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "removeEffect"},
					{From: "removeEffect", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "session.RemoveEffect(") {
		t.Error("Should call session.RemoveEffect")
	}
}

func TestGenerate_HasEffectNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnCheckEffect",
				Event: "CheckEffect",
				Parameters: []Parameter{
					{Name: "effectID", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "hasEffect",
						Type: "HasEffect",
						Inputs: map[string]interface{}{
							"effectId": "param:effectID",
						},
					},
					{
						ID:   "checkResult",
						Type: "If",
						Inputs: map[string]interface{}{
							"condition": "node:hasEffect:exists",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "hasEffect"},
					{From: "hasEffect", To: "checkResult"},
					{From: "checkResult", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)
	if !strings.Contains(codeStr, "session.HasEffect(") {
		t.Error("Should call session.HasEffect")
	}
}

// ============================================================================
// Function Node Generation Tests
// ============================================================================

func TestGenerate_FunctionDefinition(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Functions: []FunctionDefinition{
			{
				Name:        "CalculateDamage",
				Description: "Calculate damage based on attack and defense",
				Parameters: []Parameter{
					{Name: "attackPower", Type: "int"},
					{Name: "defense", Type: "int"},
				},
				ReturnType: "int",
				Nodes: []Node{
					{
						ID:   "subtract",
						Type: "Subtract",
						Inputs: map[string]interface{}{
							"a": "param:attackPower",
							"b": "param:defense",
						},
					},
					{
						ID:   "clamp",
						Type: "Max",
						Inputs: map[string]interface{}{
							"a": "node:subtract:result",
							"b": map[string]interface{}{"constant": 0},
						},
					},
					{
						ID:   "return",
						Type: "Return",
						Inputs: map[string]interface{}{
							"value": "node:clamp:result",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "subtract"},
					{From: "subtract", To: "clamp"},
					{From: "clamp", To: "return"},
				},
			},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Nodes: []Node{},
				Flow:  []FlowEdge{},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Check function signature is generated
	if !strings.Contains(codeStr, "func CalculateDamage(") {
		t.Error("Should generate CalculateDamage function")
	}

	// Check parameters are in signature
	if !strings.Contains(codeStr, "attackPower int") {
		t.Error("Should have attackPower parameter")
	}
	if !strings.Contains(codeStr, "defense int") {
		t.Error("Should have defense parameter")
	}

	// Check return type
	if !strings.Contains(codeStr, ") int {") {
		t.Error("Should have int return type")
	}
}

func TestGenerate_CallFunctionNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Functions: []FunctionDefinition{
			{
				Name: "CalculateDamage",
				Parameters: []Parameter{
					{Name: "attackPower", Type: "int"},
					{Name: "defense", Type: "int"},
				},
				ReturnType: "int",
				Nodes:      []Node{},
				Flow:       []FlowEdge{},
			},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnAttack",
				Event: "Attack",
				Parameters: []Parameter{
					{Name: "attackPower", Type: "int"},
					{Name: "defense", Type: "int"},
				},
				Nodes: []Node{
					{
						ID:   "calcDamage",
						Type: "CallFunction",
						Inputs: map[string]interface{}{
							"function": map[string]interface{}{"constant": "CalculateDamage"},
							"args": map[string]interface{}{
								"attackPower": "param:attackPower",
								"defense":     "param:defense",
							},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "calcDamage"},
					{From: "calcDamage", To: "end"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Check function call is generated
	if !strings.Contains(codeStr, "CalculateDamage(state, session,") {
		t.Error("Should call CalculateDamage function with state and session")
	}
}

func TestGenerate_ReturnNode(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Functions: []FunctionDefinition{
			{
				Name:       "GetConstant",
				ReturnType: "int",
				Nodes: []Node{
					{
						ID:   "ret",
						Type: "Return",
						Inputs: map[string]interface{}{
							"value": map[string]interface{}{"constant": 42},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "ret"},
				},
			},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Nodes: []Node{},
				Flow:  []FlowEdge{},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Check return statement is generated
	if !strings.Contains(codeStr, "return 42") {
		t.Error("Should generate return statement with constant value")
	}
}

// ============================================================================
// Validation Tests
// ============================================================================

func TestValidate_MissingRequiredInput(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Nodes: []Node{
					{
						ID:     "compare",
						Type:   "Compare",
						Inputs: map[string]interface{}{
							// Missing "left", "right", "op" which are all required
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "compare"},
					{From: "compare", To: "end"},
				},
			},
		},
	}

	validator := NewValidator(ng)
	err := validator.Validate()

	if err == nil {
		t.Error("Validation should fail for missing required inputs")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "missing required input") {
		t.Errorf("Error should mention missing required input, got: %s", errStr)
	}
}

func TestValidate_DuplicateNodeID(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Nodes: []Node{
					{
						ID:   "node1",
						Type: "Compare",
						Inputs: map[string]interface{}{
							"left":  map[string]interface{}{"constant": 1},
							"right": map[string]interface{}{"constant": 2},
							"op":    map[string]interface{}{"constant": "=="},
						},
					},
					{
						ID:   "node1", // Duplicate!
						Type: "Compare",
						Inputs: map[string]interface{}{
							"left":  map[string]interface{}{"constant": 3},
							"right": map[string]interface{}{"constant": 4},
							"op":    map[string]interface{}{"constant": "=="},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "node1"},
					{From: "node1", To: "end"},
				},
			},
		},
	}

	validator := NewValidator(ng)
	err := validator.Validate()

	if err == nil {
		t.Error("Validation should fail for duplicate node ID")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "duplicate node ID") {
		t.Errorf("Error should mention duplicate node ID, got: %s", errStr)
	}
}

func TestValidate_UnknownFlowTarget(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Nodes: []Node{
					{
						ID:   "node1",
						Type: "Compare",
						Inputs: map[string]interface{}{
							"left":  map[string]interface{}{"constant": 1},
							"right": map[string]interface{}{"constant": 2},
							"op":    map[string]interface{}{"constant": "=="},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "node1"},
					{From: "node1", To: "nonexistent"}, // Unknown node!
				},
			},
		},
	}

	validator := NewValidator(ng)
	err := validator.Validate()

	if err == nil {
		t.Error("Validation should fail for unknown flow target")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "unknown node") {
		t.Errorf("Error should mention unknown node, got: %s", errStr)
	}
}

func TestValidate_ValidGraph(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Nodes: []Node{
					{
						ID:   "compare",
						Type: "Compare",
						Inputs: map[string]interface{}{
							"left":  map[string]interface{}{"constant": 1},
							"right": map[string]interface{}{"constant": 2},
							"op":    map[string]interface{}{"constant": "=="},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "compare"},
					{From: "compare", To: "end"},
				},
			},
		},
	}

	validator := NewValidator(ng)
	err := validator.Validate()

	if err != nil {
		t.Errorf("Valid graph should pass validation, got error: %v", err)
	}
}

func TestValidate_FunctionMissingInput(t *testing.T) {
	ng := &NodeGraph{
		Package: "test",
		Functions: []FunctionDefinition{
			{
				Name:       "TestFunc",
				ReturnType: "int",
				Nodes: []Node{
					{
						ID:     "add",
						Type:   "Add",
						Inputs: map[string]interface{}{
							// Missing "a" and "b"
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "add"},
				},
			},
		},
		Handlers: []EventHandler{
			{Name: "OnTest", Event: "Test"},
		},
	}

	validator := NewValidator(ng)
	err := validator.Validate()

	if err == nil {
		t.Error("Validation should fail for function with missing required inputs")
	}
}

// ============================================================================
// Function Return Value Usage Tests
// ============================================================================

func TestGenerate_FunctionReturnValueUsage(t *testing.T) {
	// Test that function return values can be used by subsequent nodes
	ng := &NodeGraph{
		Package: "test",
		Functions: []FunctionDefinition{
			{
				Name:        "GetValue",
				Description: "Returns a constant value",
				Parameters: []Parameter{
					{Name: "multiplier", Type: "int"},
				},
				ReturnType: "int",
				Nodes: []Node{
					{
						ID:   "multiply",
						Type: "Multiply",
						Inputs: map[string]interface{}{
							"a": map[string]interface{}{"constant": 10},
							"b": "param:multiplier",
						},
					},
					{
						ID:   "return",
						Type: "Return",
						Inputs: map[string]interface{}{
							"value": "node:multiply:result",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "multiply"},
					{From: "multiply", To: "return"},
				},
			},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Nodes: []Node{
					{
						ID:   "callFunc",
						Type: "CallFunction",
						Inputs: map[string]interface{}{
							"function": map[string]interface{}{"constant": "GetValue"},
							"args": map[string]interface{}{
								"multiplier": map[string]interface{}{"constant": 5},
							},
						},
					},
					{
						ID:   "useResult",
						Type: "Add",
						Inputs: map[string]interface{}{
							"a": "node:callFunc:result",
							"b": map[string]interface{}{"constant": 100},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "callFunc"},
					{From: "callFunc", To: "useResult"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Check that the function is called and result is stored
	if !strings.Contains(codeStr, "callFunc_result := GetValue(state, session,") {
		t.Error("Should store function result in callFunc_result variable")
	}

	// Check that the result is used in the Add operation
	if !strings.Contains(codeStr, "callFunc_result") {
		t.Error("Should reference callFunc_result in subsequent node")
	}
}

func TestGenerate_MultipleFunctionCalls(t *testing.T) {
	// Test calling the same function multiple times with different arguments
	ng := &NodeGraph{
		Package: "test",
		Functions: []FunctionDefinition{
			{
				Name:        "Double",
				Description: "Doubles a number",
				Parameters: []Parameter{
					{Name: "value", Type: "int"},
				},
				ReturnType: "int",
				Nodes: []Node{
					{
						ID:   "multiply",
						Type: "Multiply",
						Inputs: map[string]interface{}{
							"a": "param:value",
							"b": map[string]interface{}{"constant": 2},
						},
					},
					{
						ID:   "return",
						Type: "Return",
						Inputs: map[string]interface{}{
							"value": "node:multiply:result",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "multiply"},
					{From: "multiply", To: "return"},
				},
			},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnCompare",
				Event: "Compare",
				Parameters: []Parameter{
					{Name: "a", Type: "int"},
					{Name: "b", Type: "int"},
				},
				Nodes: []Node{
					{
						ID:   "doubleA",
						Type: "CallFunction",
						Inputs: map[string]interface{}{
							"function": map[string]interface{}{"constant": "Double"},
							"args": map[string]interface{}{
								"value": "param:a",
							},
						},
					},
					{
						ID:   "doubleB",
						Type: "CallFunction",
						Inputs: map[string]interface{}{
							"function": map[string]interface{}{"constant": "Double"},
							"args": map[string]interface{}{
								"value": "param:b",
							},
						},
					},
					{
						ID:   "compare",
						Type: "Compare",
						Inputs: map[string]interface{}{
							"left":  "node:doubleA:result",
							"op":    map[string]interface{}{"constant": ">"},
							"right": "node:doubleB:result",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "doubleA"},
					{From: "doubleA", To: "doubleB"},
					{From: "doubleB", To: "compare"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Check that both function calls are generated
	if !strings.Contains(codeStr, "doubleA_result := Double(state, session,") {
		t.Error("Should generate first function call with result")
	}
	if !strings.Contains(codeStr, "doubleB_result := Double(state, session,") {
		t.Error("Should generate second function call with result")
	}

	// Check that comparison uses both results
	if !strings.Contains(codeStr, "doubleA_result") || !strings.Contains(codeStr, "doubleB_result") {
		t.Error("Comparison should use both function results")
	}
}

func TestGenerate_VoidFunction(t *testing.T) {
	// Test function without return type (void)
	ng := &NodeGraph{
		Package: "test",
		Functions: []FunctionDefinition{
			{
				Name:        "LogMessage",
				Description: "Logs a message (no return)",
				Parameters: []Parameter{
					{Name: "message", Type: "string"},
				},
				// No ReturnType - void function
				Nodes: []Node{
					{
						ID:   "log",
						Type: "Constant",
						Inputs: map[string]interface{}{
							"value": "param:message",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "log"},
				},
			},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnTest",
				Event: "Test",
				Nodes: []Node{
					{
						ID:   "callLog",
						Type: "CallFunction",
						Inputs: map[string]interface{}{
							"function": map[string]interface{}{"constant": "LogMessage"},
							"args": map[string]interface{}{
								"message": map[string]interface{}{"constant": "Hello"},
							},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "callLog"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Function signature should not have return type
	if strings.Contains(codeStr, "func LogMessage(") && strings.Contains(codeStr, ") int {") {
		t.Error("Void function should not have return type")
	}

	// Function call should not capture result
	if strings.Contains(codeStr, "callLog_result := LogMessage") {
		t.Error("Void function call should not capture result")
	}

	// Should just call the function
	if !strings.Contains(codeStr, "LogMessage(state, session,") {
		t.Error("Should generate void function call")
	}
}

func TestValidate_ReturnTypeMismatch(t *testing.T) {
	// Test that validation catches when Return node is missing value for non-void function
	ng := &NodeGraph{
		Package: "test",
		Functions: []FunctionDefinition{
			{
				Name:       "GetNumber",
				ReturnType: "int",
				Nodes: []Node{
					{
						ID:     "return",
						Type:   "Return",
						Inputs: map[string]interface{}{
							// Missing "value" - should error for non-void function
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "return"},
				},
			},
		},
		Handlers: []EventHandler{
			{Name: "OnTest", Event: "Test"},
		},
	}

	validator := NewValidator(ng)
	err := validator.Validate()

	if err == nil {
		t.Error("Validation should fail when Return node has no value for non-void function")
	}
}

// ============================================================================
// Card Game Example - Complex Function Usage
// ============================================================================

func TestGenerate_CardGameHighestCardComparison(t *testing.T) {
	// This test demonstrates a realistic use case:
	// - GetPlayerHighestCard(playerId) function returns the highest card value for a player
	// - CompareCards event calls the function twice (for two players)
	// - Compares the results to determine the winner
	ng := &NodeGraph{
		Package: "cardgame",
		Functions: []FunctionDefinition{
			{
				Name:        "GetPlayerHighestCard",
				Description: "Finds the highest card value in a player's hand",
				Parameters: []Parameter{
					{Name: "playerId", Type: "string"},
				},
				ReturnType: "int",
				Nodes: []Node{
					// Get the player by ID
					{
						ID:   "getPlayer",
						Type: "GetPlayer",
						Inputs: map[string]interface{}{
							"playerID": "param:playerId",
						},
					},
					// Get the player's hand (array of card values)
					{
						ID:   "getHand",
						Type: "GetField",
						Inputs: map[string]interface{}{
							"path": map[string]interface{}{"constant": "player.Hand"},
						},
					},
					// Find the maximum value using a simple approach:
					// Initialize max to 0
					{
						ID:   "initMax",
						Type: "Constant",
						Inputs: map[string]interface{}{
							"value": map[string]interface{}{"constant": 0},
						},
					},
					// Return the max value (simplified - in real case would use ForEach with Max)
					{
						ID:   "return",
						Type: "Return",
						Inputs: map[string]interface{}{
							"value": "node:initMax:value",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "getPlayer"},
					{From: "getPlayer", To: "getHand"},
					{From: "getHand", To: "initMax"},
					{From: "initMax", To: "return"},
				},
			},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnCompareHighestCards",
				Event: "CompareHighestCards",
				Parameters: []Parameter{
					{Name: "player1Id", Type: "string"},
					{Name: "player2Id", Type: "string"},
				},
				Nodes: []Node{
					// Call GetPlayerHighestCard for player 1
					{
						ID:   "getCard1",
						Type: "CallFunction",
						Inputs: map[string]interface{}{
							"function": map[string]interface{}{"constant": "GetPlayerHighestCard"},
							"args": map[string]interface{}{
								"playerId": "param:player1Id",
							},
						},
					},
					// Call GetPlayerHighestCard for player 2
					{
						ID:   "getCard2",
						Type: "CallFunction",
						Inputs: map[string]interface{}{
							"function": map[string]interface{}{"constant": "GetPlayerHighestCard"},
							"args": map[string]interface{}{
								"playerId": "param:player2Id",
							},
						},
					},
					// Compare the two highest cards
					{
						ID:   "compare",
						Type: "Compare",
						Inputs: map[string]interface{}{
							"left":  "node:getCard1:result",
							"op":    map[string]interface{}{"constant": ">"},
							"right": "node:getCard2:result",
						},
					},
					// If player1 has higher card
					{
						ID:   "ifPlayer1Wins",
						Type: "If",
						Inputs: map[string]interface{}{
							"condition": "node:compare:result",
						},
					},
					// Emit winner event for player 1
					{
						ID:   "emitPlayer1Wins",
						Type: "EmitToAll",
						Inputs: map[string]interface{}{
							"eventType": map[string]interface{}{"constant": "RoundWinner"},
							"payload": map[string]interface{}{
								"winnerId":    "param:player1Id",
								"winningCard": "node:getCard1:result",
							},
						},
					},
					// Emit winner event for player 2 (else branch)
					{
						ID:   "emitPlayer2Wins",
						Type: "EmitToAll",
						Inputs: map[string]interface{}{
							"eventType": map[string]interface{}{"constant": "RoundWinner"},
							"payload": map[string]interface{}{
								"winnerId":    "param:player2Id",
								"winningCard": "node:getCard2:result",
							},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "getCard1"},
					{From: "getCard1", To: "getCard2"},
					{From: "getCard2", To: "compare"},
					{From: "compare", To: "ifPlayer1Wins"},
					{From: "ifPlayer1Wins", To: "emitPlayer1Wins", Condition: "true"},
					{From: "ifPlayer1Wins", To: "emitPlayer2Wins", Condition: "false"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Verify the function is generated with correct signature
	if !strings.Contains(codeStr, "func GetPlayerHighestCard(") {
		t.Error("Should generate GetPlayerHighestCard function")
	}
	if !strings.Contains(codeStr, "playerId string") {
		t.Error("Function should have playerId parameter")
	}
	if !strings.Contains(codeStr, ") int {") {
		t.Error("Function should return int")
	}

	// Verify both function calls are made in the handler
	if !strings.Contains(codeStr, "getCard1_result := GetPlayerHighestCard(state, session,") {
		t.Error("Should call GetPlayerHighestCard for player 1")
	}
	if !strings.Contains(codeStr, "getCard2_result := GetPlayerHighestCard(state, session,") {
		t.Error("Should call GetPlayerHighestCard for player 2")
	}

	// Verify the comparison uses both results
	if !strings.Contains(codeStr, "getCard1_result") && !strings.Contains(codeStr, "getCard2_result") {
		t.Error("Comparison should reference both function results")
	}

	// Verify conditional branching is generated
	if !strings.Contains(codeStr, "if ") {
		t.Error("Should generate if statement for conditional flow")
	}
}

func TestGenerate_FunctionWithStructReturn(t *testing.T) {
	// Test a function that returns a complex type (Card struct)
	ng := &NodeGraph{
		Package: "cardgame",
		Functions: []FunctionDefinition{
			{
				Name:        "CreateCard",
				Description: "Creates a card with given suit and value",
				Parameters: []Parameter{
					{Name: "suit", Type: "string"},
					{Name: "value", Type: "int"},
				},
				ReturnType: "Card", // Custom struct type
				Nodes: []Node{
					{
						ID:   "createStruct",
						Type: "CreateStruct",
						Inputs: map[string]interface{}{
							"type": map[string]interface{}{"constant": "Card"},
							"fields": map[string]interface{}{
								"Suit":  "param:suit",
								"Value": "param:value",
							},
						},
					},
					{
						ID:   "return",
						Type: "Return",
						Inputs: map[string]interface{}{
							"value": "node:createStruct:result",
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "createStruct"},
					{From: "createStruct", To: "return"},
				},
			},
		},
		Handlers: []EventHandler{
			{
				Name:  "OnDealCard",
				Event: "DealCard",
				Parameters: []Parameter{
					{Name: "playerId", Type: "string"},
				},
				Nodes: []Node{
					{
						ID:   "createCard",
						Type: "CallFunction",
						Inputs: map[string]interface{}{
							"function": map[string]interface{}{"constant": "CreateCard"},
							"args": map[string]interface{}{
								"suit":  map[string]interface{}{"constant": "hearts"},
								"value": map[string]interface{}{"constant": 10},
							},
						},
					},
				},
				Flow: []FlowEdge{
					{From: "start", To: "createCard"},
				},
			},
		},
	}

	gen := NewCodeGenerator(ng, nil)
	code, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	codeStr := string(code)

	// Function should return Card type
	if !strings.Contains(codeStr, ") Card {") {
		t.Error("Function should return Card type")
	}

	// Handler should capture result as Card type
	if !strings.Contains(codeStr, "createCard_result := CreateCard(state, session,") {
		t.Error("Should call CreateCard and capture result")
	}
}
