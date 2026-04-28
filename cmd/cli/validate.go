//go:build !runtime

package cli

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/utils"
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
		entries, err := k.KomposeRuntimeKatalog(kfg, m.m)
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println(utils.Bold("Validating Katalog..."))
		fmt.Println()

		_, err = k.ValidateConfig(kfg)
		if err != nil {
			return err
		}

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

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringSliceP("katalog", "k", nil, "Path to katalog.yaml")

	// Shadow global flags so they don't appear under `ork validate`
	validateCmd.Flags().Bool("debug", false, "")
	validateCmd.Flags().String("kubeconfig", "", "")
	validateCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	validateCmd.Flags().MarkHidden("debug")
	validateCmd.Flags().MarkHidden("kubeconfig")
	validateCmd.Flags().MarkHidden("verbose")
}
