// pkg/types/provider_katalog.go
//
// Provider types for Katalog YAML parsing and runtime dispatch.
//
// Two layers:
//
// Layer 1 — Catalog-level manifest (KatalogProviderRequirement)
//
//	Declared at the top of the Katalog under spec.providers.
//	Lists which provider libraries this Katalog requires.
//	Used by `ork validate` to warn when a required provider is not registered.
//	Used by `ork provider install` to pull provider OCI artifacts.
//
//	spec:
//	  providers:
//	    - name: aws
//	      required: true
//	    - name: mongodb
//	      required: false
//
// Layer 2 — Per-CRD provider blocks (ProviderBlock, RawProviderDeclaration)
//
//	Declared under spec.crds[].reconciler.providers.
//	Each named block is dispatched to the registered provider library.
//	Template expressions are resolved before the provider is called.
//
//	reconciler:
//	  providers:
//	    aws:
//	      - s3:
//	          bucket: "{{ .metadata.name }}-assets"
//	          region: "{{ .spec.region }}"
//	    mongodb:
//	      - database:
//	          name: "{{ .metadata.name }}"
//	      - user:
//	          name: "{{ .spec.dbUser }}"
//	          database: "{{ .metadata.name }}"
//	          credentials:
//	            secretName: "{{ .metadata.name }}-mongo-creds"
package types

// ─────────────────────────────────────────────────────────────────────────────
// Layer 1 — Catalog-level manifest
// ─────────────────────────────────────────────────────────────────────────────

// KatalogProviderRequirement declares that this Katalog uses a named provider.
// Declared at spec.providers[] in the Katalog YAML.
//
// Purpose:
//   - `ork validate` warns when a required provider is not registered at runtime
//   - `ork provider install` pulls the OCI artifact for this provider
//   - Documentation: makes explicit what external systems this Katalog touches
type KatalogProviderRequirement struct {
	// Name is the YAML block key used under reconciler.providers.
	// Must match the Name() return value of the registered Provider.
	// e.g. "aws", "mongodb", "stripe"
	Name string `yaml:"name"`

	// Required controls whether `ork validate` hard-fails on missing registration.
	// true  → validation error if not registered (operator will not function correctly)
	// false → validation warning only (provider blocks are skipped at runtime)
	Required bool `yaml:"required"`

	// Version is the expected provider library version.
	// Used by `ork provider install` to pull the correct OCI artifact.
	// Optional — if absent, the latest version is used.
	Version string `yaml:"version,omitempty"`

	// Library is the OCI artifact reference for this provider.
	// e.g. "oci://registry.orkestra.io/providers/aws:1.8.0"
	// Used by `ork provider install`. Optional for locally registered providers.
	Library string `yaml:"library,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Layer 2 — Per-CRD provider blocks
// ─────────────────────────────────────────────────────────────────────────────

// ProviderBlock is one named provider section from the reconciler.providers map.
// The Name comes from the map key ("aws", "mongodb").
// Declarations come from the list under that key.
//
// This is the parsed, structured form used at runtime.
// The raw YAML map is parsed into this during Katalog loading.
type ProviderBlock struct {
	// Name is the provider block key — matches Provider.Name().
	Name string

	// Declarations is the ordered list of resource declarations in this block.
	// Each declaration specifies one external resource to manage.
	Declarations []RawProviderDeclaration
}

// RawProviderDeclaration is one item in a provider block list.
// At parse time, template expressions are NOT yet resolved — they contain
// raw strings like "{{ .metadata.name }}" which are resolved at reconcile time.
//
// YAML shape (one list item under aws: or mongodb:):
//
//   - s3:
//     bucket: "{{ .metadata.name }}-assets"
//     region: "{{ .spec.region }}"
//     versioning: "true"
//     when:
//   - field: spec.enableStorage
//     equals: "true"
//
// Parses into:
//
//	RawProviderDeclaration{
//	    Kind: "s3",
//	    Fields: map[string]string{
//	        "bucket":     "{{ .metadata.name }}-assets",
//	        "region":     "{{ .spec.region }}",
//	        "versioning": "true",
//	    },
//	    Conditions: []Condition{{Field: "spec.enableStorage", Equals: "true"}},
//	}
type RawProviderDeclaration struct {
	// Kind is the resource type within the provider block.
	// Derived from the single key in the YAML map entry.
	// Examples: "s3", "rds", "route53", "database", "user", "collection"
	Kind string

	// Fields are the raw (unresolved) key-value pairs for this declaration.
	// Nested YAML maps are flattened with dot notation:
	//   credentials:
	//     secretName: my-secret
	// becomes Fields["credentials.secretName"] = "my-secret"
	//
	// Values may contain template expressions — resolved by the template
	// resolver before the provider is called.
	Fields map[string]string

	// Conditions are the when: conditions from this declaration.
	// Evaluated by Orkestra before calling the provider — declarations
	// whose conditions fail are removed from the list before dispatch.
	Conditions []Condition `yaml:"when,omitempty"`
}
