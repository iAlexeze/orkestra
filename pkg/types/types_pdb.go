// pkg/types/types_pdb.go
package types

// ── PodDisruptionBudget ───────────────────────────────────────────────────────

// PDBTemplateSource declares one PodDisruptionBudget to be managed by Orkestra.
//
// Example:
//
//	onReconcile:
//	  pdb:
//	    - name: "{{ .metadata.name }}-pdb"
//	      minAvailable: "1"
//	      selector:
//	        app: "{{ .metadata.name }}"
//	      forEach:
//	        field: spec.services
//	        as: item
type PDBTemplateSource struct {
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Name — PDB resource name. Default: "{{ .metadata.name }}-pdb"
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Namespace — target namespace. Default: CR namespace.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// Selector — label selector identifying the pods this PDB protects.
	// Keys are static; values support template expressions.
	Selector Labels `yaml:"selector,omitempty" json:"selector,omitempty"`

	// MinAvailable — minimum number of pods that must remain available.
	// Accepts integer strings ("1") or percentage strings ("50%").
	// Mutually exclusive with MaxUnavailable.
	MinAvailable string `yaml:"minAvailable,omitempty" json:"minAvailable,omitempty"`

	// MaxUnavailable — maximum number of pods that may be unavailable.
	// Accepts integer strings ("1") or percentage strings ("25%").
	// Mutually exclusive with MinAvailable.
	MaxUnavailable string `yaml:"maxUnavailable,omitempty" json:"maxUnavailable,omitempty"`

	// Labels applied to PDB metadata. Values support template expressions.
	Labels Labels `yaml:"labels,omitempty" json:"labels,omitempty"`

	// Behavior — disruption limit configuration.
	// Set behavior.profile for a named preset, or declare minAvailable/maxUnavailable explicitly.
	// Profile and explicit fields are mutually exclusive.
	Behavior *PDBBehavior `yaml:"behavior,omitempty" json:"behavior,omitempty"`

	Reconcile  bool         `yaml:"reconcile,omitempty" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}
