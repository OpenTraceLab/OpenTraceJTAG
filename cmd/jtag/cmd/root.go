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
	Use:   "jtag",
	Short: "JTAG BSDL Parser and Chain Controller",
	Long: `A complete JTAG boundary scan tool for parsing BSDL files,
discovering devices in JTAG chains, and controlling pins via boundary scan.

Examples:
  jtag discover --adapter simulator --count 2        # Discover chain with simulator
  jtag parse device.bsd                              # Parse a BSDL file
  jtag info testdata/                                # Show info about all BSDL files`,
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
