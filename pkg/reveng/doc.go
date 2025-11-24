// Package reveng (Reverse Engineering) provides algorithms for discovering
// board-level electrical connectivity using JTAG boundary-scan.
//
// This package implements the "drive one pin, watch everyone else" algorithm
// to build a netlist of connected pins without requiring board schematics.
//
// # Overview
//
// The reverse engineering process:
//  1. Enter EXTEST mode to control all device pins via boundary scan
//  2. For each candidate pin:
//     - Drive the pin LOW and capture all input pins
//     - Drive the pin HIGH and capture all input pins
//     - Drive the pin LOW again and capture all input pins
//     - Detect which pins toggled (indicating electrical connection)
//  3. Build a netlist using union-find to group connected pins
//  4. Export to JSON or KiCad format
//
// # Usage
//
// Basic usage:
//
//	// 1. Set up BSR controller (from discovered chain)
//	bsrCtl, err := bsr.NewController(chain)
//
//	// 2. Configure reverse engineering
//	cfg := reveng.DefaultConfig()
//	cfg.SkipKnownJTAGPins = true
//
//	// 3. Create progress channel (optional)
//	progressCh := make(chan reveng.Progress)
//	go func() {
//		for p := range progressCh {
//			fmt.Printf("[%d/%d] Scanning %s.%s\n",
//				p.Index, p.Total, p.Driver.DeviceName, p.Driver.PinName)
//		}
//	}()
//
//	// 4. Run discovery
//	ctx := context.Background()
//	netlist, err := reveng.DiscoverNetlist(ctx, bsrCtl, cfg, progressCh)
//	close(progressCh)
//
//	// 5. Export results
//	jsonData, _ := netlist.ExportJSON()
//	os.WriteFile("netlist.json", jsonData, 0644)
//
// # Algorithm Details
//
// The toggle detection algorithm works by observing electrical continuity:
// - When pin A is driven HIGH, any pin electrically connected to A will
//   read HIGH (assuming tri-state buffers are disabled)
// - When pin A is driven LOW, connected pins will read LOW
// - Pins that toggle 0→1→0 or 1→0→1 in sync with the driver are connected
//
// The algorithm handles:
// - Tri-state outputs (via EXTEST control cells)
// - Bidirectional pins (both input and output cells)
// - Multiple drivers on the same net (detected as togglers)
//
// # Performance
//
// Time complexity: O(n²) where n is the number of candidate pins
// - Each pin requires 3 DR shifts (baseline, high, low)
// - For 100 pins: ~300 DR shifts + overhead
// - Typical speed: 10-50 pins/second depending on adapter
//
// Memory complexity: O(n) for union-find and netlist storage
//
// # Limitations
//
//   - Requires all devices to support EXTEST instruction
//   - Cannot detect resistor values (only connectivity)
//   - May miss high-impedance connections
//   - Pull-up/down resistors can cause false positives (use filters)
//   - Assumes no active circuitry interfering with boundary scan
//
// # Configuration Options
//
// Key configuration options:
//   - SkipKnownJTAGPins: Exclude JTAG control pins (recommended: true)
//   - SkipPowerPins: Exclude power/ground pins (recommended: true)
//   - OnlyDevices: Limit scanning to specific devices
//   - OnlyPinPattern: Regex filter for pin names
//   - RepeatsPerPin: Number of toggle cycles (default: 1)
//
// # Export Formats
//
// Supported export formats:
//   - JSON: Machine-readable format with full metadata
//   - KiCad: Standard netlist format for PCB design tools
//
// # See Also
//
// For the underlying boundary-scan runtime, see package bsr.
// For JTAG chain discovery, see package chain.
package reveng
