package cmd

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
	"github.com/spf13/cobra"
)

var (
	showInstructions bool
	showBoundary     bool
	showPins         bool
)

var parseCmd = &cobra.Command{
	Use:   "parse <bsdl-file>",
	Short: "Parse and display information from a BSDL file",
	Long: `Parse a BSDL file and display its contents including entity information,
ports, instructions, boundary scan cells, and pin mappings.

Examples:
  jtag parse device.bsd
  jtag parse -v --instructions device.bsd
  jtag parse --boundary --pins testdata/STM32F303_F334_LQFP64.bsd`,
	Args: cobra.ExactArgs(1),
	RunE: runParse,
}

func init() {
	rootCmd.AddCommand(parseCmd)

	parseCmd.Flags().BoolVarP(&showInstructions, "instructions", "i", false,
		"show all instructions")
	parseCmd.Flags().BoolVarP(&showBoundary, "boundary", "b", false,
		"show boundary scan cells")
	parseCmd.Flags().BoolVarP(&showPins, "pins", "p", false,
		"show pin mappings")
}

func runParse(cmd *cobra.Command, args []string) error {
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
