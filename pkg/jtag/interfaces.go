package jtag

import (
	"context"
	"fmt"

	"github.com/google/gousb"
)

// InterfaceKind categorizes adapter families.
type InterfaceKind string

const (
	InterfaceKindCMSISDAP InterfaceKind = "cmsis-dap"
	InterfaceKindPico     InterfaceKind = "picoprobe"
	InterfaceKindUnknown  InterfaceKind = "unknown"
	InterfaceKindSim      InterfaceKind = "simulator"
)

// InterfaceInfo describes a detected adapter interface/transport.
type InterfaceInfo struct {
	Kind        InterfaceKind
	Description string
	VendorID    uint16
	ProductID   uint16
	Serial      string
	Path        string
}

// Label returns a user-friendly description for the interface.
func (i InterfaceInfo) Label() string {
	if i.Description != "" {
		return i.Description
	}
	if i.Kind != "" {
		return fmt.Sprintf("%s (%04X:%04X)", string(i.Kind), i.VendorID, i.ProductID)
	}
	return fmt.Sprintf("Interface %04X:%04X", i.VendorID, i.ProductID)
}

// DiscoverInterfaces enumerates connected JTAG-capable USB devices that match
// known VID/PID pairs. It always returns at least the simulator entry so the
// user can exercise the UI without hardware connected.
func DiscoverInterfaces(ctx context.Context) ([]InterfaceInfo, error) {
	var results []InterfaceInfo
	usb := gousb.NewContext()
	defer usb.Close()

	_, err := usb.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		if info, ok := classifyUSBDevice(desc); ok {
			results = append(results, info)
		}
		return false
	})
	if err != nil && err != gousb.ErrorAccess {
		return results, err
	}

	results = append(results, InterfaceInfo{
		Kind:        InterfaceKindSim,
		Description: "Simulator (no hardware)",
	})

	return results, nil
}

func classifyUSBDevice(desc *gousb.DeviceDesc) (InterfaceInfo, bool) {
	for _, known := range knownCMSISDAPVIDPIDs {
		if uint16(desc.Vendor) == known.VendorID && uint16(desc.Product) == known.ProductID {
			return InterfaceInfo{
				Kind:        InterfaceKindCMSISDAP,
				Description: known.Description,
				VendorID:    known.VendorID,
				ProductID:   known.ProductID,
			}, true
		}
	}
	for _, known := range knownPicoVIDPIDs {
		if uint16(desc.Vendor) == known.VendorID && uint16(desc.Product) == known.ProductID {
			return InterfaceInfo{
				Kind:        InterfaceKindPico,
				Description: known.Description,
				VendorID:    known.VendorID,
				ProductID:   known.ProductID,
			}, true
		}
	}
	return InterfaceInfo{}, false
}

type knownUSBDevice struct {
	VendorID    uint16
	ProductID   uint16
	Description string
}

var knownCMSISDAPVIDPIDs = []knownUSBDevice{
	{VendorID: VendorIDRaspberryPi, ProductID: ProductIDCMSISDAP, Description: "Raspberry Pi CMSIS-DAP"},
	{VendorID: 0x0d28, ProductID: 0x0204, Description: "DAPLink CMSIS-DAP"},
	{VendorID: 0x1366, ProductID: 0x0101, Description: "SEGGER J-Link CMSIS-DAP"},
}

var knownPicoVIDPIDs = []knownUSBDevice{
	{VendorID: 0x2e8a, ProductID: 0x000c, Description: "PicoProbe"},
	{VendorID: 0x2e8a, ProductID: 0x000a, Description: "Raspberry Pi Pico (CDC/JTAG)"},
}
