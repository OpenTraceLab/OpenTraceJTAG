package main

import (
	"fmt"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/pcb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_parse <board_file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	board, err := pcb.ParseFile(filename)
	if err != nil {
		fmt.Printf("Error parsing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully parsed: %s\n", filename)
	fmt.Printf("\nBoard Information:\n")
	fmt.Printf("  Version: %d\n", board.Version)
	fmt.Printf("  Generator: %s\n", board.Generator)
	fmt.Printf("  Title: %s\n", board.General.Title)
	fmt.Printf("  Thickness: %.2f mm\n", board.General.Thickness)
	fmt.Printf("  Number of layers: %d\n", len(board.Layers))

	fmt.Printf("\nLayers:\n")
	for _, layer := range board.Layers {
		fmt.Printf("  [%2d] %-15s %s\n", layer.Number, layer.Name, layer.Type)
	}
}
