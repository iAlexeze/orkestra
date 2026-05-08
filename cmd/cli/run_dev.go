//go:build !runtime

package cli

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/doctor"
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

		return originalRunE(cmd, args)
	}
}
