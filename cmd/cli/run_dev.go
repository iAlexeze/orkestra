//go:build !runtime && !gateway

package cli

import (
	"fmt"

	"github.com/orkspace/orkestra/cmd/internal"
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

		// Dev mode cluster creation
		if dev {
			if err := ensureClusterReady(dev); err != nil {
				return err
			}
		}

		// Resolve katalog paths
		paths, _ := cmd.Flags().GetStringSlice("file")
		paths, err := resolveKatalogPaths(paths, kfg.Katalog().Paths())
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
}
