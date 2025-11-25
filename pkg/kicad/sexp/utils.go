package sexp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp/kicadsexp"
)

// S-expression navigation helpers

// FindNode searches for a child node with the given key (first symbol)
// Example: FindNode(sexp, "at") finds (at 100 50) in a list
func FindNode(s kicadsexp.Sexp, key string) (kicadsexp.Sexp, bool) {
	if s.IsLeaf() {
		return nil, false
	}

	// Convert to slice for safer iteration
	items := SexpToSlice(s)

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
			subItems := SexpToSlice(item)
			if len(subItems) > 0 {
				if sym, ok := subItems[0].(kicadsexp.Symbol); ok && string(sym) == key {
					return item, true
				}
			}
		}
	}

	return nil, false
}

// FindAllNodes finds all child nodes with the given key
func FindAllNodes(s kicadsexp.Sexp, key string) []kicadsexp.Sexp {
	var results []kicadsexp.Sexp

	if s.IsLeaf() {
		return results
	}

	items := SexpToSlice(s)

	for _, item := range items {
		if item == nil || item.IsLeaf() {
			continue
		}

		subItems := SexpToSlice(item)
		if len(subItems) > 0 {
			if sym, ok := subItems[0].(kicadsexp.Symbol); ok && string(sym) == key {
				results = append(results, item)
			}
		}
	}

	return results
}

// GetListItems returns all items in a list (excluding the first symbol/key)
// Example: GetListItems((layers "F.Cu" "B.Cu")) returns ["F.Cu", "B.Cu"]
func GetListItems(s kicadsexp.Sexp) []kicadsexp.Sexp {
	if s.IsLeaf() {
		return []kicadsexp.Sexp{}
	}

	allItems := SexpToSlice(s)

	// Skip first element (the key) and return the rest
	if len(allItems) <= 1 {
		return []kicadsexp.Sexp{}
	}

	return allItems[1:]
}

// Typed value extraction helpers

// GetString extracts a string value at the given index in a list
// Index 0 is the key, 1 is first value, etc.
func GetString(s kicadsexp.Sexp, index int) (string, error) {
	if s.IsLeaf() {
		return "", fmt.Errorf("expected list, got leaf")
	}

	// Convert to slice for easier indexing
	items := SexpToSlice(s)

	if index < 0 || index >= len(items) {
		return "", fmt.Errorf("index %d out of bounds (length %d)", index, len(items))
	}

	if sym, ok := items[index].(kicadsexp.Symbol); ok {
		return string(sym), nil
	}

	return "", fmt.Errorf("expected symbol at index %d, got %T", index, items[index])
}

// SexpToSlice converts an s-expression list to a Go slice
func SexpToSlice(s kicadsexp.Sexp) []kicadsexp.Sexp {
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

// GetFloat extracts a float64 value at the given index
func GetFloat(s kicadsexp.Sexp, index int) (float64, error) {
	str, err := GetString(s, index)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float %q: %w", str, err)
	}

	return val, nil
}

// GetInt extracts an int value at the given index
func GetInt(s kicadsexp.Sexp, index int) (int, error) {
	str, err := GetString(s, index)
	if err != nil {
		return 0, err
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int %q: %w", str, err)
	}

	return val, nil
}

// GetSymbol is an alias for GetString (for clarity when expecting symbol)
func GetSymbol(s kicadsexp.Sexp, index int) (string, error) {
	return GetString(s, index)
}

// Domain-specific extraction helpers

// GetPosition extracts a Position from an (at X Y [angle]) node
// Converts from nanometers to millimeters!
func GetPosition(s kicadsexp.Sexp) (PositionAngle, error) {
	// Expected format: (at X Y) or (at X Y angle)
	if s.IsLeaf() {
		return PositionAngle{}, fmt.Errorf("expected (at X Y [angle]) list")
	}

	// Verify first symbol is "at"
	key, err := GetString(s, 0)
	if err != nil {
		return PositionAngle{}, err
	}
	if key != "at" {
		return PositionAngle{}, fmt.Errorf("expected 'at', got %q", key)
	}

	// Extract X and Y (in nanometers in file, convert to mm)
	xNm, err := GetFloat(s, 1)
	if err != nil {
		return PositionAngle{}, fmt.Errorf("failed to parse X coordinate: %w", err)
	}

	yNm, err := GetFloat(s, 2)
	if err != nil {
		return PositionAngle{}, fmt.Errorf("failed to parse Y coordinate: %w", err)
	}

	result := PositionAngle{
		Position: Position{
			X: xNm, // Values are already in mm in schematic files
			Y: yNm,
		},
		Angle: 0,
	}

	// Try to extract optional angle (in decidegrees in file, convert to degrees)
	angleDecideg, err := GetFloat(s, 3)
	if err == nil {
		result.Angle = Angle(angleDecideg * DecidegreesToDegrees)
	}
	// No error if angle missing (it's optional)

	return result, nil
}

// GetPositionXY extracts just X,Y coordinates (no angle)
// Used for (start X Y), (end X Y), (center X Y), etc.
func GetPositionXY(s kicadsexp.Sexp) (Position, error) {
	if s.IsLeaf() {
		return Position{}, fmt.Errorf("expected position list")
	}

	// Format: (keyword X Y)
	xNm, err := GetFloat(s, 1)
	if err != nil {
		return Position{}, fmt.Errorf("failed to parse X: %w", err)
	}

	yNm, err := GetFloat(s, 2)
	if err != nil {
		return Position{}, fmt.Errorf("failed to parse Y: %w", err)
	}

	return Position{
		X: xNm * NanometersToMM,
		Y: yNm * NanometersToMM,
	}, nil
}

// GetAngle extracts an angle value and converts from decidegrees to degrees
func GetAngle(s kicadsexp.Sexp, index int) (Angle, error) {
	decideg, err := GetFloat(s, index)
	if err != nil {
		return 0, err
	}

	return Angle(decideg * DecidegreesToDegrees), nil
}

// GetStroke extracts stroke properties from (stroke ...) node
// Format: (stroke (width W) (type solid|dash|dot) [(color R G B A)])
func GetStroke(s kicadsexp.Sexp) (Stroke, error) {
	stroke := Stroke{
		Width: 0.15, // Default width
		Type:  "solid",
		Color: Color{R: 1, G: 1, B: 1, A: 1}, // Default white
	}

	if s.IsLeaf() {
		return stroke, fmt.Errorf("expected (stroke ...) list")
	}

	// Find width
	if widthNode, ok := FindNode(s, "width"); ok {
		width, err := GetFloat(widthNode, 1)
		if err == nil {
			stroke.Width = width * NanometersToMM
		}
	}

	// Find type
	if typeNode, ok := FindNode(s, "type"); ok {
		strokeType, err := GetString(typeNode, 1)
		if err == nil {
			stroke.Type = strokeType
		}
	}

	// Find color (optional)
	if colorNode, ok := FindNode(s, "color"); ok {
		color, err := GetColor(colorNode)
		if err == nil {
			stroke.Color = color
		}
	}

	return stroke, nil
}

// GetFill extracts fill properties from (fill ...) node
// Format: (fill (type none|solid) [(color R G B A)])
func GetFill(s kicadsexp.Sexp) (Fill, error) {
	fill := Fill{
		Type:  "none",
		Color: Color{R: 0, G: 0, B: 0, A: 1},
	}

	if s.IsLeaf() {
		return fill, fmt.Errorf("expected (fill ...) list")
	}

	// Find type
	if typeNode, ok := FindNode(s, "type"); ok {
		fillType, err := GetString(typeNode, 1)
		if err == nil {
			fill.Type = fillType
		}
	}

	// Find color (optional)
	if colorNode, ok := FindNode(s, "color"); ok {
		color, err := GetColor(colorNode)
		if err == nil {
			fill.Color = color
		}
	}

	return fill, nil
}

// GetColor extracts RGBA color from (color R G B [A]) node
// Values are 0-255 in file, we convert to 0.0-1.0
func GetColor(s kicadsexp.Sexp) (Color, error) {
	color := Color{A: 1.0} // Default alpha

	if s.IsLeaf() {
		return color, fmt.Errorf("expected (color ...) list")
	}

	// Get R, G, B (required)
	r, err := GetFloat(s, 1)
	if err != nil {
		return color, fmt.Errorf("failed to parse R: %w", err)
	}

	g, err := GetFloat(s, 2)
	if err != nil {
		return color, fmt.Errorf("failed to parse G: %w", err)
	}

	b, err := GetFloat(s, 3)
	if err != nil {
		return color, fmt.Errorf("failed to parse B: %w", err)
	}

	color.R = r / 255.0
	color.G = g / 255.0
	color.B = b / 255.0

	// Get A (optional)
	a, err := GetFloat(s, 4)
	if err == nil {
		color.A = a / 255.0
	}

	return color, nil
}

// GetQuotedString extracts a quoted string value
// KiCad strings are often quoted. The sexp library splits quoted strings with spaces
// into multiple tokens, so we need to join them and remove quotes.
// Example: (title "Example Board") becomes ["title", "\"Example", "Board\""]
func GetQuotedString(s kicadsexp.Sexp, index int) (string, error) {
	items := SexpToSlice(s)

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

// HasSymbol checks if a list contains a specific symbol
func HasSymbol(s kicadsexp.Sexp, symbol string) bool {
	if s.IsLeaf() {
		return false
	}

	items := SexpToSlice(s)
	for _, item := range items {
		if sym, ok := item.(kicadsexp.Symbol); ok && string(sym) == symbol {
			return true
		}
	}

	return false
}

// GetNodeName returns the first symbol of a list (the node type/name)
func GetNodeName(s kicadsexp.Sexp) (string, error) {
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

// GetUUID extracts a UUID from a (uuid "...") node
func GetUUID(s kicadsexp.Sexp) (UUID, error) {
	if s.IsLeaf() {
		return "", fmt.Errorf("expected (uuid ...) list")
	}

	key, err := GetString(s, 0)
	if err != nil || key != "uuid" {
		return "", fmt.Errorf("expected 'uuid' node")
	}

	uuidStr, err := GetQuotedString(s, 1)
	if err != nil {
		// Try unquoted
		uuidStr, err = GetString(s, 1)
		if err != nil {
			return "", err
		}
	}

	return UUID(uuidStr), nil
}

// GetEffects extracts text effects from an (effects ...) node
func GetEffects(s kicadsexp.Sexp) (Effects, error) {
	effects := Effects{}

	if s.IsLeaf() {
		return effects, fmt.Errorf("expected (effects ...) list")
	}

	// Parse font
	if fontNode, ok := FindNode(s, "font"); ok {
		font, err := GetFont(fontNode)
		if err == nil {
			effects.Font = font
		}
	}

	// Parse justify
	if justifyNode, ok := FindNode(s, "justify"); ok {
		justify, err := GetJustify(justifyNode)
		if err == nil {
			effects.Justify = justify
		}
	}

	// Check for hide
	effects.Hide = HasSymbol(s, "hide")

	return effects, nil
}

// GetFont extracts font properties from a (font ...) node
func GetFont(s kicadsexp.Sexp) (Font, error) {
	font := Font{}

	if s.IsLeaf() {
		return font, fmt.Errorf("expected (font ...) list")
	}

	// Parse size
	if sizeNode, ok := FindNode(s, "size"); ok {
		w, _ := GetFloat(sizeNode, 1)
		h, _ := GetFloat(sizeNode, 2)
		font.Size = Size{Width: w, Height: h} // Values are already in mm in schematic files
	}

	// Parse thickness
	if thicknessNode, ok := FindNode(s, "thickness"); ok {
		t, _ := GetFloat(thicknessNode, 1)
		font.Thickness = t * NanometersToMM
	}

	// Check for bold/italic
	font.Bold = HasSymbol(s, "bold")
	font.Italic = HasSymbol(s, "italic")

	// Parse face (optional)
	if faceNode, ok := FindNode(s, "face"); ok {
		face, _ := GetQuotedString(faceNode, 1)
		font.Face = face
	}

	return font, nil
}

// GetJustify extracts justification from a (justify ...) node
func GetJustify(s kicadsexp.Sexp) (Justify, error) {
	justify := Justify{
		Horizontal: "center",
		Vertical:   "center",
	}

	if s.IsLeaf() {
		return justify, nil
	}

	items := GetListItems(s)
	for _, item := range items {
		if sym, ok := item.(kicadsexp.Symbol); ok {
			str := string(sym)
			switch str {
			case "left":
				justify.Horizontal = "left"
			case "right":
				justify.Horizontal = "right"
			case "top":
				justify.Vertical = "top"
			case "bottom":
				justify.Vertical = "bottom"
			case "mirror":
				justify.Mirror = true
			}
		}
	}

	return justify, nil
}

// GetProperty extracts a property from a (property ...) node
func GetProperty(s kicadsexp.Sexp) (Property, error) {
	prop := Property{}

	if s.IsLeaf() {
		return prop, fmt.Errorf("expected (property ...) list")
	}

	// Format: (property "key" "value" (at X Y angle) (effects ...))
	key, err := GetQuotedString(s, 1)
	if err != nil {
		return prop, fmt.Errorf("failed to parse property key: %w", err)
	}
	prop.Key = key

	value, err := GetQuotedString(s, 2)
	if err != nil {
		value = "" // Value can be empty
	}
	prop.Value = value

	// Parse ID if present
	if idNode, ok := FindNode(s, "id"); ok {
		id, _ := GetInt(idNode, 1)
		prop.ID = id
	}

	// Parse position if present
	if atNode, ok := FindNode(s, "at"); ok {
		pos, err := GetPosition(atNode)
		if err == nil {
			prop.Position = pos
		}
	}

	// Parse effects if present
	if effectsNode, ok := FindNode(s, "effects"); ok {
		effects, err := GetEffects(effectsNode)
		if err == nil {
			prop.Effects = effects
		}
	}

	return prop, nil
}
