// pkg/kordinator/registry_cross.go
//
// GetInformerByName — satisfies reconciler.KatalogRegistry.
//
// Allows GenericReconciler to look up a sibling CRD's SharedIndexInformer
// for cross-CRD observation without importing pkg/kordinator directly
// (which would create an import cycle).
//
// The lookup is by CRD name (the map key in spec.crds — lowercase).
// The GVK string stored in the registry is used internally; callers
// only need to know the short name.
package kordinator

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

// GetInformerByLabelSelector returns the SharedIndexInformer for a CRD whose
// metadata.labelSelector contain the given key/value pair.
//
// This enables cross‑CRD observation by semantic grouping rather than
// by CRD name. Platform teams can labelSelector CRDs (e.g. "tier=platform",
// "domain=payments") and application‑level logic can reference them
// without knowing the exact CRD name.
//
// Lookup rules:
//   - Match is case‑insensitive on both key and value
//   - Returns the first CRD whose labelSelector contain key=value
//   - Returns nil, false when no CRD matches
//
// This is used by GenericReconciler via the KatalogRegistry interface
// to support cross‑context reads without importing pkg/kordinator
// directly (avoiding import cycles).
func (r *ResourceKatalog) GetInformerByLabelSelector(key, value string) (cache.SharedIndexInformer, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	keyLower := strings.ToLower(key)
	valueLower := strings.ToLower(value)

	for _, entry := range r.entries {
		labels := entry.CRD.LabelSelector
		if labels == nil {
			continue // no labels
		}

		for k, v := range labels {
			if strings.ToLower(k) == keyLower && strings.ToLower(v) == valueLower {
				return entry.Informer, true
			}
		}
	}

	return nil, false
}
