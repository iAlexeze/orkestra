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
//	  version: v16
//	inputs:
//	  - name: image
//	    required: true
//	    description: PostgreSQL image (e.g. postgres:16)
//	  - name: volumeSize
//	    default: "10Gi"
//	resources:
//	  statefulsets:
//	    - name: "{{ .metadata.name }}-postgres"
//	      image: "{{ inputs.image }}"
type Motif struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   MotifMeta      `yaml:"metadata"`
	Inputs     []MotifInput   `yaml:"inputs,omitempty"`
	Resources  *HookTemplates `yaml:"resources,omitempty"`
	Status     *StatusConfig  `yaml:"status,omitempty"`
}

// MotifMeta holds Motif identity fields.
type MotifMeta struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version,omitempty"`
	Description string `yaml:"description,omitempty"`
	Author      string `yaml:"author,omitempty"`
	License     string `yaml:"license,omitempty"`
}

// MotifInput declares one input parameter for a Motif.
type MotifInput struct {
	// Name is the input identifier referenced in templates as inputs.Name.
	Name string `yaml:"name"`

	// Description explains what this input controls.
	Description string `yaml:"description,omitempty"`

	// Required — when true, the importing Katalog must provide this input
	// in its with: block. Validation fails if required inputs are missing.
	Required bool `yaml:"required,omitempty"`

	// Default is the value used when the input is not provided in with:.
	// Only valid when Required is false.
	Default string `yaml:"default,omitempty"`
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
	Motif string `yaml:"motif"`

	// Version — explicit version (tag, branch, or SHA).
	// Ignored when @ shorthand is used in Motif.
	// Defaults to "latest" for OCI, "main" for Git.
	Version string `yaml:"version,omitempty"`

	// OCI — when true, pull the Motif artifact via OCI/ORAS protocol.
	// When false (default), pull via Git (GitHub raw URL, GitLab, or git clone).
	OCI bool `yaml:"oci,omitempty"`

	// Auth — optional credentials for the registry.
	// Same auth model as RegistrySource.Auth — resolved from environment variables.
	Auth *FileSourceAuth `yaml:"auth,omitempty"`

	// With binds the Motif's declared inputs to values.
	// Values are template expressions evaluated in the CRD's reconcile context.
	// Required inputs not provided here are a validation error.
	// Optional inputs not provided use their Motif-declared defaults.
	With map[string]string `yaml:"with,omitempty"`
}
