package cmd

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
	"github.com/spf13/cobra"
)

var jtagCmd = &cobra.Command{
	Use:   "jtag",
	Short: "JTAG boundary scan operations",
	Long:  `Commands for JTAG boundary scan operations, BSDL parsing, and device discovery`,
}

// JTAG discover command
var (
	adapterType   string
	deviceCount   int
	bsdlDir       string
	adapterSerial string
	adapterSpeed  int
	simIDCodes    []string
)

var jtagDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover devices in the JTAG chain",
	Long: `Discover and identify all devices in the JTAG chain by reading their IDCODEs
and matching them against BSDL files in the specified directory.

The discover command will:
  1. Reset the JTAG chain
  2. Read IDCODE from all devices
  3. Match IDCODEs to BSDL files
  4. Display device information

Examples:
  # Discover 2 devices using simulator
  otj jtag discover --adapter simulator --count 2 --bsdl testdata

  # Discover with CMSIS-DAP probe
  otj jtag discover --adapter cmsisdap --count 2 --bsdl testdata

  # Verbose output
  otj jtag discover -v --adapter cmsisdap --count 1 --bsdl testdata`,
	RunE: runJTAGDiscover,
}

// JTAG parse command
var (
	showInstructions bool
	showBoundary     bool
	showPins         bool
)

var jtagParseCmd = &cobra.Command{
	Use:   "parse <bsdl-file>",
	Short: "Parse and display information from a BSDL file",
	Long: `Parse a BSDL file and display its contents including entity information,
ports, instructions, boundary scan cells, and pin mappings.

Examples:
  otj jtag parse device.bsd
  otj jtag parse -v --instructions device.bsd
  otj jtag parse --boundary --pins testdata/STM32F303_F334_LQFP64.bsd`,
	Args: cobra.ExactArgs(1),
	RunE: runJTAGParse,
}

func init() {
	rootCmd.AddCommand(jtagCmd)
	jtagCmd.AddCommand(jtagDiscoverCmd)
	jtagCmd.AddCommand(jtagParseCmd)

	// Discover flags
	jtagDiscoverCmd.Flags().StringVarP(&adapterType, "adapter", "a", "simulator",
		"JTAG adapter type (simulator, cmsisdap, pico, buspirate)")
	jtagDiscoverCmd.Flags().IntVarP(&deviceCount, "count", "c", 1,
		"expected number of devices in chain")
	jtagDiscoverCmd.Flags().StringVarP(&bsdlDir, "bsdl", "b", "testdata",
		"directory containing BSDL files")
	jtagDiscoverCmd.Flags().StringVarP(&adapterSerial, "serial", "s", "",
		"adapter serial number (if multiple adapters)")
	jtagDiscoverCmd.Flags().IntVar(&adapterSpeed, "speed", 1000000,
		"TCK speed in Hz (default 1MHz)")
	jtagDiscoverCmd.Flags().StringSliceVar(&simIDCodes, "sim-ids", nil,
		"simulator: IDCODEs to return (hex, e.g., 0x06438041,0x41111043)")
	jtagDiscoverCmd.MarkFlagRequired("count")

	// Parse flags
	jtagParseCmd.Flags().BoolVarP(&showInstructions, "instructions", "i", false,
		"show all instructions")
	jtagParseCmd.Flags().BoolVarP(&showBoundary, "boundary", "b", false,
		"show boundary scan cells")
	jtagParseCmd.Flags().BoolVarP(&showPins, "pins", "p", false,
		"show pin mappings")
}

func runJTAGDiscover(cmd *cobra.Command, args []string) error {
	// Validate sim-ids if using simulator
	if adapterType == "simulator" || adapterType == "sim" {
		if len(simIDCodes) > 0 && len(simIDCodes) != deviceCount {
			return fmt.Errorf("--sim-ids count (%d) must match --count (%d)", len(simIDCodes), deviceCount)
		}
	}

	// Create adapter
	if verbose {
		fmt.Printf("Creating %s adapter...\n", adapterType)
	}

	adapter, err := createJTAGAdapter(adapterType, adapterSerial)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Set speed
	if err := adapter.SetSpeed(adapterSpeed); err != nil && err != jtag.ErrNotImplemented {
		return fmt.Errorf("failed to set speed: %w", err)
	}

	// Show adapter info
	info, err := adapter.Info()
	if err != nil && err != jtag.ErrNotImplemented {
		return fmt.Errorf("failed to get adapter info: %w", err)
	}

	if verbose {
		fmt.Printf("\nAdapter Information:\n")
		fmt.Printf("  Name: %s\n", info.Name)
		fmt.Printf("  Vendor: %s\n", info.Vendor)
		fmt.Printf("  Model: %s\n", info.Model)
		if info.SerialNumber != "" {
			fmt.Printf("  Serial: %s\n", info.SerialNumber)
		}
		if info.Firmware != "" {
			fmt.Printf("  Firmware: %s\n", info.Firmware)
		}
		if info.MaxFrequency > 0 {
			fmt.Printf("  Max Speed: %d Hz\n", info.MaxFrequency)
		}
		fmt.Println()
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
		fmt.Println("BSDL files loaded successfully")
	}

	// Create controller
	ctrl := chain.NewController(adapter, repo)

	// Discover chain
	fmt.Printf("\nDiscovering JTAG chain (expecting %d device(s))...\n", deviceCount)

	jtagChain, err := ctrl.Discover(deviceCount)
	if err != nil {
		return fmt.Errorf("chain discovery failed: %w", err)
	}

	// Display discovered devices
	devices := jtagChain.Devices()
	fmt.Printf("\n╔════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║ JTAG Chain Discovery Results                                   ║\n")
	fmt.Printf("╠════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ Found %d device(s)                                              ║\n", len(devices))
	fmt.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")

	for i, device := range devices {
		fmt.Printf("┌─ Device %d (Position %d) ─────────────────────────────────────┐\n", i+1, device.Position)
		fmt.Printf("│ IDCODE: 0x%08X                                          │\n", device.IDCode)

		if device.Info != nil {
			fmt.Printf("│ Name:   %s\n", device.Name())
			fmt.Printf("│                                                              │\n")
			fmt.Printf("│ Device Information:                                          │\n")
			fmt.Printf("│   IR Length:       %d bits                                  │\n", device.Info.InstructionLength)
			fmt.Printf("│   Boundary Length: %d bits                                  │\n", device.Info.BoundaryLength)

			if device.Info.IDCode != "" {
				// Parse IDCODE to show if it has wildcards
				value, mask, hasWildcards := bsdl.ParseBinaryString(device.Info.IDCode)
				if hasWildcards {
					fmt.Printf("│   IDCODE Pattern:  0x%08X (mask: 0x%08X)              │\n", value, mask)
					fmt.Printf("│                    Contains wildcards                     │\n")
				}
			}

			// Show instructions
			instructions := device.Instructions()
			if len(instructions) > 0 {
				fmt.Printf("│                                                              │\n")
				fmt.Printf("│ Instructions (%d total):                                     │\n", len(instructions))

				// Show first 5 instructions
				limit := len(instructions)
				if limit > 5 {
					limit = 5
				}

				for j := 0; j < limit; j++ {
					instr := instructions[j]
					opcodeVal, _ := bsdl.OpcodeToUint(instr.Opcode)
					fmt.Printf("│   %-12s  %s (0x%X)                              │\n",
						instr.Name, instr.Opcode, opcodeVal)
				}

				if len(instructions) > 5 {
					fmt.Printf("│   ... and %d more                                         │\n", len(instructions)-5)
				}
			}

			// Show TAP config if verbose
			if verbose && device.File != nil && device.File.Entity != nil {
				tapConfig := device.File.Entity.GetTAPConfig()
				if tapConfig != nil && tapConfig.ScanClock != "" {
					fmt.Printf("│                                                              │\n")
					fmt.Printf("│ TAP Configuration:                                           │\n")
					fmt.Printf("│   TDI:  %s                                              │\n", tapConfig.ScanIn)
					fmt.Printf("│   TDO:  %s                                              │\n", tapConfig.ScanOut)
					fmt.Printf("│   TMS:  %s                                              │\n", tapConfig.ScanMode)
					fmt.Printf("│   TCK:  %s                                              │\n", tapConfig.ScanClock)
					if tapConfig.MaxFreq > 0 {
						fmt.Printf("│   Max Frequency: %.0f Hz                                 │\n", tapConfig.MaxFreq)
					}
				}
			}
		} else {
			fmt.Printf("│ Name:   UNKNOWN (no matching BSDL file)                     │\n")
		}

		fmt.Printf("└──────────────────────────────────────────────────────────────┘\n\n")
	}

	// Summary
	fmt.Printf("Chain Summary:\n")
	totalIR := 0
	totalBoundary := 0
	for _, device := range devices {
		if device.Info != nil {
			totalIR += device.Info.InstructionLength
			totalBoundary += device.Info.BoundaryLength
		}
	}
	fmt.Printf("  Total IR Length:       %d bits\n", totalIR)
	fmt.Printf("  Total Boundary Length: %d bits\n", totalBoundary)

	return nil
}

func createJTAGAdapter(adapterType, serial string) (jtag.Adapter, error) {
	switch adapterType {
	case "simulator", "sim":
		if verbose {
			fmt.Println("Using simulator adapter")
		}
		info := jtag.AdapterInfo{
			Name:         "JTAG Simulator",
			Vendor:       "epkcfsm",
			Model:        "Sim-1.0",
			Firmware:     "v0.9.0",
			MinFrequency: 100,
			MaxFrequency: 10000000, // 10 MHz
		}
		sim := jtag.NewSimAdapter(info)

		// Configure simulator with IDCODEs if provided
		if len(simIDCodes) > 0 {
			ids, err := parseIDCodes(simIDCodes)
			if err != nil {
				return nil, fmt.Errorf("invalid --sim-ids: %w", err)
			}

			if verbose {
				fmt.Printf("Configuring simulator with %d IDCODE(s):\n", len(ids))
				for i, id := range ids {
					fmt.Printf("  Device %d: 0x%08X\n", i, id)
				}
			}

			// Configure OnShift hook to return these IDCODEs
			idBytes := encodeIDCodes(ids)
			sim.OnShift = func(region jtag.ShiftRegion, tms, tdi []byte, bits int) ([]byte, error) {
				if region == jtag.ShiftRegionDR && bits == len(ids)*32 {
					return append([]byte(nil), idBytes...), nil
				}
				return make([]byte, (bits+7)/8), nil
			}
		}

		return sim, nil

	case "cmsisdap", "cmsis", "jtagprobe", "dap":
		if verbose {
			fmt.Println("Opening CMSIS-DAP probe...")
		}

		// Create CMSIS-DAP adapter (default Raspberry Pi Pico VID/PID)
		adapter, err := jtag.NewCMSISDAPAdapter(jtag.VendorIDRaspberryPi, jtag.ProductIDCMSISDAP)
		if err != nil {
			return nil, fmt.Errorf("failed to open CMSIS-DAP probe: %w", err)
		}

		if verbose {
			info, _ := adapter.Info()
			fmt.Printf("Connected to: %s %s\n", info.Vendor, info.Model)
			fmt.Printf("  Serial: %s\n", info.SerialNumber)
			fmt.Printf("  Firmware: %s\n", info.Firmware)
			fmt.Printf("  Frequency range: %d - %d Hz\n", info.MinFrequency, info.MaxFrequency)
		}

		return adapter, nil

	case "pico":
		return nil, fmt.Errorf("pico adapter not yet implemented (USB transport pending)")

	case "buspirate", "bp":
		return nil, fmt.Errorf("bus pirate adapter not yet implemented")

	default:
		return nil, fmt.Errorf("unknown adapter type: %s (supported: simulator, cmsisdap, pico, buspirate)", adapterType)
	}
}

func parseIDCodes(codes []string) ([]uint32, error) {
	ids := make([]uint32, len(codes))
	for i, code := range codes {
		var id uint64
		_, err := fmt.Sscanf(code, "0x%x", &id)
		if err != nil {
			// Try without 0x prefix
			_, err = fmt.Sscanf(code, "%x", &id)
			if err != nil {
				return nil, fmt.Errorf("invalid IDCODE format: %s (expected hex like 0x12345678)", code)
			}
		}
		ids[i] = uint32(id)
	}
	return ids, nil
}

func encodeIDCodes(ids []uint32) []byte {
	out := make([]byte, len(ids)*4)
	for i, id := range ids {
		offset := i * 4
		out[offset] = byte(id)
		out[offset+1] = byte(id >> 8)
		out[offset+2] = byte(id >> 16)
		out[offset+3] = byte(id >> 24)
	}
	return out
}

func runJTAGParse(cmd *cobra.Command, args []string) error {
	filename := args[0]

	if verbose {
		fmt.Printf("Parsing BSDL file: %s\n\n", filename)
	}

	// Create parser
	parser, err := bsdl.NewParser()
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}

	// Parse file
	file, err := parser.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	entity := file.Entity

	// Entity information
	fmt.Printf("╔════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║ BSDL File Information                                          ║\n")
	fmt.Printf("╠════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ Entity: %-54s ║\n", entity.Name)
	fmt.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")

	// Generic parameters
	if entity.Generic != nil {
		fmt.Printf("Generic Parameters:\n")
		for _, gen := range entity.Generic.Generics {
			fmt.Printf("  %s : %s", gen.Name, gen.Type)
			if gen.DefaultValue != nil {
				fmt.Printf(" := %s", gen.DefaultValue.GetValue())
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// Ports
	if entity.Port != nil {
		fmt.Printf("Ports: %d total\n", len(entity.Port.Ports))
		if verbose || len(entity.Port.Ports) <= 20 {
			for _, port := range entity.Port.Ports {
				fmt.Printf("  %-20s : %-8s %s\n", port.Name, port.Mode, port.Type.Name)
			}
		} else {
			// Show first 10
			for i := 0; i < 10; i++ {
				port := entity.Port.Ports[i]
				fmt.Printf("  %-20s : %-8s %s\n", port.Name, port.Mode, port.Type.Name)
			}
			fmt.Printf("  ... and %d more ports\n", len(entity.Port.Ports)-10)
		}
		fmt.Println()
	}

	// Use clause
	useClause := entity.GetUseClause()
	if useClause != nil {
		fmt.Printf("Use Clause: %s.%s\n\n", useClause.Package, useClause.Dot)
	}

	// Device info
	info := entity.GetDeviceInfo()
	if info != nil {
		fmt.Printf("Device Information:\n")
		fmt.Printf("  IR Length:       %d bits\n", info.InstructionLength)
		fmt.Printf("  Boundary Length: %d bits\n", info.BoundaryLength)
		if info.IDCode != "" {
			value, mask, hasWildcards := bsdl.ParseBinaryString(info.IDCode)
			fmt.Printf("  IDCODE:          0x%08X", value)
			if hasWildcards {
				fmt.Printf(" (mask: 0x%08X, has wildcards)", mask)
			}
			fmt.Println()
		}
		if info.UserCode != "" {
			value, _, _ := bsdl.ParseBinaryString(info.UserCode)
			fmt.Printf("  USERCODE:        0x%08X\n", value)
		}
		if info.InstructionCapture != "" {
			fmt.Printf("  IR Capture:      %s\n", info.InstructionCapture)
		}
		fmt.Println()
	}

	// Instructions
	instructions := entity.GetInstructionOpcodes()
	if len(instructions) > 0 {
		fmt.Printf("Instructions: %d total\n", len(instructions))

		if showInstructions || verbose {
			// Show all instructions
			for _, instr := range instructions {
				opcodeVal, _ := bsdl.OpcodeToUint(instr.Opcode)
				fmt.Printf("  %-15s %s (0x%X)\n", instr.Name, instr.Opcode, opcodeVal)
			}
		} else {
			// Show first 5
			limit := len(instructions)
			if limit > 5 {
				limit = 5
			}
			for i := 0; i < limit; i++ {
				instr := instructions[i]
				opcodeVal, _ := bsdl.OpcodeToUint(instr.Opcode)
				fmt.Printf("  %-15s %s (0x%X)\n", instr.Name, instr.Opcode, opcodeVal)
			}
			if len(instructions) > 5 {
				fmt.Printf("  ... and %d more (use --instructions to show all)\n", len(instructions)-5)
			}
		}
		fmt.Println()
	}

	// TAP configuration
	tapConfig := entity.GetTAPConfig()
	if tapConfig != nil && tapConfig.ScanClock != "" {
		fmt.Printf("TAP Configuration:\n")
		if tapConfig.ScanIn != "" {
			fmt.Printf("  TDI (Scan In):    %s\n", tapConfig.ScanIn)
		}
		if tapConfig.ScanOut != "" {
			fmt.Printf("  TDO (Scan Out):   %s\n", tapConfig.ScanOut)
		}
		if tapConfig.ScanMode != "" {
			fmt.Printf("  TMS (Scan Mode):  %s\n", tapConfig.ScanMode)
		}
		if tapConfig.ScanReset != "" {
			fmt.Printf("  TRST (Scan Reset):%s\n", tapConfig.ScanReset)
		}
		if tapConfig.ScanClock != "" {
			fmt.Printf("  TCK (Scan Clock): %s\n", tapConfig.ScanClock)
		}
		if tapConfig.MaxFreq > 0 {
			fmt.Printf("  Max Frequency:    %.0f Hz (%.2f MHz)\n",
				tapConfig.MaxFreq, tapConfig.MaxFreq/1e6)
		}
		if tapConfig.Edge != "" {
			fmt.Printf("  Clock Edge:       %s\n", tapConfig.Edge)
		}
		fmt.Println()
	}

	// Boundary cells
	if showBoundary {
		cells, err := entity.GetBoundaryCells()
		if err != nil {
			fmt.Printf("Boundary Register: Error parsing - %v\n\n", err)
		} else {
			fmt.Printf("Boundary Register: %d cells\n", len(cells))

			if verbose || len(cells) <= 20 {
				// Show all cells
				for _, cell := range cells {
					fmt.Printf("  %3d: %-6s %-20s %-10s safe=%s",
						cell.Number, cell.CellType, cell.Port, cell.Function, cell.Safe)
					if cell.Control >= 0 {
						fmt.Printf(" ctrl=%d", cell.Control)
					}
					if cell.Disable >= 0 {
						fmt.Printf(" dis=%d", cell.Disable)
					}
					if cell.Result != "" {
						fmt.Printf(" res=%s", cell.Result)
					}
					fmt.Println()
				}
			} else {
				// Show first 10
				for i := 0; i < 10; i++ {
					cell := cells[i]
					fmt.Printf("  %3d: %-6s %-20s %-10s safe=%s",
						cell.Number, cell.CellType, cell.Port, cell.Function, cell.Safe)
					if cell.Control >= 0 {
						fmt.Printf(" ctrl=%d", cell.Control)
					}
					fmt.Println()
				}
				fmt.Printf("  ... and %d more cells (use -v to show all)\n", len(cells)-10)
			}
			fmt.Println()
		}
	}

	// Pin mappings
	if showPins {
		pinMap := entity.GetPinMap()
		if len(pinMap) > 0 {
			fmt.Printf("Pin Mappings: %d signals\n", len(pinMap))

			// Create sorted list of signals for consistent output
			signals := make([]string, 0, len(pinMap))
			for signal := range pinMap {
				signals = append(signals, signal)
			}

			if verbose || len(signals) <= 30 {
				for _, signal := range signals {
					fmt.Printf("  %-20s -> Pin %s\n", signal, pinMap[signal])
				}
			} else {
				for i := 0; i < 30; i++ {
					signal := signals[i]
					fmt.Printf("  %-20s -> Pin %s\n", signal, pinMap[signal])
				}
				fmt.Printf("  ... and %d more (use -v to show all)\n", len(signals)-30)
			}
			fmt.Println()
		}
	}

	fmt.Println("Parsing completed successfully!")
	return nil
}
