//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/konfig"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate an Orkestra document (Katalog, Komposer, Motif, E2E)",
	Long: `Validates any Orkestra document and reports errors.

The document kind is detected automatically from the 'kind' field:
  Katalog   — operator definition with CRD declarations
  Komposer  — multi-source katalog composer
  Motif     — reusable operator pattern
  E2E       — declarative end-to-end test spec

Reads katalog.yaml or komposer.yaml from the current directory by default.
Pass -f to validate a different file.

Examples:
  ork validate
  ork validate -f e2e.yaml
  ork validate -f motif.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, _ := cmd.Flags().GetStringSlice("file")
		expanded := parseFilePaths(paths)
		if len(expanded) == 0 {
			expanded = defaultFilePaths()
		}
		if len(expanded) == 0 {
			return fmt.Errorf(errNoKatalog)
		}

		var docKind string
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

			if konfig.IsE2EKind(kind) {
				return validateE2EFile(path)
			}

			docKind = kind
		}

		// Default path: Katalog / Komposer validation
		m, err := generateKatalog(cmd)
		if err != nil {
			return err
		}

		k, err := katalog.BuildExpanded(kfg, m.m)
		if err != nil {
			return err
		}
		entries := k.EnabledCRDs()

		kindLabel := "Katalog"
		if konfig.IsKomposerKind(docKind) {
			kindLabel = "Komposer"
		}

		if k.IsStandaloneGateway() {
			kindLabel = "Gateway Standalone"
		}

		fmt.Println()
		fmt.Println(utils.Bold("Validating " + kindLabel + "..."))
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

// validateE2EFile validates an E2E spec file and prints a summary.
func validateE2EFile(path string) error {
	fmt.Println()
	fmt.Println(utils.Bold("Validating E2E..."))
	fmt.Println()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var e2e orktypes.E2E
	if err := yaml.Unmarshal(data, &e2e); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	var errs []string

	if e2e.Metadata.Name == "" {
		errs = append(errs, "metadata.name is required")
	}
	if e2e.Spec.Katalog == "" && e2e.Spec.Init == nil {
		errs = append(errs, "spec.katalog is required (or spec.init for example packs)")
	}
	if e2e.Spec.CRD == "" && e2e.Spec.Init == nil {
		errs = append(errs, "spec.crd is required (or spec.init for example packs)")
	}
	if e2e.Spec.CR == "" && e2e.Spec.Init == nil {
		errs = append(errs, "spec.cr is required (or spec.init for example packs)")
	}
	if len(e2e.Spec.Expect) == 0 {
		errs = append(errs, "spec.expect must contain at least one expectation")
	}
	for i, exp := range e2e.Spec.Expect {
		if exp.Name == "" {
			errs = append(errs, fmt.Sprintf("spec.expect[%d].name is required", i))
		}
		if exp.After != "cr-applied" && exp.After != "cr-deleted" {
			errs = append(errs, fmt.Sprintf("spec.expect[%d].after must be cr-applied or cr-deleted (got %q)", i, exp.After))
		}
		if len(exp.Resources) == 0 && len(exp.Commands) == 0 {
			errs = append(errs, fmt.Sprintf("spec.expect[%d] (%q): must have at least one resource or command check", i, exp.Name))
		}
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Printf("  %s %s\n", utils.FailureMark(), e)
		}
		fmt.Println()
		return fmt.Errorf("%d validation error(s) in %s", len(errs), path)
	}

	icon := utils.HealthIcon("ready")
	fmt.Printf("%s %s\n", icon, utils.Bold(e2e.Metadata.Name))
	if e2e.Metadata.Description != "" {
		fmt.Printf("    %s\n", utils.Gray(e2e.Metadata.Description))
	}
	fmt.Printf("    %s\n",
		utils.Gray(fmt.Sprintf("katalog : %s\n    crd     : %s\n    cr      : %s",
			e2e.Spec.Katalog, e2e.Spec.CRD, e2e.Spec.CR)),
	)
	if len(e2e.Spec.Setup) > 0 {
		fmt.Printf("    %s\n", utils.Gray("setup   : "+strings.Join(e2e.Spec.Setup, ", ")))
	}
	fmt.Println()
	for _, exp := range e2e.Spec.Expect {
		to := exp.Timeout
		if to == "" {
			to = "60s"
		}
		fmt.Printf("    %s\n",
			utils.Gray(fmt.Sprintf("%-40s after: %-12s timeout: %s", exp.Name, exp.After, to)))
	}
	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("%d expectation(s) valid\n", len(e2e.Spec.Expect))

	return nil
}

// validateMotifFile runs Motif-specific validation and prints results.
func validateMotifFile(path string) error {
	fmt.Println()
	fmt.Println(utils.Bold("Validating Motif..."))
	fmt.Println()

	errs := katalog.ValidateMotif(path)
	if len(errs) == 0 {
		icon := utils.HealthIcon("ready")
		fmt.Printf("%s %s\n", icon, utils.Bold(path))
		fmt.Printf("    %s\n", utils.Gray("valid"))
		return nil
	}

	for _, e := range errs {
		fmt.Printf("  %s %s\n", utils.FailureMark(), e.Error())
	}
	fmt.Println()
	return fmt.Errorf("%d validation error(s) in %s", len(errs), path)
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringSliceP("file", "f", nil, "Path to an Orkestra document (repeatable or comma-separated)")

	// Shadow global flags so they don't appear under `ork validate`
	validateCmd.Flags().Bool("debug", false, "")
	validateCmd.Flags().String("kubeconfig", "", "")
	validateCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	validateCmd.Flags().MarkHidden("debug")
	validateCmd.Flags().MarkHidden("kubeconfig")
	validateCmd.Flags().MarkHidden("verbose")
}
