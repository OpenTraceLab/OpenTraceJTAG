package main

import (
	"fmt"
	"os"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/kicad/pcb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_nets <board_file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	board, err := pcb.ParseFile(filename)
	if err != nil {
		fmt.Printf("Error parsing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully parsed: %s\n", filename)
	fmt.Printf("\nNet Information:\n")
	fmt.Printf("  Total nets: %d\n", len(board.Nets))

	// Create net map for lookups
	netMap := pcb.NewNetMap(board.Nets)

	// Show first 10 nets
	fmt.Printf("\nFirst 10 nets:\n")
	count := 0
	for _, net := range board.Nets {
		if count >= 10 {
			break
		}
		name := net.Name
		if name == "" {
			name = "(empty)"
		}
		fmt.Printf("  [%3d] %s\n", net.Number, name)
		count++
	}

	// Test lookups
	fmt.Printf("\nNet lookup tests:\n")
	if gnd, ok := netMap.GetByName("GND"); ok {
		fmt.Printf("  ✓ Found GND: net %d\n", gnd.Number)
	}
	if vcc, ok := netMap.GetByName("+5V"); ok {
		fmt.Printf("  ✓ Found +5V: net %d\n", vcc.Number)
	}
	if v3, ok := netMap.GetByName("+3V3"); ok {
		fmt.Printf("  ✓ Found +3V3: net %d\n", v3.Number)
	}
	if netMap.IsUnconnected(0) {
		fmt.Printf("  ✓ Net 0 is unconnected\n")
	}
}
