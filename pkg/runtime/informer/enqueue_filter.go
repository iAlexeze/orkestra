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
	"context"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/runtime/queue"
	"github.com/orkspace/orkestra/pkg/runtime/sentinel"
)

// RegisterEnqueueFilter registers sentinel configuration for a GVK.
// declared is the list of sentinel names from preReconcile.sentinels;
//
// Simplified filter registration
func (f *Factory) RegisterEnqueueFilter(gvkStr string, declared []string) {
	if declared == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enqueueFilters[gvkStr] = &enqueueFiltersCfg{declared: declared}
}

// Sentinel computation happens here using oldObj and newObj — the result is passed
// to the gate function. Returns nil when no config is registered.
func (f *Factory) computeSentinels(gvkStr string, oldObj, newObj interface{}) map[string]string {
	f.mu.RLock()
	cfg, ok := f.enqueueFilters[gvkStr]
	f.mu.RUnlock()
	if !ok {
		return nil
	}

	oldDomain, okOld := domain.ToDomainObject(oldObj)
	newDomain, okNew := domain.ToDomainObject(newObj)
	if !okOld || !okNew {
		return nil
	}

	return sentinel.Compute(cfg.declared, oldDomain, newDomain)
}

// enqueueAllowed evaluates the registered enqueue filter for the given GVK.
// Returns true when the event should proceed to the queue (no filter, or filter passes).
// Returns false when the event should be silently dropped.
//
// Unwraps cache.DeletedFinalStateUnknown tombstones and asserts to domain.Object
// before calling the registered function — works for both typed and dynamic CRDs.
func (f *Factory) enqueueAllowed(ctx context.Context, gvkStr string, obj interface{}) bool {
	// Unwrap tombstone — produced when a deletion event arrives after the object
	// has already been removed from the cache.
	domObj, ok := domain.ToDomainObject(obj)
	if !ok {
		return true // can't evaluate — let it through
	}

	if f.katalog != nil && !f.katalog.EvaluateEnqueueFilter(ctx, gvkStr, domObj, f.cs, nil) {
		logger.Debug().
			Str("gvk", gvkStr).
			Str("name", domObj.GetName()).
			Msg("informer: event dropped — enqueue filter")
		return false
	}
	return true
}

// enqueueAllowedWithSentinelAndBehaviour evaluates the registered enqueue filter for the given GVK.
// it starts with completing the first tier evaluation started at the workqueue for this CRD.
// Applies behaviour in tier 2 of the queue behaviour evauation, and then evaluates the sentinel-aware
// filter. Returns (true, true) when both evaluation passes.
func (f *Factory) enqueueAllowedWithSentinelAndBehaviour(
	ctx context.Context, gvkStr string,
	obj interface{}, wq *queue.Workqueue,
	sentinels map[string]string) (bool, bool) {
	domObj, objFound := domain.ToDomainObject(obj)
	if !objFound {
		return false, false
	}
	// Honor the response from pkg/queue and drop any failing items before update considerations
	// This is the last step in completing what the queue started that it couldn't finish and
	// delegated to the informer here - so done first with domain.Katalog which
	if f.katalog != nil && wq != nil && wq.NeedsBehaviourEval() {
		if !f.katalog.EvaluateQueueBehaviourConditions(ctx, gvkStr, domObj, sentinels) {
			return false, true
		}
	}

	// Evaluate enqueue filters
	if f.katalog == nil {
		return true, true
	}
	return f.katalog.EvaluateEnqueueFilter(ctx, gvkStr, domObj, f.cs, sentinels), true
}
