//go:build !runtime && !gateway

package cli

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/ork"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete Orkestra infrastructure resources",
}

var deleteClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Delete a local kind cluster created by ork create cluster",
	Long: `Deletes a local kind cluster by name.

  ork delete cluster
  ork delete cluster --name ork-e2e`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		fmt.Printf("→ Deleting cluster '%s'...\n", name)
		if err := ork.DeleteKindCluster(name); err != nil {
			return err
		}
		fmt.Printf("Cluster '%s' deleted.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteClusterCmd)

	deleteClusterCmd.Flags().String("name", "ork-playground", "Cluster name")

	// Shadow global flags
	deleteCmd.PersistentFlags().StringSlice("file", nil, "")
	deleteCmd.PersistentFlags().Bool("debug", false, "")
	deleteCmd.PersistentFlags().String("kubeconfig", "", "")
	deleteCmd.PersistentFlags().Bool("verbose", false, "")

	deleteCmd.PersistentFlags().MarkHidden("file")
	deleteCmd.PersistentFlags().MarkHidden("debug")
	deleteCmd.PersistentFlags().MarkHidden("kubeconfig")
	deleteCmd.PersistentFlags().MarkHidden("verbose")
}
