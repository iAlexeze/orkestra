// pkg/types/conversion.go
package types

// CRDConversion declares declarative version conversion rules for a CRD.
// When enabled, Orkestra serves the /convert endpoint and translates
// ConversionReview requests using these rules and the template resolver.
//
// Example Katalog declaration:
//
//	conversion:
//	  storageVersion: v1
//	  paths:
//	    - from: v1alpha1
//	      to: v1
//	      spec:
//	        image: "{{ .spec.image }}"
//	        replicas: "{{ .spec.replicas }}"
//	        seo:
//	          enabled: false   # default — v1alpha1 has no SEO field
//	    - from: v1
//	      to: v1alpha1
//	      spec:
//	        image: "{{ .spec.image }}"
//	        replicas: "{{ .spec.replicas }}"
//	        theme: "default"   # default — v1 has no theme field
type CRDConversion struct {
	// Tenants represents the list of crds involved in conversion
	// This is used to compute data for metrics, and control center
	Tenants []string `yaml:"tenants"`

	// StorageVersion — the version all objects are stored as internally.
	// All conversion paths route through this version.
	StorageVersion string `yaml:"storageVersion"`

	// Paths — one entry per (from, to) pair.
	// You need at least two paths for a two-version CRD:
	//   - old → storage  (up-conversion)
	//   - storage → old  (down-conversion)
	Paths []ConversionPath `yaml:"paths"`

	// UpdateCRD — when true, Orkestra patches the CRD's
	// spec.conversion.webhook.clientConfig.caBundle with the CA bundle
	// from the generated (or configured) TLS certificate at startup.
	// Set this to true when you let Orkestra manage TLS; set it to false
	// when you manage caBundle injection yourself (e.g. cert-manager).
	// Default: false.
	UpdateCRD bool `yaml:"updateCRD,omitempty"`
}

// ConversionPath declares one explicit conversion mapping.
// Both From and To are bare version strings — not full apiVersion strings.
//
//	from: v1alpha1   ← bare version, not "demo.orkestra.io/v1alpha1"
//	to: v1
//	spec:
//	  image: "{{ .spec.image }}"
//	  seo:
//	    enabled: false
type ConversionPath struct {
	// From — the source version (bare, e.g. "v1alpha1")
	From string `yaml:"from" validate:"required"`

	// To — the target version (bare, e.g. "v1")
	To string `yaml:"to" validate:"required"`

	// Spec — the output spec in the target version's format.
	// Values support Go template expressions evaluated against the source object.
	// Static values are used as-is.
	Spec map[string]interface{} `yaml:"spec" validate:"required"`
}

// ConversionRules is the runtime form of CRDConversion, keyed by Kind.
// Registered in the InMemoryConversionRegistry at Katalog load time.
type ConversionRules struct {
	Kind           string           `json:"kind"`
	StorageVersion string           `json:"storageVersion"`
	Paths          []ConversionPath `json:"paths"`
}

// FindPath returns the conversion path for a given (from, to) pair.
// Both fromVersion and toVersion must be bare version strings.
// Returns nil when no path matches.
func (r *ConversionRules) FindPath(fromVersion, toVersion string) *ConversionPath {
	for i := range r.Paths {
		p := &r.Paths[i]
		if p.From == fromVersion && p.To == toVersion {
			return p
		}
	}
	return nil
}
