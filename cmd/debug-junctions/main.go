package main

import (
	"fmt"
	"log"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug-junctions <schematic.kicad_sch>")
		os.Exit(1)
	}

	sch, err := schematic.ParseFile(os.Args[1])
	if err != nil {
		log.Fatalf("Error parsing schematic: %v", err)
	}

	fmt.Printf("Junctions: %d\n", len(sch.Junctions))
	for i, j := range sch.Junctions {
		if i >= 5 {
			break
		}
		fmt.Printf("  [%d] Position: (%.4f, %.4f) Diameter: %.4f\n",
			i, j.Position.X, j.Position.Y, j.Diameter)
	}

	fmt.Printf("\nNo-connects: %d\n", len(sch.NoConnects))
	for i, nc := range sch.NoConnects {
		if i >= 5 {
			break
		}
		fmt.Printf("  [%d] Position: (%.4f, %.4f)\n",
			i, nc.Position.X, nc.Position.Y)
	}

	fmt.Printf("\nWires (first few endpoints): %d total\n", len(sch.Wires))
	for i, w := range sch.Wires {
		if i >= 5 {
			break
		}
		if len(w.Points) >= 2 {
			start := w.Points[0]
			end := w.Points[len(w.Points)-1]
			fmt.Printf("  [%d] Start: (%.4f, %.4f) End: (%.4f, %.4f)\n",
				i, start.X, start.Y, end.X, end.Y)
		}
	}
}
