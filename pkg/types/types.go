// pkg/orktypes/types.go
package types

import (
	"github.com/orkspace/orkestra/domain"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── Registries ────────────────────────────────────────────────────────────────
// Package-level registries — one set per Orkestra instance.
// Populated by RegisterRuntimeObjects() in zz_generated_runtime_registry.go,
// which is produced by `ork generate registry --file <path>`.
// Keyed by schema.GroupVersionKind. Set during Katalog validation.
//
// User code never reads or writes these directly. Orkestra reads them during
// Katalog validation via addRuntimeObjects(), addHooks(), and addReconcilers().

var ObjectRegistry = map[schema.GroupVersionKind]func() runtime.Object{}
var ListRegistry = map[schema.GroupVersionKind]func() runtime.Object{}
var HookRegistry = map[schema.GroupVersionKind]func() domain.AnyReconcileHooks{}
var ReconcilerRegistry = map[schema.GroupVersionKind]NewReconcilerFunc{}

// SchemeAdderFns holds AddToScheme functions collected from generated init()
// calls. Each generated zz_generated_runtime_registry.go appends to this slice
// in its init(); NewSchemeRegistry drains it via RegisterTypedScheme.
// This is the bridge between the user's generated package and the Orkestra
// internal pkg/runtime stub — no explicit call needed beyond the blank import.
var SchemeAdderFns []func(*runtime.Scheme) error

// ── CRDMode ────────────────────────────────────────────────────────────

// CRDMode controls how the GenericReconciler handles CR objects at runtime.
//
// typed
//
//	Requires compiled API types via apiTypes.location.
//	Objects are decoded into concrete Go structs by the REST client.
//	Full type safety and generated DeepCopy methods.
//	Required when Go hooks reference typed fields directly.
//	Use when you have generated API types and want compile-time guarantees.
//
// dynamic
//
//	No compiled types needed. Objects are map[string]interface{} at runtime.
//	Works with any CRD without code generation or controller-gen.
//	Required for declarative hook templates (onCreate, onReconcile, onDelete)
//	because field values are resolved at reconcile time via Go text/template
//	expressions against the live CR object map.
//	Use when you want zero-code operator behavior from the Katalog alone.
//
// Auto-detection when mode is omitted:
//
//	apiTypes.location is set   → typed       (compiled types available)
//	apiTypes.location is empty → dynamic (no compiled types)
//
// Override auto-detection by setting mode explicitly:
//
//	crd:
//	 - name: websites
//	   mode: dynamic   		# force dynamic even if location is set
//	   mode: typed          # force typed even if location is empty
type CRDMode string
type DependencyCondtion string

const (
	CRDModeTyped   CRDMode = "typed"
	CRDModeDynamic CRDMode = "dynamic"

	DependencyConditionStarted DependencyCondtion = "started"
	DependencyConditionHealthy DependencyCondtion = "healthy"

	// Future
	DependencyCondtionPending   DependencyCondtion = "pending"
	DependencyCondtionReady     DependencyCondtion = "ready"
	DependencyConditionDegraded DependencyCondtion = "degraded"
	DependencyConditionDeleted  DependencyCondtion = "deleted"
)

func (m CRDMode) String() string {
	return string(m)
}

// ── APITypes ──────────────────────────────────────────────────────────────────
// Mirrors the apiTypes block in crd-katalog.yaml.
// ork generate reads this block to emit ObjectRegistry + ListRegistry entries
// and the RegisterScheme() function.

type APITypes struct {
	// Object — Go type name for a single CR instance. Required for typed mode.
	// Used by ork generate to emit ObjectRegistry entries.
	// e.g. "Project" → func() runtime.Object { return &projv1.Project{} }
	Object string `yaml:"object,omitempty" json:"object,omitempty" validate:"omitempty"`

	// List — Go type name for the CR list. Required for typed mode.
	// Used by ork generate to emit ListRegistry entries.
	// e.g. "ProjectList" → func() runtime.Object { return &projv1.ProjectList{} }
	List string `yaml:"objectList,omitempty" json:"objectList,omitempty" validate:"omitempty"`

	// Alias — Go import alias for the API types package. Optional.
	// Auto-derived from the last two segments of Location if not set.
	// e.g. "projv1" → import projv1 "github.com/.../project/v1alpha1"
	Alias string `yaml:"alias,omitempty" json:"alias,omitempty" validate:"omitempty"`

	// Group — Kubernetes API group. Required in all modes.
	// e.g. "platform.orkestra.io"
	Group string `yaml:"group" json:"group" validate:"required,hostname_rfc1123"`

	// Version — API version. Required in all modes.
	// e.g. "v1alpha1"
	Version string `yaml:"version" json:"version" validate:"required"`

	// Kind — resource Kind. Required in all modes.
	// e.g. "Platform"
	Kind string `yaml:"kind" json:"kind" validate:"required"`

	// Plural — lowercase plural resource name. Required in all modes.
	// Used for REST client URL construction.
	// e.g. "projects"
	Plural string `yaml:"plural" json:"plural" validate:"required"`

	// APIPath — REST API path prefix. Default: /apis.
	// Override to /api only for core Kubernetes types (Pod, ConfigMap, etc.)
	// Almost always leave this empty — Orkestra defaults it to /apis.
	APIPath string `yaml:"apiPath,omitempty" json:"apiPath,omitempty" validate:"omitempty"`

	// Location — fully qualified Go import path for the API types package.
	// Required for typed mode. Used by ork generate for import statements
	// and scheme registration in RegisterScheme().
	// Not needed for dynamic mode — omit entirely.
	// e.g. "github.com/orkspace/orkestra/api/types/project/v1alpha1"
	Location string `yaml:"location,omitempty" json:"location,omitempty" validate:"omitempty"`
}

// ── Queue ─────────────────────────────────────────────────────────────────────

type Queue struct {
	// Shared — use the shared default workqueue instead of a per-CRD queue.
	// Default: false (each CRD gets its own isolated queue).
	Shared *bool `yaml:"shared,omitempty" json:"shared,omitempty"`

	// MaxDepth — max items in the queue before new items are dropped.
	// 0 → uses QUEUE_DEPTH env var (default: 100).
	MaxDepth int `yaml:"maxDepth,omitempty" json:"maxDepth,omitempty" validate:"omitempty,gte=0"`

	// FailureThreshold — consecutive failures before CRD health degrades.
	// 0 → uses FAILURE_THRESHOLD env var.
	FailureThreshold int `yaml:"failureThreshold,omitempty" json:"failureThreshold,omitempty" validate:"omitempty,gte=0"`
}
