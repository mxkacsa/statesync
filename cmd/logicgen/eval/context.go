// Package eval provides runtime evaluation of LogicGen v2 rules.
package eval

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// DebugHandler handles debug logging for rule evaluation
type DebugHandler interface {
	// OnMissingField is called when a field access fails during filtering
	OnMissingField(entityType string, fieldName string, err error)
	// OnFilterMatch logs filter match/no-match for debugging
	OnFilterMatch(entityType string, field string, op string, expected, actual interface{}, matched bool)
}

// DefaultDebugHandler logs to stdout
type DefaultDebugHandler struct{}

func (d *DefaultDebugHandler) OnMissingField(entityType, fieldName string, err error) {
	fmt.Printf("[DEBUG] Missing field '%s' on %s: %v\n", fieldName, entityType, err)
}

func (d *DefaultDebugHandler) OnFilterMatch(entityType, field, op string, expected, actual interface{}, matched bool) {
	fmt.Printf("[DEBUG] Filter %s.%s %s %v (actual: %v) = %v\n", entityType, field, op, expected, actual, matched)
}

// Context provides runtime context for rule evaluation
type Context struct {
	// State is the current game state (must be a pointer to struct)
	State interface{}

	// DeltaTime is the time since last tick
	DeltaTime time.Duration

	// Tick is the current tick number
	Tick uint64

	// Event is the triggering event (nil for OnTick triggers)
	Event *ast.Event

	// Params contains event parameters or other runtime params
	Params map[string]interface{}

	// Views contains computed view results for this rule execution
	Views map[string]interface{}

	// CurrentEntity is the entity being processed in selector context
	CurrentEntity interface{}

	// CurrentIndex is the index of the current entity
	CurrentIndex int

	// SelectedEntities contains entities selected by the selector
	SelectedEntities []interface{}

	// SenderID is the player ID who triggered the event (empty = server/rule)
	SenderID string

	// PermissionChecker validates write permissions (nil = no restrictions)
	PermissionChecker *PermissionChecker

	// Debug enables debug logging (for development)
	Debug bool

	// DebugHandler handles debug output (nil uses DefaultDebugHandler if Debug is true)
	DebugHandler DebugHandler

	// stateValue is cached reflection value of state
	stateValue reflect.Value
}

// NewContext creates a new evaluation context
func NewContext(state interface{}, dt time.Duration, tick uint64) *Context {
	return &Context{
		State:     state,
		DeltaTime: dt,
		Tick:      tick,
		Params:    make(map[string]interface{}),
		Views:     make(map[string]interface{}),
	}
}

// WithDebug returns a new context with debug logging enabled
func (c *Context) WithDebug(handler DebugHandler) *Context {
	newCtx := *c
	newCtx.Debug = true
	newCtx.DebugHandler = handler
	if handler == nil {
		newCtx.DebugHandler = &DefaultDebugHandler{}
	}
	return &newCtx
}

// getDebugHandler returns the debug handler if debug is enabled
func (c *Context) getDebugHandler() DebugHandler {
	if !c.Debug {
		return nil
	}
	if c.DebugHandler == nil {
		return &DefaultDebugHandler{}
	}
	return c.DebugHandler
}

// WithPermissions returns a new context with permission checking enabled
func (c *Context) WithPermissions(schema *PermissionSchema) *Context {
	newCtx := *c
	newCtx.PermissionChecker = NewPermissionChecker(schema)
	if c.SenderID != "" {
		newCtx.PermissionChecker = newCtx.PermissionChecker.WithSender(c.SenderID)
	}
	return &newCtx
}

// WithSender returns a new context with the given sender ID
func (c *Context) WithSender(senderID string) *Context {
	newCtx := *c
	newCtx.SenderID = senderID
	if c.PermissionChecker != nil {
		newCtx.PermissionChecker = c.PermissionChecker.WithSender(senderID)
	}
	return &newCtx
}

// WithEvent returns a new context with the given event
func (c *Context) WithEvent(event *ast.Event) *Context {
	newCtx := *c
	newCtx.Event = event
	if event != nil {
		// Set sender from event
		newCtx.SenderID = event.Sender

		// Update permission checker with sender
		if c.PermissionChecker != nil {
			newCtx.PermissionChecker = c.PermissionChecker.WithSender(event.Sender)
		}

		// Merge event params
		if event.Params != nil {
			newCtx.Params = make(map[string]interface{})
			for k, v := range c.Params {
				newCtx.Params[k] = v
			}
			for k, v := range event.Params {
				newCtx.Params[k] = v
			}
		}
	}
	return &newCtx
}

// WithEntity returns a new context with the given current entity
func (c *Context) WithEntity(entity interface{}, index int) *Context {
	newCtx := *c
	newCtx.CurrentEntity = entity
	newCtx.CurrentIndex = index
	return &newCtx
}

// GetStateValue returns the reflect.Value of the state
func (c *Context) GetStateValue() reflect.Value {
	if !c.stateValue.IsValid() {
		c.stateValue = reflect.ValueOf(c.State)
		if c.stateValue.Kind() == reflect.Ptr {
			c.stateValue = c.stateValue.Elem()
		}
	}
	return c.stateValue
}

// Resolve resolves a path or value to its actual value
func (c *Context) Resolve(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}

	switch val := v.(type) {
	case string:
		return c.resolveString(val)
	case ast.Path:
		return c.ResolvePath(val)
	case float64, float32, int, int32, int64, bool:
		return val, nil
	case map[string]interface{}:
		// Could be a transform or nested object
		if typeStr, ok := val["type"].(string); ok && typeStr != "" {
			// This is a transform, evaluate it
			return c.evaluateTransformMap(val)
		}
		// Regular object, resolve each value
		result := make(map[string]interface{})
		for k, v := range val {
			resolved, err := c.Resolve(v)
			if err != nil {
				return nil, err
			}
			result[k] = resolved
		}
		return result, nil
	default:
		return val, nil
	}
}

// resolveString resolves a string value which may be a path or literal
func (c *Context) resolveString(s string) (interface{}, error) {
	if len(s) == 0 {
		return s, nil
	}

	// Self reference: self.field (resolved to current entity's field)
	if strings.HasPrefix(s, "self.") {
		if c.CurrentEntity == nil {
			return nil, fmt.Errorf("self reference outside entity context: %s", s)
		}
		fieldPath := s[5:] // Remove "self." prefix
		return getField(c.CurrentEntity, fieldPath)
	}
	if s == "self" {
		if c.CurrentEntity == nil {
			return nil, fmt.Errorf("self reference outside entity context")
		}
		return c.CurrentEntity, nil
	}

	// State path: $.Field or state:$.Field
	if s[0] == '$' {
		return c.ResolvePath(ast.Path(s))
	}
	if strings.HasPrefix(s, "state:") {
		return c.ResolvePath(ast.Path(s[6:]))
	}

	// Param reference: param:name
	if strings.HasPrefix(s, "param:") {
		paramName := s[6:]
		if val, ok := c.Params[paramName]; ok {
			return val, nil
		}
		return nil, fmt.Errorf("param not found: %s", paramName)
	}

	// View reference: view:name or view:name.field
	if strings.HasPrefix(s, "view:") {
		viewRef := s[5:]
		parts := strings.SplitN(viewRef, ".", 2)
		viewName := parts[0]
		if val, ok := c.Views[viewName]; ok {
			if len(parts) == 1 {
				return val, nil
			}
			// Access field on view result
			return getField(val, parts[1])
		}
		return nil, fmt.Errorf("view not found: %s", viewName)
	}

	// Const reference: const:value
	if strings.HasPrefix(s, "const:") {
		return parseConstValue(s[6:])
	}

	// Just a string literal
	return s, nil
}

// ResolvePath resolves a path expression to its value
func (c *Context) ResolvePath(path ast.Path) (interface{}, error) {
	pathStr := string(path)
	if pathStr == "" {
		return nil, fmt.Errorf("empty path")
	}

	// Handle $ (current entity)
	if pathStr == "$" {
		if c.CurrentEntity != nil {
			return c.CurrentEntity, nil
		}
		return c.State, nil
	}

	// Must start with $
	if pathStr[0] != '$' {
		return nil, fmt.Errorf("path must start with $: %s", pathStr)
	}

	// Parse and navigate path
	segments, err := parsePath(pathStr[1:]) // Skip $
	if err != nil {
		return nil, err
	}

	// Determine starting value
	var current interface{}
	if c.CurrentEntity != nil {
		current = c.CurrentEntity
	} else {
		current = c.State
	}

	// Navigate through segments
	for _, seg := range segments {
		current, err = c.navigateSegment(current, seg)
		if err != nil {
			return nil, fmt.Errorf("path %s: %w", pathStr, err)
		}
	}

	return current, nil
}

// SetPath sets a value at the given path
func (c *Context) SetPath(path ast.Path, value interface{}) error {
	pathStr := string(path)
	if pathStr == "" || pathStr[0] != '$' {
		return fmt.Errorf("invalid path: %s", pathStr)
	}

	segments, err := parsePath(pathStr[1:])
	if err != nil {
		return err
	}

	if len(segments) == 0 {
		return fmt.Errorf("cannot set root state")
	}

	// Navigate to parent
	var current interface{}
	if c.CurrentEntity != nil {
		current = c.CurrentEntity
	} else {
		current = c.State
	}

	for i := 0; i < len(segments)-1; i++ {
		current, err = c.navigateSegment(current, segments[i])
		if err != nil {
			return fmt.Errorf("path %s: %w", pathStr, err)
		}
	}

	// Check permissions before setting
	lastSeg := segments[len(segments)-1]
	if c.PermissionChecker != nil {
		fieldName := lastSeg.field
		if fieldName == "" && lastSeg.index != nil {
			// For array/map access, we need the parent entity
			// Permission is checked on the entity being modified
		}
		if fieldName != "" {
			if err := c.PermissionChecker.CanWrite(current, fieldName); err != nil {
				return err
			}
		}
	}

	// Set the final field
	return c.setField(current, lastSeg, value)
}

// pathSegment represents a parsed path segment
type pathSegment struct {
	field      string
	index      interface{} // int for array, string for map, "*" for wildcard
	isWildcard bool
}

// parsePath parses a path string into segments
func parsePath(path string) ([]pathSegment, error) {
	if path == "" {
		return nil, nil
	}

	// Skip leading dot
	if path[0] == '.' {
		path = path[1:]
	}

	var segments []pathSegment
	var current strings.Builder
	var inBracket bool
	var bracketContent strings.Builder

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			if inBracket {
				bracketContent.WriteByte(ch)
			} else if current.Len() > 0 {
				segments = append(segments, pathSegment{field: current.String()})
				current.Reset()
			}
		case '[':
			if current.Len() > 0 {
				segments = append(segments, pathSegment{field: current.String()})
				current.Reset()
			}
			inBracket = true
			bracketContent.Reset()
		case ']':
			if inBracket {
				inBracket = false
				content := bracketContent.String()
				// Parse bracket content
				if content == "*" {
					if len(segments) > 0 {
						segments[len(segments)-1].isWildcard = true
					}
				} else if idx, err := strconv.Atoi(content); err == nil {
					if len(segments) > 0 {
						segments[len(segments)-1].index = idx
					}
				} else {
					// String key (strip quotes if present)
					key := strings.Trim(content, `"'`)
					if len(segments) > 0 {
						segments[len(segments)-1].index = key
					}
				}
			}
		default:
			if inBracket {
				bracketContent.WriteByte(ch)
			} else {
				current.WriteByte(ch)
			}
		}
	}

	// Handle remaining content
	if current.Len() > 0 {
		segments = append(segments, pathSegment{field: current.String()})
	}

	return segments, nil
}

// navigateSegment navigates one segment of a path
func (c *Context) navigateSegment(current interface{}, seg pathSegment) (interface{}, error) {
	if current == nil {
		return nil, fmt.Errorf("cannot navigate on nil")
	}

	val := reflect.ValueOf(current)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Get field
	if seg.field != "" {
		switch val.Kind() {
		case reflect.Struct:
			field := val.FieldByName(seg.field)
			if !field.IsValid() {
				// Try method call (for getter methods)
				method := reflect.ValueOf(current).MethodByName(seg.field)
				if method.IsValid() && method.Type().NumIn() == 0 {
					results := method.Call(nil)
					if len(results) > 0 {
						val = results[0]
					} else {
						return nil, fmt.Errorf("field not found: %s", seg.field)
					}
				} else {
					return nil, fmt.Errorf("field not found: %s", seg.field)
				}
			} else {
				val = field
			}
		case reflect.Map:
			key := reflect.ValueOf(seg.field)
			val = val.MapIndex(key)
			if !val.IsValid() {
				return nil, nil
			}
		default:
			return nil, fmt.Errorf("cannot access field on %s", val.Kind())
		}
	}

	// Handle index
	if seg.index != nil {
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
			idx, ok := seg.index.(int)
			if !ok {
				return nil, fmt.Errorf("invalid array index: %v", seg.index)
			}
			if idx < 0 || idx >= val.Len() {
				return nil, fmt.Errorf("index out of bounds: %d", idx)
			}
			val = val.Index(idx)
		case reflect.Map:
			key := reflect.ValueOf(seg.index)
			val = val.MapIndex(key)
			if !val.IsValid() {
				return nil, nil
			}
		default:
			return nil, fmt.Errorf("cannot index %s", val.Kind())
		}
	}

	if val.IsValid() && val.CanInterface() {
		return val.Interface(), nil
	}
	return nil, nil
}

// setField sets a field value on an object
func (c *Context) setField(obj interface{}, seg pathSegment, value interface{}) error {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Convert value to appropriate type
	valueVal := reflect.ValueOf(value)

	if seg.field != "" {
		switch val.Kind() {
		case reflect.Struct:
			field := val.FieldByName(seg.field)
			if !field.IsValid() {
				return fmt.Errorf("field not found: %s", seg.field)
			}
			if !field.CanSet() {
				// Try setter method
				method := reflect.ValueOf(obj).MethodByName("Set" + seg.field)
				if method.IsValid() {
					method.Call([]reflect.Value{valueVal})
					return nil
				}
				return fmt.Errorf("cannot set field: %s", seg.field)
			}
			if valueVal.Type().ConvertibleTo(field.Type()) {
				field.Set(valueVal.Convert(field.Type()))
			} else {
				field.Set(valueVal)
			}
			return nil
		case reflect.Map:
			val.SetMapIndex(reflect.ValueOf(seg.field), valueVal)
			return nil
		}
	}

	if seg.index != nil {
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
			idx, ok := seg.index.(int)
			if !ok {
				return fmt.Errorf("invalid array index: %v", seg.index)
			}
			if idx < 0 || idx >= val.Len() {
				return fmt.Errorf("index out of bounds: %d", idx)
			}
			val.Index(idx).Set(valueVal)
			return nil
		case reflect.Map:
			val.SetMapIndex(reflect.ValueOf(seg.index), valueVal)
			return nil
		}
	}

	return fmt.Errorf("cannot set value")
}

// getField gets a field from an object using reflection
func getField(obj interface{}, field string) (interface{}, error) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		f := val.FieldByName(field)
		if !f.IsValid() {
			return nil, fmt.Errorf("field not found: %s", field)
		}
		return f.Interface(), nil
	case reflect.Map:
		v := val.MapIndex(reflect.ValueOf(field))
		if !v.IsValid() {
			return nil, nil
		}
		return v.Interface(), nil
	default:
		return nil, fmt.Errorf("cannot get field from %s", val.Kind())
	}
}

// parseConstValue parses a constant value string
func parseConstValue(s string) (interface{}, error) {
	// Try as number
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	// Try as bool
	if s == "true" {
		return true, nil
	}
	if s == "false" {
		return false, nil
	}
	// Return as string
	return s, nil
}

// evaluateTransformMap evaluates a transform defined as a map
func (c *Context) evaluateTransformMap(m map[string]interface{}) (interface{}, error) {
	// Convert map to Transform struct
	transform := &ast.Transform{}

	if typeStr, ok := m["type"].(string); ok {
		transform.Type = ast.TransformType(typeStr)
	}

	// Copy relevant fields
	if v, ok := m["left"]; ok {
		transform.Left = v
	}
	if v, ok := m["right"]; ok {
		transform.Right = v
	}
	if v, ok := m["value"]; ok {
		transform.Value = v
	}
	if v, ok := m["min"]; ok {
		transform.Min = v
	}
	if v, ok := m["max"]; ok {
		transform.Max = v
	}
	if v, ok := m["current"]; ok {
		transform.Current = v
	}
	if v, ok := m["target"]; ok {
		transform.Target = v
	}
	if v, ok := m["speed"].(float64); ok {
		transform.Speed = v
	}
	if v, ok := m["unit"].(string); ok {
		transform.Unit = v
	}
	if v, ok := m["from"]; ok {
		transform.From = v
	}
	if v, ok := m["to"]; ok {
		transform.To = v
	}
	if v, ok := m["condition"]; ok {
		transform.Condition = v
	}
	if v, ok := m["then"]; ok {
		transform.Then = v
	}
	if v, ok := m["else"]; ok {
		transform.Else = v
	}
	if v, ok := m["format"].(string); ok {
		transform.Format = v
	}
	if v, ok := m["strings"].([]interface{}); ok {
		transform.Strings = v
	}
	if v, ok := m["args"].([]interface{}); ok {
		transform.Args = v
	}

	// Evaluate the transform
	eval := NewTransformEvaluator()
	return eval.Evaluate(c, transform)
}
