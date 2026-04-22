package secrets

import orktypes "github.com/orkspace/orkestra/pkg/types"

// ── Secret ────────────────────────────────────────────────────────────────────

// SecretTemplateSource declares one Secret to be managed by Orkestra.
//
// Three usage patterns:
//
// 1. Static data — declare values directly in the Katalog:
//
//	onCreate:
//	  secrets:
//	    - name: "{{ .metadata.name }}-config"
//	      data:
//	        API_KEY: my-api-key
//	        DB_URL: postgres://...
//
// 2. Copy from existing Secret — Orkestra reads the source and copies it:
//
//	onCreate:
//	  secrets:
//	    - name: db-credentials
//	      fromSecret: master-db-creds
//	      fromNamespace: platform
//
// 3. Copy to multiple namespaces — Orkestra creates one copy per namespace:
//
//	onCreate:
//	  secrets:
//	    - name: db-credentials
//	      fromSecret: master-db-creds
//	      fromNamespace: platform
//	      toNamespaces:
//	        - "{{ .metadata.namespace }}"
//	        - monitoring
//	        - staging
//
// All copies are owned by the CR — deleted automatically when the CR is deleted.
type SecretTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string `yaml:"version" validate:"omitempty"`

	// Name — Secret name in the target namespace.
	// Default: "{{ .metadata.name }}-secret"
	Name string `yaml:"name" validate:"omitempty"`

	// Namespace — primary target namespace.
	// Default: "{{ .metadata.namespace }}"
	// When ToNamespaces is set, this field is ignored.
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// ToNamespaces — create one copy of this Secret in each listed namespace.
	// Each element supports template expressions.
	// e.g. ["{{ .metadata.namespace }}", "monitoring", "staging"]
	// When set, Namespace is ignored — ToNamespaces controls all target namespaces.
	ToNamespaces []string `yaml:"toNamespaces" validate:"omitempty"`

	// FromSecret — name of an existing Secret to copy data from.
	// When set, Orkestra reads this Secret at reconcile time and copies its data.
	// This means the copy stays in sync — if the source changes, the copy updates.
	// Omit to use static Data entries instead.
	FromSecret string `yaml:"fromSecret" validate:"omitempty"`

	// FromNamespace — namespace where FromSecret lives.
	// Default: same namespace as the CR.
	FromNamespace string `yaml:"fromNamespace" validate:"omitempty"`

	// Data — static key-value Secret entries (string values).
	// Kubernetes encodes them to base64 automatically.
	// When FromSecret is also set, these entries override matching keys from the source.
	Data map[string]string `yaml:"data" validate:"omitempty"`

	// Type — Kubernetes Secret type.
	// Default: Opaque.
	// e.g. "kubernetes.io/tls", "kubernetes.io/dockerconfigjson"
	Type string `yaml:"type" validate:"omitempty"`

	// Labels — applied to all created Secret copies.
	Labels []orktypes.ResourceLabel `yaml:"labels" validate:"omitempty"`

	// Reconcile: true — also sync on every reconcile (drift correction).
	// When true, if the source Secret changes, all copies are updated automatically.
	Reconcile bool `yaml:"reconcile" validate:"omitempty"`
}
