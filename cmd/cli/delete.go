//go:build !runtime && !gateway

package cli

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/tools/cluster"
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
		if err := cluster.DeleteKindCluster(name); err != nil {
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
	shadowGlobalCommandFlags(deleteClusterCmd, "file")
}
