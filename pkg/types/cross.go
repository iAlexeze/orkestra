// pkg/types/cross.go
//
// Cross-CRD observation declarations.
//
// The cross: block under a CRD's reconciler allows one CR to observe
// another CRD's CR instances without making API server calls — data is
// read from the target CRD's informer cache directly.
//
// YAML:
//
//	spec:
//	  crds:
//	    application:
//	      cross:
//	        - crd: database             # target CRD name (lowercase, matches spec.crds key)
//	          selector:
//	            name: "{{ .metadata.name }}"       # find by name
//	            namespace: "{{ .metadata.namespace }}"
//	          as: database              # available as .cross.database.*
//
//	    database:
//	      operatorBox: ...               # database CRD — already has an informer
//
// After ReadCross runs, the Application reconciler has access to:
//
//	{{ .cross.database.found }}                → "true" if CR was found
//	{{ .cross.database.status.phase }}         → Database CR's status.phase
//	{{ .cross.database.status.endpoint }}      → Database CR's status.endpoint
//	{{ .cross.database.spec.storageGb }}       → Database CR's spec field
//
// This is zero API calls for same-binary CRDs — data comes from the
// informer cache that is already maintained for the target CRD.
// For cross-binary or cross-cluster observation, set source.endpoint.
//
// Use cases:
//   - Wait for a Database CR to be Ready before creating the Application Deployment
//   - Copy a Database endpoint into the Application status
//   - Gate resource creation on another CRD's phase
//   - Service mesh without a service mesh: Application watches Database watches Network
package types

// ── Cross declaration type ────────────────────────────────────────────────────
// The cross: YAML block should use explicit fields rather than overloading
// the crd: string with "key=value" syntax.
//
// YAML:
//
//   cross:
//     - crd: managed-database          # name-based (existing path)
//       selector:
//         name: "{{ .metadata.name }}-db"
//       as: db
//
//     - labelSelector:                  # label-based (new path)
//         tier: platform
//       selector:
//         name: "{{ .metadata.name }}-platform"
//       as: platform
//
// The reconciler checks CrossSelector.LabelSelector first. If non-empty,
// it calls ReadCrossFromInformerByLabel. Otherwise it uses the crd name
// to look up the informer and calls ReadCrossFromInformer with the
// resolved namespace/name key.

// CrossCRDDeclaration declares one cross-CRD observation.
// Declared in the operatorBox config under cross: for a CRD.
type CrossCRDDeclaration struct {
	// Crd is the target CRD name (lowercase, matches the map key in spec.crds).
	//   crd: database
	Crd string `yaml:"crd" json:"crd"`

	// LabelSelector is a label key/value pair for label-based informer lookup.
	// Mutually exclusive with Kind.
	LabelSelector map[string]string `yaml:"labelSelector,omitempty" json:"labelSelector,omitempty"`

	// Selector identifies which CR instance to observe.
	Selector CrossSelector `yaml:"selector" json:"selector"`

	// As is the key under .cross.* where the result is accessible.
	//   as: database → .cross.database.status.phase
	// Default: same as Crd.
	As string `yaml:"as,omitempty" json:"as,omitempty"`

	// Strategy controls what happens when multiple CRs match the selector.
	//   first (default) — use the first match
	//   all             — put all matches in .cross.<as>[] (array)
	Strategy string `yaml:"strategy,omitempty" json:"strategy,omitempty"`

	// Source is the fallback for cross-binary or cross-cluster observation.
	// When absent, the informer cache is used (zero API calls).
	// When set, the endpoint is called when the informer is unavailable.
	Source *CrossSource `yaml:"source,omitempty" json:"source,omitempty"`
}

// CrossSelector identifies a CR in the target CRD.
// The selector block on a cross: declaration.
// Exactly one of Name or LabelSelector should be set per entry.
type CrossSelector struct {
	// Name is the CR name to look up. Template expressions supported.
	//   name: "{{ .metadata.name }}"    → same name as the current CR
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Namespace is the CR namespace. Template expressions supported.
	// Default: same namespace as the current CR.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// LabelSelector selects CRs by label — for 1:N or N:1 relationships.
	//   labelSelector: "tenant={{ .spec.tenant }},env={{ .spec.environment }}"
	// When set, Name and Namespace are ignored.
	LabelSelector string `yaml:"labelSelector,omitempty" json:"labelSelector,omitempty"`
}

// CrossSource declares an HTTP fallback for cross-binary/cluster observation.
// The endpoint must return the same JSON shape as the informer cache path —
// i.e., the Orkestra CR detail endpoint format.
type CrossSource struct {
	// Endpoint is the URL to call when the informer is unavailable.
	// Template expressions supported.
	//   endpoint: "http://database-operator:8080/katalog/database/cr/{{ .metadata.namespace }}/{{ .metadata.name }}"
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// Token is a bearer token for the endpoint. $ENV_VAR syntax supported.
	Token string `yaml:"token,omitempty" json:"token,omitempty"`

	// CacheFor is how long to cache the result before calling again.
	// Default: 30s — prevents hammering the endpoint on every resync.
	CacheFor string `yaml:"cacheFor,omitempty" json:"cacheFor,omitempty"`
}
