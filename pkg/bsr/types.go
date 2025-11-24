package bsr

import "github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"

// PinRef uniquely identifies a physical board pin across the entire chain.
type PinRef struct {
	ChainIndex int    // Index into chain.Devices (0 = closest to TDI)
	DeviceName string // Entity name from BSDL, e.g., "STM32F103"
	PinName    string // Package pin name from BSDL, e.g., "PA0", "A1"
}

// PinMode represents the current drive state of a pin.
type PinMode int

const (
	// PinHiZ indicates the pin is in high-impedance (tri-stated) mode.
	PinHiZ PinMode = iota
	// PinOutput indicates the pin is actively driving a value (0 or 1).
	PinOutput
)

// PinState tracks runtime state for a single pin.
type PinState struct {
	Ref       PinRef
	Mode      PinMode
	DrivenVal *bool // Non-nil when Mode == PinOutput
	LastRead  *bool // Last captured input value from CaptureAll
}

// DeviceRuntime wraps a chain.Device with boundary-scan-specific runtime state.
type DeviceRuntime struct {
	ChainDev *chain.Device

	// Quick lookup: package pin name -> runtime state
	Pins map[string]*PinState

	// Precomputed from BSDL:
	boundaryLength int     // Total boundary register length
	extestOpcode   []bool  // Precomputed EXTEST instruction bits
	bypassOpcode   []bool  // Precomputed BYPASS instruction bits
}

// DRMapEntry maps a global DR bit index to a specific device and boundary cell.
type DRMapEntry struct {
	DeviceIndex int // Index into Controller.Devices
	CellIndex   int // Index into device's boundary cells (0-based)
}

// DRLayout precomputes the global DR bit layout for the entire chain.
// The DR chain is the concatenation of all devices' boundary scan registers.
type DRLayout struct {
	TotalBits int          // Total bits in the global DR chain
	Cells     []DRMapEntry // Length = TotalBits; maps each bit to device+cell
}
