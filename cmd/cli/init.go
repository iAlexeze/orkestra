package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Orkestra project",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Initializing Orkestra project...")
		// TODO: scaffold registry.yaml or Go registry
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
