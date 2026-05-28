package types

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── CRDEntry helpers ──────────────────────────────────────────────────────────

// Config returns the OperatorBox configuration for this CRD.
// Safe to call when OperatorBox is the zero value.
func (e CRDEntry) Config() OperatorBoxConfig {
	return e.OperatorBox
}

// IsBuiltInType reports whether this CRD represents a built‑in Kubernetes resource.
// Built‑ins rely on enrichment to populate group, version, plural, and scope.
func (c *CRDEntry) IsBuiltInType() bool {
	return c.IsBuiltIn
}

// SkipStatusSubresource reports whether this CRD belongs to a list to be ignored during status patches.
// This is applied mainly to builtins or if specifically required by the crd through crd.IgnoreStatusPatch
func (c *CRDEntry) SkipStatusSubresource() bool {
	return c.IgnoreStatusPatch
}

// SkipObservedGeneration reports whether this CRD should ignore the
// status.observedGeneration field during readiness checks.
//
// This is applied mainly to built‑in Kubernetes resources or CRDs that do not
// implement observedGeneration semantics. When true, Orkestra will NOT use
// generation-based readiness logic for this CRD.
func (c *CRDEntry) SkipObservedGeneration() bool {
	return c.IgnoreObservedGeneration
}

// ShouldEnrich returns true when the given enrichment target is enabled —
// either via EnrichAll: true or an explicit entry in Enrich.
// Condition gates (when:/anyOf:) are not evaluated here — they are handled
// higher up by ActiveEnrichTargets before the CRDEntry reaches each enricher.
func (c *CRDEntry) ShouldEnrich(target string) bool {
	if c.EnrichAll {
		return true
	}
	for _, t := range c.Enrich {
		if t.Key == target {
			return true
		}
	}
	return false
}

// ActiveEnrichTargets returns the subset of Enrich entries whose when:/anyOf:
// conditions pass for the given data map and evaluator. Unconditional entries
// (no when: or anyOf:) always pass. Called from ReadChildren to pre-filter
// crd.Enrich before dispatching to individual enricher functions.
func (c *CRDEntry) ActiveEnrichTargets(data map[string]interface{}, eval TemplateEvaluator) []EnrichTarget {
	if c.EnrichAll {
		return c.Enrich
	}
	result := make([]EnrichTarget, 0, len(c.Enrich))
	for _, t := range c.Enrich {
		if len(t.When) == 0 && len(t.AnyOf) == 0 {
			result = append(result, t)
			continue
		}
		if EvaluateWhen(data, t.When, t.AnyOf, eval) {
			result = append(result, t)
		}
	}
	return result
}

// IsStatusless reports whether this CRD has no meaningful readiness semantics.
// These resources become "Ready" immediately upon creation.
func (c *CRDEntry) IsStatuslessType() bool {
	return c.IsStatusless
}

// GetRuntimeObjects returns the object and list constructors appropriate for the
// current mode (dynamic or typed). Used by the reconciler to instantiate new
// runtime objects for watches, lists, and reconciliation.
func (c *CRDEntry) GetRuntimeObjects() (runtime.Object, runtime.Object) {
	return c.DynamicModeObject(), c.ListDynamicModeObject()
}

// SetMaxQueueDepth resolves the queue depth for this CRD. If a per‑CRD value is
// provided, it is used; otherwise the Orkestra/Konduktor‑level default is applied.
func (c *CRDEntry) SetMaxQueueDepth(def int) int {
	if c.Queue.MaxQueueDepth == 0 {
		return def
	}
	return c.Queue.MaxQueueDepth
}

// SetWorkers resolves the worker count for this CRD. If a per‑CRD value is
// provided, it is used; otherwise the global default worker count is applied.
func (c *CRDEntry) SetWorkers(def int) int {
	if c.Workers == 0 {
		return def
	}
	return c.Workers
}

// SetResync resolves the resync period for this CRD. If a per‑CRD non‑zero value is
// set, it is used; otherwise the global default resync is applied.
func (c *CRDEntry) SetResync(def time.Duration) time.Duration {
	if c.Resync != 0 {
		return c.Resync
	}
	return def
}

// IsDynamic determines whether this CRD should operate in dynamic mode.
// Resolution order (first match wins):
//  1. mode: dynamic explicitly declared → true
//  2. mode: typed explicitly declared   → false
//  3. APITypes.Location is empty        → true  (no compiled types available)
//  4. APITypes.Location is set          → false (compiled types available)
func (c *CRDEntry) IsDynamic() bool {
	switch c.Mode {
	case CRDModeDynamic:
		return true
	case CRDModeTyped:
		return false
	}
	return c.APITypes.Location == ""
}

// WithHooksDecl returns true if the CRD has a hooks declaration (HookDeclaration).
// Used to determine whether to generate registry entries for the HookRegistry.
// Does not imply anything about Default: true/false — a typed CRD can have hooks
// even when Default: true (generic reconciler) or false (custom reconciler).
func (c *CRDEntry) WithHooksDecl() bool {
	return c.OperatorBox.Hooks != nil && c.OperatorBox.Hooks.Location != ""
}

// WithConstructorDecl returns true if the CRD has a constructor declaration.
// Required when Default: false in the Katalog. The generated registry will
// emit a ReconcilerRegistry entry for this CRD.
func (c *CRDEntry) WithConstructorDecl() bool {
	return c.OperatorBox.ConstructorDecl != nil && c.OperatorBox.ConstructorDecl.Location != ""
}

// WithHookManagedResources reports whether this CRD has hooks that declare
// managed resources for RBAC generation.
func (c *CRDEntry) WithHookManagedResources() bool {
	return c.WithHooksDecl() &&
		len(c.OperatorBox.Hooks.Resources) > 0
}

// WithConstructorManagedResources reports whether this CRD has a constructor
// that declares managed resources for RBAC generation.
func (c *CRDEntry) WithConstructorManagedResources() bool {
	return c.WithConstructorDecl() &&
		len(c.OperatorBox.ConstructorDecl.Resources) > 0
}

// WithAnyManagedResources reports whether hooks or constructor declare resources.
func (c *CRDEntry) WithAnyManagedResources() bool {
	return c.WithHookManagedResources() || c.WithConstructorManagedResources()
}

// HookManagedResources returns the list of managed resources declared under
// the hooks block. Returns nil if hooks are not declared or no resources exist.
func (c *CRDEntry) HookManagedResources() []ManagedResource {
	if !c.WithHooksDecl() {
		return nil
	}
	return c.OperatorBox.Hooks.Resources
}

// ConstructorManagedResources returns the list of managed resources declared
// under the constructor block. Returns nil if constructor is not declared or
// no resources exist.
func (c *CRDEntry) ConstructorManagedResources() []ManagedResource {
	if !c.WithConstructorDecl() {
		return nil
	}
	return c.OperatorBox.ConstructorDecl.Resources
}

// HasTemplates reports whether this CRD declares any declarative hook templates.
// Used by `ork generate` to determine whether to emit generated runtime hooks.
func (c *CRDEntry) HasTemplates() bool {
	rc := c.OperatorBox
	return rc.OnCreate != nil || rc.OnReconcile != nil || rc.OnDelete != nil
}

// GVK returns the fully resolved GroupVersionKind for this CRD. Used for logging,
// routing, and dynamic client operations.
func (c *CRDEntry) GVK() schema.GroupVersionKind {
	return c.GroupVersionKind
}

// GVKString returns the fully resolved GroupVersionKind for this CRD as a string.
func (c *CRDEntry) GVKString() string {
	return c.GroupVersionKind.String()
}

// GVR returns the fully resolved GroupVersionResource for this CRD. Used for
// dynamic client list/watch operations.
func (c *CRDEntry) GVR() schema.GroupVersionResource {
	return c.GroupVersionResource
}

// GVRString returns the fully resolved GroupVersionResource for this CRD as a string.
func (c *CRDEntry) GVRString() string {
	return c.GroupVersionResource.String()
}

// IsEnabled reports whether this CRD is enabled. Defaults to true when omitted.
func (c *CRDEntry) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// IsNamespaced reports whether this CRD is namespaced. Defaults to true unless
// explicitly overridden or determined by enrichment.
func (c *CRDEntry) IsNamespaced() bool {
	if c.Namespaced == nil {
		return true
	}
	return *c.Namespaced
}

// DefaultReconcile reports whether this CRD uses the default reconciler behavior.
// Defaults to true unless explicitly enabled.
func (c *CRDEntry) DefaultReconcile() bool {
	if c.OperatorBox.Default == nil {
		return true
	}
	return *c.OperatorBox.Default
}

// SharedQueue reports whether this CRD uses the shared default workqueue.
// Defaults to false when omitted.
func (c *CRDEntry) SharedQueue() bool {
	if c.Queue.Shared == nil {
		return false
	}
	return *c.Queue.Shared
}

// CustomHooksEnabled reports whether the reconcile behaviour uses custom hooks.
// Defaults to false when omitted.
func (c *CRDEntry) CustomHooksEnabled() bool {
	return c.OperatorBox.Hooks == nil
}

// ConstructorEnabled reports whether the reconcile behaviour uses a constructor.
// Defaults to false when omitted.
func (c *CRDEntry) ConstructorEnabled() bool {
	return c.OperatorBox.Constructor == nil
}

// IsHealthEnabled reports whether the /health endpoint is enabled for this CRD.
// Defaults to true when omitted.
func (c *CRDEntry) IsHealthEnabled() bool {
	if c.Endpoints.Health == nil {
		return true
	}
	return *c.Endpoints.Health
}

// IsInfoEnabled reports whether the /info endpoint is enabled for this CRD.
// Defaults to true when omitted.
func (c *CRDEntry) IsInfoEnabled() bool {
	if c.Endpoints.Info == nil {
		return true
	}
	return *c.Endpoints.Info
}

// IsEnabledAllEndpoints reports whether the all endpoints are disabled for this CRD.
// Defaults to false when omitted.
func (c *CRDEntry) IsEnabledAllEndpoints() bool {
	if c.Endpoints.Enabled == nil {
		return true
	}
	return *c.Endpoints.Enabled
}

// GetDependencies returns the dependency names for this CRD in sorted order.
func (c *CRDEntry) GetDependencies() []string {
	return c.DependsOn.Names()
}

// Returns true when either validation or mutation rules are declared.
// Used to decide whether to create the endpoints and/or populate the admission block in the health response.
// Even when ENABLE_ADMISSION_WEBHOOK=true
func (c *CRDEntry) HasValidationOrMutationRules() bool {
	return c.HasValidationRules() || c.HasMutationRules()
}

// Separate helpers for hasMutationRules and hasValidationRules
func (c *CRDEntry) HasMutationRules() bool {
	if c.Mutation == nil {
		return false
	}
	return len(c.Mutation.Rules) > 0
}

// ShouldMutateFirst reports whether this CRD prefers mutation first or not
//
// Default is true
func (c *CRDEntry) ShouldMutateFirst() bool {
	if c.Mutation == nil {
		return true
	}
	return c.Mutation.MutateFirst
}

// HasValidationRules reports whether this CRD has any validation behavior configured
func (c *CRDEntry) HasValidationRules() bool {
	if c.Validation == nil {
		return false
	}
	return len(c.Validation.Rules) > 0
}

// HasProviders reports whether this CRD declares any provider blocks.
func (c *CRDEntry) HasProviders() bool {
	return len(c.OperatorBox.ProviderBlocks) > 0
}

// AutoscaleEnabled reports whether this CRD declares the autoscale block
func (c *CRDEntry) AutoscaleEnabled() bool {
	return c.OperatorBox.Autoscale != nil
}

// HasRollbackRules reports whether this CRD has any rollback behavior configured —
// either via an explicit rollback: block or the rollBackOnError: true shorthand.
func (c *CRDEntry) HasRollbackRules() bool {
	return c.OperatorBox.Rollback != nil || c.OperatorBox.RollBackOnError
}

// HasCRDFile reports whether this CRDEntry declares a CRD file
// to be auto-applied before the operator starts.
func (c *CRDEntry) HasCRDFile() bool {
	return c != nil && c.CRDFile != ""
}

// HasCRFiles reports whether this CRDEntry declares CR YAML files
// to be applied before the runtime starts.
func (c *CRDEntry) HasCRFiles() bool {
	return c != nil && len(c.CRFiles) > 0
}

// HasSetup reports whether this CRDEntry declares any setup work
// to be done before Orkestra starts.
func (c *CRDEntry) HasSetup() bool {
	if c == nil || c.Setup == nil {
		return false
	}
	return len(c.Setup.Apply) > 0 || len(c.Setup.Helm) > 0 || len(c.Setup.Wait) > 0
}

// NotificationEnabled reports whether this CRD declares the notification block
// Enabled by default
func (c *CRDEntry) IsNotificationEnabled() bool {
	if c.NotificationEnabled == nil {
		return true
	}
	return *c.NotificationEnabled
}

// ValidateMetricField returns an error if the field is not a known autoscale metric.
func (c *CRDEntry) ValidateMetricField(field string) error {
	known := map[string]struct{}{
		"metrics.workersBusyPercent":     {},
		"metrics.workersIdlePercent":     {},
		"metrics.queueDepth":             {},
		"metrics.reconcileDurationP95Ms": {},
		"metrics.errorRatePercent":       {},
	}

	if _, ok := known[field]; !ok {
		return fmt.Errorf(
			"unknown autoscale metric field %q — valid fields: %s",
			field,
			strings.Join([]string{
				"metrics.workersBusyPercent",
				"metrics.workersIdlePercent",
				"metrics.queueDepth",
				"metrics.reconcileDurationP95Ms",
				"metrics.errorRatePercent",
			}, ", "),
		)
	}

	return nil
}

// HasAutoscaleProfile reports whether this crd defined autoscale profile
func (c *CRDEntry) HasAutoscaleProfile() bool {
	return c.OperatorBox.Autoscale != nil && c.OperatorBox.Autoscale.Profile != ""
}

// AutoScaleProfile returns the string value of the autoscale profile
func (c *CRDEntry) AutoScaleProfile() string {
	return c.OperatorBox.Autoscale.Profile
}

// UpdateCRDCaBundle reports whether this CRD declares an updateCRD field
// Used to update the crd when certificate is autogenerted by orkestra
func (c *CRDEntry) UpdateCRDCaBundle() bool {
	if c.Conversion == nil {
		return false
	}
	return c.Conversion.UpdateCRD
}

// InvolvedInConversion reports whether this CRD is involved in version conversion.
// A CRD is involved when it either declares conversion paths (the CRD that hosts
// the /convert logic) or explicitly opts in with participant: true (the stable/
// storage-version CRD on the other side of the pair).
func (c *CRDEntry) InvolvedInConversion() bool {
	if c.Conversion == nil {
		return false
	}
	return len(c.Conversion.Paths) > 0 || c.Conversion.Participant
}

// HasNamespaceRules reports whether this CRD declares any namespace rules.
func (c *CRDEntry) HasNamespaceRules() bool {
	return len(c.AllowedNamespaces) > 0 || len(c.RestrictedNamespaces) > 0
}

// HasOnCreate reports whether this CRD declares any onCreate hooks.
func (c *CRDEntry) HasOnCreate() bool {
	return c.OperatorBox.OnCreate != nil
}

// HasOnReconcile reports whether this CRD declares any onReconcile hooks.
func (c *CRDEntry) HasOnReconcile() bool {
	return c.OperatorBox.OnReconcile != nil
}

// HasOnDelete reports whether this CRD declares any onDelete hooks.
func (c *CRDEntry) HasOnDelete() bool {
	return c.OperatorBox.OnDelete != nil
}

// HasStatusFields reports whether this CRD declares any status fields.
func (c *CRDEntry) HasStatusFields() bool {
	return c.OperatorBox.Status != nil && c.OperatorBox.Status.HasFields()
}

// AllRestrictedNamespaces returns a list of restricted namespaces for this crd
func (c *CRDEntry) AllRestrictedNamespaces() RestrictedNamespaces {
	return c.RestrictedNamespaces
}

// AllAllowedNamespaces returns a list of allowed namespaces for this crd
func (c *CRDEntry) AllAllowedNamespaces() AllowedNamespaces {
	return c.AllowedNamespaces
}

// AllowedNamespacesOnly reports if only allowedNamespaces is defined for this crd.
func (c *CRDEntry) AllowedNamespacesOnly() bool {
	return len(c.AllowedNamespaces) > 0 && len(c.RestrictedNamespaces) == 0
}

// RestrictedNamespacesOnly reports if only restrictedNamespaces is defined for this crd.
func (c *CRDEntry) RestrictedNamespacesOnly() bool {
	return len(c.RestrictedNamespaces) > 0 && len(c.AllowedNamespaces) == 0
}

// HasAllowedNamespaces reports if allowedNamespaces is defined for this crd.
func (c *CRDEntry) HasAllowedNamespaces() bool {
	return len(c.AllowedNamespaces) > 0
}

// HasRestrictedNamespaces reports if restrictedNamespaces is defined for this crd.
func (c *CRDEntry) HasRestrictedNamespaces() bool {
	return len(c.RestrictedNamespaces) > 0
}

// IsValidServiceType reports whether the provided service type is valid.
// Accepted values (case‑insensitive):
//   - ClusterIP
//   - NodePort
//   - LoadBalancer
func IsValidServiceType(t string) bool {
	switch strings.ToLower(t) {
	case "", "clusterip", "nodeport", "loadbalancer":
		return true
	default:
		return false
	}
}

// IsValidProtocol reports whether the provided protocol is valid.
// Accepted values (case‑insensitive):
//   - TCP
//   - UDP
//   - SCTP
func IsValidProtocol(p string) bool {
	switch strings.ToUpper(p) {
	case "", "TCP", "UDP", "SCTP":
		return true
	default:
		return false
	}
}
