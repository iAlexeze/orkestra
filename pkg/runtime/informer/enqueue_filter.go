// pkg/informer/enqueue_filter.go
//
// EnqueueFilter — pre-enqueue condition gate for the informer factory.
//
// Sits after the namespace filter (Tier 2) and before the queue.
// When a CRD declares operatorBox.preReconcile.filter:, a condition
// function is registered here. handleEvent evaluates it before calling
// wq.Enqueue. Objects that fail the filter are dropped silently —
// no queue pressure, no kordinator overhead.
//
// Works for both dynamic (*unstructured.Unstructured) and typed CRDs —
// both implement domain.Object, which the template resolver already handles
// via objectToMap (JSON round-trip for typed, direct map for unstructured).
package informer

import (
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/runtime/sentinel"
	"k8s.io/client-go/tools/cache"
)

// RegisterEnqueueFilter stores a pre-enqueue condition gate for a GVK.
// fn returns true when the object should be enqueued, false to drop it.
// Called during CRD registration, before informers are started.
func (f *Factory) RegisterEnqueueFilter(gvkStr string, fn func(domain.Object) bool) {
	if fn == nil {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.enqueueFilters[gvkStr] = fn
}

// RegisterUpdateEnqueueFilter registers sentinel configuration for a GVK.
// declared is the list of sentinel names from preReconcile.sentinels; gate
// decides whether to enqueue and receives the already-computed sentinel values.
// Sentinel computation (old vs new comparison) happens inside handleUpdateEvent —
// the caller only passes configuration, not computation.
// Only one config per GVK — subsequent calls overwrite.
func (f *Factory) RegisterUpdateEnqueueFilter(gvkStr string, declared []string, gate func(domain.Object, map[string]string) bool) {
	if gate == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateFilters[gvkStr] = &updateFilterCfg{declared: declared, gate: gate}
}

// updateEnqueueAllowed evaluates the registered update config for the given GVK.
// Sentinel computation happens here using oldObj and newObj — the result is passed
// to the gate function. Returns (true, nil, false) when no config is registered.
func (f *Factory) updateEnqueueAllowed(gvkStr string, oldObj, newObj interface{}) (bool, map[string]string, bool) {
	f.mu.RLock()
	cfg, ok := f.updateFilters[gvkStr]
	f.mu.RUnlock()
	if !ok {
		return true, nil, false
	}

	oldDomain, okOld := toDomainObject(oldObj)
	newDomain, okNew := toDomainObject(newObj)
	if !okOld || !okNew {
		return true, nil, true
	}
	sentinels := sentinel.Compute(cfg.declared, oldDomain, newDomain)
	allowed := cfg.gate(newDomain, sentinels)
	return allowed, sentinels, true
}

// toDomainObject unwraps a cache tombstone and asserts to domain.Object.
func toDomainObject(obj interface{}) (domain.Object, bool) {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	d, ok := obj.(domain.Object)
	return d, ok
}

// enqueueAllowed evaluates the registered enqueue filter for the given GVK.
// Returns true when the event should proceed to the queue (no filter, or filter passes).
// Returns false when the event should be silently dropped.
//
// Unwraps cache.DeletedFinalStateUnknown tombstones and asserts to domain.Object
// before calling the registered function — works for both typed and dynamic CRDs.
func (f *Factory) enqueueAllowed(gvkStr string, obj interface{}) bool {
	f.mu.RLock()
	fn, ok := f.enqueueFilters[gvkStr]
	f.mu.RUnlock()

	if !ok {
		return true
	}

	// Unwrap tombstone — produced when a deletion event arrives after the object
	// has already been removed from the cache.
	domObj, ok := toDomainObject(obj)
	if !ok {
		return true // can't evaluate — let it through
	}

	if !fn(domObj) {
		logger.Debug().
			Str("gvk", gvkStr).
			Str("name", domObj.GetName()).
			Msg("informer: event dropped — enqueue filter")
		return false
	}
	return true
}
