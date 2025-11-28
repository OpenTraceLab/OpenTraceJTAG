package main

import (
	"fmt"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/pcb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_footprints <board_file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	board, err := pcb.ParseFile(filename)
	if err != nil {
		fmt.Printf("Error parsing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully parsed: %s\n", filename)
	fmt.Printf("\nFootprints Summary:\n")
	fmt.Printf("  Total footprints: %d\n", len(board.Footprints))

	// Count total pads
	totalPads := 0
	for _, fp := range board.Footprints {
		totalPads += len(fp.Pads)
	}
	fmt.Printf("  Total pads:       %d\n", totalPads)

	// Show first 10 footprints
	if len(board.Footprints) > 0 {
		fmt.Printf("\nFirst 10 footprints:\n")
		for i, fp := range board.Footprints {
			if i >= 10 {
				break
			}
			libraryName := fp.Library
			if libraryName != "" {
				libraryName = libraryName + ":"
			}
			fmt.Printf("  [%3d] %-10s | %s%-30s | pads: %d | (%.2f, %.2f) @ %.0f°\n",
				i+1, fp.Reference, libraryName, fp.Name, len(fp.Pads),
				fp.Position.X, fp.Position.Y, fp.Position.Angle)
		}
	}

	// Count pad types
	fmt.Printf("\nPad type breakdown:\n")
	padTypes := make(map[string]int)
	padShapes := make(map[string]int)
	for _, fp := range board.Footprints {
		for _, pad := range fp.Pads {
			padTypes[pad.Type]++
			padShapes[pad.Shape]++
		}
	}
	for padType, count := range padTypes {
		fmt.Printf("  %-15s: %d\n", padType, count)
	}

	fmt.Printf("\nPad shape breakdown:\n")
	for shape, count := range padShapes {
		fmt.Printf("  %-15s: %d\n", shape, count)
	}

	// Count footprints by library
	fmt.Printf("\nTop 10 libraries:\n")
	libraries := make(map[string]int)
	for _, fp := range board.Footprints {
		if fp.Library != "" {
			libraries[fp.Library]++
		}
	}

	// Simple sorting
	type libCount struct {
		name  string
		count int
	}
	var counts []libCount
	for name, count := range libraries {
		counts = append(counts, libCount{name, count})
	}
	for i := 0; i < len(counts); i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[j].count > counts[i].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}
	for i, lc := range counts {
		if i >= 10 {
			break
		}
		fmt.Printf("  %-30s: %d\n", lc.name, lc.count)
	}

	// Layer breakdown
	fmt.Printf("\nFootprints by layer:\n")
	layers := make(map[string]int)
	for _, fp := range board.Footprints {
		layers[fp.Layer]++
	}
	for layer, count := range layers {
		fmt.Printf("  %-20s: %d\n", layer, count)
	}
}
