package renderer

import (
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
)

// RenderSymbols renders all symbol instances in the schematic
func RenderSymbols(gtx layout.Context, camera *renderer.Camera, sch *schematic.Schematic, colors *SchematicColors) {
	if sch == nil || len(sch.Symbols) == 0 {
		return
	}

	for _, symbol := range sch.Symbols {
		renderSymbol(gtx, camera, symbol, sch, colors)
	}
}

// renderSymbol renders a single symbol instance
func renderSymbol(gtx layout.Context, camera *renderer.Camera, symbol schematic.Symbol, sch *schematic.Schematic, colors *SchematicColors) {
	// Find the library symbol definition
	var libSym *schematic.LibSymbol
	for i := range sch.LibSymbols {
		if sch.LibSymbols[i].Name == symbol.LibID {
			libSym = &sch.LibSymbols[i]
			break
		}
	}

	if libSym == nil {
		// Symbol definition not found - skip rendering
		return
	}

	// Debug: Log if this is a complex symbol
	//if len(libSym.Graphics) > 10 {
	//	log.Printf("Rendering %s at (%.2f, %.2f) with %d graphics",
	//		symbol.LibID, symbol.Position.X, symbol.Position.Y, len(libSym.Graphics))
	//}

	// Convert symbol position to screen coordinates
	x, y := camera.WorldToScreen(symbol.Position)

	// Save transformation state
	stack := op.Affine(f32.Affine2D{}.Offset(f32.Pt(float32(x), float32(y)))).Push(gtx.Ops)
	defer stack.Pop()

	// Apply symbol transformations
	transform := calculateSymbolTransform(symbol)
	op.Affine(transform).Add(gtx.Ops)

	// Render symbol graphics
	renderSymbolGraphics(gtx, camera, libSym.Graphics, colors)

	// Render pins
	renderSymbolPins(gtx, camera, libSym.Pins, colors)

	// Render symbol properties (Reference, Value, etc.)
	renderSymbolProperties(gtx, camera, symbol, colors)
}

// calculateSymbolTransform calculates the transformation matrix for a symbol
func calculateSymbolTransform(symbol schematic.Symbol) f32.Affine2D {
	transform := f32.Affine2D{}

	// KiCad symbol graphics are defined with Y increasing upward
	// But schematic coordinates have Y increasing downward
	// So we need to flip Y for all symbol graphics
	transform = transform.Scale(f32.Pt(0, 0), f32.Pt(1, -1))

	// Apply rotation
	if symbol.Angle != 0 {
		radians := float64(symbol.Angle) * math.Pi / 180.0
		transform = transform.Rotate(f32.Pt(0, 0), float32(radians))
	}

	// Apply mirror
	switch symbol.Mirror {
	case "x":
		transform = transform.Scale(f32.Pt(0, 0), f32.Pt(1, -1))
	case "y":
		transform = transform.Scale(f32.Pt(0, 0), f32.Pt(-1, 1))
	case "xy":
		transform = transform.Scale(f32.Pt(0, 0), f32.Pt(-1, -1))
	}

	return transform
}

// renderSymbolGraphics renders the graphical elements of a symbol
func renderSymbolGraphics(gtx layout.Context, camera *renderer.Camera, graphics []schematic.SymGraphic, colors *SchematicColors) {
	for _, graphic := range graphics {
		switch graphic.Type {
		case "rectangle":
			renderRectangle(gtx, camera, graphic, colors)
		case "circle":
			renderCircle(gtx, camera, graphic, colors)
		case "arc":
			renderArc(gtx, camera, graphic, colors)
		case "polyline":
			renderGraphicPolyline(gtx, camera, graphic, colors)
		case "text":
			// Text rendering handled separately
			// TODO: Implement symbol text rendering
		}
	}
}

// renderRectangle renders a rectangle graphic element
func renderRectangle(gtx layout.Context, camera *renderer.Camera, graphic schematic.SymGraphic, colors *SchematicColors) {
	// Convert coordinates to screen space (but we're already in symbol space)
	// For symbol graphics, we work in symbol coordinate space, not world space
	// The zoom factor is used to scale stroke widths appropriately

	x1 := float32(graphic.Start.X * camera.Zoom)
	y1 := float32(graphic.Start.Y * camera.Zoom)
	x2 := float32(graphic.End.X * camera.Zoom)
	y2 := float32(graphic.End.Y * camera.Zoom)

	// Determine what to draw based on fill type:
	// "none" = outline only
	// "outline" or "background" = fill + outline
	// "color" = fill only (solid fill)
	drawFill := graphic.Fill.Type == "background" || graphic.Fill.Type == "outline" || graphic.Fill.Type == "color"
	drawOutline := graphic.Fill.Type != "color" // Draw outline unless solid fill

	// Draw fill if needed
	if drawFill {
		var fillPath clip.Path
		fillPath.Begin(gtx.Ops)
		fillPath.MoveTo(f32.Pt(x1, y1))
		fillPath.LineTo(f32.Pt(x2, y1))
		fillPath.LineTo(f32.Pt(x2, y2))
		fillPath.LineTo(f32.Pt(x1, y2))
		fillPath.Close()

		paint.FillShape(gtx.Ops, colors.SymbolFill, clip.Outline{
			Path: fillPath.End(),
		}.Op())
	}

	// Draw outline if needed
	if drawOutline {
		var strokePath clip.Path
		strokePath.Begin(gtx.Ops)
		strokePath.MoveTo(f32.Pt(x1, y1))
		strokePath.LineTo(f32.Pt(x2, y1))
		strokePath.LineTo(f32.Pt(x2, y2))
		strokePath.LineTo(f32.Pt(x1, y2))
		strokePath.Close()

		// Scale stroke width with zoom, use larger default for visibility
		strokeWidth := 0.25 // Default 0.25mm (KiCad default)
		if graphic.Stroke.Width > 0 {
			strokeWidth = graphic.Stroke.Width
		}
		strokeWidth *= float64(camera.Zoom)
		// Ensure minimum visible width
		if strokeWidth < 3.0 {
			strokeWidth = 3.0 // Minimum 3px for visibility
		}
		paint.FillShape(gtx.Ops, colors.SymbolBody, clip.Stroke{
			Path:  strokePath.End(),
			Width: float32(strokeWidth),
		}.Op())
	}
}

// renderCircle renders a circle graphic element
func renderCircle(gtx layout.Context, camera *renderer.Camera, graphic schematic.SymGraphic, colors *SchematicColors) {
	cx := float32(graphic.Center.X * camera.Zoom)
	cy := float32(graphic.Center.Y * camera.Zoom)
	radius := float32(graphic.Radius * camera.Zoom)

	// Helper to create circle path
	makeCirclePath := func() clip.Path {
		var p clip.Path
		p.Begin(gtx.Ops)
		const segments = 32
		for i := 0; i <= segments; i++ {
			angle := float32(i) * 2.0 * math.Pi / segments
			x := cx + radius*float32(math.Cos(float64(angle)))
			y := cy + radius*float32(math.Sin(float64(angle)))
			if i == 0 {
				p.MoveTo(f32.Pt(x, y))
			} else {
				p.LineTo(f32.Pt(x, y))
			}
		}
		p.Close()
		return p
	}

	// Determine what to draw based on fill type
	drawFill := graphic.Fill.Type == "background" || graphic.Fill.Type == "outline" || graphic.Fill.Type == "color"
	drawOutline := graphic.Fill.Type != "color"

	// Draw fill if needed
	if drawFill {
		fillPath := makeCirclePath()
		paint.FillShape(gtx.Ops, colors.SymbolFill, clip.Outline{
			Path: fillPath.End(),
		}.Op())
	}

	// Draw outline if needed
	if drawOutline {
		strokePath := makeCirclePath()
		// Scale stroke width with zoom, use larger default for visibility
		strokeWidth := 0.25 // Default 0.25mm (KiCad default)
		if graphic.Stroke.Width > 0 {
			strokeWidth = graphic.Stroke.Width
		}
		strokeWidth *= float64(camera.Zoom)
		// Ensure minimum visible width
		if strokeWidth < 3.0 {
			strokeWidth = 3.0 // Minimum 3px for visibility
		}
		paint.FillShape(gtx.Ops, colors.SymbolBody, clip.Stroke{
			Path:  strokePath.End(),
			Width: float32(strokeWidth),
		}.Op())
	}
}

// renderArc renders an arc graphic element
func renderArc(gtx layout.Context, camera *renderer.Camera, graphic schematic.SymGraphic, colors *SchematicColors) {
	cx := float32(graphic.Center.X * camera.Zoom)
	cy := float32(graphic.Center.Y * camera.Zoom)

	// Calculate radius from start point to center
	dx := graphic.Start.X - graphic.Center.X
	dy := graphic.Start.Y - graphic.Center.Y
	radius := float32(math.Sqrt(dx*dx+dy*dy) * camera.Zoom)

	startAngle := float32(graphic.Angles[0] * math.Pi / 180.0)
	endAngle := float32(graphic.Angles[1] * math.Pi / 180.0)

	var path clip.Path
	path.Begin(gtx.Ops)

	// Draw arc using line segments
	const segments = 32
	angleRange := endAngle - startAngle
	for i := 0; i <= segments; i++ {
		t := float32(i) / segments
		angle := startAngle + t*angleRange
		x := cx + radius*float32(math.Cos(float64(angle)))
		y := cy + radius*float32(math.Sin(float64(angle)))

		if i == 0 {
			path.MoveTo(f32.Pt(x, y))
		} else {
			path.LineTo(f32.Pt(x, y))
		}
	}

	// Scale stroke width with zoom, use larger default for visibility
	strokeWidth := 0.25 // Default 0.25mm (KiCad default)
	if graphic.Stroke.Width > 0 {
		strokeWidth = graphic.Stroke.Width
	}
	strokeWidth *= float64(camera.Zoom)
	// Ensure minimum visible width
	if strokeWidth < 3.0 {
		strokeWidth = 3.0 // Minimum 3px for visibility
	}
	paint.FillShape(gtx.Ops, colors.SymbolBody, clip.Stroke{
		Path:  path.End(),
		Width: float32(strokeWidth),
	}.Op())
}

// renderGraphicPolyline renders a polyline graphic element
func renderGraphicPolyline(gtx layout.Context, camera *renderer.Camera, graphic schematic.SymGraphic, colors *SchematicColors) {
	if len(graphic.Points) < 2 {
		return
	}

	// Helper to create polyline path
	makePolyPath := func() clip.Path {
		var p clip.Path
		p.Begin(gtx.Ops)
		x0 := float32(graphic.Points[0].X * camera.Zoom)
		y0 := float32(graphic.Points[0].Y * camera.Zoom)
		p.MoveTo(f32.Pt(x0, y0))
		for i := 1; i < len(graphic.Points); i++ {
			x := float32(graphic.Points[i].X * camera.Zoom)
			y := float32(graphic.Points[i].Y * camera.Zoom)
			p.LineTo(f32.Pt(x, y))
		}
		return p
	}

	// Determine what to draw based on fill type
	drawFill := graphic.Fill.Type == "background" || graphic.Fill.Type == "outline" || graphic.Fill.Type == "color"
	drawOutline := graphic.Fill.Type != "color"

	// Draw fill if needed
	if drawFill {
		fillPath := makePolyPath()
		fillPath.Close()
		paint.FillShape(gtx.Ops, colors.SymbolFill, clip.Outline{
			Path: fillPath.End(),
		}.Op())
	}

	// Draw outline if needed
	if drawOutline {
		strokePath := makePolyPath()
		// Scale stroke width with zoom, use larger default for visibility
		strokeWidth := 0.25 // Default 0.25mm (KiCad default)
		if graphic.Stroke.Width > 0 {
			strokeWidth = graphic.Stroke.Width
		}
		strokeWidth *= float64(camera.Zoom)
		// Ensure minimum visible width
		if strokeWidth < 3.0 {
			strokeWidth = 3.0 // Minimum 3px for visibility
		}
		paint.FillShape(gtx.Ops, colors.SymbolBody, clip.Stroke{
			Path:  strokePath.End(),
			Width: float32(strokeWidth),
		}.Op())
	}
}

// renderSymbolPins renders all pins of a symbol
func renderSymbolPins(gtx layout.Context, camera *renderer.Camera, pins []schematic.Pin, colors *SchematicColors) {
	for _, pin := range pins {
		renderPin(gtx, camera, pin, colors)
	}
}

// renderPin renders a single pin
func renderPin(gtx layout.Context, camera *renderer.Camera, pin schematic.Pin, colors *SchematicColors) {
	if pin.Hide {
		return
	}

	// Pin position in symbol space
	px := float32(pin.Position.X * camera.Zoom)
	py := float32(pin.Position.Y * camera.Zoom)

	// Calculate pin endpoint based on angle and length
	length := float32(pin.Length * camera.Zoom)
	var ex, ey float32

	switch pin.Angle {
	case 0: // Right
		ex = px + length
		ey = py
	case 90: // Up
		ex = px
		ey = py - length
	case 180: // Left
		ex = px - length
		ey = py
	case 270: // Down
		ex = px
		ey = py + length
	default:
		// Handle arbitrary angles
		radians := float64(pin.Angle) * math.Pi / 180.0
		ex = px + length*float32(math.Cos(radians))
		ey = py + length*float32(math.Sin(radians))
	}

	// Draw pin line based on style
	switch pin.Style {
	case "inverted":
		renderInvertedPin(gtx, px, py, ex, ey, colors)
	case "clock":
		renderClockPin(gtx, px, py, ex, ey, colors)
	case "inverted_clock":
		renderInvertedClockPin(gtx, px, py, ex, ey, colors)
	default: // "line" or unspecified
		renderLinePin(gtx, px, py, ex, ey, colors)
	}

	// TODO: Render pin name and number
	// This requires proper text rendering which we'll add later
}

// renderLinePin renders a simple line pin
func renderLinePin(gtx layout.Context, px, py, ex, ey float32, colors *SchematicColors) {
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(px, py))
	path.LineTo(f32.Pt(ex, ey))

	paint.FillShape(gtx.Ops, colors.SymbolPin, clip.Stroke{
		Path:  path.End(),
		Width: 2.0,
	}.Op())
}

// renderInvertedPin renders a pin with an inversion bubble
func renderInvertedPin(gtx layout.Context, px, py, ex, ey float32, colors *SchematicColors) {
	// Draw bubble at pin position
	const bubbleRadius = 4.0

	// Calculate bubble center
	dx := ex - px
	dy := ey - py
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if length > 0 {
		dx /= length
		dy /= length
	}

	bubbleCenterX := px + dx*bubbleRadius
	bubbleCenterY := py + dy*bubbleRadius

	// Draw circle for bubble
	var circlePath clip.Path
	circlePath.Begin(gtx.Ops)
	const segments = 16
	for i := 0; i <= segments; i++ {
		angle := float32(i) * 2.0 * math.Pi / segments
		x := bubbleCenterX + bubbleRadius*float32(math.Cos(float64(angle)))
		y := bubbleCenterY + bubbleRadius*float32(math.Sin(float64(angle)))
		if i == 0 {
			circlePath.MoveTo(f32.Pt(x, y))
		} else {
			circlePath.LineTo(f32.Pt(x, y))
		}
	}
	circlePath.Close()

	paint.FillShape(gtx.Ops, colors.SymbolPin, clip.Stroke{
		Path:  circlePath.End(),
		Width: 2.0,
	}.Op())

	// Draw line from bubble to endpoint
	lineStartX := px + dx*(bubbleRadius*2)
	lineStartY := py + dy*(bubbleRadius*2)

	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(lineStartX, lineStartY))
	path.LineTo(f32.Pt(ex, ey))

	paint.FillShape(gtx.Ops, colors.SymbolPin, clip.Stroke{
		Path:  path.End(),
		Width: 2.0,
	}.Op())
}

// renderClockPin renders a pin with a clock symbol
func renderClockPin(gtx layout.Context, px, py, ex, ey float32, colors *SchematicColors) {
	// Draw main pin line
	renderLinePin(gtx, px, py, ex, ey, colors)

	// Draw clock triangle at pin position
	const triangleSize = 6.0

	dx := ex - px
	dy := ey - py
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if length > 0 {
		dx /= length
		dy /= length
	}

	// Perpendicular vector
	perpX := -dy
	perpY := dx

	// Triangle points
	tipX := px + dx*triangleSize
	tipY := py + dy*triangleSize
	base1X := px + perpX*triangleSize*0.5
	base1Y := py + perpY*triangleSize*0.5
	base2X := px - perpX*triangleSize*0.5
	base2Y := py - perpY*triangleSize*0.5

	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(tipX, tipY))
	path.LineTo(f32.Pt(base1X, base1Y))
	path.LineTo(f32.Pt(base2X, base2Y))
	path.Close()

	paint.FillShape(gtx.Ops, colors.SymbolPin, clip.Stroke{
		Path:  path.End(),
		Width: 2.0,
	}.Op())
}

// renderInvertedClockPin renders a pin with both inversion bubble and clock symbol
func renderInvertedClockPin(gtx layout.Context, px, py, ex, ey float32, colors *SchematicColors) {
	// Draw inversion bubble
	renderInvertedPin(gtx, px, py, ex, ey, colors)

	// Draw clock triangle after the bubble
	const bubbleRadius = 4.0
	const triangleSize = 6.0

	dx := ex - px
	dy := ey - py
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if length > 0 {
		dx /= length
		dy /= length
	}

	// Position clock triangle after bubble
	clockPX := px + dx*(bubbleRadius*2)
	clockPY := py + dy*(bubbleRadius*2)

	perpX := -dy
	perpY := dx

	tipX := clockPX + dx*triangleSize
	tipY := clockPY + dy*triangleSize
	base1X := clockPX + perpX*triangleSize*0.5
	base1Y := clockPY + perpY*triangleSize*0.5
	base2X := clockPX - perpX*triangleSize*0.5
	base2Y := clockPY - perpY*triangleSize*0.5

	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(tipX, tipY))
	path.LineTo(f32.Pt(base1X, base1Y))
	path.LineTo(f32.Pt(base2X, base2Y))
	path.Close()

	paint.FillShape(gtx.Ops, colors.SymbolPin, clip.Stroke{
		Path:  path.End(),
		Width: 2.0,
	}.Op())
}

// renderSymbolProperties renders the symbol's properties (Reference, Value, etc.)
func renderSymbolProperties(gtx layout.Context, camera *renderer.Camera, symbol schematic.Symbol, colors *SchematicColors) {
	// TODO: Implement proper text rendering for symbol properties
	// This requires proper text shaping and positioning
	// For now, we just set the color for when text rendering is implemented
	paint.ColorOp{Color: colors.SymbolText}.Add(gtx.Ops)

	// Properties to render:
	// - Reference (e.g., "R1", "U1", "C2")
	// - Value (e.g., "10k", "74HC04", "100nF")
	// - Other visible properties

	for _, prop := range symbol.Properties {
		if prop.Key == "Reference" || prop.Key == "Value" {
			// TODO: Render property text at appropriate position
			// This will be implemented when we add proper text rendering
			_ = prop.Value
		}
	}
}
