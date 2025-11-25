package main

import (
	"fmt"
	"log"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/schematic"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug-graphics <schematic.kicad_sch>")
		os.Exit(1)
	}

	sch, err := schematic.ParseFile(os.Args[1])
	if err != nil {
		log.Fatalf("Error parsing schematic: %v", err)
	}

	fmt.Printf("Library Symbols: %d\n\n", len(sch.LibSymbols))

	// Show library symbols and their graphics counts
	for _, libSym := range sch.LibSymbols {
		fmt.Printf("%s:\n", libSym.Name)
		fmt.Printf("  Graphics: %d\n", len(libSym.Graphics))
		fmt.Printf("  Pins: %d\n", len(libSym.Pins))
		fmt.Printf("  Units: %d\n", len(libSym.Units))

		// Show graphic types
		typeCount := make(map[string]int)
		for _, g := range libSym.Graphics {
			typeCount[g.Type]++
		}
		if len(typeCount) > 0 {
			fmt.Printf("  Graphic types: ")
			for t, c := range typeCount {
				fmt.Printf("%s=%d ", t, c)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}
