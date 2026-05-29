//go:build !runtime && !gateway

package cli

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/ork"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create Orkestra infrastructure resources",
}

var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create a local kind cluster for Orkestra development or testing",
	Long: `Creates a local kind cluster and switches kubectl to its context.
Downloads kind automatically if not found in PATH.

  ork create cluster
  ork create cluster --name ork-e2e
  ork create cluster --provider kind --name my-cluster`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		provider, _ := cmd.Flags().GetString("provider")

		if provider != "kind" {
			return fmt.Errorf("provider %q not supported — only 'kind' is available", provider)
		}

		fmt.Printf("→ Creating cluster '%s'...\n", name)
		if err := ork.EnsureKindCluster(name); err != nil {
			return err
		}
		fmt.Printf("\nCluster '%s' is ready.\n", name)
		fmt.Printf("kubectl is now pointing to kind-%s.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createClusterCmd)

	createClusterCmd.Flags().String("name", "ork-playground", "Cluster name")
	createClusterCmd.Flags().String("provider", "kind", "Cluster provider (only 'kind' is supported)")

	// Shadow global flags
	createCmd.PersistentFlags().Bool("debug", false, "")
	createCmd.PersistentFlags().String("kubeconfig", "", "")
	createCmd.PersistentFlags().Bool("verbose", false, "")
	createCmd.PersistentFlags().MarkHidden("debug")
	createCmd.PersistentFlags().MarkHidden("kubeconfig")
	createCmd.PersistentFlags().MarkHidden("verbose")
}
