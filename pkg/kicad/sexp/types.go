// Package sexp provides shared S-expression parsing infrastructure for KiCad files.
// This package contains types and utilities common to both PCB and schematic parsers.
package sexp

// Coordinate conversion constants
// KiCad internally stores coordinates in nanometers, but we work in millimeters
const (
	NanometersToMM       = 1e-6  // Convert nm to mm (multiply by this)
	MMToNanometers       = 1e6   // Convert mm to nm (multiply by this)
	DecidegreesToDegrees = 0.1   // KiCad angles are in decidegrees (tenths of a degree)
	DegreesToDecidegrees = 10.0  // Convert degrees to decidegrees
)

// Position represents a 2D coordinate in the KiCad coordinate system
// NOTE: KiCad stores coordinates internally in nanometers, but this struct
// stores them in millimeters for easier use. Parser functions handle conversion.
type Position struct {
	X float64 // X coordinate in mm (converted from nanometers during parsing)
	Y float64 // Y coordinate in mm (converted from nanometers during parsing)
}

// Angle represents rotation in degrees
// NOTE: KiCad stores angles in decidegrees (tenths of degrees), but this type
// stores them in degrees for easier use. Parser functions handle conversion.
type Angle float64

// PositionAngle combines position with rotation
type PositionAngle struct {
	Position
	Angle Angle
}

// Size represents dimensions
type Size struct {
	Width  float64 // Width in mm
	Height float64 // Height in mm
}

// Color represents RGBA color
type Color struct {
	R, G, B, A float64 // Color components (0.0-1.0)
}

// Stroke defines line/outline appearance
type Stroke struct {
	Width float64 // Line width in mm
	Type  string  // Line type (solid, dash, dot, etc.)
	Color Color   // Line color
}

// Fill defines area fill
type Fill struct {
	Type  string // Fill type (solid, none, etc.)
	Color Color  // Fill color
}

// BoundingBox represents a rectangular boundary
type BoundingBox struct {
	Min Position // Minimum (top-left) corner
	Max Position // Maximum (bottom-right) corner
}

// Intersects checks if two bounding boxes intersect
func (bb BoundingBox) Intersects(other BoundingBox) bool {
	return bb.Min.X <= other.Max.X && bb.Max.X >= other.Min.X &&
		bb.Min.Y <= other.Max.Y && bb.Max.Y >= other.Min.Y
}

// Contains checks if a position is within the bounding box
func (bb BoundingBox) Contains(pos Position) bool {
	return pos.X >= bb.Min.X && pos.X <= bb.Max.X &&
		pos.Y >= bb.Min.Y && pos.Y <= bb.Max.Y
}

// NewBoundingBox creates an empty bounding box
func NewBoundingBox() BoundingBox {
	return BoundingBox{
		Min: Position{X: 1e9, Y: 1e9},   // Start with very large values
		Max: Position{X: -1e9, Y: -1e9}, // Start with very small values
	}
}

// IsEmpty checks if the bounding box is empty
func (bb BoundingBox) IsEmpty() bool {
	return bb.Min.X > bb.Max.X || bb.Min.Y > bb.Max.Y
}

// Expand expands the bounding box to include a position
func (bb *BoundingBox) Expand(pos Position) {
	if pos.X < bb.Min.X {
		bb.Min.X = pos.X
	}
	if pos.Y < bb.Min.Y {
		bb.Min.Y = pos.Y
	}
	if pos.X > bb.Max.X {
		bb.Max.X = pos.X
	}
	if pos.Y > bb.Max.Y {
		bb.Max.Y = pos.Y
	}
}

// ExpandBox expands to include another bounding box
func (bb *BoundingBox) ExpandBox(other BoundingBox) {
	if !other.IsEmpty() {
		bb.Expand(other.Min)
		bb.Expand(other.Max)
	}
}

// Width returns the width of the bounding box
func (bb BoundingBox) Width() float64 {
	return bb.Max.X - bb.Min.X
}

// Height returns the height of the bounding box
func (bb BoundingBox) Height() float64 {
	return bb.Max.Y - bb.Min.Y
}

// Center returns the center point of the bounding box
func (bb BoundingBox) Center() Position {
	return Position{
		X: (bb.Min.X + bb.Max.X) / 2.0,
		Y: (bb.Min.Y + bb.Max.Y) / 2.0,
	}
}

// UUID represents a unique identifier (used in KiCad v6+ files)
type UUID string

// Effects represents text effects (font, justification, etc.)
type Effects struct {
	Font    Font
	Justify Justify
	Hide    bool
}

// Font represents font properties
type Font struct {
	Face      string  // Font face name (optional)
	Size      Size    // Font size
	Thickness float64 // Line thickness for stroke fonts
	Bold      bool
	Italic    bool
}

// Justify represents text justification
type Justify struct {
	Horizontal string // left, center, right
	Vertical   string // top, center, bottom
	Mirror     bool
}

// Property represents a key-value property (used in symbols, footprints, etc.)
type Property struct {
	Key      string
	Value    string
	ID       int
	Position PositionAngle
	Effects  Effects
}

// GrLine represents a line graphic element
type GrLine struct {
	Start  Position // Start position
	End    Position // End position
	Stroke Stroke   // Line stroke
	Layer  string   // Layer name
}

// GrCircle represents a circle graphic element
// In KiCad, circles are defined by center and a point on the circumference
type GrCircle struct {
	Center Position // Center position
	End    Position // Point on circumference (defines radius)
	Stroke Stroke   // Circle stroke
	Fill   Fill     // Circle fill
	Layer  string   // Layer name
}

// GrArc represents an arc graphic element
// Arcs are defined by three points: start, mid (on arc), and end
type GrArc struct {
	Start  Position // Start position
	Mid    Position // Mid position (point on arc)
	End    Position // End position
	Stroke Stroke   // Arc stroke
	Layer  string   // Layer name
}

// GrRect represents a rectangle graphic element
type GrRect struct {
	Start  Position // Top-left corner
	End    Position // Bottom-right corner
	Stroke Stroke   // Rectangle stroke
	Fill   Fill     // Rectangle fill
	Layer  string   // Layer name
}

// GrPoly represents a polygon graphic element
type GrPoly struct {
	Points []Position // Polygon vertices
	Stroke Stroke     // Polygon stroke
	Fill   Fill       // Polygon fill
	Layer  string     // Layer name
}

// GrText represents a text graphic element
type GrText struct {
	Text      string   // Text content
	Position  Position // Position
	Angle     Angle    // Rotation angle
	Layer     string   // Layer name
	Size      Size     // Font size
	Thickness float64  // Text thickness
	Bold      bool     // Bold flag
	Italic    bool     // Italic flag
	Justify   string   // Justification (e.g., "left", "right", "mirror")
}

// Graphics contains all graphic elements
type Graphics struct {
	Lines   []GrLine   // Line graphics
	Circles []GrCircle // Circle graphics
	Arcs    []GrArc    // Arc graphics
	Rects   []GrRect   // Rectangle graphics
	Polys   []GrPoly   // Polygon graphics
	Texts   []GrText   // Text graphics
}
