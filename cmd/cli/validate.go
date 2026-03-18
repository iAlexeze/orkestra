package cli

import (
	"fmt"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/utils"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Orkestra katalog configuration",
}

var validateRegistryCmd = &cobra.Command{
	Use:   "katalog <file>",
	Short: "Validate a katalog file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Missing katalog file")
			return
		}

		var katalog katalog.Katalog
		_, err := katalog.KomposeKatalogFromYaml(args[0])
		if err != nil {
			fmt.Println(err)
			return
		}

		_, err = katalog.ValidateConfig()
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Success: %sKatalog is valid%s\n", utils.ColorGreen, utils.ColorReset)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.AddCommand(validateRegistryCmd)
}
