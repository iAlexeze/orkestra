// pkg/kontroller/registry_cross.go
//
// GetInformerByName — satisfies reconciler.KatalogRegistry.
//
// Allows GenericReconciler to look up a sibling CRD's SharedIndexInformer
// for cross-CRD observation without importing pkg/kontroller directly
// (which would create an import cycle).
//
// The lookup is by CRD name (the map key in spec.crds — lowercase).
// The GVK string stored in the registry is used internally; callers
// only need to know the short name.
package kontroller

import (
	"strings"

	"k8s.io/client-go/tools/cache"
)

// GetInformerByName returns the SharedIndexInformer for a CRD by its lowercase
// name (the spec.crds map key: "pipeline", "database", "website").
//
// The registry stores entries keyed by GVK string
// (e.g. "demo.orkestra.io/v1alpha1, Kind=Pipeline"). This method searches
// by Kind name match (case-insensitive) so callers use simple names.
//
// Returns nil, false when no CRD with that name is registered.
func (r *ResourceKatalog) GetInformerByName(name string) (cache.SharedIndexInformer, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nameLower := strings.ToLower(name)

	for _, entry := range r.entries {
		if strings.ToLower(entry.CRD.Name) == nameLower {
			return entry.Informer, true
		}
	}
	return nil, false
}
