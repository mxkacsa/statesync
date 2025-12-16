// trackgen generates Trackable implementations for Go structs.
//
// Usage:
//
//	//go:generate trackgen -type=GameState,Player
//
// This will generate tracked wrappers with automatic change detection.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

var (
	typeNames = flag.String("type", "", "comma-separated list of type names")
	output    = flag.String("output", "", "output file name; default srcdir/<type>_tracked.go")
)

func main() {
	flag.Parse()

	if *typeNames == "" {
		fmt.Fprintln(os.Stderr, "trackgen: -type flag is required")
		os.Exit(1)
	}

	types := strings.Split(*typeNames, ",")
	for i := range types {
		types[i] = strings.TrimSpace(types[i])
	}

	// Find the package directory
	dir := "."
	if args := flag.Args(); len(args) > 0 {
		dir = args[0]
	}

	g := &Generator{
		types: make(map[string]*TypeInfo),
	}

	if err := g.parsePackage(dir); err != nil {
		fmt.Fprintf(os.Stderr, "trackgen: %v\n", err)
		os.Exit(1)
	}

	for _, typeName := range types {
		if _, ok := g.types[typeName]; !ok {
			fmt.Fprintf(os.Stderr, "trackgen: type %q not found\n", typeName)
			os.Exit(1)
		}
	}

	var buf bytes.Buffer
	if err := g.generate(&buf, types); err != nil {
		fmt.Fprintf(os.Stderr, "trackgen: %v\n", err)
		os.Exit(1)
	}

	// Format the output
	src, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "trackgen: format error: %v\n%s\n", err, buf.String())
		os.Exit(1)
	}

	outputName := *output
	if outputName == "" {
		baseName := strings.ToLower(types[0]) + "_tracked.go"
		outputName = filepath.Join(dir, baseName)
	}

	if err := os.WriteFile(outputName, src, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "trackgen: %v\n", err)
		os.Exit(1)
	}
}

// Generator collects type information and generates code
type Generator struct {
	pkg   string
	types map[string]*TypeInfo
}

// TypeInfo holds parsed information about a type
type TypeInfo struct {
	Name   string
	Fields []FieldInfo

	// Visibility support
	IdentityField    string   // Field name that identifies the owner
	TeamKeyField     string   // Field name that identifies the team
	HasVisibility    bool     // True if any field has visibility rules
	CustomPredicates []string // List of custom predicate names used
}

// FieldInfo holds parsed information about a field
type FieldInfo struct {
	Name      string
	Type      string
	Index     int
	IsPointer bool
	IsSlice   bool
	IsMap     bool
	ElemType  string
	KeyType   string
	KeyField  string // For array key-based tracking
	Tag       string

	// Visibility support
	Visibility string // "self", "team", "public", "private", or custom predicate name
	IsIdentity bool   // This field identifies the owner (for "self" checks)
	IsTeamKey  bool   // This field identifies the team (for "team" checks)
	OwnerField string // For nested visibility: which sub-field contains owner ID
}

func (g *Generator) parsePackage(dir string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for pkgName, pkg := range pkgs {
		// Skip test packages
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}
		g.pkg = pkgName

		for _, file := range pkg.Files {
			g.parseFile(file)
		}
	}

	return nil
}

func (g *Generator) parseFile(file *ast.File) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			typeInfo := &TypeInfo{
				Name:   typeSpec.Name.Name,
				Fields: make([]FieldInfo, 0),
			}

			fieldIndex := 0
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue // Skip embedded fields
				}

				for _, name := range field.Names {
					if !ast.IsExported(name.Name) {
						continue // Skip unexported fields
					}

					fi := FieldInfo{
						Name:  name.Name,
						Index: fieldIndex,
					}

					// Parse field type
					fi.Type, fi.IsPointer, fi.IsSlice, fi.IsMap, fi.ElemType, fi.KeyType = parseFieldType(field.Type)

					// Parse struct tag for track directive
					if field.Tag != nil {
						tag := strings.Trim(field.Tag.Value, "`")
						fi.Tag = tag
						fi.KeyField = parseKeyField(tag)

						// Check for explicit index in tag
						if idx := parseTrackIndex(tag); idx >= 0 {
							fi.Index = idx
						}

						// Parse visibility tags
						fi.Visibility = parseVisibility(tag)
						fi.IsIdentity = parseTagBool(tag, "identity")
						fi.IsTeamKey = parseTagBool(tag, "teamKey")
						fi.OwnerField = parseTagValue(tag, "owner")
					}

					typeInfo.Fields = append(typeInfo.Fields, fi)
					fieldIndex++
				}
			}

			// Post-process: find identity/team fields and collect visibility info
			for _, f := range typeInfo.Fields {
				if f.IsIdentity {
					typeInfo.IdentityField = f.Name
				}
				if f.IsTeamKey {
					typeInfo.TeamKeyField = f.Name
				}
				if f.Visibility != "" && f.Visibility != "public" {
					typeInfo.HasVisibility = true
					// Collect custom predicates
					if !isBuiltinVisibility(f.Visibility) {
						if !containsString(typeInfo.CustomPredicates, f.Visibility) {
							typeInfo.CustomPredicates = append(typeInfo.CustomPredicates, f.Visibility)
						}
					}
				}
			}

			g.types[typeInfo.Name] = typeInfo
		}
	}
}

func parseFieldType(expr ast.Expr) (typeName string, isPtr, isSlice, isMap bool, elemType, keyType string) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, false, false, false, "", ""

	case *ast.StarExpr:
		inner, _, _, _, _, _ := parseFieldType(t.X)
		return "*" + inner, true, false, false, "", ""

	case *ast.ArrayType:
		elem, _, _, _, _, _ := parseFieldType(t.Elt)
		return "[]" + elem, false, true, false, elem, ""

	case *ast.MapType:
		key, _, _, _, _, _ := parseFieldType(t.Key)
		val, _, _, _, _, _ := parseFieldType(t.Value)
		return "map[" + key + "]" + val, false, false, true, val, key

	case *ast.SelectorExpr:
		pkg, _, _, _, _, _ := parseFieldType(t.X)
		return pkg + "." + t.Sel.Name, false, false, false, "", ""

	default:
		return "interface{}", false, false, false, "", ""
	}
}

func parseTrackIndex(tag string) int {
	// Parse `track:"0"` or `track:"0,key=ID"`
	for _, part := range strings.Split(tag, " ") {
		if strings.HasPrefix(part, "track:") {
			val := strings.Trim(strings.TrimPrefix(part, "track:"), "\"")
			parts := strings.Split(val, ",")
			if len(parts) > 0 {
				if idx, err := strconv.Atoi(parts[0]); err == nil {
					return idx
				}
			}
		}
	}
	return -1
}

func parseKeyField(tag string) string {
	// Parse `track:"0,key=ID"`
	for _, part := range strings.Split(tag, " ") {
		if strings.HasPrefix(part, "track:") {
			val := strings.Trim(strings.TrimPrefix(part, "track:"), "\"")
			for _, subpart := range strings.Split(val, ",") {
				if strings.HasPrefix(subpart, "key=") {
					return strings.TrimPrefix(subpart, "key=")
				}
			}
		}
	}
	return ""
}

// parseVisibility extracts the visibility tag value
// Examples: `visible:"self"`, `visible:"team"`, `visible:"customPredicate"`
func parseVisibility(tag string) string {
	return parseTagValue(tag, "visible")
}

// parseTagValue extracts a string value from a tag
// Example: `owner:"PlayerID"` -> "PlayerID"
func parseTagValue(tag, key string) string {
	prefix := key + ":"
	for _, part := range strings.Split(tag, " ") {
		if strings.HasPrefix(part, prefix) {
			val := strings.TrimPrefix(part, prefix)
			return strings.Trim(val, "\"")
		}
	}
	return ""
}

// parseTagBool checks if a boolean tag is present and true
// Example: `identity:"true"` -> true
func parseTagBool(tag, key string) bool {
	val := parseTagValue(tag, key)
	return val == "true" || val == "1" || val == key // `identity:"true"` or just `identity:"identity"`
}

// isBuiltinVisibility checks if the visibility is a built-in type
func isBuiltinVisibility(v string) bool {
	switch v {
	case "self", "team", "public", "private", "self|team", "team|self":
		return true
	}
	return false
}

// containsString checks if a slice contains a string
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func (g *Generator) generate(buf *bytes.Buffer, types []string) error {
	tmpl, err := template.New("tracked").Funcs(template.FuncMap{
		"lower":           strings.ToLower,
		"title":           strings.Title,
		"goFieldType":     goFieldType,
		"schemaType":      schemaType,
		"needsChildType":  needsChildType,
		"isTrackable":     func(t string) bool { return g.types[t] != nil },
		"hasVisibility":   func(t *TypeInfo) bool { return t.HasVisibility },
		"isBuiltinVis":    isBuiltinVisibility,
		"zeroValue":       zeroValue,
		"visibilityCheck": visibilityCheck,
		"needsCloneField": needsCloneField,
	}).Parse(trackedTemplate)
	if err != nil {
		return err
	}

	data := struct {
		Package string
		Types   []*TypeInfo
	}{
		Package: g.pkg,
		Types:   make([]*TypeInfo, 0, len(types)),
	}

	for _, name := range types {
		data.Types = append(data.Types, g.types[name])
	}

	return tmpl.Execute(buf, data)
}

func goFieldType(fi FieldInfo) string {
	return fi.Type
}

func schemaType(fi FieldInfo) string {
	baseType := fi.Type
	if fi.IsPointer {
		baseType = strings.TrimPrefix(baseType, "*")
	}

	if fi.IsSlice {
		return "statediff.TypeArray"
	}
	if fi.IsMap {
		return "statediff.TypeMap"
	}

	switch baseType {
	case "int8":
		return "statediff.TypeInt8"
	case "int16":
		return "statediff.TypeInt16"
	case "int32":
		return "statediff.TypeInt32"
	case "int", "int64":
		return "statediff.TypeInt64"
	case "uint8", "byte":
		return "statediff.TypeUint8"
	case "uint16":
		return "statediff.TypeUint16"
	case "uint32":
		return "statediff.TypeUint32"
	case "uint", "uint64":
		return "statediff.TypeUint64"
	case "float32":
		return "statediff.TypeFloat32"
	case "float64":
		return "statediff.TypeFloat64"
	case "string":
		return "statediff.TypeString"
	case "bool":
		return "statediff.TypeBool"
	case "[]byte":
		return "statediff.TypeBytes"
	default:
		return "statediff.TypeStruct"
	}
}

func needsChildType(fi FieldInfo) bool {
	if fi.IsSlice {
		switch fi.ElemType {
		case "int8", "int16", "int32", "int", "int64",
			"uint8", "uint16", "uint32", "uint", "uint64",
			"float32", "float64", "string", "bool", "byte":
			return false
		}
		return true
	}
	if fi.IsMap {
		switch fi.ElemType {
		case "int8", "int16", "int32", "int", "int64",
			"uint8", "uint16", "uint32", "uint", "uint64",
			"float32", "float64", "string", "bool", "byte":
			return false
		}
		return true
	}
	return false
}

// zeroValue returns the zero value for a Go type
func zeroValue(fi FieldInfo) string {
	if fi.IsPointer {
		return "nil"
	}
	if fi.IsSlice {
		return "nil"
	}
	if fi.IsMap {
		return "nil"
	}

	baseType := fi.Type
	switch baseType {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "byte":
		return "0"
	case "string":
		return `""`
	case "bool":
		return "false"
	default:
		// Struct type - return empty struct
		return fi.Type + "{}"
	}
}

// visibilityCheck generates the condition code for a visibility check
func visibilityCheck(fi FieldInfo, ownerExpr string) string {
	switch fi.Visibility {
	case "self":
		if fi.OwnerField != "" {
			// Nested: check sub-field
			return ownerExpr + "." + fi.OwnerField + " == ctx.ViewerID"
		}
		return ownerExpr + " == ctx.ViewerID"
	case "team":
		return "ctx.ViewerTeam != \"\" && " + ownerExpr + " == ctx.ViewerTeam"
	case "self|team", "team|self":
		if fi.OwnerField != "" {
			return "(" + ownerExpr + "." + fi.OwnerField + " == ctx.ViewerID || (ctx.ViewerTeam != \"\" && " + ownerExpr + " == ctx.ViewerTeam))"
		}
		return "(" + ownerExpr + " == ctx.ViewerID || ctx.ViewerTeam != \"\")"
	case "private":
		return "false"
	case "public", "":
		return "true"
	default:
		// Custom predicate
		return "ctx." + strings.Title(fi.Visibility) + " != nil && ctx." + strings.Title(fi.Visibility) + "(ctx.ViewerID)"
	}
}

// needsCloneField checks if a field needs to be explicitly cloned (slices/maps)
func needsCloneField(fi FieldInfo) bool {
	return fi.IsSlice || fi.IsMap
}

const trackedTemplate = `// Code generated by trackgen. DO NOT EDIT.

package {{.Package}}

import (
	"sync"

	"statediff"
)

{{range .Types}}
{{$type := .}}
// Tracked{{.Name}} is a change-tracking wrapper for {{.Name}}
type Tracked{{.Name}} struct {
	mu      sync.RWMutex
	data    {{.Name}}
	changes *statediff.ChangeSet
	schema  *statediff.Schema
}

// New{{.Name}} creates a new tracked {{.Name}}
func NewTracked{{.Name}}(initial {{.Name}}) *Tracked{{.Name}} {
	t := &Tracked{{.Name}}{
		data:    initial,
		changes: statediff.NewChangeSet(),
		schema:  {{.Name}}Schema(),
	}
	return t
}

// {{.Name}}Schema returns the schema for {{.Name}}
func {{.Name}}Schema() *statediff.Schema {
	return statediff.NewSchemaBuilder("{{.Name}}").
		{{- range .Fields}}
		{{- if eq (schemaType .) "statediff.TypeArray"}}
		Array("{{.Name}}", {{schemaType .}}, {{if needsChildType .}}{{.ElemType}}Schema(){{else}}nil{{end}}).
		{{- else if eq (schemaType .) "statediff.TypeMap"}}
		Map("{{.Name}}", {{schemaType .}}, {{if needsChildType .}}{{.ElemType}}Schema(){{else}}nil{{end}}).
		{{- else if eq (schemaType .) "statediff.TypeStruct"}}
		Struct("{{.Name}}", {{.Type}}Schema()).
		{{- else}}
		{{- $st := schemaType .}}
		{{- if eq $st "statediff.TypeInt8"}}Int8{{else if eq $st "statediff.TypeInt16"}}Int16{{else if eq $st "statediff.TypeInt32"}}Int32{{else if eq $st "statediff.TypeInt64"}}Int64{{else if eq $st "statediff.TypeUint8"}}Uint8{{else if eq $st "statediff.TypeUint16"}}Uint16{{else if eq $st "statediff.TypeUint32"}}Uint32{{else if eq $st "statediff.TypeUint64"}}Uint64{{else if eq $st "statediff.TypeFloat32"}}Float32{{else if eq $st "statediff.TypeFloat64"}}Float64{{else if eq $st "statediff.TypeString"}}String{{else if eq $st "statediff.TypeBool"}}Bool{{else}}String{{end}}("{{.Name}}").
		{{- end}}
		{{- end}}
		Build()
}

// Data returns the underlying data (read-only)
func (t *Tracked{{.Name}}) Data() {{.Name}} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.data
}

// Schema implements Trackable
func (t *Tracked{{.Name}}) Schema() *statediff.Schema {
	return t.schema
}

// Changes implements Trackable
func (t *Tracked{{.Name}}) Changes() *statediff.ChangeSet {
	return t.changes
}

// ClearChanges implements Trackable
func (t *Tracked{{.Name}}) ClearChanges() {
	t.changes.Clear()
}

// MarkAllDirty implements Trackable
func (t *Tracked{{.Name}}) MarkAllDirty() {
	t.changes.MarkAll({{len .Fields | printf "%d"}})
}

// GetFieldValue implements Trackable
func (t *Tracked{{.Name}}) GetFieldValue(index uint8) interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	switch index {
	{{- range .Fields}}
	case {{.Index}}:
		return t.data.{{.Name}}
	{{- end}}
	}
	return nil
}

// Getters and Setters
{{range .Fields}}
// {{.Name}} returns the current value
func (t *Tracked{{$type.Name}}) {{.Name}}() {{goFieldType .}} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.data.{{.Name}}
}

{{if not .IsSlice}}{{if not .IsMap}}
// Set{{.Name}} sets the value and marks it as changed
func (t *Tracked{{$type.Name}}) Set{{.Name}}(v {{goFieldType .}}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.data.{{.Name}} != v {
		t.data.{{.Name}} = v
		t.changes.Mark({{.Index}}, statediff.OpReplace)
	}
}
{{end}}{{end}}

{{if .IsSlice}}
// Set{{.Name}} replaces the entire slice
func (t *Tracked{{$type.Name}}) Set{{.Name}}(v {{goFieldType .}}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data.{{.Name}} = v
	t.changes.Mark({{.Index}}, statediff.OpReplace)
}

// Append{{.Name}} adds an element to the slice
func (t *Tracked{{$type.Name}}) Append{{.Name}}(v {{.ElemType}}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data.{{.Name}} = append(t.data.{{.Name}}, v)
	arr := t.changes.GetOrCreateArray({{.Index}})
	arr.MarkAdd(len(t.data.{{.Name}})-1, v)
}

// Remove{{.Name}}At removes an element at index
func (t *Tracked{{$type.Name}}) Remove{{.Name}}At(index int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if index >= 0 && index < len(t.data.{{.Name}}) {
		t.data.{{.Name}} = append(t.data.{{.Name}}[:index], t.data.{{.Name}}[index+1:]...)
		arr := t.changes.GetOrCreateArray({{.Index}})
		arr.MarkRemove(index)
	}
}

// Update{{.Name}}At updates an element at index
func (t *Tracked{{$type.Name}}) Update{{.Name}}At(index int, v {{.ElemType}}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if index >= 0 && index < len(t.data.{{.Name}}) {
		t.data.{{.Name}}[index] = v
		arr := t.changes.GetOrCreateArray({{.Index}})
		arr.MarkReplace(index, v)
	}
}
{{end}}

{{if .IsMap}}
// Set{{.Name}} replaces the entire map
func (t *Tracked{{$type.Name}}) Set{{.Name}}(v {{goFieldType .}}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data.{{.Name}} = v
	t.changes.Mark({{.Index}}, statediff.OpReplace)
}

// Set{{.Name}}Key sets a map key
func (t *Tracked{{$type.Name}}) Set{{.Name}}Key(key {{.KeyType}}, v {{.ElemType}}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.data.{{.Name}} == nil {
		t.data.{{.Name}} = make({{goFieldType .}})
	}
	_, existed := t.data.{{.Name}}[key]
	t.data.{{.Name}}[key] = v
	m := t.changes.GetOrCreateMap({{.Index}})
	if existed {
		m.MarkReplace(key, v)
	} else {
		m.MarkAdd(key, v)
	}
}

// Delete{{.Name}}Key deletes a map key
func (t *Tracked{{$type.Name}}) Delete{{.Name}}Key(key {{.KeyType}}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.data.{{.Name}} != nil {
		if _, ok := t.data.{{.Name}}[key]; ok {
			delete(t.data.{{.Name}}, key)
			m := t.changes.GetOrCreateMap({{.Index}})
			m.MarkRemove(key)
		}
	}
}
{{end}}
{{end}}

// ==================== Visibility Support ====================

// {{.Name}}VisibilityCtx defines the visibility context for {{.Name}}
// ViewerID: The ID of the player viewing the state
// ViewerTeam: The team of the viewer (optional, for team-based visibility)
// Custom predicates are function fields that return true if the viewer can see the data
type {{.Name}}VisibilityCtx struct {
	ViewerID   string
	ViewerTeam string
{{- range .CustomPredicates}}
	{{. | title}} func(viewerID string) bool
{{- end}}
}

{{if .HasVisibility}}
// FilterFor returns a filtered copy by value (zero allocation when possible)
// Only clones slices/maps that will be kept visible
func (s *{{.Name}}) FilterFor(ctx *{{.Name}}VisibilityCtx) {{.Name}} {
	if ctx == nil {
		return *s
	}
{{if .IdentityField}}
	// Self sees everything
	if s.{{.IdentityField}} == ctx.ViewerID {
		return *s
	}
{{end}}
	// Start with shallow copy (no slice/map allocation yet)
	filtered := *s

	// Determine visibility BEFORE any slice cloning
{{- range .Fields}}
{{- if eq .Visibility "self"}}
	// {{.Name}}: visible:"self" - not self, so hide
	filtered.{{.Name}} = {{zeroValue .}}
{{- else if eq .Visibility "team"}}
	// {{.Name}}: visible:"team"
{{- if $type.TeamKeyField}}
	if ctx.ViewerTeam == "" || s.{{$type.TeamKeyField}} != ctx.ViewerTeam {
		filtered.{{.Name}} = {{zeroValue .}}
	}{{if or .IsSlice .IsMap}} else if s.{{.Name}} != nil {
		{{- if .IsSlice}}
		filtered.{{.Name}} = make({{goFieldType .}}, len(s.{{.Name}}))
		copy(filtered.{{.Name}}, s.{{.Name}})
		{{- else if .IsMap}}
		filtered.{{.Name}} = make({{goFieldType .}}, len(s.{{.Name}}))
		for k, v := range s.{{.Name}} { filtered.{{.Name}}[k] = v }
		{{- end}}
	}{{end}}
{{- end}}
{{- else if or (eq .Visibility "self|team") (eq .Visibility "team|self")}}
	// {{.Name}}: visible:"self|team" - not self, check team
{{- if $type.TeamKeyField}}
	if ctx.ViewerTeam == "" || s.{{$type.TeamKeyField}} != ctx.ViewerTeam {
		filtered.{{.Name}} = {{zeroValue .}}
	}{{if or .IsSlice .IsMap}} else if s.{{.Name}} != nil {
		{{- if .IsSlice}}
		filtered.{{.Name}} = make({{goFieldType .}}, len(s.{{.Name}}))
		copy(filtered.{{.Name}}, s.{{.Name}})
		{{- else if .IsMap}}
		filtered.{{.Name}} = make({{goFieldType .}}, len(s.{{.Name}}))
		for k, v := range s.{{.Name}} { filtered.{{.Name}}[k] = v }
		{{- end}}
	}{{end}}
{{- else}}
	filtered.{{.Name}} = {{zeroValue .}}
{{- end}}
{{- else if eq .Visibility "private"}}
	// {{.Name}}: visible:"private" - always hidden
	filtered.{{.Name}} = {{zeroValue .}}
{{- else if and .Visibility (not (eq .Visibility "public"))}}
	// {{.Name}}: visible:"{{.Visibility}}" - custom predicate
	if ctx.{{.Visibility | title}} == nil || !ctx.{{.Visibility | title}}(ctx.ViewerID) {
		filtered.{{.Name}} = {{zeroValue .}}
	}{{if or .IsSlice .IsMap}} else if s.{{.Name}} != nil {
		{{- if .IsSlice}}
		filtered.{{.Name}} = make({{goFieldType .}}, len(s.{{.Name}}))
		copy(filtered.{{.Name}}, s.{{.Name}})
		{{- else if .IsMap}}
		filtered.{{.Name}} = make({{goFieldType .}}, len(s.{{.Name}}))
		for k, v := range s.{{.Name}} { filtered.{{.Name}}[k] = v }
		{{- end}}
	}{{end}}
{{- end}}
{{- end}}

	return filtered
}

// FilterSliceTo filters a slice into the provided output buffer (zero allocation if reused)
// Pass nil for out to allocate a new slice
func Filter{{.Name}}SliceTo(items []{{.Name}}, ctx *{{.Name}}VisibilityCtx, out []{{.Name}}) []{{.Name}} {
	if ctx == nil || len(items) == 0 {
		return items
	}
	if cap(out) >= len(items) {
		out = out[:len(items)]
	} else {
		out = make([]{{.Name}}, len(items))
	}
	for i := range items {
		out[i] = items[i].FilterFor(ctx)
	}
	return out
}

// FilterSliceFor filters a slice (allocates new output slice)
func Filter{{.Name}}SliceFor(items []{{.Name}}, ctx *{{.Name}}VisibilityCtx) []{{.Name}} {
	return Filter{{.Name}}SliceTo(items, ctx, nil)
}

// FilterForInPlace modifies the struct in place (zero allocation)
// WARNING: Destroys original data - only use if you don't need the original
func (s *{{.Name}}) FilterForInPlace(ctx *{{.Name}}VisibilityCtx) {
	if ctx == nil {
		return
	}
{{if .IdentityField}}
	if s.{{.IdentityField}} == ctx.ViewerID {
		return
	}
{{end}}
{{- range .Fields}}
{{- if eq .Visibility "self"}}
	s.{{.Name}} = {{zeroValue .}}
{{- else if eq .Visibility "team"}}
{{- if $type.TeamKeyField}}
	if ctx.ViewerTeam == "" || s.{{$type.TeamKeyField}} != ctx.ViewerTeam {
		s.{{.Name}} = {{zeroValue .}}
	}
{{- end}}
{{- else if or (eq .Visibility "self|team") (eq .Visibility "team|self")}}
{{- if $type.TeamKeyField}}
	if ctx.ViewerTeam == "" || s.{{$type.TeamKeyField}} != ctx.ViewerTeam {
		s.{{.Name}} = {{zeroValue .}}
	}
{{- else}}
	s.{{.Name}} = {{zeroValue .}}
{{- end}}
{{- else if eq .Visibility "private"}}
	s.{{.Name}} = {{zeroValue .}}
{{- else if and .Visibility (not (eq .Visibility "public"))}}
	if ctx.{{.Visibility | title}} == nil || !ctx.{{.Visibility | title}}(ctx.ViewerID) {
		s.{{.Name}} = {{zeroValue .}}
	}
{{- end}}
{{- end}}
}
{{else}}
// FilterFor returns a copy (no visibility rules defined)
func (s *{{.Name}}) FilterFor(ctx *{{.Name}}VisibilityCtx) {{.Name}} {
	return *s
}

// FilterSliceFor returns the slice as-is (no visibility rules)
func Filter{{.Name}}SliceFor(items []{{.Name}}, ctx *{{.Name}}VisibilityCtx) []{{.Name}} {
	return items
}
{{end}}

{{end}}
`
