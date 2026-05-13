//go:build !runtime

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Orkestra katalog configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		katalogPaths, _ := cmd.Flags().GetStringSlice("file")
		expanded := parseKatalogPaths(katalogPaths)

		// If any path is a Motif, route to Motif validation
		for _, path := range expanded {
			kind, err := detectKindFromFile(path)
			if err != nil {
				return fmt.Errorf("reading %s: %w", path, err)
			}

			// Validate document type
			if !konfig.IsValidDocumentKind(kind) {
				if kind == "" {
					return fmt.Errorf(
						"not an Orkestra document — expected a 'kind' field (allowed kinds: %s)",
						konfig.ValidKindsString(),
					)
				}
				return fmt.Errorf(
					"invalid Orkestra document kind %q (allowed kinds: %s)",
					kind, konfig.ValidKindsString(),
				)
			}

			if konfig.IsMotifKind(kind) {
				return validateMotifFile(path)
			}
		}

		// Default: Katalog / Komposer validation
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

// detectKindFromFile peeks at a YAML file to read its kind field.
func detectKindFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var doc struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	return doc.Kind, nil
}

// validateMotifFile runs Motif-specific validation and prints results.
func validateMotifFile(path string) error {
	fmt.Println()
	fmt.Printf("%s\n", utils.Bold("Validating Motif: "+path))
	fmt.Println()

	errs := katalog.ValidateMotif(path)
	if len(errs) == 0 {
		fmt.Printf("  ✓ %s is valid\n", path)
		return nil
	}

	for _, e := range errs {
		fmt.Printf("  ✗ %s\n", e.Error())
	}
	fmt.Println()
	return fmt.Errorf("%d validation error(s) in %s", len(errs), path)
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringSliceP("file", "f", nil, "Path to katalog.yaml or komposer.yaml (can be specified multiple times or as comma-separated)")

	// Shadow global flags so they don't appear under `ork validate`
	validateCmd.Flags().Bool("debug", false, "")
	validateCmd.Flags().String("kubeconfig", "", "")
	validateCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	validateCmd.Flags().MarkHidden("debug")
	validateCmd.Flags().MarkHidden("kubeconfig")
	validateCmd.Flags().MarkHidden("verbose")
}
