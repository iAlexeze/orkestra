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
	if ts, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = ts.Obj
	}

	domObj, ok := obj.(domain.Object)
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
