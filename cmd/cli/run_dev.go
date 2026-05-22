//go:build !runtime && !gateway

package cli

import (
	"fmt"

	"github.com/orkspace/orkestra/cmd/internal"
	"github.com/orkspace/orkestra/pkg/doctor"
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

		if dev {
			if err := doctor.EnsureDependencies(); err != nil {
				return fmt.Errorf("installing dependencies: %w", err)
			}

			if !doctor.ClusterReachable() {
				fmt.Println("\n  Cannot reach Kubernetes cluster.")
				fmt.Printf("  Creating local Kind cluster '%s'...\n", doctor.KindClusterName)
				if err := doctor.EnsureKindCluster(doctor.KindClusterName); err != nil {
					return fmt.Errorf("setting up kind cluster: %w", err)
				}
			}
		} else if !doctor.ClusterReachable() {
			fmt.Println("\n  Cannot reach Kubernetes cluster.")
			fmt.Println("  Check your kubeconfig, or run with --dev to deploy to a local kind cluster.")
			// Confirm missing dependencies
			var (
				missing []string
				helm    = doctor.HelmAvailable()
				kubectl = doctor.KubectlAvailable()
			)
			if !helm {
				missing = append(missing, "kubectl")
			}
			if !kubectl {
				missing = append(missing, "helm")
			}
			if len(missing) > 0 {
				text := "these missing dependencies"
				if len(missing) == 1 {
					text = "this missing dependency"
				}
				fmt.Printf("  This will install %s:\n", text)
				for _, m := range missing {
					fmt.Printf("    • %s\n", m)
				}
			}
			fmt.Println()
			return fmt.Errorf("cluster not reachable\n")
		}

		paths, _ := cmd.Flags().GetStringSlice("file")
		if len(paths) == 0 {
			paths = defaultFilePaths()
		}
		if len(paths) == 0 {
			paths = kfg.Katalog().Paths()
		}
		if len(paths) == 0 {
			return fmt.Errorf(errNoKatalog)
		}

		m := merger.New(paths...)
		if err := m.Merge(); err != nil {
			return fmt.Errorf("merging katalogs: %w", err)
		}

		logger.Debug().
			Strs("katalogs", paths).
			Int("total", m.Count()).
			Int("enabled", m.EnabledCount()).
			Msg("katalogs merged")

		// Apply declared crdFile and crFiles paths before handing off to the runtime.
		if len(paths) > 0 {
			applyCRDFilesIfNeeded(cmd.Context(), paths[0], m)
			waitForCRDsEstablished(cmd.Context(), m)
			applyCRFilesIfNeeded(cmd.Context(), paths[0], m)
			applySetupIfNeeded(cmd.Context(), paths[0], m)
		}

		internal.KonductRuntime(kfg, m, ctx)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringSliceP("file", "f", nil, "Path(s) to katalog.yaml (repeatable)")
	runCmd.Flags().Bool("dev", false, "Create a local Kind cluster if none is reachable (development only)")
}
