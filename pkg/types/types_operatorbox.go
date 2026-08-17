// pkg/types/types_operatorbox.go
package types

import (
	"github.com/orkspace/orkestra/domain"
)

// ── PreReconcileConfig ────────────────────────────────────────────────────────────

// GateConditions declares when/anyOf conditions and optional external calls
// shared by both preReconcile gates.
type GateConditions struct {
	// External declares HTTP or gRPC calls made before conditions are evaluated.
	// Results are injected into the resolver under .external.<name>.* and are
	// available in when:/anyOf: field expressions.
	External []ExternalCallSpec `yaml:"external,omitempty" json:"external,omitempty"`

	// When declares AND conditions. All must be true for the gate to pass.
	When []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// AnyOf declares OR conditions. At least one must be true.
	// When both When and AnyOf are declared, both must pass.
	AnyOf []Condition `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
}

// HasConditions reports whether any when/anyOf conditions are declared.
func (g *GateConditions) HasConditions() bool {
	return g != nil && (len(g.When) > 0 || len(g.AnyOf) > 0)
}

// HasGate reports whether the gate has anything to evaluate — conditions or external calls.
func (g *GateConditions) HasGate() bool {
	return g != nil && (len(g.When) > 0 || len(g.AnyOf) > 0 || len(g.External) > 0)
}

// WhenConditions returns the AND conditions, safe on nil receiver.
func (g *GateConditions) WhenConditions() []Condition {
	if g == nil {
		return nil
	}
	return g.When
}

// AnyOfConditions returns the OR conditions, safe on nil receiver.
func (g *GateConditions) AnyOfConditions() []Condition {
	if g == nil {
		return nil
	}
	return g.AnyOf
}

// PreReconcileConfig groups the two pre-reconcile gates under operatorBox.preReconcile.
//
// YAML:
//
//	operatorBox:
//	  preReconcile:
//	    enqueueGate:
//	      when:
//	        - field: "{{ .spec.active }}"
//	          equals: "true"
//	    reconcileGate:
//	      when:
//	        - field: "{{ .spec.enabled }}"
//	          equals: "true"
//	      anyOf:
//	        - field: "{{ .status.phase }}"
//	          equals: "Ready"
type PreReconcileConfig struct {
	// External declares HTTP or gRPC calls made once before either gate is evaluated.
	// Results are injected into the resolver under .external.<name>.* and are
	// available in both enqueueGate and reconcileGate field expressions.
	External []ExternalCallSpec `yaml:"external,omitempty" json:"external,omitempty"`

	// EnqueueGate declares informer-level gate conditions evaluated in handleEvent
	// before the object enters the work queue. When the gate fires the object is
	// silently dropped — it never reaches the kordinator or reconciler.
	// No health state change; kordinator is never involved.
	EnqueueGate *GateConditions `yaml:"enqueueGate,omitempty" json:"enqueueGate,omitempty"`

	// ReconcileGate declares kordinator-level gate conditions evaluated after
	// dequeue, before the reconciler is called. When conditions are not met
	// the item is discarded and CRD health is set to gated.
	ReconcileGate *GateConditions `yaml:"reconcileGate,omitempty" json:"reconcileGate,omitempty"`
}

// HasPreReconcileConditions reports whether reconcileGate has any when/anyOf conditions declared.
func (r *PreReconcileConfig) HasPreReconcileConditions() bool {
	return r != nil && (r.ReconcileGate.HasConditions() || r.EnqueueGate.HasConditions())
}

// HasEnqueueGate reports whether the enqueue gate has anything to evaluate.
func (r *PreReconcileConfig) HasEnqueueGate() bool {
	return r != nil && (r.EnqueueGate.HasGate() || len(r.External) > 0)
}

// HasReconcileGate reports whether the reconcile gate has anything to evaluate.
func (r *PreReconcileConfig) HasReconcileGate() bool {
	return r != nil && (r.ReconcileGate.HasGate() || len(r.External) > 0)
}

// HasPreReconcileExternal reports whether preReconcile-level external calls are declared.
func (r *PreReconcileConfig) HasPreReconcileExternal() bool {
	return r != nil && len(r.External) > 0
}

// HasEnqueueGateExternal reports whether the enqueueGate declares external calls.
func (r *PreReconcileConfig) HasEnqueueGateExternal() bool {
	return r != nil && r.EnqueueGate != nil && len(r.EnqueueGate.External) > 0
}

// HasReconcileGateExternal reports whether the reconcileGate declares external calls.
func (r *PreReconcileConfig) HasReconcileGateExternal() bool {
	return r != nil && r.ReconcileGate != nil && len(r.ReconcileGate.External) > 0
}

// WhenConditions returns the reconcileGate AND conditions, safe on nil receiver.
func (r *PreReconcileConfig) WhenConditions() []Condition {
	if r == nil {
		return nil
	}
	return r.ReconcileGate.WhenConditions()
}

// AnyOfConditions returns the reconcileGate OR conditions, safe on nil receiver.
func (r *PreReconcileConfig) AnyOfConditions() []Condition {
	if r == nil {
		return nil
	}
	return r.ReconcileGate.AnyOfConditions()
}

// ── OperatorBoxConfig ──────────────────────────────────────────────────────────

// ReconcilerConfig groups the reconciler identity fields that are declared
// under operatorBox.reconciler: in a katalog. Separating them from the rest of
// OperatorBoxConfig makes the YAML shape explicit: everything under reconciler:
// concerns which implementation runs; everything at the operatorBox: level
// concerns what resources to manage and how.
type ReconcilerConfig struct {
	// Default controls which reconciler implementation is used for this CRD.
	//
	// true  — GenericReconciler manages the full lifecycle automatically.
	//         Handles: finalizer add/remove, Kubernetes events, metrics, health state.
	//         HookFactory is optional — set for custom business logic.
	//         OnCreate/OnReconcile/OnDelete templates are only valid when Default: true.
	//
	// false — Custom reconciler. The user provides the full reconcile implementation.
	//         Constructor must be declared (in YAML mode) or set directly (Go mode).
	//         GenericReconciler is not used — the user owns the entire lifecycle.
	//
	// Omit reconciler: entirely for declarative-only CRDs — GenericReconciler is
	// the default and default: true is implied.
	Default *bool `yaml:"default,omitempty" json:"default,omitempty" validate:"omitempty"`

	// Hooks — declares a Go hook function for GenericReconciler CRDs in typed or dynamic mode.
	// The function at Location.Function must match: func() domain.AnyReconcileHooks
	// Use this when you want full Go control over reconcile logic.
	// For declarative resource management without Go code, use OnCreate/OnReconcile/OnDelete.
	// Only one of Hooks or OnCreate/OnReconcile/OnDelete should be used — not both.
	Hooks *HookDeclaration `yaml:"hooks,omitempty" json:"hooks,omitempty" validate:"omitempty"`

	// ConstructorDecl — declares a custom reconciler constructor for default: false CRDs.
	// The function at Location.Function must match: NewReconcilerFunc
	// Required when default: false in YAML mode.
	ConstructorDecl *ConstructorDeclaration `yaml:"constructor,omitempty" json:"constructor,omitempty" validate:"omitempty"`

	// Profile — named reconciler tuning preset. Built-ins: high-throughput, conservative, development.
	// User-defined profiles declared in profiles.reconciler take precedence over built-ins.
	// Inline Workers/Resync/Queue override the profile when both are declared.
	Profile string `yaml:"profile,omitempty" json:"profile,omitempty"`

	// Workers — number of concurrent reconcile goroutines for this CRD.
	// 0 → uses Orkestra-level default (DEFAULT_WORKERS env var).
	Workers int `yaml:"workers,omitempty" json:"workers,omitempty" validate:"omitempty,gte=1,lte=50"`

	// Resync — full re-list interval for the informer cache.
	// 0 → uses Orkestra-level default (DEFAULT_RESYNC env var).
	Resync Duration `yaml:"resync,omitempty" json:"resync,omitempty"`

	// Queue — work queue tuning for this CRD.
	Queue Queue `yaml:"queue,omitempty" json:"queue,omitempty"`
}

// IsDefault returns true when the reconciler should use the GenericReconciler.
// When Default is nil (not declared), it defaults to true.
func (r *ReconcilerConfig) IsDefault() bool {
    if r == nil {
        return true
    }
    if r.Default == nil {
        return true
    }
    return *r.Default
}

// HasHooksDecl reports whether a hook declaration exists.
func (r *ReconcilerConfig) HasHooksDecl() bool {
    if r == nil {
        return false
    }
    return r.Hooks != nil
}

// HasConstructorDecl reports whether a constructor declaration exists.
func (r *ReconcilerConfig) HasConstructorDecl() bool {
    if r == nil {
        return false
    }
    return r.ConstructorDecl != nil
}

// IsEmpty reports whether the reconciler config has no meaningful settings.
// Used to skip unnecessary config blocks in the Katalog.
func (r *ReconcilerConfig) IsEmpty() bool {
    if r == nil {
        return true
    }
    if r.Default != nil {
        return false
    }
    if r.Hooks != nil {
        return false
    }
    if r.ConstructorDecl != nil {
        return false
    }
    if r.Profile != "" {
        return false
    }
    if r.Workers != 0 {
        return false
    }
    if r.Resync.Duration != 0 {
        return false
    }
    if !r.Queue.IsEmpty() {
        return false
    }
    return true
}

// OperatorBoxConfig is the per-CRD configuration block in a Katalog. It controls
// which reconciler implementation runs, what resources to manage, and how lifecycle
// hooks, status, admission, autoscaling, and rollback behave.
//
// The reconciler: sub-block is the only field that determines reconciler identity.
// All other fields (onCreate, status, admission, etc.) are independent of which
// reconciler is in use and remain at the top level.
type OperatorBoxConfig struct {
	// Reconciler groups the reconciler identity fields. Omit for declarative-only CRDs.
	// nil → GenericReconciler with default: true.
	Reconciler *ReconcilerConfig `yaml:"reconciler,omitempty" json:"reconciler,omitempty"`

	// PreReconcile declares pre-reconcile gate conditions. When declared, the kordinator
	// evaluates when/anyOf before calling the reconciler. If conditions are not met
	// the reconciler is never called — the item is discarded and re-evaluated on the
	// next informer tick.
	// nil → no gate; reconciler is always called (default behavior).
	PreReconcile *PreReconcileConfig `yaml:"preReconcile,omitempty" json:"preReconcile,omitempty"`

	// Finalizers — per-CRD finalizer list. Overrides the Katalog-level finalizer.
	// Applied by GenericReconciler when a CR is first created.
	// Stripped one-by-one before delete to unblock Kubernetes garbage collection.
	// If empty, falls back to the Katalog-level finalizer declaration.
	Finalizers []string `yaml:"finalizers,omitempty" json:"finalizers,omitempty" validate:"omitempty"`

	// ── Declarative hook templates ────────────────────────────────────────────
	// Only valid when Default: true and mode: dynamic.
	// ork generate reads these declarations and emits complete hook implementations
	// in __generated_runtime_hooks.go that call OrkestraRegistry resource functions
	// with resolved field values. No Go code required from the user.
	// Registered automatically in HookRegistry at startup via generated init().

	// OnCreate — resources to create when the CR is first reconciled.
	OnCreate *HookTemplates `yaml:"onCreate,omitempty" json:"onCreate,omitempty" validate:"omitempty"`

	// OnReconcile — drift correction resources applied on every reconcile.
	// Omit if onCreate alone is sufficient.
	OnReconcile *HookTemplates `yaml:"onReconcile,omitempty" json:"onReconcile,omitempty" validate:"omitempty"`

	// OnDelete — cleanup resources applied before finalizer removal.
	// Omit for resources covered by owner reference cascade deletion.
	OnDelete *HookTemplates `yaml:"onDelete,omitempty" json:"onDelete,omitempty" validate:"omitempty"`

	// HookFactory — called once at startCRDWorkers time to produce typed hooks.
	// nil → GenericReconciler runs with no user hooks.
	//       Finalizers, events, and metrics are still handled automatically.
	HookFactory func() domain.AnyReconcileHooks `yaml:"-" json:"-"`

	// Constructor — called once at startCRDWorkers time to build a custom reconciler.
	// Must not be nil when Default: false — enforced by Katalog validation at startup.
	Constructor NewReconcilerFunc `yaml:"-" json:"-"`

	// Status declares how Orkestra manages the CR's /status subresource.
	// nil (default): Layer 1 only — standard Ready condition after every reconcile.
	// non-nil: Layer 1 + Layer 2 declarative fields from Status.Fields.
	Status *StatusConfig `yaml:"status,omitempty" json:"status,omitempty"`

	// ProviderBlocks holds the parsed provider declarations from the Katalog.
	// Populated during Katalog loading via ParseProviderBlocks.
	// Not a YAML field — parsed from RawProviders after unmarshal.
	ProviderBlocks []ProviderBlock `yaml:"-" json:"-"`

	// RawProviders is the raw YAML map, populated during unmarshal.
	// Converted to ProviderBlocks in the Katalog loading step.
	RawProviders map[string][]map[string]interface{} `yaml:"providers,omitempty" json:"providers,omitempty"`

	// Cross declares cross-CRD observations.
	// Read before any resource groups — results available as .cross.<as>.status.*
	Cross []CrossCRDDeclaration `yaml:"cross,omitempty" json:"cross,omitempty"`

	// Autoscale declares runtime autoscale behavior for this operatorbox.
	// When declared, the autoscaler evaluates conditions on a ticker and applies
	// or restores worker/queue/resync overrides automatically.
	// nil → no autoscaling; CRD runs with its declared static worker count.
	Autoscale *AutoscaleSpec `yaml:"autoscale,omitempty" json:"autoscale,omitempty"`

	// Rollback declares failure-recovery behavior for this operatorbox.
	// When declared, Orkestra tracks consecutive reconcile failures and re-applies
	// the last known good spec when the trigger threshold is crossed.
	// nil → no rollback; failures are retried indefinitely.
	Rollback *RollbackBlock `yaml:"rollback,omitempty" json:"rollback,omitempty"`

	// RollBackOnError is a zero-config rollback shorthand.
	//
	// When true, Orkestra automatically rolls back to the previous spec whenever
	// the default trigger threshold is reached (3 consecutive failures), without
	// requiring a separate rollback: block.
	//
	// The rollback templates are derived from all resource declarations that have
	// reconcile: true in onCreate: and onReconcile:. Those same templates are
	// re-applied using the previous spec as the base context — .spec.* resolves
	// to the previous spec values, so no .previous.spec.* references are needed.
	//
	// Resources with once: true are excluded — they are never regenerated by rollback.
	//
	// Combine with an explicit rollback.trigger to adjust the threshold without
	// redeclaring the rollback templates:
	//
	//	operatorBox:
	//	  rollBackOnError: true
	//	  rollback:
	//	    trigger:
	//	      consecutiveFailures: 5
	//	      withinDuration: 10m
	//
	// Declare rollback.onRollback alongside rollBackOnError: true to override the
	// derived templates for specific resources while keeping the shorthand trigger.
	RollBackOnError bool `yaml:"rollBackOnError,omitempty" json:"rollBackOnError,omitempty"`

	// When is an optional list of conditions that must all pass before
	// this field is written. If absent or empty, the field is always written.
	//
	// All conditions are AND-ed together.
	// To express OR logic, declare multiple StatusField entries for the same path.
	//
	// Conditions are evaluated against the full CR object map — the same
	// map available to template expressions. This means .status.phase,
	// .spec.image, .children.job.status.succeeded are all accessible.
	When []Condition `yaml:"when,omitempty"`

	AnyOf []Condition `yaml:"anyOf,omitempty"`
}

// IsEmpty reports true when this operatorBox is empty
func (box *OperatorBoxConfig) IsEmpty() bool {
	return box == nil
}


// HookDeclaration declares where a Go hook function lives.
// Read by ork generate to emit HookRegistry entries in zz_generated_runtime_registry.go.
// The declared function must match the signature: func() domain.AnyReconcileHooks
type HookDeclaration struct {
	// Location — fully qualified Go import path. Local or remote module.
	// e.g. "github.com/myorg/hooks" or "github.com/orkspace/orkestra/pkg/reconciler/hooks"
	Location string `yaml:"location" json:"location" validate:"required"`

	// Version — optional module version to pin for this hook.
	Version string `yaml:"version,omitempty" json:"version,omitempty" validate:"omitempty"`

	// Fetch — when true, ork generate will run:
	// `go get <location>@<version>` to fetch the requested version.
	Fetch bool `yaml:"fetch,omitempty" json:"fetch,omitempty"`

	// Function — exported function name at Location that returns hooks.
	// e.g. "ProjectHooks"
	Function string `yaml:"function" json:"function" validate:"required"`

	// Alias — Go import alias. Optional, auto-derived from Location if omitted.
	// e.g. "projecthooks"
	Alias string `yaml:"alias,omitempty" json:"alias,omitempty" validate:"omitempty"`

	// Resources — Kubernetes resource types this hook manages (used for RBAC generation).
	Resources []ManagedResource `json:"resources,omitempty" yaml:"resources,omitempty"`

	// RunHooksFirst — when true, the hook runs before declarative templates.
	// When false (default), declarative templates run first and the hook is
	// additive — the 90/10 hybrid pattern.
	RunHooksFirst bool `yaml:"runHooksFirst,omitempty" json:"runHooksFirst,omitempty"`

	// Args — arbitrary key/value pairs passed to the hook at reconcile time.
	// Read in the hook via kube.Args().String("key"), .Bool("key"), etc.
	// or via kube.Args().BindArgs(&myStruct).
	Args map[string]interface{} `yaml:"args,omitempty" json:"args,omitempty"`

	// External — HTTP calls the runtime executes before invoking the hook.
	// Results are injected into the resolver under .external.<name>.* so
	// args template expressions can reference them:
	//
	//   external:
	//     - name: flags
	//       url: "{{ .spec.serviceUrl }}/flags/{{ .metadata.name }}/v2Enabled"
	//       method: GET
	//       continueOnError: true
	//   args:
	//     featureEnabled: '{{ .external.flags.body }}'
	//
	// The hook reads kube.Args().String("featureEnabled") — no HTTP client needed.
	External []ExternalCallSpec `yaml:"external,omitempty" json:"external,omitempty"`
}

// ConstructorDeclaration declares where a custom reconciler constructor lives.
// Read by ork generate to emit ReconcilerRegistry entries.
// The declared function must match: NewReconcilerFunc
type ConstructorDeclaration struct {
	// Location — fully qualified Go import path. Local or remote module.
	Location string `yaml:"location" json:"location" validate:"required"`

	// Version — optional module version to pin for this constructor.
	Version string `yaml:"version,omitempty" json:"version,omitempty" validate:"omitempty"`

	// Fetch — when true, ork generate will run:
	// `go get <location>@<version>` to fetch the requested version.
	Fetch bool `yaml:"fetch,omitempty" json:"fetch,omitempty"`

	// Function — exported constructor function name at Location.
	// e.g. "NewManagedNamespaceReconciler"
	Function string `yaml:"function" json:"function" validate:"required"`

	// Alias — Go import alias. Optional, auto-derived from Location if omitted.
	Alias string `yaml:"alias,omitempty" json:"alias,omitempty" validate:"omitempty"`

	// Resources — Kubernetes resource types this constructor manages (used for RBAC generation).
	Resources []ManagedResource `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Args — arbitrary key/value pairs passed to the constructor at startup.
	// Read via kube.Args().String("key"), .Bool("key"), etc.
	// or via kube.Args().BindArgs(&myStruct).
	Args map[string]interface{} `yaml:"args,omitempty" json:"args,omitempty"`
}

// ManagedResource describes a Kubernetes resource type that a typed extension
// (either a hook or a constructor) will manage.
//
// Orkestra uses this information to generate RBAC rules for the operator
// ServiceAccount. Each declared resource results in permissions to
// get/list/watch/create/update/patch/delete that resource type.
//
// For built‑in Kubernetes resources, Kind alone is sufficient because Orkestra
// resolves the full GroupVersionResource from its internal registry.
//
// For custom resources or non‑core API groups, APIVersion and/or explicit
// group/version/plural may be provided.
//
// Example (in katalog.yaml):
//
//	hooks:
//	  resources:
//	    - kind: StatefulSet
//	    - kind: Service
//	    - kind: CronJob
//	    - kind: Widget
//	      group: widgets.example.com
//	      version: v1alpha1
//	      plural: widgets
type ManagedResource struct {
	// Kind is the Kubernetes Kind of the resource (e.g. "StatefulSet",
	// "Service", "CronJob"). This is the primary identifier and is required.
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	// APIVersion is optional and only needed when the Kind cannot be resolved
	// from Orkestra's built‑in registry. Example: "apps/v1", "batch/v1",
	// "widgets.example.com/v1alpha1".
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`

	// Group is optional and used for custom resources or non‑core API groups
	// when you want to fully specify the GroupVersionResource explicitly.
	// Example: "widgets.example.com".
	Group string `json:"group,omitempty" yaml:"group,omitempty"`

	// Version is optional and used together with Group for custom resources.
	// Example: "v1alpha1".
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Plural is optional and used when the plural name cannot be inferred
	// from Orkestra's built‑in registry or CRD metadata. Example: "widgets".
	Plural string `json:"plural,omitempty" yaml:"plural,omitempty"`
}
