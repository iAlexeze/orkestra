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

// Sentinel computation happens here using oldObj and newObj — the result is passed
// to the gate function. Returns nil when no config is registered.
func (f *Factory) computeSentinels(gvkStr string, oldObj, newObj interface{}) map[string]string {
	if f.katalog == nil {
		return nil
	}

	declared := f.katalog.GetPreReconcileSentinels(gvkStr)
	if len(declared) == 0 {
		return nil
	}

	oldDomain, okOld := domain.ToDomainObject(oldObj)
	newDomain, okNew := domain.ToDomainObject(newObj)
	if !okOld || !okNew {
		return nil
	}

	return sentinel.Compute(declared, oldDomain, newDomain)
}

// enqueueAllowed evaluates the enqueue gate for a GVK.
// Objects that fail the gate are dropped before entering the workqueue.
// Events are allowed through when no enqueue gate is configured.
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

// allowEnqueue completes the queue's behaviour evaluation for the event
// and then evaluates the enqueue gate using the event's sentinel context.
// It returns (allowed, evaluated), where evaluated reports whether the
// sentinel-aware enqueue path was applicable.
func (f *Factory) allowEnqueue(
	ctx context.Context, gvkStr string,
	obj interface{}, wq *queue.Workqueue,
	sentinels map[string]string) (bool, bool) {
	domObj, objFound := domain.ToDomainObject(obj)
	if !objFound {
		return false, false
	}
	// Complete queue behaviour evaluation before applying the enqueue gate.
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
