package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "otj",
	Short: "OpenTraceJTAG - Unified JTAG, PCB, and Schematic tools",
	Long: `OpenTraceJTAG (otj) provides a unified interface for working with:
  - JTAG boundary scan operations and BSDL parsing
  - KiCad PCB file analysis and visualization
  - KiCad schematic file analysis

Examples:
  otj ui                              # Launch interactive GUI
  otj jtag discover --adapter sim     # Discover JTAG chain
  otj pcb view board.kicad_pcb        # View PCB file
  otj sch info schematic.kicad_sch    # Show schematic info`,
	Version: "0.9.0",
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Fix Fyne locale parsing error when LANG=C
	// This needs to run before any Fyne imports are initialized
	if lang := os.Getenv("LANG"); lang == "" || lang == "C" {
		os.Setenv("LANG", "en_US.UTF-8")
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
