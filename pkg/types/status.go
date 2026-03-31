// pkg/types/status.go
package orktypes

// ── Status management ──────────────────────────────────────────────────────
//
// Orkestra writes status in two layers:
//
// Layer 1 — automatic standard Kubernetes conditions.
// After every reconcile, regardless of outcome, Orkestra patches the CR's
// /status subresource with a standard Ready condition and observedGeneration.
// No Katalog declaration required. Every managed CR gets this automatically,
// the same way every CR gets the managed label and finalizers.
//
// Layer 2 — declarative status fields.
// Optional. Declared in the Katalog under reconciler.status.fields.
// Resolved after reconcile templates complete, patched to /status.
// Field values support the same Go template expressions as onCreate templates.
//
// Example:
//
//	reconciler:
//	  status:
//	    fields:
//	      - path: phase
//	        value: "Running"
//	      - path: observedReplicas
//	        value: "{{ .spec.replicas }}"
//	      - path: endpoint
//	        value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"
//	      - path: database.host    # nested — becomes status.database.host
//	        value: "{{ .spec.host }}"
//
// Paths are relative to status — "phase" becomes status.phase.
// Dot-notation is supported for nested fields at any depth.
//
// Layer 3 — child resource status propagation — is designed but not yet
// implemented. It will add a "children" context to the resolver, making
// child resource status fields accessible in path expressions.

// StatusFieldSpec declares one field to write into the CR's status.
//
// Path is relative to the status subobject. "phase" writes to status.phase.
// "database.host" writes to status.database.host.
//
// Value supports Go text/template expressions evaluated against the CR:
//   - "{{ .spec.replicas }}"    — from the CR's spec
//   - "{{ .metadata.name }}"   — CR name
//   - "Running"                 — static string (fast path, no template parsing)
type StatusFieldSpec struct {
	// Path — dot-notation path relative to status.
	// "phase"            → status.phase
	// "database.host"    → status.database.host
	// "ready"            → status.ready
	Path string `yaml:"path" json:"path" validate:"required"`

	// Value — the value to write. Supports template expressions.
	// Resolved against the full CR object map at reconcile time.
	// Empty string is written as-is — declare a static empty string to clear a field.
	Value string `yaml:"value" json:"value"`
}

// StatusConfig declares the declarative status behavior for a CRD.
// Declared under spec.crds[].reconciler.status in the Katalog.
type StatusConfig struct {
	// Fields — list of status fields to write after every successful reconcile.
	// Resolved in declaration order. Later fields win on path conflict.
	Fields []StatusFieldSpec `yaml:"fields,omitempty" json:"fields,omitempty"`

	// Conditions — whether to write the standard Ready condition automatically.
	// Default: true. Set to false to opt out of automatic condition management.
	// When true, Orkestra writes:
	//   status.conditions[type=Ready]:
	//     status: "True" | "False"
	//     reason: ReconcileSucceeded | ReconcileError
	//     message: "" | "<error message>"
	//     lastTransitionTime: <now>
	//   status.observedGeneration: <metadata.generation>
	//
	// Setting false is rarely needed. Do it only when your CRD's status
	// schema explicitly forbids a conditions field, or when you manage
	// conditions entirely via Go hooks.
	Conditions *bool `yaml:"conditions,omitempty" json:"conditions,omitempty"`
}

// ConditionsEnabled reports whether automatic standard condition management is on.
// Defaults to true when Conditions is nil.
func (s *StatusConfig) ConditionsEnabled() bool {
	if s == nil || s.Conditions == nil {
		return true // default on
	}
	return *s.Conditions
}

// HasFields reports whether any declarative status fields are declared.
func (s *StatusConfig) HasFields() bool {
	return s != nil && len(s.Fields) > 0
}
