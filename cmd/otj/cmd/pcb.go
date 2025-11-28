package cmd

import (
	"fmt"
	"log"
	"os"
	"sort"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/pcb"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/renderer"
	"github.com/spf13/cobra"
)

var pcbCmd = &cobra.Command{
	Use:   "pcb",
	Short: "KiCad PCB file operations",
	Long:  `Commands for working with KiCad PCB files (.kicad_pcb)`,
}

var pcbViewCmd = &cobra.Command{
	Use:   "view <board_file>",
	Short: "View PCB file in interactive viewer",
	Long: `Opens a PCB file in an interactive Gio-based viewer with pan, zoom, and rotation controls.

Controls:
  Left Click / R    - Rotate 90°
  Right Click / F   - Flip board
  Scroll Wheel      - Zoom in/out
  Space             - Fit board to window
  Q / Escape        - Quit`,
	Args: cobra.ExactArgs(1),
	RunE: runPCBView,
}

var pcbNetsCmd = &cobra.Command{
	Use:   "nets <board_file> [net_name]",
	Short: "Show PCB net information",
	Long: `Display information about nets in a PCB file.

Without net_name: Lists all nets with pad/track/via counts
With net_name: Shows detailed information for that specific net`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runPCBNets,
}

func init() {
	rootCmd.AddCommand(pcbCmd)
	pcbCmd.AddCommand(pcbViewCmd)
	pcbCmd.AddCommand(pcbNetsCmd)
}

func runPCBView(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Parse the board file
	fmt.Printf("Loading board: %s\n", filename)
	board, err := pcb.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("error parsing board: %w", err)
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
		w.Option(app.Title("KiCad Board Viewer - " + filename))
		w.Option(app.Size(unit.Dp(1000), unit.Dp(800)))

		if err := runViewerWindow(w, board, bbox); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
	return nil
}

func runViewerWindow(w *app.Window, board *pcb.Board, bbox pcb.BoundingBox) error {
	// Initialize camera
	camera := renderer.NewCamera(1000, 800)
	if !bbox.IsEmpty() {
		camera.FitBoard(bbox)
	}

	var ops op.Ops

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

			// Handle keyboard events
			for {
				ev, ok := gtx.Event(key.Filter{})
				if !ok {
					break
				}

				if ke, ok := ev.(key.Event); ok {
					if ke.State == key.Press {
						if handleKeyPress(ke.Name, camera, bbox) {
							return nil // Close window
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

			// Render board
			renderer.RenderBoard(gtx, camera, board)

			// Submit frame
			e.Frame(&ops)
		}
	}
}

func handleKeyPress(k key.Name, camera *renderer.Camera, bbox pcb.BoundingBox) bool {
	switch k {
	case key.NameEscape, "Q":
		return true // Signal to close
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
	}
	return false
}

func runPCBNets(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Parse board
	board, err := pcb.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	// If net name provided, show details for that net
	if len(args) >= 2 {
		netName := args[1]
		return showNetDetails(board, netName)
	}

	// Otherwise, list all nets
	listAllNets(board)
	return nil
}

func listAllNets(board *pcb.Board) {
	fmt.Printf("Board: %d nets\n\n", len(board.Nets))
	fmt.Printf("%-30s %6s %6s %6s\n", "Net Name", "Pads", "Tracks", "Vias")
	fmt.Println("─────────────────────────────────────────────────────────")

	// Get all net names and sort them
	netNames := board.GetAllNetNames()
	sort.Strings(netNames)

	for _, netName := range netNames {
		info := board.GetNetInfo(netName)
		if info != nil {
			fmt.Printf("%-30s %6d %6d %6d\n",
				netName,
				len(info.Pads),
				len(info.Tracks),
				len(info.Vias))
		}
	}
}

func showNetDetails(board *pcb.Board, netName string) error {
	info := board.GetNetInfo(netName)
	if info == nil {
		return fmt.Errorf("net '%s' not found", netName)
	}

	fmt.Printf("Net: %s (number %d)\n\n", info.Net.Name, info.Net.Number)

	// Show pads
	fmt.Printf("Pads (%d):\n", len(info.Pads))
	for _, pad := range info.Pads {
		fmt.Printf("  Pad %-4s: %s %.2f×%.2f mm at (%.2f, %.2f)\n",
			pad.Number, pad.Shape,
			pad.Size.Width, pad.Size.Height,
			pad.Position.X, pad.Position.Y)
	}

	// Show tracks
	fmt.Printf("\nTracks (%d):\n", len(info.Tracks))
	for i, track := range info.Tracks {
		fmt.Printf("  Track %d: %.2f mm wide on %s from (%.2f, %.2f) to (%.2f, %.2f)\n",
			i+1, track.Width, track.Layer,
			track.Start.X, track.Start.Y,
			track.End.X, track.End.Y)
	}

	// Show vias
	fmt.Printf("\nVias (%d):\n", len(info.Vias))
	for i, via := range info.Vias {
		fmt.Printf("  Via %d: %.2f mm diameter, %.2f mm drill at (%.2f, %.2f)\n",
			i+1, via.Size, via.Drill,
			via.Position.X, via.Position.Y)
	}

	return nil
}
