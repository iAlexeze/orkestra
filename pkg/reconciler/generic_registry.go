// pkg/reconciler/generic_registry.go
//
// KatalogRegistry — the interface GenericReconciler needs from the kontroller
// registry to perform zero-API-call cross-CRD observation.
//
// Why a local interface instead of importing pkg/kontroller directly:
//
//	pkg/reconciler → pkg/kontroller would create an import cycle.
//	pkg/kontroller already imports pkg/reconciler (for domain.Reconciler).
//	A local interface breaks the cycle — Go's implicit interface satisfaction
//	means *kontroller.ResourceKatalog satisfies this automatically.
package reconciler

import "k8s.io/client-go/tools/cache"

// KatalogRegistry is the interface GenericReconciler uses to look up sibling
// CRD informers for cross-CRD observation.
// Satisfied by *kontroller.ResourceKatalog without importing that package.
type KatalogRegistry interface {
	// GetInformerByName returns the SharedIndexInformer for a CRD by its
	// lowercase name (the map key in spec.crds).
	// Returns nil, false when the CRD is not registered.
	GetInformerByName(name string) (cache.SharedIndexInformer, bool)
}
