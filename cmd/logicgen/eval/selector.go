package eval

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// SelectorEvaluator evaluates selectors to get entities
type SelectorEvaluator struct{}

// NewSelectorEvaluator creates a new selector evaluator
func NewSelectorEvaluator() *SelectorEvaluator {
	return &SelectorEvaluator{}
}

// Select selects entities based on the selector
func (se *SelectorEvaluator) Select(ctx *Context, selector *ast.Selector) ([]interface{}, error) {
	if selector == nil {
		return []interface{}{ctx.State}, nil
	}

	switch selector.Type {
	case ast.SelectorTypeAll:
		return se.selectAll(ctx, selector)
	case ast.SelectorTypeFilter:
		return se.selectFilter(ctx, selector)
	case ast.SelectorTypeSingle:
		return se.selectSingle(ctx, selector)
	case ast.SelectorTypeRelated:
		return se.selectRelated(ctx, selector)
	case ast.SelectorTypeNearest:
		return se.selectNearest(ctx, selector, true)
	case ast.SelectorTypeFarthest:
		return se.selectNearest(ctx, selector, false)
	default:
		return nil, fmt.Errorf("unknown selector type: %s", selector.Type)
	}
}

// selectAll selects all entities of a type
func (se *SelectorEvaluator) selectAll(ctx *Context, selector *ast.Selector) ([]interface{}, error) {
	return se.getEntities(ctx, selector.Entity)
}

// selectFilter selects entities matching a condition
func (se *SelectorEvaluator) selectFilter(ctx *Context, selector *ast.Selector) ([]interface{}, error) {
	entities, err := se.getEntities(ctx, selector.Entity)
	if err != nil {
		return nil, err
	}

	if selector.Where == nil {
		return entities, nil
	}

	var result []interface{}
	for _, entity := range entities {
		matches, err := se.matchesWhere(ctx, entity, selector.Where)
		if err != nil {
			return nil, err
		}
		if matches {
			result = append(result, entity)
		}
	}

	return result, nil
}

// selectSingle selects a single entity by ID
func (se *SelectorEvaluator) selectSingle(ctx *Context, selector *ast.Selector) ([]interface{}, error) {
	entities, err := se.getEntities(ctx, selector.Entity)
	if err != nil {
		return nil, err
	}

	// Resolve the ID to look for
	targetID, err := ctx.Resolve(selector.ID)
	if err != nil {
		return nil, err
	}

	// Determine key field
	keyField := "ID"
	if selector.Key != "" {
		keyField = string(selector.Key)
	}

	// Find matching entity
	for _, entity := range entities {
		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() == reflect.Struct {
			field := val.FieldByName(keyField)
			if field.IsValid() && field.CanInterface() {
				if fmt.Sprintf("%v", field.Interface()) == fmt.Sprintf("%v", targetID) {
					return []interface{}{entity}, nil
				}
			}
		}
	}

	return []interface{}{}, nil
}

// selectRelated selects related entities
func (se *SelectorEvaluator) selectRelated(ctx *Context, selector *ast.Selector) ([]interface{}, error) {
	// Get source entity
	sourceVal, err := ctx.Resolve(selector.From)
	if err != nil {
		return nil, err
	}

	// Get all target entities
	entities, err := se.getEntities(ctx, selector.Entity)
	if err != nil {
		return nil, err
	}

	// Find related entities based on relation field
	// Relation format: "fieldName" - the field on source that points to target
	relationField := selector.Relation
	if relationField == "" {
		return nil, fmt.Errorf("relation field not specified")
	}

	// Get the relation value from source
	sourceReflect := reflect.ValueOf(sourceVal)
	if sourceReflect.Kind() == reflect.Ptr {
		sourceReflect = sourceReflect.Elem()
	}

	if sourceReflect.Kind() != reflect.Struct {
		return nil, fmt.Errorf("source must be a struct")
	}

	relationVal := sourceReflect.FieldByName(relationField)
	if !relationVal.IsValid() {
		return nil, fmt.Errorf("relation field not found: %s", relationField)
	}

	// Match entities by ID
	var result []interface{}
	relationIDs := se.extractIDs(relationVal)

	for _, entity := range entities {
		entityID := se.getEntityID(entity)
		for _, relID := range relationIDs {
			if fmt.Sprintf("%v", entityID) == fmt.Sprintf("%v", relID) {
				result = append(result, entity)
				break
			}
		}
	}

	return result, nil
}

// selectNearest selects entities nearest (or farthest) from a point
func (se *SelectorEvaluator) selectNearest(ctx *Context, selector *ast.Selector, nearest bool) ([]interface{}, error) {
	entities, err := se.getEntities(ctx, selector.Entity)
	if err != nil {
		return nil, err
	}

	// Get origin position
	origin, err := ctx.Resolve(selector.Origin)
	if err != nil {
		return nil, err
	}
	originPoint, err := toGeoPoint(origin)
	if err != nil {
		return nil, fmt.Errorf("origin: %w", err)
	}

	// Determine position field
	posField := "Position"
	if selector.Position != "" {
		// Extract field name from path
		posPath := string(selector.Position)
		if len(posPath) > 2 && posPath[:2] == "$." {
			posField = posPath[2:]
		}
	}

	type entityWithDist struct {
		entity   interface{}
		distance float64
	}

	var withDist []entityWithDist

	for _, entity := range entities {
		// Get entity position
		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			continue
		}

		field := val.FieldByName(posField)
		if !field.IsValid() || !field.CanInterface() {
			continue
		}

		entityPoint, err := toGeoPoint(field.Interface())
		if err != nil {
			continue
		}

		dist := haversineDistance(originPoint, entityPoint)

		// Apply distance constraints
		if selector.MaxDistance > 0 && dist > selector.MaxDistance {
			continue
		}
		if selector.MinDistance > 0 && dist < selector.MinDistance {
			continue
		}

		withDist = append(withDist, entityWithDist{entity: entity, distance: dist})
	}

	// Sort by distance
	sort.Slice(withDist, func(i, j int) bool {
		if nearest {
			return withDist[i].distance < withDist[j].distance
		}
		return withDist[i].distance > withDist[j].distance
	})

	// Apply limit
	limit := len(withDist)
	if selector.Limit > 0 && selector.Limit < limit {
		limit = selector.Limit
	}

	result := make([]interface{}, limit)
	for i := 0; i < limit; i++ {
		result[i] = withDist[i].entity
	}

	return result, nil
}

// getEntities gets all entities of a type from state
func (se *SelectorEvaluator) getEntities(ctx *Context, entityType string) ([]interface{}, error) {
	stateVal := ctx.GetStateValue()

	// Try to find field with entity name
	field := stateVal.FieldByName(entityType)
	if !field.IsValid() {
		return nil, fmt.Errorf("entity type not found: %s", entityType)
	}

	// Handle slice/array
	if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
		result := make([]interface{}, field.Len())
		for i := 0; i < field.Len(); i++ {
			elem := field.Index(i)
			if elem.CanInterface() {
				result[i] = elem.Interface()
			}
		}
		return result, nil
	}

	// Handle map
	if field.Kind() == reflect.Map {
		result := make([]interface{}, 0, field.Len())
		iter := field.MapRange()
		for iter.Next() {
			if iter.Value().CanInterface() {
				result = append(result, iter.Value().Interface())
			}
		}
		return result, nil
	}

	// Single entity
	if field.CanInterface() {
		return []interface{}{field.Interface()}, nil
	}

	return nil, fmt.Errorf("cannot iterate entity type: %s", entityType)
}

// matchesWhere checks if an entity matches a where clause
func (se *SelectorEvaluator) matchesWhere(ctx *Context, entity interface{}, where *ast.WhereClause) (bool, error) {
	// Get field value
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return false, fmt.Errorf("entity must be a struct")
	}

	field := val.FieldByName(where.Field)
	if !field.IsValid() {
		return false, nil
	}

	var fieldVal interface{}
	if field.CanInterface() {
		fieldVal = field.Interface()
	} else {
		return false, nil
	}

	// Resolve the comparison value
	compareVal, err := ctx.Resolve(where.Value)
	if err != nil {
		return false, err
	}

	// Compare
	return compare(fieldVal, compareVal, where.Op)
}

// extractIDs extracts IDs from a relation value (single or slice)
func (se *SelectorEvaluator) extractIDs(val reflect.Value) []interface{} {
	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		result := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			if val.Index(i).CanInterface() {
				result[i] = val.Index(i).Interface()
			}
		}
		return result
	}
	if val.CanInterface() {
		return []interface{}{val.Interface()}
	}
	return nil
}

// getEntityID gets the ID field from an entity
func (se *SelectorEvaluator) getEntityID(entity interface{}) interface{} {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		field := val.FieldByName("ID")
		if field.IsValid() && field.CanInterface() {
			return field.Interface()
		}
	}
	return nil
}
