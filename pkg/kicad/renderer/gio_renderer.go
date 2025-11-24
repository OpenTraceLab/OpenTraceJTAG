package renderer

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser"
)

// RenderBoard renders the entire board using Gio operations
func RenderBoard(gtx layout.Context, camera *Camera, board *parser.Board) {
	RenderBoardWithDebug(gtx, camera, board, 0.0)
}

// RenderBoardWithConfig renders the board with layer visibility control
func RenderBoardWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	renderBoardWithOptions(gtx, camera, board, 0.0, "", config)
}

// RenderBoardWithHighlight renders the board with a specific net highlighted
func RenderBoardWithHighlight(gtx layout.Context, camera *Camera, board *parser.Board, highlightNet string) {
	renderBoardWithOptions(gtx, camera, board, 0.0, highlightNet, nil)
}

// RenderBoardWithDebug renders the entire board with a debug rotation offset for J1/J2/U5
func RenderBoardWithDebug(gtx layout.Context, camera *Camera, board *parser.Board, debugRotationOffset float64) {
	renderBoardWithOptions(gtx, camera, board, debugRotationOffset, "", nil)
}

// renderBoardWithOptions is the internal render function with all options
func renderBoardWithOptions(gtx layout.Context, camera *Camera, board *parser.Board, debugRotationOffset float64, highlightNet string, config *LayerConfig) {
	// Use default config if none provided
	if config == nil {
		config = NewLayerConfig()
	}

	// Render in layer order (bottom to top)
	renderZonesWithConfig(gtx, camera, board, config)
	
	if highlightNet != "" {
		renderTracksWithHighlight(gtx, camera, board, highlightNet)
	} else {
		renderTracksWithConfig(gtx, camera, board, config)
	}
	
	renderGraphicsWithConfig(gtx, camera, board, config)
	renderFootprintAdhesiveWithConfig(gtx, camera, board, config)
	renderFootprintPasteWithConfig(gtx, camera, board, config)
	renderFootprintFabWithConfig(gtx, camera, board, config)
	renderFootprintTextWithConfig(gtx, camera, board, debugRotationOffset, config)
	renderFootprintCourtyardsWithConfig(gtx, camera, board, config)
	renderFootprintSilkscreenWithConfig(gtx, camera, board, debugRotationOffset, config)
	renderFootprintMaskWithConfig(gtx, camera, board, config)
	
	if highlightNet != "" {
		renderViasWithHighlight(gtx, camera, board, highlightNet)
		renderPadsWithHighlight(gtx, camera, board, debugRotationOffset, highlightNet)
	} else {
		renderViasWithConfig(gtx, camera, board, config)
		renderPadsWithConfig(gtx, camera, board, debugRotationOffset, config)
	}
}

// renderPads renders all pads with proper rotation support
func renderPads(gtx layout.Context, camera *Camera, board *parser.Board, debugRotationOffset float64) {
	for _, fp := range board.Footprints {
		for _, pad := range fp.Pads {
			// Get absolute pad position in board coordinates
			absPos := fp.TransformPosition(pad.Position)
			sx, sy := camera.BoardToScreen(absPos)

			// Calculate pad rotation - negate to match coordinate system
			totalAngle := -float64(pad.Position.Angle)

			// Apply debug rotation offset for J1, J2, and U5
			if (fp.Reference == "J1" || fp.Reference == "J2" || fp.Reference == "U5") && debugRotationOffset != 0.0 {
				totalAngle += debugRotationOffset
			}

			// Convert to radians
			radians := totalAngle * math.Pi / 180.0

			// Calculate pad size in screen pixels
			width := pad.Size.Width * camera.Zoom
			height := pad.Size.Height * camera.Zoom

			// Enforce minimum size while preserving aspect ratio
			if width < 2.0 || height < 2.0 {
				aspectRatio := pad.Size.Width / pad.Size.Height
				if width < height {
					width = 2.0
					height = width / aspectRatio
				} else {
					height = 2.0
					width = height * aspectRatio
				}
			}

			// Use KiCad colors
			padColor := ColorPadTH
			if pad.Type == "smd" {
				padColor = ColorPadSMD
			}

			// Render pad based on shape
			switch pad.Shape {
			case "circle":
				renderCirclePad(gtx, sx, sy, width, height, padColor, padColor)

			case "oval":
				// Check if it's nearly circular
				aspectRatio := width / height
				if aspectRatio > 0.95 && aspectRatio < 1.05 {
					renderCirclePad(gtx, sx, sy, width, height, padColor, padColor)
				} else {
					renderRotatedRRect(gtx, sx, sy, width, height, radians,
						math.Min(width, height)*0.5, padColor, padColor)
				}

			case "roundrect":
				cornerRadius := math.Min(width, height) * 0.25
				renderRotatedRRect(gtx, sx, sy, width, height, radians,
					cornerRadius, padColor, padColor)

			case "rect":
				renderRotatedRect(gtx, sx, sy, width, height, radians, padColor, padColor)

			default:
				renderRotatedRect(gtx, sx, sy, width, height, radians, padColor, padColor)
			}

			// Render drill hole if through-hole
			if pad.Drill > 0 {
				drillRadius := pad.Drill / 2.0 * camera.Zoom
				if drillRadius < 1.0 {
					drillRadius = 1.0
				}
				renderCircle(gtx, sx, sy, drillRadius, ColorDrill)
			}
		}
	}
}

// renderPadsWithHighlight renders pads with highlighted net
func renderPadsWithHighlight(gtx layout.Context, camera *Camera, board *parser.Board, debugRotationOffset float64, highlightNet string) {
	for _, fp := range board.Footprints {
		for _, pad := range fp.Pads {
			// Get absolute pad position in board coordinates
			absPos := fp.TransformPosition(pad.Position)
			sx, sy := camera.BoardToScreen(absPos)

			// Calculate pad rotation
			totalAngle := -float64(pad.Position.Angle)
			if (fp.Reference == "J1" || fp.Reference == "J2" || fp.Reference == "U5") && debugRotationOffset != 0.0 {
				totalAngle += debugRotationOffset
			}
			radians := totalAngle * math.Pi / 180.0

			// Calculate pad size in screen pixels
			width := pad.Size.Width * camera.Zoom
			height := pad.Size.Height * camera.Zoom

			if width < 2.0 || height < 2.0 {
				aspectRatio := pad.Size.Width / pad.Size.Height
				if width < height {
					width = 2.0
					height = width / aspectRatio
				} else {
					height = 2.0
					width = height * aspectRatio
				}
			}

			// Determine pad color with highlight
			padColor := ColorPadTH
			if pad.Type == "smd" {
				padColor = ColorPadSMD
			}

			if highlightNet != "" && (pad.Net == nil || pad.Net.Name != highlightNet) {
				// Dim non-highlighted pads
				padColor.A = 60
			} else if highlightNet != "" && pad.Net != nil && pad.Net.Name == highlightNet {
				// Highlight color (bright yellow)
				padColor = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
			}

			// Render pad based on shape
			switch pad.Shape {
			case "circle":
				renderCirclePad(gtx, sx, sy, width, height, padColor, padColor)
			case "oval":
				aspectRatio := width / height
				if aspectRatio > 0.95 && aspectRatio < 1.05 {
					renderCirclePad(gtx, sx, sy, width, height, padColor, padColor)
				} else {
					renderRotatedRRect(gtx, sx, sy, width, height, radians,
						math.Min(width, height)*0.5, padColor, padColor)
				}
			case "roundrect":
				cornerRadius := math.Min(width, height) * 0.25
				renderRotatedRRect(gtx, sx, sy, width, height, radians,
					cornerRadius, padColor, padColor)
			case "rect":
				renderRotatedRect(gtx, sx, sy, width, height, radians, padColor, padColor)
			default:
				renderRotatedRect(gtx, sx, sy, width, height, radians, padColor, padColor)
			}

			// Render drill hole if through-hole
			if pad.Drill > 0 {
				drillRadius := pad.Drill / 2.0 * camera.Zoom
				if drillRadius < 1.0 {
					drillRadius = 1.0
				}
				renderCircle(gtx, sx, sy, drillRadius, ColorDrill)
			}
		}
	}
}

// renderCirclePad renders a circular pad
func renderCirclePad(gtx layout.Context, x, y, width, height float64, fillColor, strokeColor color.NRGBA) {
	radius := (width + height) / 4.0 // Average
	if radius < 1.0 {
		radius = 1.0
	}

	stack := op.Affine(f32.Affine2D{}.Offset(f32.Pt(float32(x), float32(y)))).Push(gtx.Ops)
	defer stack.Pop()

	// Create square bounding rectangle for circle
	rect := image.Rectangle{
		Min: image.Pt(int(-radius), int(-radius)),
		Max: image.Pt(int(radius), int(radius)),
	}
	path := clip.Ellipse(rect).Op(gtx.Ops)
	paint.FillShape(gtx.Ops, fillColor, path)
}

// renderCircle renders a simple filled circle
func renderCircle(gtx layout.Context, x, y, radius float64, fillColor color.NRGBA) {
	stack := op.Affine(f32.Affine2D{}.Offset(f32.Pt(float32(x), float32(y)))).Push(gtx.Ops)
	defer stack.Pop()

	// Create square bounding rectangle for circle
	rect := image.Rectangle{
		Min: image.Pt(int(-radius), int(-radius)),
		Max: image.Pt(int(radius), int(radius)),
	}
	path := clip.Ellipse(rect).Op(gtx.Ops)
	paint.FillShape(gtx.Ops, fillColor, path)
}

// renderRotatedRect renders a rotated rectangle
func renderRotatedRect(gtx layout.Context, x, y, width, height, radians float64, fillColor, strokeColor color.NRGBA) {
	// Create transformation: translate to position
	transform := f32.Affine2D{}.Offset(f32.Pt(float32(x), float32(y)))

	stack := op.Affine(transform).Push(gtx.Ops)
	defer stack.Pop()

	// Create a path for the rotated rectangle
	var path clip.Path
	path.Begin(gtx.Ops)

	// Calculate rotated corners
	cos := float32(math.Cos(float64(radians)))
	sin := float32(math.Sin(float64(radians)))
	hw := float32(width / 2)
	hh := float32(height / 2)

	// Four corners of rectangle, rotated
	x1 := -hw*cos - (-hh)*sin
	y1 := -hw*sin + (-hh)*cos
	x2 := hw*cos - (-hh)*sin
	y2 := hw*sin + (-hh)*cos
	x3 := hw*cos - hh*sin
	y3 := hw*sin + hh*cos
	x4 := -hw*cos - hh*sin
	y4 := -hw*sin + hh*cos

	// Draw the rotated rectangle
	path.MoveTo(f32.Pt(x1, y1))
	path.LineTo(f32.Pt(x2, y2))
	path.LineTo(f32.Pt(x3, y3))
	path.LineTo(f32.Pt(x4, y4))
	path.Close()

	paint.FillShape(gtx.Ops, fillColor, clip.Outline{Path: path.End()}.Op())
}

// renderRotatedRRect renders a rotated rounded rectangle
func renderRotatedRRect(gtx layout.Context, x, y, width, height, radians, cornerRadius float64, fillColor, strokeColor color.NRGBA) {
	// Create transformation: rotate around pad center, then translate
	transform := f32.Affine2D{}.
		Rotate(f32.Pt(0, 0), float32(radians)).
		Offset(f32.Pt(float32(x), float32(y)))

	stack := op.Affine(transform).Push(gtx.Ops)
	defer stack.Pop()

	// Draw rounded rectangle centered at origin using Gio's built-in support
	rrect := clip.UniformRRect(
		image.Rectangle{
			Min: image.Pt(int(-width/2), int(-height/2)),
			Max: image.Pt(int(width/2), int(height/2)),
		},
		int(cornerRadius),
	).Op(gtx.Ops)

	paint.FillShape(gtx.Ops, fillColor, rrect)
}

// Old implementation that had clipping issues
func renderRotatedRRectOld(gtx layout.Context, x, y, width, height, radians, cornerRadius float64, fillColor, strokeColor color.NRGBA) {
	// Create transformation: rotate around pad center, then translate
	// Order matters: rotation first, then translation
	transform := f32.Affine2D{}.
		Rotate(f32.Pt(0, 0), float32(radians)).
		Offset(f32.Pt(float32(x), float32(y)))

	stack := op.Affine(transform).Push(gtx.Ops)
	defer stack.Pop()

	// Draw rounded rectangle centered at origin
	rrect := clip.UniformRRect(
		image.Rectangle{
			Min: image.Pt(int(-width/2), int(-height/2)),
			Max: image.Pt(int(width/2), int(height/2)),
		},
		int(cornerRadius),
	).Op(gtx.Ops)

	paint.FillShape(gtx.Ops, fillColor, rrect)
}

// renderTracks renders all tracks
func renderTracks(gtx layout.Context, camera *Camera, board *parser.Board) {
	for _, track := range board.Tracks {
		x1, y1 := camera.BoardToScreen(track.Start)
		x2, y2 := camera.BoardToScreen(track.End)

		strokeWidth := track.Width * camera.Zoom
		if strokeWidth < 1.0 {
			strokeWidth = 1.0
		}

		trackColor := GetLayerColor(track.Layer)
		renderLine(gtx, x1, y1, x2, y2, strokeWidth, trackColor)
	}
}

// renderTracksWithHighlight renders tracks with highlighted net
func renderTracksWithHighlight(gtx layout.Context, camera *Camera, board *parser.Board, highlightNet string) {
	for _, track := range board.Tracks {
		x1, y1 := camera.BoardToScreen(track.Start)
		x2, y2 := camera.BoardToScreen(track.End)

		strokeWidth := track.Width * camera.Zoom
		if strokeWidth < 1.0 {
			strokeWidth = 1.0
		}

		trackColor := GetLayerColor(track.Layer)
		if highlightNet != "" && (track.Net == nil || track.Net.Name != highlightNet) {
			// Dim non-highlighted tracks
			trackColor.A = 60
		} else if highlightNet != "" && track.Net != nil && track.Net.Name == highlightNet {
			// Highlight color (bright yellow)
			trackColor = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
			strokeWidth *= 1.5 // Make highlighted tracks thicker
		}
		renderLine(gtx, x1, y1, x2, y2, strokeWidth, trackColor)
	}
}

// renderLine renders a line with given width
func renderLine(gtx layout.Context, x1, y1, x2, y2, width float64, lineColor color.NRGBA) {
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(float32(x1), float32(y1)))
	path.LineTo(f32.Pt(float32(x2), float32(y2)))

	stroke := clip.Stroke{
		Path:  path.End(),
		Width: float32(width),
	}.Op()

	paint.FillShape(gtx.Ops, lineColor, stroke)
}

// renderVias renders all vias
func renderVias(gtx layout.Context, camera *Camera, board *parser.Board) {
	for _, via := range board.Vias {
		x, y := camera.BoardToScreen(via.Position)
		radius := via.Size / 2.0 * camera.Zoom

		if radius < 2.0 {
			radius = 2.0
		}

		renderCircle(gtx, x, y, radius, ColorVia)

		drillRadius := via.Drill / 2.0 * camera.Zoom
		if drillRadius < 1.0 {
			drillRadius = 1.0
		}
		if drillRadius < radius {
			renderCircle(gtx, x, y, drillRadius, ColorViaDrill)
		}
	}
}

// renderViasWithHighlight renders vias with highlighted net
func renderViasWithHighlight(gtx layout.Context, camera *Camera, board *parser.Board, highlightNet string) {
	for _, via := range board.Vias {
		x, y := camera.BoardToScreen(via.Position)
		radius := via.Size / 2.0 * camera.Zoom

		if radius < 2.0 {
			radius = 2.0
		}

		viaColor := ColorVia
		if highlightNet != "" && (via.Net == nil || via.Net.Name != highlightNet) {
			// Dim non-highlighted vias
			viaColor.A = 60
		} else if highlightNet != "" && via.Net != nil && via.Net.Name == highlightNet {
			// Highlight color (bright yellow)
			viaColor = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
			radius *= 1.3 // Make highlighted vias larger
		}

		renderCircle(gtx, x, y, radius, viaColor)

		drillRadius := via.Drill / 2.0 * camera.Zoom
		if drillRadius < 1.0 {
			drillRadius = 1.0
		}
		if drillRadius < radius {
			renderCircle(gtx, x, y, drillRadius, ColorViaDrill)
		}
	}
}

// renderZones renders copper fill zones
func renderZones(gtx layout.Context, camera *Camera, board *parser.Board) {
	for _, zone := range board.Zones {
		zoneColor := GetLayerColor(zone.Layer)
		// Make zones semi-transparent
		zoneColor.A = 180

		for _, fill := range zone.Fills {
			if len(fill) < 3 {
				continue
			}

			var path clip.Path
			path.Begin(gtx.Ops)

			for i, pt := range fill {
				x, y := camera.BoardToScreen(pt)
				if i == 0 {
					path.MoveTo(f32.Pt(float32(x), float32(y)))
				} else {
					path.LineTo(f32.Pt(float32(x), float32(y)))
				}
			}
			path.Close()

			paint.FillShape(gtx.Ops, zoneColor, clip.Outline{Path: path.End()}.Op())
		}
	}
}

// renderFootprintAdhesive renders F.Adhes and B.Adhes layers
func renderFootprintAdhesive(gtx layout.Context, camera *Camera, board *parser.Board) {
	renderFootprintLayer(gtx, camera, board, "F.Adhes", "B.Adhes", 0.0)
}

// renderFootprintPaste renders F.Paste and B.Paste layers
func renderFootprintPaste(gtx layout.Context, camera *Camera, board *parser.Board) {
	renderFootprintLayer(gtx, camera, board, "F.Paste", "B.Paste", 0.0)
}

// renderFootprintMask renders F.Mask and B.Mask layers
func renderFootprintMask(gtx layout.Context, camera *Camera, board *parser.Board) {
	renderFootprintLayer(gtx, camera, board, "F.Mask", "B.Mask", 0.0)
}

// renderFootprintText renders footprint Reference and Value text
func renderFootprintText(gtx layout.Context, camera *Camera, board *parser.Board, debugRotationOffset float64) {
	collection := gofont.Collection()
	shaper := text.NewShaper(text.WithCollection(collection))
	
	for _, fp := range board.Footprints {
		// Skip if no reference
		if fp.Reference == "" {
			continue
		}
		
		// Get footprint rotation
		fpAngleDeg := -float64(fp.Position.Angle)
		if (fp.Reference == "J1" || fp.Reference == "J2" || fp.Reference == "U5") && debugRotationOffset != 0.0 {
			fpAngleDeg += debugRotationOffset
		}
		
		// For now, render Reference at footprint position
		// TODO: Parse actual fp_text position and rotation from properties
		x, y := camera.BoardToScreen(parser.Position{X: fp.Position.X, Y: fp.Position.Y})
		
		fontSize := 1.0 * camera.Zoom // 1mm text
		if fontSize < 8.0 {
			continue
		}
		if fontSize > 50.0 {
			fontSize = 50.0
		}
		
		// Create isolated rendering context
		macro := op.Record(gtx.Ops)
		
		angleRad := float32(fpAngleDeg) * math.Pi / 180.0
		transform := f32.Affine2D{}.
			Rotate(f32.Pt(0, 0), angleRad).
			Offset(f32.Pt(float32(x), float32(y)))
		
		stack := op.Affine(transform).Push(gtx.Ops)
		
		// Use silkscreen color
		textColor := GetLayerColor("F.SilkS")
		paint.ColorOp{Color: textColor}.Add(gtx.Ops)
		
		label := widget.Label{
			Alignment: text.Start,
			MaxLines:  1,
		}
		label.Layout(gtx, shaper, font.Font{}, unit.Sp(fontSize), fp.Reference, op.CallOp{})
		
		stack.Pop()
		call := macro.Stop()
		call.Add(gtx.Ops)
	}
}

// renderFootprintFab renders F.Fab and B.Fab layers
func renderFootprintFab(gtx layout.Context, camera *Camera, board *parser.Board) {
	renderFootprintLayer(gtx, camera, board, "F.Fab", "B.Fab", 0.0)
}

// renderFootprintCourtyards renders F.CrtYd and B.CrtYd layers
func renderFootprintCourtyards(gtx layout.Context, camera *Camera, board *parser.Board) {
	renderFootprintLayer(gtx, camera, board, "F.CrtYd", "B.CrtYd", 0.0)
}

// renderFootprintSilkscreen renders F.SilkS and B.SilkS layers
func renderFootprintSilkscreen(gtx layout.Context, camera *Camera, board *parser.Board, debugRotationOffset float64) {
	renderFootprintLayer(gtx, camera, board, "F.SilkS", "B.SilkS", debugRotationOffset)
}

// renderFootprintLayer renders specific footprint layers
func renderFootprintLayer(gtx layout.Context, camera *Camera, board *parser.Board, frontLayer, backLayer string, debugRotationOffset float64) {
	for _, fp := range board.Footprints {
		if len(fp.Graphics) == 0 {
			continue
		}

		fpAngleDeg := -float64(fp.Position.Angle)
		if (fp.Reference == "J1" || fp.Reference == "J2" || fp.Reference == "U5") && debugRotationOffset != 0.0 {
			fpAngleDeg += debugRotationOffset
		}

		fpAngleRad := fpAngleDeg * math.Pi / 180.0
		cos := math.Cos(fpAngleRad)
		sin := math.Sin(fpAngleRad)

		for _, gr := range fp.Graphics {
			if gr.Layer != frontLayer && gr.Layer != backLayer {
				continue
			}

			lineColor := GetLayerColor(gr.Layer)

			if gr.Type == "line" {
				x1 := gr.Start.X*cos - gr.Start.Y*sin + fp.Position.X
				y1 := gr.Start.X*sin + gr.Start.Y*cos + fp.Position.Y
				x2 := gr.End.X*cos - gr.End.Y*sin + fp.Position.X
				y2 := gr.End.X*sin + gr.End.Y*cos + fp.Position.Y

				sx1, sy1 := camera.BoardToScreen(parser.Position{X: x1, Y: y1})
				sx2, sy2 := camera.BoardToScreen(parser.Position{X: x2, Y: y2})

				strokeWidth := gr.Stroke.Width * camera.Zoom
				if strokeWidth < 1.0 {
					strokeWidth = 1.0
				}

				renderLine(gtx, sx1, sy1, sx2, sy2, strokeWidth, lineColor)
			} else if gr.Type == "circle" {
				// Transform center
				cx := gr.Center.X*cos - gr.Center.Y*sin + fp.Position.X
				cy := gr.Center.X*sin + gr.Center.Y*cos + fp.Position.Y
				
				// Calculate radius from center to end point
				dx := gr.End.X - gr.Center.X
				dy := gr.End.Y - gr.Center.Y
				radius := math.Sqrt(dx*dx + dy*dy) * camera.Zoom
				
				scx, scy := camera.BoardToScreen(parser.Position{X: cx, Y: cy})
				
				if radius < 1.0 {
					radius = 1.0
				}
				
				// Draw circle
				var path clip.Path
				path.Begin(gtx.Ops)
				segments := 64
				for i := 0; i <= segments; i++ {
					angle := float64(i) * 2.0 * math.Pi / float64(segments)
					px := scx + radius*math.Cos(angle)
					py := scy + radius*math.Sin(angle)
					if i == 0 {
						path.MoveTo(f32.Pt(float32(px), float32(py)))
					} else {
						path.LineTo(f32.Pt(float32(px), float32(py)))
					}
				}
				
				stroke := clip.Stroke{
					Path:  path.End(),
					Width: float32(gr.Stroke.Width * camera.Zoom),
				}.Op()
				paint.FillShape(gtx.Ops, lineColor, stroke)
			} else if gr.Type == "arc" {
				// Transform arc points
				sx := gr.Start.X*cos - gr.Start.Y*sin + fp.Position.X
				sy := gr.Start.X*sin + gr.Start.Y*cos + fp.Position.Y
				mx := gr.Center.X*cos - gr.Center.Y*sin + fp.Position.X
				my := gr.Center.X*sin + gr.Center.Y*cos + fp.Position.Y
				ex := gr.End.X*cos - gr.End.Y*sin + fp.Position.X
				ey := gr.End.X*sin + gr.End.Y*cos + fp.Position.Y
				
				ssx, ssy := camera.BoardToScreen(parser.Position{X: sx, Y: sy})
				smx, smy := camera.BoardToScreen(parser.Position{X: mx, Y: my})
				sex, sey := camera.BoardToScreen(parser.Position{X: ex, Y: ey})
				
				// Approximate arc with line segments
				var path clip.Path
				path.Begin(gtx.Ops)
				path.MoveTo(f32.Pt(float32(ssx), float32(ssy)))
				path.LineTo(f32.Pt(float32(smx), float32(smy)))
				path.LineTo(f32.Pt(float32(sex), float32(sey)))
				
				stroke := clip.Stroke{
					Path:  path.End(),
					Width: float32(gr.Stroke.Width * camera.Zoom),
				}.Op()
				paint.FillShape(gtx.Ops, lineColor, stroke)
			}
		}
	}
}

// renderGraphics renders graphic elements
func renderGraphics(gtx layout.Context, camera *Camera, board *parser.Board) {
	// Render lines
	for _, grLine := range board.Graphics.Lines {
		x1, y1 := camera.BoardToScreen(grLine.Start)
		x2, y2 := camera.BoardToScreen(grLine.End)

		strokeWidth := grLine.Stroke.Width * camera.Zoom
		if strokeWidth < 1.0 {
			strokeWidth = 1.0
		}

		lineColor := GetLayerColor(grLine.Layer)
		renderLine(gtx, x1, y1, x2, y2, strokeWidth, lineColor)
	}

	// Render circles
	for _, grCircle := range board.Graphics.Circles {
		// Calculate radius from center to end point
		dx := grCircle.End.X - grCircle.Center.X
		dy := grCircle.End.Y - grCircle.Center.Y
		boardRadius := math.Sqrt(dx*dx + dy*dy)

		cx, cy := camera.BoardToScreen(grCircle.Center)
		radius := boardRadius * camera.Zoom

		if radius < 1.0 {
			radius = 1.0
		}

		circleColor := GetLayerColor(grCircle.Layer)

		// Draw circle outline
		var path clip.Path
		path.Begin(gtx.Ops)

		// Approximate circle with segments
		segments := 64
		for i := 0; i <= segments; i++ {
			angle := float64(i) * 2.0 * math.Pi / float64(segments)
			px := cx + radius*math.Cos(angle)
			py := cy + radius*math.Sin(angle)

			if i == 0 {
				path.MoveTo(f32.Pt(float32(px), float32(py)))
			} else {
				path.LineTo(f32.Pt(float32(px), float32(py)))
			}
		}

		stroke := clip.Stroke{
			Path:  path.End(),
			Width: float32(grCircle.Stroke.Width * camera.Zoom),
		}.Op()

		paint.FillShape(gtx.Ops, circleColor, stroke)
	}

	// Render rectangles
	for _, grRect := range board.Graphics.Rects {
		x1, y1 := camera.BoardToScreen(grRect.Start)
		x2, y2 := camera.BoardToScreen(grRect.End)

		minX := math.Min(x1, x2)
		minY := math.Min(y1, y2)
		maxX := math.Max(x1, x2)
		maxY := math.Max(y1, y2)

		rectColor := GetLayerColor(grRect.Layer)

		rect := clip.Rect{
			Min: image.Pt(int(minX), int(minY)),
			Max: image.Pt(int(maxX), int(maxY)),
		}.Op()

		if grRect.Fill.Type == "solid" {
			paint.FillShape(gtx.Ops, rectColor, rect)
		} else {
			// Draw outline
			var path clip.Path
			path.Begin(gtx.Ops)
			path.MoveTo(f32.Pt(float32(minX), float32(minY)))
			path.LineTo(f32.Pt(float32(maxX), float32(minY)))
			path.LineTo(f32.Pt(float32(maxX), float32(maxY)))
			path.LineTo(f32.Pt(float32(minX), float32(maxY)))
			path.Close()

			stroke := clip.Stroke{
				Path:  path.End(),
				Width: float32(grRect.Stroke.Width * camera.Zoom),
			}.Op()

			paint.FillShape(gtx.Ops, rectColor, stroke)
		}
	}

	// Render polygons
	for _, grPoly := range board.Graphics.Polys {
		if len(grPoly.Points) < 2 {
			continue
		}

		polyColor := GetLayerColor(grPoly.Layer)

		var path clip.Path
		path.Begin(gtx.Ops)

		for i, pt := range grPoly.Points {
			x, y := camera.BoardToScreen(pt)
			if i == 0 {
				path.MoveTo(f32.Pt(float32(x), float32(y)))
			} else {
				path.LineTo(f32.Pt(float32(x), float32(y)))
			}
		}
		path.Close()

		stroke := clip.Stroke{
			Path:  path.End(),
			Width: float32(grPoly.Stroke.Width * camera.Zoom),
		}.Op()

		paint.FillShape(gtx.Ops, polyColor, stroke)
	}

	// Render text
	collection := gofont.Collection()
	shaper := text.NewShaper(text.WithCollection(collection))
	
	for _, grText := range board.Graphics.Texts {
		textColor := GetLayerColor(grText.Layer)
		x, y := camera.BoardToScreen(grText.Position)
		
		// Calculate font size
		fontSize := grText.Size.Height * camera.Zoom
		if fontSize < 6.0 {
			continue // Skip text that's too small
		}
		if fontSize > 100.0 {
			fontSize = 100.0 // Cap maximum size
		}
		
		// Create a new macro to isolate this text rendering
		macro := op.Record(gtx.Ops)
		
		// Apply transformations: rotate then translate
		angleRad := float32(-grText.Angle) * math.Pi / 180.0
		transform := f32.Affine2D{}.
			Rotate(f32.Pt(0, 0), angleRad).
			Offset(f32.Pt(float32(x), float32(y)))
		
		stack := op.Affine(transform).Push(gtx.Ops)
		
		// Set text color
		paint.ColorOp{Color: textColor}.Add(gtx.Ops)
		
		// Render text
		label := widget.Label{
			Alignment: text.Start,
			MaxLines:  10,
		}
		label.Layout(gtx, shaper, font.Font{}, unit.Sp(fontSize), grText.Text, op.CallOp{})
		
		stack.Pop()
		
		// Play back the macro
		call := macro.Stop()
		call.Add(gtx.Ops)
	}
}

// renderFootprintGraphics renders graphics for each footprint with rotation
func renderFootprintGraphics(gtx layout.Context, camera *Camera, board *parser.Board) {
	renderFootprintGraphicsWithDebug(gtx, camera, board, 0.0)
}

// renderFootprintGraphicsWithDebug renders graphics for each footprint with debug rotation offset
func renderFootprintGraphicsWithDebug(gtx layout.Context, camera *Camera, board *parser.Board, debugRotationOffset float64) {
	graphicsColor := color.NRGBA{R: 180, G: 180, B: 180, A: 255} // Silkscreen white

	debugLogged := false // Only log once per frame

	for _, fp := range board.Footprints {
		// Skip if no graphics
		if len(fp.Graphics) == 0 {
			continue
		}

		// Get footprint rotation in radians
		// Negate for Y-flip coordinate system
		fpAngleDeg := -float64(fp.Position.Angle)

		// Apply debug rotation offset for J1, J2, and U5
		if (fp.Reference == "J1" || fp.Reference == "J2" || fp.Reference == "U5") && debugRotationOffset != 0.0 {
			fpAngleDeg += debugRotationOffset
			if !debugLogged && fp.Reference == "J1" {
				fmt.Printf("[DEBUG] %s Graphics Rotation: fpAngle=%.1f° debug=%.1f° total=%.1f°\n",
					fp.Reference, -float64(fp.Position.Angle), debugRotationOffset, fpAngleDeg)
				debugLogged = true
			}
		}

		fpAngleRad := fpAngleDeg * math.Pi / 180.0
		cos := math.Cos(fpAngleRad)
		sin := math.Sin(fpAngleRad)

		// Render each graphic element
		for _, gr := range fp.Graphics {
			if gr.Type == "line" {
				// Rotate and translate start point
				startX := gr.Start.X*cos - gr.Start.Y*sin + fp.Position.X
				startY := gr.Start.X*sin + gr.Start.Y*cos + fp.Position.Y
				
				// Rotate and translate end point
				endX := gr.End.X*cos - gr.End.Y*sin + fp.Position.X
				endY := gr.End.X*sin + gr.End.Y*cos + fp.Position.Y
				
				// Convert to screen coordinates
				sx1, sy1 := camera.BoardToScreen(parser.Position{X: startX, Y: startY})
				sx2, sy2 := camera.BoardToScreen(parser.Position{X: endX, Y: endY})
				
				strokeWidth := gr.Stroke.Width * camera.Zoom
				if strokeWidth < 1.0 {
					strokeWidth = 1.0
				}
				
				renderLine(gtx, sx1, sy1, sx2, sy2, strokeWidth, graphicsColor)
			}
		}
	}
}

// Layer-aware rendering functions

func renderTracksWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	for _, track := range board.Tracks {
		if !config.IsVisible(track.Layer) {
			continue
		}
		x1, y1 := camera.BoardToScreen(track.Start)
		x2, y2 := camera.BoardToScreen(track.End)
		strokeWidth := track.Width * camera.Zoom
		if strokeWidth < 1.0 {
			strokeWidth = 1.0
		}
		trackColor := GetLayerColor(track.Layer)
		renderLine(gtx, x1, y1, x2, y2, strokeWidth, trackColor)
	}
}

func renderViasWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	// Vias are visible if any copper layer is visible
	hasVisibleCopper := config.IsVisible("F.Cu") || config.IsVisible("B.Cu") ||
		config.IsVisible("In1.Cu") || config.IsVisible("In2.Cu")
	if !hasVisibleCopper {
		return
	}
	renderVias(gtx, camera, board)
}

func renderPadsWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, debugRotationOffset float64, config *LayerConfig) {
	for _, fp := range board.Footprints {
		for _, pad := range fp.Pads {
			// Check if any of the pad's layers are visible
			visible := false
			for _, layer := range pad.Layers {
				if config.IsVisible(layer) {
					visible = true
					break
				}
			}
			if !visible {
				continue
			}

			// Get absolute pad position in board coordinates
			absPos := fp.TransformPosition(pad.Position)
			sx, sy := camera.BoardToScreen(absPos)

			// Calculate pad rotation
			totalAngle := -float64(pad.Position.Angle)
			if (fp.Reference == "J1" || fp.Reference == "J2" || fp.Reference == "U5") && debugRotationOffset != 0.0 {
				totalAngle += debugRotationOffset
			}
			radians := totalAngle * math.Pi / 180.0

			// Calculate pad size in screen pixels
			width := pad.Size.Width * camera.Zoom
			height := pad.Size.Height * camera.Zoom

			// Enforce minimum size while preserving aspect ratio
			if width < 2.0 || height < 2.0 {
				aspectRatio := pad.Size.Width / pad.Size.Height
				if width < height {
					width = 2.0
					height = width / aspectRatio
				} else {
					height = 2.0
					width = height * aspectRatio
				}
			}

			// Use KiCad colors
			padColor := ColorPadTH
			if pad.Type == "smd" {
				padColor = ColorPadSMD
			}

			// Render pad based on shape
			switch pad.Shape {
			case "circle":
				renderCirclePad(gtx, sx, sy, width, height, padColor, padColor)
			case "oval":
				aspectRatio := width / height
				if aspectRatio > 0.95 && aspectRatio < 1.05 {
					renderCirclePad(gtx, sx, sy, width, height, padColor, padColor)
				} else {
					renderRotatedRRect(gtx, sx, sy, width, height, radians,
						math.Min(width, height)*0.5, padColor, padColor)
				}
			case "roundrect":
				cornerRadius := math.Min(width, height) * 0.25
				renderRotatedRRect(gtx, sx, sy, width, height, radians,
					cornerRadius, padColor, padColor)
			case "rect":
				renderRotatedRect(gtx, sx, sy, width, height, radians, padColor, padColor)
			default:
				renderRotatedRect(gtx, sx, sy, width, height, radians, padColor, padColor)
			}

			// Render drill hole if through-hole
			if pad.Drill > 0 {
				drillRadius := pad.Drill / 2.0 * camera.Zoom
				if drillRadius < 1.0 {
					drillRadius = 1.0
				}
				renderCircle(gtx, sx, sy, drillRadius, ColorDrill)
			}
		}
	}
}

func renderZonesWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	for _, zone := range board.Zones {
		if !config.IsVisible(zone.Layer) {
			continue
		}
		zoneColor := GetLayerColor(zone.Layer)
		zoneColor.A = 180
		for _, fill := range zone.Fills {
			if len(fill) < 3 {
				continue
			}
			var path clip.Path
			path.Begin(gtx.Ops)
			for i, pt := range fill {
				x, y := camera.BoardToScreen(pt)
				if i == 0 {
					path.MoveTo(f32.Pt(float32(x), float32(y)))
				} else {
					path.LineTo(f32.Pt(float32(x), float32(y)))
				}
			}
			path.Close()
			paint.FillShape(gtx.Ops, zoneColor, clip.Outline{Path: path.End()}.Op())
		}
	}
}

func renderGraphicsWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	for _, line := range board.Graphics.Lines {
		if !config.IsVisible(line.Layer) {
			continue
		}
		x1, y1 := camera.BoardToScreen(line.Start)
		x2, y2 := camera.BoardToScreen(line.End)
		strokeWidth := line.Stroke.Width * camera.Zoom
		if strokeWidth < 1.0 {
			strokeWidth = 1.0
		}
		lineColor := GetLayerColor(line.Layer)
		renderLine(gtx, x1, y1, x2, y2, strokeWidth, lineColor)
	}
	
	// Render arcs
	for _, arc := range board.Graphics.Arcs {
		if !config.IsVisible(arc.Layer) {
			continue
		}
		strokeWidth := arc.Stroke.Width * camera.Zoom
		if strokeWidth < 1.0 {
			strokeWidth = 1.0
		}
		arcColor := GetLayerColor(arc.Layer)
		renderArc(gtx, camera, arc, strokeWidth, arcColor)
	}

	for _, circle := range board.Graphics.Circles {
		if !config.IsVisible(circle.Layer) {
			continue
		}
		cx, cy := camera.BoardToScreen(circle.Center)
		dx := circle.End.X - circle.Center.X
		dy := circle.End.Y - circle.Center.Y
		radius := math.Sqrt(dx*dx+dy*dy) * camera.Zoom
		if radius < 1.0 {
			radius = 1.0
		}
		strokeWidth := circle.Stroke.Width * camera.Zoom
		if strokeWidth < 1.0 {
			strokeWidth = 1.0
		}
		circleColor := GetLayerColor(circle.Layer)
		
		// Render circle outline
		var path clip.Path
		path.Begin(gtx.Ops)
		path.Move(f32.Pt(float32(cx+radius), float32(cy)))
		for i := 0; i <= 64; i++ {
			angle := float64(i) * 2.0 * math.Pi / 64.0
			x := cx + radius*math.Cos(angle)
			y := cy + radius*math.Sin(angle)
			path.LineTo(f32.Pt(float32(x), float32(y)))
		}
		path.Close()
		stroke := clip.Stroke{Path: path.End(), Width: float32(strokeWidth)}.Op()
		paint.FillShape(gtx.Ops, circleColor, stroke)
	}
}

// renderArc renders an arc
func renderArc(gtx layout.Context, camera *Camera, arc parser.GrArc, strokeWidth float64, color color.NRGBA) {
	// Get arc parameters
	cx, cy := camera.BoardToScreen(arc.Start)
	ex, ey := camera.BoardToScreen(arc.End)
	mx, my := camera.BoardToScreen(arc.Mid)
	
	// Approximate arc with line segments
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(float32(cx), float32(cy)))
	
	// Simple approximation: draw through midpoint
	path.LineTo(f32.Pt(float32(mx), float32(my)))
	path.LineTo(f32.Pt(float32(ex), float32(ey)))
	
	stroke := clip.Stroke{Path: path.End(), Width: float32(strokeWidth)}.Op()
	paint.FillShape(gtx.Ops, color, stroke)
}

func renderFootprintSilkscreenWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, debugRotationOffset float64, config *LayerConfig) {
	if !config.IsVisible("F.SilkS") && !config.IsVisible("B.SilkS") {
		return
	}
	renderFootprintSilkscreen(gtx, camera, board, debugRotationOffset)
}

func renderFootprintFabWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	if !config.IsVisible("F.Fab") && !config.IsVisible("B.Fab") {
		return
	}
	renderFootprintFab(gtx, camera, board)
}

func renderFootprintTextWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, debugRotationOffset float64, config *LayerConfig) {
	// Text can be on various layers, check in the actual render function
	renderFootprintText(gtx, camera, board, debugRotationOffset)
}

func renderFootprintCourtyardsWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	if !config.IsVisible("F.CrtYd") && !config.IsVisible("B.CrtYd") {
		return
	}
	renderFootprintCourtyards(gtx, camera, board)
}

func renderFootprintMaskWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	if !config.IsVisible("F.Mask") && !config.IsVisible("B.Mask") {
		return
	}
	renderFootprintMask(gtx, camera, board)
}

func renderFootprintPasteWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	if !config.IsVisible("F.Paste") && !config.IsVisible("B.Paste") {
		return
	}
	renderFootprintPaste(gtx, camera, board)
}

func renderFootprintAdhesiveWithConfig(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	if !config.IsVisible("F.Adhes") && !config.IsVisible("B.Adhes") {
		return
	}
	renderFootprintAdhesive(gtx, camera, board)
}

// renderBoardSubstrate renders the PCB substrate (board material) as background
func renderBoardSubstrate(gtx layout.Context, camera *Camera, board *parser.Board, config *LayerConfig) {
	// Get board bounding box
	bbox := board.GetBoundingBox()
	if bbox.IsEmpty() {
		return
	}
	
	// Use theme-specific substrate color
	substrateColor := GetSubstrateColor()
	
	x1, y1 := camera.BoardToScreen(bbox.Min)
	x2, y2 := camera.BoardToScreen(bbox.Max)
	
	// Create rectangle path for substrate background
	var path clip.Path
	path.Begin(gtx.Ops)
	path.MoveTo(f32.Pt(float32(x1), float32(y1)))
	path.LineTo(f32.Pt(float32(x2), float32(y1)))
	path.LineTo(f32.Pt(float32(x2), float32(y2)))
	path.LineTo(f32.Pt(float32(x1), float32(y2)))
	path.Close()
	
	paint.FillShape(gtx.Ops, substrateColor, clip.Outline{Path: path.End()}.Op())
}
