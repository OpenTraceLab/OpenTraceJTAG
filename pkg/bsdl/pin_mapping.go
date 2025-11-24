package bsdl

import (
	"strings"
)

// PinMapping maps boundary scan register indices to pin names.
type PinMapping struct {
	BSRIndexToPin map[int]string // BSR index → pin name (e.g., 0 → "PA0")
	PinToBSRIndex map[string]int // pin name → INPUT cell BSR index
}

// ExtractPinMapping extracts pin names from the boundary register.
func ExtractPinMapping(bsdlFile *BSDLFile) *PinMapping {
	if bsdlFile == nil || bsdlFile.Entity == nil {
		return &PinMapping{
			BSRIndexToPin: make(map[int]string),
			PinToBSRIndex: make(map[string]int),
		}
	}
	
	mapping := &PinMapping{
		BSRIndexToPin: make(map[int]string),
		PinToBSRIndex: make(map[string]int),
	}
	
	// Get boundary register cells
	cells, err := bsdlFile.Entity.GetBoundaryCells()
	if err != nil {
		// Return empty mapping if boundary cells can't be extracted
		return mapping
	}
	
	for _, cell := range cells {
		// Skip internal cells (marked with "*")
		if cell.Port == "*" || cell.Port == "" {
			continue
		}
		
		// Clean up pin name (remove quotes, whitespace)
		pinName := strings.Trim(cell.Port, "\" ")
		if pinName == "" {
			continue
		}
		
		// Map BSR index to pin name
		mapping.BSRIndexToPin[cell.Number] = pinName
		
		// Map pin name to INPUT cell BSR index (prefer INPUT over OUTPUT/CONTROL)
		if strings.HasPrefix(strings.ToUpper(cell.Function), "INPUT") {
			mapping.PinToBSRIndex[pinName] = cell.Number
		} else if _, exists := mapping.PinToBSRIndex[pinName]; !exists {
			// Fallback to first occurrence if no INPUT cell found yet
			mapping.PinToBSRIndex[pinName] = cell.Number
		}
	}
	
	return mapping
}

// GetPinName returns the pin name for a given BSR index.
func (pm *PinMapping) GetPinName(bsrIndex int) string {
	if name, ok := pm.BSRIndexToPin[bsrIndex]; ok {
		return name
	}
	return ""
}

// GetBSRIndex returns the BSR index for a given pin name.
func (pm *PinMapping) GetBSRIndex(pinName string) int {
	if idx, ok := pm.PinToBSRIndex[pinName]; ok {
		return idx
	}
	return -1
}

// GetAllPins returns all pin names in the mapping.
func (pm *PinMapping) GetAllPins() []string {
	pins := make([]string, 0, len(pm.PinToBSRIndex))
	for pin := range pm.PinToBSRIndex {
		pins = append(pins, pin)
	}
	return pins
}
