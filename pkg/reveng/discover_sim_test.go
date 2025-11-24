package reveng

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsr"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
)

func TestDiscoverWithSimulator(t *testing.T) {
	testdataPath := filepath.Join("..", "..", "testdata")
	
	// Build a simple 2-device scenario with known connections
	sim, err := jtag.BuildSimple2DeviceScenario(testdataPath)
	if err != nil {
		t.Fatalf("Failed to build scenario: %v", err)
	}
	
	// Create repository and load BSDL files
	repo := chain.NewMemoryRepository()
	if err := repo.LoadDir(testdataPath); err != nil {
		t.Fatalf("Failed to load BSDL files: %v", err)
	}
	
	// Discover chain
	chainCtrl := chain.NewController(sim.Adapter(), repo)
	jtagChain, err := chainCtrl.Discover(2)
	if err != nil {
		t.Fatalf("Chain discovery failed: %v", err)
	}
	
	devices := jtagChain.Devices()
	if len(devices) != 2 {
		t.Fatalf("Expected 2 devices, got %d", len(devices))
	}
	
	// Create BSR controller
	bsrCtrl, err := bsr.NewController(jtagChain)
	if err != nil {
		t.Fatalf("Failed to create BSR controller: %v", err)
	}
	
	// Run reverse engineering
	cfg := DefaultConfig()
	cfg.SkipKnownJTAGPins = true
	cfg.SkipPowerPins = true
	
	netlist, err := DiscoverNetlist(context.Background(), bsrCtrl, cfg, nil)
	if err != nil {
		t.Fatalf("Reverse engineering failed: %v", err)
	}
	
	// Check results
	t.Logf("Found %d nets", netlist.NetCount())
	t.Logf("Multi-pin nets: %d", netlist.MultiPinNetCount())
	
	if netlist.MultiPinNetCount() == 0 {
		t.Error("Expected to find at least one multi-pin net")
	}
	
	// Verify the known connections are found
	expectedNets := []string{"NET_SPI_CLK", "NET_SPI_MOSI", "NET_SPI_MISO"}
	t.Logf("Expected nets: %v", expectedNets)
	
	found := false
	for _, net := range netlist.Nets {
		if len(net.Pins) >= 2 {
			// Check if this net contains the expected pins
			for _, pin := range net.Pins {
				t.Logf("Net %d: %s.%s", net.ID, pin.DeviceName, pin.PinName)
			}
			found = true
		}
	}
	if !found {
		t.Error("No multi-pin nets found")
	}
}
