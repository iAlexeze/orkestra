package customresources

import orktypes "github.com/orkspace/orkestra/pkg/types"

// ── ConfigMap ─────────────────────────────────────────────────────────────────

// ConfigMapTemplateSource declares one ConfigMap to be managed by Orkestra.
//
// Three usage patterns:
//
// 1. Static data:
//
//	onCreate:
//	  configMaps:
//	    - name: "{{ .metadata.name }}-config"
//	      data:
//	        LOG_LEVEL: info
//	        MAX_CONNECTIONS: "100"
//
// 2. Copy from existing ConfigMap:
//
//	onCreate:
//	  configMaps:
//	    - name: app-config
//	      fromConfigMap: base-app-config
//	      fromNamespace: platform
//
// 3. Copy + override specific keys:
//
//	onCreate:
//	  configMaps:
//	    - name: app-config
//	      fromConfigMap: base-app-config
//	      fromNamespace: platform
//	      data:
//	        LOG_LEVEL: debug     # overrides the base value
//
// 4. Copy to multiple namespaces:
//
//	onCreate:
//	  configMaps:
//	    - name: app-config
//	      fromConfigMap: base-app-config
//	      toNamespaces:
//	        - "{{ .metadata.namespace }}"
//	        - staging
//	        - production
type CustomResourceTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string

	// APIVersion is required and must be a group/version string (e.g. "foo.io/v1").
	// This field is used to derive the GroupVersionKind for REST mapping.
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is required and must be a valid Kubernetes Kind (e.g. "Bar").
	// Used together with APIVersion to resolve the GVR for dynamic client calls.
	Kind string `json:"kind" yaml:"kind"`

	// Metadata mirrors the subset of metav1.ObjectMeta Orkestra needs.
	// Implementations must ensure metadata.Name is present after templating.
	// Namespace is required for namespaced CRDs; for cluster-scoped CRDs the
	// namespace field should be empty. Whether a CRD is namespaced is determined
	// by discovery/validation and not by this struct alone.
	Metadata orktypes.CustomResourceMetadata `json:"metadata" yaml:"metadata"`

	// Spec is the conventional spec block for CRDs. It is schema-agnostic and
	// may contain templated values. Only template syntax is validated by
	// Orkestra; structural/schema validation is deferred to the API server.
	Spec map[string]any `json:"spec,omitempty" yaml:"spec,omitempty"`

	// Status is allowed in the declaration for convenience (for example when
	// bootstrapping resources that expect an initial status). Orkestra will
	// only attempt to write status if HasStatus() returns true.
	// Users should prefer letting the controller that owns the CR populate status.
	Status map[string]any `json:"status,omitempty" yaml:"status,omitempty"`

	// Other captures any top-level fields that are not spec/status/metadata.
	// This supports CRDs that place configuration at the top level instead of
	// under spec. This field is inlined during YAML/JSON unmarshalling.
	Other map[string]any `json:"-" yaml:",inline"`

	// HasStatus is an explicit hint about whether the CRD exposes a status
	// subresource. Three states are useful:
	//   - nil: auto-detect via discovery at runtime
	//   - true: force status writes (patches)
	//   - false: never attempt status writes
	// Use this to avoid API errors for CRDs that do not support status.
	HasStatus *bool `json:"hasStatus,omitempty" yaml:"hasStatus,omitempty"`

	Reconcile bool `yaml:"reconcile" json:"reconcile,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}
