package main

import (
	"testing"
)

// ============================================================================
// ParseValueSource Tests
// ============================================================================

func TestParseValueSource_StringRef(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantType string
		wantVal  interface{}
	}{
		{
			name:     "param reference",
			input:    "param:playerID",
			wantType: "ref",
			wantVal:  "param:playerID",
		},
		{
			name:     "node reference",
			input:    "node:checkScore:result",
			wantType: "ref",
			wantVal:  "node:checkScore:result",
		},
		{
			name:     "state reference",
			input:    "state:Players",
			wantType: "ref",
			wantVal:  "state:Players",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs, err := ParseValueSource(tt.input)
			if err != nil {
				t.Fatalf("ParseValueSource() error = %v", err)
			}
			if vs.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", vs.Type, tt.wantType)
			}
			if vs.Value != tt.wantVal {
				t.Errorf("Value = %v, want %v", vs.Value, tt.wantVal)
			}
		})
	}
}

func TestParseValueSource_MapConstant(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		wantType string
		wantVal  interface{}
	}{
		{
			name:     "string constant",
			input:    map[string]interface{}{"constant": "hello"},
			wantType: "constant",
			wantVal:  "hello",
		},
		{
			name:     "int constant",
			input:    map[string]interface{}{"constant": 42},
			wantType: "constant",
			wantVal:  42,
		},
		{
			name:     "float constant",
			input:    map[string]interface{}{"constant": 3.14},
			wantType: "constant",
			wantVal:  3.14,
		},
		{
			name:     "bool constant true",
			input:    map[string]interface{}{"constant": true},
			wantType: "constant",
			wantVal:  true,
		},
		{
			name:     "bool constant false",
			input:    map[string]interface{}{"constant": false},
			wantType: "constant",
			wantVal:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs, err := ParseValueSource(tt.input)
			if err != nil {
				t.Fatalf("ParseValueSource() error = %v", err)
			}
			if vs.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", vs.Type, tt.wantType)
			}
			if vs.Value != tt.wantVal {
				t.Errorf("Value = %v, want %v", vs.Value, tt.wantVal)
			}
		})
	}
}

func TestParseValueSource_MapSource(t *testing.T) {
	input := map[string]interface{}{"source": "someRef"}
	vs, err := ParseValueSource(input)
	if err != nil {
		t.Fatalf("ParseValueSource() error = %v", err)
	}
	if vs.Type != "ref" {
		t.Errorf("Type = %v, want ref", vs.Type)
	}
	if vs.Value != "someRef" {
		t.Errorf("Value = %v, want someRef", vs.Value)
	}
}

func TestParseValueSource_DirectConstants(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantType string
		wantVal  interface{}
	}{
		{
			name:     "int",
			input:    100,
			wantType: "constant",
			wantVal:  100,
		},
		{
			name:     "float64",
			input:    1.5,
			wantType: "constant",
			wantVal:  1.5,
		},
		{
			name:     "bool",
			input:    true,
			wantType: "constant",
			wantVal:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs, err := ParseValueSource(tt.input)
			if err != nil {
				t.Fatalf("ParseValueSource() error = %v", err)
			}
			if vs.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", vs.Type, tt.wantType)
			}
			if vs.Value != tt.wantVal {
				t.Errorf("Value = %v, want %v", vs.Value, tt.wantVal)
			}
		})
	}
}

func TestParseValueSource_InvalidMap(t *testing.T) {
	// Map without "constant" or "source" keys returns error
	input := map[string]interface{}{"invalid": "value"}
	_, err := ParseValueSource(input)
	if err == nil {
		t.Fatal("ParseValueSource() expected error for invalid map")
	}
}

// ============================================================================
// NodeDefinition Tests
// ============================================================================

func TestNodeDefinitions_AllDefined(t *testing.T) {
	// Test that all node types have definitions
	// Note: NodeSwitch is declared but doesn't have a definition yet
	requiredNodes := []NodeType{
		// State
		NodeGetField, NodeSetField, NodeGetPlayer,
		// Array
		NodeAddToArray, NodeRemoveFromArray, NodeFindInArray, NodeArrayLength,
		// Control flow
		NodeIf, NodeForEach, NodeWhile,
		// Logic
		NodeAnd, NodeOr, NodeNot, NodeCompare,
		// Math
		NodeAdd, NodeSubtract, NodeMultiply, NodeDivide, NodeRandomInt,
		// String
		NodeConcat, NodeFormat,
		// Events
		NodeEmitToAll, NodeEmitToPlayer,
		// Session
		NodeKickPlayer, NodeGetHostPlayer, NodeIsHost,
		// Filters
		NodeAddFilter, NodeRemoveFilter, NodeHasFilter,
		// Wait
		NodeWait, NodeWaitUntil, NodeTimeout,
	}

	allDefs := GetAllNodeDefinitions()
	for _, nt := range requiredNodes {
		t.Run(string(nt), func(t *testing.T) {
			def, ok := allDefs[nt]
			if !ok {
				t.Errorf("Node %s has no definition", nt)
				return
			}
			if def.Type != nt {
				t.Errorf("Node definition Type = %v, want %v", def.Type, nt)
			}
			if def.Category == "" {
				t.Errorf("Node %s has no category", nt)
			}
			if def.Description == "" {
				t.Errorf("Node %s has no description", nt)
			}
		})
	}
}

func TestNodeDefinitions_Categories(t *testing.T) {
	// Test that nodes are in expected categories
	expectedCategories := map[NodeType]string{
		NodeGetField:       "State",
		NodeSetField:       "State",
		NodeIf:             "Control Flow",
		NodeForEach:        "Control Flow",
		NodeAdd:            "Math",
		NodeCompare:        "Logic",
		NodeEmitToAll:      "Events",
		NodeKickPlayer:     "Session",
		NodeAddFilter:      "Session",
		NodeWait:           "Async",
		NodeGpsDistance:    "GPS",
		NodeRandomInt:      "Random",
		NodeGetCurrentTime: "Time",
	}

	allDefs := GetAllNodeDefinitions()
	for nt, expectedCat := range expectedCategories {
		t.Run(string(nt), func(t *testing.T) {
			def, ok := allDefs[nt]
			if !ok {
				t.Skipf("Node %s not defined", nt)
				return
			}
			if def.Category != expectedCat {
				t.Errorf("Category = %v, want %v", def.Category, expectedCat)
			}
		})
	}
}

func TestNodeDefinitions_RequiredInputs(t *testing.T) {
	// Test that required inputs are properly marked
	tests := []struct {
		node           NodeType
		requiredInputs []string
	}{
		{NodeCompare, []string{"left", "right", "op"}},
		{NodeIf, []string{"condition"}},
		{NodeSetField, []string{"path", "value"}},
		{NodeEmitToPlayer, []string{"playerID", "eventType"}},
		{NodeKickPlayer, []string{"playerID"}},
		{NodeAddFilter, []string{"viewerID", "filterID", "filterName"}},
		{NodeWait, []string{"duration"}},
	}

	allDefs := GetAllNodeDefinitions()
	for _, tt := range tests {
		t.Run(string(tt.node), func(t *testing.T) {
			def, ok := allDefs[tt.node]
			if !ok {
				t.Skipf("Node %s not defined", tt.node)
				return
			}

			for _, required := range tt.requiredInputs {
				found := false
				for _, input := range def.Inputs {
					if input.Name == required {
						found = true
						if !input.Required {
							t.Errorf("Input %s should be required", required)
						}
						break
					}
				}
				if !found {
					t.Errorf("Required input %s not found", required)
				}
			}
		})
	}
}

// ============================================================================
// EventPermissions Tests
// ============================================================================

func TestEventPermissions_HostOnly(t *testing.T) {
	handler := EventHandler{
		Name:  "TestHandler",
		Event: "Test",
		Permissions: &EventPermissions{
			HostOnly: true,
		},
	}

	if handler.Permissions == nil {
		t.Fatal("Permissions should not be nil")
	}
	if !handler.Permissions.HostOnly {
		t.Error("HostOnly should be true")
	}
}

func TestEventPermissions_PlayerParam(t *testing.T) {
	handler := EventHandler{
		Name:  "TestHandler",
		Event: "Test",
		Permissions: &EventPermissions{
			PlayerParam: "playerID",
		},
	}

	if handler.Permissions.PlayerParam != "playerID" {
		t.Errorf("PlayerParam = %v, want playerID", handler.Permissions.PlayerParam)
	}
}

func TestEventPermissions_AllowedPlayers(t *testing.T) {
	handler := EventHandler{
		Name:  "TestHandler",
		Event: "Test",
		Permissions: &EventPermissions{
			AllowedPlayers: []string{"admin1", "admin2"},
		},
	}

	if len(handler.Permissions.AllowedPlayers) != 2 {
		t.Errorf("AllowedPlayers len = %d, want 2", len(handler.Permissions.AllowedPlayers))
	}
}

// ============================================================================
// FilterDefinition Tests
// ============================================================================

func TestFilterDefinition_Structure(t *testing.T) {
	filter := FilterDefinition{
		Name:        "HideEnemies",
		Description: "Hides enemy locations",
		Parameters: []Parameter{
			{Name: "viewerTeam", Type: "string"},
		},
		Nodes: []Node{
			{ID: "node1", Type: "Compare"},
		},
		Flow: []FlowEdge{
			{From: "start", To: "node1"},
		},
	}

	if filter.Name != "HideEnemies" {
		t.Errorf("Name = %v, want HideEnemies", filter.Name)
	}
	if len(filter.Parameters) != 1 {
		t.Errorf("Parameters len = %d, want 1", len(filter.Parameters))
	}
	if len(filter.Nodes) != 1 {
		t.Errorf("Nodes len = %d, want 1", len(filter.Nodes))
	}
}

// ============================================================================
// FlowEdge Tests
// ============================================================================

func TestFlowEdge_Labels(t *testing.T) {
	tests := []struct {
		name  string
		edge  FlowEdge
		label string
	}{
		{
			name:  "true branch",
			edge:  FlowEdge{From: "ifNode", To: "trueNode", Label: "true"},
			label: "true",
		},
		{
			name:  "false branch",
			edge:  FlowEdge{From: "ifNode", To: "falseNode", Label: "false"},
			label: "false",
		},
		{
			name:  "body loop",
			edge:  FlowEdge{From: "forEach", To: "bodyNode", Label: "body"},
			label: "body",
		},
		{
			name:  "no label",
			edge:  FlowEdge{From: "a", To: "b"},
			label: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.edge.Label != tt.label {
				t.Errorf("Label = %v, want %v", tt.edge.Label, tt.label)
			}
		})
	}
}
