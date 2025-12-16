package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// Parser parses .schema files
type Parser struct {
	scanner *bufio.Scanner
	line    int
	file    *SchemaFile
}

// Parse parses a .schema file from a reader
func Parse(r io.Reader) (*SchemaFile, error) {
	p := &Parser{
		scanner: bufio.NewScanner(r),
		file: &SchemaFile{
			Types: make([]*TypeDef, 0),
			Views: make([]*ViewDef, 0),
		},
	}

	return p.parse()
}

func (p *Parser) parse() (*SchemaFile, error) {
	for p.scanner.Scan() {
		p.line++
		line := strings.TrimSpace(p.scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse directives and type definitions
		if strings.HasPrefix(line, "package ") {
			p.file.Package = strings.TrimPrefix(line, "package ")
			continue
		}

		if strings.HasPrefix(line, "@id(") {
			// Type definition with ID annotation
			id, err := p.parseIDAnnotation(line)
			if err != nil {
				return nil, p.errorf("invalid @id annotation: %v", err)
			}

			// Next non-empty line should be "type Name {"
			typeDef, err := p.parseType(id)
			if err != nil {
				return nil, err
			}
			p.file.Types = append(p.file.Types, typeDef)
			continue
		}

		if strings.HasPrefix(line, "type ") {
			// Type definition without explicit ID (auto-assign)
			typeDef, err := p.parseTypeFromLine(line, len(p.file.Types)+1)
			if err != nil {
				return nil, err
			}
			p.file.Types = append(p.file.Types, typeDef)
			continue
		}

		if strings.HasPrefix(line, "view ") {
			viewDef, err := p.parseView(line)
			if err != nil {
				return nil, err
			}
			p.file.Views = append(p.file.Views, viewDef)
			continue
		}
	}

	if err := p.scanner.Err(); err != nil {
		return nil, err
	}

	return p.file, nil
}

func (p *Parser) parseIDAnnotation(line string) (int, error) {
	// Parse @id(123)
	start := strings.Index(line, "(")
	end := strings.Index(line, ")")
	if start == -1 || end == -1 || end <= start {
		return 0, fmt.Errorf("invalid @id syntax")
	}
	idStr := strings.TrimSpace(line[start+1 : end])
	return strconv.Atoi(idStr)
}

func (p *Parser) parseType(id int) (*TypeDef, error) {
	// Find next non-empty line with "type Name {"
	for p.scanner.Scan() {
		p.line++
		line := strings.TrimSpace(p.scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		return p.parseTypeFromLine(line, id)
	}
	return nil, p.errorf("unexpected end of file after @id")
}

func (p *Parser) parseTypeFromLine(line string, id int) (*TypeDef, error) {
	// Parse "type Name {" or "type Name {"
	if !strings.HasPrefix(line, "type ") {
		return nil, p.errorf("expected 'type', got: %s", line)
	}

	rest := strings.TrimPrefix(line, "type ")
	rest = strings.TrimSpace(rest)

	// Extract name
	nameEnd := strings.IndexAny(rest, " {")
	if nameEnd == -1 {
		return nil, p.errorf("invalid type definition: %s", line)
	}
	name := rest[:nameEnd]

	typeDef := &TypeDef{
		Name:   name,
		ID:     id,
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

		if fieldLine == "" || strings.HasPrefix(fieldLine, "//") {
			continue
		}
		if fieldLine == "}" {
			break
		}

		field, err := p.parseField(fieldLine)
		if err != nil {
			return nil, err
		}
		typeDef.Fields = append(typeDef.Fields, field)
	}

	return typeDef, nil
}

func (p *Parser) parseField(line string) (*FieldDef, error) {
	// Parse: "name  type  @key(ID) @view(owner)"
	// Split by whitespace first
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil, p.errorf("invalid field definition: %s", line)
	}

	field := &FieldDef{
		Name:  parts[0],
		Type:  parts[1],
		Views: []string{}, // Default: visible to all
	}

	// Parse annotations
	for i := 2; i < len(parts); i++ {
		ann := parts[i]

		if strings.HasPrefix(ann, "@key(") {
			end := strings.Index(ann, ")")
			if end == -1 {
				return nil, p.errorf("invalid @key annotation: %s", ann)
			}
			field.Key = ann[5:end]
			continue
		}

		if strings.HasPrefix(ann, "@view(") {
			end := strings.Index(ann, ")")
			if end == -1 {
				return nil, p.errorf("invalid @view annotation: %s", ann)
			}
			views := ann[6:end]
			// Split by comma if multiple views
			for _, v := range strings.Split(views, ",") {
				field.Views = append(field.Views, strings.TrimSpace(v))
			}
			continue
		}

		if ann == "@optional" {
			field.Optional = true
			continue
		}
	}

	return field, nil
}

func (p *Parser) parseView(line string) (*ViewDef, error) {
	// Parse: "view name { includes: other }" or "view name {}"
	rest := strings.TrimPrefix(line, "view ")
	rest = strings.TrimSpace(rest)

	// Find name
	nameEnd := strings.IndexAny(rest, " {")
	if nameEnd == -1 {
		return nil, p.errorf("invalid view definition: %s", line)
	}
	name := rest[:nameEnd]

	view := &ViewDef{
		Name:     name,
		Includes: []string{},
	}

	// Check for includes
	if strings.Contains(rest, "includes:") {
		start := strings.Index(rest, "includes:")
		end := strings.Index(rest, "}")
		if end == -1 {
			end = len(rest)
		}
		includesStr := strings.TrimSpace(rest[start+9 : end])
		for _, inc := range strings.Split(includesStr, ",") {
			inc = strings.TrimSpace(inc)
			if inc != "" {
				view.Includes = append(view.Includes, inc)
			}
		}
	}

	// If view definition spans multiple lines, consume until }
	if !strings.Contains(line, "}") {
		for p.scanner.Scan() {
			p.line++
			viewLine := strings.TrimSpace(p.scanner.Text())
			if viewLine == "}" {
				break
			}
			if strings.HasPrefix(viewLine, "includes:") {
				incStr := strings.TrimPrefix(viewLine, "includes:")
				incStr = strings.TrimSpace(incStr)
				for _, inc := range strings.Split(incStr, ",") {
					inc = strings.TrimSpace(inc)
					if inc != "" {
						view.Includes = append(view.Includes, inc)
					}
				}
			}
		}
	}

	return view, nil
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
