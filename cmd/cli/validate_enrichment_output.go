//go:build !runtime

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
// Shows enrichment clearly when it occurred.
func printCRDValidationLine(entry orktypes.CRDEntry) {
	icon := utils.HealthIcon("ready")
	fmt.Printf("%s %s\n", icon, utils.Bold(entry.Name))

	if entry.IsBuiltIn {
		// Enriched from built-in — tell the user what was resolved
		scope := "Namespaced"
		if !entry.IsNamespaced() {
			scope = "ClusterScoped"
		}

		fmt.Printf("    %s kind: %s %s built-in registry\n",
			utils.ColorGray,
			utils.Bold(entry.APITypes.Kind),
			"→ enriched from",
		)
		fmt.Printf("    group: %s / version: %s / plural: %s / scope: %s%s\n",
			entry.BuiltInGroup,
			entry.APITypes.Version,
			entry.APITypes.Plural,
			scope,
			utils.ColorReset,
		)
	} else {
		// Custom CRD — show the declared values
		scope := "Namespaced"
		if !entry.IsNamespaced() {
			scope = "ClusterScoped"
		}
		fmt.Printf("    %skind: %s / group: %s / version: %s / plural: %s / scope: %s%s\n",
			utils.ColorGray,
			entry.APITypes.Kind,
			entry.APITypes.Group,
			entry.APITypes.Version,
			entry.APITypes.Plural,
			scope,
			utils.ColorReset,
		)
	}

	// Add mode / workers / resync
	fmt.Printf("    %smode: %s / workers: %v / resync: %v%s\n",
		utils.ColorGray,
		entry.Mode,
		entry.Workers,
		entry.Resync,
		utils.ColorReset,
	)
}
