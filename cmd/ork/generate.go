package ork

import (
	"fmt"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Orkestra komponents",
}

var generateCRDCmd = &cobra.Command{
	Use:   "crd <name>",
	Short: "Generate a new CRD scaffold",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Missing CRD name")
			return
		}
		fmt.Printf("Generating CRD: %s\n", args[0])
		// TODO: scaffold API types
	},
}

var generateReconcilerCmd = &cobra.Command{
	Use:   "reconciler <name>",
	Short: "Generate a new reconciler scaffold",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Missing reconciler name")
			return
		}
		fmt.Printf("Generating reconciler: %s\n", args[0])
		// TODO: scaffold reconciler
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generateCRDCmd)
	generateCmd.AddCommand(generateReconcilerCmd)
}
