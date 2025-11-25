// Package schematic provides parsing for KiCad schematic files (.kicad_sch)
package schematic

import (
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp"
)

// Re-export shared types from sexp package for convenience
type Position = sexp.Position
type Angle = sexp.Angle
type PositionAngle = sexp.PositionAngle
type Size = sexp.Size
type Color = sexp.Color
type Stroke = sexp.Stroke
type Fill = sexp.Fill
type UUID = sexp.UUID
type Effects = sexp.Effects
type Font = sexp.Font
type Justify = sexp.Justify
type Property = sexp.Property

// Schematic represents a complete KiCad schematic file
type Schematic struct {
	Version       int            // File format version
	Generator     string         // Generator info (e.g., "eeschema")
	GeneratorVer  string         // Generator version
	UUID          UUID           // Schematic UUID
	Paper         string         // Paper size (e.g., "A4")
	TitleBlock    TitleBlock     // Title block information
	LibSymbols    []LibSymbol    // Embedded library symbols
	Symbols       []Symbol       // Symbol instances on the schematic
	Wires         []Wire         // Wire connections
	Buses         []Bus          // Bus connections
	BusEntries    []BusEntry     // Bus entry points
	Junctions     []Junction     // Wire junctions
	NoConnects    []NoConnect    // No-connect markers
	Labels        []Label        // Local labels
	GlobalLabels  []GlobalLabel  // Global labels
	HierLabels    []HierLabel    // Hierarchical labels
	Sheets        []Sheet        // Hierarchical sheet references
	SheetInstances []SheetInstance // Sheet instance paths
	Polylines     []Polyline     // Graphical polylines
	Texts         []Text         // Graphical text
	Images        []Image        // Embedded images
}

// TitleBlock contains schematic title block information
type TitleBlock struct {
	Title    string
	Date     string
	Revision string
	Company  string
	Comment1 string
	Comment2 string
	Comment3 string
	Comment4 string
}

// LibSymbol represents an embedded library symbol definition
type LibSymbol struct {
	Name        string       // Symbol name (e.g., "Device:R")
	PinNumbers  bool         // Show pin numbers
	PinNames    bool         // Show pin names
	InBom       bool         // Include in BOM
	OnBoard     bool         // Place on board
	Properties  []Property   // Symbol properties
	Pins        []Pin        // Pin definitions
	Graphics    []SymGraphic // Graphical elements
	Units       []SymbolUnit // Symbol units (for multi-unit symbols)
}

// SymbolUnit represents a unit of a multi-unit symbol
type SymbolUnit struct {
	Name     string       // Unit name
	Graphics []SymGraphic // Unit graphics
	Pins     []Pin        // Unit pins
}

// SymGraphic represents a graphical element in a symbol
type SymGraphic struct {
	Type   string     // rectangle, circle, arc, polyline, text
	Start  Position   // Start point
	End    Position   // End point
	Center Position   // Center (for circles/arcs)
	Points []Position // Points (for polylines)
	Radius float64    // Radius (for circles)
	Angles [2]float64 // Start/end angles (for arcs)
	Stroke Stroke     // Stroke style
	Fill   Fill       // Fill style
	Text   string     // Text content (for text elements)
}

// Pin represents a symbol pin
type Pin struct {
	Type      string   // Pin type (input, output, bidirectional, etc.)
	Style     string   // Pin style (line, inverted, clock, etc.)
	Position  Position // Pin position
	Angle     Angle    // Pin angle (0, 90, 180, 270)
	Length    float64  // Pin length
	Name      PinName  // Pin name
	Number    PinNum   // Pin number
	Hide      bool     // Hidden pin
	Alternate []AltPin // Alternate pin functions
}

// PinName contains pin name information
type PinName struct {
	Name    string
	Effects Effects
}

// PinNum contains pin number information
type PinNum struct {
	Number  string
	Effects Effects
}

// AltPin represents an alternate pin function
type AltPin struct {
	Name  string
	Type  string
	Style string
}

// Symbol represents a symbol instance placed on the schematic
type Symbol struct {
	LibID      string     // Library identifier (e.g., "Device:R")
	Position   Position   // Position on schematic
	Angle      Angle      // Rotation angle
	Mirror     string     // Mirror mode (x, y, or empty)
	Unit       int        // Unit number (for multi-unit symbols)
	InBom      bool       // Include in BOM
	OnBoard    bool       // Place on board
	UUID       UUID       // Instance UUID
	Properties []Property // Instance properties (Reference, Value, etc.)
	Pins       []PinRef   // Pin references
}

// PinRef represents a pin reference in a symbol instance
type PinRef struct {
	Number string // Pin number
	UUID   UUID   // Pin UUID
}

// Wire represents a wire connection
type Wire struct {
	Points []Position // Wire points (at least 2)
	Stroke Stroke     // Wire stroke style
	UUID   UUID       // Wire UUID
}

// Bus represents a bus connection
type Bus struct {
	Points []Position // Bus points
	Stroke Stroke     // Bus stroke style
	UUID   UUID       // Bus UUID
}

// BusEntry represents a bus entry point
type BusEntry struct {
	Position Position // Entry position
	Size     Size     // Entry size
	Stroke   Stroke   // Entry stroke
	UUID     UUID     // Entry UUID
}

// Junction represents a wire junction
type Junction struct {
	Position Position // Junction position
	Diameter float64  // Junction diameter
	Color    Color    // Junction color
	UUID     UUID     // Junction UUID
}

// NoConnect represents a no-connect marker
type NoConnect struct {
	Position Position // Marker position
	UUID     UUID     // Marker UUID
}

// Label represents a local wire label
type Label struct {
	Text     string   // Label text
	Position Position // Label position
	Angle    Angle    // Label rotation
	Effects  Effects  // Text effects
	UUID     UUID     // Label UUID
}

// GlobalLabel represents a global label (visible across sheets)
type GlobalLabel struct {
	Text     string   // Label text
	Shape    string   // Label shape (input, output, bidirectional, etc.)
	Position Position // Label position
	Angle    Angle    // Label rotation
	Effects  Effects  // Text effects
	UUID     UUID     // Label UUID
	Properties []Property // Label properties
}

// HierLabel represents a hierarchical label (connects to sheet pins)
type HierLabel struct {
	Text     string   // Label text
	Shape    string   // Label shape
	Position Position // Label position
	Angle    Angle    // Label rotation
	Effects  Effects  // Text effects
	UUID     UUID     // Label UUID
}

// Sheet represents a hierarchical sheet reference
type Sheet struct {
	Position   Position      // Sheet position
	Size       Size          // Sheet size
	Stroke     Stroke        // Border stroke
	Fill       Fill          // Background fill
	UUID       UUID          // Sheet UUID
	Name       SheetName     // Sheet name
	FileName   SheetFileName // Sheet file name
	Pins       []SheetPin    // Hierarchical pins
	Properties []Property    // Sheet properties
}

// SheetName contains sheet name information
type SheetName struct {
	Name    string
	Effects Effects
}

// SheetFileName contains sheet file name information
type SheetFileName struct {
	Name    string
	Effects Effects
}

// SheetPin represents a hierarchical pin on a sheet
type SheetPin struct {
	Name     string   // Pin name
	Shape    string   // Pin shape
	Position Position // Pin position
	Effects  Effects  // Text effects
	UUID     UUID     // Pin UUID
}

// SheetInstance represents a sheet instance path
type SheetInstance struct {
	Path string // Instance path
	Page string // Page number
}

// Polyline represents a graphical polyline
type Polyline struct {
	Points []Position
	Stroke Stroke
	UUID   UUID
}

// Text represents graphical text on the schematic
type Text struct {
	Text     string
	Position Position
	Angle    Angle
	Effects  Effects
	UUID     UUID
}

// Image represents an embedded image
type Image struct {
	Position Position
	Scale    float64
	UUID     UUID
	Data     string // Base64 encoded image data
}

// GetSymbol returns a symbol by reference designator
func (s *Schematic) GetSymbol(ref string) *Symbol {
	for i := range s.Symbols {
		for _, prop := range s.Symbols[i].Properties {
			if prop.Key == "Reference" && prop.Value == ref {
				return &s.Symbols[i]
			}
		}
	}
	return nil
}

// GetSymbols returns all symbols with the given library ID
func (s *Schematic) GetSymbolsByLib(libID string) []Symbol {
	var result []Symbol
	for _, sym := range s.Symbols {
		if sym.LibID == libID {
			result = append(result, sym)
		}
	}
	return result
}

// GetAllReferences returns all reference designators
func (s *Schematic) GetAllReferences() []string {
	var refs []string
	for _, sym := range s.Symbols {
		for _, prop := range sym.Properties {
			if prop.Key == "Reference" && prop.Value != "" {
				refs = append(refs, prop.Value)
				break
			}
		}
	}
	return refs
}

// GetLabels returns all label names (local + global + hierarchical)
func (s *Schematic) GetLabels() []string {
	seen := make(map[string]bool)
	var labels []string

	for _, l := range s.Labels {
		if !seen[l.Text] {
			seen[l.Text] = true
			labels = append(labels, l.Text)
		}
	}
	for _, l := range s.GlobalLabels {
		if !seen[l.Text] {
			seen[l.Text] = true
			labels = append(labels, l.Text)
		}
	}
	for _, l := range s.HierLabels {
		if !seen[l.Text] {
			seen[l.Text] = true
			labels = append(labels, l.Text)
		}
	}

	return labels
}

// GetBoundingBox calculates the bounding box of all elements in the schematic
func (s *Schematic) GetBoundingBox() sexp.BoundingBox {
	bbox := sexp.NewBoundingBox()

	// Add wire endpoints
	for _, wire := range s.Wires {
		for _, pt := range wire.Points {
			bbox.Expand(pt)
		}
	}

	// Add bus endpoints
	for _, bus := range s.Buses {
		for _, pt := range bus.Points {
			bbox.Expand(pt)
		}
	}

	// Add symbol positions
	for _, sym := range s.Symbols {
		bbox.Expand(sym.Position)
		// TODO: Add actual symbol bounds based on graphics when rendering is implemented
	}

	// Add label positions
	for _, label := range s.Labels {
		bbox.Expand(label.Position)
	}

	for _, label := range s.GlobalLabels {
		bbox.Expand(label.Position)
	}

	for _, label := range s.HierLabels {
		bbox.Expand(label.Position)
	}

	// Add sheet positions
	for _, sheet := range s.Sheets {
		bbox.Expand(sheet.Position)
		// Add opposite corner
		bbox.Expand(Position{
			X: sheet.Position.X + sheet.Size.Width,
			Y: sheet.Position.Y + sheet.Size.Height,
		})
	}

	// Add junctions
	for _, junc := range s.Junctions {
		bbox.Expand(junc.Position)
	}

	// Add no-connects
	for _, nc := range s.NoConnects {
		bbox.Expand(nc.Position)
	}

	// Add text positions
	for _, txt := range s.Texts {
		bbox.Expand(txt.Position)
	}

	return bbox
}
