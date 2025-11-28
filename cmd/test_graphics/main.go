package main

import (
	"fmt"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/pcb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_graphics <board_file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	board, err := pcb.ParseFile(filename)
	if err != nil {
		fmt.Printf("Error parsing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully parsed: %s\n", filename)
	fmt.Printf("\nGraphics Summary:\n")
	fmt.Printf("  Lines:      %d\n", len(board.Graphics.Lines))
	fmt.Printf("  Circles:    %d\n", len(board.Graphics.Circles))
	fmt.Printf("  Arcs:       %d\n", len(board.Graphics.Arcs))
	fmt.Printf("  Rectangles: %d\n", len(board.Graphics.Rects))
	fmt.Printf("  Polygons:   %d\n", len(board.Graphics.Polys))
	fmt.Printf("  Text:       %d\n", len(board.Graphics.Texts))

	total := len(board.Graphics.Lines) + len(board.Graphics.Circles) +
		len(board.Graphics.Arcs) + len(board.Graphics.Rects) +
		len(board.Graphics.Polys) + len(board.Graphics.Texts)
	fmt.Printf("  Total:      %d\n", total)

	// Show first few lines if any
	if len(board.Graphics.Lines) > 0 {
		fmt.Printf("\nFirst 5 lines:\n")
		for i, line := range board.Graphics.Lines {
			if i >= 5 {
				break
			}
			fmt.Printf("  [%d] (%.2f, %.2f) -> (%.2f, %.2f) on %s (width: %.3f)\n",
				i+1, line.Start.X, line.Start.Y, line.End.X, line.End.Y,
				line.Layer, line.Stroke.Width)
		}
	}

	// Show first few text elements if any
	if len(board.Graphics.Texts) > 0 {
		fmt.Printf("\nFirst 5 text elements:\n")
		for i, text := range board.Graphics.Texts {
			if i >= 5 {
				break
			}
			fmt.Printf("  [%d] \"%s\" at (%.2f, %.2f) on %s\n",
				i+1, text.Text, text.Position.X, text.Position.Y, text.Layer)
		}
	}

	// Show layer breakdown
	fmt.Printf("\nGraphics by layer:\n")
	layerCounts := make(map[string]int)

	for _, line := range board.Graphics.Lines {
		layerCounts[line.Layer]++
	}
	for _, circle := range board.Graphics.Circles {
		layerCounts[circle.Layer]++
	}
	for _, arc := range board.Graphics.Arcs {
		layerCounts[arc.Layer]++
	}
	for _, rect := range board.Graphics.Rects {
		layerCounts[rect.Layer]++
	}
	for _, poly := range board.Graphics.Polys {
		layerCounts[poly.Layer]++
	}
	for _, text := range board.Graphics.Texts {
		layerCounts[text.Layer]++
	}

	for layer, count := range layerCounts {
		fmt.Printf("  %-20s: %d\n", layer, count)
	}
}
