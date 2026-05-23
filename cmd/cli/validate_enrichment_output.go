//go:build !runtime && !gateway

package cli

// printEnrichmentReport prints the enrichment summary as part of ork validate output.
//
// Example output for a Katalog with built-in and custom CRDs:
//
//   ✓ deployment-governance
//     kind: Deployment → enriched from built-in registry
//     group: apps / version: v1 / plural: deployments / scope: Namespaced
//
//   ✓ pod-governance
//     kind: Pod → enriched from built-in registry
//     group: core / version: v1 / plural: pods / scope: Namespaced
//
//   ✓ website
//     kind: Website / group: demo.orkestra.io / version: v1alpha1 / plural: websites
//
//   ✓ platform-namespace
//     kind: PlatformNamespace / group: platform.orkestra.io / version: v1alpha1

import (
	"fmt"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// printCRDValidationLine prints one CRD entry's validation result.
// Shows enrichment clearly when it occurred, and any warnings.
func printCRDValidationLine(entry orktypes.CRDEntry, protected bool) {
	// Decide icon: warning mark if warnings exist, otherwise success mark
	var icon string
	if entry.Warnings.HasWarnings() {
		icon = utils.HealthIconWarning()
	} else {
		icon = utils.HealthIconReady()
	}
	fmt.Printf("%s %s\n", icon, utils.Bold(entry.Name))

	// Print kind/group/version/scope line (existing)
	if entry.IsBuiltIn {
		// Enriched from built-in — tell the user what was resolved
		scope := "Namespaced"
		if !entry.IsNamespaced() {
			scope = "ClusterScoped"
		}
		fmt.Printf("    %s\n", utils.Gray(fmt.Sprintf(
			"kind: %s → enriched from built-in registry", utils.Bold(entry.APITypes.Kind),
		)))
		fmt.Printf("    %s\n", utils.Gray(fmt.Sprintf(
			"group: %s / version: %s / plural: %s / scope: %s",
			entry.BuiltInGroup, entry.APITypes.Version, entry.APITypes.Plural, scope,
		)))
	} else {
		// Custom CRD — show the declared values
		scope := "Namespaced"
		if !entry.IsNamespaced() {
			scope = "ClusterScoped"
		}
		fmt.Printf("    %s\n", utils.Gray(fmt.Sprintf(
			"kind: %s / group: %s / version: %s / plural: %s / scope: %s",
			entry.APITypes.Kind, entry.APITypes.Group, entry.APITypes.Version, entry.APITypes.Plural, scope,
		)))
	}

	// Mode / workers / resync line
	fmt.Printf("    %s\n", utils.Gray(fmt.Sprintf(
		"mode: %s / workers: %v / resync: %v",
		entry.Mode, entry.Workers, entry.Resync,
	)))

	// Protection status (only meaningful for custom CRDs; built‑ins show a simplified message)
	if entry.IsBuiltInType() {
		fmt.Printf("    %s\n", utils.Gray("protection: label‑based (built‑in)"))
	} else {
		if protected {
			protectCRD := entry.ShouldProtectCRD()
			protectCRs := entry.ShouldProtectCRs()
			switch {
			case protectCRD && protectCRs:
				fmt.Printf("    %s\n", utils.Gray("protection: "+utils.SecureMark()+"  full (CRD + CRs)"))
			case protectCRD && !protectCRs:
				fmt.Printf("    %s\n", utils.Gray("protection: "+utils.SomeSecureMark()+" CRD only (CRs not protected)"))
			case !protectCRD && protectCRs:
				fmt.Printf("    %s\n", utils.Gray("protection: "+utils.WarningMark()+" CRs only (CRD not protected – see warning)"))
			default:
				fmt.Printf("    %s\n", utils.Gray("protection: "+utils.NoSecurityMark()+" none"))
			}
		}
	}

	// Print any warnings (indented, one per line)
	for _, w := range entry.Warnings {
		fmt.Printf("    warning: %s\n", utils.Gray(w))
	}
}
