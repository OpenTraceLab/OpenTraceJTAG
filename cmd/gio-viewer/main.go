package main

import (
	"fmt"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
)

// Tag for keyboard events
type keyTag struct{}

var kTag keyTag

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gio-viewer <board_file>")
		fmt.Println("\nNote: If keyboard doesn't work on Wayland, run with:")
		fmt.Println("  GIO_BACKEND=x11 gio-viewer <board_file>")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Parse the board file
	fmt.Printf("Loading board: %s\n", filename)
	board, err := parser.ParseFile(filename)
	if err != nil {
		fmt.Printf("Error parsing board: %v\n", err)
		os.Exit(1)
	}

	// Print board info
	fmt.Printf("✓ Loaded board successfully\n")
	fmt.Printf("  Version: %d\n", board.Version)
	fmt.Printf("  Generator: %s\n", board.Generator)
	fmt.Printf("  Layers: %d\n", len(board.Layers))
	fmt.Printf("  Nets: %d\n", len(board.Nets))
	fmt.Printf("  Footprints: %d\n", len(board.Footprints))
	fmt.Printf("  Tracks: %d\n", len(board.Tracks))
	fmt.Printf("  Vias: %d\n", len(board.Vias))
	fmt.Printf("  Zones: %d\n", len(board.Zones))

	bbox := board.GetBoundingBox()
	if !bbox.IsEmpty() {
		fmt.Printf("  Board size: %.2f x %.2f mm\n", bbox.Width(), bbox.Height())
		fmt.Printf("  Board center: (%.2f, %.2f) mm\n", bbox.Center().X, bbox.Center().Y)
	}

	// Run the Gio application
	go func() {
		w := new(app.Window)
		w.Option(app.Title("KiCad Board Viewer (Gio) - " + filename))
		w.Option(app.Size(unit.Dp(1000), unit.Dp(800)))

		if err := run(w, board, bbox); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window, board *parser.Board, bbox parser.BoundingBox) error {
	// Initialize camera
	camera := renderer.NewCamera(1000, 800)
	if !bbox.IsEmpty() {
		camera.FitBoard(bbox)
	}

	var ops op.Ops
	debugRotationOffset := 0.0 // Debug rotation offset for J1/J2

	// Log initial rotation info for J1, J2, and U5
	for _, fp := range board.Footprints {
		if fp.Reference == "J1" || fp.Reference == "J2" || fp.Reference == "U5" {
			fmt.Printf("\n=== %s Initial Info ===\n", fp.Reference)
			fmt.Printf("  Position: (%.2f, %.2f) Angle: %.2f°\n",
				fp.Position.X, fp.Position.Y, fp.Position.Angle)
			fmt.Printf("  Pads: %d\n", len(fp.Pads))
			for i, pad := range fp.Pads {
				if i < 3 { // Show first 3 pads
					fmt.Printf("    Pad %s: Pos=(%.2f, %.2f) Angle=%.2f° Size=(%.2f x %.2f) Shape=%s\n",
						pad.Number, pad.Position.X, pad.Position.Y, pad.Position.Angle,
						pad.Size.Width, pad.Size.Height, pad.Shape)
				}
			}
		}
	}

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err

		case app.FrameEvent:
			// Reset operations for new frame
			ops.Reset()

			// Create graphics context
			gtx := layout.Context{
				Ops:         &ops,
				Constraints: layout.Exact(e.Size),
				Metric:      e.Metric,
				Now:         e.Now,
				Source:      e.Source,
			}

			// Update camera screen size
			camera.UpdateScreenSize(e.Size.X, e.Size.Y)

			// Handle keyboard events using gtx.Event
			for {
				ev, ok := gtx.Event(key.Filter{})
				if !ok {
					break
				}
				
				if ke, ok := ev.(key.Event); ok {
					if ke.State == key.Press {
						closeWindow, newOffset := handleKeyPress(ke.Name, camera, bbox, debugRotationOffset)
						if closeWindow {
							return nil
						}
						if newOffset != debugRotationOffset {
							debugRotationOffset = newOffset
							fmt.Printf(">>> Debug Rotation Offset: %.1f°\n", debugRotationOffset)
						}
						w.Invalidate()
					}
				}
			}

			// Handle mouse events
			for {
				ev, ok := gtx.Event(pointer.Filter{
					Kinds: pointer.Press | pointer.Scroll,
				})
				if !ok {
					break
				}
				
				if pe, ok := ev.(pointer.Event); ok {
					switch pe.Kind {
					case pointer.Press:
						if pe.Buttons == pointer.ButtonPrimary {
							// Left click - rotate
							camera.Rotate(90)
							w.Invalidate()
						} else if pe.Buttons == pointer.ButtonSecondary {
							// Right click - flip
							camera.Flip()
							w.Invalidate()
						}
					case pointer.Scroll:
						// Scroll wheel - zoom
						zoomFactor := 1.0 + float64(pe.Scroll.Y)*0.1
						camera.ZoomAt(float64(pe.Position.X), float64(pe.Position.Y), zoomFactor)
						w.Invalidate()
					}
				}
			}

			// Clear background
			paint.Fill(&ops, renderer.ColorBackground)

			// Render board with debug rotation offset
			renderBoardWithDebug(gtx, camera, board, debugRotationOffset)

			// Submit frame
			e.Frame(&ops)
		}
	}
}

func handleKeyPress(k key.Name, camera *renderer.Camera, bbox parser.BoundingBox, currentOffset float64) (bool, float64) {
	switch k {
	case key.NameEscape, "Q":
		return true, currentOffset // Signal to close
	case "F":
		camera.Flip()
	case "R":
		camera.Rotate(90)
	case key.NameLeftArrow:
		camera.Rotate(-90)
	case key.NameSpace:
		if !bbox.IsEmpty() {
			camera.FitBoard(bbox)
		}
	// Debug rotation controls
	case "1":
		return false, currentOffset + 10.0 // Add 10 degrees
	case "2":
		return false, currentOffset - 10.0 // Subtract 10 degrees
	case "3":
		return false, currentOffset + 90.0 // Add 90 degrees
	case "4":
		return false, currentOffset - 90.0 // Subtract 90 degrees
	case "5":
		return false, currentOffset + 1.0 // Add 1 degree
	case "6":
		return false, currentOffset - 1.0 // Subtract 1 degree
	case "0":
		return false, 0.0 // Reset to 0
	}
	return false, currentOffset
}

func renderBoard(gtx layout.Context, camera *renderer.Camera, board *parser.Board) {
	// Call the Gio renderer
	renderer.RenderBoard(gtx, camera, board)
}

func renderBoardWithDebug(gtx layout.Context, camera *renderer.Camera, board *parser.Board, debugRotationOffset float64) {
	// Call the Gio renderer with debug rotation offset
	renderer.RenderBoardWithDebug(gtx, camera, board, debugRotationOffset)
}
