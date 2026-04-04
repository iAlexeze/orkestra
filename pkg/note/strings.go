package note

import (
	"strings"
	"text/template"
)

func stringNotes() template.FuncMap {
	return template.FuncMap{
		"toLower":      strings.ToLower,
		"toUpper":      strings.ToUpper,
		"trimSpace":    strings.TrimSpace,
		"trim":         strings.Trim,
		"trimPrefix":   strings.TrimPrefix,
		"trimSuffix":   strings.TrimSuffix,
		"hasPrefix":    strings.HasPrefix,
		"hasSuffix":    strings.HasSuffix,
		"contains":     strings.Contains,
		"replace":      strings.ReplaceAll,
		"split":        strSplit,
		"join":         strings.Join,
		"repeat":       strings.Repeat,
		"camelToKebab": camelToKebab,
		"truncate":     strTruncate,
	}
}

// strSplit splits s by sep. Returns empty slice for empty input.
// Compose with index to extract specific elements:
//
//	{{ index (split .spec.tags ",") 0 }}   →  first tag
func strSplit(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, sep)
}

// camelToKebab converts CamelCase or PascalCase to kebab-case.
//
//	{{ camelToKebab "WebsiteOperator" }}   →  "website-operator"
//	{{ camelToKebab "myAppName" }}         →  "my-app-name"
func camelToKebab(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r + 32)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// strTruncate truncates s to at most n characters, appending "..." if truncated.
// Label values in Kubernetes have a 63-character limit — use this to enforce it.
//
//	{{ truncate .metadata.name 63 }}
func strTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}
