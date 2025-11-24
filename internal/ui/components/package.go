package components

import (
	"fmt"
	"image/color"
)

// PackagePin describes metadata shown next to a rendered package pad (or ball).
type PackagePin struct {
	Number int
	Name   string
	State  PinState
}

// PinState describes the logical/electrical condition of a pin for rendering.
type PinState string

const (
	PinStateUnknown PinState = "unknown"
	PinStateHigh    PinState = "high"
	PinStateLow     PinState = "low"
	PinStateHighZ   PinState = "hi-z"
	PinStatePower   PinState = "power"
)

var pinStateColors = map[PinState]color.NRGBA{
	PinStateHigh:  {R: 220, G: 68, B: 68, A: 255},
	PinStateLow:   {R: 66, G: 135, B: 245, A: 255},
	PinStateHighZ: {R: 235, G: 138, B: 52, A: 255},
	PinStatePower: {R: 160, G: 160, B: 160, A: 255},
	PinStateUnknown: {
		R: 200, G: 200, B: 210, A: 255,
	},
}

func colorForState(state PinState) color.NRGBA {
	if c, ok := pinStateColors[state]; ok {
		return c
	}
	return pinStateColors[PinStateUnknown]
}

// contrastColor returns white or black depending on background luminance
func contrastColor(bg color.NRGBA) color.NRGBA {
	// Calculate relative luminance (ITU-R BT.709)
	r := float32(bg.R) / 255.0
	g := float32(bg.G) / 255.0
	b := float32(bg.B) / 255.0
	
	luminance := 0.2126*r + 0.7152*g + 0.0722*b
	
	// Use white text on dark backgrounds, black on light
	if luminance < 0.5 {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255} // White
	}
	return color.NRGBA{R: 0, G: 0, B: 0, A: 255} // Black
}

// DefaultPins creates a sequential set of placeholder pins for the provided
// count.
func DefaultPins(count int) []PackagePin {
	pins := make([]PackagePin, count)
	for i := 0; i < count; i++ {
		pins[i] = PackagePin{
			Number: i + 1,
			Name:   fmt.Sprintf("PIN%d", i+1),
		}
	}
	return pins
}

// EnsurePins pads or truncates the slice to match the requested count, cloning
// any provided entries and filling the remainder with sequential defaults.
func EnsurePins(pins []PackagePin, count int) []PackagePin {
	if count <= 0 {
		return nil
	}

	if len(pins) >= count {
		return append([]PackagePin(nil), pins[:count]...)
	}

	normalized := append([]PackagePin(nil), pins...)
	defaults := DefaultPins(count)
	for i := len(normalized); i < count; i++ {
		normalized = append(normalized, defaults[i])
	}
	return normalized
}

// RenderOptions control scaling and label visibility when rendering packages.
type RenderOptions struct {
	Scale      float32
	ShowLabels bool
}

// PackageRender captures the rendered canvas object and metadata for pin pads.
type PackageRender struct {
	Size       Vec
	Title      string
	Rectangles []RectShape
	Circles    []CircleShape
	Labels     []LabelShape
	Pads       []PadArea
}

// Vec represents a 2D vector or size in logical pixels.
type Vec struct {
	X float32
	Y float32
}

// RectShape describes a filled rectangle.
type RectShape struct {
	Position     Vec
	Size         Vec
	Fill         color.NRGBA
	Stroke       color.NRGBA
	StrokeWidth  float32
	CornerRadius float32
}

// CircleShape describes a circle primitive.
type CircleShape struct {
	Center      Vec
	Radius      float32
	Fill        color.NRGBA
	Stroke      color.NRGBA
	StrokeWidth float32
}

// LabelAlignment indicates text alignment relative to Position.
type LabelAlignment int

const (
	AlignStart LabelAlignment = iota
	AlignCenter
	AlignEnd
)

// LabelShape represents a text label.
type LabelShape struct {
	Position Vec
	Text     string
	Color    color.NRGBA
	Size     float32
	Align    LabelAlignment
}

// PadArea describes the bounds and pin metadata for a specific pad.
type PadArea struct {
	Index    int
	Pin      PackagePin
	Position Vec
	Size     Vec
	Label    string
	PinName  string // Logical pin name from BSDL (e.g., "PA0", "RESET")
}

func normalizeOptions(opts *RenderOptions) RenderOptions {
	cfg := RenderOptions{
		Scale:      1.0,
		ShowLabels: true,
	}
	if opts == nil {
		return cfg
	}
	if opts.Scale > 0 {
		cfg.Scale = opts.Scale
	}
	if !opts.ShowLabels {
		cfg.ShowLabels = false
	}
	return cfg
}

func scaledSize(value float32, opts RenderOptions) float32 {
	return value * opts.Scale
}

func scaledTextSize(value float32, opts RenderOptions) float32 {
	size := value * opts.Scale
	if size < 6 {
		return 6
	}
	return size
}

func addPadArea(pads *[]PadArea, idx int, pin PackagePin, pos Vec, size Vec, label string) {
	*pads = append(*pads, PadArea{
		Index:    idx,
		Pin:      pin,
		Position: pos,
		Size:     size,
		Label:    label,
	})
}

func addLabel(labels *[]LabelShape, text string, align LabelAlignment, size float32, pos Vec, col color.NRGBA) {
	*labels = append(*labels, LabelShape{
		Text:     text,
		Align:    align,
		Size:     size,
		Position: pos,
		Color:    col,
	})
}
