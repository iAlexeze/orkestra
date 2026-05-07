package doctor

import (
	"fmt"
	"strings"
)

// ImageTag returns the full image reference for a build.
// tag defaults to the git commit short SHA when empty.
func ImageTag(registry, appName, tag string) string {
	return fmt.Sprintf("%s/%s:%s", registry, appName, tag)
}

// truncate returns the string s shortened to at most n runes.
// Safe for Unicode: it never splits multi‑byte characters.
func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n])
}

// cleanUp normalizes a string for use in identifiers by removing commas,
// replacing dashes and dots with underscores, and stripping spaces.
func cleanUp(s string) string {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

// indent prefixes each non‑empty line in s with the given number of spaces.
// Used for embedding marshalled YAML blocks inside larger YAML documents.
// Keeps generated katalog output properly aligned and readable.
func indent(s string, spaces int) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lines[i] != "" {
			lines[i] = pad + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}
