package main

import (
	"fmt"
	"log"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug-props <schematic.kicad_sch>")
		os.Exit(1)
	}

	sch, err := schematic.ParseFile(os.Args[1])
	if err != nil {
		log.Fatalf("Error parsing schematic: %v", err)
	}

	fmt.Printf("Symbols: %d\n\n", len(sch.Symbols))

	// Show first few symbols and their properties
	for i, sym := range sch.Symbols {
		if i >= 3 { // Only show first 3 symbols
			break
		}
		fmt.Printf("Symbol %d: %s\n", i, sym.LibID)
		fmt.Printf("  Position: (%.2f, %.2f)\n", sym.Position.X, sym.Position.Y)
		fmt.Printf("  Properties: %d\n", len(sym.Properties))
		for _, prop := range sym.Properties {
			fmt.Printf("    %s = '%s'\n", prop.Key, prop.Value)
			fmt.Printf("      Position: (%.2f, %.2f) Angle: %.2f\n",
				prop.Position.X, prop.Position.Y, prop.Position.Angle)
			fmt.Printf("      Font Size: %.4f x %.4f\n",
				prop.Effects.Font.Size.Width, prop.Effects.Font.Size.Height)
		}
		fmt.Println()
	}
}
