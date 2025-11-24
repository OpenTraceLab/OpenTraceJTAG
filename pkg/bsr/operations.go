package bsr

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
)

// EnterExtest programs all devices in the chain with the EXTEST instruction.
// This puts all devices into boundary-scan test mode where the boundary
// register controls pin values instead of the device's internal logic.
func (c *Controller) EnterExtest() error {
	// Build instruction mapping: all devices get EXTEST
	instMap := make(map[*chain.Device]string)
	for _, dev := range c.Devices {
		instMap[dev.ChainDev] = "EXTEST"
	}

	if err := c.chain.ProgramInstructions(instMap); err != nil {
		return fmt.Errorf("bsr: failed to program EXTEST: %w", err)
	}

	return nil
}

// SetAllPinsHiZ tri-states all pins on all devices by setting their control
// cells to disable outputs. This is typically the first operation after
// entering EXTEST mode to ensure no conflicts.
func (c *Controller) SetAllPinsHiZ() error {
	// Build DR vector with all pins tri-stated
	var globalDR []bool

	// Devices are ordered from TDI (index 0) to TDO (last index).
	// DR scan shifts from TDI to TDO, so we reverse when building the vector.
	for devIdx := len(c.Devices) - 1; devIdx >= 0; devIdx-- {
		dev := c.Devices[devIdx]
		segment, err := setAllPinsHiZ(dev)
		if err != nil {
			return fmt.Errorf("bsr: failed to build HiZ segment for device %s: %w", dev.ChainDev.Name(), err)
		}
		globalDR = append(globalDR, segment...)
	}

	// Shift DR
	_, err := c.chain.ShiftDRBits(globalDR)
	if err != nil {
		return fmt.Errorf("bsr: failed to shift DR: %w", err)
	}

	// Update cached DR state
	c.currentDR = globalDR

	// Update all pin states to HiZ
	for _, dev := range c.Devices {
		for _, ps := range dev.Pins {
			ps.Mode = PinHiZ
			ps.DrivenVal = nil
		}
	}

	return nil
}

// DrivePin drives a single pin to the specified value (high=true, low=false).
// All other pins on the same device are set to HiZ. Pins on other devices
// retain their current state.
func (c *Controller) DrivePin(ref PinRef, value bool) error {
	// Validate the pin reference
	if ref.ChainIndex < 0 || ref.ChainIndex >= len(c.Devices) {
		return fmt.Errorf("bsr: invalid chain index %d", ref.ChainIndex)
	}

	targetDev := c.Devices[ref.ChainIndex]
	if _, ok := targetDev.Pins[ref.PinName]; !ok {
		return fmt.Errorf("bsr: pin %s not found on device %s", ref.PinName, ref.DeviceName)
	}

	// Debug logging for PA5
	if ref.PinName == "PA5" {
		fmt.Printf("[BSR] DrivePin: dev%d.%s = %v\n", ref.ChainIndex, ref.PinName, value)
	}

	// Build DR vector
	var globalDR []bool

	for devIdx := len(c.Devices) - 1; devIdx >= 0; devIdx-- {
		dev := c.Devices[devIdx]
		var segment []bool
		var err error

		if devIdx == ref.ChainIndex {
			// This is the target device - set the pin
			pinOverrides := map[string]bool{ref.PinName: value}
			segment, err = buildDRSegment(dev, pinOverrides)
			if err != nil {
				return fmt.Errorf("bsr: failed to build segment for device %s: %w", dev.ChainDev.Name(), err)
			}
			
			// Debug: show what we're writing for PA5
			if ref.PinName == "PA5" {
				cells, _ := dev.ChainDev.BoundaryCells()
				for _, cell := range cells {
					if cell.Port == "PA5" && (cell.Number >= 94 && cell.Number <= 96) {
						fmt.Printf("[BSR]   Cell %d (%s): %v\n", cell.Number, cell.Function, segment[cell.Number])
					}
				}
			}
		} else {
			// Other devices - keep them in HiZ
			segment, err = setAllPinsHiZ(dev)
			if err != nil {
				return fmt.Errorf("bsr: failed to build HiZ segment for device %s: %w", dev.ChainDev.Name(), err)
			}
		}

		globalDR = append(globalDR, segment...)
	}

	// Shift DR
	_, err := c.chain.ShiftDRBits(globalDR)
	if err != nil {
		return fmt.Errorf("bsr: failed to shift DR: %w", err)
	}

	// Update cached DR state
	c.currentDR = globalDR

	// Update pin states
	for _, dev := range c.Devices {
		for _, ps := range dev.Pins {
			if ps.Ref.ChainIndex == ref.ChainIndex && ps.Ref.PinName == ref.PinName {
				ps.Mode = PinOutput
				val := value
				ps.DrivenVal = &val
			} else if ps.Ref.ChainIndex == ref.ChainIndex {
				// Other pins on same device go to HiZ
				ps.Mode = PinHiZ
				ps.DrivenVal = nil
			}
			// Pins on other devices retain their state
		}
	}

	return nil
}

// CaptureAll performs a DR scan to capture the current state of all input pins.
// It returns a map from PinRef to the captured boolean value.
// This does not change the driven state of any pins.
func (c *Controller) CaptureAll() (map[PinRef]bool, error) {
	// Use the current DR state as TDI
	// The DR scan will capture the current pin states into TDO
	if len(c.currentDR) == 0 {
		// If no currentDR cached, build a default all-zeros vector
		c.currentDR = make([]bool, c.Layout.TotalBits)
	}

	tdo, err := c.chain.ShiftDRBits(c.currentDR)
	if err != nil {
		return nil, fmt.Errorf("bsr: failed to capture DR: %w", err)
	}

	// Decode the captured bits
	result, err := decodeDRBits(c.Layout, c.Devices, tdo)
	if err != nil {
		return nil, fmt.Errorf("bsr: failed to decode DR bits: %w", err)
	}

	return result, nil
}
