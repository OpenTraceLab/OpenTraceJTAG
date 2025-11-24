// Package bsr (Boundary Scan Runtime) provides a pin-centric API for
// boundary-scan operations on JTAG chains.
//
// This package builds on top of the chain package, translating high-level
// pin operations (HiZ, Drive, Capture) into low-level JTAG DR shifts.
// It manages the runtime state of all pins across all devices in the chain
// and handles the complex mapping between pin names and boundary scan cells.
//
// # Overview
//
// The BSR package provides:
//   - PinRef: A unique identifier for physical board pins
//   - Controller: Manages boundary-scan operations on a JTAG chain
//   - Operations: EnterExtest, SetAllPinsHiZ, DrivePin, CaptureAll
//
// # Usage
//
// Basic usage follows this pattern:
//
//	// 1. Discover the chain
//	chainCtl := chain.NewController(adapter, repo)
//	ch, err := chainCtl.Discover(deviceCount)
//
//	// 2. Create BSR controller
//	bsrCtl, err := bsr.NewController(ch)
//
//	// 3. Enter EXTEST mode
//	err = bsrCtl.EnterExtest()
//
//	// 4. Set all pins to high-impedance
//	err = bsrCtl.SetAllPinsHiZ()
//
//	// 5. Drive a specific pin
//	pinRef := bsr.PinRef{
//		ChainIndex: 0,
//		DeviceName: "STM32F103",
//		PinName:    "PA0",
//	}
//	err = bsrCtl.DrivePin(pinRef, true) // Drive high
//
//	// 6. Capture all input pins
//	values, err := bsrCtl.CaptureAll()
//	for ref, value := range values {
//		fmt.Printf("%s.%s = %v\n", ref.DeviceName, ref.PinName, value)
//	}
//
// # Boundary Scan Concepts
//
// Boundary scan (IEEE 1149.1) allows external control and observation of device
// pins through a shift register (the boundary scan register, or BSR). Each pin
// typically has three boundary scan cells:
//   - Input cell: Captures the value on the pin
//   - Output cell: Drives a value to the pin
//   - Control cell: Enables/disables the output driver (for tri-state)
//
// The EXTEST instruction connects the BSR to the device pins, allowing software
// to control pin states independent of the device's internal logic.
//
// # DR Layout
//
// The global DR chain is the concatenation of all devices' boundary scan registers.
// Devices are ordered from TDI (closest, index 0) to TDO (farthest, last index).
// During DR scan, bits shift from TDI toward TDO, so the DR vector must be built
// in reverse order: TDO device first, TDI device last.
//
// # Performance
//
// The Controller caches the current DR state to minimize USB traffic. Operations
// like DrivePin only modify the necessary bits and reuse the cached vector.
// For bulk operations, consider using the underlying chain.Batch API if the
// pin-centric abstraction is not needed.
//
// # Limitations
//
//   - Only EXTEST mode is currently supported (not SAMPLE or PRELOAD)
//   - Pin filtering excludes power pins by name heuristics (VCC, GND, etc.)
//   - Control cell disable logic assumes common BSDL conventions
//   - No support for differential pairs or multi-bit buses
package bsr
