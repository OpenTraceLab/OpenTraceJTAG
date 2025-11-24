package main

import (
	"fmt"
	"image/color"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

func main() {
	fmt.Println("=== Gio v0.9 Keyboard Test ===")
	fmt.Println("Press any key - should print below")
	
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Keyboard Test"))
		w.Option(app.Size(unit.Dp(800), unit.Dp(600)))
		
		var ops op.Ops
		
		for {
			e := w.Event()
			
			switch e := e.(type) {
			case app.DestroyEvent:
				return
				
			case app.FrameEvent:
				ops.Reset()
				
				gtx := layout.Context{
					Ops:         &ops,
					Constraints: layout.Exact(e.Size),
					Metric:      e.Metric,
					Now:         e.Now,
					Source:      e.Source,
				}
				
				// Try to read key events using gtx.Event
				for {
					ev, ok := gtx.Event(key.Filter{})
					if !ok {
						break
					}
					
					if ke, ok := ev.(key.Event); ok {
						fmt.Printf("[KEY] %q State=%v\n", ke.Name, ke.State)
					}
				}
				
				// Green background
				paint.Fill(&ops, color.NRGBA{R: 0x20, G: 0x80, B: 0x20, A: 0xFF})
				
				e.Frame(&ops)
			}
		}
	}()
	
	app.Main()
}
