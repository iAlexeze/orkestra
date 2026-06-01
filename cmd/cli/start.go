//go:build !runtime && !gateway

package cli

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/devserver"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start an Orkestra background service",
}

var startDevServerCmd = &cobra.Command{
	Use:   "dev-server",
	Short: "Start the mock dev server for external: examples",
	Long: `Starts a lightweight mock HTTP server on :9999 that handles all endpoints
used by external:, full-stack, and feature-flag examples — no real services needed.

Press Ctrl+C to stop.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")

		if err := devserver.Start(port); err != nil {
			return fmt.Errorf("starting dev server: %w", err)
		}

		fmt.Println("Dev server running. Press Ctrl+C to stop.")
		<-cmd.Context().Done()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.AddCommand(startDevServerCmd)
	startDevServerCmd.Flags().Int("port", devserver.Port, "Port to listen on")
}
