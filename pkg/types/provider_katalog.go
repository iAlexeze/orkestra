// pkg/types/provider_katalog.go
//
// Provider types for Katalog YAML parsing and runtime dispatch.
//
// # Layer 1 — Katalog-level manifest (KatalogProviderRequirement)
//
// Declared at the top level under providers[]. Lists which providers this Katalog uses
// and supplies the credentials for each. Credentials support $ENV_VAR expansion.
// Only providers registered here are active — per-CRD blocks for unregistered
// providers are skipped with a warning.
//
// providers: is a top-level sibling of spec: and security:, not nested under spec:,
// because providers represent operational infrastructure dependencies — distinct state
// from the CRD definitions in spec:.
//
//	providers:
//	  - name: aws
//	    required: true
//	    auth:
//	      accessKeyId: "$AWS_ACCESS_KEY_ID"
//	      secretAccessKey: "$AWS_SECRET_ACCESS_KEY"
//	      region: "$AWS_REGION"
//	  - name: mongodb
//	    required: true
//	    auth:
//	      mongoUri: "$MONGODB_URL"
//
// # Layer 2 — Per-CRD provider blocks (ProviderBlock, RawProviderDeclaration)
//
// Declared under spec.crds[].operatorBox.providers.
// Each named block is dispatched to the registered provider library.
// Template expressions are resolved before the provider is called.
// A per-CRD auth block can override the katalog-level credentials for that CRD.
//
//	operatorBox:
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

import "os"

// ─────────────────────────────────────────────────────────────────────────────
// Layer 1 — Katalog-level manifest
// ─────────────────────────────────────────────────────────────────────────────

// KatalogProviderRequirement declares that this Katalog uses a named provider.
// Declared at the top-level providers[] in the Katalog YAML.
//
// Purpose:
//   - `ork validate` warns when a required provider is not registered at runtime
//   - Supplies credentials (with $ENV_VAR expansion) to the provider at startup
//   - `ork provider install` pulls the OCI artifact for this provider
//   - Documentation: makes explicit what external systems this Katalog touches
//
// Only providers declared here are registered. Per-CRD provider blocks for
// undeclared providers are silently skipped with a warning log.
type KatalogProviderRequirement struct {
	// Name is the YAML block key used under operatorBox.providers.
	// Must match the Name() return value of the registered Provider.
	// e.g. "aws", "mongodb", "stripe"
	Name string `yaml:"name" json:"name"`

	// Required controls whether `ork validate` hard-fails on missing registration.
	// true  → validation error if not registered (operator will not function correctly)
	// false → validation warning only (provider blocks are skipped at runtime)
	Required bool `yaml:"required" json:"required"`

	// Auth holds the provider credentials. Values support $ENV_VAR expansion —
	// use "$MY_SECRET" and Orkestra will substitute os.Getenv("MY_SECRET") at startup.
	//
	// AWS example:
	//   auth:
	//     accessKeyId: "$AWS_ACCESS_KEY_ID"
	//     secretAccessKey: "$AWS_SECRET_ACCESS_KEY"
	//     region: "us-east-1"
	//
	// MongoDB example:
	//   auth:
	//     mongoUri: "$MONGODB_URL"
	Auth map[string]string `yaml:"auth,omitempty" json:"auth,omitempty"`

	// Version is the expected provider library version.
	// Used by `ork provider install` to pull the correct OCI artifact.
	// Optional — if absent, the latest version is used.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Library is the OCI artifact reference for this provider.
	// e.g. "oci://registry.orkestra.io/providers/aws:1.8.0"
	// Used by `ork provider install`. Optional for locally registered providers.
	//
	// Future
	Library string `yaml:"library,omitempty" json:"library,omitempty"`
}

// ResolvedAuth returns a copy of Auth with all $ENV_VAR values substituted.
// Values that do not start with "$" are returned unchanged.
// Called at startup before credentials are passed to the provider constructor.
func (r KatalogProviderRequirement) ResolvedAuth() map[string]string {
	if len(r.Auth) == 0 {
		return nil
	}
	out := make(map[string]string, len(r.Auth))
	for k, v := range r.Auth {
		if len(v) > 1 && v[0] == '$' {
			out[k] = os.Getenv(v[1:])
		} else {
			out[k] = v
		}
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// Layer 2 — Per-CRD provider blocks
// ─────────────────────────────────────────────────────────────────────────────

// ProviderBlock is one named provider section from the operatorBox.providers map.
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

	// Conditions are the when: conditions from this declaration (AND semantics).
	// Evaluated by Orkestra before calling the provider — declarations
	// whose conditions fail are removed from the list before dispatch.
	Conditions []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// Or holds OR conditions — at least one must pass.
	Or []Condition `yaml:"or,omitempty" json:"or,omitempty"`
}
