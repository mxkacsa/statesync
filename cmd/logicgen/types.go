package main

import "fmt"

// NodeGraph represents the complete node-based logic definition
type NodeGraph struct {
	Version   string               `json:"version"`
	Package   string               `json:"package"`
	Imports   []string             `json:"imports"`
	Filters   []FilterDefinition   `json:"filters,omitempty"`
	Functions []FunctionDefinition `json:"functions,omitempty"`
	Handlers  []EventHandler       `json:"handlers"`
}

// FilterDefinition defines a filter that transforms state for viewers
type FilterDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  []Parameter `json:"parameters"`
	Nodes       []Node      `json:"nodes"`
	Flow        []FlowEdge  `json:"flow"`
}

// FunctionDefinition defines a reusable node-based function that can be called from other nodes.
// Functions allow creating reusable logic that accepts parameters and returns a value.
//
// Example usage in JSON:
//
//	{
//	    "functions": [{
//	        "name": "CalculateDamage",
//	        "description": "Calculate damage based on attacker and defender stats",
//	        "parameters": [
//	            {"name": "attackPower", "type": "int"},
//	            {"name": "defense", "type": "int"}
//	        ],
//	        "returnType": "int",
//	        "nodes": [...],
//	        "flow": [...]
//	    }]
//	}
//
// Then call it from a handler:
//
//	{
//	    "id": "calcDmg",
//	    "type": "CallFunction",
//	    "inputs": {
//	        "function": "CalculateDamage",
//	        "args": {
//	            "attackPower": "node:getAttacker:power",
//	            "defense": "node:getDefender:defense"
//	        }
//	    }
//	}
type FunctionDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  []Parameter `json:"parameters"`
	ReturnType  string      `json:"returnType,omitempty"` // Optional return type (e.g., "int", "string", "bool")
	Nodes       []Node      `json:"nodes"`
	Flow        []FlowEdge  `json:"flow"`
}

// EventHandler represents a handler for a specific event
type EventHandler struct {
	Name        string            `json:"name"`
	Event       string            `json:"event"`
	Permissions *EventPermissions `json:"permissions,omitempty"`
	Parameters  []Parameter       `json:"parameters"`
	Nodes       []Node            `json:"nodes"`
	Flow        []FlowEdge        `json:"flow"`
}

// EventPermissions defines who can trigger an event
type EventPermissions struct {
	// HostOnly - only the session host can trigger this event
	HostOnly bool `json:"hostOnly,omitempty"`

	// PlayerParam - the named parameter must match senderID
	// e.g., "playerID" means the playerID param must equal senderID
	PlayerParam string `json:"playerParam,omitempty"`

	// AllowedPlayers - static list of player IDs that can trigger
	AllowedPlayers []string `json:"allowedPlayers,omitempty"`
}

// Parameter represents an input parameter to an event handler
type Parameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Node represents a single node in the graph
type Node struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Inputs  map[string]interface{} `json:"inputs"`
	Outputs map[string]string      `json:"outputs"`
}

// FlowEdge represents execution flow between nodes
type FlowEdge struct {
	From      string            `json:"from"`
	To        string            `json:"to"`
	Condition string            `json:"condition,omitempty"` // For conditional flow
	Label     string            `json:"label,omitempty"`     // For switch cases
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ValueSource represents where a value comes from
type ValueSource struct {
	Type  string      // "param", "node", "constant", "variable"
	Value interface{} // The actual value or reference
}

// ParseValueSource parses a value source from interface{}
func ParseValueSource(v interface{}) (*ValueSource, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		if source, ok := val["source"].(string); ok {
			return &ValueSource{Type: "ref", Value: source}, nil
		}
		if constant, ok := val["constant"]; ok {
			return &ValueSource{Type: "constant", Value: constant}, nil
		}
	case string:
		return &ValueSource{Type: "ref", Value: val}, nil
	default:
		return &ValueSource{Type: "constant", Value: val}, nil
	}
	return nil, fmt.Errorf("invalid value source: %v", v)
}

// NodeType represents the type of a node
type NodeType string

const (
	// Event nodes
	NodeOnPlayerConnect    NodeType = "OnPlayerConnect"
	NodeOnPlayerDisconnect NodeType = "OnPlayerDisconnect"
	NodeOnCustomEvent      NodeType = "OnCustomEvent"
	NodeOnTick             NodeType = "OnTick"

	// State access
	NodeGetField        NodeType = "GetField"
	NodeSetField        NodeType = "SetField"
	NodeGetPlayer       NodeType = "GetPlayer"
	NodeGetCurrentState NodeType = "GetCurrentState"

	// Array operations
	NodeAddToArray      NodeType = "AddToArray"
	NodeArrayAppend     NodeType = "ArrayAppend" // Alias for AddToArray with path
	NodeRemoveFromArray NodeType = "RemoveFromArray"
	NodeFilterArray     NodeType = "FilterArray"
	NodeMapArray        NodeType = "MapArray"
	NodeFindInArray     NodeType = "FindInArray"
	NodeArrayLength     NodeType = "ArrayLength"
	NodeArrayAt         NodeType = "ArrayAt"

	// Map operations
	NodeSetMapValue  NodeType = "SetMapValue"
	NodeGetMapValue  NodeType = "GetMapValue"
	NodeRemoveMapKey NodeType = "RemoveMapKey"
	NodeHasMapKey    NodeType = "HasMapKey"
	NodeMapKeys      NodeType = "MapKeys"
	NodeMapValues    NodeType = "MapValues"
	NodeFilterMap    NodeType = "FilterMap"

	// Control flow
	NodeIf       NodeType = "If"
	NodeSwitch   NodeType = "Switch"
	NodeForEach  NodeType = "ForEach"
	NodeWhile    NodeType = "While"
	NodeBreak    NodeType = "Break"
	NodeContinue NodeType = "Continue"

	// Logic
	NodeAnd     NodeType = "And"
	NodeOr      NodeType = "Or"
	NodeNot     NodeType = "Not"
	NodeCompare NodeType = "Compare"
	NodeIsNull  NodeType = "IsNull"
	NodeIsEmpty NodeType = "IsEmpty"

	// Math
	NodeAdd      NodeType = "Add"
	NodeSubtract NodeType = "Subtract"
	NodeMultiply NodeType = "Multiply"
	NodeDivide   NodeType = "Divide"
	NodeModulo   NodeType = "Modulo"
	NodeMin      NodeType = "Min"
	NodeMax      NodeType = "Max"
	NodeRandom   NodeType = "Random"
	NodeRound    NodeType = "Round"
	NodeFloor    NodeType = "Floor"
	NodeCeil     NodeType = "Ceil"
	NodeSqrt     NodeType = "Sqrt"
	NodePow      NodeType = "Pow"
	NodeAbs      NodeType = "Abs"
	NodeSin      NodeType = "Sin"
	NodeCos      NodeType = "Cos"
	NodeAtan2    NodeType = "Atan2"

	// GPS/Geometry
	NodeGpsDistance    NodeType = "GpsDistance"    // Distance between two GPS points in meters
	NodeGpsMoveToward  NodeType = "GpsMoveToward"  // Move from point A toward point B by distance
	NodePointInCircle  NodeType = "PointInCircle"  // Check if point is within circle radius
	NodePointInPolygon NodeType = "PointInPolygon" // Check if point is inside polygon

	// Random
	NodeRandomInt   NodeType = "RandomInt"   // Random integer in range [min, max]
	NodeRandomFloat NodeType = "RandomFloat" // Random float in range [min, max)
	NodeRandomBool  NodeType = "RandomBool"  // Random boolean with probability

	// Time
	NodeGetCurrentTime NodeType = "GetCurrentTime" // Get current Unix timestamp
	NodeTimeSince      NodeType = "TimeSince"      // Time elapsed since timestamp

	// Wait/Async (requires context)
	NodeWait      NodeType = "Wait"      // Pause execution for duration
	NodeWaitUntil NodeType = "WaitUntil" // Wait until condition is true
	NodeTimeout   NodeType = "Timeout"   // Execute with timeout

	// Object creation
	NodeCreateStruct   NodeType = "CreateStruct"   // Create a new struct with field values
	NodeUpdateStruct   NodeType = "UpdateStruct"   // Update specific fields on a struct
	NodeGetStructField NodeType = "GetStructField" // Get a field value from a struct

	// String
	NodeConcat   NodeType = "Concat"
	NodeFormat   NodeType = "Format"
	NodeContains NodeType = "Contains"
	NodeSplit    NodeType = "Split"
	NodeToUpper  NodeType = "ToUpper"
	NodeToLower  NodeType = "ToLower"
	NodeTrim     NodeType = "Trim"

	// Variables
	NodeGetVariable NodeType = "GetVariable"
	NodeSetVariable NodeType = "SetVariable"
	NodeConstant    NodeType = "Constant"

	// Functions
	NodeCallFunction NodeType = "CallFunction"
	NodeReturn       NodeType = "Return"

	// Events
	NodeEmitEvent    NodeType = "EmitEvent"
	NodeEmitToPlayer NodeType = "EmitToPlayer"
	NodeEmitToAll    NodeType = "EmitToAll"
	NodeEmitExcept   NodeType = "EmitExcept"
	NodeEmitToMany   NodeType = "EmitToMany"

	// Session management
	NodeKickPlayer    NodeType = "KickPlayer"    // Kick a player from session
	NodeGetHostPlayer NodeType = "GetHostPlayer" // Get the host player ID
	NodeIsHost        NodeType = "IsHost"        // Check if player is host

	// Filters
	NodeAddFilter    NodeType = "AddFilter"    // Add a filter for a viewer
	NodeRemoveFilter NodeType = "RemoveFilter" // Remove a filter from a viewer
	NodeHasFilter    NodeType = "HasFilter"    // Check if a filter exists

	// Effects
	NodeAddEffect    NodeType = "AddEffect"
	NodeRemoveEffect NodeType = "RemoveEffect"
	NodeHasEffect    NodeType = "HasEffect"

	// Batch operations
	NodeForEachWhere NodeType = "ForEachWhere" // Iterate + filter in one: forEach where condition
	NodeUpdateWhere  NodeType = "UpdateWhere"  // Update all matching: array.where(cond).set(field, value)
	NodeFindWhere    NodeType = "FindWhere"    // Find first matching
	NodeCountWhere   NodeType = "CountWhere"   // Count matching elements
)

// NodeDefinition describes the inputs, outputs, and behavior of a node type
type NodeDefinition struct {
	Type        NodeType
	Category    string
	Description string
	Inputs      []PortDefinition
	Outputs     []PortDefinition
	// Generator is called during code generation to produce the node's code.
	// For built-in nodes, this is typically nil and a switch statement is used.
	// For custom nodes registered via RegisterNode(), this function is required.
	Generator func(*GeneratorContext, *Node) error
}

// PortDefinition describes an input or output port
type PortDefinition struct {
	Name     string
	Type     string
	Required bool
	Default  interface{}
}

// GetNodeDefinition returns the definition for a node type.
// All node definitions are now registered via the registry system in nodes_*.go files.
func GetNodeDefinition(nodeType string) (*NodeDefinition, error) {
	allDefs := GetAllNodeDefinitions()
	if def, ok := allDefs[NodeType(nodeType)]; ok {
		return def, nil
	}
	return nil, fmt.Errorf("unknown node type: %s", nodeType)
}

// nodeDefinitions is now empty - all nodes are registered via the registry system.
// See nodes_core.go, nodes_logic.go, nodes_events.go, nodes_async.go, nodes_gps.go
var nodeDefinitions = map[NodeType]*NodeDefinition{}
