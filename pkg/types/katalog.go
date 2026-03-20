// pkg/types/katalog.go
package orktypes

// KatalogFile is the top-level structure of a crd-katalog.yaml file.
// It contains optional sources (files and helm charts) plus inline CRDs.
// The merger resolves all sources and merges everything into one KatalogSpec.
type KatalogFile struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   KatalogMeta     `yaml:"metadata"`
	Sources    *KatalogSources `yaml:"sources,omitempty"`
	Spec       KatalogSpec     `yaml:"spec"`
}

// KatalogMeta holds identifying metadata for the Katalog.
type KatalogMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// KatalogSources declares additional CRD definitions to merge in.
// Sources are loaded before spec.crds — inline CRDs are merged last
// and win on name conflict (allowing local overrides of remote definitions).
type KatalogSources struct {
	// Files — local file paths or remote URLs to other Katalog files.
	// Each entry must be a valid Katalog YAML (apiVersion, kind, spec.crds).
	// Supports environment variable references: $MY_KATALOG_URL
	Files []string `yaml:"files,omitempty"`

	// Helm — Helm chart sources. Each chart is rendered with the provided
	// value files and the resulting Katalog templates are extracted and merged.
	Helm []HelmSource `yaml:"helm,omitempty"`
}

// HelmSource declares a Helm chart that produces Katalog CRD definitions.
// The chart must render at least one template with kind: Katalog.
//
// Example chart template (templates/katalog.yaml):
//
//	apiVersion: orkestra.konductor.io/v1Alpha
//	kind: Katalog
//	spec:
//	  crds:
//	    {{- range .Values.crds }}
//	    - name: {{ .name }}
//	      enabled: {{ .enabled }}
//	      ...
//	    {{- end }}
type HelmSource struct {
	// Repo — Helm repository URL.
	// e.g. "https://charts.myorg.io"
	Repo string `yaml:"repo" validate:"required"`

	// Chart — chart name within the repository.
	// e.g. "platform-crds"
	Chart string `yaml:"chart" validate:"required"`

	// Version — chart version to use. Required for reproducibility.
	// And also used as git ref
	// e.g. "1.2.0"
	Version string `yaml:"version" validate:"required"`

	// Path — chart path within git repo
	Path string `yaml:"path"       validate:"omitempty"`

	// ValueFiles — list of values files to apply when rendering the chart.
	// Each entry can be a local path or a remote URL.
	// Supports environment variable references: $MY_VALUES_FILE
	// Applied in order — later files override earlier ones (same as helm -f).
	ValueFiles []string `yaml:"valueFiles,omitempty"`

	// Values — inline key-value pairs applied after valueFiles.
	// Same as helm --set key=value.
	Values map[string]interface{} `yaml:"values,omitempty"`
}

// KatalogSpec holds the actual CRD definitions.
// This is what the merger produces after resolving all sources.
type KatalogSpec struct {
	// Finalizers — Katalog-level finalizers applied to all CRDs
	// unless overridden at the CRD level.
	Finalizers []string `yaml:"finalizers,omitempty"`

	// CRDs — the CRD entries managed by this Orkestra instance.
	CRDs []CRDEntry `yaml:"crds"`
}
