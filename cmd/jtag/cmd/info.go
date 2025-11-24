package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/chain"
	"github.com/spf13/cobra"
)

var (
	outputJSON bool
)

// ChainInfo represents structured chain information for visualization
type ChainInfo struct {
	DeviceCount int          `json:"device_count"`
	TotalIR     int          `json:"total_ir_length"`
	TotalBound  int          `json:"total_boundary_length"`
	Devices     []DeviceInfo `json:"devices"`
}

// DeviceInfo represents structured device information
type DeviceInfo struct {
	Position       int              `json:"position"`
	IDCode         string           `json:"idcode"`
	Name           string           `json:"name"`
	Manufacturer   string           `json:"manufacturer,omitempty"`
	IRLength       int              `json:"ir_length"`
	BoundaryLength int              `json:"boundary_length"`
	Package        string           `json:"package,omitempty"`
	Instructions   []InstructionInfo `json:"instructions"`
	Pins           []PinInfo        `json:"pins,omitempty"`
	TAPConfig      *TAPInfo         `json:"tap_config,omitempty"`
}

// InstructionInfo represents an instruction
type InstructionInfo struct {
	Name   string `json:"name"`
	Opcode string `json:"opcode"`
	Value  uint   `json:"value"`
}

// PinInfo represents pin mapping
type PinInfo struct {
	Signal     string `json:"signal"`
	PinNumber  string `json:"pin_number"`
	BoundaryCells []int `json:"boundary_cells,omitempty"`
}

// TAPInfo represents TAP configuration
type TAPInfo struct {
	TDI         string  `json:"tdi"`
	TDO         string  `json:"tdo"`
	TMS         string  `json:"tms"`
	TCK         string  `json:"tck"`
	TRST        string  `json:"trst,omitempty"`
	MaxFreq     float64 `json:"max_frequency"`
	ClockEdge   string  `json:"clock_edge,omitempty"`
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Query chain and device information",
	Long: `Query the JTAG chain and return structured information about devices,
suitable for visualization tools and programmatic access.

Supports JSON output format for integration with other tools.

Examples:
  # Get chain info as JSON
  jtag info --json --count 2 --sim-ids 0x06438041,0x41111043

  # Get chain info with human-readable output
  jtag info --count 1 --sim-ids 0x06438041

  # Verbose mode with full details
  jtag info -v --json --count 2 --sim-ids 0x06438041,0x41111043`,
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)

	// Output format
	infoCmd.Flags().BoolVar(&outputJSON, "json", false,
		"output as JSON (for programmatic access)")

	// Chain setup flags
	infoCmd.Flags().IntVarP(&deviceCount, "count", "c", 1,
		"number of devices in chain")
	infoCmd.Flags().StringSliceVar(&simIDCodes, "sim-ids", nil,
		"simulator: IDCODEs to return")
	infoCmd.Flags().StringVarP(&bsdlDir, "bsdl", "b", "testdata",
		"directory containing BSDL files")
	infoCmd.Flags().StringVarP(&adapterType, "adapter", "a", "simulator",
		"JTAG adapter type")

	infoCmd.MarkFlagRequired("count")
}

func runInfo(cmd *cobra.Command, args []string) error {
	// Validate simulator config
	if adapterType == "simulator" || adapterType == "sim" {
		if len(simIDCodes) > 0 && len(simIDCodes) != deviceCount {
			return fmt.Errorf("--sim-ids count (%d) must match --count (%d)", len(simIDCodes), deviceCount)
		}
	}

	// Create adapter
	adapter, err := createAdapter(adapterType, adapterSerial)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Load BSDL files
	repo := chain.NewMemoryRepository()
	if err := repo.LoadDir(bsdlDir); err != nil {
		return fmt.Errorf("failed to load BSDL files: %w", err)
	}

	// Discover chain
	ctrl := chain.NewController(adapter, repo)
	jtagChain, err := ctrl.Discover(deviceCount)
	if err != nil {
		return fmt.Errorf("chain discovery failed: %w", err)
	}

	// Build chain info
	chainInfo := buildChainInfo(jtagChain)

	// Output
	if outputJSON {
		return outputJSONFormat(chainInfo)
	}

	return outputHumanFormat(chainInfo)
}

func buildChainInfo(jtagChain *chain.Chain) *ChainInfo {
	devices := jtagChain.Devices()
	info := &ChainInfo{
		DeviceCount: len(devices),
		Devices:     make([]DeviceInfo, len(devices)),
	}

	for i, dev := range devices {
		devInfo := DeviceInfo{
			Position: dev.Position,
			IDCode:   fmt.Sprintf("0x%08X", dev.IDCode),
			Name:     dev.Name(),
		}

		if dev.Info != nil {
			devInfo.IRLength = dev.Info.InstructionLength
			devInfo.BoundaryLength = dev.Info.BoundaryLength

			info.TotalIR += dev.Info.InstructionLength
			info.TotalBound += dev.Info.BoundaryLength
		}

		// Extract manufacturer from name
		devInfo.Manufacturer = extractManufacturer(dev.Name())

		// Extract package from name
		devInfo.Package = extractPackage(dev.Name())

		// Instructions
		instructions := dev.Instructions()
		devInfo.Instructions = make([]InstructionInfo, len(instructions))
		for j, instr := range instructions {
			val, _ := bsdl.OpcodeToUint(instr.Opcode)
			devInfo.Instructions[j] = InstructionInfo{
				Name:   instr.Name,
				Opcode: instr.Opcode,
				Value:  val,
			}
		}

		// Pin mappings
		if dev.File != nil && dev.File.Entity != nil {
			pinMap := dev.File.Entity.GetPinMap()
			if len(pinMap) > 0 {
				devInfo.Pins = make([]PinInfo, 0, len(pinMap))
				for signal, pin := range pinMap {
					devInfo.Pins = append(devInfo.Pins, PinInfo{
						Signal:    signal,
						PinNumber: pin,
					})
				}
			}

			// TAP config
			tapConfig := dev.File.Entity.GetTAPConfig()
			if tapConfig != nil {
				devInfo.TAPConfig = &TAPInfo{
					TDI:       tapConfig.ScanIn,
					TDO:       tapConfig.ScanOut,
					TMS:       tapConfig.ScanMode,
					TCK:       tapConfig.ScanClock,
					TRST:      tapConfig.ScanReset,
					MaxFreq:   tapConfig.MaxFreq,
					ClockEdge: tapConfig.Edge,
				}
			}
		}

		info.Devices[i] = devInfo
	}

	return info
}

func outputJSONFormat(info *ChainInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}

func outputHumanFormat(info *ChainInfo) error {
	fmt.Printf("╔════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║ JTAG Chain Information                                         ║\n")
	fmt.Printf("╠════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ Devices: %-6d                                                ║\n", info.DeviceCount)
	fmt.Printf("║ Total IR Length: %-6d bits                                   ║\n", info.TotalIR)
	fmt.Printf("║ Total Boundary:  %-6d bits                                   ║\n", info.TotalBound)
	fmt.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")

	for i, dev := range info.Devices {
		fmt.Printf("Device %d: %s\n", i+1, dev.Name)
		fmt.Printf("  Position:     %d\n", dev.Position)
		fmt.Printf("  IDCODE:       %s\n", dev.IDCode)
		if dev.Manufacturer != "" {
			fmt.Printf("  Manufacturer: %s\n", dev.Manufacturer)
		}
		if dev.Package != "" {
			fmt.Printf("  Package:      %s\n", dev.Package)
		}
		fmt.Printf("  IR Length:    %d bits\n", dev.IRLength)
		fmt.Printf("  Boundary:     %d bits\n", dev.BoundaryLength)
		fmt.Printf("  Instructions: %d total\n", len(dev.Instructions))

		if verbose && len(dev.Instructions) > 0 {
			fmt.Printf("    Instructions:\n")
			for _, instr := range dev.Instructions {
				fmt.Printf("      - %-20s %s (0x%X)\n", instr.Name, instr.Opcode, instr.Value)
			}
		}

		if verbose && len(dev.Pins) > 0 {
			fmt.Printf("    Pins: %d total\n", len(dev.Pins))
		}

		if verbose && dev.TAPConfig != nil {
			fmt.Printf("    TAP Configuration:\n")
			fmt.Printf("      TDI: %s, TDO: %s, TMS: %s, TCK: %s\n",
				dev.TAPConfig.TDI, dev.TAPConfig.TDO,
				dev.TAPConfig.TMS, dev.TAPConfig.TCK)
			if dev.TAPConfig.MaxFreq > 0 {
				fmt.Printf("      Max Frequency: %.0f Hz\n", dev.TAPConfig.MaxFreq)
			}
		}

		fmt.Println()
	}

	return nil
}

// extractManufacturer tries to determine manufacturer from device name
func extractManufacturer(name string) string {
	nameUpper := strings.ToUpper(name)

	if strings.Contains(nameUpper, "STM32") || strings.Contains(nameUpper, "STM") {
		return "STMicroelectronics"
	}
	if strings.Contains(nameUpper, "LFE") || strings.Contains(nameUpper, "LATTICE") {
		return "Lattice Semiconductor"
	}
	if strings.Contains(nameUpper, "ADSP") || strings.Contains(nameUpper, "ANALOG") {
		return "Analog Devices"
	}
	if strings.Contains(nameUpper, "XC") || strings.Contains(nameUpper, "XILINX") {
		return "Xilinx/AMD"
	}
	if strings.Contains(nameUpper, "EP") || strings.Contains(nameUpper, "CYCLONE") || strings.Contains(nameUpper, "ALTERA") {
		return "Intel/Altera"
	}

	return ""
}

// extractPackage tries to extract package type from device name
func extractPackage(name string) string {
	nameUpper := strings.ToUpper(name)

	if strings.Contains(nameUpper, "LQFP") {
		// Extract pin count
		for _, suffix := range []string{"64", "48", "100", "144", "176"} {
			if strings.Contains(nameUpper, "LQFP"+suffix) {
				return "LQFP-" + suffix
			}
		}
		return "LQFP"
	}
	if strings.Contains(nameUpper, "BGA") || strings.Contains(nameUpper, "CABGA") {
		// Extract ball count
		for _, suffix := range []string{"256", "381", "484", "672", "756"} {
			if strings.Contains(nameUpper, suffix) {
				return "BGA-" + suffix
			}
		}
		return "BGA"
	}
	if strings.Contains(nameUpper, "QFP") {
		return "QFP"
	}
	if strings.Contains(nameUpper, "TQFP") {
		return "TQFP"
	}

	return ""
}
