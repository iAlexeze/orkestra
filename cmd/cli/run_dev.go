//go:build !runtime

package cli

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/doctor"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
)

func init() {
	// Add --dev flag for development builds
	runCmd.Flags().Bool("dev", false, "Create a local Kind cluster if none is reachable (development only)")

	// Replace the command's RunE with a wrapper that handles cluster creation
	originalRunE := runCmd.RunE
	runCmd.RunE = func(cmd *cobra.Command, args []string) error {
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
				// After cluster creation, the kubeconfig is pointed to the new cluster.
				// The original run will now be able to connect.
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

		// Apply declared crdFile paths before handing off to the runtime.
		// In-cluster check is inside — this is a no-op in production.
		paths, _ := cmd.Flags().GetStringSlice("file")
		if len(paths) > 0 {
			m := merger.New(paths...)
			if err := m.Merge(); err == nil {
				applyCRDFilesIfNeeded(cmd.Context(), paths[0], m)
			}
		}

		return originalRunE(cmd, args)
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
			// Best effort — CRD may already exist at the correct version.
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
