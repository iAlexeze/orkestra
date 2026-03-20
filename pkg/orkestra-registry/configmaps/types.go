package configmaps

import orktypes "github.com/ialexeze/orkestra/pkg/types"

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
type ConfigMapTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string `yaml:"version" validate:"omitempty"`

	// Name — ConfigMap name.
	// Default: "{{ .metadata.name }}-config"
	Name string `yaml:"name" validate:"omitempty"`

	// Namespace — primary target namespace.
	// Default: "{{ .metadata.namespace }}"
	Namespace string `yaml:"namespace" validate:"omitempty"`

	// ToNamespaces — create one copy in each listed namespace.
	// Each element supports template expressions.
	ToNamespaces []string `yaml:"toNamespaces" validate:"omitempty"`

	// FromConfigMap — name of an existing ConfigMap to copy data from.
	// Orkestra reads this at reconcile time — copies stay in sync with the source.
	FromConfigMap string `yaml:"fromConfigMap" validate:"omitempty"`

	// FromNamespace — namespace where FromConfigMap lives.
	// Default: same namespace as the CR.
	FromNamespace string `yaml:"fromNamespace" validate:"omitempty"`

	// Data — static key-value entries.
	// When FromConfigMap is also set, these entries override matching keys from the source.
	Data map[string]string `yaml:"data" validate:"omitempty"`

	// Labels — applied to all created ConfigMap copies.
	Labels []orktypes.ResourceLabel `yaml:"labels" validate:"omitempty"`

	// Reconcile: true — sync on every reconcile.
	// When true, if the source ConfigMap changes, all copies are updated automatically.
	Reconcile bool `yaml:"reconcile" validate:"omitempty"`
}
