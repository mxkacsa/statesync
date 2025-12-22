// Package main provides the CLI for LogicGen v2 - a declarative rule engine.
//
// Usage:
//
//	logicgenv2 -input rules.json -output rules_gen.go -package game -state GameState
//	logicgenv2 -validate -input rules.json
//	logicgenv2 -run -input rules.json -state state.json
//
// V2 uses a declarative rule format with:
//   - Triggers: OnTick, OnEvent, OnChange, Distance, Timer, Condition
//   - Selectors: All, Filter, Single, Related, Nearest, Farthest
//   - Views: Sum, Max, Min, Count, Avg, GroupBy, Sort, Distance
//   - Effects: Set, Increment, Append, Remove, Emit, Spawn, Transform
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
	"github.com/mxkacsa/statesync/cmd/logicgen/eval"
	"github.com/mxkacsa/statesync/cmd/logicgen/gen"
	"github.com/mxkacsa/statesync/cmd/logicgen/parse"
)

var (
	inputFile   = flag.String("input", "", "Input rules JSON file or directory (required)")
	outputFile  = flag.String("output", "", "Output Go file for code generation")
	packageName = flag.String("package", "generated", "Package name for generated code")
	stateType   = flag.String("state", "State", "State struct type name")
	validate    = flag.Bool("validate", false, "Only validate rules, don't generate")
	run         = flag.Bool("run", false, "Run rules interactively (for testing)")
	stateFile   = flag.String("statefile", "", "Initial state JSON file (for -run mode)")
	tickRate    = flag.Duration("tick", 100*time.Millisecond, "Tick rate for -run mode")
	strict      = flag.Bool("strict", false, "Use strict validation")
)

func main() {
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "logicgenv2: -input flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse rules
	parser := parse.NewParser()
	if *strict {
		parser.AddValidator(parse.StrictValidator())
	}

	var ruleSet *ast.RuleSet
	var err error

	// Check if input is a directory
	fi, err := os.Stat(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logicgenv2: cannot access input: %v\n", err)
		os.Exit(1)
	}

	if fi.IsDir() {
		ruleSet, err = parser.ParseDirectory(*inputFile)
	} else {
		ruleSet, err = parser.ParseFile(*inputFile)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "logicgenv2: parse error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed %d rules\n", len(ruleSet.Rules))

	// Validate only mode
	if *validate {
		fmt.Println("Validation passed!")
		for _, rule := range ruleSet.Rules {
			status := "enabled"
			if !rule.IsEnabled() {
				status = "disabled"
			}
			fmt.Printf("  - %s (priority: %d, %s)\n", rule.Name, rule.GetPriority(), status)
		}
		return
	}

	// Run mode - execute rules interactively
	if *run {
		if err := runInteractive(ruleSet); err != nil {
			fmt.Fprintf(os.Stderr, "logicgenv2: run error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Generate mode
	if *outputFile == "" {
		fmt.Fprintln(os.Stderr, "logicgenv2: -output flag is required for code generation")
		flag.Usage()
		os.Exit(1)
	}

	generator := gen.NewGenerator(*packageName, *stateType)
	code, err := generator.Generate(ruleSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logicgenv2: generation error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputFile, code, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "logicgenv2: cannot write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated: %s\n", *outputFile)
}

// runInteractive runs the rules in an interactive test mode
func runInteractive(ruleSet *ast.RuleSet) error {
	// Create a simple test state
	var state interface{}

	if *stateFile != "" {
		data, err := os.ReadFile(*stateFile)
		if err != nil {
			return fmt.Errorf("read state file: %w", err)
		}
		if err := json.Unmarshal(data, &state); err != nil {
			return fmt.Errorf("parse state file: %w", err)
		}
	} else {
		// Create default empty state
		state = &map[string]interface{}{}
	}

	// Create engine
	engine := eval.NewEngine(state, ruleSet.Rules)
	engine.SetTickRate(*tickRate)

	ctx := context.Background()

	fmt.Println("Running rules engine...")
	fmt.Printf("Tick rate: %v\n", *tickRate)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	ticker := time.NewTicker(*tickRate)
	defer ticker.Stop()

	for tick := uint64(1); ; tick++ {
		select {
		case <-ticker.C:
			if err := engine.Tick(ctx); err != nil {
				fmt.Printf("Tick %d error: %v\n", tick, err)
			} else if tick%10 == 0 {
				fmt.Printf("Tick %d completed\n", tick)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
