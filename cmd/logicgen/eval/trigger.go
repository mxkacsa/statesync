package eval

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// TriggerEvaluator evaluates trigger conditions
type TriggerEvaluator struct {
	// tickCounters tracks tick counts for interval-based triggers
	tickCounters map[string]uint64
	// timerStarts tracks timer start times
	timerStarts map[string]uint64
	// waitFired tracks which one-shot wait triggers have fired
	waitFired map[string]bool
	// previousState tracks previous state values for OnChange detection
	previousState map[string]interface{}
	// lastScheduleRun tracks last run time for schedule triggers
	lastScheduleRun map[string]time.Time
}

// NewTriggerEvaluator creates a new trigger evaluator
func NewTriggerEvaluator() *TriggerEvaluator {
	return &TriggerEvaluator{
		tickCounters:    make(map[string]uint64),
		timerStarts:     make(map[string]uint64),
		waitFired:       make(map[string]bool),
		previousState:   make(map[string]interface{}),
		lastScheduleRun: make(map[string]time.Time),
	}
}

// ResetTimer resets a timer for a specific rule
func (te *TriggerEvaluator) ResetTimer(ruleName string) {
	key := "timer:" + ruleName
	delete(te.timerStarts, key)
	// Also reset wait triggers
	waitKey := "wait:" + ruleName
	delete(te.waitFired, waitKey)
}

// Evaluate evaluates a trigger and returns true if it should fire
func (te *TriggerEvaluator) Evaluate(ctx *Context, trigger *ast.Trigger) (bool, error) {
	if trigger == nil {
		return true, nil // No trigger means always fire
	}

	// Check if trigger is enabled
	if !trigger.IsEnabled() {
		return false, nil
	}

	switch trigger.Type {
	case ast.TriggerTypeOnTick:
		return te.evaluateOnTick(ctx, trigger)
	case ast.TriggerTypeOnEvent:
		return te.evaluateOnEvent(ctx, trigger)
	case ast.TriggerTypeOnChange:
		return te.evaluateOnChange(ctx, trigger)
	case ast.TriggerTypeDistance:
		return te.evaluateDistance(ctx, trigger)
	case ast.TriggerTypeTimer:
		return te.evaluateTimer(ctx, trigger)
	case ast.TriggerTypeCondition:
		return te.evaluateCondition(ctx, trigger)
	case ast.TriggerTypeCron:
		return te.evaluateCron(ctx, trigger)
	case ast.TriggerTypeWait:
		return te.evaluateWait(ctx, trigger)
	case ast.TriggerTypeSchedule:
		return te.evaluateSchedule(ctx, trigger)
	default:
		return false, fmt.Errorf("unknown trigger type: %s", trigger.Type)
	}
}

// evaluateOnTick evaluates an OnTick trigger
func (te *TriggerEvaluator) evaluateOnTick(ctx *Context, trigger *ast.Trigger) (bool, error) {
	if trigger.Interval <= 0 {
		// Every tick
		return true, nil
	}

	// Check interval
	intervalTicks := uint64(trigger.Interval) / uint64(ctx.DeltaTime.Milliseconds())
	if intervalTicks < 1 {
		intervalTicks = 1
	}

	return ctx.Tick%intervalTicks == 0, nil
}

// evaluateOnEvent evaluates an OnEvent trigger
func (te *TriggerEvaluator) evaluateOnEvent(ctx *Context, trigger *ast.Trigger) (bool, error) {
	if ctx.Event == nil {
		return false, nil
	}
	return ctx.Event.Name == trigger.Event, nil
}

// evaluateOnChange evaluates an OnChange trigger
func (te *TriggerEvaluator) evaluateOnChange(ctx *Context, trigger *ast.Trigger) (bool, error) {
	if len(trigger.Watch) == 0 {
		return false, nil
	}

	changed := false
	for _, path := range trigger.Watch {
		key := string(path)
		current, err := ctx.ResolvePath(path)
		if err != nil {
			// Path doesn't exist yet, treat as unchanged
			continue
		}

		previous, exists := te.previousState[key]
		if !exists {
			// First time seeing this path, store it
			te.previousState[key] = deepCopy(current)
			continue
		}

		// Compare values
		if !reflect.DeepEqual(current, previous) {
			changed = true
			te.previousState[key] = deepCopy(current)
		}
	}

	return changed, nil
}

// deepCopy creates a deep copy of a value for change detection
func deepCopy(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Slice:
		if val.IsNil() {
			return nil
		}
		cp := reflect.MakeSlice(val.Type(), val.Len(), val.Len())
		for i := 0; i < val.Len(); i++ {
			cp.Index(i).Set(reflect.ValueOf(deepCopy(val.Index(i).Interface())))
		}
		return cp.Interface()
	case reflect.Map:
		if val.IsNil() {
			return nil
		}
		cp := reflect.MakeMap(val.Type())
		for _, key := range val.MapKeys() {
			cp.SetMapIndex(key, reflect.ValueOf(deepCopy(val.MapIndex(key).Interface())))
		}
		return cp.Interface()
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}
		cp := reflect.New(val.Elem().Type())
		cp.Elem().Set(reflect.ValueOf(deepCopy(val.Elem().Interface())))
		return cp.Interface()
	case reflect.Struct:
		cp := reflect.New(val.Type()).Elem()
		for i := 0; i < val.NumField(); i++ {
			if cp.Field(i).CanSet() {
				cp.Field(i).Set(reflect.ValueOf(deepCopy(val.Field(i).Interface())))
			}
		}
		return cp.Interface()
	default:
		return v
	}
}

// evaluateDistance evaluates a Distance trigger
func (te *TriggerEvaluator) evaluateDistance(ctx *Context, trigger *ast.Trigger) (bool, error) {
	// Resolve from and to positions
	fromVal, err := ctx.ResolvePath(trigger.From)
	if err != nil {
		return false, err
	}
	toVal, err := ctx.ResolvePath(trigger.To)
	if err != nil {
		return false, err
	}

	// Convert to GeoPoints
	fromPoint, err := toGeoPoint(fromVal)
	if err != nil {
		return false, fmt.Errorf("from: %w", err)
	}
	toPoint, err := toGeoPoint(toVal)
	if err != nil {
		return false, fmt.Errorf("to: %w", err)
	}

	// Calculate distance
	distance := haversineDistance(fromPoint, toPoint)

	// Convert units
	threshold := trigger.Value
	if trigger.Unit == "kilometers" || trigger.Unit == "km" {
		threshold *= 1000
	}

	// Compare
	switch trigger.Operator {
	case "<=", "":
		return distance <= threshold, nil
	case "<":
		return distance < threshold, nil
	case ">=":
		return distance >= threshold, nil
	case ">":
		return distance > threshold, nil
	case "==":
		return distance == threshold, nil
	default:
		return false, fmt.Errorf("unknown operator: %s", trigger.Operator)
	}
}

// evaluateTimer evaluates a Timer trigger
func (te *TriggerEvaluator) evaluateTimer(ctx *Context, trigger *ast.Trigger) (bool, error) {
	// Use rule name for stable key instead of pointer
	key := "timer:" + trigger.RuleName
	if trigger.RuleName == "" {
		// Fallback to pointer if rule name not set (shouldn't happen)
		key = fmt.Sprintf("timer:%p", trigger)
	}

	startTick, exists := te.timerStarts[key]
	if !exists {
		// Start the timer
		te.timerStarts[key] = ctx.Tick
		startTick = ctx.Tick

		// Handle start delay
		if trigger.StartDelay > 0 {
			return false, nil
		}
	}

	// Calculate elapsed ticks
	elapsed := ctx.Tick - startTick
	durationTicks := uint64(trigger.Duration) / uint64(ctx.DeltaTime.Milliseconds())
	if durationTicks < 1 {
		durationTicks = 1
	}

	if elapsed >= durationTicks {
		if trigger.Repeat {
			// Reset timer
			te.timerStarts[key] = ctx.Tick
		} else {
			// Remove timer
			delete(te.timerStarts, key)
		}
		return true, nil
	}

	return false, nil
}

// evaluateWait evaluates a Wait trigger (one-shot delay)
func (te *TriggerEvaluator) evaluateWait(ctx *Context, trigger *ast.Trigger) (bool, error) {
	key := "wait:" + trigger.RuleName
	if trigger.RuleName == "" {
		key = fmt.Sprintf("wait:%p", trigger)
	}

	// Check if already fired
	if te.waitFired[key] {
		return false, nil
	}

	// Check timer
	startTick, exists := te.timerStarts[key]
	if !exists {
		te.timerStarts[key] = ctx.Tick
		return false, nil
	}

	elapsed := ctx.Tick - startTick
	durationTicks := uint64(trigger.Duration) / uint64(ctx.DeltaTime.Milliseconds())
	if durationTicks < 1 {
		durationTicks = 1
	}

	if elapsed >= durationTicks {
		te.waitFired[key] = true
		delete(te.timerStarts, key)
		return true, nil
	}

	return false, nil
}

// evaluateCron evaluates a Cron trigger
func (te *TriggerEvaluator) evaluateCron(ctx *Context, trigger *ast.Trigger) (bool, error) {
	if trigger.Cron == "" {
		return false, nil
	}

	// Simple cron parsing - supports: */N * * * * (every N minutes)
	// Full cron: minute hour dayOfMonth month dayOfWeek
	parts := strings.Fields(trigger.Cron)
	if len(parts) < 5 {
		return false, fmt.Errorf("invalid cron expression: %s", trigger.Cron)
	}

	now := time.Now()
	minute := now.Minute()
	hour := now.Hour()
	dayOfMonth := now.Day()
	month := int(now.Month())
	dayOfWeek := int(now.Weekday())

	// Check each field
	if !matchCronField(parts[0], minute) ||
		!matchCronField(parts[1], hour) ||
		!matchCronField(parts[2], dayOfMonth) ||
		!matchCronField(parts[3], month) ||
		!matchCronField(parts[4], dayOfWeek) {
		return false, nil
	}

	// Prevent firing multiple times in the same minute
	key := "cron:" + trigger.RuleName
	if trigger.RuleName == "" {
		key = fmt.Sprintf("cron:%p", trigger)
	}

	lastRun := te.lastScheduleRun[key]
	if now.Sub(lastRun) < time.Minute {
		return false, nil
	}

	te.lastScheduleRun[key] = now
	return true, nil
}

// matchCronField checks if a value matches a cron field expression
func matchCronField(expr string, value int) bool {
	if expr == "*" {
		return true
	}

	// Handle */N (every N)
	if strings.HasPrefix(expr, "*/") {
		n, err := strconv.Atoi(expr[2:])
		if err != nil || n <= 0 {
			return false
		}
		return value%n == 0
	}

	// Handle comma-separated values
	if strings.Contains(expr, ",") {
		for _, part := range strings.Split(expr, ",") {
			if matchCronField(part, value) {
				return true
			}
		}
		return false
	}

	// Handle range N-M
	if strings.Contains(expr, "-") {
		parts := strings.Split(expr, "-")
		if len(parts) == 2 {
			start, err1 := strconv.Atoi(parts[0])
			end, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil {
				return value >= start && value <= end
			}
		}
		return false
	}

	// Simple numeric match
	n, err := strconv.Atoi(expr)
	if err != nil {
		return false
	}
	return value == n
}

// evaluateSchedule evaluates a Schedule trigger
func (te *TriggerEvaluator) evaluateSchedule(ctx *Context, trigger *ast.Trigger) (bool, error) {
	key := "schedule:" + trigger.RuleName
	if trigger.RuleName == "" {
		key = fmt.Sprintf("schedule:%p", trigger)
	}

	now := time.Now()

	// Check weekday constraint
	if len(trigger.Weekdays) > 0 {
		currentWeekday := int(now.Weekday())
		found := false
		for _, wd := range trigger.Weekdays {
			if wd == currentWeekday {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}

	// Handle "at" time (e.g., "14:30")
	if trigger.At != "" {
		parts := strings.Split(trigger.At, ":")
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid time format: %s (expected HH:MM)", trigger.At)
		}
		hour, err1 := strconv.Atoi(parts[0])
		minute, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return false, fmt.Errorf("invalid time format: %s", trigger.At)
		}

		if now.Hour() != hour || now.Minute() != minute {
			return false, nil
		}

		// Prevent firing multiple times in the same minute
		lastRun := te.lastScheduleRun[key]
		if now.Sub(lastRun) < time.Minute {
			return false, nil
		}
		te.lastScheduleRun[key] = now
		return true, nil
	}

	// Handle "every" interval (e.g., "5m", "1h", "30s")
	if trigger.Every != "" {
		duration, err := parseDuration(trigger.Every)
		if err != nil {
			return false, err
		}

		lastRun := te.lastScheduleRun[key]
		if lastRun.IsZero() {
			te.lastScheduleRun[key] = now
			return true, nil // Fire immediately on first run
		}

		if now.Sub(lastRun) >= duration {
			te.lastScheduleRun[key] = now
			return true, nil
		}
	}

	return false, nil
}

// parseDuration parses duration strings like "5m", "1h", "30s", "24h"
func parseDuration(s string) (time.Duration, error) {
	// time.ParseDuration already handles these formats
	return time.ParseDuration(s)
}

// evaluateCondition evaluates a Condition trigger
func (te *TriggerEvaluator) evaluateCondition(ctx *Context, trigger *ast.Trigger) (bool, error) {
	if trigger.Condition == nil {
		return true, nil
	}
	return evaluateExpression(ctx, trigger.Condition)
}

// evaluateExpression evaluates a boolean expression
func evaluateExpression(ctx *Context, expr *ast.Expression) (bool, error) {
	if expr == nil {
		return true, nil
	}

	// Handle logical operators
	if len(expr.And) > 0 {
		for _, sub := range expr.And {
			result, err := evaluateExpression(ctx, &sub)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	if len(expr.Or) > 0 {
		for _, sub := range expr.Or {
			result, err := evaluateExpression(ctx, &sub)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	if expr.Not != nil {
		result, err := evaluateExpression(ctx, expr.Not)
		if err != nil {
			return false, err
		}
		return !result, nil
	}

	// Handle comparison
	if expr.Left == nil {
		return true, nil
	}

	left, err := ctx.Resolve(expr.Left)
	if err != nil {
		return false, err
	}
	right, err := ctx.Resolve(expr.Right)
	if err != nil {
		return false, err
	}

	return compare(left, right, expr.Op)
}

// compare compares two values with an operator
func compare(left, right interface{}, op string) (bool, error) {
	// Handle nil
	if left == nil || right == nil {
		switch op {
		case "==":
			return left == right, nil
		case "!=":
			return left != right, nil
		default:
			return false, nil
		}
	}

	// Try numeric comparison
	leftNum, leftOk := toFloat64(left)
	rightNum, rightOk := toFloat64(right)
	if leftOk && rightOk {
		switch op {
		case "==":
			return leftNum == rightNum, nil
		case "!=":
			return leftNum != rightNum, nil
		case ">":
			return leftNum > rightNum, nil
		case ">=":
			return leftNum >= rightNum, nil
		case "<":
			return leftNum < rightNum, nil
		case "<=":
			return leftNum <= rightNum, nil
		}
	}

	// String comparison
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	switch op {
	case "==":
		return leftStr == rightStr, nil
	case "!=":
		return leftStr != rightStr, nil
	case ">":
		return leftStr > rightStr, nil
	case ">=":
		return leftStr >= rightStr, nil
	case "<":
		return leftStr < rightStr, nil
	case "<=":
		return leftStr <= rightStr, nil
	case "contains":
		return contains(left, right), nil
	case "in":
		return contains(right, left), nil
	default:
		return false, fmt.Errorf("unknown operator: %s", op)
	}
}

// toFloat64 converts a value to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}

// contains checks if container contains item
func contains(container, item interface{}) bool {
	// Handle string contains
	if str, ok := container.(string); ok {
		if itemStr, ok := item.(string); ok {
			return len(str) > 0 && len(itemStr) > 0 && (str == itemStr ||
				(len(str) > len(itemStr) && (str[:len(itemStr)] == itemStr || str[len(str)-len(itemStr):] == itemStr)))
		}
	}
	// TODO: Handle array/slice contains
	return false
}

// toGeoPoint converts a value to GeoPoint
func toGeoPoint(v interface{}) (ast.GeoPoint, error) {
	switch val := v.(type) {
	case ast.GeoPoint:
		return val, nil
	case *ast.GeoPoint:
		if val == nil {
			return ast.GeoPoint{}, fmt.Errorf("nil GeoPoint")
		}
		return *val, nil
	case map[string]interface{}:
		lat, latOk := val["lat"].(float64)
		lon, lonOk := val["lon"].(float64)
		if !latOk || !lonOk {
			// Try Lat/Lon
			lat, latOk = val["Lat"].(float64)
			lon, lonOk = val["Lon"].(float64)
		}
		if latOk && lonOk {
			return ast.GeoPoint{Lat: lat, Lon: lon}, nil
		}
		return ast.GeoPoint{}, fmt.Errorf("invalid GeoPoint map")
	default:
		return ast.GeoPoint{}, fmt.Errorf("cannot convert %T to GeoPoint", v)
	}
}
