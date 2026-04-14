package main

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"
)

// encoderMethod returns the Encoder.WriteXxx method name for a type
func encoderMethod(typ string) string {
	switch typ {
	case "int8":
		return "WriteInt8"
	case "int16":
		return "WriteInt16"
	case "int32":
		return "WriteInt32"
	case "int", "int64":
		return "WriteInt64"
	case "uint8", "byte":
		return "WriteUint8"
	case "uint16":
		return "WriteUint16"
	case "uint32":
		return "WriteUint32"
	case "uint", "uint64":
		return "WriteUint64"
	case "float32":
		return "WriteFloat32"
	case "float64":
		return "WriteFloat64"
	case "string", "uuid":
		return "WriteString"
	case "bool":
		return "WriteBool"
	case "bytes", "[]byte":
		return "WriteBytes"
	default:
		return "" // Complex types (map/array/struct) need slow-path encoding
	}
}

// goDefaultValue generates Go code for the default value of a field
func goDefaultValue(f *FieldDef) string {
	pt := ParseType(f.Type)

	// Array or map without explicit default -> nil
	if pt.IsArray || pt.IsMap {
		return "nil"
	}

	// Auto-generated UUID
	if f.AutoGen == AutoGenUUID {
		return "uuid.New().String()"
	}

	// No default source -> zero value
	if f.DefaultSource == "" || f.DefaultSource == DefaultNone {
		return goZeroValue(f.Type)
	}

	// Config reference: @default(config:GameConfig.Speed)
	if f.DefaultSource == DefaultConfig {
		parts := strings.Split(f.DefaultValue, ".")
		if len(parts) == 2 {
			return fmt.Sprintf("Get%s().%s", parts[0], parts[1])
		}
		return f.DefaultValue
	}

	// Literal value
	switch pt.BaseType {
	case "string", "uuid":
		return fmt.Sprintf("%q", f.DefaultValue)
	case "bool":
		return f.DefaultValue
	default:
		return f.DefaultValue
	}
}

// hasAutoGenUUID checks if any field uses auto-generated UUID
func hasAutoGenUUID(schema *SchemaFile) bool {
	for _, t := range schema.Types {
		for _, f := range t.Fields {
			if f.AutoGen == AutoGenUUID {
				return true
			}
		}
	}
	return false
}

// goZeroValue returns the Go zero value for a type
func goZeroValue(t string) string {
	pt := ParseType(t)

	if pt.IsArray || pt.IsMap {
		return "nil"
	}

	switch pt.BaseType {
	case "string", "uuid":
		return `""`
	case "bool":
		return "false"
	case "int8", "int16", "int32", "int64", "int",
		"uint8", "uint16", "uint32", "uint64", "uint":
		return "0"
	case "float32", "float64":
		return "0.0"
	default:
		return "nil"
	}
}

// hasConfigDefaults checks if any field uses config defaults
func hasConfigDefaults(schema *SchemaFile) bool {
	for _, t := range schema.Types {
		for _, f := range t.Fields {
			if f.DefaultSource == DefaultConfig {
				return true
			}
		}
	}
	return false
}

// typeHasComplexFields returns true if a type has any map, array, or struct fields
// that cannot be handled by the fast encoder path
func typeHasComplexFields(t *TypeDef) bool {
	for _, f := range t.Fields {
		if encoderMethod(f.Type) == "" {
			return true
		}
	}
	return false
}

// toCamelCase converts PascalCase to camelCase for JSON tags
// "Phase" → "phase", "HostID" → "hostId", "IsCaught" → "isCaught"
func toCamelCase(s string) string {
	if len(s) <= 1 {
		return strings.ToLower(s)
	}
	if strings.HasSuffix(s, "ID") {
		if len(s) == 2 {
			return "id"
		}
		s = s[:len(s)-2] + "Id"
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// jsonOmitEmpty returns true if the JSON tag should include omitempty
func jsonOmitEmpty(f *FieldDef) bool {
	pt := ParseType(f.Type)
	if f.Optional {
		return true
	}
	if f.Type == "bytes" || f.Type == "[]byte" {
		return true
	}
	if pt.IsMap || pt.IsArray {
		return true
	}
	return false
}

// jsonGoType returns the Go type to use in the JSON struct
// bytes → json.RawMessage, map[K]StructT → map[K]*StructT, everything else → same as goType.
// Rationale: Go's encoding/json cannot call pointer-receiver MarshalJSON on
// non-addressable map values, so a field typed as map[K]StructT would serialize
// every entry as "{}". Using pointer values makes each entry addressable so the
// generated MarshalJSON runs per entry.
func jsonGoType(f *FieldDef) string {
	if f.Type == "bytes" || f.Type == "[]byte" {
		return "json.RawMessage"
	}
	if mapValueIsStruct(f) {
		pt := ParseType(f.Type)
		return "map[" + GoType(pt.KeyType) + "]*" + GoType(pt.ElemType)
	}
	return GoType(f.Type)
}

// mapValueIsStruct reports whether a field is a map whose value type is a
// non-primitive, non-pointer struct — the shape that triggers the
// pointer-receiver MarshalJSON bug described on jsonGoType.
func mapValueIsStruct(f *FieldDef) bool {
	pt := ParseType(f.Type)
	if !pt.IsMap {
		return false
	}
	if IsPrimitive(pt.ElemType) {
		return false
	}
	if len(pt.ElemType) > 0 && pt.ElemType[0] == '*' {
		return false
	}
	return true
}

// getRootSchemas returns only root schemas
func getRootSchemas(schema *SchemaFile) []*TypeDef {
	var roots []*TypeDef
	for _, t := range schema.Types {
		if t.Role == RoleRoot {
			roots = append(roots, t)
		}
	}
	return roots
}

// GenerateGo generates Go code from a schema file
func GenerateGo(schema *SchemaFile) ([]byte, error) {
	tmpl, err := template.New("go").Funcs(template.FuncMap{
		"goType":            GoType,
		"fieldType":         FieldTypeEnum,
		"parseType":         ParseType,
		"isPrimitive":       IsPrimitive,
		"lower":             strings.ToLower,
		"sub":               func(a, b int) int { return a - b },
		"hasViewFilter":     func(f *FieldDef) bool { return len(f.Views) > 0 },
		"encoderMethod":     encoderMethod,
		"goDefaultValue":    goDefaultValue,
		"goZeroValue":       goZeroValue,
		"hasConfigDefaults": hasConfigDefaults,
		"hasAutoGenUUID":    hasAutoGenUUID,
		"mapValueIsStruct":  mapValueIsStruct,
		"getRootSchemas":    getRootSchemas,
		"isRoot":            func(t *TypeDef) bool { return t.Role == RoleRoot },
		"isActiveByDefault": func(t *TypeDef) bool { return t.DefaultState == "active" },
		"hasComplexFields":  typeHasComplexFields,
		"needsMutex":        func(t *TypeDef) bool { return t.Role == RoleRoot },
		"isSynced":          func(f *FieldDef) bool { return !f.NoSync },
		"syncedFields": func(t *TypeDef) []*FieldDef {
			var result []*FieldDef
			for _, f := range t.Fields {
				if !f.NoSync {
					result = append(result, f)
				}
			}
			return result
		},
		"maxSyncIndex": func(t *TypeDef) int {
			max := -1
			for _, f := range t.Fields {
				if !f.NoSync && f.SyncIndex > max {
					max = f.SyncIndex
				}
			}
			return max
		},
		"toCamelCase":       toCamelCase,
		"jsonOmitEmpty":     jsonOmitEmpty,
		"jsonGoType":        jsonGoType,
		"lowerFirst":        func(s string) string {
			if len(s) == 0 { return s }
			return strings.ToLower(s[:1]) + s[1:]
		},
	}).Parse(goTemplate)
	if err != nil {
		return nil, fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, schema); err != nil {
		return nil, fmt.Errorf("template execute error: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted with error info
		return nil, fmt.Errorf("format error: %w\n%s", err, buf.String())
	}

	return formatted, nil
}

const goTemplate = `// Code generated by schemagen. DO NOT EDIT.

package {{.Package}}

import (
	"encoding/json"
	"fmt"
	"sync"
{{if hasAutoGenUUID .}}
	"github.com/google/uuid"
{{end}}
	"github.com/mxkacsa/statesync"
)

// bytesToRawJSON converts a byte slice to json.RawMessage (nil if empty).
func bytesToRawJSON(b []byte) json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	return json.RawMessage(b)
}

// mapValuesToPtr converts a map[K]V to map[K]*V so Go's encoding/json can
// call pointer-receiver MarshalJSON on each entry (map values are not
// addressable, so value-typed entries would serialize as "{}").
func mapValuesToPtr[K comparable, V any](m map[K]V) map[K]*V {
	if m == nil {
		return nil
	}
	out := make(map[K]*V, len(m))
	for k, v := range m {
		vc := v
		out[k] = &vc
	}
	return out
}

// mapValuesFromPtr is the inverse of mapValuesToPtr, used during unmarshal.
func mapValuesFromPtr[K comparable, V any](m map[K]*V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		if v != nil {
			out[k] = *v
		} else {
			var zero V
			out[k] = zero
		}
	}
	return out
}

{{range $t := .Types}}
// {{$t.Name}} is a tracked state type
type {{$t.Name}} struct {
	{{- if needsMutex $t}}
	mu      sync.RWMutex
	{{- end}}
	changes *statesync.ChangeSet
	schema  *statesync.Schema

	{{range $t.Fields}}
	{{lower .Name}} {{goType .Type}}
	{{- end}}
}

// New{{$t.Name}} creates a new {{$t.Name}} with default values
func New{{$t.Name}}() *{{$t.Name}} {
	t := &{{$t.Name}}{
		changes: statesync.NewChangeSet(),
		schema:  {{$t.Name}}Schema(),
	}
	t.ResetToDefaults()
	return t
}

// ResetToDefaults resets all fields to their default values
func (t *{{$t.Name}}) ResetToDefaults() {
	{{- if needsMutex $t}}
	t.mu.Lock()
	defer t.mu.Unlock()
	{{- end}}
	{{- range $i, $f := $t.Fields}}
	{{- $pt := parseType $f.Type}}
	{{- if and (isRoot $t) $pt.IsMap}}
	t.{{lower $f.Name}} = make({{goType $f.Type}})
	{{- else if and (isRoot $t) $pt.IsArray}}
	t.{{lower $f.Name}} = make({{goType $f.Type}}, 0)
	{{- else}}
	t.{{lower $f.Name}} = {{goDefaultValue $f}}
	{{- end}}
	{{- end}}
	{{$max := maxSyncIndex $t}}{{if ge $max 0}}t.changes.MarkAll({{$max}}){{end}}
}

// {{$t.Name}}Schema returns the schema for {{$t.Name}}
func {{$t.Name}}Schema() *statesync.Schema {
	return statesync.NewSchemaBuilder("{{$t.Name}}").
		WithID({{$t.ID}}).
		{{- range $i, $f := $t.Fields}}
		{{- if isSynced $f}}
		{{- $pt := parseType $f.Type}}
		{{- if $pt.IsArray}}
		{{- if $f.Key}}
		ArrayByKey("{{$f.Name}}", statesync.{{fieldType $pt.ElemType}}, {{if not (isPrimitive $pt.ElemType)}}{{$pt.ElemType}}Schema(){{else}}nil{{end}}, "{{$f.Key}}").
		{{- else}}
		Array("{{$f.Name}}", statesync.{{fieldType $pt.ElemType}}, {{if not (isPrimitive $pt.ElemType)}}{{$pt.ElemType}}Schema(){{else}}nil{{end}}).
		{{- end}}
		{{- else if $pt.IsMap}}
		Map("{{$f.Name}}", statesync.{{fieldType $pt.ElemType}}, {{if not (isPrimitive $pt.ElemType)}}{{$pt.ElemType}}Schema(){{else}}nil{{end}}).
		{{- else}}
		{{- $ft := fieldType $f.Type}}
		{{- if eq $ft "TypeInt8"}}Int8{{else if eq $ft "TypeInt16"}}Int16{{else if eq $ft "TypeInt32"}}Int32{{else if eq $ft "TypeInt64"}}Int64{{else if eq $ft "TypeUint8"}}Uint8{{else if eq $ft "TypeUint16"}}Uint16{{else if eq $ft "TypeUint32"}}Uint32{{else if eq $ft "TypeUint64"}}Uint64{{else if eq $ft "TypeFloat32"}}Float32{{else if eq $ft "TypeFloat64"}}Float64{{else if eq $ft "TypeString"}}String{{else if eq $ft "TypeBool"}}Bool{{else if eq $ft "TypeBytes"}}Bytes{{else}}Struct{{end}}("{{$f.Name}}"{{if eq $ft "TypeStruct"}}, {{$f.Type}}Schema(){{end}}).
		{{- end}}
		{{- end}}
		{{- end}}
		Build()
}

// Trackable implementation

func (t *{{$t.Name}}) Schema() *statesync.Schema     { return t.schema }
func (t *{{$t.Name}}) Changes() *statesync.ChangeSet { return t.changes }
func (t *{{$t.Name}}) ClearChanges()                 { t.changes.Clear() }
func (t *{{$t.Name}}) MarkAllDirty()                 { {{$max := maxSyncIndex $t}}{{if ge $max 0}}t.changes.MarkAll({{$max}}){{end}} }

func (t *{{$t.Name}}) GetFieldValue(index uint8) interface{} {
	{{- if needsMutex $t}}
	t.mu.RLock()
	defer t.mu.RUnlock()
	{{- end}}
	switch index {
	{{- range $i, $f := $t.Fields}}
	{{- if isSynced $f}}
	case {{$f.SyncIndex}}:
		{{- $pt := parseType $f.Type}}
		{{- if or (eq $f.Type "int") (eq $f.Type "uint")}}
		return int64(t.{{lower $f.Name}})
		{{- else if $pt.IsMap}}
		if t.{{lower $f.Name}} == nil {
			return nil
		}
		cp := make({{goType $f.Type}}, len(t.{{lower $f.Name}}))
		for k, v := range t.{{lower $f.Name}} {
			cp[k] = v
		}
		return cp
		{{- else if $pt.IsArray}}
		if t.{{lower $f.Name}} == nil {
			return nil
		}
		cp := make({{goType $f.Type}}, len(t.{{lower $f.Name}}))
		copy(cp, t.{{lower $f.Name}})
		return cp
		{{- else}}
		return t.{{lower $f.Name}}
		{{- end}}
	{{- end}}
	{{- end}}
	}
	return nil
}

{{if not (hasComplexFields $t)}}
// FastEncoder implementation - zero allocation encoding
// Generated only for types with all primitive fields (no maps/arrays/structs)

func (t *{{$t.Name}}) EncodeChangesTo(e *statesync.Encoder) {
	{{- if needsMutex $t}}
	t.mu.RLock()
	defer t.mu.RUnlock()
	{{- end}}

	// Count and write number of changes
	changes := t.changes
	count := 0
	{{- range $i, $f := $t.Fields}}
	{{- if isSynced $f}}
	if changes.IsFieldDirty({{$f.SyncIndex}}) { count++ }
	{{- end}}
	{{- end}}
	e.WriteChangeCount(count)

	// Encode each changed field directly (no interface{} boxing)
	{{- range $i, $f := $t.Fields}}
	{{- if isSynced $f}}
	if changes.IsFieldDirty({{$f.SyncIndex}}) {
		e.WriteFieldHeader({{$f.SyncIndex}}, statesync.OpReplace)
		e.{{encoderMethod $f.Type}}(t.{{lower $f.Name}})
	}
	{{- end}}
	{{- end}}
}

func (t *{{$t.Name}}) EncodeAllTo(e *statesync.Encoder) {
	{{- if needsMutex $t}}
	t.mu.RLock()
	defer t.mu.RUnlock()
	{{- end}}

	// Encode all synced fields directly (no interface{} boxing)
	{{- range $i, $f := $t.Fields}}
	{{- if isSynced $f}}
	e.{{encoderMethod $f.Type}}(t.{{lower $f.Name}})
	{{- end}}
	{{- end}}
}
{{end}}

// Getters and Setters
{{range $i, $f := $t.Fields}}
{{- $pt := parseType $f.Type}}
{{- $goT := goType $f.Type}}
{{- $lower := lower $f.Name}}

// {{$f.Name}} returns the current value
{{- if and $pt.IsMap (needsMutex $t)}}
// Returns a snapshot copy of the map (safe for concurrent iteration)
{{- end}}
func (t *{{$t.Name}}) {{$f.Name}}() {{$goT}} {
	{{- if needsMutex $t}}
	t.mu.RLock()
	defer t.mu.RUnlock()
	{{- end}}
	{{- if $pt.IsMap}}
	if t.{{$lower}} == nil {
		return nil
	}
	cp := make({{$goT}}, len(t.{{$lower}}))
	for k, v := range t.{{$lower}} {
		cp[k] = v
	}
	return cp
	{{- else}}
	return t.{{$lower}}
	{{- end}}
}

{{if not $pt.IsArray}}{{if not $pt.IsMap}}
// Set{{$f.Name}} sets the value{{if isSynced $f}} and marks it as changed{{end}}
func (t *{{$t.Name}}) Set{{$f.Name}}(v {{$goT}}) {
	{{- if needsMutex $t}}
	t.mu.Lock()
	defer t.mu.Unlock()
	{{- end}}
	{{- if not (isSynced $f)}}
	t.{{$lower}} = v
	{{- else if and (isPrimitive $f.Type) (ne $f.Type "bytes")}}
	if t.{{$lower}} != v {
		t.{{$lower}} = v
		t.changes.Mark({{$f.SyncIndex}}, statesync.OpReplace)
	}
	{{- else}}
	t.{{$lower}} = v
	t.changes.Mark({{$f.SyncIndex}}, statesync.OpReplace)
	{{- end}}
}
{{end}}{{end}}

{{if $pt.IsArray}}
// Set{{$f.Name}} replaces the entire slice
func (t *{{$t.Name}}) Set{{$f.Name}}(v {{$goT}}) {
	{{- if needsMutex $t}}
	t.mu.Lock()
	defer t.mu.Unlock()
	{{- end}}
	t.{{$lower}} = v
	t.changes.Mark({{$f.SyncIndex}}, statesync.OpReplace)
}

// Append{{$f.Name}} adds an element to the slice
func (t *{{$t.Name}}) Append{{$f.Name}}(v {{$pt.ElemType}}) {
	{{- if needsMutex $t}}
	t.mu.Lock()
	defer t.mu.Unlock()
	{{- end}}
	t.{{$lower}} = append(t.{{$lower}}, v)
	arr := t.changes.GetOrCreateArray({{$f.SyncIndex}})
	arr.MarkAdd(len(t.{{$lower}})-1, v)
}

// Remove{{$f.Name}}At removes an element at index
func (t *{{$t.Name}}) Remove{{$f.Name}}At(index int) {
	{{- if needsMutex $t}}
	t.mu.Lock()
	defer t.mu.Unlock()
	{{- end}}
	if index >= 0 && index < len(t.{{$lower}}) {
		t.{{$lower}} = append(t.{{$lower}}[:index], t.{{$lower}}[index+1:]...)
		arr := t.changes.GetOrCreateArray({{$f.SyncIndex}})
		arr.MarkRemove(index)
	}
}

// Update{{$f.Name}}At updates an element at index
func (t *{{$t.Name}}) Update{{$f.Name}}At(index int, v {{$pt.ElemType}}) {
	{{- if needsMutex $t}}
	t.mu.Lock()
	defer t.mu.Unlock()
	{{- end}}
	if index >= 0 && index < len(t.{{$lower}}) {
		t.{{$lower}}[index] = v
		arr := t.changes.GetOrCreateArray({{$f.SyncIndex}})
		arr.MarkReplace(index, v)
	}
}

// {{$f.Name}}Len returns the length of the slice
func (t *{{$t.Name}}) {{$f.Name}}Len() int {
	{{- if needsMutex $t}}
	t.mu.RLock()
	defer t.mu.RUnlock()
	{{- end}}
	return len(t.{{$lower}})
}

// {{$f.Name}}At returns the element at index
func (t *{{$t.Name}}) {{$f.Name}}At(index int) {{$pt.ElemType}} {
	{{- if needsMutex $t}}
	t.mu.RLock()
	defer t.mu.RUnlock()
	{{- end}}
	if index >= 0 && index < len(t.{{$lower}}) {
		return t.{{$lower}}[index]
	}
	var zero {{$pt.ElemType}}
	return zero
}
{{end}}

{{if $pt.IsMap}}
// Set{{$f.Name}} replaces the entire map
func (t *{{$t.Name}}) Set{{$f.Name}}(v {{$goT}}) {
	{{- if needsMutex $t}}
	t.mu.Lock()
	defer t.mu.Unlock()
	{{- end}}
	t.{{$lower}} = v
	t.changes.Mark({{$f.SyncIndex}}, statesync.OpReplace)
}

// Set{{$f.Name}}Key sets a map key
func (t *{{$t.Name}}) Set{{$f.Name}}Key(key {{$pt.KeyType}}, v {{$pt.ElemType}}) {
	{{- if needsMutex $t}}
	t.mu.Lock()
	defer t.mu.Unlock()
	{{- end}}
	if t.{{$lower}} == nil {
		t.{{$lower}} = make({{$goT}})
	}
	_, existed := t.{{$lower}}[key]
	t.{{$lower}}[key] = v
	m := t.changes.GetOrCreateMap({{$f.SyncIndex}})
{{- if not (isPrimitive $pt.ElemType)}}
	vp := v
	if existed {
		m.MarkReplace(key, &vp)
	} else {
		m.MarkAdd(key, &vp)
	}
{{- else}}
	if existed {
		m.MarkReplace(key, v)
	} else {
		m.MarkAdd(key, v)
	}
{{- end}}
}

// Delete{{$f.Name}}Key deletes a map key
func (t *{{$t.Name}}) Delete{{$f.Name}}Key(key {{$pt.KeyType}}) {
	{{- if needsMutex $t}}
	t.mu.Lock()
	defer t.mu.Unlock()
	{{- end}}
	if t.{{$lower}} != nil {
		if _, ok := t.{{$lower}}[key]; ok {
			delete(t.{{$lower}}, key)
			m := t.changes.GetOrCreateMap({{$f.SyncIndex}})
			m.MarkRemove(key)
		}
	}
}

// {{$f.Name}}Get returns the value for a key
func (t *{{$t.Name}}) {{$f.Name}}Get(key {{$pt.KeyType}}) ({{$pt.ElemType}}, bool) {
	{{- if needsMutex $t}}
	t.mu.RLock()
	defer t.mu.RUnlock()
	{{- end}}
	if t.{{$lower}} == nil {
		var zero {{$pt.ElemType}}
		return zero, false
	}
	v, ok := t.{{$lower}}[key]
	return v, ok
}
{{- if not (isPrimitive $pt.ElemType)}}

// Modify{{$f.Name}}Key retrieves the value for a key, passes it to the callback for modification,
// then writes it back only if the callback made changes (detected via inner ChangeSet).
// Returns false if the key was not found.
func (t *{{$t.Name}}) Modify{{$f.Name}}Key(key {{$pt.KeyType}}, fn func(*{{$pt.ElemType}})) bool {
	{{- if needsMutex $t}}
	t.mu.Lock()
	defer t.mu.Unlock()
	{{- end}}
	if t.{{$lower}} == nil {
		return false
	}
	v, ok := t.{{$lower}}[key]
	if !ok {
		return false
	}
	v.Changes().Clear()
	fn(&v)
	if !v.Changes().HasChanges() {
		return true
	}
	t.{{$lower}}[key] = v
	{{- if isSynced $f}}
	m := t.changes.GetOrCreateMap({{$f.SyncIndex}})
	vp := v
	m.MarkReplace(key, &vp)
	{{- end}}
	return true
}
{{- end}}
{{end}}
{{end}}

{{if needsMutex $t}}
// ShallowClone creates a shallow copy of {{$t.Name}} suitable for projections/filters.
// The clone gets a deep-copied ChangeSet so projection modifications don't corrupt the original.
// Maps are shallow-copied (new map, same value entries). Slices are deep-copied.
func (s *{{$t.Name}}) ShallowClone() *{{$t.Name}} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &{{$t.Name}}{
		changes: s.changes.CloneForFilter(),
		schema:  s.schema,
		{{- range $i, $f := $t.Fields}}
		{{- $pt := parseType $f.Type}}
		{{- if not (or $pt.IsMap $pt.IsArray)}}
		{{lower $f.Name}}: s.{{lower $f.Name}},
		{{- end}}
		{{- end}}
	}

	{{- range $i, $f := $t.Fields}}
	{{- $pt := parseType $f.Type}}
	{{- if $pt.IsMap}}
	if s.{{lower $f.Name}} != nil {
		clone.{{lower $f.Name}} = make({{goType $f.Type}}, len(s.{{lower $f.Name}}))
		for k, v := range s.{{lower $f.Name}} {
			clone.{{lower $f.Name}}[k] = v
		}
	}
	{{- end}}
	{{- if $pt.IsArray}}
	if s.{{lower $f.Name}} != nil {
		clone.{{lower $f.Name}} = make({{goType $f.Type}}, len(s.{{lower $f.Name}}))
		copy(clone.{{lower $f.Name}}, s.{{lower $f.Name}})
	}
	{{- end}}
	{{- end}}

	return clone
}
{{end}}

// ---- JSON serialization ----

type {{lowerFirst $t.Name}}JSON struct {
	{{- range $i, $f := $t.Fields}}
	{{$f.Name}} {{jsonGoType $f}} ` + "`" + `json:"{{toCamelCase $f.Name}}{{if jsonOmitEmpty $f}},omitempty{{end}}"` + "`" + `
	{{- end}}
}

func (t *{{$t.Name}}) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}
	{{- if needsMutex $t}}
	{{- /* Root types: use getters for maps (returns copies), RLock for scalars */}}
	{{- range $i, $f := $t.Fields}}
	{{- $pt := parseType $f.Type}}
	{{- if or $pt.IsMap $pt.IsArray}}
	{{lower $f.Name}} := t.{{$f.Name}}()
	{{- end}}
	{{- end}}
	t.mu.RLock()
	defer t.mu.RUnlock()
	{{- end}}
	return json.Marshal({{lowerFirst $t.Name}}JSON{
		{{- range $i, $f := $t.Fields}}
		{{- $pt := parseType $f.Type}}
		{{- if or $pt.IsMap $pt.IsArray}}
		{{- if mapValueIsStruct $f}}
		{{- if needsMutex $t}}
		{{$f.Name}}: mapValuesToPtr({{lower $f.Name}}),
		{{- else}}
		{{$f.Name}}: mapValuesToPtr(t.{{lower $f.Name}}),
		{{- end}}
		{{- else}}
		{{- if needsMutex $t}}
		{{$f.Name}}: {{lower $f.Name}},
		{{- else}}
		{{$f.Name}}: t.{{lower $f.Name}},
		{{- end}}
		{{- end}}
		{{- else if or (eq $f.Type "bytes") (eq $f.Type "[]byte")}}
		{{$f.Name}}: bytesToRawJSON(t.{{lower $f.Name}}),
		{{- else}}
		{{$f.Name}}: t.{{lower $f.Name}},
		{{- end}}
		{{- end}}
	})
}

func (t *{{$t.Name}}) UnmarshalJSON(data []byte) error {
	var j {{lowerFirst $t.Name}}JSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	init := New{{$t.Name}}()
	*t = *init
	{{- range $i, $f := $t.Fields}}
	{{- $pt := parseType $f.Type}}
	{{- if or $pt.IsMap $pt.IsArray}}
	{{- if mapValueIsStruct $f}}
	if j.{{$f.Name}} != nil {
		t.Set{{$f.Name}}(mapValuesFromPtr(j.{{$f.Name}}))
	}
	{{- else}}
	if j.{{$f.Name}} != nil {
		t.Set{{$f.Name}}(j.{{$f.Name}})
	}
	{{- end}}
	{{- else if or (eq $f.Type "bytes") (eq $f.Type "[]byte")}}
	if j.{{$f.Name}} != nil {
		t.Set{{$f.Name}}([]byte(j.{{$f.Name}}))
	}
	{{- else}}
	t.Set{{$f.Name}}(j.{{$f.Name}})
	{{- end}}
	{{- end}}
	return nil
}

{{end}}

// ==================================================
// Schema Registry - manages root schemas activation
// ==================================================

// SchemaRegistry manages activation state of root schemas
type SchemaRegistry struct {
	mu       sync.RWMutex
	active   map[string]bool
	schemas  map[string]func() interface{} // Factory functions
	instances map[string]interface{}       // Active instances
}

var (
	schemaRegistryInstance *SchemaRegistry
	schemaRegistryOnce     sync.Once
)

// GetSchemaRegistry returns the singleton schema registry
func GetSchemaRegistry() *SchemaRegistry {
	schemaRegistryOnce.Do(func() {
		schemaRegistryInstance = &SchemaRegistry{
			active:    make(map[string]bool),
			schemas:   make(map[string]func() interface{}),
			instances: make(map[string]interface{}),
		}
		// Register all root schemas
		{{- range $t := .Types}}
		{{- if isRoot $t}}
		schemaRegistryInstance.schemas["{{$t.Name}}"] = func() interface{} { return New{{$t.Name}}() }
		schemaRegistryInstance.active["{{$t.Name}}"] = {{isActiveByDefault $t}}
		{{- if isActiveByDefault $t}}
		schemaRegistryInstance.instances["{{$t.Name}}"] = New{{$t.Name}}()
		{{- end}}
		{{- end}}
		{{- end}}
	})
	return schemaRegistryInstance
}

// IsActive returns true if the schema is currently active
func (r *SchemaRegistry) IsActive(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active[name]
}

// Activate activates a schema (resets to defaults)
func (r *SchemaRegistry) Activate(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	factory, ok := r.schemas[name]
	if !ok {
		return fmt.Errorf("unknown schema: %s", name)
	}

	// Create new instance with defaults
	r.instances[name] = factory()
	r.active[name] = true
	return nil
}

// Deactivate deactivates a schema
func (r *SchemaRegistry) Deactivate(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.schemas[name]; !ok {
		return fmt.Errorf("unknown schema: %s", name)
	}

	delete(r.instances, name)
	r.active[name] = false
	return nil
}

// Get returns the active instance of a schema (nil if not active)
func (r *SchemaRegistry) Get(name string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.instances[name]
}

// GetActive returns all active schema names
func (r *SchemaRegistry) GetActive() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []string
	for name, active := range r.active {
		if active {
			result = append(result, name)
		}
	}
	return result
}

// ResetAll resets all schemas to their default activation state
func (r *SchemaRegistry) ResetAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.instances = make(map[string]interface{})
	{{- range $t := .Types}}
	{{- if isRoot $t}}
	r.active["{{$t.Name}}"] = {{isActiveByDefault $t}}
	{{- if isActiveByDefault $t}}
	r.instances["{{$t.Name}}"] = r.schemas["{{$t.Name}}"]()
	{{- end}}
	{{- end}}
	{{- end}}
}

// Typed getters for convenience
{{range $t := .Types}}
{{- if isRoot $t}}
// Get{{$t.Name}} returns the active {{$t.Name}} instance (nil if not active)
func Get{{$t.Name}}Instance() *{{$t.Name}} {
	inst := GetSchemaRegistry().Get("{{$t.Name}}")
	if inst == nil {
		return nil
	}
	return inst.(*{{$t.Name}})
}

// Activate{{$t.Name}} activates the {{$t.Name}} schema with defaults
func Activate{{$t.Name}}() error {
	return GetSchemaRegistry().Activate("{{$t.Name}}")
}

// Deactivate{{$t.Name}} deactivates the {{$t.Name}} schema
func Deactivate{{$t.Name}}() error {
	return GetSchemaRegistry().Deactivate("{{$t.Name}}")
}

// Is{{$t.Name}}Active returns true if {{$t.Name}} is active
func Is{{$t.Name}}Active() bool {
	return GetSchemaRegistry().IsActive("{{$t.Name}}")
}
{{end}}
{{end}}
`
