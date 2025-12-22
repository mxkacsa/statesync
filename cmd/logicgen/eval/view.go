package eval

import (
	"fmt"
	"math"
	"reflect"
	"sort"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// ViewEvaluator computes view values
type ViewEvaluator struct{}

// NewViewEvaluator creates a new view evaluator
func NewViewEvaluator() *ViewEvaluator {
	return &ViewEvaluator{}
}

// Compute computes a view value from entities
func (ve *ViewEvaluator) Compute(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	switch view.Type {
	case ast.ViewTypeField:
		return ve.computeField(ctx, view, entities)
	case ast.ViewTypeMax:
		return ve.computeMax(ctx, view, entities)
	case ast.ViewTypeMin:
		return ve.computeMin(ctx, view, entities)
	case ast.ViewTypeSum:
		return ve.computeSum(ctx, view, entities)
	case ast.ViewTypeCount:
		return ve.computeCount(ctx, view, entities)
	case ast.ViewTypeAvg:
		return ve.computeAvg(ctx, view, entities)
	case ast.ViewTypeFirst:
		return ve.computeFirst(ctx, view, entities)
	case ast.ViewTypeLast:
		return ve.computeLast(ctx, view, entities)
	case ast.ViewTypeGroupBy:
		return ve.computeGroupBy(ctx, view, entities)
	case ast.ViewTypeDistinct:
		return ve.computeDistinct(ctx, view, entities)
	case ast.ViewTypeMap:
		return ve.computeMap(ctx, view, entities)
	case ast.ViewTypeSort:
		return ve.computeSort(ctx, view, entities)
	case ast.ViewTypeDistance:
		return ve.computeDistance(ctx, view, entities)
	default:
		return nil, fmt.Errorf("unknown view type: %s", view.Type)
	}
}

// computeField extracts a field value from entities
func (ve *ViewEvaluator) computeField(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	if len(entities) == 0 {
		return nil, nil
	}

	// If single entity, return its field
	if len(entities) == 1 {
		entityCtx := ctx.WithEntity(entities[0], 0)
		return entityCtx.ResolvePath(view.Field)
	}

	// Multiple entities - return array of field values
	result := make([]interface{}, len(entities))
	for i, entity := range entities {
		entityCtx := ctx.WithEntity(entity, i)
		val, err := entityCtx.ResolvePath(view.Field)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return result, nil
}

// computeMax finds the maximum value
func (ve *ViewEvaluator) computeMax(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	if len(entities) == 0 {
		return nil, nil
	}

	var maxVal float64 = math.Inf(-1)
	var maxEntity interface{}

	for i, entity := range entities {
		entityCtx := ctx.WithEntity(entity, i)
		val, err := entityCtx.ResolvePath(view.Field)
		if err != nil {
			continue
		}

		numVal, ok := toFloat64(val)
		if !ok {
			continue
		}

		if numVal > maxVal {
			maxVal = numVal
			maxEntity = entity
		}
	}

	if view.Return == "entity" {
		return maxEntity, nil
	}
	if math.IsInf(maxVal, -1) {
		return nil, nil
	}
	return maxVal, nil
}

// computeMin finds the minimum value
func (ve *ViewEvaluator) computeMin(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	if len(entities) == 0 {
		return nil, nil
	}

	var minVal float64 = math.Inf(1)
	var minEntity interface{}

	for i, entity := range entities {
		entityCtx := ctx.WithEntity(entity, i)
		val, err := entityCtx.ResolvePath(view.Field)
		if err != nil {
			continue
		}

		numVal, ok := toFloat64(val)
		if !ok {
			continue
		}

		if numVal < minVal {
			minVal = numVal
			minEntity = entity
		}
	}

	if view.Return == "entity" {
		return minEntity, nil
	}
	if math.IsInf(minVal, 1) {
		return nil, nil
	}
	return minVal, nil
}

// computeSum calculates the sum of values
func (ve *ViewEvaluator) computeSum(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	var sum float64

	for i, entity := range entities {
		entityCtx := ctx.WithEntity(entity, i)
		val, err := entityCtx.ResolvePath(view.Field)
		if err != nil {
			continue
		}

		numVal, ok := toFloat64(val)
		if ok {
			sum += numVal
		}
	}

	return sum, nil
}

// computeCount counts entities
func (ve *ViewEvaluator) computeCount(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	if view.Where == nil {
		return len(entities), nil
	}

	// Count with condition
	count := 0
	for _, entity := range entities {
		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			continue
		}

		field := val.FieldByName(view.Where.Field)
		if !field.IsValid() || !field.CanInterface() {
			continue
		}

		compareVal, err := ctx.Resolve(view.Where.Value)
		if err != nil {
			continue
		}

		matches, _ := compare(field.Interface(), compareVal, view.Where.Op)
		if matches {
			count++
		}
	}

	return count, nil
}

// computeAvg calculates the average
func (ve *ViewEvaluator) computeAvg(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	if len(entities) == 0 {
		return 0.0, nil
	}

	sum, err := ve.computeSum(ctx, view, entities)
	if err != nil {
		return nil, err
	}

	sumVal, _ := toFloat64(sum)
	return sumVal / float64(len(entities)), nil
}

// computeFirst returns the first entity
func (ve *ViewEvaluator) computeFirst(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	if len(entities) == 0 {
		return nil, nil
	}
	return entities[0], nil
}

// computeLast returns the last entity
func (ve *ViewEvaluator) computeLast(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	if len(entities) == 0 {
		return nil, nil
	}
	return entities[len(entities)-1], nil
}

// computeGroupBy groups entities by a field
func (ve *ViewEvaluator) computeGroupBy(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	groups := make(map[interface{}][]interface{})

	for _, entity := range entities {
		// Get group key
		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			continue
		}

		field := val.FieldByName(view.GroupField)
		if !field.IsValid() || !field.CanInterface() {
			continue
		}

		key := field.Interface()
		groups[key] = append(groups[key], entity)
	}

	// If aggregate is specified, compute it for each group
	if view.Aggregate != nil {
		result := make(map[interface{}]interface{})
		for key, groupEntities := range groups {
			aggVal, err := ve.Compute(ctx, view.Aggregate, groupEntities)
			if err != nil {
				return nil, err
			}
			result[key] = aggVal
		}
		return result, nil
	}

	return groups, nil
}

// computeDistinct returns distinct values of a field
func (ve *ViewEvaluator) computeDistinct(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	seen := make(map[interface{}]bool)
	var result []interface{}

	for i, entity := range entities {
		entityCtx := ctx.WithEntity(entity, i)
		val, err := entityCtx.ResolvePath(view.Field)
		if err != nil {
			continue
		}

		key := fmt.Sprintf("%v", val)
		if !seen[key] {
			seen[key] = true
			result = append(result, val)
		}
	}

	return result, nil
}

// computeMap transforms entities
func (ve *ViewEvaluator) computeMap(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	result := make([]map[string]interface{}, len(entities))

	for i, entity := range entities {
		entityCtx := ctx.WithEntity(entity, i)
		mapped := make(map[string]interface{})

		for key, valExpr := range view.Transform {
			val, err := entityCtx.Resolve(valExpr)
			if err != nil {
				return nil, fmt.Errorf("transform %s: %w", key, err)
			}
			mapped[key] = val
		}

		result[i] = mapped
	}

	return result, nil
}

// computeSort sorts entities
func (ve *ViewEvaluator) computeSort(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	if len(entities) <= 1 {
		return entities, nil
	}

	// Create sortable copy
	sorted := make([]interface{}, len(entities))
	copy(sorted, entities)

	// Extract sort field from path
	sortField := string(view.By)
	if len(sortField) > 2 && sortField[:2] == "$." {
		sortField = sortField[2:]
	}

	// Sort
	sort.Slice(sorted, func(i, j int) bool {
		iVal := getFieldValue(sorted[i], sortField)
		jVal := getFieldValue(sorted[j], sortField)

		iNum, iOk := toFloat64(iVal)
		jNum, jOk := toFloat64(jVal)

		var less bool
		if iOk && jOk {
			less = iNum < jNum
		} else {
			less = fmt.Sprintf("%v", iVal) < fmt.Sprintf("%v", jVal)
		}

		if view.Order == "desc" {
			return !less
		}
		return less
	})

	// Apply limit
	if view.Limit > 0 && view.Limit < len(sorted) {
		sorted = sorted[:view.Limit]
	}

	return sorted, nil
}

// computeDistance calculates distance between two points
func (ve *ViewEvaluator) computeDistance(ctx *Context, view *ast.View, entities []interface{}) (interface{}, error) {
	fromVal, err := ctx.Resolve(view.From)
	if err != nil {
		return nil, err
	}
	toVal, err := ctx.Resolve(view.To)
	if err != nil {
		return nil, err
	}

	fromPoint, err := toGeoPoint(fromVal)
	if err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}
	toPoint, err := toGeoPoint(toVal)
	if err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}

	distance := haversineDistance(fromPoint, toPoint)

	// Convert units
	if view.Unit == "kilometers" || view.Unit == "km" {
		distance /= 1000
	}

	return distance, nil
}

// getFieldValue gets a field value from an entity
func getFieldValue(entity interface{}, field string) interface{} {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		f := val.FieldByName(field)
		if f.IsValid() && f.CanInterface() {
			return f.Interface()
		}
	}
	return nil
}
