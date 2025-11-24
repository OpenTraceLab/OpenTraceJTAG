package cmd

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/spf13/cobra"
)

var (
	pinDeviceName string
	pinName       string
	pinHigh       bool
	pinLow        bool
)

var pinCmd = &cobra.Command{
	Use:   "pin",
	Short: "Control pins via boundary scan",
	Long: `Control individual pins on devices in the JTAG chain using boundary scan.
This command requires a discovered chain (use discover first) or can work with
the simulator for testing.

Examples:
  # Drive pin PA0 high on STM32F303
  jtag pin --device STM32F303 --pin PA0 --high

  # Drive pin PA1 low on STM32F303
  jtag pin --device STM32F303 --pin PA1 --low

  # With simulator (single device)
  jtag pin --count 1 --sim-ids 0x06438041 --device STM32F303_F334_LQFP64 --pin PA0 --high`,
	RunE: runPin,
}

func init() {
	rootCmd.AddCommand(pinCmd)

	// Pin-specific flags
	pinCmd.Flags().StringVarP(&pinDeviceName, "device", "d", "",
		"device name (entity name from BSDL)")
	pinCmd.Flags().StringVarP(&pinName, "pin", "p", "",
		"pin name (e.g., PA0, PB5)")
	pinCmd.Flags().BoolVar(&pinHigh, "high", false,
		"drive pin high (true/1)")
	pinCmd.Flags().BoolVar(&pinLow, "low", false,
		"drive pin low (false/0)")

	// Chain setup flags (for simulator mode)
	pinCmd.Flags().IntVarP(&deviceCount, "count", "c", 1,
		"number of devices in chain (for simulator)")
	pinCmd.Flags().StringSliceVar(&simIDCodes, "sim-ids", nil,
		"simulator: IDCODEs to return")
	pinCmd.Flags().StringVarP(&bsdlDir, "bsdl", "b", "testdata",
		"directory containing BSDL files")
	pinCmd.Flags().StringVarP(&adapterType, "adapter", "a", "simulator",
		"JTAG adapter type")

	// Mark required
	pinCmd.MarkFlagRequired("device")
	pinCmd.MarkFlagRequired("pin")
}

func runPin(cmd *cobra.Command, args []string) error {
	// Validate flags
	if !pinHigh && !pinLow {
		return fmt.Errorf("must specify either --high or --low")
	}
	if pinHigh && pinLow {
		return fmt.Errorf("cannot specify both --high and --low")
	}

	// Validate simulator config
	if adapterType == "simulator" || adapterType == "sim" {
		if len(simIDCodes) > 0 && len(simIDCodes) != deviceCount {
			return fmt.Errorf("--sim-ids count (%d) must match --count (%d)", len(simIDCodes), deviceCount)
		}
	}

	// Create adapter
	if verbose {
		fmt.Printf("Creating %s adapter...\n", adapterType)
	}

	adapter, err := createAdapter(adapterType, adapterSerial)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Load BSDL files
	if verbose {
		fmt.Printf("Loading BSDL files from: %s\n", bsdlDir)
	}

	repo := chain.NewMemoryRepository()
	if err := repo.LoadDir(bsdlDir); err != nil {
		return fmt.Errorf("failed to load BSDL files: %w", err)
	}

	// Create controller and discover chain
	ctrl := chain.NewController(adapter, repo)

	if verbose {
		fmt.Printf("Discovering chain with %d device(s)...\n", deviceCount)
	}

	jtagChain, err := ctrl.Discover(deviceCount)
	if err != nil {
		return fmt.Errorf("chain discovery failed: %w", err)
	}

	devices := jtagChain.Devices()
	if verbose {
		fmt.Printf("Found %d device(s) in chain\n", len(devices))
	}

	// Find the target device
	var targetDevice *chain.Device
	for _, dev := range devices {
		if dev.Name() == pinDeviceName {
			targetDevice = dev
			break
		}
	}

	if targetDevice == nil {
		// List available devices
		fmt.Printf("Device '%s' not found in chain.\n\nAvailable devices:\n", pinDeviceName)
		for i, dev := range devices {
			fmt.Printf("  %d. %s (IDCODE: 0x%08X)\n", i+1, dev.Name(), dev.IDCode)
		}
		return fmt.Errorf("device not found: %s", pinDeviceName)
	}

	if verbose {
		fmt.Printf("Target device: %s (position %d)\n", targetDevice.Name(), targetDevice.Position)
	}

	// Toggle the pin
	level := pinHigh // true if --high, false if --low
	action := "low"
	if level {
		action = "high"
	}

	fmt.Printf("Setting pin %s on device %s to %s...\n", pinName, pinDeviceName, action)

	err = jtagChain.TogglePin(pinDeviceName, pinName, level)
	if err != nil {
		return fmt.Errorf("failed to toggle pin: %w", err)
	}

	fmt.Printf("✓ Pin %s set to %s successfully\n", pinName, action)

	if verbose {
		fmt.Println("\nBoundary scan operation completed.")
		fmt.Printf("The output cell for pin %s has been programmed to drive %s.\n", pinName, action)
		if targetDevice.Info != nil {
			fmt.Printf("Total boundary cells: %d\n", targetDevice.Info.BoundaryLength)
		}
	}

	return nil
}
