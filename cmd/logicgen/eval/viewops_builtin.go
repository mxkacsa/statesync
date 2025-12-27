package eval

import (
	"fmt"
	"math"
	"sort"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

func init() {
	registerCollectionOps()
	registerAggregationOps()
	registerGeoOps()
}

// =============================================================================
// Collection Operations
// =============================================================================

func registerCollectionOps() {
	// Filter
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpFilter),
		Category:    CategoryCollection,
		Description: "Filters entities based on a where clause",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "where", Type: "WhereClause", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]Entity"},
		},
		Func: viewOpFilter,
	})

	// FlatMap - flatten nested arrays
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpFlatMap),
		Category:    CategoryCollection,
		Description: "Flattens a nested array field from each entity",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "field", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]interface{}"},
		},
		Func: viewOpFlatMap,
	})

	// Map
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpMap),
		Category:    CategoryCollection,
		Description: "Transforms entities by projecting fields",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "fields", Type: "map[string]string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]map[string]interface{}"},
		},
		Func: viewOpMap,
	})

	// OrderBy
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpOrderBy),
		Category:    CategoryCollection,
		Description: "Sorts entities by a field",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "by", Type: "string", Required: true},
			{Name: "order", Type: "string", Required: false, Default: "asc"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]Entity"},
		},
		Func: viewOpOrderBy,
	})

	// GroupBy
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpGroupBy),
		Category:    CategoryCollection,
		Description: "Groups entities by a field",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "groupField", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "map[interface{}][]Entity"},
		},
		Func: viewOpGroupBy,
	})

	// First
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpFirst),
		Category:    CategoryCollection,
		Description: "Returns the first entity",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "Entity"},
		},
		Func: viewOpFirst,
	})

	// Last
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpLast),
		Category:    CategoryCollection,
		Description: "Returns the last entity",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "Entity"},
		},
		Func: viewOpLast,
	})

	// Limit
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpLimit),
		Category:    CategoryCollection,
		Description: "Limits the number of results",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "count", Type: "int", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]Entity"},
		},
		Func: viewOpLimit,
	})

	// Distinct
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpDistinct),
		Category:    CategoryCollection,
		Description: "Returns distinct values of a field",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "field", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]interface{}"},
		},
		Func: viewOpDistinct,
	})
}

// =============================================================================
// Aggregation Operations
// =============================================================================

func registerAggregationOps() {
	// Min
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpMin),
		Category:    CategoryAggregation,
		Description: "Finds the minimum value",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "field", Type: "string", Required: true},
			{Name: "return", Type: "string", Required: false, Default: "value"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64|Entity"},
		},
		Func: viewOpMin,
	})

	// Max
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpMax),
		Category:    CategoryAggregation,
		Description: "Finds the maximum value",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "field", Type: "string", Required: true},
			{Name: "return", Type: "string", Required: false, Default: "value"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64|Entity"},
		},
		Func: viewOpMax,
	})

	// Sum
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpSum),
		Category:    CategoryAggregation,
		Description: "Calculates the sum of a field",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "field", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: viewOpSum,
	})

	// Count
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpCount),
		Category:    CategoryAggregation,
		Description: "Counts entities",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "int"},
		},
		Func: viewOpCount,
	})

	// Avg
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpAvg),
		Category:    CategoryAggregation,
		Description: "Calculates the average of a field",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "field", Type: "string", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64"},
		},
		Func: viewOpAvg,
	})
}

// =============================================================================
// Geo Operations
// =============================================================================

func registerGeoOps() {
	// Distance
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpDistance),
		Category:    CategoryGeo,
		Description: "Calculates distance between points or from origin to entities",
		Inputs: []PortDefinition{
			{Name: "from", Type: "GeoPoint", Required: true},
			{Name: "to", Type: "GeoPoint", Required: false},
			{Name: "position", Type: "string", Required: false},
			{Name: "unit", Type: "string", Required: false, Default: "meters"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "float64|[]float64"},
		},
		Func: viewOpDistance,
	})

	// Nearest
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpNearest),
		Category:    CategoryGeo,
		Description: "Finds the nearest entities to an origin",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "origin", Type: "GeoPoint", Required: true},
			{Name: "position", Type: "string", Required: true},
			{Name: "count", Type: "int", Required: false, Default: 1},
			{Name: "maxDistance", Type: "float64", Required: false},
			{Name: "minDistance", Type: "float64", Required: false},
			{Name: "return", Type: "string", Required: false, Default: "entities"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]Entity|float64|[]map"},
		},
		Func: viewOpNearest,
	})

	// Farthest
	MustRegisterViewOp(&ViewOpDefinition{
		Name:        string(ast.ViewOpFarthest),
		Category:    CategoryGeo,
		Description: "Finds the farthest entities from an origin",
		Inputs: []PortDefinition{
			{Name: "entities", Type: "[]Entity", Required: true},
			{Name: "origin", Type: "GeoPoint", Required: true},
			{Name: "position", Type: "string", Required: true},
			{Name: "count", Type: "int", Required: false, Default: 1},
			{Name: "maxDistance", Type: "float64", Required: false},
			{Name: "minDistance", Type: "float64", Required: false},
			{Name: "return", Type: "string", Required: false, Default: "entities"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "[]Entity|float64|[]map"},
		},
		Func: viewOpFarthest,
	})
}

// =============================================================================
// View Operation Implementations
// =============================================================================

func viewOpFilter(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("filter requires entity list input")
	}

	if op.Where == nil {
		return entities, nil
	}

	var result []interface{}
	for _, entity := range entities {
		matches, err := matchesWhereClause(ctx, entity, op.Where)
		if err != nil {
			return nil, err
		}
		if matches {
			result = append(result, entity)
		}
	}

	return result, nil
}

func viewOpMap(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("map requires entity list input")
	}

	result := make([]map[string]interface{}, len(entities))
	for i, entity := range entities {
		entityCtx := ctx.WithEntity(entity, i)
		mapped := make(map[string]interface{})

		for key, valExpr := range op.Fields {
			val, err := entityCtx.Resolve(valExpr)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", key, err)
			}
			mapped[key] = val
		}

		result[i] = mapped
	}

	return result, nil
}

func viewOpFlatMap(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("flatMap requires entity list input")
	}

	fieldPath := string(op.Field)
	if len(fieldPath) > 2 && fieldPath[:2] == "$." {
		fieldPath = fieldPath[2:]
	}

	var result []interface{}
	for _, entity := range entities {
		val, err := getFieldValue(entity, fieldPath)
		if err != nil {
			continue
		}

		// If the value is a slice, flatten it
		valSlice, ok := toEntitySlice(val)
		if ok {
			result = append(result, valSlice...)
		} else {
			// Single value, just add it
			result = append(result, val)
		}
	}

	return result, nil
}

func viewOpOrderBy(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("orderBy requires entity list input")
	}

	if len(entities) <= 1 {
		return entities, nil
	}

	sortField := string(op.By)
	if len(sortField) > 2 && sortField[:2] == "$." {
		sortField = sortField[2:]
	}

	// Schwartzian transform: pre-compute sort keys to avoid repeated reflection
	// This reduces getFieldValue calls from O(N log N) to O(N)
	type sortItem struct {
		entity interface{}
		numKey float64
		strKey string
		isNum  bool
	}

	items := make([]sortItem, len(entities))
	for i, entity := range entities {
		val, _ := getFieldValue(entity, sortField)
		numVal, isNum := toFloat64(val)
		items[i] = sortItem{
			entity: entity,
			numKey: numVal,
			strKey: fmt.Sprintf("%v", val),
			isNum:  isNum,
		}
	}

	sort.Slice(items, func(i, j int) bool {
		var less bool
		if items[i].isNum && items[j].isNum {
			less = items[i].numKey < items[j].numKey
		} else {
			less = items[i].strKey < items[j].strKey
		}

		if op.Order == "desc" {
			return !less
		}
		return less
	})

	// Extract sorted entities
	sorted := make([]interface{}, len(items))
	for i, item := range items {
		sorted[i] = item.entity
	}

	return sorted, nil
}

func viewOpGroupBy(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("groupBy requires entity list input")
	}

	groups := make(map[interface{}][]interface{})
	for _, entity := range entities {
		key, err := getFieldValue(entity, op.GroupField)
		if err != nil {
			continue
		}
		groups[key] = append(groups[key], entity)
	}

	return groups, nil
}

func viewOpFirst(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("first requires entity list input")
	}

	if len(entities) == 0 {
		return nil, nil
	}
	return entities[0], nil
}

func viewOpLast(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("last requires entity list input")
	}

	if len(entities) == 0 {
		return nil, nil
	}
	return entities[len(entities)-1], nil
}

func viewOpLimit(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("limit requires entity list input")
	}

	if op.Count <= 0 || op.Count >= len(entities) {
		return entities, nil
	}
	return entities[:op.Count], nil
}

func viewOpDistinct(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("distinct requires entity list input")
	}

	seen := make(map[interface{}]bool)
	var result []interface{}

	for _, entity := range entities {
		val, err := getFieldValue(entity, string(op.Field))
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

func viewOpMin(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("min requires entity list input")
	}

	if len(entities) == 0 {
		return nil, nil
	}

	var minVal float64 = math.Inf(1)
	var minEntity interface{}

	for _, entity := range entities {
		val, err := getFieldValue(entity, string(op.Field))
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

	if op.Return == "entity" {
		return minEntity, nil
	}
	if math.IsInf(minVal, 1) {
		return nil, nil
	}
	return minVal, nil
}

func viewOpMax(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("max requires entity list input")
	}

	if len(entities) == 0 {
		return nil, nil
	}

	var maxVal float64 = math.Inf(-1)
	var maxEntity interface{}

	for _, entity := range entities {
		val, err := getFieldValue(entity, string(op.Field))
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

	if op.Return == "entity" {
		return maxEntity, nil
	}
	if math.IsInf(maxVal, -1) {
		return nil, nil
	}
	return maxVal, nil
}

func viewOpSum(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("sum requires entity list input")
	}

	var sum float64
	for _, entity := range entities {
		val, err := getFieldValue(entity, string(op.Field))
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

func viewOpCount(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("count requires entity list input")
	}

	return len(entities), nil
}

func viewOpAvg(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		return nil, fmt.Errorf("avg requires entity list input")
	}

	if len(entities) == 0 {
		return 0.0, nil
	}

	sum, err := viewOpSum(ctx, op, input)
	if err != nil {
		return nil, err
	}

	sumVal, _ := toFloat64(sum)
	return sumVal / float64(len(entities)), nil
}

func viewOpDistance(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	fromVal, err := ctx.Resolve(op.From)
	if err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}
	fromPoint, err := toGeoPoint(fromVal)
	if err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}

	if op.Position != "" {
		entities, ok := toEntitySlice(input)
		if !ok {
			if input != nil {
				entities = []interface{}{input}
			} else {
				return nil, fmt.Errorf("distance with position requires entity input")
			}
		}

		posField := string(op.Position)
		if len(posField) > 2 && posField[:2] == "$." {
			posField = posField[2:]
		}

		if len(entities) == 1 {
			posVal, err := getFieldValue(entities[0], posField)
			if err != nil {
				return nil, fmt.Errorf("position field: %w", err)
			}
			toPoint, err := toGeoPoint(posVal)
			if err != nil {
				return nil, fmt.Errorf("position: %w", err)
			}
			distance := haversineDistance(fromPoint, toPoint)
			if op.Unit == "kilometers" || op.Unit == "km" {
				distance /= 1000
			}
			return distance, nil
		}

		distances := make([]float64, len(entities))
		for i, entity := range entities {
			posVal, err := getFieldValue(entity, posField)
			if err != nil {
				continue
			}
			toPoint, err := toGeoPoint(posVal)
			if err != nil {
				continue
			}
			distances[i] = haversineDistance(fromPoint, toPoint)
			if op.Unit == "kilometers" || op.Unit == "km" {
				distances[i] /= 1000
			}
		}
		return distances, nil
	}

	toVal, err := ctx.Resolve(op.To)
	if err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}
	toPoint, err := toGeoPoint(toVal)
	if err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}

	distance := haversineDistance(fromPoint, toPoint)
	if op.Unit == "kilometers" || op.Unit == "km" {
		distance /= 1000
	}

	return distance, nil
}

func viewOpNearest(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	return viewOpByDistance(ctx, op, input, true)
}

func viewOpFarthest(ctx *Context, op ast.ViewOperation, input interface{}) (interface{}, error) {
	return viewOpByDistance(ctx, op, input, false)
}

func viewOpByDistance(ctx *Context, op ast.ViewOperation, input interface{}, nearest bool) (interface{}, error) {
	entities, ok := toEntitySlice(input)
	if !ok {
		opName := "nearest"
		if !nearest {
			opName = "farthest"
		}
		return nil, fmt.Errorf("%s requires entity list input", opName)
	}

	originVal, err := ctx.Resolve(op.Origin)
	if err != nil {
		return nil, fmt.Errorf("origin: %w", err)
	}
	origin, err := toGeoPoint(originVal)
	if err != nil {
		return nil, fmt.Errorf("origin: %w", err)
	}

	posField := string(op.Position)
	if len(posField) > 2 && posField[:2] == "$." {
		posField = posField[2:]
	}

	type entityDist struct {
		entity   interface{}
		distance float64
	}

	var withDist []entityDist
	for _, entity := range entities {
		posVal, err := getFieldValue(entity, posField)
		if err != nil {
			continue
		}
		pos, err := toGeoPoint(posVal)
		if err != nil {
			continue
		}

		dist := haversineDistance(origin, pos)

		if op.MaxDistance > 0 && dist > op.MaxDistance {
			continue
		}
		if op.MinDistance > 0 && dist < op.MinDistance {
			continue
		}

		withDist = append(withDist, entityDist{entity, dist})
	}

	sort.Slice(withDist, func(i, j int) bool {
		if nearest {
			return withDist[i].distance < withDist[j].distance
		}
		return withDist[i].distance > withDist[j].distance
	})

	limit := len(withDist)
	if op.Count > 0 && op.Count < limit {
		limit = op.Count
	}

	switch op.Return {
	case "distance":
		if limit == 0 {
			return nil, nil
		}
		if limit == 1 || op.Count == 1 {
			return withDist[0].distance, nil
		}
		distances := make([]float64, limit)
		for i := 0; i < limit; i++ {
			distances[i] = withDist[i].distance
		}
		return distances, nil

	case "both":
		result := make([]map[string]interface{}, limit)
		for i := 0; i < limit; i++ {
			result[i] = map[string]interface{}{
				"entity":   withDist[i].entity,
				"distance": withDist[i].distance,
			}
		}
		return result, nil

	default:
		result := make([]interface{}, limit)
		for i := 0; i < limit; i++ {
			result[i] = withDist[i].entity
		}
		return result, nil
	}
}

// matchesWhereClause checks if an entity matches a where clause.
//
// NOTE: If a field doesn't exist on the entity, it returns false (no match)
// rather than an error. This is intentional ECS-style behavior where not all
// entities have all fields. For example, filtering "where Team == 'catcher'"
// should simply not match entities without a Team field.
//
// If you need to debug missing fields, enable context debugging or validate
// your schema at rule load time.
func matchesWhereClause(ctx *Context, entity interface{}, where *ast.WhereClause) (bool, error) {
	if len(where.And) > 0 {
		for _, clause := range where.And {
			matches, err := matchesWhereClause(ctx, entity, clause)
			if err != nil {
				return false, err
			}
			if !matches {
				return false, nil
			}
		}
		return true, nil
	}

	if len(where.Or) > 0 {
		for _, clause := range where.Or {
			matches, err := matchesWhereClause(ctx, entity, clause)
			if err != nil {
				return false, err
			}
			if matches {
				return true, nil
			}
		}
		return false, nil
	}

	if where.Not != nil {
		matches, err := matchesWhereClause(ctx, entity, where.Not)
		if err != nil {
			return false, err
		}
		return !matches, nil
	}

	fieldVal, err := getFieldValue(entity, where.Field)
	if err != nil {
		// Field doesn't exist on entity - treat as "no match" (ECS-style)
		// This allows filtering across heterogeneous entity types
		if handler := ctx.getDebugHandler(); handler != nil {
			entityType := fmt.Sprintf("%T", entity)
			handler.OnMissingField(entityType, where.Field, err)
		}
		return false, nil
	}

	entityCtx := ctx.WithEntity(entity, 0)
	compareVal, err := entityCtx.Resolve(where.Value)
	if err != nil {
		return false, err
	}

	matched, err := compare(fieldVal, compareVal, where.Op)
	if err != nil {
		return false, err
	}

	// Debug log the comparison result
	if handler := ctx.getDebugHandler(); handler != nil {
		entityType := fmt.Sprintf("%T", entity)
		handler.OnFilterMatch(entityType, where.Field, where.Op, compareVal, fieldVal, matched)
	}

	return matched, nil
}
