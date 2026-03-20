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
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		var k katalog.Katalog
		if _, err = k.KomposeKatalogFromYaml(m.m); err != nil {
			return err
		}
		if _, err = k.ValidateConfig(); err != nil {
			return err
		}

		fmt.Printf("Success: %sKatalog is valid%s\n", utils.ColorGreen, utils.ColorReset)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringSlice("katalog", nil, "Path(s) or URL(s) to crd-katalog.yaml (repeatable)")
}
