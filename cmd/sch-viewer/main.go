package main

import (
	"fmt"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
	schrenderer "github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic/renderer"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("KiCad Schematic Viewer"))
		w.Option(app.Size(unit.Dp(1200), unit.Dp(800)))

		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

type ViewerApp struct {
	window   *app.Window
	theme    *material.Theme
	explorer *explorer.Explorer

	schematic  *schematic.Schematic
	camera     *renderer.Camera
	colorTheme schrenderer.Theme
	colors     *schrenderer.SchematicColors

	// UI widgets
	openFileBtn widget.Clickable
	themeBtn    widget.Clickable
	fitBtn      widget.Clickable

	// Mouse interaction
	lastPointerPos  f32.Point
	isDragging      bool
	dragStartPos    f32.Point

	filepath string
}

func run(w *app.Window) error {
	viewer := &ViewerApp{
		window:     w,
		theme:      material.NewTheme(),
		explorer:   explorer.NewExplorer(w),
		camera:     renderer.NewCamera(1200, 800),
		colorTheme: schrenderer.ThemeLight,
	}
	viewer.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	viewer.colors = schrenderer.GetSchematicColors(viewer.colorTheme)

	// Disable Y-axis inversion for schematics (KiCad schematics use screen-like coordinates)
	viewer.camera.InvertY = false

	// Load file from command line if provided
	if len(os.Args) > 1 {
		viewer.loadSchematic(os.Args[1])
	}

	var ops op.Ops

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err

		case app.FrameEvent:
			gtx := layout.Context{
				Ops:         &ops,
				Constraints: layout.Exact(e.Size),
				Metric:      e.Metric,
				Now:         e.Now,
				Source:      e.Source,
			}

			viewer.camera.UpdateScreenSize(e.Size.X, e.Size.Y)
			viewer.handleInput(gtx)
			viewer.layout(gtx)
			e.Frame(&ops)
		}
	}
}

func (v *ViewerApp) handleInput(gtx layout.Context) {
	// Handle button clicks
	if v.openFileBtn.Clicked(gtx) {
		v.openFilePicker()
	}

	if v.themeBtn.Clicked(gtx) {
		v.toggleTheme()
	}

	if v.fitBtn.Clicked(gtx) {
		v.fitToView()
	}

	// Handle keyboard shortcuts
	for {
		ev, ok := gtx.Event(key.Filter{Name: "O", Required: key.ModShortcut})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			v.openFilePicker()
		}
	}

	// Ctrl+T for theme toggle
	for {
		ev, ok := gtx.Event(key.Filter{Name: "T", Required: key.ModShortcut})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			v.toggleTheme()
		}
	}

	// F for fit to view
	for {
		ev, ok := gtx.Event(key.Filter{Name: "F"})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			v.fitToView()
		}
	}

	// Q or Escape to quit
	for {
		ev, ok := gtx.Event(key.Filter{Name: "Q"})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			os.Exit(0)
		}
	}

	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			os.Exit(0)
		}
	}

	// Handle mouse events
	for {
		ev, ok := gtx.Event(
			pointer.Filter{
				Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Scroll,
			},
		)
		if !ok {
			break
		}

		if pe, ok := ev.(pointer.Event); ok {
			switch pe.Kind {
			case pointer.Press:
				if pe.Buttons == pointer.ButtonPrimary {
					v.isDragging = true
					v.dragStartPos = pe.Position
					v.lastPointerPos = pe.Position
				}

			case pointer.Drag:
				if v.isDragging && pe.Buttons == pointer.ButtonPrimary {
					deltaX := float64(pe.Position.X - v.lastPointerPos.X)
					deltaY := float64(pe.Position.Y - v.lastPointerPos.Y)
					v.camera.Pan(deltaX, deltaY)
					v.lastPointerPos = pe.Position
					v.window.Invalidate()
				}

			case pointer.Release:
				v.isDragging = false

			case pointer.Scroll:
				// Zoom at cursor position
				zoomFactor := 1.0 + float64(pe.Scroll.Y)*0.1
				v.camera.ZoomAt(float64(pe.Position.X), float64(pe.Position.Y), zoomFactor)
				v.window.Invalidate()
			}
		}
	}
}

func (v *ViewerApp) openFilePicker() {
	go func() {
		// Use empty string to allow all files - some platforms have issues with extension filters
		file, err := v.explorer.ChooseFile("")
		if err != nil {
			if err != explorer.ErrUserDecline {
				log.Printf("File picker error: %v", err)
			}
			return
		}
		defer file.Close()

		if f, ok := file.(*os.File); ok {
			log.Printf("Selected file: %s", f.Name())
			v.loadSchematic(f.Name())
			v.window.Invalidate()
		}
	}()
}

func (v *ViewerApp) loadSchematic(filepath string) {
	sch, err := schematic.ParseFile(filepath)
	if err != nil {
		log.Printf("Error loading schematic: %v", err)
		return
	}

	v.schematic = sch
	v.filepath = filepath
	v.window.Option(app.Title("KiCad Schematic Viewer - " + filepath))

	// Fit to view after loading
	v.fitToView()

	log.Printf("✓ Loaded schematic: %s", filepath)
	log.Printf("  Version: %d", sch.Version)
	log.Printf("  Generator: %s v%s", sch.Generator, sch.GeneratorVer)
	log.Printf("  Components: %d", len(sch.Symbols))
	log.Printf("  Wires: %d", len(sch.Wires))
	log.Printf("  Labels: %d", len(sch.Labels)+len(sch.GlobalLabels)+len(sch.HierLabels))
}

func (v *ViewerApp) toggleTheme() {
	if v.colorTheme == schrenderer.ThemeLight {
		v.colorTheme = schrenderer.ThemeDark
	} else {
		v.colorTheme = schrenderer.ThemeLight
	}
	v.colors = schrenderer.GetSchematicColors(v.colorTheme)
	log.Printf("Theme switched to: %s", v.colorTheme.String())
	v.window.Invalidate()
}

func (v *ViewerApp) fitToView() {
	if v.schematic == nil {
		return
	}

	bbox := v.schematic.GetBoundingBox()
	if bbox.IsEmpty() {
		log.Println("Schematic has no content to fit")
		return
	}

	v.camera.Fit(bbox)
	log.Printf("Fit to view: bbox (%.2f, %.2f) to (%.2f, %.2f)",
		bbox.Min.X, bbox.Min.Y, bbox.Max.X, bbox.Max.Y)
	v.window.Invalidate()
}

func (v *ViewerApp) layout(gtx layout.Context) layout.Dimensions {
	// Fill background
	paint.Fill(gtx.Ops, v.colors.Background)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Toolbar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return v.layoutToolbar(gtx)
		}),

		// Canvas
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return v.layoutCanvas(gtx)
		}),
	)
}

func (v *ViewerApp) layoutToolbar(gtx layout.Context) layout.Dimensions {
	inset := layout.Inset{Top: 8, Bottom: 8, Left: 8, Right: 8}

	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
			// Left side buttons
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(v.theme, &v.openFileBtn, "Open (Ctrl+O)")
						return btn.Layout(gtx)
					}),

					layout.Rigid(layout.Spacer{Width: 8}.Layout),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						themeName := v.colorTheme.String()
						btn := material.Button(v.theme, &v.themeBtn, "Theme: "+themeName+" (Ctrl+T)")
						return btn.Layout(gtx)
					}),

					layout.Rigid(layout.Spacer{Width: 8}.Layout),

					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(v.theme, &v.fitBtn, "Fit (F)")
						return btn.Layout(gtx)
					}),
				)
			}),

			// Right side info
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if v.schematic == nil {
					label := material.Body1(v.theme, "No schematic loaded")
					return label.Layout(gtx)
				}

				info := fmt.Sprintf("Components: %d | Wires: %d | Zoom: %.1fx",
					len(v.schematic.Symbols),
					len(v.schematic.Wires),
					v.camera.Zoom/10.0)
				label := material.Body1(v.theme, info)
				return label.Layout(gtx)
			}),
		)
	})
}

func (v *ViewerApp) layoutCanvas(gtx layout.Context) layout.Dimensions {
	if v.schematic == nil {
		// Show welcome message
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					title := material.H4(v.theme, "KiCad Schematic Viewer")
					return title.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: 16}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					msg := material.Body1(v.theme, "Click 'Open' or press Ctrl+O to select a schematic")
					return msg.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: 8}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					msg := material.Body2(v.theme, "Or launch with: sch-viewer <file.kicad_sch>")
					return msg.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: 16}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					msg := material.Body2(v.theme, "Controls: Left-drag to pan | Scroll to zoom | F to fit | Ctrl+T toggle theme | Q or Esc to quit")
					return msg.Layout(gtx)
				}),
			)
		})
	}

	// Render schematic
	v.renderSchematic(gtx)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (v *ViewerApp) renderSchematic(gtx layout.Context) {
	// Render the schematic using the schematic renderer
	schrenderer.RenderSchematic(gtx, v.camera, v.schematic, v.colors)
}
