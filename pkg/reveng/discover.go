package reveng

import (
	"context"
	"fmt"
	"strings"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsr"
)

// Progress reports the current state of the reverse engineering process.
type Progress struct {
	Phase     string      // "init", "scanning", "finalizing"
	Driver    bsr.PinRef  // Currently driven pin
	Index     int         // Current pin index (0-based)
	Total     int         // Total number of pins to scan
	NetsFound int         // Number of multi-pin nets found so far
}

// DiscoverNetlist performs boundary-scan reverse engineering to discover
// board-level electrical connectivity. It drives each candidate pin and
// observes which other pins toggle in response.
//
// The algorithm:
// 1. Initialize: Enter EXTEST mode, set all pins to HiZ
// 2. For each candidate pin:
//    a. Set all pins HiZ
//    b. Drive pin LOW, capture baseline
//    c. Drive pin HIGH, capture high state
//    d. Drive pin LOW again, capture final state
//    e. Detect pins that toggled (0→1→0 or 1→0→1)
//    f. Connect driver to all togglers in netlist
// 3. Finalize: Build net list from union-find structure
//
// Parameters:
//   - ctx: Context for cancellation support
//   - ctl: BSR controller (must be initialized with a discovered chain)
//   - cfg: Configuration options (use DefaultConfig() if unsure)
//   - progress: Optional channel for progress updates (can be nil)
//
// Returns the discovered netlist or an error.
func DiscoverNetlist(
	ctx context.Context,
	ctl *bsr.Controller,
	cfg *Config,
	progress chan<- Progress,
) (*Netlist, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("reveng: invalid config: %w", err)
	}

	// Phase 1: Initialize
	if progress != nil {
		progress <- Progress{Phase: "init", Index: 0, Total: 0}
	}

	// Enter EXTEST mode
	if err := ctl.EnterExtest(); err != nil {
		return nil, fmt.Errorf("reveng: failed to enter EXTEST: %w", err)
	}

	// Set all pins to HiZ as baseline
	if err := ctl.SetAllPinsHiZ(); err != nil {
		return nil, fmt.Errorf("reveng: failed to set HiZ: %w", err)
	}

	// Select candidate pins
	candidates := selectCandidatePins(ctl, cfg)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("reveng: no candidate pins found")
	}

	// Initialize netlist
	nl := NewNetlist(candidates)

	// Phase 2: Scan each pin
	netsFound := 0

	for i, driver := range candidates {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Report progress
		if progress != nil {
			progress <- Progress{
				Phase:     "scanning",
				Driver:    driver,
				Index:     i,
				Total:     len(candidates),
				NetsFound: netsFound,
			}
		}

		// Perform toggle detection for this driver pin
		togglers, err := detectTogglers(ctl, driver, cfg)
		if err != nil {
			return nil, fmt.Errorf("reveng: failed to detect togglers for %s.%s: %w",
				driver.DeviceName, driver.PinName, err)
		}

		// Connect driver to all pins that toggled
		for _, toggler := range togglers {
			nl.Connect(driver, toggler)
		}

		// Update nets found count
		if len(togglers) > 0 {
			netsFound++
		}
	}

	// Phase 3: Finalize
	if progress != nil {
		progress <- Progress{
			Phase:     "finalizing",
			Index:     len(candidates),
			Total:     len(candidates),
			NetsFound: netsFound,
		}
	}

	nl.Finalize()

	return nl, nil
}

// selectCandidatePins filters the controller's pins to select which ones
// should be scanned during reverse engineering.
func selectCandidatePins(ctl *bsr.Controller, cfg *Config) []bsr.PinRef {
	var candidates []bsr.PinRef

	for _, dev := range ctl.Devices {
		// Check device filter
		if !cfg.ShouldScanDevice(dev.ChainDev.Name()) {
			continue
		}

		for pinName, ps := range dev.Pins {
			// Check pin pattern filter
			if !cfg.ShouldScanPin(pinName) {
				continue
			}

			// Skip JTAG pins if configured
			if cfg.SkipKnownJTAGPins && isJTAGPin(pinName) {
				continue
			}

			// Skip power pins if configured
			if cfg.SkipPowerPins && isPowerPin(pinName) {
				continue
			}

			candidates = append(candidates, ps.Ref)
		}
	}

	return candidates
}

// detectTogglers drives the given pin through a 0→1→0 cycle and detects
// which other pins toggled in response.
func detectTogglers(
	ctl *bsr.Controller,
	driver bsr.PinRef,
	cfg *Config,
) ([]bsr.PinRef, error) {
	// Set all pins to HiZ
	if err := ctl.SetAllPinsHiZ(); err != nil {
		return nil, err
	}

	// Drive LOW and capture baseline
	if err := ctl.DrivePin(driver, false); err != nil {
		// Pin can't be driven (no output cell) - skip it
		return nil, nil
	}
	baseline, err := ctl.CaptureAll()
	if err != nil {
		return nil, err
	}

	// Drive HIGH and capture
	if err := ctl.DrivePin(driver, true); err != nil {
		return nil, nil
	}
	high, err := ctl.CaptureAll()
	if err != nil {
		return nil, err
	}

	// Drive LOW again and capture
	if err := ctl.DrivePin(driver, false); err != nil {
		return nil, err
	}
	low2, err := ctl.CaptureAll()
	if err != nil {
		return nil, err
	}

	// Detect togglers
	return findTogglers(driver, baseline, high, low2, cfg), nil
}

// findTogglers analyzes captured pin states to find pins that toggled
// in response to the driver pin changing.
func findTogglers(
	driver bsr.PinRef,
	baseline, high, low2 map[bsr.PinRef]bool,
	cfg *Config,
) []bsr.PinRef {
	var togglers []bsr.PinRef

	for ref, baseVal := range baseline {
		// Skip the driver itself
		if ref.ChainIndex == driver.ChainIndex &&
			ref.DeviceName == driver.DeviceName &&
			ref.PinName == driver.PinName {
			continue
		}

		// Get values from all three captures
		highVal, okHigh := high[ref]
		low2Val, okLow2 := low2[ref]

		if !okHigh || !okLow2 {
			continue // Pin not captured in all phases
		}

		// Detect 0→1→0 pattern (toggled up)
		toggledUp := (baseVal == false && highVal == true && low2Val == false)

		// Detect 1→0→1 pattern (toggled down)
		toggledDown := (baseVal == true && highVal == false && low2Val == true)

		if cfg.RequireSymmetricToggle {
			// For strict mode, we would need to run the test twice
			// (once with 0→1→0 and once with 1→0→1)
			// For now, accept either pattern
			if toggledUp || toggledDown {
				togglers = append(togglers, ref)
			}
		} else {
			// Accept any toggle pattern
			if toggledUp || toggledDown {
				togglers = append(togglers, ref)
			}
		}
	}

	return togglers
}

// isJTAGPin returns true if the pin name appears to be a JTAG control pin.
func isJTAGPin(pinName string) bool {
	upper := strings.ToUpper(pinName)
	return strings.Contains(upper, "TCK") ||
		strings.Contains(upper, "TMS") ||
		strings.Contains(upper, "TDI") ||
		strings.Contains(upper, "TDO") ||
		strings.Contains(upper, "TRST") ||
		strings.Contains(upper, "JTAG")
}

// isPowerPin returns true if the pin name appears to be a power or ground pin.
func isPowerPin(pinName string) bool {
	upper := strings.ToUpper(pinName)
	return strings.Contains(upper, "VCC") ||
		strings.Contains(upper, "VDD") ||
		strings.Contains(upper, "VSS") ||
		strings.Contains(upper, "GND") ||
		strings.Contains(upper, "VBAT") ||
		strings.Contains(upper, "VREF")
}
