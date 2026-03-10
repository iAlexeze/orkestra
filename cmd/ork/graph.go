package ork

import (
	"fmt"

	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Visualize Orkestra dependency graphs",
}

var graphDepsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Show the CRD dependency graph",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Dependency graph:")
		// TODO: load registry + print DAG
	},
}

func init() {
	rootCmd.AddCommand(graphCmd)
	graphCmd.AddCommand(graphDepsCmd)
}
