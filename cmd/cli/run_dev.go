//go:build !runtime && !gateway

package cli

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/cmd/internal"
	"github.com/orkspace/orkestra/pkg/doctor"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/orkspace/orkestra/pkg/utils"
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
			fmt.Println("  This will install any missing dependencies:")
			fmt.Println("    • kubectl")
			fmt.Println("    • helm")
			fmt.Println()
			return fmt.Errorf("cluster not reachable\n")
		}

		paths, _ := cmd.Flags().GetStringSlice("file")
		if len(paths) == 0 {
			paths = defaultFilePaths()
		}
		if len(paths) == 0 {
			paths = kfg.Katalog().Paths
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
		}

		internal.Konduct(kfg, m, ctx)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringSliceP("file", "f", nil, "Path(s) to katalog.yaml (repeatable)")
	runCmd.Flags().Bool("dev", false, "Create a local Kind cluster if none is reachable (development only)")
}

// applyCRFilesIfNeeded applies crFiles declarations via kubectl in order before
// the runtime starts. Only runs outside the cluster (dev mode).
func applyCRFilesIfNeeded(ctx context.Context, katalogPath string, m *merger.Merger) {
	if utils.IsRunningInCluster() {
		return
	}

	katalogDir := filepath.Dir(katalogPath)

	for crdName, entry := range m.All() {
		for _, crFile := range entry.CRFiles {
			path := crFile
			if !filepath.IsAbs(path) && !strings.HasPrefix(path, "http") {
				path = filepath.Join(katalogDir, path)
			}

			out, err := exec.CommandContext(ctx, "kubectl", "apply", "-f", path).CombinedOutput()
			if err != nil {
				logger.Warn().
					Str("crd", crdName).
					Str("path", path).
					Str("output", strings.TrimSpace(string(out))).
					Err(err).
					Msg("crFile pre-apply failed (continuing)")
			} else {
				logger.Info().
					Str("crd", crdName).
					Str("path", path).
					Msg("crFile applied")
			}
		}
	}
}

// waitForCRDsEstablished waits for any CRDs that have crFiles to be Established
// in the cluster before CRs are applied. Only blocks for CRDs that need it.
func waitForCRDsEstablished(ctx context.Context, m *merger.Merger) {
	if utils.IsRunningInCluster() {
		return
	}

	for crdName, entry := range m.All() {
		if len(entry.CRFiles) == 0 {
			continue
		}

		plural := entry.APITypes.Plural
		group := entry.APITypes.Group
		if plural == "" || group == "" {
			continue
		}

		crdFullName := plural + "." + group

		out, err := exec.CommandContext(ctx, "kubectl", "wait",
			"--for=condition=Established",
			"--timeout=30s",
			"crd/"+crdFullName,
		).CombinedOutput()
		if err != nil {
			logger.Warn().
				Str("crd", crdName).
				Str("name", crdFullName).
				Str("output", strings.TrimSpace(string(out))).
				Err(err).
				Msg("CRD not established in time — applying CRs anyway")
		} else {
			logger.Info().
				Str("crd", crdName).
				Str("name", crdFullName).
				Msg("CRD established")
		}
	}
}

// applyCRDFilesIfNeeded applies any crdFile declarations via kubectl before the
// operator starts. Only runs outside the cluster (dev mode). In production,
// CRDs must be pre-applied by the platform operator.
func applyCRDFilesIfNeeded(ctx context.Context, katalogPath string, m *merger.Merger) {
	if utils.IsRunningInCluster() {
		return
	}

	katalogDir := filepath.Dir(katalogPath)

	for crdName, entry := range m.All() {
		if entry.CRDFile == "" {
			continue
		}

		path := entry.CRDFile
		if !filepath.IsAbs(path) && !strings.HasPrefix(path, "http") {
			path = filepath.Join(katalogDir, path)
		}

		out, err := exec.CommandContext(ctx, "kubectl", "apply", "-f", path).CombinedOutput()
		if err != nil {
			logger.Warn().
				Str("crd", crdName).
				Str("path", path).
				Str("output", strings.TrimSpace(string(out))).
				Err(err).
				Msg("crdFile pre-apply failed (continuing)")
		} else {
			logger.Info().
				Str("crd", crdName).
				Str("path", path).
				Msg("crdFile applied")
		}
	}
}
