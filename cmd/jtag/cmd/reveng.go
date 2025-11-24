package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsr"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/reveng"
	"github.com/spf13/cobra"
)

var (
	// Flags for reveng command
	revengOutputJSON  string
	revengOutputKiCad string
	revengRepeats     int
	revengSkipJTAG    bool
	revengSkipPower   bool
	revengOnlyDevices []string
	revengOnlyPins    string
	revengTimeout     int // timeout in seconds
)

var revengCmd = &cobra.Command{
	Use:   "reveng",
	Short: "Reverse engineer board connectivity via boundary-scan",
	Long: `Discover electrical connectivity between pins using JTAG boundary-scan.

The reveng command performs the "drive one pin, watch everyone else" algorithm
to build a netlist of connected pins without requiring board schematics.

The algorithm:
  1. Enter EXTEST mode to control all device pins
  2. For each candidate pin:
     - Drive pin LOW and capture all inputs
     - Drive pin HIGH and capture all inputs
     - Drive pin LOW again and capture all inputs
     - Detect pins that toggled (electrically connected)
  3. Build netlist from detected connections
  4. Export to JSON and/or KiCad format

Examples:
  # Basic usage with simulator
  jtag reveng --adapter simulator --count 2 --bsdl testdata --output netlist.json

  # Real hardware with CMSIS-DAP
  jtag reveng --adapter cmsisdap --count 2 --bsdl /path/to/bsdl \
    --output netlist.json --output-kicad netlist.net

  # Filter specific devices or pins
  jtag reveng --adapter cmsisdap --count 3 --bsdl testdata \
    --only-devices "STM32,FLASH" --only-pins "P[AB][0-9]+" \
    --output result.json

  # Skip JTAG and power pins (recommended)
  jtag reveng --adapter cmsisdap --count 2 --bsdl testdata \
    --skip-jtag --skip-power --output netlist.json

Performance:
  - Time complexity: O(n²) where n = number of pins
  - Typical speed: 10-50 pins/second
  - 100 pins ≈ 2-10 minutes depending on adapter speed`,
	RunE: runReveng,
}

func init() {
	rootCmd.AddCommand(revengCmd)

	// Reuse flags from discover command
	revengCmd.Flags().StringVarP(&adapterType, "adapter", "a", "simulator",
		"JTAG adapter type (simulator, cmsisdap, pico, buspirate)")
	revengCmd.Flags().IntVarP(&deviceCount, "count", "c", 1,
		"expected number of devices in chain")
	revengCmd.Flags().StringVarP(&bsdlDir, "bsdl", "b", "testdata",
		"directory containing BSDL files")
	revengCmd.Flags().StringVarP(&adapterSerial, "serial", "s", "",
		"adapter serial number (if multiple adapters)")
	revengCmd.Flags().IntVar(&adapterSpeed, "speed", 1000000,
		"TCK speed in Hz (default 1MHz)")

	// Reveng-specific flags
	revengCmd.Flags().StringVarP(&revengOutputJSON, "output", "o", "",
		"output JSON file path (e.g., netlist.json)")
	revengCmd.Flags().StringVar(&revengOutputKiCad, "output-kicad", "",
		"output KiCad netlist file path (e.g., netlist.net)")
	revengCmd.Flags().IntVar(&revengRepeats, "repeats", 1,
		"number of toggle cycles per pin (default: 1)")
	revengCmd.Flags().BoolVar(&revengSkipJTAG, "skip-jtag", true,
		"skip JTAG control pins (TCK, TMS, TDI, TDO)")
	revengCmd.Flags().BoolVar(&revengSkipPower, "skip-power", true,
		"skip power/ground pins (VCC, GND, etc)")
	revengCmd.Flags().StringSliceVar(&revengOnlyDevices, "only-devices", nil,
		"only scan pins from these devices (comma-separated)")
	revengCmd.Flags().StringVar(&revengOnlyPins, "only-pins", "",
		"only scan pins matching this regex pattern")
	revengCmd.Flags().IntVar(&revengTimeout, "timeout", 0,
		"timeout in seconds (0 = no timeout)")

	revengCmd.MarkFlagRequired("count")
}

func runReveng(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	// Create adapter
	if verbose {
		fmt.Printf("Creating %s adapter...\n", adapterType)
	}

	adapter, err := createAdapter(adapterType, adapterSerial)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Set speed
	if err := adapter.SetSpeed(adapterSpeed); err != nil && err != jtag.ErrNotImplemented {
		return fmt.Errorf("failed to set speed: %w", err)
	}

	// Show adapter info
	if verbose {
		info, err := adapter.Info()
		if err == nil {
			fmt.Printf("\nAdapter: %s\n", info.Name)
			if info.SerialNumber != "" {
				fmt.Printf("Serial: %s\n", info.SerialNumber)
			}
			fmt.Println()
		}
	}

	// Create repository and load BSDL files
	if verbose {
		fmt.Printf("Loading BSDL files from: %s\n", bsdlDir)
	}

	repo := chain.NewMemoryRepository()
	if err := repo.LoadDir(bsdlDir); err != nil {
		return fmt.Errorf("failed to load BSDL files: %w", err)
	}

	if verbose {
		fmt.Println("BSDL files loaded successfully\n")
	}

	// Discover chain
	fmt.Printf("Discovering JTAG chain (expecting %d device(s))...\n", deviceCount)

	chainCtrl := chain.NewController(adapter, repo)
	jtagChain, err := chainCtrl.Discover(deviceCount)
	if err != nil {
		return fmt.Errorf("chain discovery failed: %w", err)
	}

	devices := jtagChain.Devices()
	fmt.Printf("✓ Found %d device(s)\n", len(devices))

	// Display device summary
	for i, device := range devices {
		fmt.Printf("  [%d] %s (IDCODE: 0x%08X)\n", i, device.Name(), device.IDCode)
	}
	fmt.Println()

	// Create BSR controller
	if verbose {
		fmt.Println("Initializing boundary-scan runtime...")
	}

	bsrCtrl, err := bsr.NewController(jtagChain)
	if err != nil {
		return fmt.Errorf("failed to create BSR controller: %w", err)
	}

	// Count total pins
	totalPins := len(bsrCtrl.AllPins())
	fmt.Printf("Total IO pins: %d\n\n", totalPins)

	// Configure reverse engineering
	cfg := reveng.DefaultConfig()
	cfg.RepeatsPerPin = revengRepeats
	cfg.SkipKnownJTAGPins = revengSkipJTAG
	cfg.SkipPowerPins = revengSkipPower
	cfg.OnlyDevices = revengOnlyDevices
	cfg.OnlyPinPattern = revengOnlyPins

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Count candidate pins after filtering
	candidateCount := countCandidates(bsrCtrl, cfg)
	fmt.Printf("Candidate pins (after filtering): %d\n", candidateCount)

	if cfg.SkipKnownJTAGPins {
		fmt.Println("  • Skipping JTAG pins (TCK, TMS, TDI, TDO)")
	}
	if cfg.SkipPowerPins {
		fmt.Println("  • Skipping power pins (VCC, GND)")
	}
	if len(cfg.OnlyDevices) > 0 {
		fmt.Printf("  • Only scanning devices: %s\n", strings.Join(cfg.OnlyDevices, ", "))
	}
	if cfg.OnlyPinPattern != "" {
		fmt.Printf("  • Pin pattern filter: %s\n", cfg.OnlyPinPattern)
	}
	fmt.Println()

	if candidateCount == 0 {
		return fmt.Errorf("no candidate pins to scan after filtering")
	}

	// Estimate time
	estimatedSeconds := candidateCount * 3 / 10 // Assume ~10 pins/second
	fmt.Printf("Estimated time: ~%d seconds\n\n", estimatedSeconds)

	// Create progress channel
	progressCh := make(chan reveng.Progress, 10)
	go displayProgress(progressCh)

	// Set up context with optional timeout
	ctx := context.Background()
	if revengTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(revengTimeout)*time.Second)
		defer cancel()
	}

	// Run reverse engineering
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║ Starting Reverse Engineering                                   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	netlist, err := reveng.DiscoverNetlist(ctx, bsrCtrl, cfg, progressCh)
	close(progressCh)

	if err != nil {
		return fmt.Errorf("reverse engineering failed: %w", err)
	}

	elapsed := time.Since(startTime)

	// Print summary
	fmt.Println()
	printNetlistSummary(netlist, elapsed, candidateCount)

	// Export results
	if revengOutputJSON != "" {
		if err := exportJSON(netlist, revengOutputJSON); err != nil {
			return err
		}
		fmt.Printf("\n✓ JSON netlist saved to: %s\n", revengOutputJSON)
	}

	if revengOutputKiCad != "" {
		if err := exportKiCad(netlist, revengOutputKiCad); err != nil {
			return err
		}
		fmt.Printf("✓ KiCad netlist saved to: %s\n", revengOutputKiCad)
	}

	if revengOutputJSON == "" && revengOutputKiCad == "" {
		fmt.Println("\n⚠ No output files specified. Use --output or --output-kicad to save results.")
	}

	return nil
}

// displayProgress shows real-time progress updates
func displayProgress(progressCh <-chan reveng.Progress) {
	lastPercent := -1

	for p := range progressCh {
		if p.Phase == "init" {
			fmt.Println("Initializing EXTEST mode...")
			continue
		}

		if p.Phase == "finalizing" {
			fmt.Printf("\r%-80s\r", "") // Clear line
			fmt.Println("Finalizing netlist...")
			continue
		}

		// Scanning phase
		percent := 0
		if p.Total > 0 {
			percent = (p.Index * 100) / p.Total
		}

		// Only update on percent change to reduce flicker
		if percent != lastPercent {
			device := p.Driver.DeviceName
			pin := p.Driver.PinName

			// Progress bar
			barWidth := 40
			filled := (percent * barWidth) / 100
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

			fmt.Printf("\r[%s] %3d%% | Pin %d/%d | %s.%s | Nets: %d",
				bar, percent, p.Index, p.Total, device, pin, p.NetsFound)

			lastPercent = percent
		}
	}

	fmt.Println() // New line after progress
}

// printNetlistSummary displays a summary of the discovered netlist
func printNetlistSummary(nl *reveng.Netlist, elapsed time.Duration, scannedPins int) {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║ Reverse Engineering Complete                                   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("Total nets found:      %d\n", nl.NetCount())
	fmt.Printf("Multi-pin nets:        %d\n", nl.MultiPinNetCount())
	fmt.Printf("Isolated pins:         %d\n", nl.NetCount()-nl.MultiPinNetCount())
	fmt.Printf("Pins scanned:          %d\n", scannedPins)
	fmt.Printf("Time elapsed:          %s\n", elapsed.Round(time.Second))
	fmt.Printf("Average speed:         %.1f pins/second\n", float64(scannedPins)/elapsed.Seconds())

	// Show multi-pin nets if there aren't too many
	if nl.MultiPinNetCount() > 0 && nl.MultiPinNetCount() <= 20 {
		fmt.Println("\nDiscovered connections:")
		for _, net := range nl.Nets {
			if len(net.Pins) < 2 {
				continue
			}

			fmt.Printf("\n  Net %d (%d pins):\n", net.ID, len(net.Pins))
			for _, pin := range net.Pins {
				fmt.Printf("    • %s.%s\n", pin.DeviceName, pin.PinName)
			}
		}
	} else if nl.MultiPinNetCount() > 20 {
		fmt.Printf("\n(Use --output to save full netlist - too many nets to display)\n")
	}
}

// exportJSON saves the netlist to a JSON file
func exportJSON(nl *reveng.Netlist, path string) error {
	data, err := nl.ExportJSON()
	if err != nil {
		return fmt.Errorf("failed to export JSON: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

// exportKiCad saves the netlist to a KiCad file
func exportKiCad(nl *reveng.Netlist, path string) error {
	data, err := nl.ExportKiCad()
	if err != nil {
		return fmt.Errorf("failed to export KiCad: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write KiCad file: %w", err)
	}

	return nil
}

// countCandidates estimates how many pins will be scanned
func countCandidates(ctl *bsr.Controller, cfg *reveng.Config) int {
	count := 0
	for _, dev := range ctl.Devices {
		if !cfg.ShouldScanDevice(dev.ChainDev.Name()) {
			continue
		}

		for pinName := range dev.Pins {
			if !cfg.ShouldScanPin(pinName) {
				continue
			}
			if cfg.SkipKnownJTAGPins && isJTAGPin(pinName) {
				continue
			}
			if cfg.SkipPowerPins && isPowerPin(pinName) {
				continue
			}
			count++
		}
	}
	return count
}

// Helper functions (duplicated from reveng package for CLI use)
func isJTAGPin(pinName string) bool {
	upper := strings.ToUpper(pinName)
	return strings.Contains(upper, "TCK") ||
		strings.Contains(upper, "TMS") ||
		strings.Contains(upper, "TDI") ||
		strings.Contains(upper, "TDO") ||
		strings.Contains(upper, "TRST") ||
		strings.Contains(upper, "JTAG")
}

func isPowerPin(pinName string) bool {
	upper := strings.ToUpper(pinName)
	return strings.Contains(upper, "VCC") ||
		strings.Contains(upper, "VDD") ||
		strings.Contains(upper, "VSS") ||
		strings.Contains(upper, "GND") ||
		strings.Contains(upper, "VBAT") ||
		strings.Contains(upper, "VREF")
}
