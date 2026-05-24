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
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// printCRDValidationLine prints one CRD entry's validation result.
// Shows enrichment clearly when it occurred, and any warnings.
func printCRDValidationLine(entry orktypes.CRDEntry, katalogProtected, katalogStrictMode bool) {
	printCRDHeader(entry)
	strictModeText := getStrictModeText(entry, katalogStrictMode)
	printKindInfo(entry)
	printModeResync(entry)
	printProtectionStatus(entry, katalogProtected, strictModeText)
	printWarnings(entry)
}

// printCRDHeader prints the CRD name with appropriate icon.
func printCRDHeader(entry orktypes.CRDEntry) {
	icon := utils.HealthIconReady()
	if entry.Warnings.HasWarnings() {
		icon = utils.HealthIconWarning()
	}
	fmt.Printf("%s %s\n", icon, utils.Bold(entry.Name))
}

// getStrictModeText returns a formatted string indicating strict mode status,
// or an empty string if strict mode is not enforced for this CRD.
func getStrictModeText(entry orktypes.CRDEntry, katalogStrictMode bool) string {
	if katalogStrictMode && entry.IsStrictDeletionProtection(katalogStrictMode) {
		return utils.InfoMark() + " (strict)"
	}
	return ""
}

// printKindInfo prints the kind/group/version/plural/scope line.
// For built‑in types, it shows enrichment info; for custom, the declared values.
func printKindInfo(entry orktypes.CRDEntry) {
	scope := "Namespaced"
	if !entry.IsNamespaced() {
		scope = "ClusterScoped"
	}

	if entry.IsBuiltIn {
		fmt.Printf("    %s\n", utils.Gray(fmt.Sprintf(
			"kind: %s → enriched from built-in registry", utils.Bold(entry.APITypes.Kind),
		)))
		fmt.Printf("    %s\n", utils.Gray(fmt.Sprintf(
			"group: %s / version: %s / plural: %s / scope: %s",
			entry.BuiltInGroup, entry.APITypes.Version, entry.APITypes.Plural, scope,
		)))
	} else {
		fmt.Printf("    %s\n", utils.Gray(fmt.Sprintf(
			"kind: %s / group: %s / version: %s / plural: %s / scope: %s",
			entry.APITypes.Kind, entry.APITypes.Group, entry.APITypes.Version, entry.APITypes.Plural, scope,
		)))
	}
}

// printModeResync prints the mode, workers, and resync period.
func printModeResync(entry orktypes.CRDEntry) {
	fmt.Printf("    %s\n", utils.Gray(fmt.Sprintf(
		"mode: %s / workers: %v / resync: %v",
		entry.Mode, entry.Workers, entry.Resync,
	)))
}

// printProtectionStatus prints the protection level for custom CRDs.
// For built‑ins, a simplified message is shown.
// printProtectionStatus prints the protection level for custom CRDs.
// For built‑ins, a simplified message is shown.
// When strict mode is active, the label changes from "protection:" to "strict‑protection:".
func printProtectionStatus(entry orktypes.CRDEntry, katalogProtected bool, strictModeText string) {
	if entry.IsBuiltInType() {
		fmt.Printf("    %s\n", utils.Gray("protection: label-based (built-in)"))
		return
	}

	if !katalogProtected {
		return
	}

	protectCRD := entry.ShouldProtectCRD()
	protectCRs := entry.ShouldProtectCRs()
	strictActive := strictModeText != "" // strictModeText already contains the arrow+text, but we only need a boolean

	var icon string
	var baseText string
	switch {
	case protectCRD && protectCRs:
		icon = utils.SecureMark()
		baseText = " full (CRD + CRs)"
	case protectCRD && !protectCRs:
		icon = utils.SomeSecureMark()
		baseText = "CRD only (CRs not protected)"
	case !protectCRD && protectCRs:
		icon = utils.WarningMark()
		baseText = "CRs only (CRD not protected - see warning)"
	default:
		icon = utils.NoSecurityMark()
		baseText = "none"
	}

	// Choose label based on strict mode
	label := "protection:"
	if strictActive {
		label = "strict-protection:"
	}
	fullText := fmt.Sprintf("%s %s %s", label, icon, baseText)
	fmt.Printf("    %s\n", utils.Gray(fullText))
}

// printWarnings prints any warnings associated with the CRD, with proper indentation.
func printWarnings(entry orktypes.CRDEntry) {
	for _, w := range entry.Warnings {
		lines := strings.Split(w, "\n")
		for i, line := range lines {
			if i == 0 {
				fmt.Printf("    warning: %s\n", utils.Gray(line))
			} else {
				fmt.Printf("             %s\n", utils.Gray(line)) // 13 spaces aligns with "warning: "
			}
		}
	}
}
