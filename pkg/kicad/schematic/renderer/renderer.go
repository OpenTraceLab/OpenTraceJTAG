package renderer

import (
	"gioui.org/layout"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
)

// RenderSchematic renders the entire schematic
func RenderSchematic(gtx layout.Context, camera *renderer.Camera, sch *schematic.Schematic, colors *SchematicColors) {
	if sch == nil {
		return
	}

	// Render in order (back to front)
	// This ensures proper layering of elements

	// 1. Graphical elements (polylines, images, text)
	RenderPolylines(gtx, camera, sch.Polylines, colors)
	// TODO: RenderImages(gtx, camera, sch.Images, colors)
	// TODO: RenderTexts(gtx, camera, sch.Texts, colors)

	// 2. Hierarchical sheets (background)
	// TODO: RenderSheets(gtx, camera, sch.Sheets, colors)

	// 3. Buses (drawn before wires for proper layering)
	RenderBuses(gtx, camera, sch.Buses, colors)
	RenderBusEntries(gtx, camera, sch.BusEntries, colors)

	// 4. Wires
	RenderWires(gtx, camera, sch.Wires, colors)

	// 5. Junctions and no-connects (on top of wires)
	RenderJunctions(gtx, camera, sch.Junctions, colors)
	RenderNoConnects(gtx, camera, sch.NoConnects, colors)

	// 6. Symbols (before labels so labels are on top)
	RenderSymbols(gtx, camera, sch, colors)

	// 7. Labels (on top of everything)
	RenderLabels(gtx, camera, sch.Labels, colors)
	RenderGlobalLabels(gtx, camera, sch.GlobalLabels, colors)
	RenderHierLabels(gtx, camera, sch.HierLabels, colors)
}

// RenderSchematicWithOptions renders the schematic with additional options
func RenderSchematicWithOptions(gtx layout.Context, camera *renderer.Camera, sch *schematic.Schematic, colors *SchematicColors, opts RenderOptions) {
	if sch == nil {
		return
	}

	// Render elements based on options
	if opts.ShowPolylines {
		RenderPolylines(gtx, camera, sch.Polylines, colors)
	}

	if opts.ShowBuses {
		RenderBuses(gtx, camera, sch.Buses, colors)
		RenderBusEntries(gtx, camera, sch.BusEntries, colors)
	}

	if opts.ShowWires {
		RenderWires(gtx, camera, sch.Wires, colors)
	}

	if opts.ShowJunctions {
		RenderJunctions(gtx, camera, sch.Junctions, colors)
	}

	if opts.ShowNoConnects {
		RenderNoConnects(gtx, camera, sch.NoConnects, colors)
	}

	if opts.ShowSymbols {
		RenderSymbols(gtx, camera, sch, colors)
	}

	if opts.ShowLabels {
		RenderLabels(gtx, camera, sch.Labels, colors)
		RenderGlobalLabels(gtx, camera, sch.GlobalLabels, colors)
		RenderHierLabels(gtx, camera, sch.HierLabels, colors)
	}
}

// RenderOptions controls what elements are rendered
type RenderOptions struct {
	ShowWires      bool
	ShowBuses      bool
	ShowJunctions  bool
	ShowNoConnects bool
	ShowLabels     bool
	ShowSymbols    bool
	ShowText       bool
	ShowSheets     bool
	ShowPolylines  bool
}

// DefaultRenderOptions returns default rendering options (all enabled)
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		ShowWires:      true,
		ShowBuses:      true,
		ShowJunctions:  true,
		ShowNoConnects: true,
		ShowLabels:     true,
		ShowSymbols:    true,
		ShowText:       true,
		ShowSheets:     true,
		ShowPolylines:  true,
	}
}
