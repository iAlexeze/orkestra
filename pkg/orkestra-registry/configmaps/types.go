package configmaps

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
type ConfigMapTemplateSource struct {
	// Version — OrkestraRegistry implementation version. Omit for latest.
	Version string

	// Name — ConfigMap name.
	// Default: "{{ .metadata.name }}-config"
	Name string

	// Namespace — primary target namespace.
	// Default: "{{ .metadata.namespace }}"
	Namespace string

	// ToNamespaces — create one copy in each listed namespace.
	// Each element supports template expressions.
	ToNamespaces []string

	// FromConfigMap — name of an existing ConfigMap to copy data from.
	// Orkestra reads this at reconcile time — copies stay in sync with the source.
	FromConfigMap string

	// FromNamespace — namespace where FromConfigMap lives.
	// Default: same namespace as the CR.
	FromNamespace string

	// Data — static key-value entries.
	// When FromConfigMap is also set, these entries override matching keys from the source.
	Data map[string]string

	// Labels — applied to all created ConfigMap copies.
	Labels []orktypes.ResourceLabel

	// Reconcile: true — sync on every reconcile.
	// When true, if the source ConfigMap changes, all copies are updated automatically.
	Reconcile bool

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string
}
