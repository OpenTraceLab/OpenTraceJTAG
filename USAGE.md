# Using KiBrd in Your Gio Application

## Installation

```bash
go get github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser
go get github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer
```

## Basic Usage

```go
package main

import (
    "log"
    
    "gioui.org/app"
    "gioui.org/layout"
    "gioui.org/op"
    
    "github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser"
    "github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
)

func main() {
    // Parse board file
    board, err := parser.ParseFile("board.kicad_pcb")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create camera for view control
    camera := renderer.NewCamera(800, 600)
    bbox := board.GetBoundingBox()
    if !bbox.IsEmpty() {
        camera.FitBoard(bbox)
    }
    
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
                
                // Update camera size
                camera.UpdateScreenSize(e.Size.X, e.Size.Y)
                
                // Render the board
                renderer.RenderBoard(gtx, camera, board)
                
                e.Frame(&ops)
            }
        }
    }()
    app.Main()
}
```

## Camera Controls

The `Camera` provides view transformation:

```go
camera := renderer.NewCamera(width, height)

// Fit board in view
camera.FitBoard(board.GetBoundingBox())

// Zoom
camera.Zoom(1.5)  // 150% zoom
camera.ZoomAt(x, y, 1.2)  // Zoom at specific point

// Pan
camera.Pan(dx, dy)  // Pan by delta

// Rotate
camera.Rotate(90)  // Rotate 90° clockwise

// Flip
camera.Flip()  // Toggle top/bottom view
```

## Rendering Options

```go
// Basic rendering (all layers visible)
renderer.RenderBoard(gtx, camera, board)

// With layer visibility control
config := renderer.NewLayerConfig()
config.SetVisible("B.Cu", false)  // Hide back copper
renderer.RenderBoardWithConfig(gtx, camera, board, config)

// Highlight a specific net (dims all other elements)
renderer.RenderBoardWithHighlight(gtx, camera, board, "GND")

// With debug rotation offset (for development)
renderer.RenderBoardWithDebug(gtx, camera, board, rotationOffset)
```

### Layer Visibility Control

Control which layers are rendered:

```go
config := renderer.NewLayerConfig()

// Hide/show specific layers
config.SetVisible("B.Cu", false)      // Hide back copper
config.SetVisible("F.SilkS", false)   // Hide front silkscreen
config.SetVisible("F.Mask", false)    // Hide front solder mask

// Show only specific layers
config.ShowOnly("F.Cu", "F.SilkS")    // Only front copper and silkscreen

// Convenience methods for layer groups
config.ShowCopperOnly()               // Only copper layers
config.ShowSilkscreenOnly()           // Only silkscreen layers
config.ShowFabOnly()                  // Only fabrication layers

config.HideCopper()                   // Hide all copper layers
config.HideSilkscreen()               // Hide all silkscreen

// Hide/show all
config.HideAll()                      // Hide everything
config.ShowAll()                      // Show everything (default)

// Render with config
renderer.RenderBoardWithConfig(gtx, camera, board, config)
```

### Common Layer Names

- **Copper**: `F.Cu`, `B.Cu`, `In1.Cu`, `In2.Cu`, etc.
- **Silkscreen**: `F.SilkS`, `B.SilkS`
- **Solder Mask**: `F.Mask`, `B.Mask`
- **Paste**: `F.Paste`, `B.Paste`
- **Fabrication**: `F.Fab`, `B.Fab`
- **Courtyard**: `F.CrtYd`, `B.CrtYd`
- **Adhesive**: `F.Adhes`, `B.Adhes`

### Net Highlighting

Highlight a specific net to make it stand out:

```go
// Highlight the GND net
renderer.RenderBoardWithHighlight(gtx, camera, board, "GND")

// Highlight a signal net
renderer.RenderBoardWithHighlight(gtx, camera, board, "/SCL")

// No highlighting (empty string)
renderer.RenderBoardWithHighlight(gtx, camera, board, "")
```

When a net is highlighted:
- Highlighted elements (tracks, vias, pads) are shown in bright yellow
- Highlighted tracks are 1.5x thicker
- Highlighted vias are 1.3x larger
- All other elements are dimmed to 60% opacity

## What Gets Rendered

- Tracks (copper traces)
- Vias (plated through-holes)
- Pads (footprint pads with drill holes)
- Graphics (silkscreen, fab layer lines, circles, rectangles, polygons)
- Zones (filled copper areas)

## Net Query Utilities

Query nets and their connected elements:

```go
// Get a specific net by name
net := board.GetNet("GND")
if net != nil {
    fmt.Printf("Net: %s (number %d)\n", net.Name, net.Number)
}

// Get all pads connected to a net
pads := board.GetNetPads("GND")
fmt.Printf("GND has %d pads\n", len(pads))

// Get all tracks on a net
tracks := board.GetNetTracks("/SCL")
fmt.Printf("/SCL has %d track segments\n", len(tracks))

// Get all vias on a net
vias := board.GetNetVias("VCC")
fmt.Printf("VCC has %d vias\n", len(vias))

// Get complete net information
info := board.GetNetInfo("GND")
if info != nil {
    fmt.Printf("Net: %s\n", info.Net.Name)
    fmt.Printf("  Pads: %d\n", len(info.Pads))
    fmt.Printf("  Tracks: %d\n", len(info.Tracks))
    fmt.Printf("  Vias: %d\n", len(info.Vias))
}

// Get all net names
netNames := board.GetAllNetNames()
for _, name := range netNames {
    fmt.Println(name)
}
```

### Example: Interactive Net Selection

```go
// List all nets with their connection counts
for _, netName := range board.GetAllNetNames() {
    info := board.GetNetInfo(netName)
    if info != nil {
        fmt.Printf("%-20s: %d pads, %d tracks, %d vias\n",
            netName, len(info.Pads), len(info.Tracks), len(info.Vias))
    }
}
```

## Layer Colors

Default colors are defined in `renderer/colors.go`:
- Tracks: Red/Blue (front/back copper)
- Pads: Gold
- Silkscreen: White
- Background: Dark gray

You can customize by modifying the color constants or implementing your own rendering.
