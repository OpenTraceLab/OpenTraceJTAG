package cmd

import (
	"github.com/OpenTraceLab/OpenTraceJTAG/internal/ui/newui"
	"github.com/spf13/cobra"
)

var newUICmd = &cobra.Command{
	Use:   "newui",
	Short: "Launch the experimental GioView-based UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := newui.New(nil)
		return app.Run()
	},
}

func init() {
	rootCmd.AddCommand(newUICmd)
}
