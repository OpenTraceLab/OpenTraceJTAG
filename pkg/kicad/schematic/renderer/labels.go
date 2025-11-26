package renderer

import (
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
)

// Global theme for text rendering
var defaultTheme = material.NewTheme()

func init() {
	defaultTheme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
}

// RenderLabels renders all local labels in the schematic
func RenderLabels(gtx layout.Context, camera *renderer.Camera, labels []schematic.Label, colors *SchematicColors) {
	if len(labels) == 0 {
		return
	}

	for _, label := range labels {
		renderLocalLabel(gtx, camera, label, colors)
	}
}

// RenderGlobalLabels renders all global labels in the schematic
func RenderGlobalLabels(gtx layout.Context, camera *renderer.Camera, labels []schematic.GlobalLabel, colors *SchematicColors) {
	if len(labels) == 0 {
		return
	}

	for _, label := range labels {
		renderGlobalLabel(gtx, camera, label, colors)
	}
}

// RenderHierLabels renders all hierarchical labels in the schematic
func RenderHierLabels(gtx layout.Context, camera *renderer.Camera, labels []schematic.HierLabel, colors *SchematicColors) {
	if len(labels) == 0 {
		return
	}

	for _, label := range labels {
		renderHierLabel(gtx, camera, label, colors)
	}
}

// renderLocalLabel renders a single local label
func renderLocalLabel(gtx layout.Context, camera *renderer.Camera, label schematic.Label, colors *SchematicColors) {
	x, y := camera.WorldToScreen(label.Position)

	// Save the current transformation
	stack := op.Affine(f32.Affine2D{}.Offset(f32.Pt(float32(x), float32(y)))).Push(gtx.Ops)
	defer stack.Pop()

	// Apply rotation if needed
	if label.Angle != 0 {
		radians := float64(label.Angle) * math.Pi / 180.0
		rot := f32.Affine2D{}.Rotate(f32.Pt(0, 0), float32(radians))
		op.Affine(rot).Add(gtx.Ops)
	}

	// Render the text
	renderLabelText(gtx, label.Text, label.Effects, colors.LocalLabel)
}

// renderGlobalLabel renders a single global label with its shape
func renderGlobalLabel(gtx layout.Context, camera *renderer.Camera, label schematic.GlobalLabel, colors *SchematicColors) {
	x, y := camera.WorldToScreen(label.Position)

	// Save the current transformation
	stack := op.Affine(f32.Affine2D{}.Offset(f32.Pt(float32(x), float32(y)))).Push(gtx.Ops)
	defer stack.Pop()

	// Apply rotation if needed
	if label.Angle != 0 {
		radians := float64(label.Angle) * math.Pi / 180.0
		rot := f32.Affine2D{}.Rotate(f32.Pt(0, 0), float32(radians))
		op.Affine(rot).Add(gtx.Ops)
	}

	// Draw shape indicator based on label shape
	drawLabelShape(gtx, label.Shape, label.Text, label.Effects, colors)

	// Render the text
	renderLabelText(gtx, label.Text, label.Effects, colors.GlobalLabel)
}

// renderHierLabel renders a single hierarchical label
func renderHierLabel(gtx layout.Context, camera *renderer.Camera, label schematic.HierLabel, colors *SchematicColors) {
	x, y := camera.WorldToScreen(label.Position)

	// Save the current transformation
	stack := op.Affine(f32.Affine2D{}.Offset(f32.Pt(float32(x), float32(y)))).Push(gtx.Ops)
	defer stack.Pop()

	// Apply rotation if needed
	if label.Angle != 0 {
		radians := float64(label.Angle) * math.Pi / 180.0
		rot := f32.Affine2D{}.Rotate(f32.Pt(0, 0), float32(radians))
		op.Affine(rot).Add(gtx.Ops)
	}

	// Draw shape indicator
	drawLabelShape(gtx, label.Shape, label.Text, label.Effects, colors)

	// Render the text
	renderLabelText(gtx, label.Text, label.Effects, colors.HierLabel)
}

// renderLabelText renders the text content of a label
func renderLabelText(gtx layout.Context, textStr string, effects schematic.Effects, labelColor color.NRGBA) {
	if textStr == "" {
		return
	}

	// Determine font size - KiCad schematic text is typically around 1.27mm (50mil)
	fontSize := 12.0 // Default size in points
	if effects.Font.Size.Height > 0 {
		// Convert from mm to points (1mm ≈ 2.83 points)
		fontSize = effects.Font.Size.Height * 2.83
	}

	// Create a material label for text rendering
	lbl := material.Label(defaultTheme, unit.Sp(fontSize), textStr)
	lbl.Color = labelColor
	lbl.Alignment = text.Start

	// Render the text
	lbl.Layout(gtx)
}

// drawLabelShape draws the background shape for global/hierarchical labels
func drawLabelShape(gtx layout.Context, shape string, text string, effects schematic.Effects, colors *SchematicColors) {
	if shape == "" {
		return
	}

	// Estimate text width (rough approximation)
	textWidth := float32(len(text) * 8)
	textHeight := float32(16)

	if effects.Font.Size.Height > 0 {
		textHeight = float32(effects.Font.Size.Height * 3.5)
		textWidth = float32(len(text)) * textHeight * 0.6
	}

	const arrowSize = 8.0
	const padding = 4.0

	var path clip.Path
	path.Begin(gtx.Ops)

	switch shape {
	case "input":
		// Arrow pointing into the label (left side)
		// Draw rectangle with arrow on left
		path.MoveTo(f32.Pt(-arrowSize, textHeight/2))
		path.LineTo(f32.Pt(0, 0))
		path.LineTo(f32.Pt(textWidth+padding, 0))
		path.LineTo(f32.Pt(textWidth+padding, textHeight))
		path.LineTo(f32.Pt(0, textHeight))
		path.LineTo(f32.Pt(-arrowSize, textHeight/2))
		path.Close()

	case "output":
		// Arrow pointing out of the label (right side)
		path.MoveTo(f32.Pt(0, 0))
		path.LineTo(f32.Pt(textWidth+padding, 0))
		path.LineTo(f32.Pt(textWidth+padding+arrowSize, textHeight/2))
		path.LineTo(f32.Pt(textWidth+padding, textHeight))
		path.LineTo(f32.Pt(0, textHeight))
		path.Close()

	case "bidirectional":
		// Arrows on both sides
		path.MoveTo(f32.Pt(-arrowSize, textHeight/2))
		path.LineTo(f32.Pt(0, 0))
		path.LineTo(f32.Pt(textWidth+padding, 0))
		path.LineTo(f32.Pt(textWidth+padding+arrowSize, textHeight/2))
		path.LineTo(f32.Pt(textWidth+padding, textHeight))
		path.LineTo(f32.Pt(0, textHeight))
		path.Close()

	case "passive", "3state", "unspecified":
		// Simple rectangle
		path.MoveTo(f32.Pt(0, 0))
		path.LineTo(f32.Pt(textWidth+padding, 0))
		path.LineTo(f32.Pt(textWidth+padding, textHeight))
		path.LineTo(f32.Pt(0, textHeight))
		path.Close()

	default:
		// Unknown shape - draw simple rectangle
		path.MoveTo(f32.Pt(0, 0))
		path.LineTo(f32.Pt(textWidth+padding, 0))
		path.LineTo(f32.Pt(textWidth+padding, textHeight))
		path.LineTo(f32.Pt(0, textHeight))
		path.Close()
	}

	// Draw the shape outline
	paint.FillShape(gtx.Ops, colors.GlobalLabel, clip.Stroke{
		Path:  path.End(),
		Width: 2.0,
	}.Op())
}
