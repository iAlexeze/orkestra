// pkg/types/motif.go
package types

// Motif is the smallest reusable primitive in Orkestra's composition model.
// A Motif declares named inputs and resource blocks. It cannot run alone —
// it must be imported by a Katalog that provides its inputs via with:.
//
// YAML:
//
//	apiVersion: orkestra.orkspace.io/v1
//	kind: Motif
//	metadata:
//	  name: postgres
//	inputs:
//	  - name: image
//	  - name: volumeSize
//	resources: ...
//	status: ...
//	admission:
//	  validation:
//	    rules:
//	      - field: spec.image
//	        prefix: "myregistry.com/"
//	        action: deny
//	  mutation:
//	    rules:
//	      - field: spec.replicas
//	        default: "2"
type Motif struct {
	APIVersion string          `yaml:"apiVersion" json:"apiVersion"`
	Kind       string          `yaml:"kind" json:"kind"`
	Metadata   MotifMeta       `yaml:"metadata" json:"metadata"`
	Inputs     []MotifInput    `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Profiles   ProfileRegistry `yaml:"profiles,omitempty" json:"profiles,omitempty"`
	Resources  *MotifResources `yaml:"resources,omitempty" json:"resources,omitempty"`
	Status     *StatusConfig   `yaml:"status,omitempty" json:"status,omitempty"`
	Admission  *Admission      `yaml:"admission,omitempty" json:"admission,omitempty"`
}

// MotifResources groups the resources a Motif contributes to a CRD entry.
// Resources declared directly under resources: are merged into onReconcile.
// Resources declared under resources.onCreate: are merged into onCreate,
// making them immune to the update=true path (correct for once: true secrets).
type MotifResources struct {
	// OnCreate groups resources that must only be processed during creation —
	// never updated on subsequent reconciles. Secrets with once: true belong here.
	OnCreate *HookTemplates `yaml:"onCreate,omitempty" json:"onCreate,omitempty"`

	// All remaining HookTemplates fields are promoted to the resources: level
	// and merged into the CRD's onReconcile phase.
	HookTemplates `yaml:",inline"`
}

// MotifMeta holds Motif identity fields.
type MotifMeta struct {
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version,omitempty" json:"version,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Author      string   `yaml:"author,omitempty" json:"author,omitempty"`
	License     string   `yaml:"license,omitempty" json:"license,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// MotifInput declares one input parameter for a Motif.
type MotifInput struct {
	// Name is the input identifier referenced in templates as inputs.Name.
	Name string `yaml:"name" json:"name"`

	// Description explains what this input controls.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Required — when true, the importing Katalog must provide this input
	// in its with: block. Validation fails if required inputs are missing.
	Required bool `yaml:"required,omitempty" json:"required,omitempty"`

	// Type hints at the expected type of the input value. Not currently enforced,
	// but reserved for future type checking or schema generation.
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Default is the value used when the input is not provided in with:.
	// Only valid when Required is false.
	Default string `yaml:"default,omitempty" json:"default,omitempty"`
}

// MotifImport declares one Motif import inside an operatorBox.
// Follows the same resolution semantics as RegistrySource in a Komposer —
// if you know how to pull a pattern, you already know how to pull a Motif.
//
// YAML inside operatorBox:
//
// File (developer path):
//
//	imports:
//	  - motif: ./motifs/postgres/motif.yaml
//	    with:
//	      image: "postgres:16"
//
// OCI registry (the Orkestra registry houses both patterns and motifs):
//
//	imports:
//	  - motif: ghcr.io/orkspace/orkestra-registry/postgres@v16
//	    oci: true
//	    with:
//	      image: "{{ .spec.postgresImage }}"
//
// Git registry:
//
//	imports:
//	  - motif: https://github.com/myorg/postgres-motif@main
//	    with:
//	      image: "{{ .spec.postgresImage }}"
type MotifImport struct {
	// Motif is the registry URL, file path, or short name.
	// Same formats as RegistrySource.URL:
	//   File:  ./postgres/motif.yaml
	//   OCI:   ghcr.io/orkspace/orkestra-registry/postgres@v16
	//   Git:   https://github.com/myorg/postgres-motif@main
	// @ shorthand encodes the version inline: url@version
	Motif string `yaml:"motif" json:"motif,omitempty"`

	// Version — explicit version (tag, branch, or SHA).
	// Ignored when @ shorthand is used in Motif.
	// Defaults to "latest" for OCI, "main" for Git.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// OCI — when true, pull the Motif artifact via OCI/ORAS protocol.
	// When false (default), pull via Git (GitHub raw URL, GitLab, or git clone).
	OCI bool `yaml:"oci,omitempty" json:"oci,omitempty"`

	// Auth — optional credentials for the registry.
	// Same auth model as RegistrySource.Auth — resolved from environment variables.
	Auth *FileSourceAuth `yaml:"auth,omitempty" json:"auth,omitempty"`

	// With binds the Motif's declared inputs to values.
	// Values are template expressions evaluated in the CRD's reconcile context.
	// Required inputs not provided here are a validation error.
	// Optional inputs not provided use their Motif-declared defaults.
	With map[string]string `yaml:"with,omitempty" json:"with,omitempty"`
}
