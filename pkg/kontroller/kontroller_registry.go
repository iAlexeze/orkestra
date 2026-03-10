// pkg/config/pkg/kontroller/registry.go
package kontroller

import (
	"sync"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/initialize"
	"k8s.io/client-go/tools/cache"
)

type RegistryEntry struct {
	CRD        initialize.CRDEntry
	Informer   cache.SharedIndexInformer
	Reconciler domain.Reconciler
}

type ResourceRegistry struct {
	mu      sync.Mutex
	entries map[string]RegistryEntry
}

func NewKontrollerRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		entries: make(map[string]RegistryEntry),
	}
}

func (r *ResourceRegistry) Register(
	gvk string,
	crd initialize.CRDEntry,
	inf cache.SharedIndexInformer,
	rec domain.Reconciler,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries[gvk] = RegistryEntry{
		CRD:        crd,
		Informer:   inf,
		Reconciler: rec,
	}
}

func (r *ResourceRegistry) Unregister(gvk string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.entries, gvk)
}

func (r *ResourceRegistry) Get(gvk string) (RegistryEntry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.entries[gvk]
	return entry, ok
}

func (r *ResourceRegistry) ListGVKs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var gvkList []string
	for gvk := range r.entries {
		gvkList = append(gvkList, gvk)
	}
	return gvkList
}

func (r *ResourceRegistry) GetWorkers(gvk string, defaultWorkers int) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.entries[gvk]
	if !ok {
		return defaultWorkers
	}
	return entry.CRD.Workers
}

func (r *ResourceRegistry) Entries() map[string]RegistryEntry {
	return r.entries
}
