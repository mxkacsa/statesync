package main

import (
	"fmt"
	"strings"
)

// ============================================================================
// Event and Session Nodes - Event emission and session management
// ============================================================================

func init() {
	registerEventNodes()
	registerSessionNodes()
	registerFilterNodes()
}

// ============================================================================
// Event Nodes
// ============================================================================

func registerEventNodes() {
	// EmitToAll - Emits an event to all players
	MustRegisterNode(NodeDefinition{
		Type:        NodeEmitToAll,
		Category:    "Events",
		Description: "Emits an event to all players",
		Inputs: []PortDefinition{
			{Name: "eventType", Type: "string", Required: true},
			{Name: "payload", Type: "map[string]any", Required: false},
		},
		Outputs:   []PortDefinition{},
		Generator: generateEmitToAllNode,
	})

	// EmitToPlayer - Emits an event to a specific player
	MustRegisterNode(NodeDefinition{
		Type:        NodeEmitToPlayer,
		Category:    "Events",
		Description: "Emits an event to a specific player",
		Inputs: []PortDefinition{
			{Name: "playerID", Type: "string", Required: true},
			{Name: "eventType", Type: "string", Required: true},
			{Name: "payload", Type: "map[string]any", Required: false},
		},
		Outputs:   []PortDefinition{},
		Generator: generateEmitToPlayerNode,
	})

	// EmitToMany - Emits an event to multiple players
	MustRegisterNode(NodeDefinition{
		Type:        NodeEmitToMany,
		Category:    "Events",
		Description: "Emits an event to multiple players",
		Inputs: []PortDefinition{
			{Name: "playerIDs", Type: "[]string", Required: true},
			{Name: "eventType", Type: "string", Required: true},
			{Name: "payload", Type: "map[string]any", Required: false},
		},
		Outputs:   []PortDefinition{},
		Generator: generateEmitToManyNode,
	})
}

// ============================================================================
// Session Nodes
// ============================================================================

func registerSessionNodes() {
	// KickPlayer - Kicks a player from the session
	MustRegisterNode(NodeDefinition{
		Type:        NodeKickPlayer,
		Category:    "Session",
		Description: "Kicks a player from the session",
		Inputs: []PortDefinition{
			{Name: "playerID", Type: "string", Required: true},
			{Name: "reason", Type: "string", Required: false, Default: ""},
		},
		Outputs: []PortDefinition{
			{Name: "kicked", Type: "bool", Required: true},
		},
		Generator: generateKickPlayerNode,
	})

	// GetHostPlayer - Gets the current host player ID
	MustRegisterNode(NodeDefinition{
		Type:        NodeGetHostPlayer,
		Category:    "Session",
		Description: "Gets the current host player ID",
		Inputs:      []PortDefinition{},
		Outputs: []PortDefinition{
			{Name: "hostPlayerID", Type: "string", Required: true},
		},
		Generator: generateGetHostPlayerNode,
	})

	// IsHost - Checks if a player is the host
	MustRegisterNode(NodeDefinition{
		Type:        NodeIsHost,
		Category:    "Session",
		Description: "Checks if a player is the host",
		Inputs: []PortDefinition{
			{Name: "playerID", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "isHost", Type: "bool", Required: true},
		},
		Generator: generateIsHostNode,
	})
}

// ============================================================================
// Filter Nodes
// ============================================================================

func registerFilterNodes() {
	// AddFilter - Adds a filter for a viewer
	MustRegisterNode(NodeDefinition{
		Type:        NodeAddFilter,
		Category:    "Session",
		Description: "Adds a filter for a viewer",
		Inputs: []PortDefinition{
			{Name: "viewerID", Type: "string", Required: true},
			{Name: "filterID", Type: "string", Required: true},
			{Name: "filterName", Type: "string", Required: true},
			{Name: "params", Type: "map[string]any", Required: false},
		},
		Outputs:   []PortDefinition{},
		Generator: generateAddFilterNode,
	})

	// RemoveFilter - Removes a filter from a viewer
	MustRegisterNode(NodeDefinition{
		Type:        NodeRemoveFilter,
		Category:    "Session",
		Description: "Removes a filter from a viewer",
		Inputs: []PortDefinition{
			{Name: "viewerID", Type: "string", Required: true},
			{Name: "filterID", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "removed", Type: "bool", Required: true},
		},
		Generator: generateRemoveFilterNode,
	})

	// HasFilter - Checks if a filter exists for a viewer
	MustRegisterNode(NodeDefinition{
		Type:        NodeHasFilter,
		Category:    "Session",
		Description: "Checks if a filter exists for a viewer",
		Inputs: []PortDefinition{
			{Name: "viewerID", Type: "string", Required: true},
			{Name: "filterID", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "exists", Type: "bool", Required: true},
		},
		Generator: generateHasFilterNode,
	})
}

// ============================================================================
// Generator Functions - Events
// ============================================================================

func generateEmitToAllNode(ctx *GeneratorContext, node *Node) error {
	eventTypeSrc, err := ctx.ResolveInput(node, "eventType")
	if err != nil {
		return err
	}

	if _, hasPayload := node.Inputs["payload"]; hasPayload {
		payloadSrc, _ := ctx.ResolveInput(node, "payload")
		ctx.WriteLine("session.Emit(%s, %s)", eventTypeSrc, payloadSrc)
	} else {
		ctx.WriteLine("session.Emit(%s, nil)", eventTypeSrc)
	}

	return nil
}

func generateEmitToPlayerNode(ctx *GeneratorContext, node *Node) error {
	playerIDSrc, err := ctx.ResolveInput(node, "playerID")
	if err != nil {
		return err
	}
	eventTypeSrc, err := ctx.ResolveInput(node, "eventType")
	if err != nil {
		return err
	}

	if _, hasPayload := node.Inputs["payload"]; hasPayload {
		payloadSrc, _ := ctx.ResolveInput(node, "payload")
		ctx.WriteLine("session.EmitTo(%s, %s, %s)", playerIDSrc, eventTypeSrc, payloadSrc)
	} else {
		ctx.WriteLine("session.EmitTo(%s, %s, nil)", playerIDSrc, eventTypeSrc)
	}

	return nil
}

func generateEmitToManyNode(ctx *GeneratorContext, node *Node) error {
	playerIDsSrc, err := ctx.ResolveInput(node, "playerIDs")
	if err != nil {
		return err
	}
	eventTypeSrc, err := ctx.ResolveInput(node, "eventType")
	if err != nil {
		return err
	}

	if _, hasPayload := node.Inputs["payload"]; hasPayload {
		payloadSrc, _ := ctx.ResolveInput(node, "payload")
		ctx.WriteLine("session.EmitToMany(%s, %s, %s)", playerIDsSrc, eventTypeSrc, payloadSrc)
	} else {
		ctx.WriteLine("session.EmitToMany(%s, %s, nil)", playerIDsSrc, eventTypeSrc)
	}

	return nil
}

// ============================================================================
// Generator Functions - Session
// ============================================================================

func generateKickPlayerNode(ctx *GeneratorContext, node *Node) error {
	playerID, err := ctx.ResolveInput(node, "playerID")
	if err != nil {
		return err
	}

	reason := `""`
	if reasonVal, ok := node.Inputs["reason"]; ok {
		if resolved, err := ctx.ResolveInput(node, "reason"); err == nil {
			reason = resolved
		} else if reasonMap, ok := reasonVal.(map[string]interface{}); ok {
			if constVal, ok := reasonMap["constant"]; ok {
				reason = fmt.Sprintf(`"%v"`, constVal)
			}
		}
	}

	kickedVar := ctx.AllocateVariable(node, "kicked", "bool")
	ctx.WriteLine("%s := session.Kick(%s, %s)", kickedVar, playerID, reason)

	return nil
}

func generateGetHostPlayerNode(ctx *GeneratorContext, node *Node) error {
	hostVar := ctx.AllocateVariable(node, "hostPlayerID", "string")
	ctx.WriteLine("%s := session.HostPlayerID()", hostVar)
	return nil
}

func generateIsHostNode(ctx *GeneratorContext, node *Node) error {
	playerID, err := ctx.ResolveInput(node, "playerID")
	if err != nil {
		return err
	}

	isHostVar := ctx.AllocateVariable(node, "isHost", "bool")
	ctx.WriteLine("%s := session.IsHost(%s)", isHostVar, playerID)

	return nil
}

// ============================================================================
// Generator Functions - Filters
// ============================================================================

func generateAddFilterNode(ctx *GeneratorContext, node *Node) error {
	viewerID, err := ctx.ResolveInput(node, "viewerID")
	if err != nil {
		return err
	}

	filterID, err := ctx.ResolveInput(node, "filterID")
	if err != nil {
		return err
	}

	filterNameRaw, err := ctx.ResolveInput(node, "filterName")
	if err != nil {
		return err
	}

	// filterName should be used as a function identifier
	filterName := strings.Trim(filterNameRaw, `"`)

	paramsInput := node.Inputs["params"]

	filterVar := fmt.Sprintf("%s_filter", node.ID)

	if paramsInput != nil {
		paramsMap, ok := paramsInput.(map[string]interface{})
		if ok && len(paramsMap) > 0 {
			var paramValues []string
			for _, paramValue := range paramsMap {
				resolved, err := resolveInputValue(ctx, paramValue)
				if err != nil {
					return fmt.Errorf("failed to resolve filter param: %w", err)
				}
				paramValues = append(paramValues, resolved)
			}
			ctx.WriteLine("%s := %s(%s)", filterVar, filterName, strings.Join(paramValues, ", "))
		} else {
			ctx.WriteLine("%s := %s()", filterVar, filterName)
		}
	} else {
		ctx.WriteLine("%s := %s()", filterVar, filterName)
	}

	ctx.WriteLine("filterRegistry.Add(%s, %s, %s)", viewerID, filterID, filterVar)
	ctx.WriteLine("session.SetFilter(%s, filterRegistry.GetComposed(%s))", viewerID, viewerID)

	return nil
}

func generateRemoveFilterNode(ctx *GeneratorContext, node *Node) error {
	viewerID, err := ctx.ResolveInput(node, "viewerID")
	if err != nil {
		return err
	}

	filterID, err := ctx.ResolveInput(node, "filterID")
	if err != nil {
		return err
	}

	removedVar := ctx.AllocateVariable(node, "removed", "bool")
	ctx.WriteLine("%s := filterRegistry.Remove(%s, %s)", removedVar, viewerID, filterID)
	ctx.WriteLine("session.SetFilter(%s, filterRegistry.GetComposed(%s))", viewerID, viewerID)

	return nil
}

func generateHasFilterNode(ctx *GeneratorContext, node *Node) error {
	viewerID, err := ctx.ResolveInput(node, "viewerID")
	if err != nil {
		return err
	}

	filterID, err := ctx.ResolveInput(node, "filterID")
	if err != nil {
		return err
	}

	existsVar := ctx.AllocateVariable(node, "exists", "bool")
	ctx.WriteLine("%s := filterRegistry.Has(%s, %s)", existsVar, viewerID, filterID)

	return nil
}
