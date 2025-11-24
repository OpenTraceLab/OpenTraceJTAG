package main

import (
	"image"
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type App struct {
	zoom float32
}

func main() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Gio Affine Transform Color Bug (Minimal)"),
			app.Size(unit.Dp(900), unit.Dp(350)),
		)
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	var ops op.Ops
	appState := &App{zoom: 1.0}

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err

		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// Zoom controls: '+' and '-'
			for {
				ev, ok := gtx.Event(key.Filter{Name: "+"})
				if !ok {
					break
				}
				if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
					appState.zoom *= 1.2
				}
			}
			for {
				ev, ok := gtx.Event(key.Filter{Name: "-"})
				if !ok {
					break
				}
				if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
					appState.zoom /= 1.2
					if appState.zoom < 0.5 {
						appState.zoom = 0.5
					}
				}
			}

			// register for events
			event.Op(gtx.Ops, appState)

			appState.Layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (a *App) Layout(gtx layout.Context) layout.Dimensions {
	// Background
	paint.Fill(gtx.Ops, color.NRGBA{R: 240, G: 240, B: 240, A: 255})

	// Left: reference (zoom = 1.0)
	// Right: same shapes under affine scale (zoom = a.zoom)
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawShapes(gtx, 1.0)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawShapes(gtx, a.zoom)
		}),
	)
}

func drawShapes(gtx layout.Context, zoom float32) layout.Dimensions {
	if zoom != 1.0 {
		defer op.Affine(
			f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(zoom, zoom)),
		).Push(gtx.Ops).Pop()
	}

	colors := []color.NRGBA{
		{R: 30, G: 35, B: 40, A: 255},   // dark gray
		{R: 63, G: 81, B: 181, A: 255},  // blue
		{R: 76, G: 175, B: 80, A: 255},  // green
		{R: 244, G: 67, B: 54, A: 255},  // red
	}

	size := 80
	spacing := 10

	// Rectangles
	for i, col := range colors {
		x := i * (size + spacing)
		paint.FillShape(gtx.Ops, col, clip.Rect{
			Min: image.Pt(x, 0),
			Max: image.Pt(x+size, size),
		}.Op())
	}

	// Circle
	cx := len(colors) * (size + spacing)
	paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 193, B: 7, A: 255},
		clip.Ellipse{
			Min: image.Pt(cx, 0),
			Max: image.Pt(cx+size, size),
		}.Op(gtx.Ops),
	)

	return layout.Dimensions{
		Size: image.Pt((len(colors)+1)*(size+spacing), size),
	}
}

