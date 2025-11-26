package pcb

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp/kicadsexp"
)

// parsePad extracts a pad definition from a footprint
// Expected format: (pad "number" type shape (at x y [angle]) (size w h) (layers ...) (net n) ...)
func parsePad(node kicadsexp.Sexp, netMap *NetMap) (*Pad, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected pad list, got leaf")
	}

	pad := &Pad{}

	// Parse pad number/name (second element after "pad")
	number, err := getQuotedString(node, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pad number: %w", err)
	}
	pad.Number = number

	// Parse pad type (third element: thru_hole, smd, connect, np_thru_hole)
	padType, err := getString(node, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pad type: %w", err)
	}
	pad.Type = padType

	// Parse pad shape (fourth element: circle, rect, oval, roundrect, trapezoid, custom)
	shape, err := getString(node, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pad shape: %w", err)
	}
	pad.Shape = shape

	// Parse position (at x y [angle])
	if atNode, found := findNode(node, "at"); found {
		x, err := getFloat(atNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pad X position: %w", err)
		}
		y, err := getFloat(atNode, 2)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pad Y position: %w", err)
		}
		pad.Position.X = x
		pad.Position.Y = y

		// Angle is optional
		if angle, err := getFloat(atNode, 3); err == nil {
			pad.Position.Angle = Angle(angle)
		}
	} else {
		return nil, fmt.Errorf("missing required 'at' position")
	}

	// Parse size
	if sizeNode, found := findNode(node, "size"); found {
		width, err := getFloat(sizeNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pad width: %w", err)
		}
		height, err := getFloat(sizeNode, 2)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pad height: %w", err)
		}
		pad.Size = Size{Width: width, Height: height}
	} else {
		return nil, fmt.Errorf("missing required 'size' field")
	}

	// Parse drill (for through-hole pads)
	if drillNode, found := findNode(node, "drill"); found {
		// Drill can be just a number or (drill (diameter d))
		drill, err := getFloat(drillNode, 1)
		if err == nil {
			pad.Drill = drill
		}
	}

	// Parse layers
	if layersNode, found := findNode(node, "layers"); found {
		if layersNode.IsLeaf() {
			return nil, fmt.Errorf("expected layers list, got leaf")
		}

		items := getListItems(layersNode)
		var layers []string
		for _, item := range items {
			if item.IsLeaf() {
				layerName := item.String()
				// Remove quotes
				if len(layerName) > 0 && layerName[0] == '"' {
					layerName = layerName[1:]
				}
				if len(layerName) > 0 && layerName[len(layerName)-1] == '"' {
					layerName = layerName[:len(layerName)-1]
				}
				// Skip "layers" keyword
				if layerName != "layers" && layerName != "" {
					layers = append(layers, layerName)
				}
			}
		}
		pad.Layers = LayerSet(layers)
	} else {
		return nil, fmt.Errorf("missing required 'layers' field")
	}

	// Parse net (optional)
	if netNode, found := findNode(node, "net"); found {
		netNum, err := getInt(netNode, 1)
		if err == nil && netMap != nil {
			if net, ok := netMap.GetByNumber(netNum); ok {
				pad.Net = net
			}
		}
	}

	return pad, nil
}

// parseFootprint extracts a footprint (component) definition
// Expected format: (footprint "library:name" (layer "layer") (at x y [angle]) ...)
func parseFootprint(node kicadsexp.Sexp, netMap *NetMap) (*Footprint, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected footprint list, got leaf")
	}

	footprint := &Footprint{}

	// Parse footprint name (library:name format, second element after "footprint")
	fpName, err := getQuotedString(node, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to parse footprint name: %w", err)
	}

	// Split library:name format
	// Example: "Resistor_SMD:R_0603_1608Metric"
	colonIdx := -1
	for i, c := range fpName {
		if c == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx > 0 {
		footprint.Library = fpName[:colonIdx]
		footprint.Name = fpName[colonIdx+1:]
	} else {
		footprint.Name = fpName
	}

	// Parse layer
	if layerNode, found := findNode(node, "layer"); found {
		layer, err := getQuotedString(layerNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layer: %w", err)
		}
		footprint.Layer = layer
	} else {
		return nil, fmt.Errorf("missing required 'layer' field")
	}

	// Parse position (at x y [angle])
	if atNode, found := findNode(node, "at"); found {
		x, err := getFloat(atNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse X position: %w", err)
		}
		y, err := getFloat(atNode, 2)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Y position: %w", err)
		}
		footprint.Position.X = x
		footprint.Position.Y = y

		// Angle is optional
		if angle, err := getFloat(atNode, 3); err == nil {
			footprint.Position.Angle = Angle(angle)
		}
	} else {
		return nil, fmt.Errorf("missing required 'at' position")
	}

	// Parse properties (Reference and Value are most important)
	propertyNodes := findAllNodes(node, "property")
	for _, propNode := range propertyNodes {
		propName, err := getQuotedString(propNode, 1)
		if err != nil {
			continue
		}
		propValue, err := getQuotedString(propNode, 2)
		if err != nil {
			continue
		}

		switch propName {
		case "Reference":
			footprint.Reference = propValue
		case "Value":
			footprint.Value = propValue
		}
	}

	// Parse pads
	padNodes := findAllNodes(node, "pad")
	for _, padNode := range padNodes {
		pad, err := parsePad(padNode, netMap)
		if err != nil {
			// Log error but continue parsing other pads
			continue
		}
		footprint.Pads = append(footprint.Pads, *pad)
	}

	// Parse graphics within footprint (fp_line, fp_circle, fp_arc, fp_text, etc.)
	// Parse fp_line elements
	fpLineNodes := findAllNodes(node, "fp_line")
	for _, lineNode := range fpLineNodes {
		line, err := parseGrLine(lineNode)
		if err != nil {
			// Skip lines that fail to parse
			continue
		}
		// Convert GrLine to Graphic
		footprint.Graphics = append(footprint.Graphics, Graphic{
			Type:   "line",
			Layer:  line.Layer,
			Start:  line.Start,
			End:    line.End,
			Stroke: line.Stroke,
		})
	}

	// Parse fp_circle elements
	fpCircleNodes := findAllNodes(node, "fp_circle")
	for _, circleNode := range fpCircleNodes {
		circle, err := parseGrCircle(circleNode)
		if err != nil {
			continue
		}
		// Convert GrCircle to Graphic
		footprint.Graphics = append(footprint.Graphics, Graphic{
			Type:   "circle",
			Layer:  circle.Layer,
			Center: circle.Center,
			End:    circle.End,
			Stroke: circle.Stroke,
			Fill:   circle.Fill,
		})
	}

	// Parse fp_arc elements
	fpArcNodes := findAllNodes(node, "fp_arc")
	for _, arcNode := range fpArcNodes {
		arc, err := parseGrArc(arcNode)
		if err != nil {
			continue
		}
		// Convert GrArc to Graphic
		footprint.Graphics = append(footprint.Graphics, Graphic{
			Type:   "arc",
			Layer:  arc.Layer,
			Start:  arc.Start,
			Center: arc.Mid, // Using Mid as center approximation
			End:    arc.End,
			Stroke: arc.Stroke,
		})
	}

	// Parse fp_rect elements
	fpRectNodes := findAllNodes(node, "fp_rect")
	for _, rectNode := range fpRectNodes {
		rect, err := parseGrRect(rectNode)
		if err != nil {
			continue
		}
		// Convert GrRect to Graphic
		footprint.Graphics = append(footprint.Graphics, Graphic{
			Type:   "rect",
			Layer:  rect.Layer,
			Start:  rect.Start,
			End:    rect.End,
			Stroke: rect.Stroke,
			Fill:   rect.Fill,
		})
	}

	// Parse fp_poly elements
	fpPolyNodes := findAllNodes(node, "fp_poly")
	for _, polyNode := range fpPolyNodes {
		poly, err := parseGrPoly(polyNode)
		if err != nil {
			continue
		}
		// Convert GrPoly to Graphic
		footprint.Graphics = append(footprint.Graphics, Graphic{
			Type:   "polygon",
			Layer:  poly.Layer,
			Points: poly.Points,
			Stroke: poly.Stroke,
			Fill:   poly.Fill,
		})
	}

	return footprint, nil
}

// parseFootprints extracts all footprint definitions from the root node
// Finds and parses all (footprint ...) nodes
func parseFootprints(root kicadsexp.Sexp, netMap *NetMap) ([]Footprint, error) {
	if root.IsLeaf() {
		return nil, fmt.Errorf("expected root list")
	}

	// Find all (footprint ...) nodes
	footprintNodes := findAllNodes(root, "footprint")
	if len(footprintNodes) == 0 {
		// No footprints is valid (empty board)
		return []Footprint{}, nil
	}

	var footprints []Footprint

	for _, fpNode := range footprintNodes {
		footprint, err := parseFootprint(fpNode, netMap)
		if err != nil {
			// Log error but continue parsing other footprints
			// In a real application, you might want to collect errors
			continue
		}
		footprints = append(footprints, *footprint)
	}

	return footprints, nil
}
