// schemagen generates Go and TypeScript code from .schema files.
//
// Usage:
//
//	schemagen -input=game.schema -go=game_state.go -ts=game_state.ts
//
// Schema file format:
//
//	package main
//
//	@id(1)
//	type GameState {
//	    Round    int32
//	    Phase    string
//	    Players  []Player  @key(ID)
//	    Secret   string    @view(admin)
//	}
//
//	@id(2)
//	type Player {
//	    ID     string
//	    Name   string
//	    Score  int64
//	    Hand   []int32  @view(owner)
//	}
//
//	view all {}
//	view admin { includes: all }
//	view owner { includes: all }
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func jsonMarshalIndent(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

var (
	inputFile  = flag.String("input", "", "input .schema file (required)")
	goOutput   = flag.String("go", "", "Go output file (optional)")
	tsOutput   = flag.String("ts", "", "TypeScript output file (optional)")
	jsonOutput = flag.String("json", "", "JSON schema output file (optional)")
)

func main() {
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "schemagen: -input flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse the schema file
	f, err := os.Open(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "schemagen: cannot open input file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	schema, err := Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "schemagen: parse error: %v\n", err)
		os.Exit(1)
	}

	// Default package name from input file
	if schema.Package == "" {
		base := filepath.Base(*inputFile)
		schema.Package = strings.TrimSuffix(base, filepath.Ext(base))
	}

	// Generate Go code
	if *goOutput != "" {
		goCode, err := GenerateGo(schema)
		if err != nil {
			fmt.Fprintf(os.Stderr, "schemagen: Go generation error: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*goOutput, goCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "schemagen: cannot write Go output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated: %s\n", *goOutput)
	}

	// Generate TypeScript code
	if *tsOutput != "" {
		tsCode, err := GenerateTS(schema)
		if err != nil {
			fmt.Fprintf(os.Stderr, "schemagen: TypeScript generation error: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*tsOutput, tsCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "schemagen: cannot write TypeScript output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated: %s\n", *tsOutput)
	}

	// Generate JSON schema (for debugging/tooling)
	if *jsonOutput != "" {
		jsonCode, err := GenerateJSON(schema)
		if err != nil {
			fmt.Fprintf(os.Stderr, "schemagen: JSON generation error: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*jsonOutput, jsonCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "schemagen: cannot write JSON output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated: %s\n", *jsonOutput)
	}

	if *goOutput == "" && *tsOutput == "" && *jsonOutput == "" {
		fmt.Fprintln(os.Stderr, "schemagen: no output specified, use -go, -ts, or -json")
		os.Exit(1)
	}
}

// GenerateJSON outputs the parsed schema as JSON (for debugging)
func GenerateJSON(schema *SchemaFile) ([]byte, error) {
	return jsonMarshalIndent(schema)
}
