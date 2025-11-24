package parser

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

// Layer represents a PCB layer
type Layer struct {
	Number int    // Layer number (ordinal)
	Name   string // Layer name (e.g., "F.Cu", "B.Cu", "F.SilkS")
	Type   string // Layer type (e.g., "signal", "user")
}

// Net represents an electrical net
type Net struct {
	Number int    // Net number (ordinal)
	Name   string // Net name
}

// Color represents RGBA color
type Color struct {
	R, G, B, A float64 // Color components (0.0-1.0)
}

// Stroke defines line/outline appearance
type Stroke struct {
	Width float64   // Line width in mm
	Type  string    // Line type (solid, dash, dot, etc.)
	Color Color     // Line color
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

// LayerSet represents a set of layers
type LayerSet []string

// LayerMap provides efficient lookup of layers by number or name
type LayerMap struct {
	byNumber map[int]*Layer
	byName   map[string]*Layer
}

// NewLayerMap creates a LayerMap from a slice of layers
func NewLayerMap(layers []Layer) *LayerMap {
	lm := &LayerMap{
		byNumber: make(map[int]*Layer),
		byName:   make(map[string]*Layer),
	}

	for i := range layers {
		layer := &layers[i]
		lm.byNumber[layer.Number] = layer
		lm.byName[layer.Name] = layer
	}

	return lm
}

// GetByName retrieves a layer by its name (e.g., "F.Cu")
func (lm *LayerMap) GetByName(name string) (*Layer, bool) {
	layer, ok := lm.byName[name]
	return layer, ok
}

// GetByNumber retrieves a layer by its number
func (lm *LayerMap) GetByNumber(num int) (*Layer, bool) {
	layer, ok := lm.byNumber[num]
	return layer, ok
}

// IsCopperLayer checks if a layer is a copper layer
func (lm *LayerMap) IsCopperLayer(name string) bool {
	layer, ok := lm.byName[name]
	if !ok {
		return false
	}
	return layer.Type == "signal" || layer.Type == "power" || layer.Type == "mixed"
}

// NetMap provides efficient lookup of nets by number or name
type NetMap struct {
	byNumber map[int]*Net
	byName   map[string]*Net
}

// NewNetMap creates a NetMap from a slice of nets
func NewNetMap(nets []Net) *NetMap {
	nm := &NetMap{
		byNumber: make(map[int]*Net),
		byName:   make(map[string]*Net),
	}

	for i := range nets {
		net := &nets[i]
		nm.byNumber[net.Number] = net
		// Only index non-empty names
		if net.Name != "" {
			nm.byName[net.Name] = net
		}
	}

	return nm
}

// GetByName retrieves a net by its name (e.g., "GND", "+5V")
func (nm *NetMap) GetByName(name string) (*Net, bool) {
	net, ok := nm.byName[name]
	return net, ok
}

// GetByNumber retrieves a net by its number
func (nm *NetMap) GetByNumber(num int) (*Net, bool) {
	net, ok := nm.byNumber[num]
	return net, ok
}

// IsUnconnected checks if a net number represents an unconnected net
// In KiCad, net 0 is reserved for unconnected pins
func (nm *NetMap) IsUnconnected(num int) bool {
	return num == 0
}

// Graphic element types

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

// Graphics contains all graphic elements on a board
type Graphics struct {
	Lines   []GrLine   // Line graphics
	Circles []GrCircle // Circle graphics
	Arcs    []GrArc    // Arc graphics
	Rects   []GrRect   // Rectangle graphics
	Polys   []GrPoly   // Polygon graphics
	Texts   []GrText   // Text graphics
}
