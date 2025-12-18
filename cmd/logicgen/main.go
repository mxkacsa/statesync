// logicgen generates Go code from node-based logic graphs.
//
// Usage:
//
//	logicgen -input=game_logic.json -output=game_logic_gen.go
//
// Node graph format:
//
//	{
//	  "version": "1.0",
//	  "package": "main",
//	  "handlers": [
//	    {
//	      "name": "OnCardPlayed",
//	      "event": "CardPlayed",
//	      "parameters": [
//	        {"name": "playerID", "type": "string"},
//	        {"name": "cardID", "type": "int32"}
//	      ],
//	      "nodes": [...],
//	      "flow": [...]
//	    }
//	  ]
//	}
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

var (
	inputFile  = flag.String("input", "", "input node graph JSON file (required)")
	outputFile = flag.String("output", "", "output Go file (required)")
	validate   = flag.Bool("validate", false, "only validate, don't generate")
)

func main() {
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "logicgen: -input flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if !*validate && *outputFile == "" {
		fmt.Fprintln(os.Stderr, "logicgen: -output flag is required (unless -validate is set)")
		flag.Usage()
		os.Exit(1)
	}

	// Read input file
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logicgen: cannot read input file: %v\n", err)
		os.Exit(1)
	}

	// Parse node graph
	var nodeGraph NodeGraph
	if err := json.Unmarshal(data, &nodeGraph); err != nil {
		fmt.Fprintf(os.Stderr, "logicgen: cannot parse JSON: %v\n", err)
		os.Exit(1)
	}

	// Set default package
	if nodeGraph.Package == "" {
		nodeGraph.Package = "generated"
	}

	// Validate
	validator := NewValidator(&nodeGraph)
	if err := validator.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "logicgen: validation failed: %v\n", err)
		os.Exit(1)
	}

	if *validate {
		fmt.Println("Validation passed!")
		return
	}

	// Generate code
	generator := NewCodeGenerator(&nodeGraph)
	code, err := generator.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logicgen: code generation failed: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if err := os.WriteFile(*outputFile, code, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "logicgen: cannot write output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated: %s\n", *outputFile)
}
