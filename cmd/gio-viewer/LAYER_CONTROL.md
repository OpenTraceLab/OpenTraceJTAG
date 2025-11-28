# Layer Control Example

Add layer visibility control to the gio-viewer.

## Example Implementation

```go
package main

import (
    "gioui.org/app"
    "gioui.org/layout"
    "gioui.org/op"
    "github.com/epkcfsm/kibrd/pkg/kicad/pcb"
    "github.com/epkcfsm/kibrd/pkg/kicad/renderer"
)

func main() {
    board, _ := parser.ParseFile("board.kicad_pcb")
    camera := renderer.NewCamera(800, 600)
    config := renderer.NewLayerConfig()
    
    go func() {
        w := new(app.Window)
        var ops op.Ops
        
        for {
            switch e := w.Event().(type) {
            case app.FrameEvent:
                ops.Reset()
                gtx := layout.Context{
                    Ops:         &ops,
                    Constraints: layout.Exact(e.Size),
                    Metric:      e.Metric,
                }
                
                // Handle keyboard for layer toggles
                for {
                    ev, ok := gtx.Event(key.Filter{})
                    if !ok {
                        break
                    }
                    if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
                        switch ke.Name {
                        case "1":
                            // Toggle front copper
                            visible := config.IsVisible("F.Cu")
                            config.SetVisible("F.Cu", !visible)
                        case "2":
                            // Toggle back copper
                            visible := config.IsVisible("B.Cu")
                            config.SetVisible("B.Cu", !visible)
                        case "3":
                            // Toggle silkscreen
                            visible := config.IsVisible("F.SilkS")
                            config.SetVisible("F.SilkS", !visible)
                            config.SetVisible("B.SilkS", !visible)
                        case "C":
                            // Show copper only
                            config.ShowCopperOnly()
                        case "S":
                            // Show silkscreen only
                            config.ShowSilkscreenOnly()
                        case "A":
                            // Show all layers
                            config.ShowAll()
                        }
                        w.Invalidate()
                    }
                }
                
                // Render with layer config
                renderer.RenderBoardWithConfig(gtx, camera, board, config)
                
                e.Frame(&ops)
            }
        }
    }()
    app.Main()
}
```

## Keyboard Shortcuts

Add these to your viewer:

- **1**: Toggle front copper (F.Cu)
- **2**: Toggle back copper (B.Cu)
- **3**: Toggle silkscreen (F.SilkS, B.SilkS)
- **C**: Show copper layers only
- **S**: Show silkscreen only
- **A**: Show all layers

## UI Integration

For a more advanced UI, you could add checkboxes:

```go
// Create layer checkboxes
var layerToggles = map[string]*widget.Bool{
    "F.Cu":    new(widget.Bool),
    "B.Cu":    new(widget.Bool),
    "F.SilkS": new(widget.Bool),
    "B.SilkS": new(widget.Bool),
}

// Initialize all to checked
for _, toggle := range layerToggles {
    toggle.Value = true
}

// In your layout function:
for layer, toggle := range layerToggles {
    material.CheckBox(th, toggle, layer).Layout(gtx)
    config.SetVisible(layer, toggle.Value)
}
```
