// Package gen provides Go code generation for LogicGen v2 rules.
package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// Generator generates Go code from rules
type Generator struct {
	// Package name for generated code
	PackageName string
	// State type name
	StateType string
	// Whether to generate comments
	Comments bool
}

// NewGenerator creates a new code generator
func NewGenerator(packageName, stateType string) *Generator {
	return &Generator{
		PackageName: packageName,
		StateType:   stateType,
		Comments:    true,
	}
}

// Generate generates Go code for a rule set
func (g *Generator) Generate(ruleSet *ast.RuleSet) ([]byte, error) {
	data := &templateData{
		PackageName: g.PackageName,
		StateType:   g.StateType,
		Rules:       make([]*ruleData, 0, len(ruleSet.Rules)),
		Imports:     make(map[string]bool),
	}

	// Add base imports
	data.Imports["context"] = true
	data.Imports["time"] = true
	data.Imports["fmt"] = true

	// Process each rule
	for _, rule := range ruleSet.Rules {
		rd, err := g.processRule(rule, data)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", rule.Name, err)
		}
		data.Rules = append(data.Rules, rd)
	}

	// Generate code using template
	tmpl, err := template.New("rules").Funcs(templateFuncs).Parse(rulesTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted code with error for debugging
		return buf.Bytes(), fmt.Errorf("format code: %w (unformatted code returned)", err)
	}

	return formatted, nil
}

// templateData holds data for the template
type templateData struct {
	PackageName string
	StateType   string
	Rules       []*ruleData
	Imports     map[string]bool
}

// ruleData holds processed rule data
type ruleData struct {
	Name        string
	FuncName    string
	Description string
	Priority    int
	TriggerCode string
	ViewsCode   string
	EffectsCode string
	HasViews    bool
	TriggerType string // "tick", "event", "timer", "condition"
	EventName   string // For event-based triggers
}

// processRule processes a rule into template data
func (g *Generator) processRule(rule *ast.Rule, data *templateData) (*ruleData, error) {
	rd := &ruleData{
		Name:        rule.Name,
		FuncName:    toFuncName(rule.Name),
		Description: rule.Description,
		Priority:    rule.GetPriority(),
		HasViews:    len(rule.Views) > 0,
		TriggerType: "tick", // default
	}

	// Determine trigger type and generate code
	if rule.Trigger != nil {
		switch rule.Trigger.Type {
		case ast.TriggerTypeOnEvent:
			rd.TriggerType = "event"
			rd.EventName = rule.Trigger.Event
		case ast.TriggerTypeOnTick:
			rd.TriggerType = "tick"
		case ast.TriggerTypeTimer:
			rd.TriggerType = "timer"
		case ast.TriggerTypeCondition:
			rd.TriggerType = "condition"
		case ast.TriggerTypeDistance:
			rd.TriggerType = "tick" // Distance triggers run on tick
		}

		code, err := g.generateTriggerCode(rule.Trigger, data)
		if err != nil {
			return nil, fmt.Errorf("trigger: %w", err)
		}
		rd.TriggerCode = code
	}

	// Generate views code
	if len(rule.Views) > 0 {
		code, err := g.generateViewsCode(rule.Views, data)
		if err != nil {
			return nil, fmt.Errorf("views: %w", err)
		}
		rd.ViewsCode = code
	}

	// Generate effects code
	if len(rule.Effects) > 0 {
		code, err := g.generateEffectsCode(rule.Effects, rule.Views, data)
		if err != nil {
			return nil, fmt.Errorf("effects: %w", err)
		}
		rd.EffectsCode = code
	}

	return rd, nil
}

// generateTriggerCode generates code for a trigger
func (g *Generator) generateTriggerCode(trigger *ast.Trigger, data *templateData) (string, error) {
	var buf bytes.Buffer

	switch trigger.Type {
	case ast.TriggerTypeOnTick:
		if trigger.Interval > 0 {
			fmt.Fprintf(&buf, "if ctx.Tick %% %d != 0 { return nil }", trigger.Interval)
		}
		// OnTick with no interval always fires

	case ast.TriggerTypeOnEvent:
		fmt.Fprintf(&buf, "if ctx.Event == nil || ctx.Event.Name != %q { return nil }", trigger.Event)

	case ast.TriggerTypeTimer:
		data.Imports["sync"] = true
		fmt.Fprintf(&buf, `// Timer trigger - duration: %dms
		if !r.timerFired_%s(ctx) { return nil }`, trigger.Duration, toFuncName(trigger.Event))

	case ast.TriggerTypeDistance:
		data.Imports["math"] = true
		fromField := pathToField(ast.Path(trigger.From))
		toField := pathToField(ast.Path(trigger.To))
		fmt.Fprintf(&buf, `// Distance trigger
		fromPos := state.%s
		toPos := state.%s
		dist := haversineDistance(fromPos, toPos)
		if !(dist %s %v) { return nil }`,
			fromField, toField, trigger.Operator, trigger.Value)

	case ast.TriggerTypeCondition:
		if trigger.Condition != nil {
			condCode, err := g.generateExpressionCode(trigger.Condition)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&buf, "if !(%s) { return nil }", condCode)
		}
	}

	return buf.String(), nil
}

// generateViewsCode generates code for views
func (g *Generator) generateViewsCode(views map[string]*ast.View, data *templateData) (string, error) {
	var buf bytes.Buffer

	for name, view := range views {
		// Generate view evaluation code
		fmt.Fprintf(&buf, "// View: %s\n", name)
		fmt.Fprintf(&buf, "entities_%s := state.%s\n", name, view.Source)

		// Apply pipeline operations
		for i, op := range view.Pipeline {
			switch op.Type {
			case ast.ViewOpFilter:
				fmt.Fprintf(&buf, "var filtered_%s_%d []interface{}\n", name, i)
				fmt.Fprintf(&buf, "for _, e := range entities_%s {\n", name)
				if op.Where != nil {
					condCode := g.generateWhereCode(op.Where, "e")
					fmt.Fprintf(&buf, "  if %s {\n    filtered_%s_%d = append(filtered_%s_%d, e)\n  }\n", condCode, name, i, name, i)
				}
				fmt.Fprintf(&buf, "}\n")
				fmt.Fprintf(&buf, "entities_%s = filtered_%s_%d\n", name, name, i)

			case ast.ViewOpCount:
				fmt.Fprintf(&buf, "views[%q] = len(entities_%s)\n", name, name)

			case ast.ViewOpSum:
				fmt.Fprintf(&buf, "var sum_%s float64\n", name)
				fmt.Fprintf(&buf, "for _, e := range entities_%s {\n", name)
				fmt.Fprintf(&buf, "  sum_%s += float64(e.%s)\n", name, pathToField(op.Field))
				fmt.Fprintf(&buf, "}\nviews[%q] = sum_%s\n", name, name)

			case ast.ViewOpMax:
				data.Imports["math"] = true
				fmt.Fprintf(&buf, "var max_%s float64 = math.Inf(-1)\n", name)
				fmt.Fprintf(&buf, "for _, e := range entities_%s {\n", name)
				fmt.Fprintf(&buf, "  if v := float64(e.%s); v > max_%s { max_%s = v }\n",
					pathToField(op.Field), name, name)
				fmt.Fprintf(&buf, "}\nviews[%q] = max_%s\n", name, name)

			case ast.ViewOpMin:
				data.Imports["math"] = true
				fmt.Fprintf(&buf, "var min_%s float64 = math.Inf(1)\n", name)
				fmt.Fprintf(&buf, "for _, e := range entities_%s {\n", name)
				fmt.Fprintf(&buf, "  if v := float64(e.%s); v < min_%s { min_%s = v }\n",
					pathToField(op.Field), name, name)
				fmt.Fprintf(&buf, "}\nviews[%q] = min_%s\n", name, name)

			case ast.ViewOpAvg:
				fmt.Fprintf(&buf, "var sum_%s float64\n", name)
				fmt.Fprintf(&buf, "for _, e := range entities_%s {\n", name)
				fmt.Fprintf(&buf, "  sum_%s += float64(e.%s)\n", name, pathToField(op.Field))
				fmt.Fprintf(&buf, "}\nif len(entities_%s) > 0 { views[%q] = sum_%s / float64(len(entities_%s)) }\n",
					name, name, name, name)
			}
		}

		// Store final result
		fmt.Fprintf(&buf, "views[%q] = entities_%s\n", name, name)
	}

	return buf.String(), nil
}

// generateEffectsCode generates code for effects
func (g *Generator) generateEffectsCode(effects []*ast.Effect, views map[string]*ast.View, data *templateData) (string, error) {
	var buf bytes.Buffer

	for _, effect := range effects {
		code, err := g.generateEffectCode(effect, views, data)
		if err != nil {
			return "", err
		}
		buf.WriteString(code)
	}

	return buf.String(), nil
}

// generateEffectCode generates code for a single effect
func (g *Generator) generateEffectCode(effect *ast.Effect, views map[string]*ast.View, data *templateData) (string, error) {
	var buf bytes.Buffer

	// Get targets
	targetsVar := "state"
	if effect.Targets != nil {
		if viewName, ok := effect.Targets.(string); ok {
			targetsVar = fmt.Sprintf("views[%q].([]interface{})", viewName)
			fmt.Fprintf(&buf, "for i, entity := range %s {\n", targetsVar)
			fmt.Fprintf(&buf, "  _ = i\n")
		}
	}

	switch effect.Type {
	case ast.EffectTypeSet:
		field := pathToField(effect.Path)
		valueCode, err := g.generateValueCode(effect.Value)
		if err != nil {
			return "", err
		}
		if effect.Targets != nil {
			fmt.Fprintf(&buf, "  entity.%s = %s\n", field, valueCode)
		} else {
			fmt.Fprintf(&buf, "state.%s = %s\n", field, valueCode)
		}

	case ast.EffectTypeIncrement:
		field := pathToField(effect.Path)
		valueCode, err := g.generateValueCode(effect.Value)
		if err != nil {
			return "", err
		}
		if effect.Targets != nil {
			fmt.Fprintf(&buf, "  entity.%s += %s\n", field, valueCode)
		} else {
			fmt.Fprintf(&buf, "state.%s += %s\n", field, valueCode)
		}

	case ast.EffectTypeDecrement:
		field := pathToField(effect.Path)
		valueCode, err := g.generateValueCode(effect.Value)
		if err != nil {
			return "", err
		}
		if effect.Targets != nil {
			fmt.Fprintf(&buf, "  entity.%s -= %s\n", field, valueCode)
		} else {
			fmt.Fprintf(&buf, "state.%s -= %s\n", field, valueCode)
		}

	case ast.EffectTypeEmit:
		fmt.Fprintf(&buf, "// Emit event: %s\n", effect.Event)
		fmt.Fprintf(&buf, "ctx.Emit(%q, map[string]interface{}{\n", effect.Event)
		for k, v := range effect.Payload {
			valueCode, _ := g.generateValueCode(v)
			fmt.Fprintf(&buf, "  %q: %s,\n", k, valueCode)
		}
		fmt.Fprintf(&buf, "})\n")
	}

	if effect.Targets != nil {
		fmt.Fprintf(&buf, "}\n")
	}

	return buf.String(), nil
}

// generateValueCode generates code for a value
func (g *Generator) generateValueCode(v interface{}) (string, error) {
	switch val := v.(type) {
	case string:
		if strings.HasPrefix(val, "$.") {
			return "entity." + pathToField(ast.Path(val)), nil
		}
		if strings.HasPrefix(val, "self.") {
			return "entity." + val[5:], nil
		}
		if strings.HasPrefix(val, "view:") {
			return fmt.Sprintf("views[%q]", val[5:]), nil
		}
		if strings.HasPrefix(val, "param:") {
			return fmt.Sprintf("ctx.Params[%q]", val[6:]), nil
		}
		return fmt.Sprintf("%q", val), nil
	case float64:
		return fmt.Sprintf("%v", val), nil
	case int:
		return fmt.Sprintf("%d", val), nil
	case bool:
		return fmt.Sprintf("%v", val), nil
	case ast.Path:
		return "entity." + pathToField(val), nil
	default:
		return fmt.Sprintf("%#v", v), nil
	}
}

// generateExpressionCode generates code for an expression
func (g *Generator) generateExpressionCode(expr *ast.Expression) (string, error) {
	if expr == nil {
		return "true", nil
	}

	if len(expr.And) > 0 {
		var parts []string
		for _, sub := range expr.And {
			code, err := g.generateExpressionCode(&sub)
			if err != nil {
				return "", err
			}
			parts = append(parts, "("+code+")")
		}
		return strings.Join(parts, " && "), nil
	}

	if len(expr.Or) > 0 {
		var parts []string
		for _, sub := range expr.Or {
			code, err := g.generateExpressionCode(&sub)
			if err != nil {
				return "", err
			}
			parts = append(parts, "("+code+")")
		}
		return strings.Join(parts, " || "), nil
	}

	if expr.Not != nil {
		code, err := g.generateExpressionCode(expr.Not)
		if err != nil {
			return "", err
		}
		return "!(" + code + ")", nil
	}

	left, _ := g.generateValueCode(expr.Left)
	right, _ := g.generateValueCode(expr.Right)
	return fmt.Sprintf("%s %s %s", left, expr.Op, right), nil
}

// generateWhereCode generates code for a where clause
func (g *Generator) generateWhereCode(where *ast.WhereClause, varName string) string {
	valueCode, _ := g.generateValueCode(where.Value)
	return fmt.Sprintf("%s.%s %s %s", varName, where.Field, where.Op, valueCode)
}

// Helper functions

func toFuncName(name string) string {
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}

func pathToField(path ast.Path) string {
	s := string(path)
	if strings.HasPrefix(s, "$.") {
		s = s[2:]
	}
	if strings.HasPrefix(s, "self.") {
		s = s[5:]
	}
	return s
}

var templateFuncs = template.FuncMap{
	"toFuncName":  toFuncName,
	"pathToField": pathToField,
	"join":        strings.Join,
}
