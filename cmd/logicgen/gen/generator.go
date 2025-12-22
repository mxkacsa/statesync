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
	Name         string
	FuncName     string
	Description  string
	Priority     int
	TriggerCode  string
	SelectorCode string
	ViewsCode    string
	EffectsCode  string
	HasSelector  bool
	HasViews     bool
}

// processRule processes a rule into template data
func (g *Generator) processRule(rule *ast.Rule, data *templateData) (*ruleData, error) {
	rd := &ruleData{
		Name:        rule.Name,
		FuncName:    toFuncName(rule.Name),
		Description: rule.Description,
		Priority:    rule.GetPriority(),
		HasSelector: rule.Selector != nil,
		HasViews:    len(rule.Views) > 0,
	}

	// Generate trigger code
	if rule.Trigger != nil {
		code, err := g.generateTriggerCode(rule.Trigger, data)
		if err != nil {
			return nil, fmt.Errorf("trigger: %w", err)
		}
		rd.TriggerCode = code
	}

	// Generate selector code
	if rule.Selector != nil {
		code, err := g.generateSelectorCode(rule.Selector, data)
		if err != nil {
			return nil, fmt.Errorf("selector: %w", err)
		}
		rd.SelectorCode = code
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
		code, err := g.generateEffectsCode(rule.Effects, data)
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
		fmt.Fprintf(&buf, `// Distance trigger
		fromPos := ctx.ResolvePath(%q)
		toPos := ctx.ResolvePath(%q)
		dist := haversineDistance(fromPos, toPos)
		if !(dist %s %v) { return nil }`,
			trigger.From, trigger.To, trigger.Operator, trigger.Value)

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

// generateSelectorCode generates code for a selector
func (g *Generator) generateSelectorCode(selector *ast.Selector, data *templateData) (string, error) {
	var buf bytes.Buffer

	switch selector.Type {
	case ast.SelectorTypeAll:
		fmt.Fprintf(&buf, "entities := state.%s", selector.Entity)

	case ast.SelectorTypeFilter:
		fmt.Fprintf(&buf, "var entities []%s\n", selector.Entity)
		fmt.Fprintf(&buf, "for _, e := range state.%s {\n", selector.Entity)
		if selector.Where != nil {
			condCode := g.generateWhereCode(selector.Where, "e")
			fmt.Fprintf(&buf, "  if %s {\n    entities = append(entities, e)\n  }\n", condCode)
		} else {
			fmt.Fprintf(&buf, "  entities = append(entities, e)\n")
		}
		fmt.Fprintf(&buf, "}")

	case ast.SelectorTypeSingle:
		fmt.Fprintf(&buf, "var entities []%s\n", selector.Entity)
		fmt.Fprintf(&buf, "targetID := ctx.Resolve(%q)\n", selector.ID)
		fmt.Fprintf(&buf, "for _, e := range state.%s {\n", selector.Entity)
		fmt.Fprintf(&buf, "  if e.ID == targetID {\n")
		fmt.Fprintf(&buf, "    entities = append(entities, e)\n    break\n  }\n}")

	case ast.SelectorTypeNearest:
		data.Imports["sort"] = true
		data.Imports["math"] = true
		fmt.Fprintf(&buf, `// Select nearest entities
		origin := ctx.Resolve(%q)
		type entityDist struct { e %s; dist float64 }
		var withDist []entityDist
		for _, e := range state.%s {
			dist := haversineDistance(origin, e.Position)
			if dist <= %v {
				withDist = append(withDist, entityDist{e, dist})
			}
		}
		sort.Slice(withDist, func(i, j int) bool { return withDist[i].dist < withDist[j].dist })
		entities := make([]%s, 0, %d)
		for i := 0; i < len(withDist) && i < %d; i++ {
			entities = append(entities, withDist[i].e)
		}`,
			selector.Origin, selector.Entity, selector.Entity,
			selector.MaxDistance, selector.Entity, selector.Limit, selector.Limit)
	}

	return buf.String(), nil
}

// generateViewsCode generates code for views
func (g *Generator) generateViewsCode(views map[string]*ast.View, data *templateData) (string, error) {
	var buf bytes.Buffer

	for name, view := range views {
		switch view.Type {
		case ast.ViewTypeCount:
			fmt.Fprintf(&buf, "views[%q] = len(entities)\n", name)

		case ast.ViewTypeSum:
			fmt.Fprintf(&buf, "var sum_%s float64\n", name)
			fmt.Fprintf(&buf, "for _, e := range entities {\n")
			fmt.Fprintf(&buf, "  sum_%s += float64(e.%s)\n", name, pathToField(view.Field))
			fmt.Fprintf(&buf, "}\nviews[%q] = sum_%s\n", name, name)

		case ast.ViewTypeMax:
			data.Imports["math"] = true
			fmt.Fprintf(&buf, "var max_%s float64 = math.Inf(-1)\n", name)
			fmt.Fprintf(&buf, "for _, e := range entities {\n")
			fmt.Fprintf(&buf, "  if v := float64(e.%s); v > max_%s { max_%s = v }\n",
				pathToField(view.Field), name, name)
			fmt.Fprintf(&buf, "}\nviews[%q] = max_%s\n", name, name)

		case ast.ViewTypeMin:
			data.Imports["math"] = true
			fmt.Fprintf(&buf, "var min_%s float64 = math.Inf(1)\n", name)
			fmt.Fprintf(&buf, "for _, e := range entities {\n")
			fmt.Fprintf(&buf, "  if v := float64(e.%s); v < min_%s { min_%s = v }\n",
				pathToField(view.Field), name, name)
			fmt.Fprintf(&buf, "}\nviews[%q] = min_%s\n", name, name)

		case ast.ViewTypeAvg:
			fmt.Fprintf(&buf, "var sum_%s float64\n", name)
			fmt.Fprintf(&buf, "for _, e := range entities {\n")
			fmt.Fprintf(&buf, "  sum_%s += float64(e.%s)\n", name, pathToField(view.Field))
			fmt.Fprintf(&buf, "}\nif len(entities) > 0 { views[%q] = sum_%s / float64(len(entities)) }\n", name, name)
		}
	}

	return buf.String(), nil
}

// generateEffectsCode generates code for effects
func (g *Generator) generateEffectsCode(effects []*ast.Effect, data *templateData) (string, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "for i, entity := range entities {\n")
	fmt.Fprintf(&buf, "  _ = i\n") // Avoid unused variable

	for _, effect := range effects {
		code, err := g.generateEffectCode(effect, data)
		if err != nil {
			return "", err
		}
		buf.WriteString(code)
	}

	fmt.Fprintf(&buf, "}")

	return buf.String(), nil
}

// generateEffectCode generates code for a single effect
func (g *Generator) generateEffectCode(effect *ast.Effect, data *templateData) (string, error) {
	var buf bytes.Buffer

	switch effect.Type {
	case ast.EffectTypeSet:
		field := pathToField(effect.Path)
		valueCode, err := g.generateValueCode(effect.Value)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&buf, "  entity.%s = %s\n", field, valueCode)

	case ast.EffectTypeIncrement:
		field := pathToField(effect.Path)
		valueCode, err := g.generateValueCode(effect.Value)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&buf, "  entity.%s += %s\n", field, valueCode)

	case ast.EffectTypeDecrement:
		field := pathToField(effect.Path)
		valueCode, err := g.generateValueCode(effect.Value)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&buf, "  entity.%s -= %s\n", field, valueCode)

	case ast.EffectTypeTransform:
		if effect.Transform != nil {
			field := pathToField(effect.Path)
			transformCode, err := g.generateTransformCode(effect.Transform, data)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&buf, "  entity.%s = %s\n", field, transformCode)
		}

	case ast.EffectTypeEmit:
		fmt.Fprintf(&buf, "  // Emit event: %s\n", effect.Event)
		fmt.Fprintf(&buf, "  ctx.Emit(%q, map[string]interface{}{\n", effect.Event)
		for k, v := range effect.Payload {
			valueCode, _ := g.generateValueCode(v)
			fmt.Fprintf(&buf, "    %q: %s,\n", k, valueCode)
		}
		fmt.Fprintf(&buf, "  })\n")
	}

	return buf.String(), nil
}

// generateTransformCode generates code for a transform
func (g *Generator) generateTransformCode(t *ast.Transform, data *templateData) (string, error) {
	switch t.Type {
	case ast.TransformTypeMoveTowards:
		data.Imports["math"] = true
		return fmt.Sprintf("moveTowards(entity.%s, %s, %v * float64(ctx.DeltaTime.Milliseconds()) / 1000.0)",
			pathToField(ast.Path(fmt.Sprintf("%v", t.Current))),
			g.generatePathOrValue(t.Target),
			t.Speed), nil

	case ast.TransformTypeAdd:
		left, _ := g.generateValueCode(t.Left)
		right, _ := g.generateValueCode(t.Right)
		return fmt.Sprintf("(%s + %s)", left, right), nil

	case ast.TransformTypeSubtract:
		left, _ := g.generateValueCode(t.Left)
		right, _ := g.generateValueCode(t.Right)
		return fmt.Sprintf("(%s - %s)", left, right), nil

	case ast.TransformTypeMultiply:
		left, _ := g.generateValueCode(t.Left)
		right, _ := g.generateValueCode(t.Right)
		return fmt.Sprintf("(%s * %s)", left, right), nil

	case ast.TransformTypeClamp:
		data.Imports["math"] = true
		value, _ := g.generateValueCode(t.Value)
		min, _ := g.generateValueCode(t.Min)
		max, _ := g.generateValueCode(t.Max)
		return fmt.Sprintf("math.Max(%s, math.Min(%s, %s))", min, max, value), nil

	default:
		return fmt.Sprintf("/* TODO: transform %s */", t.Type), nil
	}
}

// generateValueCode generates code for a value
func (g *Generator) generateValueCode(v interface{}) (string, error) {
	switch val := v.(type) {
	case string:
		if strings.HasPrefix(val, "$.") {
			return "entity." + pathToField(ast.Path(val)), nil
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

// generatePathOrValue generates code for a path or literal value
func (g *Generator) generatePathOrValue(v interface{}) string {
	code, _ := g.generateValueCode(v)
	return code
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
	return s
}

var templateFuncs = template.FuncMap{
	"toFuncName":  toFuncName,
	"pathToField": pathToField,
	"join":        strings.Join,
}
