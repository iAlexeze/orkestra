package cli

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/inspect"
	"github.com/orkspace/orkestra/pkg/katalog"
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
		entries, err := k.KomposeKatalogFromYaml(kfg, m.m)
		if err != nil {
			return err
		}

		_, err = k.ValidateConfig(kfg)
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println(inspect.Bold("Validating Katalog..."))
		fmt.Println()

		builtIn := 0
		custom := 0

		// Print each CRD entry with enrichment info
		for _, entry := range entries {
			printCRDValidationLine(entry)
			fmt.Println()

			if entry.IsBuiltIn {
				builtIn++
			} else {
				custom++
			}
		}

		// Summary
		fmt.Println(strings.Repeat("─", 60))
		fmt.Printf("%d CRDs valid (%d built-in, %d custom)\n", len(entries), builtIn, custom)

		// fmt.Println()
		// fmt.Println("Built-in resources are enriched automatically from the Kubernetes API.")
		// fmt.Println("No apiTypes.location or code generation required.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
