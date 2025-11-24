package bsdl

import (
	"strconv"
	"strings"
)

// Instruction represents a JTAG instruction with name and opcode
type Instruction struct {
	Name   string // e.g., "BYPASS", "EXTEST", "IDCODE"
	Opcode string // Binary string, e.g., "11111"
}

// BoundaryCell represents a single boundary scan cell
type BoundaryCell struct {
	Number   int    // Cell number in the boundary register
	CellType string // e.g., "BC_1", "BC_4", "BC_7"
	Port     string // Port name or "*" for internal
	Function string // e.g., "input", "output3", "control"
	Safe     string // Safe value (X, 0, 1)
	Control  int    // Control cell number (for output3)
	Disable  int    // Disable value
	Result   string // Result value (Z for tri-state)
}

// TAPConfig represents TAP (Test Access Port) configuration
type TAPConfig struct {
	ScanIn    string  // TAP_SCAN_IN signal name
	ScanOut   string  // TAP_SCAN_OUT signal name
	ScanMode  string  // TAP_SCAN_MODE signal name
	ScanReset string  // TAP_SCAN_RESET signal name
	ScanClock string  // TAP_SCAN_CLOCK signal name
	MaxFreq   float64 // Maximum TCK frequency in Hz
	Edge      string  // Clock edge: "BOTH", "RISING", "FALLING"
}

// DeviceInfo represents IEEE 1149.1 device information
type DeviceInfo struct {
	IDCode             string // 32-bit ID code (may contain wildcards)
	UserCode           string // 32-bit user code
	InstructionLength  int    // IR length in bits
	InstructionCapture string // IR capture pattern (may contain wildcards)
	BoundaryLength     int    // Boundary register length in bits
}

// GetInstructions extracts instruction opcodes from INSTRUCTION_OPCODE attribute
// Format: "BYPASS (11111), EXTEST (00000), ..."
func GetInstructions(expr *Expression) []Instruction {
	if expr == nil {
		return nil
	}

	// Get the concatenated string
	str := expr.GetConcatenatedString()
	if str == "" {
		return nil
	}

	var instructions []Instruction

	// Split by comma
	parts := strings.Split(str, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Find instruction name and opcode
		// Format: "NAME (OPCODE)" or "NAME(OPCODE)"
		openParen := strings.Index(part, "(")
		closeParen := strings.Index(part, ")")

		if openParen > 0 && closeParen > openParen {
			name := strings.TrimSpace(part[:openParen])
			opcode := strings.TrimSpace(part[openParen+1 : closeParen])

			instructions = append(instructions, Instruction{
				Name:   name,
				Opcode: opcode,
			})
		}
	}

	return instructions
}

// ParseBinaryString converts a binary string (potentially with wildcards) to uint32
// Returns the value and a mask indicating which bits are wildcards
func ParseBinaryString(s string) (value uint32, mask uint32, hasWildcards bool) {
	s = strings.TrimSpace(s)

	for _, ch := range s {
		switch ch {
		case '1':
			value <<= 1
			value |= 1
			mask <<= 1
			mask |= 1
		case '0':
			value <<= 1
			// value bit stays 0
			mask <<= 1
			mask |= 1
		case 'X', 'x':
			value <<= 1
			mask <<= 1
			// Wildcard - value bit is 0, mask bit is 0
			hasWildcards = true
		default:
			// Ignore other characters (whitespace, etc.)
			continue
		}
	}

	return value, mask, hasWildcards
}

// GetDeviceInfo extracts device identification information from attributes
func (e *Entity) GetDeviceInfo() *DeviceInfo {
	info := &DeviceInfo{}

	attrs := e.GetAttributes()
	for _, attr := range attrs {
		if attr.Spec == nil {
			continue
		}

		switch attr.Spec.Name {
		case "INSTRUCTION_LENGTH":
			if val, ok := attr.Spec.Is.GetInteger(); ok {
				info.InstructionLength = val
			}

		case "INSTRUCTION_CAPTURE":
			info.InstructionCapture = attr.Spec.Is.GetConcatenatedString()

		case "BOUNDARY_LENGTH":
			if val, ok := attr.Spec.Is.GetInteger(); ok {
				info.BoundaryLength = val
			}

		case "IDCODE_REGISTER":
			info.IDCode = attr.Spec.Is.GetConcatenatedString()

		case "USERCODE_REGISTER":
			info.UserCode = attr.Spec.Is.GetConcatenatedString()
		}
	}

	return info
}

// GetInstructionOpcodes returns the instruction set for this device
func (e *Entity) GetInstructionOpcodes() []Instruction {
	attrs := e.GetAttributes()
	for _, attr := range attrs {
		if attr.Spec != nil && attr.Spec.Name == "INSTRUCTION_OPCODE" {
			return GetInstructions(attr.Spec.Is)
		}
	}
	return nil
}

// GetTAPConfig extracts TAP configuration from attributes
func (e *Entity) GetTAPConfig() *TAPConfig {
	config := &TAPConfig{}

	attrs := e.GetAttributes()
	for _, attr := range attrs {
		if attr.Spec == nil {
			continue
		}

		switch attr.Spec.Name {
		case "TAP_SCAN_IN":
			config.ScanIn = attr.Spec.Of

		case "TAP_SCAN_OUT":
			config.ScanOut = attr.Spec.Of

		case "TAP_SCAN_MODE":
			config.ScanMode = attr.Spec.Of

		case "TAP_SCAN_RESET":
			config.ScanReset = attr.Spec.Of

		case "TAP_SCAN_CLOCK":
			config.ScanClock = attr.Spec.Of

			// Parse the tuple value (freq, edge)
			if attr.Spec.Is != nil && len(attr.Spec.Is.Terms) > 0 {
				if tuple := attr.Spec.Is.Terms[0].Tuple; tuple != nil && len(tuple.Values) >= 1 {
					// First value is the frequency
					if len(tuple.Values[0].Terms) > 0 {
						if tuple.Values[0].Terms[0].Real != nil {
							config.MaxFreq = *tuple.Values[0].Terms[0].Real
						} else if tuple.Values[0].Terms[0].Integer != nil {
							config.MaxFreq = float64(*tuple.Values[0].Terms[0].Integer)
						}
					}

					// Second value is the edge type
					if len(tuple.Values) >= 2 && len(tuple.Values[1].Terms) > 0 {
						if tuple.Values[1].Terms[0].Ident != nil {
							config.Edge = *tuple.Values[1].Terms[0].Ident
						}
					}
				}
			}
		}
	}

	return config
}

// GetPinMap extracts the physical pin mapping from PIN_MAP_STRING constant
// Returns a map of signal name to pin number
func (e *Entity) GetPinMap() map[string]string {
	pinMap := make(map[string]string)

	attrs := e.GetAttributes()
	for _, attr := range attrs {
		if attr.Constant == nil {
			continue
		}

		// Look for PIN_MAP_STRING constants
		if attr.Constant.Type == "PIN_MAP_STRING" {
			// Parse the concatenated string
			str := attr.Constant.Value.GetConcatenatedString()

			// Format: "SIGNAL_NAME : PIN_NUMBER ,"
			lines := strings.Split(str, ",")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// Split by colon
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					signal := strings.TrimSpace(parts[0])
					pin := strings.TrimSpace(parts[1])
					pinMap[signal] = pin
				}
			}
		}
	}

	return pinMap
}

// OpcodeToUint converts a binary opcode string to uint
func OpcodeToUint(opcode string) (uint, error) {
	val, err := strconv.ParseUint(opcode, 2, 32)
	return uint(val), err
}
