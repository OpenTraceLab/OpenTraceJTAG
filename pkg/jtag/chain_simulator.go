package jtag

import (
	"fmt"
	"strings"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
)

// SimulatedDevice represents a single device in a simulated JTAG chain.
type SimulatedDevice struct {
	BSDLFile *bsdl.BSDLFile
	Info     *bsdl.DeviceInfo
	IDCode   uint32
	IRLength int
	BSRState []byte // Current boundary scan register state
}

// NetConnection represents a simulated electrical connection between pins.
type NetConnection struct {
	NetName string
	Pins    []PinRef // List of connected pins across devices
}

// PinRef identifies a specific pin on a device.
type PinRef struct {
	DeviceIndex int    // Index in chain (0-based)
	PinName     string // Pin name from BSDL
	BSRIndex    int    // Boundary scan register bit index
}

// ChainSimulator simulates a multi-device JTAG chain with configurable connections.
type ChainSimulator struct {
	Devices     []SimulatedDevice
	Connections []NetConnection
	
	// Current TAP state
	currentIR map[int][]byte // Current instruction register per device
	
	// Adapter interface
	adapter *SimAdapter
}

// NewChainSimulator creates a simulator with the specified devices and connections.
func NewChainSimulator(devices []SimulatedDevice, connections []NetConnection) *ChainSimulator {
	sim := &ChainSimulator{
		Devices:     devices,
		Connections: connections,
		currentIR:   make(map[int][]byte),
	}
	
	// Initialize BSR state for each device
	for i := range sim.Devices {
		bsrLen := sim.Devices[i].Info.BoundaryLength
		sim.Devices[i].BSRState = make([]byte, (bsrLen+7)/8)
	}
	
	// Create adapter with custom shift hook
	sim.adapter = NewSimAdapter(AdapterInfo{
		Name: "Chain Simulator",
	})
	sim.adapter.OnShift = sim.handleShift
	
	return sim
}

// Adapter returns the underlying SimAdapter for use with JTAG operations.
func (cs *ChainSimulator) Adapter() Adapter {
	return cs.adapter
}

// handleShift simulates the shift operation across the entire chain.
func (cs *ChainSimulator) handleShift(region ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
	tdo := make([]byte, len(tdi))
	
	switch region {
	case ShiftRegionIR:
		return cs.shiftIR(tdi, bits)
		
	case ShiftRegionDR:
		// Check if all devices are in EXTEST mode
		allEXTEST := len(cs.currentIR) == len(cs.Devices)
		for i := range cs.Devices {
			ir, ok := cs.currentIR[i]
			if !ok || len(ir) == 0 {
				allEXTEST = false
				break
			}
		}
		
		if allEXTEST {
			// Calculate expected BSR length
			totalBSRLen := 0
			for _, dev := range cs.Devices {
				totalBSRLen += dev.Info.BoundaryLength
			}
			// Only shift BSR if bit count matches (actual data shift, not state transition)
			if bits == totalBSRLen {
				return cs.shiftBSR(tdi, bits)
			}
			// Otherwise it's just a state transition, return zeros
			return tdo, nil
		}
		
		// Default state: return IDCODEs
		if bits == len(cs.Devices)*32 {
			bitPos := 0
			for _, dev := range cs.Devices {
				for j := 0; j < 32; j++ {
					byteIdx := (bitPos + j) / 8
					bitIdx := (bitPos + j) % 8
					if (dev.IDCode & (1 << j)) != 0 {
						tdo[byteIdx] |= 1 << bitIdx
					}
				}
				bitPos += 32
			}
		}
		return tdo, nil
	}
	
	return tdo, nil
}

// shiftIR simulates shifting through all instruction registers in the chain.
func (cs *ChainSimulator) shiftIR(tdi []byte, bits int) ([]byte, error) {
	// Calculate total IR length
	totalIRLen := 0
	for _, dev := range cs.Devices {
		totalIRLen += dev.IRLength
	}
	
	// Only validate length for actual IR shifts, not state transitions
	if bits != totalIRLen {
		// State transition, just return zeros
		tdo := make([]byte, len(tdi))
		return tdo, nil
	}
	
	tdo := make([]byte, len(tdi))
	bitPos := 0
	
	// Shift through each device's IR
	for i := range cs.Devices {
		irLen := cs.Devices[i].IRLength
		
		// Extract IR value for this device from TDI
		ir := make([]byte, (irLen+7)/8)
		for j := 0; j < irLen; j++ {
			byteIdx := (bitPos + j) / 8
			bitIdx := (bitPos + j) % 8
			if byteIdx < len(tdi) && (tdi[byteIdx]&(1<<bitIdx)) != 0 {
				ir[j/8] |= 1 << (j % 8)
			}
		}
		
		// Store current IR
		cs.currentIR[i] = ir
		
		// TDO returns previous IR (simulate with zeros for now)
		bitPos += irLen
	}
	
	return tdo, nil
}

// shiftBSR simulates shifting through all boundary scan registers.
func (cs *ChainSimulator) shiftBSR(tdi []byte, bits int) ([]byte, error) {
	// Calculate total BSR length
	totalBSRLen := 0
	for _, dev := range cs.Devices {
		totalBSRLen += dev.Info.BoundaryLength
	}
	
	if bits != totalBSRLen {
		return nil, fmt.Errorf("BSR shift length mismatch: got %d, expected %d", bits, totalBSRLen)
	}
	
	// First update BSR state from TDI
	// DR bits are ordered from TDO device to TDI device (reverse of device array)
	bitPos := 0
	for devIdx := len(cs.Devices) - 1; devIdx >= 0; devIdx-- {
		dev := &cs.Devices[devIdx]
		bsrLen := dev.Info.BoundaryLength
		
		for j := 0; j < bsrLen; j++ {
			byteIdx := (bitPos + j) / 8
			bitIdx := (bitPos + j) % 8
			dstByteIdx := j / 8
			dstBitIdx := j % 8
			
			if byteIdx < len(tdi) {
				oldVal := (dev.BSRState[dstByteIdx] & (1 << dstBitIdx)) != 0
				newVal := (tdi[byteIdx] & (1 << bitIdx)) != 0
				
				// Debug PA5 cells (94=INPUT, 95=OUTPUT, 96=CONTROL)
				if j >= 94 && j <= 96 {
					if oldVal != newVal {
						fmt.Printf("[SIM] BSR Update: dev%d cell %d: %v -> %v\n", devIdx, j, oldVal, newVal)
					}
				}
				
				if newVal {
					dev.BSRState[dstByteIdx] |= 1 << dstBitIdx
				} else {
					dev.BSRState[dstByteIdx] &^= 1 << dstBitIdx
				}
			}
		}
		
		bitPos += bsrLen
	}
	
	// Propagate connections so driven values affect connected pins
	cs.propagateConnections()
	
	// Now capture BSR state to TDO (same reverse order)
	tdo := make([]byte, len(tdi))
	bitPos = 0
	
	for devIdx := len(cs.Devices) - 1; devIdx >= 0; devIdx-- {
		dev := &cs.Devices[devIdx]
		bsrLen := dev.Info.BoundaryLength
		
		for j := 0; j < bsrLen; j++ {
			byteIdx := (bitPos + j) / 8
			bitIdx := (bitPos + j) % 8
			srcByteIdx := j / 8
			srcBitIdx := j % 8
			
			if byteIdx < len(tdo) && srcByteIdx < len(dev.BSRState) {
				if (dev.BSRState[srcByteIdx] & (1 << srcBitIdx)) != 0 {
					tdo[byteIdx] |= 1 << bitIdx
				}
			}
		}
		
		bitPos += bsrLen
	}
	
	return tdo, nil
}

// propagateConnections simulates electrical connections between pins.
// When a pin is driven high/low, all connected pins should reflect that state.
func (cs *ChainSimulator) propagateConnections() {
	for _, conn := range cs.Connections {
		// Find the driven pin (output) and propagate to inputs
		var drivenValue *bool
		
		fmt.Printf("[SIM] Checking connection %s with %d pins\n", conn.NetName, len(conn.Pins))
		
		for _, pin := range conn.Pins {
			if pin.DeviceIndex >= len(cs.Devices) {
				continue
			}
			
			dev := &cs.Devices[pin.DeviceIndex]
			cells, err := dev.BSDLFile.Entity.GetBoundaryCells()
			if err != nil {
				continue
			}
			
			// Find OUTPUT cell for this pin
			var outputCell *bsdl.BoundaryCell
			for i := range cells {
				if cells[i].Port == pin.PinName && strings.HasPrefix(strings.ToUpper(cells[i].Function), "OUTPUT") {
					outputCell = &cells[i]
					break
				}
			}
			
			if outputCell != nil {
				byteIdx := outputCell.Number / 8
				bitIdx := outputCell.Number % 8
				
				if byteIdx < len(dev.BSRState) {
					value := (dev.BSRState[byteIdx] & (1 << bitIdx)) != 0
					
					if drivenValue == nil {
						drivenValue = &value
						fmt.Printf("[SIM]   Driver: dev%d.%s output cell %d = %v\n", pin.DeviceIndex, pin.PinName, outputCell.Number, value)
					}
				}
			}
		}
		
		// Propagate driven value to all INPUT cells on this net
		if drivenValue != nil {
			for _, pin := range conn.Pins {
				if pin.DeviceIndex >= len(cs.Devices) {
					continue
				}
				
				dev := &cs.Devices[pin.DeviceIndex]
				
				// pin.BSRIndex should be the INPUT cell
				if pin.BSRIndex >= 0 && pin.BSRIndex < dev.Info.BoundaryLength {
					byteIdx := pin.BSRIndex / 8
					bitIdx := pin.BSRIndex % 8
					
					if byteIdx < len(dev.BSRState) {
						oldVal := (dev.BSRState[byteIdx] & (1 << bitIdx)) != 0
						if *drivenValue {
							dev.BSRState[byteIdx] |= 1 << bitIdx
						} else {
							dev.BSRState[byteIdx] &^= 1 << bitIdx
						}
						if oldVal != *drivenValue {
							fmt.Printf("[SIM]   Propagated to dev%d.%s input cell %d: %v -> %v\n", pin.DeviceIndex, pin.PinName, pin.BSRIndex, oldVal, *drivenValue)
						}
					}
				}
			}
		}
	}
}

// GetDeviceCount returns the number of devices in the simulated chain.
func (cs *ChainSimulator) GetDeviceCount() int {
	return len(cs.Devices)
}

// GetDevice returns information about a specific device in the chain.
func (cs *ChainSimulator) GetDevice(index int) (*SimulatedDevice, error) {
	if index < 0 || index >= len(cs.Devices) {
		return nil, fmt.Errorf("device index %d out of range", index)
	}
	return &cs.Devices[index], nil
}

// Reset clears the simulator state (current instructions).
func (cs *ChainSimulator) Reset() {
	cs.currentIR = make(map[int][]byte)
}
