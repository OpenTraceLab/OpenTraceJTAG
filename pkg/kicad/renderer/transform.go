package renderer

import (
	"math"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/pcb"
)

// Transform represents a 2D transformation (translate + rotate + scale)
type Transform struct {
	TranslateX float64 // Translation in X
	TranslateY float64 // Translation in Y
	Rotate     float64 // Rotation in degrees
	ScaleX     float64 // Scale factor in X
	ScaleY     float64 // Scale factor in Y
}

// NewTransform creates an identity transform
func NewTransform() Transform {
	return Transform{
		ScaleX: 1.0,
		ScaleY: 1.0,
	}
}

// Apply applies the transformation to a position
func (t Transform) Apply(pos pcb.Position) pcb.Position {
	x, y := pos.X, pos.Y

	// Apply scale
	x *= t.ScaleX
	y *= t.ScaleY

	// Apply rotation (convert to radians)
	if t.Rotate != 0 {
		rad := t.Rotate * math.Pi / 180.0
		cos := math.Cos(rad)
		sin := math.Sin(rad)
		newX := x*cos - y*sin
		newY := x*sin + y*cos
		x = newX
		y = newY
	}

	// Apply translation
	x += t.TranslateX
	y += t.TranslateY

	return pcb.Position{X: x, Y: y}
}

// ApplyInverse applies the inverse transformation (for screen to world)
func (t Transform) ApplyInverse(pos pcb.Position) pcb.Position {
	x, y := pos.X, pos.Y

	// Inverse translation
	x -= t.TranslateX
	y -= t.TranslateY

	// Inverse rotation
	if t.Rotate != 0 {
		rad := -t.Rotate * math.Pi / 180.0 // Negative for inverse
		cos := math.Cos(rad)
		sin := math.Sin(rad)
		newX := x*cos - y*sin
		newY := x*sin + y*cos
		x = newX
		y = newY
	}

	// Inverse scale
	if t.ScaleX != 0 {
		x /= t.ScaleX
	}
	if t.ScaleY != 0 {
		y /= t.ScaleY
	}

	return pcb.Position{X: x, Y: y}
}

// NOTE: Camera has been moved to camera.go and is now generic (works with both PCB and schematics)
// The new Camera uses sexp.Position but includes backward-compatible methods like BoardToScreen and FitBoard
