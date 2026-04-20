package types

import (
	"fmt"
	"strings"

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

// SkipObservedGeneration reports whether this CRD belongs to a list to be ignored during status patches.
// This is applied mainly to builtins or if specifically required by the crd through crd.IgnoreStatusPatch
func (c *CRDEntry) SkipObservedGeneration() bool {
	return c.IgnoreObservedGeneration
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

// IsCritical reports whether this CRD is marked as critical. Critical CRDs may
// influence startup ordering or health evaluation. Defaults to false.
// func (c *CRDEntry) IsCritical() bool {
// 	if c.Critical == nil {
// 		return false
// 	}
// 	return *c.Critical
// }

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

// DefaultQueue reports whether this CRD uses the default queue configuration.
// Defaults to false when omitted.
func (c *CRDEntry) DefaultQueue() bool {
	if c.Queue.Default == nil {
		return false
	}
	return *c.Queue.Default
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
	return len(c.Validation.Rules) > 0 || len(c.Mutation.Rules) > 0
}

// Separate helpers for hasMutationRules and hasValidationRules
func (c *CRDEntry) HasMutationRules() bool {
	if c.Mutation == nil {
		return false
	}
	return len(c.Mutation.Rules) > 0
}

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

// HasRollbackRules reports whether this CRD declares a rollback block.
func (c *CRDEntry) HasRollbackRules() bool {
	return c.OperatorBox.Rollback != nil
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

// HasNamespaceRules reports whether this CRD declares any namespace rules.
func (c *CRDEntry) HasNamespaceRules() bool {
	return len(c.AllowedNamespaces) > 0 || len(c.RestrictedNamespaces) > 0
}

// HasOnCreate reports whether this CRD declares any onCreate hooks.
func (c *CRDEntry) HasOnCreate() bool {
	return c.OperatorBox.OnCreate == nil
}

// HasOnReconcile reports whether this CRD declares any onReconcile hooks.
func (c *CRDEntry) HasOnReconcile() bool {
	return c.OperatorBox.OnReconcile == nil
}

// HasOnDelete reports whether this CRD declares any onDelete hooks.
func (c *CRDEntry) HasOnDelete() bool {
	return c.OperatorBox.OnDelete == nil
}

// HasAnyHooks reports whether this CRD declares any onCreate, onReconcile, or onDelete hooks.
func (c *CRDEntry) HasAnyHooks() bool {
	return c.HasOnCreate() || c.HasOnReconcile() || c.HasOnDelete()
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
