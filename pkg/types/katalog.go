// pkg/types/katalog.go
package types

// GatewayConfig declares how the Orkestra gateway is deployed for this Katalog.
//
// YAML shape:
//
//	gateway:
//	  enabled: true     # explicitly enable gateway installation
//	  standalone: true  # gateway runs without a companion runtime operator
//	  endpoint: ""      # leave empty when standalone; sets this when paired with runtime
//	  applyAPI:
//	    enabled: true
//	    auth:
//	      tokens:
//	        - name: ci-pipeline
//	          secretRef:
//	            name: ork-apply-token
//	            key: token
//	            rotateAfter: 90d
type GatewayConfig struct {
	// Enabled declares whether the gateway should be installed for this katalog.
	// When true, this means:
	//   - Helm installation was done with '--set gateway.enabled=true'
	//   - The runtime expects a gateway to exist
	// Default: false.
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Standalone declares that this Katalog is deployed as a gateway-only installation
	// with no companion runtime operator. When true:
	//   - gatewayEndpoint validation is skipped (the gateway is self-contained)
	//   - spec: may be empty (no CRDs required)
	// Default: false.
	Standalone bool `yaml:"standalone,omitempty" json:"standalone,omitempty"`

	// Endpoint is the HTTP base URL of the gateway, used by the runtime to locate it.
	// Leave empty in standalone deployments.
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`

	// ApplyAPI enables the CRUD REST surface for CRs on this gateway.
	ApplyAPI *ApplyAPIConfig `yaml:"applyAPI,omitempty" json:"applyAPI,omitempty"`
}

// ApplyAPIConfig enables and configures the Gateway Apply API.
type ApplyAPIConfig struct {
	// Enabled activates the Apply API handlers on the gateway.
	// When true, the gateway registers POST /api/v1/apply,
	// GET/DELETE /api/v1/resources/..., and GET /api/v1/schema/... routes.
	// Default: false.
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Auth configures bearer token authentication for Apply API requests.
	Auth ApplyAPIAuth `yaml:"auth,omitempty" json:"auth,omitempty"`
}

// ApplyAPIAuth holds the token list for Apply API authentication.
type ApplyAPIAuth struct {
	// Tokens is the list of accepted bearer tokens. Every Apply API request
	// must include Authorization: Bearer <token> matching one entry.
	Tokens []ApplyAPIToken `yaml:"tokens,omitempty" json:"tokens,omitempty"`
}

// ApplyAPIToken is one bearer token entry.
// Exactly one of SecretRef or Token must be set.
type ApplyAPIToken struct {
	// Name is a human-readable identifier used in logs and audit output.
	Name string `yaml:"name" json:"name"`

	// SecretRef reads the token value from a Kubernetes Secret at startup.
	// If the Secret does not exist, the gateway creates it with a generated
	// uuidv4 token. If rotateAfter is set, the gateway rotates the token
	// using the same annotation-based rotation as pkg/runners.
	SecretRef *ApplyAPISecretRef `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`

	// Token is an ${ENV_VAR} reference expanded at startup.
	// Set the variable via extraEnv in the gateway and controlCenter Helm values.
	// Literal values are not accepted.
	Token string `yaml:"token,omitempty" json:"token,omitempty"`
}

// ApplyAPISecretRef locates a Kubernetes Secret that holds a bearer token.
type ApplyAPISecretRef struct {
	// Name is the Kubernetes Secret name.
	Name string `yaml:"name" json:"name"`

	// Key is the data key within the Secret whose value is the token.
	Key string `yaml:"key" json:"key"`

	// Namespace is the Secret's namespace.
	// Defaults to Orkestra's own namespace when empty.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// RotateAfter is an optional duration (e.g. "90d", "720h").
	// When set, the gateway checks the orkestra.io/generated-at annotation on
	// the Secret; if the age exceeds this duration, it deletes and recreates
	// the Secret with a new uuidv4 token.
	RotateAfter string `yaml:"rotateAfter,omitempty" json:"rotateAfter,omitempty"`
}

// ── GatewayConfig methods ──────────────────────────────────────────

// HasApplyAPI reports whether the Apply API is enabled and configured.
func (g *GatewayConfig) HasApplyAPI() bool {
	if g == nil {
		return false
	}
	if g.ApplyAPI == nil {
		return false
	}
	if !g.ApplyAPI.Enabled {
		return false
	}
	return g.ApplyAPI.HasAuth()
}

// ── ApplyAPIConfig methods ─────────────────────────────────────────

// HasAuth reports whether the Apply API has authentication configured.
func (a *ApplyAPIConfig) HasAuth() bool {
	if a == nil {
		return false
	}
	return a.Auth.HasTokens()
}

// ── ApplyAPIAuth methods ───────────────────────────────────────────

// HasTokens reports whether at least one token is configured.
func (a ApplyAPIAuth) HasTokens() bool {
	return len(a.Tokens) > 0
}

// IsEmpty reports whether the auth struct is completely unconfigured.
func (a ApplyAPIAuth) IsEmpty() bool {
	return len(a.Tokens) == 0
}

// KatalogFile is the top-level structure of a katalog.yaml file.
// It contains optional imports (files and helm charts) plus inline CRDs.
// Orkestra's in-built merger resolves all imports and merges everything into one KatalogSpec.
type KatalogFile struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   KatalogMeta     `yaml:"metadata"`
	Imports    *KatalogSources `yaml:"imports,omitempty"`
	Spec       KatalogSpec     `yaml:"spec"`
	Security   KatalogSecurity `yaml:"security"`

	// CrossAccess sets the default cross-read policy for all CRDs in this Katalog.
	// When false, no other Katalog may read any CRD in this one via cross:.
	// Individual CRDs may override with their own crossAccess field.
	// Defaults to true (open) when not declared.
	CrossAccess *bool `yaml:"crossAccess,omitempty" json:"crossAccess,omitempty"`

	// Gateway declares how the gateway is deployed for this Katalog.
	// When gateway.standalone: true, the gateway runs without a runtime operator
	// and spec: may be empty.
	Gateway *GatewayConfig `yaml:"gateway,omitempty" json:"gateway,omitempty"`

	// Notification holds the top-level alerting configuration for this Katalog.
	// Defines channels (email, Slack) and per-team routing rules that fire when
	// a managed CRD's conditions transition. When a Komposer references multiple
	// source Katalogs, notification blocks are merged — source teams are inherited
	// and the Komposer's own teams win on name conflict.
	Notification *KatalogNotification `yaml:"notification,omitempty"`

	// Notes declares user-defined note functions available to all CRDs in this Katalog.
	// Notes are named template expressions that compose built-in notes and Go template
	// syntax. Once declared, a note is callable by name in any template expression.
	Notes NoteRegistry `yaml:"notes,omitempty"`

	// Profiles declares named profiles available to all CRDs in this Katalog.
	// Profiles are resolved before built-in Orkestra profiles at both validate
	// and reconcile time. Template expressions in profile field values are
	// resolved at reconcile time; validation skips fields that contain {{ }}.
	Profiles ProfileRegistry `yaml:"profiles,omitempty"`

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

// KatalogMeta holds identifying metadata for a Katalog.
type KatalogMeta struct {
	// Name is the required unique identifier of the Katalog.
	Name string `yaml:"name" json:"name,omitempty"`

	// Namespace scopes this Katalog to a logical tenant or team within a single
	// runtime. Defaults to "default" when not declared — identical to Kubernetes
	// namespace semantics. The Control Center groups CRDs by namespace so each
	// team sees only its own panel.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// ClusterName identifies the cluster this Katalog runs in.
	// Used by the Control Center for cluster-level filtering when multiple
	// runtimes are connected. Katalog value takes precedence over the
	// CLUSTER_NAME env var. Empty when neither is set.
	ClusterName string `yaml:"clusterName,omitempty" json:"clusterName,omitempty"`

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
	// `ork patterns --tag <tag>` and for indexing in Artifact Hub.
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

	// Projects holds developer-side metadata injected by ork-doctor at generation
	// time. The operator and runtime ignore this field — it is purely for
	// persona-aware tooling and Control Center UI.
	Projects map[string]interface{} `yaml:"projects,omitempty" json:"projects,omitempty"`

	// Deprecation marks this pattern as deprecated. When set, consumers
	// (ork validate, ork inspect, ork patterns) display a warning.
	Deprecation *KatalogDeprecation `yaml:"deprecation,omitempty" json:"deprecation,omitempty"`
}

// KatalogDeprecation carries deprecation guidance for registry consumers.
type KatalogDeprecation struct {
	MigratedTo string `yaml:"migratedTo,omitempty" json:"migratedTo,omitempty"`
	Message    string `yaml:"message,omitempty"    json:"message,omitempty"`
}

// IsDeprecated indicates that this katalog is deprecated and should surface
// warnings in ork validate, ork inspect, and registry consumers.
func (d *KatalogDeprecation) IsDeprecated() bool {
	if d == nil {
		return false
	}
	return d.MigratedTo != "" || d.Message != ""
}

// MigrationTarget returns the value of the MigratedTo field.
// If the deprecation block is nil or empty, it returns an empty string.
func (d *KatalogDeprecation) MigrationTarget() string {
	if d == nil {
		return ""
	}
	return d.MigratedTo
}

// Message returns the deprecation message.
// If the deprecation block is nil or empty, it returns an empty string.
func (d *KatalogDeprecation) MigrationMessage() string {
	if d == nil {
		return ""
	}
	return d.Message
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
	// Imports declares Motifs whose profiles are merged into the Katalog-wide
	// ProfileRegistry. Only profiles: from each Motif are consumed here;
	// resources, status, and admission declarations in the Motif are ignored
	// at this level. Use spec.crds[name].imports for those.
	Imports []MotifImport `yaml:"imports,omitempty"`

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
