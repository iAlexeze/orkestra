package types

// NormalizeConfig declares template-driven spec normalization.
// This phase runs BEFORE mutation, validation, and reconciliation.
//
// Purpose:
//   - Accept multiple user-facing shapes (string vs map, list vs scalar, etc.)
//   - Collapse them into a single canonical spec used internally.
//   - Avoid drift where the CR stores one shape but children require another.
//   - Provide a declarative alternative to conversion webhooks.
//
// Behavior:
//   - normalize.spec is a map of field → template string.
//   - Each template is evaluated against the RAW CR object.
//   - The rendered values overwrite the corresponding fields in .spec.
//   - Only declared fields are overwritten; others remain untouched.
//   - The normalized object is passed to NewResolver() and all downstream phases.
//   - The stored CR in etcd is NOT modified.
//
// Nested paths are supported:
//
//	normalize:
//	  spec:
//	    resources.requests.cpu: "{{ default .spec.resources.requests.cpu \"100m\" }}"
//	    containers.0.image: "{{ .spec.image }}:{{ .spec.tag }}"
//
// NormalizeConfig declares field normalizations that run before mutation,
// validation, and template rendering.
//
// Keys are dot-notation paths into spec (e.g. "schedule", "resources.limits.cpu").
// Values are template expressions evaluated against the raw CR.
// Results are written back into the in-memory spec copy.
type NormalizeConfig struct {
	// Spec contains field-level normalization templates.
	// Example:
	//
	//   normalize:
	//     spec:
	//       schedule: >
	//         {{ if typeMap .spec.schedule }}
	//           {{ cronFromMap .spec.schedule }}
	//         {{ else }}
	//           {{ cronNormalize .spec.schedule }}
	//         {{ end }}
	//
	// Spec maps a dot-notation field path to a template expression.
	// The template sees the raw CR (before any normalization of other fields).
	// Results are coerced to the appropriate Go type via YAML parsing:
	//   "3"     → int
	//   "true"  → bool
	//   "*/5 * * * *" → string
	// Empty result ("") sets the field to empty string — not nil.
	Spec map[string]string `yaml:"spec,omitempty" json:"spec,omitempty"`
}
