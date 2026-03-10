package ork

import (
	"fmt"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Orkestra configuration",
}

var validateRegistryCmd = &cobra.Command{
	Use:   "registry <file>",
	Short: "Validate a CRD registry file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Missing registry file")
			return
		}
		fmt.Printf("Validating registry: %s\n", args[0])
		// TODO: validate YAML registry
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.AddCommand(validateRegistryCmd)
}
