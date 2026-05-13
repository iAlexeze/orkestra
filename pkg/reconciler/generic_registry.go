// pkg/reconciler/generic_registry.go
//
// KatalogRegistry — the interface GenericReconciler needs from the Kordinator
// registry to perform zero-API-call cross-CRD observation.
//
// Why a local interface instead of importing pkg/kordinator directly:
//
//	pkg/reconciler → pkg/kordinator would create an import cycle.
//	pkg/kordinator already imports pkg/reconciler (for domain.Reconciler).
//	A local interface breaks the cycle — Go's implicit interface satisfaction
//	means *kordinator.ResourceKatalog satisfies this automatically.
package reconciler

import "k8s.io/client-go/tools/cache"

// KatalogRegistry is the interface GenericReconciler uses to look up sibling
// CRD informers for cross-CRD observation.
// Satisfied by *kordinator.ResourceKatalog without importing that package.
type KatalogRegistry interface {
	// GetInformerByName returns the SharedIndexInformer for a CRD by its
	// lowercase name (the map key in spec.crds).
	// Returns nil, false when the CRD is not registered.
	GetInformerByName(name string) (cache.SharedIndexInformer, bool)

	// GetInformerByLabelSelector returns the SharedIndexInformer for a CRD whose
	// metadata.labelSelector contain the given key/value pair.
	//
	// This enables semantic grouping of CRDs for cross-CRD observation
	// (e.g. "tier=platform", "domain=payments") without requiring callers
	// to know the CRD's short name. Lookup is case-insensitive on both
	// key and value. Returns nil, false when no CRD matches.
	GetInformerByLabel(key, value string) (cache.SharedIndexInformer, bool)
}
