//go:build !runtime && !gateway

package cli

import "github.com/orkspace/orkestra/pkg/utils"

// E2E status strings for registry info / list output.

func e2eVerified(suffix string) string {
	s := utils.Green("✓ Verified")
	if suffix != "" {
		s += " · " + suffix
	}
	return s
}

func e2eSkipped() string {
	return utils.Yellow("~ Skipped") + " (pushed with --force or --no-e2e)"
}

func e2eNotVerified() string {
	return utils.Gray("- Not verified")
}

// Diff change icons used in simulate output.

func iconAdded() string   { return utils.Green("+") }
func iconRemoved() string { return utils.Red("-") }
func iconChanged() string { return utils.Yellow("~") }
