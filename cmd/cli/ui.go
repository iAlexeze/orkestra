//go:build !runtime && !gateway

package cli

import "strings"

// E2E status strings for registry info / list output.

func e2eVerified(suffix string) string {
	s := green("✓ Verified")
	if suffix != "" {
		s += " · " + suffix
	}
	return s
}

func e2eSkipped() string {
	return yellow("⊘ Skipped") + " (pushed with --force or --no-e2e)"
}

func e2eNotVerified() string {
	return gray("- Not verified")
}

func simulateVerified(suffix string) string {
	s := green("✓ Verified")
	if suffix != "" {
		s += " · " + suffix
	}
	return s
}

func simulateSkipped() string {
	return yellow("⊘ Skipped") + " (pushed with --force or --no-simulate)"
}

func simulateNoAssertion() string {
	return yellow("⚠ No assertions") + " (add expect: to simulate.yaml to enforce behavior)"
}

// Diff change icons used in simulate output.

func iconAdded() string   { return green("+") }
func iconRemoved() string { return red("-") }
func iconChanged() string { return yellow("~") }

// visibleLen returns the visible rune count of s, stripping ANSI escape sequences.
func visibleLen(s string) int {
	inEscape := false
	n := 0
	for _, r := range s {
		if r == '\033' {
			inEscape = true
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		n++
	}
	return n
}

// padRight pads s with trailing spaces until its visible width reaches width.
func padRight(s string, width int) string {
	if w := visibleLen(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}
