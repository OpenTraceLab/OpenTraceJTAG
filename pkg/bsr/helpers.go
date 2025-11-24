package bsr

import (
	"fmt"
	"strings"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
)

// buildDRLayout constructs the global DR bit map from a list of devices.
// The DR chain is the concatenation of all devices' boundary scan registers,
// ordered from TDO to TDI (device 0 is closest to TDI, last device closest to TDO).
func buildDRLayout(devices []*DeviceRuntime) *DRLayout {
	layout := &DRLayout{
		TotalBits: 0,
		Cells:     []DRMapEntry{},
	}

	// Devices are ordered from TDI (index 0) to TDO (last index).
	// In DR scan, bits shift from TDI toward TDO, so we need to reverse
	// the order when building the layout: TDO device comes first in the DR vector.
	for devIdx := len(devices) - 1; devIdx >= 0; devIdx-- {
		dev := devices[devIdx]
		for cellIdx := 0; cellIdx < dev.boundaryLength; cellIdx++ {
			layout.Cells = append(layout.Cells, DRMapEntry{
				DeviceIndex: devIdx,
				CellIndex:   cellIdx,
			})
		}
		layout.TotalBits += dev.boundaryLength
	}

	return layout
}

// CanDrivePin returns true if the pin has an output cell and can be driven
func CanDrivePin(dev *DeviceRuntime, pinName string) bool {
	cells, err := dev.ChainDev.BoundaryCells()
	if err != nil {
		return false
	}

	cellsByPort := make(map[string][]*bsdl.BoundaryCell)
	for i := range cells {
		port := strings.ToUpper(cells[i].Port)
		cellsByPort[port] = append(cellsByPort[port], &cells[i])
	}

	key := strings.ToUpper(pinName)
	pinCells, ok := cellsByPort[key]
	if !ok {
		return false
	}

	// Check if pin has an output cell
	for _, cell := range pinCells {
		if strings.HasPrefix(strings.ToUpper(cell.Function), "OUTPUT") {
			return true
		}
	}

	return false
}

// buildDRSegment creates the DR bit vector for a single device.
// By default, all cells are set to their safe values.
// pinOverrides maps pin name -> output value for pins that should be driven.
func buildDRSegment(dev *DeviceRuntime, pinOverrides map[string]bool) ([]bool, error) {
	cells, err := dev.ChainDev.BoundaryCells()
	if err != nil {
		return nil, fmt.Errorf("bsr: failed to get boundary cells: %w", err)
	}

	// Start with safe values for all cells
	bits := make([]bool, dev.boundaryLength)
	for _, cell := range cells {
		if cell.Number >= len(bits) {
			return nil, fmt.Errorf("bsr: cell %d exceeds boundary length %d", cell.Number, len(bits))
		}
		switch strings.ToUpper(strings.TrimSpace(cell.Safe)) {
		case "1":
			bits[cell.Number] = true
		case "0":
			bits[cell.Number] = false
		// X or anything else: leave as false
		}
	}

	// Build a map of control cells for outputs
	// Map package pin names to cells
	pinMap := dev.ChainDev.PinMap()
	cellByNumber := make(map[int]*bsdl.BoundaryCell)
	cellsByPort := make(map[string][]*bsdl.BoundaryCell)
	for i := range cells {
		cell := &cells[i]
		cellByNumber[cell.Number] = cell

		// Get package pin name
		packagePin, ok := pinMap[cell.Port]
		if !ok {
			packagePin = cell.Port
		}
		key := strings.ToUpper(packagePin)
		cellsByPort[key] = append(cellsByPort[key], cell)
	}

	// Apply pin overrides
	for pinName, outputVal := range pinOverrides {
		if err := applyPinToDRBits(bits, pinName, outputVal, cellsByPort, cellByNumber); err != nil {
			return nil, fmt.Errorf("bsr: failed to apply pin %s: %w", pinName, err)
		}
	}

	return bits, nil
}

// applyPinToDRBits sets the output and control cells for a specific pin.
func applyPinToDRBits(bits []bool, pinName string, outputVal bool, cellsByPort map[string][]*bsdl.BoundaryCell, cellByNumber map[int]*bsdl.BoundaryCell) error {
	key := strings.ToUpper(pinName)
	cells, ok := cellsByPort[key]
	if !ok {
		return fmt.Errorf("pin %s not found in boundary cells", pinName)
	}

	// Find the output cell
	var outputCell *bsdl.BoundaryCell
	for _, cell := range cells {
		if strings.HasPrefix(strings.ToUpper(cell.Function), "OUTPUT") {
			outputCell = cell
			break
		}
	}

	if outputCell == nil {
		return fmt.Errorf("no output cell for pin %s", pinName)
	}

	// Set the output cell value
	if outputCell.Number >= len(bits) {
		return fmt.Errorf("output cell %d exceeds boundary length", outputCell.Number)
	}
	bits[outputCell.Number] = outputVal

	// Enable the output via the control cell (if present)
	if outputCell.Control >= 0 {
		controlCell := cellByNumber[outputCell.Control]
		if controlCell == nil {
			return fmt.Errorf("control cell %d not found", outputCell.Control)
		}

		// Determine the enable value
		// The Disable field indicates what value disables the output
		// So the enable value is the opposite
		disableVal := outputCell.Disable
		if disableVal == -1 && controlCell != nil {
			disableVal = controlCell.Disable
		}

		// If disable == 0, then enable == 1
		// If disable == 1, then enable == 0
		// If disable == -1, assume enable == 1 (default to output enabled)
		enableVal := true
		if disableVal == 0 {
			enableVal = true
		} else if disableVal == 1 {
			enableVal = false
		}

		if controlCell.Number >= len(bits) {
			return fmt.Errorf("control cell %d exceeds boundary length", controlCell.Number)
		}
		bits[controlCell.Number] = enableVal
	}

	return nil
}

// setAllPinsHiZ configures the DR segment so all pins are tri-stated.
func setAllPinsHiZ(dev *DeviceRuntime) ([]bool, error) {
	cells, err := dev.ChainDev.BoundaryCells()
	if err != nil {
		return nil, fmt.Errorf("bsr: failed to get boundary cells: %w", err)
	}

	bits := make([]bool, dev.boundaryLength)

	// Set all cells to safe values
	for _, cell := range cells {
		if cell.Number >= len(bits) {
			return nil, fmt.Errorf("bsr: cell %d exceeds boundary length %d", cell.Number, len(bits))
		}
		switch strings.ToUpper(strings.TrimSpace(cell.Safe)) {
		case "1":
			bits[cell.Number] = true
		case "0":
			bits[cell.Number] = false
		}
	}

	// Build control cell map
	cellByNumber := make(map[int]*bsdl.BoundaryCell)
	for i := range cells {
		cellByNumber[cells[i].Number] = &cells[i]
	}

	// Disable all outputs by setting their control cells appropriately
	for _, cell := range cells {
		if strings.HasPrefix(strings.ToUpper(cell.Function), "OUTPUT") && cell.Control >= 0 {
			controlCell := cellByNumber[cell.Control]
			if controlCell == nil {
				continue
			}

			// Set control cell to disable value
			disableVal := cell.Disable
			if disableVal == -1 && controlCell != nil {
				disableVal = controlCell.Disable
			}

			// If disable is 0, write 0 to control cell
			// If disable is 1, write 1 to control cell
			// If disable is -1, default to 0 (assume 0 disables)
			if disableVal >= 0 && disableVal <= 1 {
				if controlCell.Number < len(bits) {
					bits[controlCell.Number] = (disableVal == 1)
				}
			} else {
				// Default: assume 0 disables
				if controlCell.Number < len(bits) {
					bits[controlCell.Number] = false
				}
			}
		}
	}

	return bits, nil
}

// decodeDRBits extracts input pin values from a captured DR bit vector.
func decodeDRBits(layout *DRLayout, devices []*DeviceRuntime, drBits []bool) (map[PinRef]bool, error) {
	if len(drBits) != layout.TotalBits {
		return nil, fmt.Errorf("bsr: DR bit count mismatch: got %d, expected %d", len(drBits), layout.TotalBits)
	}

	result := make(map[PinRef]bool)

	for bitIdx, entry := range layout.Cells {
		dev := devices[entry.DeviceIndex]
		cells, err := dev.ChainDev.BoundaryCells()
		if err != nil {
			continue
		}

		if entry.CellIndex >= len(cells) {
			continue
		}

		cell := &cells[entry.CellIndex]

		// Only capture input cells
		if !strings.HasPrefix(strings.ToUpper(cell.Function), "INPUT") {
			continue
		}

		// Skip internal cells (port == "*")
		if cell.Port == "*" {
			continue
		}

		// Get the pin name from the pin map
		pinMap := dev.ChainDev.PinMap()
		packagePin, ok := pinMap[cell.Port]
		if !ok {
			// If no pin map entry, use the port name directly
			packagePin = cell.Port
		}

		ref := PinRef{
			ChainIndex: dev.ChainDev.Position,
			DeviceName: dev.ChainDev.Name(),
			PinName:    packagePin,
		}

		result[ref] = drBits[bitIdx]

		// Update the PinState if it exists
		if ps, ok := dev.Pins[packagePin]; ok {
			val := drBits[bitIdx]
			ps.LastRead = &val
		}
	}

	return result, nil
}
