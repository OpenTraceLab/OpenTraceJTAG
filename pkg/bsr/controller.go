package bsr

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
)

// Controller provides pin-centric boundary-scan operations on top of a
// discovered JTAG chain. It manages runtime state for all pins and handles
// the translation between pin operations and low-level DR shifts.
type Controller struct {
	chain   *chain.Chain
	Devices []*DeviceRuntime
	Layout  *DRLayout

	// Cached DR state to minimize USB traffic.
	// This is updated on each SetAllPinsHiZ or DrivePin operation.
	currentDR []bool
}

// NewController builds a boundary-scan runtime controller from a discovered chain.
// It extracts all IO pins from each device and initializes them to HiZ mode.
func NewController(ch *chain.Chain) (*Controller, error) {
	if ch == nil {
		return nil, fmt.Errorf("bsr: chain is nil")
	}

	chainDevices := ch.Devices()
	if len(chainDevices) == 0 {
		return nil, fmt.Errorf("bsr: chain has no devices")
	}

	// Build runtime devices
	devices := make([]*DeviceRuntime, len(chainDevices))
	for i, dev := range chainDevices {
		runtime, err := newDeviceRuntime(dev)
		if err != nil {
			return nil, fmt.Errorf("bsr: failed to build runtime for device %s: %w", dev.Name(), err)
		}
		devices[i] = runtime
	}

	// Build global DR layout
	layout := buildDRLayout(devices)

	return &Controller{
		chain:     ch,
		Devices:   devices,
		Layout:    layout,
		currentDR: make([]bool, layout.TotalBits), // Start with all zeros
	}, nil
}

// newDeviceRuntime creates a DeviceRuntime from a chain.Device.
func newDeviceRuntime(dev *chain.Device) (*DeviceRuntime, error) {
	// Get boundary length
	cells, err := dev.BoundaryCells()
	if err != nil {
		return nil, fmt.Errorf("failed to get boundary cells: %w", err)
	}

	if dev.Info == nil {
		return nil, fmt.Errorf("device %s missing DeviceInfo", dev.Name())
	}

	boundaryLength := dev.Info.BoundaryLength
	if boundaryLength == 0 {
		boundaryLength = len(cells)
	}

	if boundaryLength == 0 {
		return nil, fmt.Errorf("device %s has zero boundary length", dev.Name())
	}

	// Get instruction opcodes
	extestOpcode, err := dev.ExtestOpcode()
	if err != nil {
		return nil, fmt.Errorf("failed to get EXTEST opcode: %w", err)
	}

	bypassOpcode, err := dev.BypassOpcode()
	if err != nil {
		return nil, fmt.Errorf("failed to get BYPASS opcode: %w", err)
	}

	// Get IO pins
	ioPins, err := dev.IOPins()
	if err != nil {
		return nil, fmt.Errorf("failed to get IO pins: %w", err)
	}

	// Initialize PinState for each IO pin
	pins := make(map[string]*PinState)
	for _, pinName := range ioPins {
		pins[pinName] = &PinState{
			Ref: PinRef{
				ChainIndex: dev.Position,
				DeviceName: dev.Name(),
				PinName:    pinName,
			},
			Mode:      PinHiZ,
			DrivenVal: nil,
			LastRead:  nil,
		}
	}

	return &DeviceRuntime{
		ChainDev:       dev,
		Pins:           pins,
		boundaryLength: boundaryLength,
		extestOpcode:   extestOpcode,
		bypassOpcode:   bypassOpcode,
	}, nil
}

// GetPinState returns the current runtime state for a specific pin.
// Returns nil if the pin is not found.
func (c *Controller) GetPinState(ref PinRef) *PinState {
	if ref.ChainIndex < 0 || ref.ChainIndex >= len(c.Devices) {
		return nil
	}

	dev := c.Devices[ref.ChainIndex]
	return dev.Pins[ref.PinName]
}

// AllPins returns a list of all pin references in the chain.
func (c *Controller) AllPins() []PinRef {
	var pins []PinRef
	for _, dev := range c.Devices {
		for _, ps := range dev.Pins {
			pins = append(pins, ps.Ref)
		}
	}
	return pins
}

// Chain returns the underlying chain.Chain for access to transport operations.
func (c *Controller) Chain() *chain.Chain {
	return c.chain
}
