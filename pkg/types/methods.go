package orktypes

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── CRDEntry helpers ──────────────────────────────────────────────────────────
// IsBuiltInType reports whether this CRD represents a built‑in Kubernetes resource.
// Built‑ins rely on enrichment to populate group, version, plural, and scope.
func (c *CRDEntry) IsBuiltInType() bool {
	return c.IsBuiltIn
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
	rc := c.ReconcilerConfig
	return rc.OnCreate != nil || rc.OnReconcile != nil || rc.OnDelete != nil
}

// GVK returns the fully resolved GroupVersionKind for this CRD. Used for logging,
// routing, and dynamic client operations.
func (c *CRDEntry) GVK() schema.GroupVersionKind {
	return c.GroupVersionKind
}

// GVR returns the fully resolved GroupVersionResource for this CRD. Used for
// dynamic client list/watch operations.
func (c *CRDEntry) GVR() schema.GroupVersionResource {
	return c.GroupVersionResource
}

// NewObject returns a new instance of the runtime object for this CRD.
// In dynamic mode, an unstructured.Unstructured is returned.
// In typed mode, the compiled Go type is returned if available.
func (c *CRDEntry) NewObject() runtime.Object {
	if c.IsDynamic() {
		return &unstructured.Unstructured{}
	}
	if c.TypedModeObject == nil {
		return c.DynamicModeObject()
	}
	return c.TypedModeObject
}

// NewList returns a new list object for this CRD. Mirrors NewObject but returns
// the list variant (typed or unstructured).
func (c *CRDEntry) NewList() runtime.Object {
	if c.IsDynamic() {
		return &unstructured.UnstructuredList{}
	}
	if c.ListTypedModeObject == nil {
		return c.ListDynamicModeObject()
	}
	return c.ListTypedModeObject
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
func (c *CRDEntry) IsCritical() bool {
	if c.Critical == nil {
		return false
	}
	return *c.Critical
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
	if c.ReconcilerConfig.Default == nil {
		return true
	}
	return *c.ReconcilerConfig.Default
}

// DefaultQueue reports whether this CRD uses the default queue configuration.
// Defaults to true unless explicitly disabled.
func (c *CRDEntry) DefaultQueue() bool {
	if c.Queue.Default == nil {
		return true
	}
	return *c.Queue.Default
}

// IsHealthEnabled reports whether the /healthz endpoint is enabled for this CRD.
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

// IsValidationEnabled reports whether the /validate endpoint is enabled for this CRD.
// Defaults to true when omitted.
func (c *CRDEntry) IsValidationEnabled() bool {
	if c.Endpoints.Validation == nil {
		return true
	}
	return *c.Endpoints.Validation
}

// IsEnabledAllEndpoints reports whether the all endpoints are disabled for this CRD.
// Defaults to false when omitted.
func (c *CRDEntry) IsEnabledAllEndpoints() bool {
	if c.Endpoints.Enabled == nil {
		return true
	}
	return *c.Endpoints.Enabled
}
