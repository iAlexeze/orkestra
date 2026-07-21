// pkg/types/types_crd_entry.go
package types

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── CRDEntry ──────────────────────────────────────────────────────────────────
// One entry per CRD in the Katalog.
//
// YAML fields are populated by the YAML parser when running in YAML mode,
// or set directly in BuildKatalogFromGo() when running in Go mode.
//
// Fields tagged yaml:"-" are populated at runtime during Katalog validation
// and wiring — they are never parsed from YAML and never set manually.

type CRDEntry struct {
	// ── Identity ──────────────────────────────────────────────────────────────

	// Name — unique CRD identifier within the Katalog. Must be lowercase.
	// Injected from the map key during loading — never set from YAML.
	Name string `yaml:"-" json:"name" validate:"required,hostname_rfc1123"`

	// KatalogName — unique identifier for the katalog in the runtime.
	KatalogName string `yaml:"katalogName,omitempty" json:"katalogName,omitempty"`

	// KatalogNamespace — the namespace this CRD's Katalog belongs to.
	// Defaults to "default" when not declared. Used by the Control Center to
	// group CRDs by team/tenant within a single runtime.
	KatalogNamespace string `yaml:"katalogNamespace,omitempty" json:"katalogNamespace,omitempty"`

	// KatalogDescription — the description from the source Katalog's metadata.
	// Falls back to the Komposer's description when the sub-Katalog has none.
	KatalogDescription string `yaml:"katalogDescription,omitempty" json:"katalogDescription,omitempty"`

	// KatalogVersion — the version from the source Katalog's metadata.
	// Falls back to the Komposer's version when the sub-Katalog has none.
	KatalogVersion string `yaml:"katalogVersion,omitempty" json:"katalogVersion,omitempty"`

	// CrossAccess controls whether other Katalogs can read this CRD's CR state
	// via the cross: block. Defaults to true (readable). Set to false to opt
	// this CRD out of cross reads — the reconciler returns empty for any
	// cross: reference that targets an opted-out CRD.
	CrossAccess *bool `yaml:"crossAccess,omitempty" json:"crossAccess,omitempty"`

	// Enabled — include this CRD in the runtime. false = skipped entirely.
	// WARNING: only set to false after stripping Orkestra finalizers from all
	// live CRs — disabled CRDs with live finalizers will cause stuck objects.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Critical — if true, Orkestra marks the entire controller as degraded when
	// this CRD's health state transitions to degraded.
	// Use for CRDs that are fundamental to the platform's correctness.
	// Critical *bool `yaml:"critical,omitempty" json:"critical,omitempty"`

	// Description — human-readable description. Shown in /katalog API responses.
	Description string `yaml:"description,omitempty" json:"description,omitempty" validate:"omitempty"`

	// Mode — see CRDMode for full documentation.
	// Auto-detected when omitted based on whether apiTypes.location is set.
	Mode CRDMode `yaml:"mode,omitempty" json:"mode,omitempty" validate:"omitempty,oneof=typed dynamic"`

	// ── API Types ─────────────────────────────────────────────────────────────
	// See APITypes for full field documentation.
	APITypes APITypes `yaml:"apiTypes" json:"apiTypes" validate:"required"`

	// CRDFile is the path to the CRD YAML file to apply before operator start.
	// Supports relative (resolved from katalog file location), absolute, or
	// remote (https://…) paths.
	//
	// Only applied when running outside the cluster (dev mode via ork run).
	// In production, CRDs must be pre-applied by the platform operator.
	//
	// During ork validate, the file is read and its group/kind are checked
	// against apiTypes to catch mismatches before deployment.
	CRDFile string `yaml:"crdFile,omitempty" json:"crdFile,omitempty"`

	// CRFiles is an ordered list of CR YAML files to apply before the runtime
	// starts. Applied in declaration order after the CRD is registered.
	// Same path resolution as CRDFile. Dev mode only.
	CRFiles []string `yaml:"crFiles,omitempty" json:"crFiles,omitempty"`

	// Setup declares prerequisite resources to apply before Orkestra starts.
	// Shorthand: a plain list of strings applies each file (backward compatible).
	Setup *SetupConfig `yaml:"setup,omitempty" json:"setup,omitempty"`

	// ── Runtime objects ───────────────────────────────────────────────────────
	// Set by addRuntimeObjects() during Katalog validation. Never set from YAML.
	//
	// Typed mode:        DynamicModeObject and ListDynamicModeObject are factory functions
	//                    from ObjectRegistry and ListRegistry respectively.
	//                    TypedModeObject and ListTypedModeObject are set in BuildKatalogFromGo().
	//
	// Dynamic mode: DynamicModeObject and ListDynamicModeObject are factory functions
	//                    that return *unstructured.Unstructured and *unstructured.UnstructuredList.
	//                    These are always set by addRuntimeObjects() — never nil after validation.
	TypedModeObject       runtime.Object        `yaml:"-" json:"-"`
	ListTypedModeObject   runtime.Object        `yaml:"-" json:"-"`
	DynamicModeObject     func() runtime.Object `yaml:"-" json:"-"`
	ListDynamicModeObject func() runtime.Object `yaml:"-" json:"-"`

	// Scheme — AddToScheme function generated by controller-gen for this API type.
	// Required for typed mode so the REST client can decode API server responses.
	// Not needed for dynamic mode — the dynamic client bypasses scheme decoding.
	// Set in BuildKatalogFromGo() for Go mode. Handled by RegisterScheme() for YAML mode.
	Scheme func(s *runtime.Scheme) error `yaml:"-" json:"-"`

	// ── Computed GVK/GVR ─────────────────────────────────────────────────────
	// Set by setGroupVersionKind() during Katalog validation.
	// Derived from APITypes fields. Never set manually.
	GroupVersion         *schema.GroupVersion        `yaml:"-" json:"-"`
	GroupVersionKind     schema.GroupVersionKind     `yaml:"-" json:"-"`
	GroupVersionResource schema.GroupVersionResource `yaml:"-" json:"-"`

	// ── Scope ─────────────────────────────────────────────────────────────────

	// Namespaced — true if this CRD is namespace-scoped, false if cluster-scoped.
	// Default is true
	Namespaced *bool `yaml:"namespaced,omitempty" json:"namespaced,omitempty"`

	// Namespace — target namespace for namespace-scoped CRDs.
	// Informer watches this namespace only. Empty = all namespaces.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty" validate:"omitempty"`

	// ── Runtime behaviour ─────────────────────────────────────────────────────

	// WorkersActive — records number of active concurrent reconcile workers for this CRD.
	WorkersActive int `yaml:"workersActive,omitempty" json:"workersActive,omitempty" validate:"omitempty,gte=1,lte=50"`

	// DependsOn — names of other CRDs that must reach a condition before this one starts.
	// Orkestra resolves the dependency graph and starts CRDs in topological order.
	// Cycle detection runs at validation time — cycles fail fast with a clear error.
	// Supports three YAML formats (list, key-value, full map) — see DependsOnMap.
	DependsOn DependsOnMap `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`

	// ── Enrichment ─────────────────────────────────────────────────────────────
	// Controls which secondary data is fetched and embedded into child resource
	// maps before they are available in template context.
	//
	// Only one of EnrichAll or Enrich may be set — setting both is a validation error.
	//
	//	enrichAll: true    — enrich all supported resource types
	//	enrich: [pods]     — enrich only the listed targets (shorthand)
	//	enrich:            — conditional enrichment (struct form)
	//	  - events:
	//	      when:
	//	        - field: "{{ replicasReady .children.deployment }}"
	//	          equals: "false"
	EnrichAll bool           `yaml:"enrichAll,omitempty" json:"enrichAll,omitempty"`
	Enrich    []EnrichTarget `yaml:"enrich,omitempty"    json:"enrich,omitempty"`

	// ── OperatorBox ────────────────────────────────────────────────────
	OperatorBox OperatorBoxConfig `yaml:"operatorBox,omitempty" json:"operatorBox,omitempty"`

	// Labels           []ResourceLabel  `yaml:"labels,omitempty" json:"labels,omitempty" validate:"omitempty"`
	// LabelSelector filters which resources this CRD entry reconciles.
	// Only resources whose labels match ALL declared key-value pairs are watched.
	// Required for built-in types (ConfigMap, Pod, etc.) — without a selector,
	// Orkestra would reconcile every instance in the cluster.
	// For custom CRDs this is optional — can narrow scope within a CRD.
	LabelSelector SelectorMap `yaml:"labelSelector,omitempty"`

	// FieldSelector filters which resources this CRD entry reconciles.
	// Only resources whose *fields* match ALL declared key-value expressions
	// are listed or watched. Field selectors operate on the server side and
	// support exact-match comparisons on well-known metadata paths
	// (e.g. "metadata.name", "metadata.namespace").
	//
	// Unlike label selectors, field selectors cannot match arbitrary user-defined
	// keys — only fields exposed by the Kubernetes API server. They are evaluated
	// before any client-side filtering, reducing load on the informer pipeline.
	//
	// Common use cases:
	//   - Restricting reconciliation to a specific namespace:
	//       {key: "metadata.namespace", value: "default"}
	//   - Targeting a single object by name:
	//       {key: "metadata.name", value: "my-config"}
	//
	// Field selectors are optional for all types. When omitted, Orkestra will
	// watch all objects permitted by LabelSelector and namespace restrictions.
	FieldSelector SelectorMap `yaml:"fieldSelector,omitempty"`

	// RegistryRef is the OCI or Git reference this CRD entry was loaded from.
	// Set by the merger after loadRegistrySource resolves the pattern.
	// Empty for CRDs loaded from local file sources.
	RegistryRef string `yaml:"-" json:"-"`

	// IsBuiltIn is set to true when this CRD entry was enriched from the
	// built-in Kubernetes resource registry. Used for ork validate output
	// and informational logging only — does not affect runtime behavior.
	IsBuiltIn bool `yaml:"-" json:"-"` // never serialized — runtime state only

	// IgnoreStatusPatch reports whether or not to patch the status of this CRD
	IgnoreStatusPatch bool `yaml:"ignoreStatusPatch,omitempty" json:"ignoreStatusPatch,omitempty"`

	// IgnoreObservedGeneration reports whether or not to ignore the observedGeneration field for this CRD.
	IgnoreObservedGeneration bool `yaml:"ignoreObservedGeneration,omitempty" json:"ignoreObservedGeneration,omitempty"`

	// IsStatusless reports whether this CRD has no meaningful readiness semantics.
	// These resources become "Ready" immediately upon creation.
	IsStatusless bool `yaml:"-" json:"IsStatusless,omitempty"`

	// BuiltInGroup is the display name of the API group for built-in resources.
	// "core" for resources in the core group (empty string group).
	// Only set when IsBuiltIn is true.
	BuiltInGroup string `yaml:"-" json:"-"` // never serialized

	// EnrichmentOutcome records the result of the API type enrichment phase.
	// During validation, built‑in Kubernetes kinds (e.g., Pod, Deployment, Secret)
	// are automatically enriched with their full API metadata — group, version,
	// plural, API path, and namespaced scope. This allows users to specify only:
	//
	//	apiTypes:
	//	  kind: Pod
	//
	// and rely on Orkestra to resolve all remaining fields based on the
	// Kubernetes discovery API. Custom resources are enriched using their declared
	// group/version/kind. This field is never serialized and is used internally to
	// report enrichment status and drive downstream runtime behavior.
	EnrichmentOutcome EnrichmentOutcome `yaml:"-" json:"-"` // never serialized

	// Endpoints defines which operator HTTP endpoints are enabled for this CRD.
	Endpoints EndpointsConfig `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`

	// Restricted Namespaces
	RestrictedNamespaces RestrictedNamespaces `yaml:"restrictedNamespaces,omitempty" json:"restrictedNamespaces,omitempty"`

	// Allowed Namespaces
	AllowedNamespaces AllowedNamespaces `yaml:"allowedNamespaces,omitempty" json:"allowedNamespaces,omitempty"`

	// Conversion is useful for handling multi-version crd
	Conversion *CRDConversion `yaml:"conversion,omitempty" json:"conversion,omitempty"`

	// Validation is a list of rules
	Validation *ValidationConfig `yaml:"validation,omitempty" json:"validation,omitempty"`

	// Mutation is a list of rules
	Mutation *MutationConfig `yaml:"mutation,omitempty" json:"mutation,omitempty"`

	// Webhooks controls per-CRD admission webhook behaviour.
	// Only meaningful when ENABLE_ADMISSION_WEBHOOK=true.
	// By default, any CRD with Validation or Mutation rules is included
	// in the corresponding webhook configuration automatically.
	// Set validation: false or mutation: false to opt a specific CRD out of
	// admission-time interception while keeping its reconcile-time enforcement.
	Webhooks AdmissionWebhookConfig `yaml:"webhooks,omitempty" json:"webhooks,omitempty"`

	// Normalize Spec fields before rendering
	Normalize *NormalizeConfig `yaml:"normalize,omitempty"`

	// NotificationEnabled returns whether this CRD belongs to katalog with notification access
	NotificationEnabled *bool `yaml:"-" json:"-"`

	// RemoveFinalizers -> testing
	RemoveFinalizers bool `yaml:"removeFinalizers,omitempty" json:"removeFinalizers,omitempty"`

	// DeletionProtection overrides the global deletion protection policy
	// for this specific CRD. If nil, both ProtectCRD and ProtectCRs default to true.
	DeletionProtection *DeletionProtectionOverride `yaml:"deletionProtection,omitempty" json:"deletionProtection,omitempty"`

	// Warnings collects non‑fatal validation messages for this CRD.
	Warnings Warnings `json:"-"` // not serialized

	// Imports declares Motif imports for this operatorBox.
	// Each import references a Motif by OCI reference, file path, or short name,
	// and binds its inputs via with:. Resources from imported Motifs are merged
	// into OnReconcile at Katalog load time.
	// Required inputs not provided in with: are a validation error.
	Imports []MotifImport `yaml:"imports,omitempty" json:"imports,omitempty"`

	// IDP exposes this CRD through the Gateway Apply API as a developer portal
	// surface. When enabled, the Control Center renders a [+ Create] button for
	// this CRD and serves its schema via GET /api/v1/schema/{kind}.
	IDP *IDPConfig `yaml:"idp,omitempty" json:"idp,omitempty"`
}

// IDPConfig declares IDP exposure settings for a CRD entry.
type IDPConfig struct {
	// Enabled surfaces this CRD in the Control Center as a self-service form.
	// Requires gateway.applyAPI.enabled: true on the Katalog.
	// Default: false.
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Include is a path (relative to the katalog file) to a YAML file whose
	// top-level keys are IDPFieldConfig entries. Expanded at load time — the
	// result is merged into Fields, with inline Fields taking precedence.
	Include string `yaml:"include,omitempty" json:"include,omitempty"`

	// Fields provides presentation hints layered on top of the CRD's OpenAPI
	// schema. Each key matches a field path in spec. Hints are merged with the
	// schema at GET /api/v1/schema/{kind} time — they do not replace the schema.
	Fields map[string]IDPFieldConfig `yaml:"fields,omitempty" json:"fields,omitempty"`

	// IgnoreFields lists spec field names that should never appear in the IDP
	// form, even though they exist in the CRD schema. Use this to hide
	// system-managed or operator-internal fields from developers.
	IgnoreFields []string `yaml:"ignoreFields,omitempty" json:"ignoreFields,omitempty"`

	// Category is a catalog label used when listing available schemas
	// via GET /api/v1/schema/. Example: "Compute", "Data", "Security".
	Category string `yaml:"category,omitempty" json:"category,omitempty"`

	// Description is a short human-readable summary shown in the service catalog.
	// Falls back to the CRD-level description when not set.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// ForceConflict, when true, sets Force: true on every server-side apply
	// for this CRD — the gateway takes ownership of any conflicting fields
	// rather than surfacing a conflict error. Equivalent to helm --force-conflict.
	// Can be overridden per-request with ?overwrite=true regardless of this setting.
	// Default: false.
	ForceConflict bool `yaml:"forceConflict,omitempty" json:"forceConflict,omitempty"`
}

// IDPFieldConfig holds display hints for one spec field in the IDP form.
type IDPFieldConfig struct {
	// Label overrides the field name in the rendered form.
	Label string `yaml:"label,omitempty" json:"label,omitempty"`

	// Placeholder is the input placeholder text.
	Placeholder string `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`

	// Hint is descriptive text rendered below the field.
	Hint string `yaml:"hint,omitempty" json:"hint,omitempty"`

	// Order controls position in the rendered form. Lower values appear first.
	// Fields with no order (0) appear after all explicitly ordered fields.
	Order int `yaml:"order,omitempty" json:"order,omitempty"`

	// Category is a section heading for visual grouping. Fields sharing a category
	// are rendered under the same heading. Works with When — if all fields in
	// a category are hidden, the heading is also hidden.
	Category string `yaml:"category,omitempty" json:"category,omitempty"`

	// When is a list of conditions that must ALL be true for this field to be
	// visible. Evaluated client-side as the user fills the form. An empty When
	// means the field is always visible.
	// Supports: equals, notEquals, time, dayOfWeek, cron, negate — same as
	// template source when: blocks.
	When []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// AnyOf is a list of conditions where at least ONE must be true for the
	// field to be visible. OR counterpart to When (AND).
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// Required, when true, marks the field as mandatory in the IDP form.
	// The browser enforces this natively — the label shows an asterisk and the
	// form cannot be submitted while the field is empty. Has no effect on
	// fields that are currently hidden by a when: or anyOf: condition.
	// For server-side enforcement use validation.rules with action: deny.
	Required bool `yaml:"required,omitempty" json:"required,omitempty"`

	// Disabled, when non-empty, renders the field greyed out with this string
	// as the reason. The field is excluded from form submission.
	// Use for maintenance windows or temporarily locked fields.
	Disabled string `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

type ConversionVersionSpec struct {
	Version string                 `json:"version" yaml:"version"`
	Spec    map[string]interface{} `json:"spec" yaml:"spec"`
}

// EndpointsConfig controls which HTTP endpoints are exposed by the operator.
//
// This allows users to selectively enable/disable endpoints while keeping
// the configuration minimal and declarative.
type EndpointsConfig struct {
	// Enabled if false disables all endpoints for this CRD
	// Default is true
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Health controls whether the /health endpoint is served.
	Health *bool `yaml:"health,omitempty" json:"health,omitempty"`

	// Info controls whether the /info endpoint is served.
	Info *bool `yaml:"info,omitempty" json:"info,omitempty"`
}

// IDPEnabled reports whether IDP is configured and enabled for this CRD.
func (c *CRDEntry) IDPEnabled() bool {
	return c.IDP != nil && c.IDP.Enabled
}

func (e EndpointsConfig) IsHealthEnabled() bool {
	if e.Enabled != nil && !*e.Enabled {
		return false
	}
	return e.Health == nil || *e.Health
}

func (e EndpointsConfig) IsInfoEnabled() bool {
	if e.Enabled != nil && !*e.Enabled {
		return false
	}
	return e.Info == nil || *e.Info
}
