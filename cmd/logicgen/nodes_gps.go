package main

// ============================================================================
// GPS/Geometry Nodes - Registered via the extensibility system
// ============================================================================
//
// This file demonstrates how to register nodes using the registry API.
// External packages can use the same pattern to add custom nodes.

func init() {
	registerGpsNodes()
}

// registerGpsNodes registers all GPS-related nodes
func registerGpsNodes() {
	// GpsDistance - Calculate distance between two GPS coordinates in meters
	MustRegisterNode(NodeDefinition{
		Type:        NodeGpsDistance,
		Category:    "GPS",
		Description: "Calculate distance between two GPS coordinates in meters (Haversine formula)",
		Inputs: []PortDefinition{
			{Name: "lat1", Type: "float64", Required: true},
			{Name: "lng1", Type: "float64", Required: true},
			{Name: "lat2", Type: "float64", Required: true},
			{Name: "lng2", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "distance", Type: "float64", Required: true},
		},
		Generator: generateGpsDistanceNode,
	})

	// GpsMoveToward - Move from point A toward point B by distance
	MustRegisterNode(NodeDefinition{
		Type:        NodeGpsMoveToward,
		Category:    "GPS",
		Description: "Calculate new GPS position after moving toward target by specified distance",
		Inputs: []PortDefinition{
			{Name: "fromLat", Type: "float64", Required: true},
			{Name: "fromLng", Type: "float64", Required: true},
			{Name: "toLat", Type: "float64", Required: true},
			{Name: "toLng", Type: "float64", Required: true},
			{Name: "distance", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "newLat", Type: "float64", Required: true},
			{Name: "newLng", Type: "float64", Required: true},
		},
		Generator: generateGpsMoveTowardNode,
	})

	// PointInCircle - Check if point is within circle radius
	MustRegisterNode(NodeDefinition{
		Type:        NodePointInCircle,
		Category:    "GPS",
		Description: "Check if a GPS point is within a circle defined by center and radius",
		Inputs: []PortDefinition{
			{Name: "pointLat", Type: "float64", Required: true},
			{Name: "pointLng", Type: "float64", Required: true},
			{Name: "centerLat", Type: "float64", Required: true},
			{Name: "centerLng", Type: "float64", Required: true},
			{Name: "radiusMeters", Type: "float64", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "isInside", Type: "bool", Required: true},
			{Name: "distance", Type: "float64", Required: true},
		},
		Generator: generatePointInCircleNode,
	})

	// PointInPolygon - Check if point is inside polygon
	MustRegisterNode(NodeDefinition{
		Type:        NodePointInPolygon,
		Category:    "GPS",
		Description: "Check if a GPS point is inside a polygon (ray casting algorithm)",
		Inputs: []PortDefinition{
			{Name: "pointLat", Type: "float64", Required: true},
			{Name: "pointLng", Type: "float64", Required: true},
			{Name: "polygon", Type: "[]GpsCoord", Required: true},
		},
		Outputs: []PortDefinition{
			{Name: "isInside", Type: "bool", Required: true},
		},
		Generator: generatePointInPolygonNode,
	})
}

// ============================================================================
// Generator Functions
// ============================================================================

func generateGpsDistanceNode(ctx *GeneratorContext, node *Node) error {
	lat1, err := ctx.ResolveInput(node, "lat1")
	if err != nil {
		return err
	}
	lng1, err := ctx.ResolveInput(node, "lng1")
	if err != nil {
		return err
	}
	lat2, err := ctx.ResolveInput(node, "lat2")
	if err != nil {
		return err
	}
	lng2, err := ctx.ResolveInput(node, "lng2")
	if err != nil {
		return err
	}

	outputVar := ctx.AllocateVariable(node, "distance", "float64")
	prefix := "_" + node.ID + "_"

	// Haversine formula for GPS distance
	ctx.WriteLine("// Haversine distance calculation")
	ctx.WriteLine("%slat1Rad := %s * math.Pi / 180.0", prefix, lat1)
	ctx.WriteLine("%slat2Rad := %s * math.Pi / 180.0", prefix, lat2)
	ctx.WriteLine("%sdeltaLat := (%s - %s) * math.Pi / 180.0", prefix, lat2, lat1)
	ctx.WriteLine("%sdeltaLng := (%s - %s) * math.Pi / 180.0", prefix, lng2, lng1)
	ctx.WriteLine("%sa := math.Sin(%sdeltaLat/2)*math.Sin(%sdeltaLat/2) + math.Cos(%slat1Rad)*math.Cos(%slat2Rad)*math.Sin(%sdeltaLng/2)*math.Sin(%sdeltaLng/2)",
		prefix, prefix, prefix, prefix, prefix, prefix, prefix)
	ctx.WriteLine("%sc := 2 * math.Atan2(math.Sqrt(%sa), math.Sqrt(1-%sa))", prefix, prefix, prefix)
	ctx.WriteLine("%s := 6371000.0 * %sc", outputVar, prefix)

	return nil
}

func generateGpsMoveTowardNode(ctx *GeneratorContext, node *Node) error {
	fromLat, err := ctx.ResolveInput(node, "fromLat")
	if err != nil {
		return err
	}
	fromLng, err := ctx.ResolveInput(node, "fromLng")
	if err != nil {
		return err
	}
	toLat, err := ctx.ResolveInput(node, "toLat")
	if err != nil {
		return err
	}
	toLng, err := ctx.ResolveInput(node, "toLng")
	if err != nil {
		return err
	}
	distance, err := ctx.ResolveInput(node, "distance")
	if err != nil {
		return err
	}

	newLatVar := ctx.AllocateVariable(node, "newLat", "float64")
	newLngVar := ctx.AllocateVariable(node, "newLng", "float64")
	prefix := "_" + node.ID + "_"

	// Calculate bearing and move
	ctx.WriteLine("// Calculate bearing and move toward target")
	ctx.WriteLine("%sfromLatRad := %s * math.Pi / 180.0", prefix, fromLat)
	ctx.WriteLine("%sfromLngRad := %s * math.Pi / 180.0", prefix, fromLng)
	ctx.WriteLine("%stoLatRad := %s * math.Pi / 180.0", prefix, toLat)
	ctx.WriteLine("%stoLngRad := %s * math.Pi / 180.0", prefix, toLng)
	ctx.WriteLine("%sdLng := %stoLngRad - %sfromLngRad", prefix, prefix, prefix)
	ctx.WriteLine("%sx := math.Cos(%stoLatRad) * math.Sin(%sdLng)", prefix, prefix, prefix)
	ctx.WriteLine("%sy := math.Cos(%sfromLatRad)*math.Sin(%stoLatRad) - math.Sin(%sfromLatRad)*math.Cos(%stoLatRad)*math.Cos(%sdLng)",
		prefix, prefix, prefix, prefix, prefix, prefix)
	ctx.WriteLine("%sbearing := math.Atan2(%sx, %sy)", prefix, prefix, prefix)
	ctx.WriteLine("%sangularDist := %s / 6371000.0", prefix, distance)
	ctx.WriteLine("%snewLatRad := math.Asin(math.Sin(%sfromLatRad)*math.Cos(%sangularDist) + math.Cos(%sfromLatRad)*math.Sin(%sangularDist)*math.Cos(%sbearing))",
		prefix, prefix, prefix, prefix, prefix, prefix)
	ctx.WriteLine("%snewLngRad := %sfromLngRad + math.Atan2(math.Sin(%sbearing)*math.Sin(%sangularDist)*math.Cos(%sfromLatRad), math.Cos(%sangularDist)-math.Sin(%sfromLatRad)*math.Sin(%snewLatRad))",
		prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix)
	ctx.WriteLine("%s := %snewLatRad * 180.0 / math.Pi", newLatVar, prefix)
	ctx.WriteLine("%s := %snewLngRad * 180.0 / math.Pi", newLngVar, prefix)

	return nil
}

func generatePointInCircleNode(ctx *GeneratorContext, node *Node) error {
	pointLat, err := ctx.ResolveInput(node, "pointLat")
	if err != nil {
		return err
	}
	pointLng, err := ctx.ResolveInput(node, "pointLng")
	if err != nil {
		return err
	}
	centerLat, err := ctx.ResolveInput(node, "centerLat")
	if err != nil {
		return err
	}
	centerLng, err := ctx.ResolveInput(node, "centerLng")
	if err != nil {
		return err
	}
	radius := ctx.ResolveInputOptional(node, "radiusMeters", "")
	if radius == "" {
		// Fallback to "radius" for backwards compatibility
		radius = ctx.ResolveInputOptional(node, "radius", "0")
	}

	distanceVar := ctx.AllocateVariable(node, "distance", "float64")
	isInsideVar := ctx.AllocateVariable(node, "isInside", "bool")
	prefix := "_" + node.ID + "_"

	// Haversine formula for GPS distance
	ctx.WriteLine("// Point in circle check using Haversine distance")
	ctx.WriteLine("%slat1Rad := %s * math.Pi / 180.0", prefix, pointLat)
	ctx.WriteLine("%slat2Rad := %s * math.Pi / 180.0", prefix, centerLat)
	ctx.WriteLine("%sdeltaLat := (%s - %s) * math.Pi / 180.0", prefix, centerLat, pointLat)
	ctx.WriteLine("%sdeltaLng := (%s - %s) * math.Pi / 180.0", prefix, centerLng, pointLng)
	ctx.WriteLine("%sa := math.Sin(%sdeltaLat/2)*math.Sin(%sdeltaLat/2) + math.Cos(%slat1Rad)*math.Cos(%slat2Rad)*math.Sin(%sdeltaLng/2)*math.Sin(%sdeltaLng/2)",
		prefix, prefix, prefix, prefix, prefix, prefix, prefix)
	ctx.WriteLine("%sc := 2 * math.Atan2(math.Sqrt(%sa), math.Sqrt(1-%sa))", prefix, prefix, prefix)
	ctx.WriteLine("%s := 6371000.0 * %sc", distanceVar, prefix)
	ctx.WriteLine("%s := %s <= %s", isInsideVar, distanceVar, radius)

	return nil
}

func generatePointInPolygonNode(ctx *GeneratorContext, node *Node) error {
	pointLat, err := ctx.ResolveInput(node, "pointLat")
	if err != nil {
		return err
	}
	pointLng, err := ctx.ResolveInput(node, "pointLng")
	if err != nil {
		return err
	}
	polygonSrc, err := ctx.ResolveInput(node, "polygon")
	if err != nil {
		return err
	}

	isInsideVar := ctx.AllocateVariable(node, "isInside", "bool")
	prefix := "_" + node.ID + "_"

	// Ray casting algorithm for point-in-polygon
	ctx.WriteLine("// Ray casting algorithm for point-in-polygon")
	ctx.WriteLine("%sn := len(%s)", prefix, polygonSrc)
	ctx.WriteLine("%sinside := false", prefix)
	ctx.WriteLine("for %si, %sj := 0, %sn-1; %si < %sn; %sj, %si = %si, %si+1 {", prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix)
	ctx.Indent()
	ctx.WriteLine("%sxi, %syi := %s[%si].Lng, %s[%si].Lat", prefix, prefix, polygonSrc, prefix, polygonSrc, prefix)
	ctx.WriteLine("%sxj, %syj := %s[%sj].Lng, %s[%sj].Lat", prefix, prefix, polygonSrc, prefix, polygonSrc, prefix)
	ctx.WriteLine("if ((%syi > %s) != (%syj > %s)) && (%s < (%sxj-%sxi)*(%s-%syi)/(%syj-%syi)+%sxi) {",
		prefix, pointLat, prefix, pointLat, pointLng, prefix, prefix, pointLat, prefix, prefix, prefix, prefix)
	ctx.Indent()
	ctx.WriteLine("%sinside = !%sinside", prefix, prefix)
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.Dedent()
	ctx.WriteLine("}")
	ctx.WriteLine("%s := %sinside", isInsideVar, prefix)

	return nil
}
