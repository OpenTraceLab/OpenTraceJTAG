package main

import (
	"fmt"
	"log"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug-symbols <schematic.kicad_sch>")
		os.Exit(1)
	}

	sch, err := schematic.ParseFile(os.Args[1])
	if err != nil {
		log.Fatalf("Error parsing schematic: %v", err)
	}

	fmt.Printf("Library Symbols: %d\n", len(sch.LibSymbols))
	fmt.Printf("Symbol Instances: %d\n\n", len(sch.Symbols))

	// Show library symbols and their graphics
	fmt.Println("=== Library Symbols ===")
	for i, libSym := range sch.LibSymbols {
		fmt.Printf("[%d] %s\n", i, libSym.Name)
		fmt.Printf("    Graphics: %d items\n", len(libSym.Graphics))
		fmt.Printf("    Pins: %d\n", len(libSym.Pins))
		if len(libSym.Graphics) > 0 {
			for j, g := range libSym.Graphics {
				if j < 3 { // Show first 3
					fmt.Printf("      [%d] Type=%s\n", j, g.Type)
				}
			}
		}
	}

	// Show symbol instances
	fmt.Println("\n=== Symbol Instances ===")
	for i, sym := range sch.Symbols {
		if i >= 5 { // Show first 5
			break
		}
		fmt.Printf("[%d] LibID=%s at (%.2f, %.2f) angle=%.0f\n",
			i, sym.LibID, sym.Position.X, sym.Position.Y, sym.Angle)

		// Find matching library symbol
		found := false
		for _, libSym := range sch.LibSymbols {
			if libSym.Name == sym.LibID {
				found = true
				fmt.Printf("    ✓ Found lib symbol with %d graphics, %d pins\n",
					len(libSym.Graphics), len(libSym.Pins))
				break
			}
		}
		if !found {
			fmt.Printf("    ✗ Library symbol NOT FOUND!\n")
		}
	}
}
