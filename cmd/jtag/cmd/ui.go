package cmd

import (
	"fmt"

	"github.com/OpenTraceLab/OpenTraceJTAG/internal/ui"
	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the JTAG graphical user interface",
	Long: `Launch the JTAG GUI mode with graphical controls for device discovery,
BSDL management, and JTAG operations.

The UI provides an interactive interface for:
  - Device discovery and visualization
  - BSDL file management
  - Real-time JTAG chain monitoring
  - Adapter configuration

Examples:
  # Launch the UI
  jtag ui

  # Launch with verbose logging
  jtag ui -v`,
	RunE: runUI,
}

func init() {
	rootCmd.AddCommand(uiCmd)
}

func runUI(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Println("Launching JTAG UI...")
	}

	state := ui.NewState()
	state.SetAppVersion(rootCmd.Version)
	state.SetStatus("Initializing UI")
	state.AppendLog("UI starting...")

	adapter, err := jtag.NewPicoProbeAdapter("")
	if err != nil {
		state.AppendLog(fmt.Sprintf("Adapter initialization failed: %v", err))
		state.SetStatus("No adapter detected")
	} else {
		state.AppendLog("Adapter connected, fetching info...")
		var info *jtag.AdapterInfo
		if adapterInfo, infoErr := adapter.Info(); infoErr != nil {
			state.AppendLog(fmt.Sprintf("Adapter info error: %v", infoErr))
		} else {
			info = &adapterInfo
		}
		state.SetAdapter(adapter, info)
		state.SetStatus("Adapter ready")
	}

	return ui.Run(state)
}
