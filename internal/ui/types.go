package newui

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"
	
	"gioui.org/f32"
	"gioui.org/widget"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
)

// ChainDevice represents a device in the JTAG chain with UI state.
type ChainDevice struct {
	Index          int
	IDCode         uint32
	IDCodeInfo     jtag.IDCodeInfo
	Name           string
	IRLength       int
	BSRLength      int
	BSDLPath       string
	BSDLFile       *bsdl.BSDLFile
	PinMapping     *bsdl.PinMapping
	FootprintType  string
	PinCount       int     // Number of pins/balls
	PackageWidth   float32 // Package width in mm (for TSOP/QFP/QFN)
	PackageHeight  float32 // Package height in mm (for TSOP/QFP/QFN)
	BallPitch      float32 // Ball pitch in mm (for BGA)
	State          string  // "discovered", "bsdl_assigned", "footprint_assigned", "ready"
	PinStates      map[int]string // Pin number → state ("high", "low", "hi-z")
	pinStatesMu    sync.RWMutex   // Protects PinStates
	ComponentRef   string  // KiCad component reference (e.g., "U1", "U2")
	
	// UI widgets
	bsdlBtn       widget.Clickable
	footprintBtn  widget.Clickable
	componentBtn  widget.Clickable
}

// RenderedPad stores the screen-space bounds of a rendered pad for hit testing
type RenderedPad struct {
	DeviceIndex int
	PinNumber   int
	PinName     string
	Bounds      image.Rectangle // Screen-space bounds after transform
	Center      f32.Point        // Exact center position in float coordinates
}

// RatsnestLine represents a connection line between two pads
type RatsnestLine struct {
	DeviceA int
	PinA    int
	DeviceB int
	PinB    int
	NetID   int
	Color   color.NRGBA
}

// GetPinName returns the logical pin name for a physical pin number from BSDL
// Returns empty string if BSDL not loaded or pin not found
func (d *ChainDevice) GetPinName(pinNumber int) string {
	if d.BSDLFile == nil || d.BSDLFile.Entity == nil {
		return ""
	}
	
	// Get pin map from BSDL (signal name → pin number/coordinate)
	pinMap := d.BSDLFile.Entity.GetPinMap()
	
	// Reverse lookup: find signal name for this pin number
	pinStr := fmt.Sprintf("%d", pinNumber)
	for signalName, pinNum := range pinMap {
		if pinNum == pinStr {
			return signalName
		}
	}
	
	return ""
}

// SetPinState sets the state of a pin (thread-safe)
func (d *ChainDevice) SetPinState(pinNumber int, state string) {
	d.pinStatesMu.Lock()
	defer d.pinStatesMu.Unlock()
	if d.PinStates == nil {
		d.PinStates = make(map[int]string)
	}
	d.PinStates[pinNumber] = state
}

// GetPinState gets the state of a pin (thread-safe)
func (d *ChainDevice) GetPinState(pinNumber int) string {
	d.pinStatesMu.RLock()
	defer d.pinStatesMu.RUnlock()
	return d.PinStates[pinNumber]
}

// IsPowerPin returns true if the pin is a power or ground pin
func (d *ChainDevice) IsPowerPin(pinName string) bool {
	upper := strings.ToUpper(pinName)
	// Common power/ground pin patterns
	powerPatterns := []string{
		"VCC", "VDD", "VDDA", "VDDIO", "VBAT", "VREF",
		"GND", "VSS", "VSSA", "GNDA",
	}
	for _, pattern := range powerPatterns {
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	return false
}

type workspaceView struct {
	Description  string
	QuickActions []string
	Metrics      []workspaceMetric
	Sections     []workspaceSection
}

type workspaceMetric struct {
	Label string
	Value string
	Sub   string
}

type workspaceSection struct {
	Title string
	Items []string
}
