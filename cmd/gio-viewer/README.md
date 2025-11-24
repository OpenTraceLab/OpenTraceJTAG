# Gio Viewer

Interactive KiCad board viewer using Gio.

## Usage

```bash
./gio-viewer <board_file.kicad_pcb>
```

## Controls

- **Left Click**: Rotate board 90° clockwise
- **Right Click**: Flip board (top/bottom view)
- **Scroll Wheel**: Zoom in/out
- **F**: Flip board
- **R**: Rotate 90° clockwise
- **Left Arrow**: Rotate 90° counter-clockwise
- **Space**: Fit board to view
- **Q / Escape**: Quit

## Net Highlighting Example

To add net highlighting to the viewer, modify the render call:

```go
// In the frame event handler, replace:
renderer.RenderBoard(gtx, camera, board)

// With:
highlightNet := "GND"  // or any net name
renderer.RenderBoardWithHighlight(gtx, camera, board, highlightNet)
```

You can add keyboard shortcuts to cycle through nets:

```go
var highlightNet string
var netIndex int

// In keyboard handler:
case "N":
    // Cycle to next net
    if len(board.Nets) > 0 {
        netIndex = (netIndex + 1) % len(board.Nets)
        highlightNet = board.Nets[netIndex].Name
        fmt.Printf("Highlighting net: %s\n", highlightNet)
    }
case "C":
    // Clear highlight
    highlightNet = ""
    fmt.Println("Highlight cleared")
```

## Wayland Note

If keyboard input doesn't work on Wayland, run with X11 backend:

```bash
GIO_BACKEND=x11 ./gio-viewer <board_file.kicad_pcb>
```
