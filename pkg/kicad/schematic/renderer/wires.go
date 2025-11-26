package renderer

import (
	"image"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
)

// RenderWires renders all wires in the schematic
func RenderWires(gtx layout.Context, camera *renderer.Camera, wires []schematic.Wire, colors *SchematicColors) {
	if len(wires) == 0 {
		return
	}

	// Wire width in screen pixels
	const wireWidth = 2.0

	for _, wire := range wires {
		if len(wire.Points) < 2 {
			continue
		}

		// Build path for this wire
		var path clip.Path
		path.Begin(gtx.Ops)

		// Move to first point
		x0, y0 := camera.WorldToScreen(wire.Points[0])
		path.MoveTo(f32.Pt(float32(x0), float32(y0)))

		// Draw lines to remaining points
		for i := 1; i < len(wire.Points); i++ {
			x, y := camera.WorldToScreen(wire.Points[i])
			path.LineTo(f32.Pt(float32(x), float32(y)))
		}

		// Stroke the path
		paint.FillShape(gtx.Ops, colors.Wire, clip.Stroke{
			Path:  path.End(),
			Width: wireWidth,
		}.Op())
	}
}

// RenderBuses renders all buses in the schematic
func RenderBuses(gtx layout.Context, camera *renderer.Camera, buses []schematic.Bus, colors *SchematicColors) {
	if len(buses) == 0 {
		return
	}

	// Bus width in screen pixels (thicker than wires)
	const busWidth = 4.0

	for _, bus := range buses {
		if len(bus.Points) < 2 {
			continue
		}

		// Build path for this bus
		var path clip.Path
		path.Begin(gtx.Ops)

		// Move to first point
		x0, y0 := camera.WorldToScreen(bus.Points[0])
		path.MoveTo(f32.Pt(float32(x0), float32(y0)))

		// Draw lines to remaining points
		for i := 1; i < len(bus.Points); i++ {
			x, y := camera.WorldToScreen(bus.Points[i])
			path.LineTo(f32.Pt(float32(x), float32(y)))
		}

		// Stroke the path
		paint.FillShape(gtx.Ops, colors.Bus, clip.Stroke{
			Path:  path.End(),
			Width: busWidth,
		}.Op())
	}
}

// RenderJunctions renders all wire junctions in the schematic
func RenderJunctions(gtx layout.Context, camera *renderer.Camera, junctions []schematic.Junction, colors *SchematicColors) {
	if len(junctions) == 0 {
		return
	}

	// Junction diameter in screen pixels
	const junctionDiameter = 8.0
	const junctionRadius = junctionDiameter / 2.0

	for _, junction := range junctions {
		x, y := camera.WorldToScreen(junction.Position)

		// Draw filled circle at junction point
		// Use clip.Ellipse for proper circle
		paint.FillShape(gtx.Ops, colors.Junction,
			clip.Ellipse{
				Min: image.Pt(int(x-junctionRadius), int(y-junctionRadius)),
				Max: image.Pt(int(x+junctionRadius), int(y+junctionRadius)),
			}.Op(gtx.Ops))
	}
}

// RenderNoConnects renders all no-connect markers in the schematic
func RenderNoConnects(gtx layout.Context, camera *renderer.Camera, noConnects []schematic.NoConnect, colors *SchematicColors) {
	if len(noConnects) == 0 {
		return
	}

	// No-connect marker size in screen pixels
	const markerSize = 10.0
	const halfSize = markerSize / 2.0
	const lineWidth = 2.0

	for _, nc := range noConnects {
		x, y := camera.WorldToScreen(nc.Position)

		// Draw X marker (two diagonal lines)
		var path clip.Path
		path.Begin(gtx.Ops)

		// First diagonal (top-left to bottom-right)
		path.MoveTo(f32.Pt(float32(x-halfSize), float32(y-halfSize)))
		path.LineTo(f32.Pt(float32(x+halfSize), float32(y+halfSize)))

		// Second diagonal (top-right to bottom-left)
		path.MoveTo(f32.Pt(float32(x+halfSize), float32(y-halfSize)))
		path.LineTo(f32.Pt(float32(x-halfSize), float32(y+halfSize)))

		// Stroke the X
		paint.FillShape(gtx.Ops, colors.NoConnect, clip.Stroke{
			Path:  path.End(),
			Width: lineWidth,
		}.Op())
	}
}

// RenderBusEntries renders all bus entry markers in the schematic
func RenderBusEntries(gtx layout.Context, camera *renderer.Camera, entries []schematic.BusEntry, colors *SchematicColors) {
	if len(entries) == 0 {
		return
	}

	const lineWidth = 2.0

	for _, entry := range entries {
		// Bus entry is a diagonal line from position to position + size
		x0, y0 := camera.WorldToScreen(entry.Position)
		x1, y1 := camera.WorldToScreen(schematic.Position{
			X: entry.Position.X + entry.Size.Width,
			Y: entry.Position.Y + entry.Size.Height,
		})

		// Draw diagonal line
		var path clip.Path
		path.Begin(gtx.Ops)
		path.MoveTo(f32.Pt(float32(x0), float32(y0)))
		path.LineTo(f32.Pt(float32(x1), float32(y1)))

		paint.FillShape(gtx.Ops, colors.Bus, clip.Stroke{
			Path:  path.End(),
			Width: lineWidth,
		}.Op())
	}
}

// RenderPolylines renders graphical polylines in the schematic
func RenderPolylines(gtx layout.Context, camera *renderer.Camera, polylines []schematic.Polyline, colors *SchematicColors) {
	if len(polylines) == 0 {
		return
	}

	const defaultWidth = 1.0

	for _, poly := range polylines {
		if len(poly.Points) < 2 {
			continue
		}

		// Build path
		var path clip.Path
		path.Begin(gtx.Ops)

		// Move to first point
		x0, y0 := camera.WorldToScreen(poly.Points[0])
		path.MoveTo(f32.Pt(float32(x0), float32(y0)))

		// Draw lines to remaining points
		for i := 1; i < len(poly.Points); i++ {
			x, y := camera.WorldToScreen(poly.Points[i])
			path.LineTo(f32.Pt(float32(x), float32(y)))
		}

		// Use stroke width from polyline if available
		var width float32 = defaultWidth
		if poly.Stroke.Width > 0 {
			width = float32(poly.Stroke.Width * camera.Zoom)
			if width < 1.0 {
				width = 1.0
			}
		}

		// Stroke the path
		paint.FillShape(gtx.Ops, colors.Wire, clip.Stroke{
			Path:  path.End(),
			Width: width,
		}.Op())
	}
}
