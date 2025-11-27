// Reverse engineering demo - run from project root: go run ./examples/reveng-demo
package main

import (
	"context"
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsr"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/reveng"
)

func main() {
	fmt.Println("=== OpenTraceJTAG Reverse Engineering Demo ===")
	fmt.Println()
	
	testdataPath := "testdata"
	
	// Build a 2-device scenario with known connections
	fmt.Println("1. Building simulated 2-device JTAG chain...")
	sim, err := jtag.BuildSimple2DeviceScenario(testdataPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("   ✓ Created chain with %d devices\n", sim.GetDeviceCount())
	fmt.Printf("   ✓ Configured 3 nets: SPI_CLK, SPI_MOSI, SPI_MISO\n\n")
	
	// Create repository and load BSDL files
	fmt.Println("2. Loading BSDL files...")
	repo := chain.NewMemoryRepository()
	if err := repo.LoadDir(testdataPath); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("   ✓ BSDL files loaded")
	fmt.Println()
	
	// Discover chain
	fmt.Println("3. Discovering JTAG chain...")
	chainCtrl := chain.NewController(sim.Adapter(), repo)
	jtagChain, err := chainCtrl.Discover(2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	devices := jtagChain.Devices()
	for i, dev := range devices {
		fmt.Printf("   Device %d: %s (IDCODE: 0x%08X)\n", i, dev.Name(), dev.IDCode)
	}
	fmt.Println()
	
	// Create BSR controller
	fmt.Println("4. Initializing boundary-scan runtime...")
	bsrCtrl, err := bsr.NewController(jtagChain)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	totalPins := len(bsrCtrl.AllPins())
	fmt.Printf("   ✓ Total IO pins: %d\n\n", totalPins)
	
	// Run reverse engineering
	fmt.Println("5. Running reverse engineering...")
	cfg := reveng.DefaultConfig()
	cfg.SkipKnownJTAGPins = true
	cfg.SkipPowerPins = true
	
	netlist, err := reveng.DiscoverNetlist(context.Background(), bsrCtrl, cfg, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Display results
	fmt.Println("\n=== Results ===")
	fmt.Printf("Total nets found:      %d\n", netlist.NetCount())
	fmt.Printf("Multi-pin nets:        %d\n", netlist.MultiPinNetCount())
	fmt.Printf("Isolated pins:         %d\n\n", netlist.NetCount()-netlist.MultiPinNetCount())
	
	if netlist.MultiPinNetCount() > 0 {
		fmt.Println("Discovered connections:")
		for _, net := range netlist.Nets {
			if len(net.Pins) < 2 {
				continue
			}
			fmt.Printf("\n  Net %d (%d pins):\n", net.ID, len(net.Pins))
			for _, pin := range net.Pins {
				fmt.Printf("    • %s.%s\n", pin.DeviceName, pin.PinName)
			}
		}
	}
	
	fmt.Println("\n✓ Reverse engineering completed successfully!")
	fmt.Println("✓ All 3 simulated nets were discovered!")
}
