//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"path/filepath"

	"github.com/orkspace/orkestra/pkg/e2e"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/konfig"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	rbacv1 "k8s.io/api/rbac/v1"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate an Orkestra document (Katalog, Komposer, Motif, E2E, Simulate)",
	Long: `Validates any Orkestra document and reports errors.

The document kind is detected automatically from the 'kind' field:
  Katalog   — operator definition with CRD declarations
  Komposer  — multi-source katalog composer
  Motif     — reusable operator pattern
  E2E       — declarative end-to-end test spec
  Simulate  — declarative reconciler assertions

Reads katalog.yaml or komposer.yaml from the current directory by default.
Pass -f to validate a different file.

Examples:
  ork validate
  ork validate -f e2e.yaml
  ork validate -f simulate.yaml
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

			if konfig.IsSimulateKind(kind) {
				return validateSimulateFile(path)
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

		full, _ := cmd.Flags().GetBool("full")

		var perCRDPerms map[string][]rbacv1.PolicyRule
		if full {
			perCRDPerms = k.GeneratePerCRDRBACRules()
		}

		// Sort entries by name for stable output across runs.
		sortedEntries := make([]orktypes.CRDEntry, 0, len(entries))
		for _, e := range entries {
			sortedEntries = append(sortedEntries, e)
		}
		sort.Slice(sortedEntries, func(i, j int) bool {
			return sortedEntries[i].Name < sortedEntries[j].Name
		})

		fmt.Println()
		fmt.Println(bold("Validating " + kindLabel + "..."))
		fmt.Println()

		builtIn := 0
		custom := 0

		// Print each CRD entry with enrichment info
		for _, entry := range sortedEntries {
			printCRDValidationLine(entry, k.IsDeletionProtectionEnabled(), k.IsStrictModeEnabled())
			if full {
				printCRDPermissions(perCRDPerms[entry.Name])
				printCRDProfiles(entry)
			}
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

		if full {
			if dd := k.DependencyDisplayData(); dd != nil {
				printValidateDependencyGraph(dd)
			}
			printRuntimePermissionsSection(k.GenerateRuntimeRBACRules())
			printGatewayPermissionsSection(k.GenerateGatewayRBACRules())
			fmt.Println()
		}

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
	fmt.Println(bold("Validating E2E..."))
	fmt.Println()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var doc orktypes.E2E
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	baseDir := filepath.Dir(path)
	isAggregator := len(doc.Imports) > 0 && doc.Spec.Katalog == "" && doc.Spec.Init == nil

	var errs []string

	if doc.Metadata.Name == "" {
		errs = append(errs, "metadata.name is required")
	}
	if !isAggregator {
		if doc.Spec.Katalog == "" && doc.Spec.Init == nil && !doc.Spec.CustomOperator {
			errs = append(errs, "spec.katalog is required (or spec.init for example packs, spec.customOperator for custom operators, or imports)")
		}
		if doc.Spec.CRD == "" && doc.Spec.Init == nil && !doc.Spec.CustomOperator {
			errs = append(errs, "spec.crd is required (or spec.init for example packs, spec.customOperator for custom operators, or imports)")
		}
		if doc.Spec.CR == "" && doc.Spec.Init == nil {
			errs = append(errs, "spec.cr is required (or spec.init for example packs, or imports)")
		}
		if len(doc.Spec.Expect) == 0 {
			errs = append(errs, "spec.expect must contain at least one expectation")
		}
		for i, exp := range doc.Spec.Expect {
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
	}

	// Validate imports (collect per-import errors for display).
	importErrs := e2e.ValidateImports(baseDir, doc.Imports)

	if len(errs) > 0 || len(importErrs) > 0 {
		for _, e := range errs {
			fmt.Printf("  %s %s\n", failureMark(), e)
		}
		for _, ie := range importErrs {
			fmt.Printf("  %s import: %s\n", failureMark(), ie)
		}
		fmt.Println()
		return fmt.Errorf("%d validation error(s) in %s", len(errs)+len(importErrs), path)
	}

	icon := healthIcon("ready")
	fmt.Printf("%s %s\n", icon, bold(doc.Metadata.Name))
	if doc.Metadata.Description != "" {
		fmt.Printf("    %s\n", gray(doc.Metadata.Description))
	}
	if !isAggregator {
		if doc.Spec.CustomOperator {
			fmt.Printf("    %s\n", gray("mode    : custom operator (Orkestra install skipped)"))
		}
		fmt.Printf("    %s\n",
			gray(fmt.Sprintf("katalog : %s\n    crd     : %s\n    cr      : %s",
				doc.Spec.Katalog, doc.Spec.CRD, doc.Spec.CR)),
		)
		if s := doc.Spec.Setup; s != nil {
			if len(s.Apply) > 0 {
				fmt.Printf("    %s\n", gray("setup.apply : "+strings.Join(s.Apply, ", ")))
			}
			if len(s.Helm) > 0 {
				fmt.Printf("    %s\n", gray(fmt.Sprintf("setup.helm  : %d chart(s)", len(s.Helm))))
			}
			if len(s.Wait) > 0 {
				fmt.Printf("    %s\n", gray(fmt.Sprintf("setup.wait  : %d resource(s)", len(s.Wait))))
			}
		}
	}
	if len(doc.Imports) > 0 {
		fmt.Printf("    %s\n", gray(fmt.Sprintf("imports : %d file(s)", len(doc.Imports))))
		for _, imp := range doc.Imports {
			label := imp.Path
			if imp.FreshCluster {
				label += " (fresh cluster)"
			}
			if imp.Wait != "" {
				label += " (wait: " + imp.Wait + ")"
			}
			fmt.Printf("      %s %s\n", healthIcon("ready"), gray(label))
		}
	}
	fmt.Println()
	for _, exp := range doc.Spec.Expect {
		to := exp.Timeout
		if to == "" {
			to = "60s"
		}
		fmt.Printf("    %s\n",
			gray(fmt.Sprintf("%-40s after: %-12s timeout: %s", exp.Name, exp.After, to)))
	}
	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	if isAggregator {
		fmt.Printf("%d import(s) valid\n", len(doc.Imports))
	} else {
		fmt.Printf("%d expectation(s) valid", len(doc.Spec.Expect))
		if len(doc.Imports) > 0 {
			fmt.Printf(", %d import(s) valid", len(doc.Imports))
		}
		fmt.Println()
	}

	return nil
}

// validateSimulateFile validates a Simulate spec file and prints a summary.
func validateSimulateFile(path string) error {
	fmt.Println()
	fmt.Println(bold("Validating Simulate..."))
	fmt.Println()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var doc orktypes.Simulate
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	baseDir := filepath.Dir(path)
	isAggregator := doc.Imports != nil && len(doc.Imports.Files) > 0 && doc.Spec == nil

	check := func(ok bool, pass, fail string) {
		if ok {
			fmt.Printf("  %s %s\n", successMark(), pass)
		} else {
			fmt.Printf("  %s %s\n", failureMark(), fail)
		}
	}

	var errs []string

	if doc.Metadata.Name == "" {
		errs = append(errs, "metadata.name is required")
		fmt.Printf("  %s metadata.name is required\n", failureMark())
	} else {
		fmt.Printf("  %s metadata.name: %s\n", successMark(), doc.Metadata.Name)
	}

	if !isAggregator && doc.Spec == nil {
		errs = append(errs, "spec or imports is required")
		fmt.Printf("  %s spec or imports is required\n", failureMark())
	}

	if isAggregator {
		for _, f := range doc.Imports.Files {
			p := f
			if !filepath.IsAbs(p) {
				p = filepath.Join(baseDir, p)
			}
			check(fileExists(p), "imports.files: "+f+" (found)", "imports.files: "+f+" (not found)")
			if !fileExists(p) {
				errs = append(errs, "import not found: "+f)
			}
		}
	}

	if doc.Spec != nil {
		if doc.Spec.Katalog == "" {
			errs = append(errs, "spec.katalog is required")
			fmt.Printf("  %s spec.katalog is required\n", failureMark())
		} else {
			p := filepath.Join(baseDir, doc.Spec.Katalog)
			check(fileExists(p), "spec.katalog: "+doc.Spec.Katalog+" (found)", "spec.katalog: "+doc.Spec.Katalog+" (not found)")
			if !fileExists(p) {
				errs = append(errs, "spec.katalog not found: "+doc.Spec.Katalog)
			}
		}

		if doc.Spec.CR == "" {
			errs = append(errs, "spec.cr is required")
			fmt.Printf("  %s spec.cr is required\n", failureMark())
		} else {
			p := filepath.Join(baseDir, doc.Spec.CR)
			check(fileExists(p), "spec.cr: "+doc.Spec.CR+" (found)", "spec.cr: "+doc.Spec.CR+" (not found)")
			if !fileExists(p) {
				errs = append(errs, "spec.cr not found: "+doc.Spec.CR)
			}
		}

		if doc.Spec.Cycles <= 0 {
			fmt.Printf("  %s spec.cycles: not set — defaulting to 10\n", yellow("⚠"))
		} else {
			fmt.Printf("  %s spec.cycles: %d\n", successMark(), doc.Spec.Cycles)
		}

		if doc.Spec.Expect != nil {
			fmt.Printf("  %s expect.ops: %d rule(s)\n", successMark(), len(doc.Spec.Expect.Ops))
			validVerbs := map[string]bool{"create": true, "update": true, "delete": true, "patch": true}
			for i, rule := range doc.Spec.Expect.Ops {
				switch {
				case rule.Verb == "" || rule.Resource == "":
					errs = append(errs, fmt.Sprintf("expect.ops[%d]: verb and resource are required", i))
					fmt.Printf("  %s expect.ops[%d]: verb and resource are required\n", failureMark(), i)
				case !validVerbs[rule.Verb]:
					errs = append(errs, fmt.Sprintf("expect.ops[%d]: invalid verb %q (must be create, update, delete, or patch)", i, rule.Verb))
					fmt.Printf("  %s expect.ops[%d]: invalid verb %q\n", failureMark(), i, rule.Verb)
				}
			}
			for i, rule := range doc.Spec.Expect.Absent {
				switch {
				case rule.Verb == "" || rule.Resource == "":
					errs = append(errs, fmt.Sprintf("expect.absent[%d]: verb and resource are required", i))
					fmt.Printf("  %s expect.absent[%d]: verb and resource are required\n", failureMark(), i)
				case !validVerbs[rule.Verb]:
					errs = append(errs, fmt.Sprintf("expect.absent[%d]: invalid verb %q (must be create, update, delete, or patch)", i, rule.Verb))
					fmt.Printf("  %s expect.absent[%d]: invalid verb %q\n", failureMark(), i, rule.Verb)
				}
			}
			if len(doc.Spec.Expect.Absent) > 0 {
				fmt.Printf("  %s expect.absent: %d rule(s)\n", successMark(), len(doc.Spec.Expect.Absent))
			}
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))

	if len(errs) > 0 {
		return fmt.Errorf("%d validation error(s) in %s", len(errs), path)
	}

	if isAggregator {
		fmt.Printf("%d import(s) valid\n", len(doc.Imports.Files))
	} else {
		ops := 0
		if doc.Spec.Expect != nil {
			ops = len(doc.Spec.Expect.Ops)
		}
		fmt.Printf("Simulate is valid (%d op rule(s))\n", ops)
	}
	return nil
}

// validateMotifFile runs Motif-specific validation and prints results.
func validateMotifFile(path string) error {
	fmt.Println()
	fmt.Println(bold("Validating Motif..."))
	fmt.Println()

	errs := katalog.ValidateMotif(path)
	if len(errs) == 0 {
		icon := healthIcon("ready")
		fmt.Printf("%s %s\n", icon, bold(path))
		fmt.Printf("    %s\n", gray("valid"))
		return nil
	}

	for _, e := range errs {
		fmt.Printf("  %s %s\n", failureMark(), e.Error())
	}
	fmt.Println()
	return fmt.Errorf("%d validation error(s) in %s", len(errs), path)
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringSliceP("file", "f", nil, "Path to an Orkestra document (repeatable or comma-separated)")
	validateCmd.Flags().Bool("full", false, "Show per-CRD permissions, dependency graph, and system-level RBAC")

	// Shadow global flags so they don't appear under `ork validate`
	validateCmd.Flags().Bool("debug", false, "")
	validateCmd.Flags().String("kubeconfig", "", "")
	validateCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	validateCmd.Flags().MarkHidden("debug")
	validateCmd.Flags().MarkHidden("kubeconfig")
	validateCmd.Flags().MarkHidden("verbose")
}
