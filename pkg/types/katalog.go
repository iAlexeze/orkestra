// pkg/types/katalog.go
package types

// KatalogFile is the top-level structure of a crd-katalog.yaml file.
// It contains optional sources (files and helm charts) plus inline CRDs.
// Orkestra's in-built merger resolves all sources and merges everything into one KatalogSpec.
type KatalogFile struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   KatalogMeta     `yaml:"metadata"`
	Imports    *KatalogSources `yaml:"imports,omitempty"`
	Spec       KatalogSpec     `yaml:"spec"`
	Security   KatalogSecurity `yaml:"security"`

	// Notification holds the top-level alerting configuration for this Katalog.
	// Defines channels (email, Slack) and per-team routing rules that fire when
	// a managed CRD's conditions transition. When a Komposer references multiple
	// source Katalogs, notification blocks are merged — source teams are inherited
	// and the Komposer's own teams win on name conflict.
	Notification *KatalogNotification `yaml:"notification,omitempty"`

	// Providers declares which external provider libraries this Katalog requires.
	// Top-level alongside spec: and security: — providers represent a distinct
	// operational concern (infrastructure dependencies) separate from CRD definitions.
	//
	//   providers:
	//     - name: aws
	//       required: true
	//       auth:
	//         accessKeyId: "$AWS_ACCESS_KEY_ID"
	//         secretAccessKey: "$AWS_SECRET_ACCESS_KEY"
	//         region: "$AWS_REGION"
	//     - name: mongodb
	//       required: true
	//       auth:
	//         mongoUri: "$MONGODB_URL"
	Providers []KatalogProviderRequirement `yaml:"providers,omitempty"`
}

// Language represents the primary language detected in the project.
type Language string

const (
	LangGo      Language = "Go"
	LangNode    Language = "Node.js"
	LangJava    Language = "Java"
	LangPython  Language = "Python"
	LangRuby    Language = "Ruby"
	LangRust    Language = "Rust"
	LangUnknown Language = "Unknown"
)

// DotEnvVar is a single variable parsed from a .env file.
type DotEnvVar struct {
	Key   string
	Value string
	IsCfg bool // true when line carries "# ork:cfg"
}

// ProjectInfo captures everything ork doctor discovers about a project
// directory. It is used directly in a Katalog (single project) and reused
// inside a Komposer (multiple projects). Only ork doctor populates this.
type ProjectInfo struct {
	// Name is the project name as written into the katalog.
	// Derived from AppName.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// AppName is the derived application name from the directory.
	AppName string `yaml:"appName,omitempty" json:"appName,omitempty"`

	// Dir is the absolute path to the project directory.
	Dir string `yaml:"dir,omitempty" json:"dir,omitempty"`

	// HasDockerfile indicates whether a Dockerfile exists in the project.
	HasDockerfile bool `yaml:"hasDockerfile,omitempty" json:"hasDockerfile,omitempty"`

	// DockerfilePath is the full path to the Dockerfile if present.
	DockerfilePath string `yaml:"dockerfilePath,omitempty" json:"dockerfilePath,omitempty"`

	// GitCommit is the short SHA of the current git commit, empty if not a repo.
	GitCommit string `yaml:"gitCommit,omitempty" json:"gitCommit,omitempty"`

	// Language is the detected project language (Python, Go, Node, etc.).
	Language Language `yaml:"language,omitempty" json:"language,omitempty"`

	// LangMarker is the file that triggered language detection (e.g. requirements.txt).
	LangMarker string `yaml:"langMarker,omitempty" json:"langMarker,omitempty"`

	// Port is the detected application port (from .env or language defaults).
	Port string `yaml:"port,omitempty" json:"port,omitempty"`

	// EnvVars contains all parsed .env variables.
	EnvVars []DotEnvVar `yaml:"-" json:"-"`

	// Secrets contains all env vars classified as secrets (IsCfg == false).
	Secrets []DotEnvVar `yaml:"-" json:"-"`

	// Config contains all env vars classified as config (IsCfg == true).
	Config []DotEnvVar `yaml:"-" json:"-"`

	// HasFrontend indicates whether a frontend was detected in the project.
	HasFrontend bool `yaml:"hasFrontend,omitempty" json:"hasFrontend,omitempty"`

	// HasSMTP is true if SMTP‑related variables were found in .env.
	HasSMTP bool `yaml:"hasSMTP,omitempty" json:"hasSMTP,omitempty"`

	// HasSlack is true if Slack webhook variables were found in .env.
	HasSlack bool `yaml:"hasSlack,omitempty" json:"hasSlack,omitempty"`

	// License is the detected project license from standard license files.
	License string `yaml:"license,omitempty" json:"license,omitempty"`

	// HasCompose indicates whether a docker-compose.yaml file exists.
	HasCompose bool `yaml:"hasCompose,omitempty" json:"hasCompose,omitempty"`

	// UseCompose indicates whether the user chose to use the compose file.
	UseCompose bool `yaml:"useCompose,omitempty" json:"useCompose,omitempty"`

	// ComposePath is the path to the docker-compose.yaml file.
	ComposePath string `yaml:"composePath,omitempty" json:"composePath,omitempty"`

	// SecretCount is the number of secret variables (derived).
	SecretCount int `yaml:"secretCount,omitempty" json:"secretCount,omitempty"`

	// ConfigCount is the number of config variables (derived).
	ConfigCount int `yaml:"configCount,omitempty" json:"configCount,omitempty"`

	// Namespace is the Kubernetes namespace this project is deployed into.
	// Written by ork doctor at deploy time so the Control Center can
	// display the internal service URL without an additional API call.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// CurrentImage is the fully-qualified image reference that was last
	// deployed for this project (e.g. docker.io/myorg/app:abc123).
	// Written by ork doctor after a successful deploy.
	CurrentImage string `yaml:"currentImage,omitempty" json:"currentImage,omitempty"`
}

// HasSecrets reports whether ork doctor discovered any secret
// environment variables in the project (.env where IsCfg == false).
func (p *ProjectInfo) HasSecrets() bool {
	return len(p.Secrets) > 0
}

// HasConfig reports whether ork doctor discovered any config
// environment variables in the project (.env where IsCfg == true).
func (p *ProjectInfo) HasConfig() bool {
	return len(p.Config) > 0
}

// HasCreds reports whether the project contains *either* secrets or config.
// Useful for high‑level checks (e.g., does this project need a ConfigMap or Secret?).
func (p *ProjectInfo) HasCreds() bool {
	return p.HasSecrets() || p.HasConfig()
}

// KatalogMeta holds identifying metadata for a Katalog.
type KatalogMeta struct {
	// Name is the required unique identifier of the Katalog.
	Name string `yaml:"name" json:"name,omitempty"`

	// Description provides a human-readable explanation of the Katalog's purpose.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Version follows semantic versioning (e.g., "1.2.3") for the Katalog schema or content.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Author identifies the creator or maintainer of the Katalog.
	Author string `yaml:"author,omitempty" json:"author,omitempty"`

	// License describes the licensing terms under which the Katalog is distributed.
	License string `yaml:"license,omitempty" json:"license,omitempty"`

	// Tags are optional keywords for categorising the Katalog in the Orkestra Registry.
	// They aid discovery (e.g., "database", "stateful", "security") when using
	// `ork registry list --tag <tag>` and for indexing in Artifact Hub.
	// Tags have no effect on runtime behaviour.
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// CreatedBy indicates which client or command generated this Katalog metadata.
	// It influences the UI presented by the Control Center:
	//   - If empty or "operator" (default) → shows an operator‑focused UI with
	//     detailed infrastructure and workload controls.
	//   - If "orkdoctor" → indicates a developer context; the Control Center
	//     shows a simplified, developer‑oriented UI with only the terminology
	//     and actions relevant to application developers (hides low‑level operator details).
	// Other values may be introduced in the future for different workflows.
	CreatedBy string `yaml:"createdBy,omitempty" json:"createdBy,omitempty"`

	// ProjectInfo captures the full developer‑side understanding of an application
	// as discovered by ork doctor during project analysis. When a Katalog is
	// generated with `createdBy=orkdoctor`, this struct is embedded into the
	// Katalog so the Orkestra Control Center can present a rich, developer‑focused
	// experience without re‑scanning the source directory.
	//
	// This metadata is *never* populated by the operator or the runtime; it is
	// strictly authored by ork doctor at generation time. It represents the
	// developer’s local project context — language, ports, env vars, Dockerfile
	// presence, compose usage, frontend detection, license, and other signals that
	// only exist on the developer’s machine.
	//
	// By making ProjectInfo a katalog‑resident object, Orkestra gains a
	// persona‑aware UI model:
	//
	//   - For `createdBy=orkdoctor`, the Control Center can show developer‑centric
	//     insights (env breakdown, detected language, suggested Dockerfile,
	//     frontend hints, notification wiring, etc.) without needing access to the
	//     original project directory.
	//
	//   - For operator‑authored katalogs, this struct remains empty, and the UI
	//     defaults to the infrastructure‑centric operator view.
	//
	// This design allows Orkestra to evolve into a multi‑persona platform where
	// developer intent, project structure, and local context travel with the
	// Katalog — enabling richer automation, better defaults, and a more intuitive
	// experience across the entire lifecycle.
	// ProjectInfo ProjectInfo `yaml:"projectInfo,omitempty" json:"projectInfo,omitempty"`

	// Projects holds the aggregated ProjectInfo entries for every application
	// participating in this Komposer workspace. Each entry represents the
	// developer‑side metadata discovered by 'ork doctor' for a single project.
	//
	// Unlike a Katalog—which contains only the ProjectInfo for its own app—the
	// Komposer acts as the multi‑project orchestrator and therefore exposes a
	// map of project name → ProjectInfo. This enables the Orkestra Control Center
	// to present a unified, workspace‑level developer view (languages, ports,
	// Dockerfile presence, env breakdowns, frontend detection, etc.) without
	// needing access to the original source directories.
	//
	// The operator and runtime ignore this field entirely; it is purely developer
	// metadata used for persona‑aware UI and tooling.
	// For today, Only 'ork doctor' populates it.
	Projects map[string]ProjectInfo `yaml:"projects,omitempty" json:"projects,omitempty"`
}

// KatalogSources declares where to load CRD definitions from.
// Sources are loaded before spec.crds — inline CRDs are merged last
// and win on name conflict (allowing local overrides of remote definitions).
//
// Only valid on kind: Komposer documents.
type KatalogSources struct {
	// Files — local paths, remote URLs, or environment variable references.
	// Each entry must be a valid Katalog YAML (apiVersion, kind, spec.crds).
	// Supports environment variable references: $MY_KATALOG_URL
	//
	// Simple form: just a path string (no auth)
	//   files:
	//     - ./katalogs/project.yaml
	//     - https://public.url/katalog.yaml
	//     - $MY_KATALOG_URL
	//
	// Authenticated form: a FileSource struct with auth block
	//   files:
	//     - url: https://private.url/katalog.yaml
	//       auth:
	//         type: bearer
	//         fromEnv: MY_TOKEN
	Files []FileSource `yaml:"files,omitempty"`

	// Helm — Helm chart sources. Each chart is rendered with the provided
	// value files and the resulting Katalog templates are extracted and merged.
	Helm []HelmSource `yaml:"helm,omitempty"`

	// Registry - Registry sources.
	Registry []RegistrySource `yaml:"registry,omitempty"`
}

// HelmSource declares a Helm chart that produces Katalog CRD definitions.
// The chart must render at least one template with kind: Katalog.
//
// Example chart template (templates/katalog.yaml):
//
//	apiVersion: orkestra.orkspace.io/v1
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
	// Map key is the CRD name; Name field is injected from the key during loading.
	CRDs map[string]CRDEntry `yaml:"crds"`
}

// KatalogForUI is a UI-friendly representation of the merged Katalog.
// It contains only the fields needed for display in the Control Center,
// excluding internal runtime fields.
type KatalogForUI struct {
	APIVersion string                       `json:"apiVersion"`          // Orkestra API version
	Kind       string                       `json:"kind"`                // Always "Katalog" at runtime
	Metadata   KatalogMeta                  `json:"metadata"`            // Katalog metadata (name, description, etc.)
	Spec       KatalogSpecForUI             `json:"spec"`                // CRD definitions
	Security   KatalogSecurity              `json:"security"`            // Security settings
	Providers  []KatalogProviderRequirement `json:"providers,omitempty"` // Provider requirements
}

// KatalogSpecForUI contains the CRD definitions for UI display.
type KatalogSpecForUI struct {
	CRDs map[string]CRDEntry `json:"crds"` // Map of CRD name to CRD definition
}
