package pcb

import (
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/sexp"
)

// Re-export shared types from sexp package for backward compatibility
// This allows existing code using pcb.Position, pcb.Stroke, etc. to continue working

// Coordinate conversion constants (re-exported from sexp)
const (
	NanometersToMM       = sexp.NanometersToMM
	MMToNanometers       = sexp.MMToNanometers
	DecidegreesToDegrees = sexp.DecidegreesToDegrees
	DegreesToDecidegrees = sexp.DegreesToDecidegrees
)

// Shared types (aliases to sexp package)
type Position = sexp.Position
type Angle = sexp.Angle
type PositionAngle = sexp.PositionAngle
type Size = sexp.Size
type Color = sexp.Color
type Stroke = sexp.Stroke
type Fill = sexp.Fill
type BoundingBox = sexp.BoundingBox
type UUID = sexp.UUID

// Graphics types (aliases to sexp package)
type GrLine = sexp.GrLine
type GrCircle = sexp.GrCircle
type GrArc = sexp.GrArc
type GrRect = sexp.GrRect
type GrPoly = sexp.GrPoly
type GrText = sexp.GrText
type Graphics = sexp.Graphics

// Re-export BoundingBox constructor
var NewBoundingBox = sexp.NewBoundingBox

// PCB-specific types below

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
