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
	Short: "Delete one or more local kind clusters created by ork create cluster",
	Long: `Deletes one or more local kind cluster by name.
	Separated by commas.


  ork delete cluster
  ork delete cluster --name ork-e2e
  ork delete cluster --name ork-1,ork-2,ork-3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		nameList := splitCommaSeparated(name)
		for _, n := range nameList {
			fmt.Printf("→ Deleting cluster '%s'...\n", n)
			if err := cluster.DeleteKindCluster(n); err != nil {
				return err
			}
			fmt.Printf("Cluster '%s' deleted.\n", n)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteClusterCmd)

	deleteClusterCmd.Flags().StringP("name", "n", "ork-playground", "Cluster name. Accepts multiple names separated by commas.")

	// Shadow global flags
	shadowGlobalCommandFlags(deleteClusterCmd, "file")
}
