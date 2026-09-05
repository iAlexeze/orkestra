// pkg/types/types_operatorbox.go
package types

import (
	"slices"
	"strings"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/runtime/sentinel"
)

// ── FailPolicy ────────────────────────────────────────────────────────────────────

// FailPolicy controls what a gate does when it cannot evaluate its conditions —
// for example when an external: call fails or times out.
type FailPolicy string

const (
	// FailPolicyOpen passes the gate on evaluation failure.
	// The object is enqueued / reconciled as if the gate was not declared.
	// This is the default when failPolicy is omitted.
	FailPolicyOpen FailPolicy = "open"

	// FailPolicyClosed holds the gate on evaluation failure.
	// The object is dropped from the queue / held back from the reconciler.
	// Use on reconcileGate when unknown state is worse than a missed reconcile.
	FailPolicyClosed FailPolicy = "closed"
)

// String() stringifies a policy
func (f FailPolicy) String() string {
	return string(f)
}

// ValidFailPolicies returns all known failPolicy values in declaration order.
func ValidFailPolicies() []string {
	return []string{string(FailPolicyOpen), string(FailPolicyClosed)}
}

// IsValidFailPolicy reports whether s is a known FailPolicy value.
func IsValidFailPolicy(s string) bool {
	switch FailPolicy(s) {
	case FailPolicyOpen, FailPolicyClosed:
		return true
	}
	return false
}

// FailPolicyJoined returns a comma-separated list of valid failPolicy values for error messages.
func FailPolicyJoined() string { return strings.Join(ValidFailPolicies(), ", ") }

// ── PreReconcileConfig ────────────────────────────────────────────────────────────

// GateConditions declares when/or conditions and optional external calls
// shared by both preReconcile gates.
type GateConditions struct {
	// External declares HTTP or gRPC calls made before conditions are evaluated.
	// Results are injected into the resolver under .external.<name>.* and are
	// available in when:/or: field expressions.
	External []ExternalCallSpec `yaml:"external,omitempty" json:"external,omitempty"`

	// Sentinels declares the event-time values this operator uses in gate conditions.
	// Declared here as a shorthand instead of when/or conditions. It uses the same
	// semantics as 'or' conditions since first match passes. Must be a valid subset of
	// preReconcile.sentinels. Checked first before the conditions. Use when to require
	// more than one sentinel
	Sentinels []string `yaml:"sentinels,omitempty" json:"sentinels,omitempty"`

	// EventAware preserves individual events through queue deduplication for this gate.
	// When true, each event that reaches the reconcile gate is evaluated as a distinct
	// work item. When false, normal queue coalescing applies and gate evaluation is
	// state-oriented: multiple events for the same resource may collapse into one
	// reconciliation.
	EventAware bool `yaml:"eventAware,omitempty" json:"eventAware,omitempty"`

	// When declares AND conditions. All must be true for the gate to pass.
	When []Condition `yaml:"when,omitempty" json:"when,omitempty"`

	// Or declares OR conditions. At least one must be true.
	// When both When and Or are declared, both must pass.
	Or []Condition `yaml:"or,omitempty" json:"or,omitempty"`

	// FailPolicy controls what the gate does when it cannot evaluate its conditions —
	// for example when an external: call fails or times out.
	// Defaults to open when omitted.
	FailPolicy FailPolicy `yaml:"failPolicy,omitempty" json:"failPolicy,omitempty"`
}

// HasConditions reports whether any when/or conditions are declared.
func (g *GateConditions) HasConditions() bool {
	return g != nil && (len(g.When) > 0 || len(g.Or) > 0)
}

// HasGate reports whether the gate has anything to evaluate — conditions or external calls.
func (g *GateConditions) HasGate() bool {
	// return g != nil && (len(g.When) > 0 || len(g.Or) > 0 || len(g.External) > 0)
	return g != nil
}

// HasSentinels reports whether the gate has declared sentinels
func (g *GateConditions) HasSentinels() bool {
	return g != nil && len(g.Sentinels) > 0
}

// IsEventAware reports whether this gate requires per-event evaluation.
func (g *GateConditions) IsEventAware() bool {
	return g != nil && g.EventAware
}

// SentinelContains reports true if s is declared in g.Sentinels.
func (g *GateConditions) SentinelContains(s string) bool {
	if !g.HasSentinels() {
		return false
	}
	for _, name := range g.Sentinels {
		if name == s {
			return true
		}
	}
	return false
}

// SentinelsAllowed implements the fast-path shorthand for gate conditions
// that declared sentinels. Returns true on the first match (OR semantics,
// same as the or: block).
//
//	preReconcile:
//	  enqueueGate:
//	    sentinels:
//	    - generationChanged
//	    - ownerReferenceChanged
//
//	  reconcileGate:
//	    sentinels:
//	     - namespaceChanged
//	     - uidChanged
func (g *GateConditions) SentinelsAllowed(declared map[string]string) bool {
	if !g.HasSentinels() {
		return false
	}

	for k, v := range declared {
		if g.SentinelContains(k) {
			if v == "true" {
				return true
			}
		}
	}

	return false
}

// WhenConditions returns the AND conditions, safe on nil receiver.
func (g *GateConditions) WhenConditions() []Condition {
	if g == nil {
		return nil
	}
	return g.When
}

// OrConditions returns the OR conditions, safe on nil receiver.
func (g *GateConditions) OrConditions() []Condition {
	if g == nil {
		return nil
	}
	return g.Or
}

// ExternalCalls returns the external calls declared on this gate, or nil when
// the gate is nil or has no external declarations.
func (g *GateConditions) ExternalCalls() []ExternalCallSpec {
	if g == nil {
		return nil
	}
	return g.External
}

// ── Watch event types ─────────────────────────────────────────────────────

// WatchEvent is the string type for watch event types used in WatchEntry.On.
type WatchEvent string

const (
	WatchEventCreate WatchEvent = "create"
	WatchEventUpdate WatchEvent = "update"
	WatchEventDelete WatchEvent = "delete"
)

// String() stringifies a watch event
func (w WatchEvent) String() string {
	return string(w)
}

// ValidWatchEvents returns all known watch event values in declaration order.
func ValidWatchEvents() []string {
	return []string{
		string(WatchEventCreate),
		string(WatchEventUpdate),
		string(WatchEventDelete),
	}
}

// IsValidWatchEvent reports whether s is a known watch event type.
func IsValidWatchEvent(s string) bool {
	switch WatchEvent(s) {
	case WatchEventCreate, WatchEventUpdate, WatchEventDelete:
		return true
	}
	return false
}

// IsAllValid reports if a slice of watch event strings are valid
func IsAllValid(events []string) (bool, []string) {
	var unknown []string
	for _, event := range events {
		if !IsValidWatchEvent(event) {
			unknown = append(unknown, event)
		}
	}
	if len(unknown) > 0 {
		return false, unknown
	}
	return true, nil
}

// ── WatchEntry ────────────────────────────────────────────────────────────────

// WatchEntry declares a secondary Kubernetes resource Orkestra should watch.
// When the resource changes, Orkestra resolves the relevant primary CR key(s)
// and enqueues them — no Go required.
//
// Key resolution: if the changed object has an ownerReference pointing to a
// primary CR, that CR is enqueued. Otherwise all known CRs of the primary kind
// are enqueued (shared-resource broadcast).
//
// YAML:
//
//	operatorBox:
//	  watch:
//	    - apiVersion: apps/v1
//	      kind: Deployment
//	    - apiVersion: v1
//	      kind: ConfigMap
//	      namespace: my-operator-system
//	      name: shared-config
//	    - apiVersion: v1
//	      kind: Node
//	      on: [update]
type WatchEntry struct {
	// APIVersion is the Kubernetes API version of the resource to watch.
	// e.g. "apps/v1", "v1", "networking.k8s.io/v1"
	APIVersion string `yaml:"apiVersion" json:"apiVersion" validate:"required"`

	// Kind is the Kubernetes Kind of the resource to watch.
	// e.g. "Deployment", "ConfigMap", "Node"
	Kind string `yaml:"kind" json:"kind" validate:"required"`

	// Namespace restricts the watch to a single namespace.
	// When empty the watch is cluster-scoped (all namespaces).
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// Name restricts the watch to a single named resource.
	// When set only events for that specific object trigger the enqueue.
	// Typically used for well-known shared resources (a specific ConfigMap or Secret).
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// On declares which event types trigger the enqueue.
	// Valid values: WatchEventCreate, WatchEventUpdate, WatchEventDelete.
	// When empty all three event types are watched.
	On []string `yaml:"on,omitempty" json:"on,omitempty"`

	// EnqueueGate declares conditions evaluated before enqueueing when the watch
	// fires. Sentinels (e.g. generationChanged) are computed against the watched
	// resource's oldObj / newObj at UpdateFunc time and are valid here.
	// When nil all events that pass the On filter are enqueued.
	EnqueueGate *GateConditions `yaml:"enqueueGate,omitempty" json:"enqueueGate,omitempty"`

	// KeyFrom overrides the default key resolution (ownerReference → broadcast).
	// Declare when the standard mechanisms do not express the mapping you need.
	// When nil the runtime checks ownerReferences first; if none match the primary
	// CRD it broadcasts to all known primary CRs.
	KeyFrom *WatchKeyFrom `yaml:"keyFrom,omitempty" json:"keyFrom,omitempty"`

	// Index declares field-path indexers that Orkestra registers on this watch's
	// informer. Each entry makes client.List(ctx, &list, client.MatchingFields{name: value})
	// serve from the cache instead of making a live API call.
	//
	//  watch:
	//    - apiVersion: v1
	//      kind: ConfigMap
	//      index:
	//        - name: metadata.ownerRef
	//          field: ".metadata.ownerReferences[0].name"
	Index []WatchIndex `yaml:"index,omitempty" json:"index,omitempty"`

	// Include is a path (relative to the katalog file) to a YAML file whose
	// "watch:" list replaces this entry in-place. When set all other fields on
	// this entry are ignored. Cleared after expansion.
	Include string `yaml:"include,omitempty" json:"include,omitempty"`
}

// WatchIndex declares one field-path indexer on a watch: entry informer.
// Name is used as the index key — it must match the key passed to client.MatchingFields.
// Field is a dot-separated JSON path into the watched object (e.g. "spec.owner").
type WatchIndex struct {
	// Name is the index name. Must match the key in client.MatchingFields.
	Name string `yaml:"name" json:"name"`
	// Field is the JSON path to index on (e.g. ".spec.owner", "metadata.labels.app").
	Field string `yaml:"field" json:"field"`
}

// WatchKeyFrom overrides the default ownerReference → broadcast key resolution
// for a watch: entry. Exactly one of Label or Name must be set.
//
//	keyFrom:
//	  label: "app.kubernetes.io/cr-owner"   # label on the watched object carries the key
//
//	keyFrom:
//	  name: "my-singleton-cr"               # always enqueue this named primary CR
//	  namespace: "my-namespace"             # optional; omit for cluster-scoped CRDs
type WatchKeyFrom struct {
	// Label names a label on the watched object whose value is the primary CR key.
	// The value must be a valid Kubernetes key: "namespace/name" or bare "name".
	// Mutually exclusive with Name.
	Label string `yaml:"label,omitempty" json:"label,omitempty"`

	// Name is a fixed primary CR name to enqueue regardless of which watched
	// object changed. Use for singleton operators (one CR per cluster).
	// Mutually exclusive with Label.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Namespace qualifies Name. Omit for cluster-scoped primary CRDs.
	// Ignored when Label is set.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
}

// Key returns the enqueue key for the fixed-name variant.
func (kf *WatchKeyFrom) Key() string {
	if kf.Namespace != "" {
		return kf.Namespace + "/" + kf.Name
	}
	return kf.Name
}

// WatchesOn reports whether the entry should fire for the given event type.
func (w WatchEntry) WatchesOn(event string) bool {
	if len(w.On) == 0 {
		return true
	}
	for _, e := range w.On {
		if e == event {
			return true
		}
	}
	return false
}

// InvalidOnValues returns any On values that are not valid WatchEvent constants.
// Returns nil when all values are valid.
func (w WatchEntry) InvalidOnValues() []string {
	var invalid []string
	for _, e := range w.On {
		if !IsValidWatchEvent(e) {
			invalid = append(invalid, e)
		}
	}
	return invalid
}

// ToManagedResource converts the entry to a ManagedResource suitable for
// ResolveGVR — used for RBAC generation.
func (w WatchEntry) ToManagedResource() ManagedResource {
	return ManagedResource{
		APIVersion: w.APIVersion,
		Kind:       w.Kind,
	}
}

// ── PreReconcileConfig ────────────────────────────────────────────────────────────

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
//	      or:
//	        - field: "{{ .status.phase }}"
//	          equals: "Ready"
type PreReconcileConfig struct {
	// External declares HTTP or gRPC calls made once before either gate is evaluated.
	// Results are injected into the resolver under .external.<name>.* and are
	// available in both enqueueGate and reconcileGate field expressions.
	External []ExternalCallSpec `yaml:"external,omitempty" json:"external,omitempty"`

	// Sentinels declares the event-time values this operator uses in gate conditions.
	// Each sentinel is computed by the informer's UpdateFunc against oldObj/newObj
	// and carried through the queue entry so both enqueueGate and reconcileGate
	// can reference it. The informer computes only declared sentinels.
	//
	// Valid values: SentinelGenerationChanged, SentinelLabelsChanged, SentinelAnnotationsChanged,
	// SentinelDeletionStarted, SentinelFinalizersChanged and 9more.
	//
	// ork validate fails if a sentinel is used in a gate template but not declared
	// here, or if a sentinel is used outside the preReconcile context.
	Sentinels []string `yaml:"sentinels,omitempty" json:"sentinels,omitempty"`

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

// HasSentinels returns true if sentinels are declared
func (r *PreReconcileConfig) HasSentinels() bool {
	if r == nil {
		return false
	}
	return r.Sentinels != nil
}

// DeclaredSentinels returns the sentinel names declared under preReconcile.sentinels.
// Returns nil when no sentinels are declared. Safe on nil receiver.
func (r *PreReconcileConfig) DeclaredSentinels() []string {
	if r == nil {
		return nil
	}
	return r.Sentinels
}

// InvalidSentinels returns any sentinel values that are not valid Sentinel constants.
// Returns nil when all values are valid. Safe on nil receiver.
func (r *PreReconcileConfig) InvalidSentinels() []string {
	if r == nil {
		return nil
	}
	_, invalid := sentinel.IsAllValid(r.Sentinels)
	return invalid
}

// InvalidGateSentinels returns any sentinel values that are not subset of preReconcile.sentinels.
// It does not check for validity of the sentinel. That is already done by the upstream preReconcile InvalidSentinels()
// Returns nil, false when all values are valid. Safe on nil receiver.
func (r *PreReconcileConfig) InvalidGateSentinels(g *GateConditions) ([]string, bool) {
	if r == nil && g == nil {
		return nil, false
	}

	invalid := []string{}
	for _, sent := range g.Sentinels {
		if !slices.Contains(r.Sentinels, sent) {
			invalid = append(invalid, sent)
		}
	}
	if len(invalid) > 0 {
		return invalid, true
	}

	return nil, false
}

// HasPreReconcileConditions reports whether reconcileGate has any when/or conditions declared.
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

// IsEventAware reports whether this reconcile gate requires per-event evaluation.
func (r *PreReconcileConfig) IsEventAware() bool {
	return r != nil && r.HasReconcileGate() && r.ReconcileGate.IsEventAware()
}

// HasPreReconcileExternal reports whether preReconcile-level external calls are declared.
func (r *PreReconcileConfig) HasPreReconcileExternal() bool {
	return r != nil && len(r.External) > 0
}

// HasEnqueueGateSentinel reports whether the enqueueGate declares sentinels.
func (r *PreReconcileConfig) HasEnqueueGateSentinel() bool {
	return r != nil && r.EnqueueGate != nil && len(r.EnqueueGate.Sentinels) > 0
}

// HasReconcileGateSentinel reports whether the reconcileGate declares sentinels.
func (r *PreReconcileConfig) HasReconcileGateSentinel() bool {
	return r != nil && r.ReconcileGate != nil && len(r.ReconcileGate.Sentinels) > 0
}

// HasEnqueueGateExternal reports whether the enqueueGate declares external calls.
func (r *PreReconcileConfig) HasEnqueueGateExternal() bool {
	return r != nil && r.EnqueueGate != nil && len(r.EnqueueGate.External) > 0
}

// HasReconcileGateExternal reports whether the reconcileGate declares external calls.
func (r *PreReconcileConfig) HasReconcileGateExternal() bool {
	return r != nil && r.ReconcileGate != nil && len(r.ReconcileGate.External) > 0
}

// GateExternalCalls returns all external calls declared across enqueueGate and
// reconcileGate. Returns nil when the receiver is nil or neither gate has calls.
func (r *PreReconcileConfig) GateExternalCalls() [][]ExternalCallSpec {
	if r == nil {
		return nil
	}
	var phases [][]ExternalCallSpec
	if calls := r.EnqueueGate.ExternalCalls(); len(calls) > 0 {
		phases = append(phases, calls)
	}
	if calls := r.ReconcileGate.ExternalCalls(); len(calls) > 0 {
		phases = append(phases, calls)
	}
	return phases
}

// WhenConditions returns the reconcileGate AND conditions, safe on nil receiver.
func (r *PreReconcileConfig) WhenConditions() []Condition {
	if r == nil {
		return nil
	}
	return r.ReconcileGate.WhenConditions()
}

// OrConditions returns the reconcileGate OR conditions, safe on nil receiver.
func (r *PreReconcileConfig) OrConditions() []Condition {
	if r == nil {
		return nil
	}
	return r.ReconcileGate.OrConditions()
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

	// Requeue declares per-object requeue behavior after successful reconciliation.
	Requeue *RequeueConfig `yaml:"requeue,omitempty"`

	// Include is a path (relative to the katalog file) to a YAML file whose
	// "reconciler:" block is loaded and merged under this config. Inline fields
	// take precedence over included ones. Cleared after expansion.
	Include string `yaml:"include,omitempty" json:"include,omitempty"`
}

// RequeueConfig declares per-object requeue behavior after a successful reconcile.
// Evaluated after every reconcile cycle that does not return an error.
// Errors are handled by queue.retryBackoff, not by requeue.
type RequeueConfig struct {
	// After is a template expression resolving to a Go duration string.
	// Evaluated against the reconciled CR after each successful cycle.
	// "0s" or empty means no requeue — wait for the next informer event.
	// Example: '{{ .spec.checkInterval | default "60s" }}'
	After string `yaml:"after,omitempty"`

	// When declares AND conditions — requeue only fires when all are true.
	// When absent, requeue fires unconditionally after every reconcile.
	When []Condition `yaml:"when,omitempty"`

	// Or declares OR conditions — requeue fires when any one is true.
	// When both When and Or are present, both must pass.
	Or []Condition `yaml:"or,omitempty"`
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

// HasRetryBackoff reports whether a retryBackoff is declared on this reconciler's queue.
func (r *ReconcilerConfig) HasRetryBackoff() bool {
	return r != nil && r.Queue.HasRetryBackoff()
}

// HasConstructorDecl reports whether a constructor declaration exists.
func (r *ReconcilerConfig) HasConstructorDecl() bool {
	if r == nil {
		return false
	}
	return r.ConstructorDecl != nil
}

// HasRequeueDecl reports whether a requeue configuration exists.
func (r *ReconcilerConfig) HasRequeueDecl() bool {
	if r == nil {
		return false
	}
	return r.Requeue != nil
}

// IsRequeueEmpty reports whether the requeue configuration is effectively empty.
func (r *ReconcilerConfig) IsRequeueEmpty() bool {
	if r == nil || r.Requeue == nil {
		return true
	}
	rc := r.Requeue
	return rc.After == "" && len(rc.When) == 0 && len(rc.Or) == 0
}

// Empty reports whether this requeue configuration has no effective behavior.
func (rc *RequeueConfig) Empty() bool {
	if rc == nil {
		return true
	}
	return rc.After == "" && len(rc.When) == 0 && len(rc.Or) == 0
}

// Empty reports whether the PreReconcile config has no meaningful settings.
// Used to skip unnecessary config blocks in the Katalog.
func (p *PreReconcileConfig) Empty() bool {
	return p == nil
}

// Empty reports whether the reconciler config has no meaningful settings.
// Used to skip unnecessary config blocks in the Katalog.
func (r *ReconcilerConfig) Empty() bool {
	return r == nil
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
	// evaluates when/or before calling the reconciler. If conditions are not met
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

	// Watch declares secondary Kubernetes resources Orkestra should watch.
	// When a watched resource changes, Orkestra resolves the relevant primary
	// CR key(s) and enqueues them. The reconciler runs normally — the watched
	// resource's current state is available via .children.* as usual.
	// nil → no secondary watches; only the primary CRD informer is active.
	Watch []WatchEntry `yaml:"watch,omitempty" json:"watch,omitempty"`

	// Autoscale declares runtime autoscale behavior for this operatorbox.
	// When declared, the autoscaler evaluates conditions on a ticker and applies
	// or restores worker/queue/resync overrides automatically.
	// nil → no autoscaling; CRD runs with its declared static worker count.
	Autoscale *AutoscaleSpec `yaml:"autoscale,omitempty" json:"autoscale,omitempty"`

	// IN DEVELOPMENT
	// Rollback declares failure-recovery behavior for this operatorbox.
	// When declared, Orkestra tracks consecutive reconcile failures and re-applies
	// the last known good spec when the trigger threshold is crossed.
	// nil → no rollback; failures are retried indefinitely.
	Rollback *RollbackBlock `yaml:"rollback,omitempty" json:"rollback,omitempty"`

	// IN DEVELOPMENT
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

	Or []Condition `yaml:"or,omitempty"`
}

// Empty reports true when this operatorBox is empty
func (box *OperatorBoxConfig) Empty() bool {
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

	// ManagedResources — Kubernetes resource types this hook manages (used for RBAC generation).
	ManagedResources []ManagedResource `json:"managedResources,omitempty" yaml:"managedResources,omitempty"`

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

	// ManagedResources — Kubernetes resource types this constructor manages (used for RBAC generation).
	ManagedResources []ManagedResource `json:"managedResources,omitempty" yaml:"managedResources,omitempty"`

	// Args — arbitrary key/value pairs passed to the constructor at startup.
	// Read via kube.Args().String("key"), .Bool("key"), etc.
	// or via kube.Args().BindArgs(&myStruct).
	Args map[string]interface{} `yaml:"args,omitempty" json:"args,omitempty"`
}

// ManagedResource describes a Kubernetes resource type that a typed extension
// (either a hook or a constructor) will manage.
//
// Orkestra uses this information for two purposes:
//
//  1. RBAC generation — each declared resource results in permissions to
//     get/list/watch/create/update/patch/delete that resource type.
//
//  2. Implicit watch informer — Orkestra automatically starts a watch informer
//     for each declared resource, identical to declaring a watch: entry with
//     all events and owner-reference key resolution. This means:
//
//     - r.client.Get / r.client.List for that type are served from cache
//     - when an owned resource changes, Orkestra enqueues the primary CR
//
//     If you need finer control (custom on:, enqueueGate:, keyFrom:, or index:),
//     declare an explicit watch: entry for that type — it takes priority over
//     the implicit informer from resources:.
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
