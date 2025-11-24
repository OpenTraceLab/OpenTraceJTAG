package deviceinfo

import "github.com/OpenTraceLab/OpenTraceJTAG/pkg/idcode"

// DeviceInfo contains rich information about a JTAG device
type DeviceInfo struct {
	// Key fields
	IDCode       idcode.IDCode
	Manufacturer idcode.Manufacturer

	// Human-friendly
	Name        string // "STM32F407VG"
	Family      string // "STM32F4"
	Description string // "ARM Cortex-M4 MCU with FPU"
	Package     string // "LQFP-100", if known

	// Capabilities / hints
	HasBoundaryScan bool
	HasARMCore      bool
	ARMCore         string // "Cortex-M4", "Cortex-A9", etc.
	IsFPGA          bool
	IsCPLD          bool
	IsMCU           bool
	IsSoC           bool

	// JTAG specifics
	IRLength     int
	BSDLURL      string
	DatasheetURL string
}
