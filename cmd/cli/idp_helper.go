//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/spf13/cobra"
)

var (
	validIDPOperations      = strings.Join(orktypes.ValidIDPOperations(), ", ")
	validIDPEndpointClasses = strings.Join(orktypes.ValidIDPEndpointClasses(), ", ")
)

// ── buildKatalog ────────────────────────────────────────────────────────────

// buildKatalog builds the expanded Katalog using the same logic as ork validate.
// It respects the --file global flag and reuses the merger.
func buildKatalog(cmd *cobra.Command) (*katalog.Katalog, error) {
	m, err := generateKatalog(cmd)
	if err != nil {
		return nil, fmt.Errorf("generating Katalog: %w", err)
	}

	k, err := katalog.BuildExpanded(kfg, m.m)
	if err != nil {
		return nil, fmt.Errorf("building expanded Katalog: %w", err)
	}

	return k, nil
}

// ── printIDPValidationSummary ──────────────────────────────────────────────

// printIDPValidationSummary prints a detailed breakdown of the IDP configuration.
func printIDPValidationSummary(kat *katalog.Katalog) {
	enabledCRDs := kat.IDPEnabledCRDs()

	if len(enabledCRDs) == 0 {
		fmt.Println("\nNo IDP-enabled CRDs found.")
		return
	}

	fmt.Println()
	fmt.Println(bold("IDP Configuration Summary"))
	fmt.Println(strings.Repeat("─", 70))

	for _, crd := range enabledCRDs {
		if crd == nil {
			continue
		}

		fmt.Printf("\n%s %s\n", healthIconReady(), bold(crd.Name))

		// ── Target ──────────────────────────────────────────────────────
		target := crd.IDPTarget()
		kind := crd.Kind()
		fmt.Printf("  %s\n", gray(fmt.Sprintf("target: %s  /  kind: %s", target, kind)))

		// ── Name/Namespace ──────────────────────────────────────────────
		if crd.HasIDPName() {
			fmt.Printf("  %s\n", gray(fmt.Sprintf("name:      %q", crd.IDP.Name)))
		} else {
			fmt.Printf("  %s\n", gray("name:      (caller must supply)"))
		}
		if crd.HasIDPNamespace() {
			fmt.Printf("  %s\n", gray(fmt.Sprintf("namespace: %q", crd.IDP.Namespace)))
		} else {
			fmt.Printf("  %s\n", gray("namespace: (cluster-scoped or not set)"))
		}

		// ── Fields ──────────────────────────────────────────────────────
		fields := crd.IDPFields()
		if len(fields) > 0 {
			var specFields, labelFields, annoFields int
			var nestedPaths int
			for name := range fields {
				if _, ok := crd.AdditionalLabelFields()[name]; ok {
					labelFields++
				} else if _, ok := crd.AdditionalAnnotationFields()[name]; ok {
					annoFields++
				} else {
					specFields++
				}
				if config, ok := crd.IDP.Fields[name]; ok && config.HasSpecPath() && strings.Contains(config.SpecPath(name), ".") {
					nestedPaths++
				}
			}
			fmt.Printf("  %s\n", gray(fmt.Sprintf("fields:    %d total (spec: %d, labels: %d, annotations: %d)", len(fields), specFields, labelFields, annoFields)))
			if nestedPaths > 0 {
				fmt.Printf("  %s\n", gray(fmt.Sprintf("nested:    %d path(s)", nestedPaths)))
			}
		} else {
			fmt.Printf("  %s\n", gray("fields:    none"))
		}

		// ── Tokens ──────────────────────────────────────────────────────
		if crd.HasIDPTokenRestrictions() {
			tokenCount := len(crd.IDP.AllowedTokensMap())
			fmt.Printf("  %s\n", gray(fmt.Sprintf("tokens:    %d token(s) with restrictions", tokenCount)))
		} else {
			fmt.Printf("  %s\n", gray("tokens:    none (all tokens allowed)"))
		}

		// ── Response Config ─────────────────────────────────────────────
		cfg := crd.GetIDPResponseConfig()
		if cfg != nil {
			payloadCount := len(cfg.Payload)
			excludeCount := len(cfg.Exclude)
			defaultVal := "true"
			if !cfg.UseDefault() {
				defaultVal = "false"
			}
			fmt.Printf("  %s\n", gray(fmt.Sprintf("response:  default: %s, payload: %d, exclude: %d", defaultVal, payloadCount, excludeCount)))
			if cfg.HasPoll() {
				pollParts := []string{}
				if cfg.Poll.URL != "" {
					pollParts = append(pollParts, "url")
				}
				if cfg.Poll.Field != "" {
					pollParts = append(pollParts, "field")
				}
				fmt.Printf("  %s\n", gray(fmt.Sprintf("poll:      %s", strings.Join(pollParts, ", "))))
			}
		} else {
			fmt.Printf("  %s\n", gray("response:  none (default CR response)"))
		}

		// ── Warnings ─────────────────────────────────────────────────────
		if crd.Warnings.HasWarnings() {
			fmt.Printf("  %s\n", gray("warnings:"))
			for _, w := range crd.Warnings {
				fmt.Printf("    %s %s\n", yellow("⚠"), gray(w))
			}
		}
	}

	// ── Summary ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Printf("%s IDP configuration is valid\n", successMark())
	fmt.Printf("  %d IDP-enabled CRD(s)\n", len(enabledCRDs))
	fmt.Println()
}
