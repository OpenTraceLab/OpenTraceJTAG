package jtag

import (
	"fmt"
	"path/filepath"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
)

// ScenarioBuilder helps construct test scenarios with real BSDL files.
type ScenarioBuilder struct {
	testdataPath string
	devices      []SimulatedDevice
	connections  []NetConnection
}

// NewScenarioBuilder creates a new scenario builder.
// testdataPath should point to the directory containing BSDL files.
func NewScenarioBuilder(testdataPath string) *ScenarioBuilder {
	return &ScenarioBuilder{
		testdataPath: testdataPath,
	}
}

// AddDevice adds a device to the scenario using a BSDL file.
func (sb *ScenarioBuilder) AddDevice(bsdlFilename string, idcode uint32) error {
	bsdlPath := filepath.Join(sb.testdataPath, bsdlFilename)
	
	parser, err := bsdl.NewParser()
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}
	
	bsdlFile, err := parser.ParseFile(bsdlPath)
	if err != nil {
		return fmt.Errorf("failed to load BSDL %s: %w", bsdlFilename, err)
	}
	
	// Extract device info from attributes
	info := bsdlFile.Entity.GetDeviceInfo()
	
	// Determine IR length from device info (default to 5 for STM32)
	irLen := 5
	if info.InstructionLength > 0 {
		irLen = info.InstructionLength
	}
	
	sb.devices = append(sb.devices, SimulatedDevice{
		BSDLFile: bsdlFile,
		Info:     info,
		IDCode:   idcode,
		IRLength: irLen,
	})
	
	return nil
}

// ConnectPins creates a net connecting the specified pins.
func (sb *ScenarioBuilder) ConnectPins(netName string, pins ...PinRef) {
	sb.connections = append(sb.connections, NetConnection{
		NetName: netName,
		Pins:    pins,
	})
}

// Build creates the ChainSimulator with the configured devices and connections.
func (sb *ScenarioBuilder) Build() *ChainSimulator {
	return NewChainSimulator(sb.devices, sb.connections)
}

// Predefined scenarios for common testing needs

// BuildSimple2DeviceScenario creates a simple 2-device chain.
func BuildSimple2DeviceScenario(testdataPath string) (*ChainSimulator, error) {
	sb := NewScenarioBuilder(testdataPath)
	
	// Add 2 STM32 devices
	if err := sb.AddDevice("STM32F303_F334_LQFP64.bsd", 0x06438041); err != nil {
		return nil, err
	}
	if err := sb.AddDevice("STM32F358_LQFP64.bsd", 0x06422041); err != nil {
		return nil, err
	}
	
	// Extract pin mappings from BSDL files
	pm0 := bsdl.ExtractPinMapping(sb.devices[0].BSDLFile)
	pm1 := bsdl.ExtractPinMapping(sb.devices[1].BSDLFile)
	
	// Create test connections with actual BSR indices
	sb.ConnectPins("NET_SPI_CLK",
		PinRef{DeviceIndex: 0, PinName: "PA5", BSRIndex: pm0.GetBSRIndex("PA5")},
		PinRef{DeviceIndex: 1, PinName: "PA5", BSRIndex: pm1.GetBSRIndex("PA5")},
	)
	
	sb.ConnectPins("NET_SPI_MOSI",
		PinRef{DeviceIndex: 0, PinName: "PA7", BSRIndex: pm0.GetBSRIndex("PA7")},
		PinRef{DeviceIndex: 1, PinName: "PA7", BSRIndex: pm1.GetBSRIndex("PA7")},
	)
	
	sb.ConnectPins("NET_SPI_MISO",
		PinRef{DeviceIndex: 0, PinName: "PA6", BSRIndex: pm0.GetBSRIndex("PA6")},
		PinRef{DeviceIndex: 1, PinName: "PA6", BSRIndex: pm1.GetBSRIndex("PA6")},
	)
	
	return sb.Build(), nil
}

// BuildComplex4DeviceScenario creates a complex 4-device chain with multiple nets.
func BuildComplex4DeviceScenario(testdataPath string) (*ChainSimulator, error) {
	sb := NewScenarioBuilder(testdataPath)
	
	// Add 4 different STM32 devices
	devices := []struct {
		file   string
		idcode uint32
	}{
		{"STM32F303_F334_LQFP64.bsd", 0x06438041},
		{"STM32F358_LQFP64.bsd", 0x06422041},
		{"STM32F405_LQFP100.bsd", 0x06413041},
		{"STM32F373_LQFP100.bsd", 0x06432041},
	}
	
	for _, dev := range devices {
		if err := sb.AddDevice(dev.file, dev.idcode); err != nil {
			return nil, err
		}
	}
	
	// Create SPI bus connecting devices 0, 1, 2
	sb.ConnectPins("SPI_CLK",
		PinRef{DeviceIndex: 0, PinName: "PA5", BSRIndex: 15},
		PinRef{DeviceIndex: 1, PinName: "PA5", BSRIndex: 15},
		PinRef{DeviceIndex: 2, PinName: "PA5", BSRIndex: 15},
	)
	
	sb.ConnectPins("SPI_MOSI",
		PinRef{DeviceIndex: 0, PinName: "PA7", BSRIndex: 21},
		PinRef{DeviceIndex: 1, PinName: "PA7", BSRIndex: 21},
		PinRef{DeviceIndex: 2, PinName: "PA7", BSRIndex: 21},
	)
	
	sb.ConnectPins("SPI_MISO",
		PinRef{DeviceIndex: 0, PinName: "PA6", BSRIndex: 18},
		PinRef{DeviceIndex: 1, PinName: "PA6", BSRIndex: 18},
		PinRef{DeviceIndex: 2, PinName: "PA6", BSRIndex: 18},
	)
	
	// Create I2C bus connecting devices 2, 3
	sb.ConnectPins("I2C_SCL",
		PinRef{DeviceIndex: 2, PinName: "PB6", BSRIndex: 45},
		PinRef{DeviceIndex: 3, PinName: "PB6", BSRIndex: 45},
	)
	
	sb.ConnectPins("I2C_SDA",
		PinRef{DeviceIndex: 2, PinName: "PB7", BSRIndex: 48},
		PinRef{DeviceIndex: 3, PinName: "PB7", BSRIndex: 48},
	)
	
	// Create UART connection between devices 0 and 3
	sb.ConnectPins("UART_TX",
		PinRef{DeviceIndex: 0, PinName: "PA9", BSRIndex: 27},
		PinRef{DeviceIndex: 3, PinName: "PA10", BSRIndex: 30},
	)
	
	sb.ConnectPins("UART_RX",
		PinRef{DeviceIndex: 0, PinName: "PA10", BSRIndex: 30},
		PinRef{DeviceIndex: 3, PinName: "PA9", BSRIndex: 27},
	)
	
	// Add some unconnected pins for testing
	sb.ConnectPins("ISOLATED_NET",
		PinRef{DeviceIndex: 1, PinName: "PC13", BSRIndex: 90},
	)
	
	return sb.Build(), nil
}

// BuildMinimalScenario creates the smallest possible test scenario (2 devices, 1 connection).
func BuildMinimalScenario(testdataPath string) (*ChainSimulator, error) {
	sb := NewScenarioBuilder(testdataPath)
	
	if err := sb.AddDevice("STM32F303_F334_LQFP64.bsd", 0x06438041); err != nil {
		return nil, err
	}
	if err := sb.AddDevice("STM32F303_F334_LQFP64.bsd", 0x06438041); err != nil {
		return nil, err
	}
	
	// Single connection
	sb.ConnectPins("TEST_NET",
		PinRef{DeviceIndex: 0, PinName: "PA0", BSRIndex: 0},
		PinRef{DeviceIndex: 1, PinName: "PA0", BSRIndex: 0},
	)
	
	return sb.Build(), nil
}
