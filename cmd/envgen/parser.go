package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// Parser parses .config files
type Parser struct {
	scanner *bufio.Scanner
	line    int
	file    *ConfigFile
}

// Parse parses a .config file from a reader
func Parse(r io.Reader) (*ConfigFile, error) {
	p := &Parser{
		scanner: bufio.NewScanner(r),
		file: &ConfigFile{
			Configs: make([]*ConfigDef, 0),
		},
	}

	return p.parse()
}

func (p *Parser) parse() (*ConfigFile, error) {
	for p.scanner.Scan() {
		p.line++
		line := strings.TrimSpace(p.scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse directives and config definitions
		if strings.HasPrefix(line, "package ") {
			p.file.Package = strings.TrimPrefix(line, "package ")
			continue
		}

		if strings.HasPrefix(line, "config ") {
			configDef, err := p.parseConfigFromLine(line)
			if err != nil {
				return nil, err
			}
			p.file.Configs = append(p.file.Configs, configDef)
			continue
		}
	}

	if err := p.scanner.Err(); err != nil {
		return nil, err
	}

	return p.file, nil
}

func (p *Parser) parseConfigFromLine(line string) (*ConfigDef, error) {
	// Parse "config Name {" or "config Name {"
	if !strings.HasPrefix(line, "config ") {
		return nil, p.errorf("expected 'config', got: %s", line)
	}

	rest := strings.TrimPrefix(line, "config ")
	rest = strings.TrimSpace(rest)

	// Extract name
	nameEnd := strings.IndexAny(rest, " {")
	if nameEnd == -1 {
		return nil, p.errorf("invalid config definition: %s", line)
	}
	name := rest[:nameEnd]

	configDef := &ConfigDef{
		Name:   name,
		Fields: make([]*FieldDef, 0),
	}

	// Check if { is on same line or need to find it
	if !strings.Contains(rest, "{") {
		// Find opening brace
		for p.scanner.Scan() {
			p.line++
			brLine := strings.TrimSpace(p.scanner.Text())
			if brLine == "{" {
				break
			}
			if brLine != "" && !strings.HasPrefix(brLine, "//") {
				return nil, p.errorf("expected '{', got: %s", brLine)
			}
		}
	}

	// Parse fields until closing brace
	for p.scanner.Scan() {
		p.line++
		fieldLine := strings.TrimSpace(p.scanner.Text())

		if fieldLine == "" {
			continue
		}

		// Check for inline comment to capture as description
		if strings.HasPrefix(fieldLine, "//") {
			// Skip standalone comments
			continue
		}

		if fieldLine == "}" {
			break
		}

		field, err := p.parseField(fieldLine)
		if err != nil {
			return nil, err
		}
		configDef.Fields = append(configDef.Fields, field)
	}

	return configDef, nil
}

func (p *Parser) parseField(line string) (*FieldDef, error) {
	// Parse: "Name  Type  @default(value) @min(0) @max(100) // Description"

	// Extract inline comment first
	var description string
	if idx := strings.Index(line, "//"); idx != -1 {
		description = strings.TrimSpace(line[idx+2:])
		line = strings.TrimSpace(line[:idx])
	}

	// Split by whitespace
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, p.errorf("invalid field definition: %s", line)
	}

	field := &FieldDef{
		Name:        parts[0],
		Type:        parts[1],
		Description: description,
	}

	// Parse annotations
	for i := 2; i < len(parts); i++ {
		ann := parts[i]

		if strings.HasPrefix(ann, "@default(") {
			end := strings.LastIndex(ann, ")")
			if end == -1 {
				return nil, p.errorf("invalid @default annotation: %s", ann)
			}
			value := ann[9:end]
			field.Default = p.parseValue(value, field.Type)
			continue
		}

		if strings.HasPrefix(ann, "@min(") {
			end := strings.Index(ann, ")")
			if end == -1 {
				return nil, p.errorf("invalid @min annotation: %s", ann)
			}
			val, err := strconv.ParseFloat(ann[5:end], 64)
			if err != nil {
				return nil, p.errorf("invalid @min value: %s", ann)
			}
			field.Min = &val
			continue
		}

		if strings.HasPrefix(ann, "@max(") {
			end := strings.Index(ann, ")")
			if end == -1 {
				return nil, p.errorf("invalid @max annotation: %s", ann)
			}
			val, err := strconv.ParseFloat(ann[5:end], 64)
			if err != nil {
				return nil, p.errorf("invalid @max value: %s", ann)
			}
			field.Max = &val
			continue
		}

		if strings.HasPrefix(ann, "@options(") {
			end := strings.LastIndex(ann, ")")
			if end == -1 {
				return nil, p.errorf("invalid @options annotation: %s", ann)
			}
			optStr := ann[9:end]
			for _, opt := range strings.Split(optStr, ",") {
				opt = strings.TrimSpace(opt)
				opt = strings.Trim(opt, `"'`)
				if opt != "" {
					field.Options = append(field.Options, opt)
				}
			}
			continue
		}

		if strings.HasPrefix(ann, "@env(") {
			end := strings.Index(ann, ")")
			if end == -1 {
				return nil, p.errorf("invalid @env annotation: %s", ann)
			}
			envVal := ann[5:end]
			// Remove quotes if present
			envVal = strings.Trim(envVal, `"'`)
			field.Env = envVal
			continue
		}

		if ann == "@required" {
			field.Required = true
			continue
		}
	}

	return field, nil
}

func (p *Parser) parseValue(s string, typ string) interface{} {
	s = strings.TrimSpace(s)

	// Remove quotes from strings
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return s[1 : len(s)-1]
	}
	if strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`) {
		return s[1 : len(s)-1]
	}

	// Parse based on type
	pt := ParseType(typ)

	switch pt.BaseType {
	case "bool":
		return s == "true"
	case "int", "int8", "int16", "int32", "int64":
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
	case "uint", "uint8", "uint16", "uint32", "uint64":
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			return v
		}
	case "float32", "float64":
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	case "duration":
		// Duration in seconds by default
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
	}

	// Default to string
	return s
}

func (p *Parser) errorf(format string, args ...interface{}) error {
	return fmt.Errorf("line %d: %s", p.line, fmt.Sprintf(format, args...))
}

// Helper to check if a string is a valid identifier
func isValidIdent(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
		} else {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return false
			}
		}
	}
	return true
}
