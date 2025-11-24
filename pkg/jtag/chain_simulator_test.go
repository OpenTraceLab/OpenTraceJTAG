package jtag

import (
	"path/filepath"
	"testing"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
)

// TestScenarioSimple creates a simple 2-device chain with known connections.
func TestScenarioSimple(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	// Load BSDL files
	bsdl1, err := parser.ParseFile(filepath.Join("../../testdata", "STM32F303_F334_LQFP64.bsd"))
	if err != nil {
		t.Fatalf("Failed to load BSDL: %v", err)
	}
	
	bsdl2, err := parser.ParseFile(filepath.Join("../../testdata", "STM32F358_LQFP64.bsd"))
	if err != nil {
		t.Fatalf("Failed to load BSDL: %v", err)
	}
	
	info1 := bsdl1.Entity.GetDeviceInfo()
	info2 := bsdl2.Entity.GetDeviceInfo()
	
	// Create devices
	devices := []SimulatedDevice{
		{
			BSDLFile: bsdl1,
			Info:     info1,
			IDCode:   0x06438041, // STM32F303 IDCODE
			IRLength: 5,
		},
		{
			BSDLFile: bsdl2,
			Info:     info2,
			IDCode:   0x06422041, // STM32F358 IDCODE
			IRLength: 5,
		},
	}
	
	// Create some test connections
	// Connect PA0 of device 0 to PA1 of device 1
	connections := []NetConnection{
		{
			NetName: "TEST_NET_1",
			Pins: []PinRef{
				{DeviceIndex: 0, PinName: "PA0", BSRIndex: 0},
				{DeviceIndex: 1, PinName: "PA1", BSRIndex: 3},
			},
		},
	}
	
	// Create simulator
	sim := NewChainSimulator(devices, connections)
	
	// Test basic operations
	adapter := sim.Adapter()
	
	// Test IR shift
	irData := make([]byte, 2) // 10 bits total (5+5)
	tdo, err := adapter.ShiftIR(nil, irData, 10)
	if err != nil {
		t.Errorf("ShiftIR failed: %v", err)
	}
	if len(tdo) != len(irData) {
		t.Errorf("TDO length mismatch: got %d, want %d", len(tdo), len(irData))
	}
	
	t.Logf("Simple scenario created with %d devices and %d connections", 
		sim.GetDeviceCount(), len(connections))
}

// TestScenarioComplex creates a 4-device chain with multiple nets.
func TestScenarioComplex(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	// Load BSDL files for different devices
	bsdlFiles := []string{
		"STM32F303_F334_LQFP64.bsd",
		"STM32F358_LQFP64.bsd",
		"STM32F405_LQFP100.bsd",
		"STM32F373_LQFP100.bsd",
	}
	
	var devices []SimulatedDevice
	
	for i, filename := range bsdlFiles {
		bsdlFile, err := parser.ParseFile(filepath.Join("../../testdata", filename))
		if err != nil {
			t.Fatalf("Failed to load BSDL %s: %v", filename, err)
		}
		
		info := bsdlFile.Entity.GetDeviceInfo()
		
		devices = append(devices, SimulatedDevice{
			BSDLFile: bsdlFile,
			Info:     info,
			IDCode:   uint32(0x06400000 + i), // Fake IDCODEs
			IRLength: 5,
		})
	}
	
	// Create multiple nets connecting various pins
	connections := []NetConnection{
		{
			NetName: "SPI_CLK",
			Pins: []PinRef{
				{DeviceIndex: 0, PinName: "PA5", BSRIndex: 15},
				{DeviceIndex: 1, PinName: "PA5", BSRIndex: 15},
				{DeviceIndex: 2, PinName: "PA5", BSRIndex: 15},
			},
		},
		{
			NetName: "SPI_MOSI",
			Pins: []PinRef{
				{DeviceIndex: 0, PinName: "PA7", BSRIndex: 21},
				{DeviceIndex: 1, PinName: "PA7", BSRIndex: 21},
			},
		},
		{
			NetName: "SPI_MISO",
			Pins: []PinRef{
				{DeviceIndex: 0, PinName: "PA6", BSRIndex: 18},
				{DeviceIndex: 2, PinName: "PA6", BSRIndex: 18},
			},
		},
		{
			NetName: "POWER_3V3",
			Pins: []PinRef{
				{DeviceIndex: 0, PinName: "VDD", BSRIndex: -1}, // Power pins might not be in BSR
				{DeviceIndex: 1, PinName: "VDD", BSRIndex: -1},
				{DeviceIndex: 2, PinName: "VDD", BSRIndex: -1},
				{DeviceIndex: 3, PinName: "VDD", BSRIndex: -1},
			},
		},
	}
	
	// Create simulator
	sim := NewChainSimulator(devices, connections)
	
	t.Logf("Complex scenario created with %d devices and %d connections", 
		sim.GetDeviceCount(), len(connections))
	
	// Verify device count
	if sim.GetDeviceCount() != 4 {
		t.Errorf("Expected 4 devices, got %d", sim.GetDeviceCount())
	}
}

// TestConnectionPropagation verifies that electrical connections are simulated correctly.
func TestConnectionPropagation(t *testing.T) {
	parser, err := bsdl.NewParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	// Load a simple BSDL
	bsdlFile, err := parser.ParseFile(filepath.Join("../../testdata", "STM32F303_F334_LQFP64.bsd"))
	if err != nil {
		t.Fatalf("Failed to load BSDL: %v", err)
	}
	
	info := bsdlFile.Entity.GetDeviceInfo()
	
	// Create 2 devices
	devices := []SimulatedDevice{
		{BSDLFile: bsdlFile, Info: info, IDCode: 0x06438041, IRLength: 5},
		{BSDLFile: bsdlFile, Info: info, IDCode: 0x06438041, IRLength: 5},
	}
	
	// Connect pin 0 of device 0 to pin 0 of device 1
	connections := []NetConnection{
		{
			NetName: "TEST_NET",
			Pins: []PinRef{
				{DeviceIndex: 0, PinName: "PA0", BSRIndex: 0},
				{DeviceIndex: 1, PinName: "PA0", BSRIndex: 0},
			},
		},
	}
	
	sim := NewChainSimulator(devices, connections)

	// Get the actual OUTPUT cell for PA0 from the BSDL
	// PA0 has: INPUT cell 0, OUTPUT cell 110, CONTROL cell 111
	// We need to set the OUTPUT cell (110) high to propagate the value
	outputCellPA0 := 110
	byteIdx := outputCellPA0 / 8
	bitIdx := outputCellPA0 % 8
	sim.Devices[0].BSRState[byteIdx] |= 1 << bitIdx

	// Also need to set the control cell to enable output (control cell 111, disable value is 1)
	// So we need control = 0 to enable output, but let's just propagate and check INPUT cell

	// Propagate connections
	sim.propagateConnections()

	// Verify INPUT cell 0 of device 1 reflects the driven value
	// The propagation reads from OUTPUT cell and writes to INPUT cells based on BSRIndex in PinRef
	// Since PinRef has BSRIndex: 0 (the INPUT cell), that's where the value should be written
	if sim.Devices[1].BSRState[0]&0x01 == 0 {
		t.Error("Connection propagation failed: pin should be high")
	}

	t.Log("Connection propagation test passed")
}
