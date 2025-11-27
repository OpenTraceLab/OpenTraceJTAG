package main

import (
	"fmt"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/pcb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_tracks <board_file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	board, err := pcb.ParseFile(filename)
	if err != nil {
		fmt.Printf("Error parsing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully parsed: %s\n", filename)
	fmt.Printf("\nTracking Summary:\n")
	fmt.Printf("  Tracks: %d\n", len(board.Tracks))
	fmt.Printf("  Vias:   %d\n", len(board.Vias))

	// Show first few tracks
	if len(board.Tracks) > 0 {
		fmt.Printf("\nFirst 10 tracks:\n")
		for i, track := range board.Tracks {
			if i >= 10 {
				break
			}
			netName := "(unconnected)"
			if track.Net != nil {
				netName = track.Net.Name
				if netName == "" {
					netName = fmt.Sprintf("net %d", track.Net.Number)
				}
			}
			locked := ""
			if track.Locked {
				locked = " [LOCKED]"
			}
			fmt.Printf("  [%3d] (%.2f, %.2f) -> (%.2f, %.2f) | width: %.3f | %s | %s%s\n",
				i+1, track.Start.X, track.Start.Y, track.End.X, track.End.Y,
				track.Width, track.Layer, netName, locked)
		}
	}

	// Show first few vias
	if len(board.Vias) > 0 {
		fmt.Printf("\nFirst 10 vias:\n")
		for i, via := range board.Vias {
			if i >= 10 {
				break
			}
			netName := "(unconnected)"
			if via.Net != nil {
				netName = via.Net.Name
				if netName == "" {
					netName = fmt.Sprintf("net %d", via.Net.Number)
				}
			}
			locked := ""
			if via.Locked {
				locked = " [LOCKED]"
			}
			fmt.Printf("  [%3d] at (%.2f, %.2f) | size: %.2f | drill: %.2f | %s%s\n",
				i+1, via.Position.X, via.Position.Y, via.Size, via.Drill, netName, locked)
		}
	}

	// Layer breakdown for tracks
	if len(board.Tracks) > 0 {
		fmt.Printf("\nTracks by layer:\n")
		layerCounts := make(map[string]int)
		for _, track := range board.Tracks {
			layerCounts[track.Layer]++
		}
		for layer, count := range layerCounts {
			fmt.Printf("  %-20s: %d\n", layer, count)
		}
	}

	// Net breakdown for tracks
	if len(board.Tracks) > 0 {
		fmt.Printf("\nTracks by net (top 10):\n")
		netCounts := make(map[string]int)
		for _, track := range board.Tracks {
			netName := "(unconnected)"
			if track.Net != nil {
				netName = track.Net.Name
				if netName == "" {
					netName = fmt.Sprintf("net %d", track.Net.Number)
				}
			}
			netCounts[netName]++
		}

		// Find top 10
		type netCount struct {
			name  string
			count int
		}
		var counts []netCount
		for name, count := range netCounts {
			counts = append(counts, netCount{name, count})
		}
		// Simple bubble sort
		for i := 0; i < len(counts); i++ {
			for j := i + 1; j < len(counts); j++ {
				if counts[j].count > counts[i].count {
					counts[i], counts[j] = counts[j], counts[i]
				}
			}
		}
		for i, nc := range counts {
			if i >= 10 {
				break
			}
			fmt.Printf("  %-30s: %d\n", nc.name, nc.count)
		}
	}
}
