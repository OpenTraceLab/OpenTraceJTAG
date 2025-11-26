package pcb

import (
	"fmt"
	"strings"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp/kicadsexp"
)

// parsePosition extracts position coordinates from a (at x y) or (start x y) node
// Expected format: (at 100.5 75.3) or (start 100.5 75.3)
func parsePosition(node kicadsexp.Sexp) (Position, error) {
	if node.IsLeaf() {
		return Position{}, fmt.Errorf("expected position list, got leaf")
	}

	x, err := getFloat(node, 1)
	if err != nil {
		return Position{}, fmt.Errorf("failed to parse X coordinate: %w", err)
	}

	y, err := getFloat(node, 2)
	if err != nil {
		return Position{}, fmt.Errorf("failed to parse Y coordinate: %w", err)
	}

	return Position{X: x, Y: y}, nil
}

// parseColor extracts RGBA color from a (color r g b a) node
// Expected format: (color 255 0 0 1)
// Values are normalized to 0.0-1.0 range
func parseColor(node kicadsexp.Sexp) (Color, error) {
	if node.IsLeaf() {
		return Color{}, fmt.Errorf("expected color list, got leaf")
	}

	r, err := getFloat(node, 1)
	if err != nil {
		return Color{}, fmt.Errorf("failed to parse R component: %w", err)
	}

	g, err := getFloat(node, 2)
	if err != nil {
		return Color{}, fmt.Errorf("failed to parse G component: %w", err)
	}

	b, err := getFloat(node, 3)
	if err != nil {
		return Color{}, fmt.Errorf("failed to parse B component: %w", err)
	}

	// Alpha is optional, defaults to 1.0
	a := 1.0
	if aVal, err := getFloat(node, 4); err == nil {
		a = aVal
	}

	// Normalize to 0.0-1.0 if values are in 0-255 range
	if r > 1.0 || g > 1.0 || b > 1.0 {
		r /= 255.0
		g /= 255.0
		b /= 255.0
	}

	return Color{R: r, G: g, B: b, A: a}, nil
}

// parseStroke extracts stroke information from a (stroke ...) node
// Expected format: (stroke (width 0.15) (type solid) (color 255 0 0 1))
func parseStroke(node kicadsexp.Sexp) (Stroke, error) {
	if node.IsLeaf() {
		return Stroke{}, fmt.Errorf("expected stroke list, got leaf")
	}

	stroke := Stroke{
		Width: 0.15, // Default width
		Type:  "solid",
		Color: Color{R: 0, G: 0, B: 0, A: 1}, // Default black
	}

	// Parse width
	if widthNode, found := findNode(node, "width"); found {
		width, err := getFloat(widthNode, 1)
		if err != nil {
			return stroke, fmt.Errorf("failed to parse stroke width: %w", err)
		}
		stroke.Width = width
	}

	// Parse type
	if typeNode, found := findNode(node, "type"); found {
		strokeType, err := getString(typeNode, 1)
		if err == nil {
			stroke.Type = strokeType
		}
	}

	// Parse color
	if colorNode, found := findNode(node, "color"); found {
		color, err := parseColor(colorNode)
		if err == nil {
			stroke.Color = color
		}
	}

	return stroke, nil
}

// parseFill extracts fill information from a (fill ...) node
// Expected format: (fill (type solid) (color 255 0 0 1))
func parseFill(node kicadsexp.Sexp) (Fill, error) {
	if node.IsLeaf() {
		return Fill{}, fmt.Errorf("expected fill list, got leaf")
	}

	fill := Fill{
		Type:  "none", // Default to no fill
		Color: Color{R: 0, G: 0, B: 0, A: 0},
	}

	// Parse type
	if typeNode, found := findNode(node, "type"); found {
		fillType, err := getString(typeNode, 1)
		if err == nil {
			fill.Type = fillType
		}
	}

	// Parse color
	if colorNode, found := findNode(node, "color"); found {
		color, err := parseColor(colorNode)
		if err == nil {
			fill.Color = color
		}
	}

	return fill, nil
}

// parseGrLine extracts a line graphic element
// Expected format: (gr_line (start x1 y1) (end x2 y2) (stroke ...) (layer "F.Cu"))
func parseGrLine(node kicadsexp.Sexp) (*GrLine, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected gr_line list, got leaf")
	}

	line := &GrLine{
		Stroke: Stroke{Width: 0.15, Type: "solid"},
	}

	// Parse start position
	if startNode, found := findNode(node, "start"); found {
		start, err := parsePosition(startNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start position: %w", err)
		}
		line.Start = start
	} else {
		return nil, fmt.Errorf("missing required 'start' position")
	}

	// Parse end position
	if endNode, found := findNode(node, "end"); found {
		end, err := parsePosition(endNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end position: %w", err)
		}
		line.End = end
	} else {
		return nil, fmt.Errorf("missing required 'end' position")
	}

	// Parse stroke
	if strokeNode, found := findNode(node, "stroke"); found {
		stroke, err := parseStroke(strokeNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stroke: %w", err)
		}
		line.Stroke = stroke
	}

	// Parse layer
	if layerNode, found := findNode(node, "layer"); found {
		layer, err := getQuotedString(layerNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layer: %w", err)
		}
		line.Layer = layer
	} else {
		return nil, fmt.Errorf("missing required 'layer' field")
	}

	return line, nil
}

// parseGrCircle extracts a circle graphic element
// Expected format: (gr_circle (center x y) (end x y) (stroke ...) (fill ...) (layer "F.Cu"))
// Note: KiCad defines circles by center and a point on the circumference (end)
func parseGrCircle(node kicadsexp.Sexp) (*GrCircle, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected gr_circle list, got leaf")
	}

	circle := &GrCircle{
		Stroke: Stroke{Width: 0.15, Type: "solid"},
		Fill:   Fill{Type: "none"},
	}

	// Parse center position
	if centerNode, found := findNode(node, "center"); found {
		center, err := parsePosition(centerNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse center position: %w", err)
		}
		circle.Center = center
	} else {
		return nil, fmt.Errorf("missing required 'center' position")
	}

	// Parse end position (point on circumference)
	if endNode, found := findNode(node, "end"); found {
		end, err := parsePosition(endNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end position: %w", err)
		}
		circle.End = end
	} else {
		return nil, fmt.Errorf("missing required 'end' position")
	}

	// Parse stroke
	if strokeNode, found := findNode(node, "stroke"); found {
		stroke, err := parseStroke(strokeNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stroke: %w", err)
		}
		circle.Stroke = stroke
	}

	// Parse fill
	if fillNode, found := findNode(node, "fill"); found {
		fill, err := parseFill(fillNode)
		if err == nil {
			circle.Fill = fill
		}
	}

	// Parse layer
	if layerNode, found := findNode(node, "layer"); found {
		layer, err := getQuotedString(layerNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layer: %w", err)
		}
		circle.Layer = layer
	} else {
		return nil, fmt.Errorf("missing required 'layer' field")
	}

	return circle, nil
}

// parseGrArc extracts an arc graphic element
// Expected format: (gr_arc (start x y) (mid x y) (end x y) (stroke ...) (layer "F.Cu"))
func parseGrArc(node kicadsexp.Sexp) (*GrArc, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected gr_arc list, got leaf")
	}

	arc := &GrArc{
		Stroke: Stroke{Width: 0.15, Type: "solid"},
	}

	// Parse start position
	if startNode, found := findNode(node, "start"); found {
		start, err := parsePosition(startNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start position: %w", err)
		}
		arc.Start = start
	} else {
		return nil, fmt.Errorf("missing required 'start' position")
	}

	// Parse mid position (point on arc)
	if midNode, found := findNode(node, "mid"); found {
		mid, err := parsePosition(midNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse mid position: %w", err)
		}
		arc.Mid = mid
	} else {
		return nil, fmt.Errorf("missing required 'mid' position")
	}

	// Parse end position
	if endNode, found := findNode(node, "end"); found {
		end, err := parsePosition(endNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end position: %w", err)
		}
		arc.End = end
	} else {
		return nil, fmt.Errorf("missing required 'end' position")
	}

	// Parse stroke
	if strokeNode, found := findNode(node, "stroke"); found {
		stroke, err := parseStroke(strokeNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stroke: %w", err)
		}
		arc.Stroke = stroke
	}

	// Parse layer
	if layerNode, found := findNode(node, "layer"); found {
		layer, err := getQuotedString(layerNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layer: %w", err)
		}
		arc.Layer = layer
	} else {
		return nil, fmt.Errorf("missing required 'layer' field")
	}

	return arc, nil
}

// parseGrRect extracts a rectangle graphic element
// Expected format: (gr_rect (start x y) (end x y) (stroke ...) (fill ...) (layer "F.Cu"))
func parseGrRect(node kicadsexp.Sexp) (*GrRect, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected gr_rect list, got leaf")
	}

	rect := &GrRect{
		Stroke: Stroke{Width: 0.15, Type: "solid"},
		Fill:   Fill{Type: "none"},
	}

	// Parse start position (top-left corner)
	if startNode, found := findNode(node, "start"); found {
		start, err := parsePosition(startNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start position: %w", err)
		}
		rect.Start = start
	} else {
		return nil, fmt.Errorf("missing required 'start' position")
	}

	// Parse end position (bottom-right corner)
	if endNode, found := findNode(node, "end"); found {
		end, err := parsePosition(endNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end position: %w", err)
		}
		rect.End = end
	} else {
		return nil, fmt.Errorf("missing required 'end' position")
	}

	// Parse stroke
	if strokeNode, found := findNode(node, "stroke"); found {
		stroke, err := parseStroke(strokeNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stroke: %w", err)
		}
		rect.Stroke = stroke
	}

	// Parse fill
	if fillNode, found := findNode(node, "fill"); found {
		fill, err := parseFill(fillNode)
		if err == nil {
			rect.Fill = fill
		}
	}

	// Parse layer
	if layerNode, found := findNode(node, "layer"); found {
		layer, err := getQuotedString(layerNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layer: %w", err)
		}
		rect.Layer = layer
	} else {
		return nil, fmt.Errorf("missing required 'layer' field")
	}

	return rect, nil
}

// parseGrPoly extracts a polygon graphic element
// Expected format: (gr_poly (pts (xy x y) (xy x y) ...) (stroke ...) (fill ...) (layer "F.Cu"))
func parseGrPoly(node kicadsexp.Sexp) (*GrPoly, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected gr_poly list, got leaf")
	}

	poly := &GrPoly{
		Stroke: Stroke{Width: 0.15, Type: "solid"},
		Fill:   Fill{Type: "none"},
	}

	// Parse points
	if ptsNode, found := findNode(node, "pts"); found {
		if ptsNode.IsLeaf() {
			return nil, fmt.Errorf("expected pts list, got leaf")
		}

		// Find all (xy x y) nodes
		xyNodes := findAllNodes(ptsNode, "xy")
		if len(xyNodes) == 0 {
			return nil, fmt.Errorf("no points defined in polygon")
		}

		for _, xyNode := range xyNodes {
			x, err := getFloat(xyNode, 1)
			if err != nil {
				return nil, fmt.Errorf("failed to parse X coordinate: %w", err)
			}

			y, err := getFloat(xyNode, 2)
			if err != nil {
				return nil, fmt.Errorf("failed to parse Y coordinate: %w", err)
			}

			poly.Points = append(poly.Points, Position{X: x, Y: y})
		}
	} else {
		return nil, fmt.Errorf("missing required 'pts' field")
	}

	// Parse stroke
	if strokeNode, found := findNode(node, "stroke"); found {
		stroke, err := parseStroke(strokeNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stroke: %w", err)
		}
		poly.Stroke = stroke
	}

	// Parse fill
	if fillNode, found := findNode(node, "fill"); found {
		fill, err := parseFill(fillNode)
		if err == nil {
			poly.Fill = fill
		}
	}

	// Parse layer
	if layerNode, found := findNode(node, "layer"); found {
		layer, err := getQuotedString(layerNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layer: %w", err)
		}
		poly.Layer = layer
	} else {
		return nil, fmt.Errorf("missing required 'layer' field")
	}

	return poly, nil
}

// parseGrText extracts a text graphic element
// Expected format: (gr_text "text" (at x y angle) (layer "F.Cu") (effects ...))
func parseGrText(node kicadsexp.Sexp) (*GrText, error) {
	if node.IsLeaf() {
		return nil, fmt.Errorf("expected gr_text list, got leaf")
	}

	text := &GrText{
		Size:      Size{Width: 1.0, Height: 1.0},
		Thickness: 0.15,
	}

	// Parse text content (second element after "gr_text")
	content, err := getQuotedString(node, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to parse text content: %w", err)
	}
	text.Text = content

	// Parse position (at x y [angle])
	if atNode, found := findNode(node, "at"); found {
		pos, err := parsePosition(atNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse position: %w", err)
		}
		text.Position = pos

		// Angle is optional third parameter
		if angle, err := getFloat(atNode, 3); err == nil {
			text.Angle = Angle(angle * DecidegreesToDegrees)
		}
	} else {
		return nil, fmt.Errorf("missing required 'at' position")
	}

	// Parse layer
	if layerNode, found := findNode(node, "layer"); found {
		layer, err := getQuotedString(layerNode, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layer: %w", err)
		}
		text.Layer = layer
	} else {
		return nil, fmt.Errorf("missing required 'layer' field")
	}

	// Parse effects (contains font size, thickness, etc.)
	if effectsNode, found := findNode(node, "effects"); found {
		// Parse font node
		if fontNode, found := findNode(effectsNode, "font"); found {
			// Parse size
			if sizeNode, found := findNode(fontNode, "size"); found {
				width, err := getFloat(sizeNode, 1)
				if err == nil {
					text.Size.Width = width
				}
				height, err := getFloat(sizeNode, 2)
				if err == nil {
					text.Size.Height = height
				}
			}

			// Parse thickness
			if thickNode, found := findNode(fontNode, "thickness"); found {
				thickness, err := getFloat(thickNode, 1)
				if err == nil {
					text.Thickness = thickness
				}
			}

			// Parse bold
			if _, found := findNode(fontNode, "bold"); found {
				text.Bold = true
			}

			// Parse italic
			if _, found := findNode(fontNode, "italic"); found {
				text.Italic = true
			}
		}

		// Parse justify
		if justifyNode, found := findNode(effectsNode, "justify"); found {
			// Justify can have multiple values: left, right, top, bottom, mirror
			items := getListItems(justifyNode)
			var justifyParts []string
			for _, item := range items {
				if !item.IsLeaf() {
					continue
				}
				part := strings.Trim(item.String(), "\"")
				if part != "justify" {
					justifyParts = append(justifyParts, part)
				}
			}
			if len(justifyParts) > 0 {
				text.Justify = strings.Join(justifyParts, " ")
			}
		}
	}

	return text, nil
}

// parseGraphics extracts all graphic elements from the root node
// Finds and parses: gr_line, gr_circle, gr_arc, gr_rect, gr_poly, gr_text
func parseGraphics(root kicadsexp.Sexp) (*Graphics, error) {
	if root.IsLeaf() {
		return nil, fmt.Errorf("expected root list")
	}

	graphics := &Graphics{}

	// Parse lines
	lineNodes := findAllNodes(root, "gr_line")
	for _, lineNode := range lineNodes {
		line, err := parseGrLine(lineNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gr_line: %w", err)
		}
		graphics.Lines = append(graphics.Lines, *line)
	}

	// Parse circles
	circleNodes := findAllNodes(root, "gr_circle")
	for _, circleNode := range circleNodes {
		circle, err := parseGrCircle(circleNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gr_circle: %w", err)
		}
		graphics.Circles = append(graphics.Circles, *circle)
	}

	// Parse arcs
	arcNodes := findAllNodes(root, "gr_arc")
	for _, arcNode := range arcNodes {
		arc, err := parseGrArc(arcNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gr_arc: %w", err)
		}
		graphics.Arcs = append(graphics.Arcs, *arc)
	}

	// Parse rectangles
	rectNodes := findAllNodes(root, "gr_rect")
	for _, rectNode := range rectNodes {
		rect, err := parseGrRect(rectNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gr_rect: %w", err)
		}
		graphics.Rects = append(graphics.Rects, *rect)
	}

	// Parse polygons
	polyNodes := findAllNodes(root, "gr_poly")
	for _, polyNode := range polyNodes {
		poly, err := parseGrPoly(polyNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gr_poly: %w", err)
		}
		graphics.Polys = append(graphics.Polys, *poly)
	}

	// Parse text
	textNodes := findAllNodes(root, "gr_text")
	for _, textNode := range textNodes {
		text, err := parseGrText(textNode)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gr_text: %w", err)
		}
		graphics.Texts = append(graphics.Texts, *text)
	}

	return graphics, nil
}
