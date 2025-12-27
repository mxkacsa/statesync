// Package parse provides JSON parsing and validation for LogicGen v2 rules.
package parse

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// Parser parses rule definition files
type Parser struct {
	// Validators to run after parsing
	validators []Validator
}

// NewParser creates a new parser
func NewParser() *Parser {
	return &Parser{
		validators: []Validator{
			&RequiredFieldsValidator{},
			&PathValidator{},
			&TriggerValidator{},
		},
	}
}

// ParseFile parses a single rule file
func (p *Parser) ParseFile(path string) (*ast.RuleSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return p.Parse(data, path)
}

// ParseDirectory parses all .json files in a directory
func (p *Parser) ParseDirectory(dir string) (*ast.RuleSet, error) {
	ruleSet := &ast.RuleSet{
		Rules:   make([]*ast.Rule, 0),
		Version: "2.0",
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("glob files: %w", err)
	}

	for _, file := range files {
		rs, err := p.ParseFile(file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(file), err)
		}
		ruleSet.Rules = append(ruleSet.Rules, rs.Rules...)
	}

	return ruleSet, nil
}

// Parse parses rule definitions from JSON data
func (p *Parser) Parse(data []byte, source string) (*ast.RuleSet, error) {
	// First, check if this is a RuleSet by looking for "rules" key
	var rawCheck map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawCheck); err == nil {
		// If it has a "rules" array, treat as RuleSet
		if _, hasRules := rawCheck["rules"]; hasRules {
			var ruleSet ast.RuleSet
			if err := json.Unmarshal(data, &ruleSet); err != nil {
				return nil, fmt.Errorf("parse RuleSet: %w", err)
			}
			if err := p.validate(&ruleSet); err != nil {
				return nil, err
			}
			return &ruleSet, nil
		}
	}

	// Try parsing as a single Rule (must have "name" and either "trigger" or "effects")
	var rule ast.Rule
	if err := json.Unmarshal(data, &rule); err == nil && rule.Name != "" && (rule.Trigger != nil || len(rule.Effects) > 0) {
		ruleSet := ast.RuleSet{
			Rules:   []*ast.Rule{&rule},
			Version: "2.0",
		}
		if err := p.validate(&ruleSet); err != nil {
			return nil, err
		}
		return &ruleSet, nil
	}

	// Try parsing as an array of Rules
	var rules []*ast.Rule
	if err := json.Unmarshal(data, &rules); err == nil && len(rules) > 0 {
		ruleSet := ast.RuleSet{
			Rules:   rules,
			Version: "2.0",
		}
		if err := p.validate(&ruleSet); err != nil {
			return nil, err
		}
		return &ruleSet, nil
	}

	return nil, fmt.Errorf("invalid rule format in %s", source)
}

// validate runs all validators on the rule set
func (p *Parser) validate(ruleSet *ast.RuleSet) error {
	var errors []error
	for _, v := range p.validators {
		if err := v.Validate(ruleSet); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}
	return nil
}

// AddValidator adds a custom validator
func (p *Parser) AddValidator(v Validator) {
	p.validators = append(p.validators, v)
}

// ParseRule parses a raw JSON map into a Rule
func ParseRule(data map[string]interface{}) (*ast.Rule, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var rule ast.Rule
	if err := json.Unmarshal(jsonData, &rule); err != nil {
		return nil, err
	}

	return &rule, nil
}

// ParseTrigger parses a raw JSON map into a Trigger
func ParseTrigger(data map[string]interface{}) (*ast.Trigger, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var trigger ast.Trigger
	if err := json.Unmarshal(jsonData, &trigger); err != nil {
		return nil, err
	}

	return &trigger, nil
}

// ParseView parses a raw JSON map into a View
func ParseView(data map[string]interface{}) (*ast.View, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var view ast.View
	if err := json.Unmarshal(jsonData, &view); err != nil {
		return nil, err
	}

	return &view, nil
}

// ParseEffect parses a raw JSON map into an Effect
func ParseEffect(data map[string]interface{}) (*ast.Effect, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var effect ast.Effect
	if err := json.Unmarshal(jsonData, &effect); err != nil {
		return nil, err
	}

	return &effect, nil
}

// ParseTransform parses a raw JSON map into a Transform
func ParseTransform(data map[string]interface{}) (*ast.Transform, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var transform ast.Transform
	if err := json.Unmarshal(jsonData, &transform); err != nil {
		return nil, err
	}

	return &transform, nil
}
