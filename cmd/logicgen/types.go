package main

import "fmt"

// NodeGraph represents the complete node-based logic definition
type NodeGraph struct {
	Version  string         `json:"version"`
	Package  string         `json:"package"`
	Imports  []string       `json:"imports"`
	Handlers []EventHandler `json:"handlers"`
}

// EventHandler represents a handler for a specific event
type EventHandler struct {
	Name       string      `json:"name"`
	Event      string      `json:"event"`
	Parameters []Parameter `json:"parameters"`
	Nodes      []Node      `json:"nodes"`
	Flow       []FlowEdge  `json:"flow"`
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
	NodeGetField       NodeType = "GetField"
	NodeSetField       NodeType = "SetField"
	NodeGetPlayer      NodeType = "GetPlayer"
	NodeGetCurrentState NodeType = "GetCurrentState"

	// Array operations
	NodeAddToArray      NodeType = "AddToArray"
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

	// Effects
	NodeAddEffect    NodeType = "AddEffect"
	NodeRemoveEffect NodeType = "RemoveEffect"
	NodeHasEffect    NodeType = "HasEffect"
)

// NodeDefinition describes the inputs, outputs, and behavior of a node type
type NodeDefinition struct {
	Type        NodeType
	Category    string
	Description string
	Inputs      []PortDefinition
	Outputs     []PortDefinition
	Generator   func(*CodeGenerator, *Node) error
}

// PortDefinition describes an input or output port
type PortDefinition struct {
	Name     string
	Type     string
	Required bool
	Default  interface{}
}

// GetNodeDefinition returns the definition for a node type
func GetNodeDefinition(nodeType string) (*NodeDefinition, error) {
	def, ok := nodeDefinitions[NodeType(nodeType)]
	if !ok {
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}
	return def, nil
}

// nodeDefinitions maps node types to their definitions
var nodeDefinitions = map[NodeType]*NodeDefinition{
	NodeGetField: {
		Type:        NodeGetField,
		Category:    "State",
		Description: "Gets a field value from the state",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "value", Type: "any", Required: true},
		},
	},
	NodeSetField: {
		Type:        NodeSetField,
		Category:    "State",
		Description: "Sets a field value in the state",
		Inputs: []PortDefinition{
			{Name: "path", Type: "string", Required: true},
			{Name: "value", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{},
	},
	NodeGetPlayer: {
		Type:        NodeGetPlayer,
		Category:    "State",
		Description: "Gets a player by ID",
		Inputs: []PortDefinition{
			{Name: "playerID", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "player", Type: "*Player", Required: true},
		},
	},
	NodeAddToArray: {
		Type:        NodeAddToArray,
		Category:    "Array",
		Description: "Adds an element to an array",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
			{Name: "element", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]any", Required: true},
		},
	},
	NodeRemoveFromArray: {
		Type:        NodeRemoveFromArray,
		Category:    "Array",
		Description: "Removes elements from an array",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
			{Name: "index", Type: "int", Required: false},
			{Name: "predicate", Type: "func(any) bool", Required: false},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]any", Required: true},
		},
	},
	NodeFilterArray: {
		Type:        NodeFilterArray,
		Category:    "Array",
		Description: "Filters an array based on a predicate",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
			{Name: "predicate", Type: "func(any) bool", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]any", Required: true},
		},
	},
	NodeCompare: {
		Type:        NodeCompare,
		Category:    "Logic",
		Description: "Compares two values",
		Inputs: []PortDefinition{
			{Name: "left", Type: "any", Required: true},
			{Name: "op", Type: "string", Required: true},
			{Name: "right", Type: "any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool", Required: true},
		},
	},
	NodeIf: {
		Type:        NodeIf,
		Category:    "Control Flow",
		Description: "Conditional branch",
		Inputs: []PortDefinition{
			{Name: "condition", Type: "bool", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "true", Type: "flow", Required: false},
			{Name: "false", Type: "flow", Required: false},
		},
	},
	NodeForEach: {
		Type:        NodeForEach,
		Category:    "Control Flow",
		Description: "Iterates over array elements",
		Inputs: []PortDefinition{
			{Name: "array", Type: "[]any", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "item", Type: "any", Required: true},
			{Name: "index", Type: "int", Required: true},
		},
	},
	NodeEmitToAll: {
		Type:        NodeEmitToAll,
		Category:    "Events",
		Description: "Emits an event to all players",
		Inputs: []PortDefinition{
			{Name: "eventType", Type: "string", Required: true},
			{Name: "payload", Type: "map[string]any", Required: false},
		},
		Outputs: []PortDefinition{},
	},
	NodeEmitToPlayer: {
		Type:        NodeEmitToPlayer,
		Category:    "Events",
		Description: "Emits an event to a specific player",
		Inputs: []PortDefinition{
			{Name: "playerID", Type: "string", Required: true},
			{Name: "eventType", Type: "string", Required: true},
			{Name: "payload", Type: "map[string]any", Required: false},
		},
		Outputs: []PortDefinition{},
	},
	NodeAdd: {
		Type:        NodeAdd,
		Category:    "Math",
		Description: "Adds two numbers",
		Inputs: []PortDefinition{
			{Name: "a", Type: "number", Required: true},
			{Name: "b", Type: "number", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "number", Required: true},
		},
	},
	NodeConcat: {
		Type:        NodeConcat,
		Category:    "String",
		Description: "Concatenates strings",
		Inputs: []PortDefinition{
			{Name: "strings", Type: "[]string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "string", Required: true},
		},
	},
}
