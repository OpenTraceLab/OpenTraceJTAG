package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: net-info <board_file.kicad_pcb> [net_name]")
		fmt.Println("\nExamples:")
		fmt.Println("  net-info board.kicad_pcb           # List all nets")
		fmt.Println("  net-info board.kicad_pcb GND       # Show GND net details")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Parse board
	board, err := parser.ParseFile(filename)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// If net name provided, show details for that net
	if len(os.Args) >= 3 {
		netName := os.Args[2]
		showNetDetails(board, netName)
		return
	}

	// Otherwise, list all nets
	listAllNets(board)
}

func listAllNets(board *parser.Board) {
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

func showNetDetails(board *parser.Board, netName string) {
	info := board.GetNetInfo(netName)
	if info == nil {
		fmt.Printf("Net '%s' not found\n", netName)
		os.Exit(1)
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
}
