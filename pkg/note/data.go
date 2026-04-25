package note

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// Data transformation
// - toBase64
// - fromBase64
// - toJSON
// - sha256sum
// - truncate
// - slugify

func dataNotes() template.FuncMap {
	return template.FuncMap{
		"toBase64":   noteToBase64,
		"fromBase64": noteFromBase64,
		"toJSON":     noteToJSON,
		"sha256sum":  noteSHA256Sum,
		"truncate":   noteTruncate,
		"slugify":    noteSlugify,
	}
}

// ── Data transformation notes ─────────────────────────────────────────────────

// noteToBase64 base64-encodes a string. Used for Secret.data values.
//
//	{{ toBase64 .spec.password }}
func noteToBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// noteFromBase64 decodes a base64 string. Used to read Secret.data values
// in status fields or cross-CRD templates.
//
//	{{ fromBase64 .children.secret.data.password }}
func noteFromBase64(s string) string {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(b)
}

// noteToJSON marshals any value to a JSON string.
// Useful for embedding structured data in annotations or ConfigMap values.
//
//	{{ toJSON .spec.config }}
func noteToJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// noteSHA256Sum returns the first 8 characters of the SHA256 hash of a string.
// Use for deterministic, stable derived names and change detection:
//
//	{{ sha256sum .spec.config }}             → "a3f5c2b1"
//	name: "config-{{ sha256sum .spec.config }}"  → content-addressed naming
func noteSHA256Sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:4])
}

// noteTruncate returns the string truncated to maxLen characters.
// Kubernetes resource names have a 253-character limit; labels have 63.
// Use to safely generate resource names from user input:
//
//	name: "{{ truncate .spec.projectName 50 }}-deployment"
func noteTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

var (
	slugNonAlnum  = regexp.MustCompile(`[^a-z0-9-]`)
	slugMultiDash = regexp.MustCompile(`-+`)
)

// noteSlugify converts a string to a Kubernetes-safe name segment.
// Lowercases, replaces non-alphanumeric characters with dashes,
// and trims leading/trailing dashes.
//
//	{{ slugify "My App / Service" }}  → "my-app-service"
//	name: "{{ slugify .spec.teamName }}-operator"
func noteSlugify(s string) string {
	s = strings.ToLower(s)
	s = slugNonAlnum.ReplaceAllString(s, "-")
	s = slugMultiDash.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
