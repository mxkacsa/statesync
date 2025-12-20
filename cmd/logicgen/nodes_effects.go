package main

import (
	"fmt"
	"strings"
)

// ============================================================================
// Effect Nodes - Effect management for state transformations
// ============================================================================
//
// Effects are reversible state transformations that don't mutate base state.
// They transform state on read, allowing temporary or conditional modifications.
//
// The effect system uses factory functions similar to filters:
// - Define effect factories in your game code: func MyEffect(param string) Effect[T, A]
// - Reference them by name in the node graph
// - The generated code calls the factory and adds the effect to the session
//
// Example effect factory:
//
//     func NoiseSignalEffect() statesync.Effect[*GameState, string] {
//         return statesync.Func[*GameState, string]("noise-signal",
//             func(state *GameState, activatorID string) *GameState {
//                 // Get activator's team
//                 var activatorTeam string
//                 for i := range state.Players() {
//                     if state.PlayersAt(i).ID == activatorID {
//                         activatorTeam = state.PlayersAt(i).Team
//                         break
//                     }
//                 }
//                 // Set noise=true for players on other teams
//                 for i := range state.Players() {
//                     p := state.PlayersAt(i)
//                     if p.Team != activatorTeam {
//                         p.SetNoise(true)
//                     }
//                 }
//                 return state
//             },
//         )
//     }
//
// ============================================================================

func init() {
	registerEffectNodes()
}

func registerEffectNodes() {
	// AddEffect - Adds an effect to the session with an activator
	MustRegisterNode(NodeDefinition{
		Type:        NodeAddEffect,
		Category:    "Effects",
		Description: "Adds an effect to the session with an activator value",
		Inputs: []PortDefinition{
			{Name: "effectId", Type: "string", Required: true},
			{Name: "effectName", Type: "string", Required: true},
			{Name: "activator", Type: "any", Required: false},
			{Name: "params", Type: "map", Required: false},
		},
		Outputs:   []PortDefinition{},
		Generator: generateAddEffectNode,
	})

	// RemoveEffect - Removes an effect from the session
	MustRegisterNode(NodeDefinition{
		Type:        NodeRemoveEffect,
		Category:    "Effects",
		Description: "Removes an effect from the session by ID",
		Inputs: []PortDefinition{
			{Name: "effectId", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "removed", Type: "bool", Required: true},
		},
		Generator: generateRemoveEffectNode,
	})

	// HasEffect - Checks if an effect exists in the session
	MustRegisterNode(NodeDefinition{
		Type:        NodeHasEffect,
		Category:    "Effects",
		Description: "Checks if an effect exists in the session",
		Inputs: []PortDefinition{
			{Name: "effectId", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "exists", Type: "bool", Required: true},
		},
		Generator: generateHasEffectNode,
	})
}

// ============================================================================
// Generator Functions - Effects
// ============================================================================

func generateAddEffectNode(ctx *GeneratorContext, node *Node) error {
	effectID, err := ctx.ResolveInput(node, "effectId")
	if err != nil {
		return err
	}

	effectNameRaw, err := ctx.ResolveInput(node, "effectName")
	if err != nil {
		return err
	}

	// effectName should be used as a function identifier
	effectName := strings.Trim(effectNameRaw, `"`)

	// Get activator value (optional)
	var activatorSrc string
	if _, hasActivator := node.Inputs["activator"]; hasActivator {
		activatorSrc, _ = ctx.ResolveInput(node, "activator")
	}

	paramsInput := node.Inputs["params"]
	effectVar := fmt.Sprintf("%s_effect", node.ID)

	// Generate effect factory call
	if paramsInput != nil {
		paramsMap, ok := paramsInput.(map[string]interface{})
		if ok && len(paramsMap) > 0 {
			var paramValues []string
			for _, paramValue := range paramsMap {
				resolved, err := resolveInputValue(ctx, paramValue)
				if err != nil {
					return fmt.Errorf("failed to resolve effect param: %w", err)
				}
				paramValues = append(paramValues, resolved)
			}
			ctx.WriteLine("%s := %s(%s)", effectVar, effectName, strings.Join(paramValues, ", "))
		} else {
			ctx.WriteLine("%s := %s()", effectVar, effectName)
		}
	} else {
		ctx.WriteLine("%s := %s()", effectVar, effectName)
	}

	// Add effect to session with activator
	if activatorSrc != "" {
		ctx.WriteLine("if err := session.AddEffect(%s, %s); err != nil {", effectVar, activatorSrc)
	} else {
		// If no activator specified, use empty string as default
		ctx.WriteLine("if err := session.AddEffect(%s, \"\"); err != nil {", effectVar)
	}
	ctx.Indent()
	ctx.WriteLine("// Effect %s already exists or error occurred", effectID)
	ctx.WriteLine("_ = err")
	ctx.Dedent()
	ctx.WriteLine("}")

	return nil
}

func generateRemoveEffectNode(ctx *GeneratorContext, node *Node) error {
	effectID, err := ctx.ResolveInput(node, "effectId")
	if err != nil {
		return err
	}

	removedVar := ctx.AllocateVariable(node, "removed", "bool")
	ctx.WriteLine("%s := session.RemoveEffect(%s)", removedVar, effectID)

	return nil
}

func generateHasEffectNode(ctx *GeneratorContext, node *Node) error {
	effectID, err := ctx.ResolveInput(node, "effectId")
	if err != nil {
		return err
	}

	existsVar := ctx.AllocateVariable(node, "exists", "bool")
	ctx.WriteLine("%s := session.HasEffect(%s)", existsVar, effectID)

	return nil
}
