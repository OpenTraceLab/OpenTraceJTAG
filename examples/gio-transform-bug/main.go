package main

import (
	"fmt"
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
	"gioui.org/widget/material"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Gio Transform Color Bug"))
		w.Option(app.Size(unit.Dp(800), unit.Dp(600)))
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

type App struct {
	zoom float32
}

func run(w *app.Window) error {
	th := material.NewTheme()
	a := &App{zoom: 1.0}
	
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			
			// Handle keyboard
			for {
				ev, ok := gtx.Event(key.Filter{Name: "+"})
				if !ok {
					break
				}
				if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
					a.zoom *= 1.2
					if a.zoom > 3.0 {
						a.zoom = 3.0
					}
				}
			}
			
			for {
				ev, ok := gtx.Event(key.Filter{Name: "-"})
				if !ok {
					break
				}
				if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
					a.zoom /= 1.2
					if a.zoom < 0.5 {
						a.zoom = 0.5
					}
				}
			}
			
			for {
				ev, ok := gtx.Event(key.Filter{Name: "R"})
				if !ok {
					break
				}
				if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
					a.zoom = 1.0
				}
			}
			
			// Render
			a.Layout(gtx, th)
			e.Frame(gtx.Ops)
		}
	}
}

func (a *App) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Background
	paint.Fill(gtx.Ops, color.NRGBA{R: 240, G: 240, B: 240, A: 255})
	
	// Register for keyboard events
	event.Op(gtx.Ops, a)
	
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Instructions
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(20), Left: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(material.H6(th, "Gio Affine Transform Color Bug Demo").Layout),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(material.Body1(th, "Press + to zoom in, - to zoom out, R to reset").Layout),
					layout.Rigid(material.Body2(th, "Notice: Colors get paler when zoom != 1.0").Layout),
					layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
					layout.Rigid(material.Caption(th, fmt.Sprintf("Current zoom: %.2f", a.zoom)).Layout),
				)
			})
		}),
		
		// Demo area
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
					// Left: Without transform (reference)
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Right: unit.Dp(50)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(material.Body2(th, "Without Transform (zoom=1.0)").Layout),
								layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return a.drawShapes(gtx, false)
								}),
							)
						})
					}),
					
					// Right: With transform
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(material.Body2(th, "With Transform (current zoom)").Layout),
							layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return a.drawShapes(gtx, true)
							}),
						)
					}),
				)
			})
		}),
	)
}

func (a *App) drawShapes(gtx layout.Context, applyTransform bool) layout.Dimensions {
	if applyTransform && a.zoom != 1.0 {
		defer op.Affine(f32.Affine2D{}.
			Scale(f32.Point{}, f32.Pt(a.zoom, a.zoom))).Push(gtx.Ops).Pop()
	}
	
	// Draw colored rectangles
	colors := []color.NRGBA{
		{R: 30, G: 35, B: 40, A: 255},    // Dark gray
		{R: 63, G: 81, B: 181, A: 255},   // Blue
		{R: 76, G: 175, B: 80, A: 255},   // Green
		{R: 244, G: 67, B: 54, A: 255},   // Red
	}
	
	size := 80
	spacing := 10
	
	for i, col := range colors {
		x := i * (size + spacing)
		paint.FillShape(gtx.Ops, col, clip.Rect{
			Min: image.Pt(x, 0),
			Max: image.Pt(x+size, size),
		}.Op())
	}
	
	// Draw circle
	paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 193, B: 7, A: 255}, 
		clip.Ellipse{
			Min: image.Pt(len(colors)*(size+spacing), 0),
			Max: image.Pt(len(colors)*(size+spacing)+size, size),
		}.Op(gtx.Ops))
	
	return layout.Dimensions{
		Size: image.Pt((len(colors)+1)*(size+spacing), size),
	}
}
