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
//	        - kind: database            # target CRD name (lowercase)
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

// CrossCRDDeclaration declares one cross-CRD observation.
// Declared in the operatorBox config under cross: for a CRD.
type CrossCRDDeclaration struct {
	// Kind is the target CRD name (lowercase, matches the map key in spec.crds).
	//   kind: database
	Kind string `yaml:"kind"`

	// Selector identifies which CR instance to observe.
	Selector CrossSelector `yaml:"selector"`

	// As is the key under .cross.* where the result is accessible.
	//   as: database → .cross.database.status.phase
	// Default: same as Kind.
	As string `yaml:"as,omitempty"`

	// Strategy controls what happens when multiple CRs match the selector.
	//   first (default) — use the first match
	//   all             — put all matches in .cross.<as>[] (array)
	Strategy string `yaml:"strategy,omitempty"`

	// Source is the fallback for cross-binary or cross-cluster observation.
	// When absent, the informer cache is used (zero API calls).
	// When set, the endpoint is called when the informer is unavailable.
	Source *CrossSource `yaml:"source,omitempty"`
}

// CrossSelector identifies a CR in the target CRD.
type CrossSelector struct {
	// Name is the CR name to look up. Template expressions supported.
	//   name: "{{ .metadata.name }}"    → same name as the current CR
	Name string `yaml:"name,omitempty"`

	// Namespace is the CR namespace. Template expressions supported.
	// Default: same namespace as the current CR.
	Namespace string `yaml:"namespace,omitempty"`

	// LabelSelector selects CRs by label — for 1:N or N:1 relationships.
	//   labelSelector: "tenant={{ .spec.tenant }},env={{ .spec.environment }}"
	// When set, Name and Namespace are ignored.
	LabelSelector string `yaml:"labelSelector,omitempty"`
}

// CrossSource declares an HTTP fallback for cross-binary/cluster observation.
// The endpoint must return the same JSON shape as the informer cache path —
// i.e., the Orkestra CR detail endpoint format.
type CrossSource struct {
	// Endpoint is the URL to call when the informer is unavailable.
	// Template expressions supported.
	//   endpoint: "http://database-operator:8080/katalog/database/cr/{{ .metadata.namespace }}/{{ .metadata.name }}"
	Endpoint string `yaml:"endpoint"`

	// Token is a bearer token for the endpoint. $ENV_VAR syntax supported.
	Token string `yaml:"token,omitempty"`

	// CacheFor is how long to cache the result before calling again.
	// Default: 30s — prevents hammering the endpoint on every resync.
	CacheFor string `yaml:"cacheFor,omitempty"`
}
