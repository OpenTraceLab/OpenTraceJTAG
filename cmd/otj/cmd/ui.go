package cmd

import (
	appui "github.com/OpenTraceLab/OpenTraceJTAG/internal/ui"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the interactive GUI",
	Long: `Launch the JTAG GUI mode with graphical controls for device discovery,
BSDL management, and JTAG operations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := appui.New(nil)
		return app.Run()
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
