package pcb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp/kicadsexp"
)

// S-expression navigation helpers

// findNode searches for a child node with the given key (first symbol)
// Example: findNode(sexp, "at") finds (at 100 50) in a list
func findNode(s kicadsexp.Sexp, key string) (kicadsexp.Sexp, bool) {
	if s.IsLeaf() {
		return nil, false
	}

	// Convert to slice for safer iteration
	items := sexpToSlice(s)

	for _, item := range items {
		if item == nil {
			continue
		}

		if item.IsLeaf() {
			// Check if this leaf is our key
			if sym, ok := item.(kicadsexp.Symbol); ok && string(sym) == key {
				return item, true
			}
		} else {
			// It's a sub-list, check if it starts with our key
			subItems := sexpToSlice(item)
			if len(subItems) > 0 {
				if sym, ok := subItems[0].(kicadsexp.Symbol); ok && string(sym) == key {
					return item, true
				}
			}
		}
	}

	return nil, false
}

// findAllNodes finds all child nodes with the given key
func findAllNodes(s kicadsexp.Sexp, key string) []kicadsexp.Sexp {
	var results []kicadsexp.Sexp

	if s.IsLeaf() {
		return results
	}

	items := sexpToSlice(s)

	for _, item := range items {
		if item == nil || item.IsLeaf() {
			continue
		}

		subItems := sexpToSlice(item)
		if len(subItems) > 0 {
			if sym, ok := subItems[0].(kicadsexp.Symbol); ok && string(sym) == key {
				results = append(results, item)
			}
		}
	}

	return results
}

// getListItems returns all items in a list (excluding the first symbol/key)
// Example: getListItems((layers "F.Cu" "B.Cu")) returns ["F.Cu", "B.Cu"]
func getListItems(s kicadsexp.Sexp) []kicadsexp.Sexp {
	if s.IsLeaf() {
		return []kicadsexp.Sexp{}
	}

	allItems := sexpToSlice(s)

	// Skip first element (the key) and return the rest
	if len(allItems) <= 1 {
		return []kicadsexp.Sexp{}
	}

	return allItems[1:]
}

// Typed value extraction helpers

// getString extracts a string value at the given index in a list
// Index 0 is the key, 1 is first value, etc.
func getString(s kicadsexp.Sexp, index int) (string, error) {
	if s.IsLeaf() {
		return "", fmt.Errorf("expected list, got leaf")
	}

	// Convert to slice for easier indexing
	items := sexpToSlice(s)

	if index < 0 || index >= len(items) {
		return "", fmt.Errorf("index %d out of bounds (length %d)", index, len(items))
	}

	if sym, ok := items[index].(kicadsexp.Symbol); ok {
		return string(sym), nil
	}

	return "", fmt.Errorf("expected symbol at index %d, got %T", index, items[index])
}

// sexpToSlice converts an s-expression list to a Go slice
func sexpToSlice(s kicadsexp.Sexp) []kicadsexp.Sexp {
	var items []kicadsexp.Sexp

	if s == nil || s.IsLeaf() {
		return items
	}

	// Safely iterate using Head/Tail
	for i := 0; i < 100000; i++ { // Safety limit for large zone fills
		if s == nil {
			break
		}

		// Check if we're at the end (empty list or single element left)
		leafCount := s.LeafCount()
		if leafCount == 0 {
			break
		}

		// It's safe to call Head() now
		head := s.Head()
		if head != nil {
			items = append(items, head)
		}

		// Try to get tail
		if leafCount <= 1 {
			break
		}

		s = s.Tail()
		if s == nil || s.IsLeaf() {
			break
		}
	}

	return items
}

// getFloat extracts a float64 value at the given index
func getFloat(s kicadsexp.Sexp, index int) (float64, error) {
	str, err := getString(s, index)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float %q: %w", str, err)
	}

	return val, nil
}

// getInt extracts an int value at the given index
func getInt(s kicadsexp.Sexp, index int) (int, error) {
	str, err := getString(s, index)
	if err != nil {
		return 0, err
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int %q: %w", str, err)
	}

	return val, nil
}

// getSymbol is an alias for getString (for clarity when expecting symbol)
func getSymbol(s kicadsexp.Sexp, index int) (string, error) {
	return getString(s, index)
}

// Domain-specific extraction helpers

// getPosition extracts a Position from an (at X Y [angle]) node
// Converts from nanometers to millimeters!
func getPosition(s kicadsexp.Sexp) (PositionAngle, error) {
	// Expected format: (at X Y) or (at X Y angle)
	if s.IsLeaf() {
		return PositionAngle{}, fmt.Errorf("expected (at X Y [angle]) list")
	}

	// Verify first symbol is "at"
	key, err := getString(s, 0)
	if err != nil {
		return PositionAngle{}, err
	}
	if key != "at" {
		return PositionAngle{}, fmt.Errorf("expected 'at', got %q", key)
	}

	// Extract X and Y (in nanometers in file, convert to mm)
	xNm, err := getFloat(s, 1)
	if err != nil {
		return PositionAngle{}, fmt.Errorf("failed to parse X coordinate: %w", err)
	}

	yNm, err := getFloat(s, 2)
	if err != nil {
		return PositionAngle{}, fmt.Errorf("failed to parse Y coordinate: %w", err)
	}

	result := PositionAngle{
		Position: Position{
			X: xNm * NanometersToMM,
			Y: yNm * NanometersToMM,
		},
		Angle: 0,
	}

	// Try to extract optional angle (in decidegrees in file, convert to degrees)
	angleDecideg, err := getFloat(s, 3)
	if err == nil {
		result.Angle = Angle(angleDecideg * DecidegreesToDegrees)
	}
	// No error if angle missing (it's optional)

	return result, nil
}

// getPositionXY extracts just X,Y coordinates (no angle)
// Used for (start X Y), (end X Y), (center X Y), etc.
func getPositionXY(s kicadsexp.Sexp) (Position, error) {
	if s.IsLeaf() {
		return Position{}, fmt.Errorf("expected position list")
	}

	// Format: (keyword X Y)
	xNm, err := getFloat(s, 1)
	if err != nil {
		return Position{}, fmt.Errorf("failed to parse X: %w", err)
	}

	yNm, err := getFloat(s, 2)
	if err != nil {
		return Position{}, fmt.Errorf("failed to parse Y: %w", err)
	}

	return Position{
		X: xNm * NanometersToMM,
		Y: yNm * NanometersToMM,
	}, nil
}

// getAngle extracts an angle value and converts from decidegrees to degrees
func getAngle(s kicadsexp.Sexp, index int) (Angle, error) {
	decideg, err := getFloat(s, index)
	if err != nil {
		return 0, err
	}

	return Angle(decideg * DecidegreesToDegrees), nil
}

// getStroke extracts stroke properties from (stroke ...) node
// Format: (stroke (width W) (type solid|dash|dot) [(color R G B A)])
func getStroke(s kicadsexp.Sexp) (Stroke, error) {
	stroke := Stroke{
		Width: 0.15, // Default width
		Type:  "solid",
		Color: Color{R: 1, G: 1, B: 1, A: 1}, // Default white
	}

	if s.IsLeaf() {
		return stroke, fmt.Errorf("expected (stroke ...) list")
	}

	// Find width
	if widthNode, ok := findNode(s, "width"); ok {
		width, err := getFloat(widthNode, 1)
		if err == nil {
			stroke.Width = width * NanometersToMM
		}
	}

	// Find type
	if typeNode, ok := findNode(s, "type"); ok {
		strokeType, err := getString(typeNode, 1)
		if err == nil {
			stroke.Type = strokeType
		}
	}

	// Find color (optional)
	if colorNode, ok := findNode(s, "color"); ok {
		color, err := getColor(colorNode)
		if err == nil {
			stroke.Color = color
		}
	}

	return stroke, nil
}

// getFill extracts fill properties from (fill ...) node
// Format: (fill (type none|solid) [(color R G B A)])
func getFill(s kicadsexp.Sexp) (Fill, error) {
	fill := Fill{
		Type:  "none",
		Color: Color{R: 0, G: 0, B: 0, A: 1},
	}

	if s.IsLeaf() {
		return fill, fmt.Errorf("expected (fill ...) list")
	}

	// Find type
	if typeNode, ok := findNode(s, "type"); ok {
		fillType, err := getString(typeNode, 1)
		if err == nil {
			fill.Type = fillType
		}
	}

	// Find color (optional)
	if colorNode, ok := findNode(s, "color"); ok {
		color, err := getColor(colorNode)
		if err == nil {
			fill.Color = color
		}
	}

	return fill, nil
}

// getColor extracts RGBA color from (color R G B [A]) node
// Values are 0-255 in file, we convert to 0.0-1.0
func getColor(s kicadsexp.Sexp) (Color, error) {
	color := Color{A: 1.0} // Default alpha

	if s.IsLeaf() {
		return color, fmt.Errorf("expected (color ...) list")
	}

	// Get R, G, B (required)
	r, err := getFloat(s, 1)
	if err != nil {
		return color, fmt.Errorf("failed to parse R: %w", err)
	}

	g, err := getFloat(s, 2)
	if err != nil {
		return color, fmt.Errorf("failed to parse G: %w", err)
	}

	b, err := getFloat(s, 3)
	if err != nil {
		return color, fmt.Errorf("failed to parse B: %w", err)
	}

	color.R = r / 255.0
	color.G = g / 255.0
	color.B = b / 255.0

	// Get A (optional)
	a, err := getFloat(s, 4)
	if err == nil {
		color.A = a / 255.0
	}

	return color, nil
}

// getLayers extracts layer specifications
// Format: (layer "F.Cu") or (layers "F.Cu" "B.Cu" "*.Mask")
func getLayers(s kicadsexp.Sexp) (LayerSet, error) {
	if s.IsLeaf() {
		return nil, fmt.Errorf("expected layer list")
	}

	// Get the keyword (layer or layers)
	keyword, err := getString(s, 0)
	if err != nil {
		return nil, err
	}

	var layers LayerSet

	if keyword == "layer" {
		// Single layer: (layer "F.Cu")
		layer, err := getString(s, 1)
		if err != nil {
			return nil, err
		}
		layers = LayerSet{layer}
	} else if keyword == "layers" {
		// Multiple layers: (layers "F.Cu" "B.Cu")
		items := getListItems(s)
		for _, item := range items {
			if sym, ok := item.(kicadsexp.Symbol); ok {
				layers = append(layers, string(sym))
			}
		}
	} else {
		return nil, fmt.Errorf("expected 'layer' or 'layers', got %q", keyword)
	}

	return layers, nil
}

// getQuotedString extracts a quoted string value
// KiCad strings are often quoted. The sexp library splits quoted strings with spaces
// into multiple tokens, so we need to join them and remove quotes.
// Example: (title "Example Board") becomes ["title", "\"Example", "Board\""]
func getQuotedString(s kicadsexp.Sexp, index int) (string, error) {
	items := sexpToSlice(s)

	if index < 0 || index >= len(items) {
		return "", fmt.Errorf("index %d out of bounds (length %d)", index, len(items))
	}

	// Get the first part
	firstSym, ok := items[index].(kicadsexp.Symbol)
	if !ok {
		return "", fmt.Errorf("expected symbol at index %d", index)
	}

	first := string(firstSym)

	// If it starts with a quote, we need to collect until we find the closing quote
	if strings.HasPrefix(first, "\"") {
		var parts []string
		parts = append(parts, strings.TrimPrefix(first, "\""))

		// If it also ends with quote, we're done
		if strings.HasSuffix(first, "\"") {
			return strings.TrimSuffix(parts[0], "\""), nil
		}

		// Otherwise, collect remaining parts until we find closing quote
		for i := index + 1; i < len(items); i++ {
			if sym, ok := items[i].(kicadsexp.Symbol); ok {
				part := string(sym)
				if strings.HasSuffix(part, "\"") {
					parts = append(parts, strings.TrimSuffix(part, "\""))
					return strings.Join(parts, " "), nil
				}
				parts = append(parts, part)
			}
		}

		// Unclosed quote - return what we have
		return strings.Join(parts, " "), nil
	}

	// No quotes, return as-is
	return first, nil
}

// hasSymbol checks if a list contains a specific symbol
func hasSymbol(s kicadsexp.Sexp, symbol string) bool {
	if s.IsLeaf() {
		return false
	}

	items := sexpToSlice(s)
	for _, item := range items {
		if sym, ok := item.(kicadsexp.Symbol); ok && string(sym) == symbol {
			return true
		}
	}

	return false
}

// getNodeName returns the first symbol of a list (the node type/name)
func getNodeName(s kicadsexp.Sexp) (string, error) {
	if s.IsLeaf() {
		if sym, ok := s.(kicadsexp.Symbol); ok {
			return string(sym), nil
		}
		return "", fmt.Errorf("expected symbol leaf")
	}

	head := s.Head()
	if sym, ok := head.(kicadsexp.Symbol); ok {
		return string(sym), nil
	}

	return "", fmt.Errorf("expected symbol at head of list")
}
