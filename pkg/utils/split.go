package utils

import (
	"strings"
)

// SplitBySeparator splits a string by the given separator into a slice of trimmed strings.
// Empty elements are omitted from the result.
//
// Examples:
//
//	SplitBySeparator("a,b,c", ",")           // → ["a", "b", "c"]
//	SplitBySeparator("a, b, c", ",")         // → ["a", "b", "c"]
//	SplitBySeparator("a,,b", ",")            // → ["a", "b"]
//	SplitBySeparator("a|b|c", "|")           // → ["a", "b", "c"]
//	SplitBySeparator("a | b | c", "|")       // → ["a", "b", "c"]
//	SplitBySeparator("", ",")                // → [] (empty slice)
//	SplitBySeparator("  ", ",")              // → [] (empty slice)
//	SplitBySeparator("a, , b", ",")          // → ["a", "b"]
func SplitBySeparator(s, separator string) []string {
	if s == "" || separator == "" {
		return []string{}
	}

	parts := strings.Split(s, separator)
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// SplitCommaSeparated splits a comma-separated string into a slice of trimmed strings.
// Empty elements are omitted from the result.
//
// Examples:
//
//	SplitCommaSeparated("a,b,c")        // → ["a", "b", "c"]
//	SplitCommaSeparated("a, b, c")      // → ["a", "b", "c"]
//	SplitCommaSeparated("a,,b")         // → ["a", "b"]
//	SplitCommaSeparated("")             // → [] (empty slice)
//	SplitCommaSeparated("  ")           // → [] (empty slice)
//	SplitCommaSeparated("a, , b")       // → ["a", "b"]
func SplitCommaSeparated(s string) []string {
	return SplitBySeparator(s, ",")
}

// SplitPipeSeparated splits a pipe-separated string into a slice of trimmed strings.
// Empty elements are omitted from the result.
//
// Examples:
//
//	SplitPipeSeparated("a|b|c")        // → ["a", "b", "c"]
//	SplitPipeSeparated("a | b | c")    // → ["a", "b", "c"]
//	SplitPipeSeparated("a||b")         // → ["a", "b"]
func SplitPipeSeparated(s string) []string {
	return SplitBySeparator(s, "|")
}

// SplitColonSeparated splits a colon-separated string into a slice of trimmed strings.
// Empty elements are omitted from the result.
//
// Examples:
//
//	SplitColonSeparated("a:b:c")        // → ["a", "b", "c"]
//	SplitColonSeparated("a: b: c")      // → ["a", "b", "c"]
func SplitColonSeparated(s string) []string {
	return SplitBySeparator(s, ":")
}

// SplitSemicolonSeparated splits a semicolon-separated string into a slice of trimmed strings.
// Empty elements are omitted from the result.
//
// Examples:
//
//	SplitSemicolonSeparated("a;b;c")        // → ["a", "b", "c"]
//	SplitSemicolonSeparated("a; b; c")      // → ["a", "b", "c"]
func SplitSemicolonSeparated(s string) []string {
	return SplitBySeparator(s, ";")
}
