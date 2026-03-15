// pkg/config/pkg/kontroller/registry.go
package kontroller

import (
	"sync"

	"github.com/ialexeze/orkestra/domain"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/client-go/tools/cache"
)

type RegistryEntry struct {
	CRD               orktypes.CRDEntry
	Informer          cache.SharedIndexInformer
	ReconcilerFactory func() domain.Reconciler // factory lives here
	DegradeThreshold  int
}

type ResourceKatalog struct {
	mu      sync.Mutex
	entries map[string]RegistryEntry
}

func NewKontrollerRegistry() *ResourceKatalog {
	return &ResourceKatalog{
		entries: make(map[string]RegistryEntry),
	}
}

func (r *ResourceKatalog) Register(
	gvk string,
	crd orktypes.CRDEntry,
	inf cache.SharedIndexInformer,
	rec func() domain.Reconciler,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries[gvk] = RegistryEntry{
		CRD:               crd,
		Informer:          inf,
		ReconcilerFactory: rec,
	}
}

func (r *ResourceKatalog) Unregister(gvk string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.entries, gvk)
}

func (r *ResourceKatalog) Get(gvk string) (RegistryEntry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.entries[gvk]
	return entry, ok
}

func (r *ResourceKatalog) ListGVKs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var gvkList []string
	for gvk := range r.entries {
		gvkList = append(gvkList, gvk)
	}
	return gvkList
}

func (r *ResourceKatalog) GetWorkers(gvk string, defaultWorkers int) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.entries[gvk]
	if !ok {
		return defaultWorkers
	}
	return entry.CRD.Workers
}

func (r *ResourceKatalog) Entries() map[string]RegistryEntry {
	return r.entries
}
