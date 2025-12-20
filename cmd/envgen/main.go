// envgen generates Go and TypeScript configuration code from .config files.
//
// Usage:
//
//	envgen -input=game.config -go=config_gen.go -ts=config.ts
//
// Config file format:
//
//	package main
//
//	config GameConfig {
//	    Speed           float64     @default(1.0)    @min(0.1) @max(10.0)
//	    MaxPlayers      int32       @default(4)      @min(2)   @max(16)
//	    EnableDebug     bool        @default(false)
//	    GameMode        string      @default("classic") @options("classic","ranked","casual")
//	    RoundDuration   int32       @default(60)        // seconds
//	}
//
// Annotations:
//
//	@default(value)     - Default value
//	@min(value)         - Minimum value (for numbers)
//	@max(value)         - Maximum value (for numbers)
//	@options(a,b,c)     - Valid options (for strings)
//	@env(NAME)          - Environment variable name override
//	@required           - Field is required (no default)
//
// Generated Go code uses tinyconf (github.com/mxkacsa/tinyconf) for:
//   - Loading config from JSON file
//   - Creating config file with defaults if not exists
//   - Environment variable overrides
//   - Runtime reload support
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	inputFile  = flag.String("input", "", "input .config file (required)")
	goOutput   = flag.String("go", "", "Go output file (optional)")
	tsOutput   = flag.String("ts", "", "TypeScript output file (optional)")
	jsonOutput = flag.String("json", "", "JSON schema output file (optional)")
)

func main() {
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "envgen: -input flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse the config file
	f, err := os.Open(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envgen: cannot open input file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	config, err := Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envgen: parse error: %v\n", err)
		os.Exit(1)
	}

	// Default package name from input file
	if config.Package == "" {
		base := filepath.Base(*inputFile)
		config.Package = strings.TrimSuffix(base, filepath.Ext(base))
	}

	// Generate Go code
	if *goOutput != "" {
		goCode, err := GenerateGo(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "envgen: Go generation error: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*goOutput, goCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "envgen: cannot write Go output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated: %s\n", *goOutput)
	}

	// Generate TypeScript code
	if *tsOutput != "" {
		tsCode, err := GenerateTS(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "envgen: TypeScript generation error: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*tsOutput, tsCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "envgen: cannot write TypeScript output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated: %s\n", *tsOutput)
	}

	// Generate JSON schema (for UI editor / tooling)
	if *jsonOutput != "" {
		jsonCode, err := GenerateJSON(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "envgen: JSON generation error: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*jsonOutput, jsonCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "envgen: cannot write JSON output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated: %s\n", *jsonOutput)
	}

	if *goOutput == "" && *tsOutput == "" && *jsonOutput == "" {
		fmt.Fprintln(os.Stderr, "envgen: no output specified, use -go, -ts, or -json")
		os.Exit(1)
	}
}

// GenerateJSON outputs the parsed config as JSON (for UI editor)
func GenerateJSON(config *ConfigFile) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}
