// pkg/config/pkg/controller/registry.go
package controller

import (
	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/registry"
	"k8s.io/client-go/tools/cache"
)

type RegistryEntry struct {
	CRD        registry.CRDInfo
	Informer   cache.SharedIndexInformer
	Reconciler domain.Reconciler
}

type ResourceRegistry struct {
	entries map[string]RegistryEntry
}

func NewControllerRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		entries: make(map[string]RegistryEntry),
	}
}

func (r *ResourceRegistry) Register(
	gvk string,
	crd registry.CRDInfo,
	inf cache.SharedIndexInformer,
	rec domain.Reconciler,
) {
	r.entries[gvk] = RegistryEntry{
		CRD:        crd,
		Informer:   inf,
		Reconciler: rec,
	}
}

func (r *ResourceRegistry) Entries() map[string]RegistryEntry {
	return r.entries
}
