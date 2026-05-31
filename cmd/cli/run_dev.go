//go:build !runtime && !gateway

package cli

import (
	"fmt"

	"github.com/orkspace/orkestra/cmd/internal"
	"github.com/orkspace/orkestra/pkg/devserver"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/spf13/cobra"
)

// runCmd is the full dev build version of ork run.
// The production version lives in run.go (//go:build runtime).
// This file owns the command registration for dev builds and layers
// the --dev cluster-setup behaviour on top of the core production logic.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Orkestra Runtime",
	RunE: func(cmd *cobra.Command, args []string) error {
		dev, _ := cmd.Flags().GetBool("dev")
		devServer, _ := cmd.Flags().GetBool("dev-server")
		devServerPort, _ := cmd.Flags().GetInt("dev-server-port")

		// Handle dev mode cluster creation
		if err := ensureClusterReady(dev); err != nil {
			return err
		}

		// Start the mock dev server before the runtime if requested.
		if devServer {
			if err := devserver.Start(devServerPort); err != nil {
				return fmt.Errorf("starting dev server: %w", err)
			}
		}

		// Resolve katalog paths
		paths, _ := cmd.Flags().GetStringSlice("file")
		paths, err := resolveKatalogPaths(paths)
		if err != nil {
			return err
		}

		// Merge katalogs
		m := merger.New(paths...)
		if err := m.Merge(); err != nil {
			return fmt.Errorf("merging katalogs: %w", err)
		}

		logger.Debug().
			Strs("katalogs", paths).
			Int("total", m.Count()).
			Int("enabled", m.EnabledCount()).
			Msg("katalogs merged")

		// Apply declared crdFile, crFiles and setup paths before handing off to the runtime.
		if len(paths) > 0 {
			applyPreRuntimeResources(cmd.Context(), paths[0], m)
		}

		// Run the runtime
		internal.KonductRuntime(kfg, m, ctx)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().Bool("dev", false, "Create a local Kind cluster if none is reachable (development only)")
	runCmd.Flags().Bool("dev-server", false, "Start the mock dev server for external: examples (no real services needed)")
	runCmd.Flags().Int("dev-server-port", devserver.Port, "Port for the mock dev server")
}
