package main

import (
	"fmt"
	"log"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug-bbox <schematic.kicad_sch>")
		os.Exit(1)
	}

	sch, err := schematic.ParseFile(os.Args[1])
	if err != nil {
		log.Fatalf("Error parsing schematic: %v", err)
	}

	fmt.Printf("Schematic: %s\n", os.Args[1])
	fmt.Printf("Components: %d\n", len(sch.Symbols))
	fmt.Printf("Wires: %d\n", len(sch.Wires))
	fmt.Printf("Labels: %d\n\n", len(sch.Labels)+len(sch.GlobalLabels)+len(sch.HierLabels))

	// Print first 5 symbol positions
	fmt.Println("First 5 symbol positions:")
	for i, sym := range sch.Symbols {
		if i >= 5 {
			break
		}
		fmt.Printf("  [%d] %s at (%.2f, %.2f)\n", i, sym.LibID, sym.Position.X, sym.Position.Y)
	}

	// Print first 5 wire endpoints
	fmt.Println("\nFirst 5 wire segments:")
	for i, wire := range sch.Wires {
		if i >= 5 {
			break
		}
		if len(wire.Points) >= 2 {
			fmt.Printf("  [%d] (%.2f, %.2f) -> (%.2f, %.2f)\n",
				i, wire.Points[0].X, wire.Points[0].Y,
				wire.Points[len(wire.Points)-1].X, wire.Points[len(wire.Points)-1].Y)
		}
	}

	// Calculate bounding box
	bbox := sch.GetBoundingBox()
	fmt.Printf("\nBounding Box:\n")
	fmt.Printf("  Min: (%.2f, %.2f)\n", bbox.Min.X, bbox.Min.Y)
	fmt.Printf("  Max: (%.2f, %.2f)\n", bbox.Max.X, bbox.Max.Y)
	fmt.Printf("  Width: %.2f\n", bbox.Width())
	fmt.Printf("  Height: %.2f\n", bbox.Height())
	fmt.Printf("  IsEmpty: %v\n", bbox.IsEmpty())
}
