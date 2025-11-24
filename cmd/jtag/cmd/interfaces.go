package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
	"github.com/spf13/cobra"
)

var interfacesCmd = &cobra.Command{
	Use:   "interfaces",
	Short: "List available JTAG interfaces",
	Long: `Scan the host for JTAG adapters (CMSIS-DAP, PicoProbe, etc.) and print a summary
of the detected transports. Use this to verify connectivity or select an interface before
launching other commands.`,
	RunE: runInterfaces,
}

func init() {
	rootCmd.AddCommand(interfacesCmd)
}

func runInterfaces(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	infos, err := jtag.DiscoverInterfaces(ctx)
	if err != nil {
		return fmt.Errorf("discover interfaces: %w", err)
	}

	if len(infos) == 0 {
		fmt.Println("No interfaces found.")
		return nil
	}

	fmt.Println("Detected JTAG interfaces:")
	for _, iface := range infos {
		fmt.Printf("  - %s [%s] (VID:PID %04X:%04X)\n", iface.Label(), iface.Kind, iface.VendorID, iface.ProductID)
	}

	return nil
}
