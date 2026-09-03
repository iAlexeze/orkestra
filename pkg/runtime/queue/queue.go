package queue

import (
	"context"
	"sync/atomic"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// For controller queueing
type QueueItem struct {
	Key string
	GVK string
	// SentinelMap carries event-time sentinel values computed in the informer's
	// UpdateFunc (oldObj vs newObj). Both enqueueGate and reconcileGate share the
	// same preReconcile context — reconcileGate rebuilds the resolver from this map
	// after dequeue, when oldObj is no longer available.
	// nil when no preReconcile.sentinels are declared (common case — deduplication
	// behaviour is unchanged). Non-nil items dedup by pointer identity, meaning
	// each sentinel-bearing enqueue is treated as a distinct work item.
	SentinelMap *map[string]string
}

type Workqueue struct {
	name         string
	queue        workqueue.TypedRateLimitingInterface[QueueItem]
	queueCfg     domain.Workqueue
	evaluateCond *BehaviourEval // whether or not to evaluate additional conditions in informer before enqueuing
	maxDepth     atomic.Int32   // 0 = unlimited; enforced atomically in Enqueue
	started      atomic.Bool
}

type BehaviourEval struct {
	OnLimit     atomic.Bool
	OnThreshold atomic.Bool
}

type WorkqueueInfo struct {
	Depth        int
	Limit        int
	DepthReached bool
}

func NewWorkqueue(name string) *Workqueue {
	if name == "" {
		name = "default workqueue"
	}
	return &Workqueue{
		name:  name,
		queue: workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[QueueItem]()),
	}
}

// Queue is the TypedRateLimitingInterface used by all orkestra operators.
// It is an interface that rate limits items being added to the queue.
//
// https://pkg.go.dev/k8s.io/client-go@v0.36.1/util/workqueue#TypedRateLimitingInterface
func (q *Workqueue) Queue() workqueue.TypedRateLimitingInterface[QueueItem] {
	if q == nil {
		return nil
	}
	return q.queue
}

// Enqueue adds the object's key to the workqueue.
// When a non-zero maxDepth is set, new items are dropped (with a warning)
// once the queue is at or beyond that limit and the effective configured
// behaviour is drop. Items already in the queue are
// not evicted — only incoming enqueues are rejected.
func (q *Workqueue) Enqueue(obj interface{}, gvk string) {
	// Handle tombstone (deleted objects)
	obj = domain.UnwrapCacheTombstone(obj)

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Error().Err(err).Str("gvk", gvk).Msg("enqueue: failed to get key")
		return
	}

	item := QueueItem{Key: key, GVK: gvk}

	if ok := q.evaluateQueueBehaviour(item); !ok {
		return
	}
	q.queue.Add(item)
}

// EnqueueKey adds a pre-computed key directly to the workqueue.
// Used when the key is resolved from an ownerReference or another indirect source
// rather than from the object itself.
func (q *Workqueue) EnqueueKey(key, gvk string) {
	item := QueueItem{Key: key, GVK: gvk}
	q.queue.Add(item)
	if ok := q.evaluateQueueBehaviour(item); !ok {
		return
	}
}

// EnqueueWithSentinels adds a key to the workqueue alongside the sentinel values
// computed at event time (oldObj vs newObj in the informer UpdateFunc).
// The sentinel map is passed as a pointer so the item remains comparable — two
// sentinel-bearing enqueues for the same key are treated as distinct items.
func (q *Workqueue) EnqueueWithSentinels(obj interface{}, gvk string, sentinels map[string]string) {
	obj = domain.UnwrapCacheTombstone(obj)

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Error().Err(err).Str("gvk", gvk).Msg("enqueue: failed to get key")
		return
	}

	item := QueueItem{Key: key, GVK: gvk, SentinelMap: &sentinels}
	if ok := q.evaluateQueueBehaviour(item); !ok {
		return
	}

	q.queue.Add(item)
	logger.Debug().Str("key", key).Str("gvk", gvk).Msg("enqueued")
}

// SetQueueDepth adjusts the queue depth limit at runtime.
// 0 means unlimited. Safe to call concurrently with Enqueue.
func (q *Workqueue) SetQueueDepth(n int) {
	q.maxDepth.Store(int32(n))
}

// Methods to implement the komponent interface
var _ domain.Komponent = (*Workqueue)(nil)

// Start is called by orkestra.Start() after all komoonents are registered
func (q *Workqueue) Start(ctx context.Context) error {
	logger.Debug().Str("name", q.name).Msg("workqueue started")
	q.started.Store(true)
	return nil
}

// Started returns true if default workqueue has started
// Used by orkestra for status check
func (q *Workqueue) Started() bool { return q.started.Load() }

// Shutdown drains the default workqueue
// This is called by orkestra.Shutdown() for graceful degradation
func (q *Workqueue) Shutdown(ctx context.Context) {
	if q.queue != nil {
		q.queue.ShutDown()
	}
}

// GetWithContext returns an item from the work queue.
// Context (e.g., timeout, cancellation).
// Future use
func (q *Workqueue) GetWithContext(ctx context.Context) (QueueItem, bool) {
	// Channel to receive the result of the blocking Get() call
	type result struct {
		item     QueueItem
		shutdown bool
	}
	resultCh := make(chan result, 1)

	// Run the blocking Get() in a goroutine
	go func() {
		item, shutdown := q.queue.Get()
		resultCh <- result{item, shutdown}
	}()

	// Wait for either context cancellation or a result
	select {
	case <-ctx.Done():
		// Context cancelled. The Get() goroutine is still running and will
		// eventually send to resultCh, but we're no longer listening.
		// We must drain that result to prevent a goroutine leak.
		go func() {
			<-resultCh
			// Optionally re-queue the item if we want to preserve it,
			// but since we're deactivating, we can just discard.
		}()
		return QueueItem{}, true // shutdown == true signals exit
	case res := <-resultCh:
		return res.item, res.shutdown
	}
}

// Name returns the name of the default workqueue
func (q *Workqueue) Name() string {
	return q.name
}

// Depth returns the length of the default workqueue
func (q *Workqueue) Depth() int {
	return q.queue.Len()
}

// IsUnlimited returns true when maxDepth is zero
func (q *Workqueue) IsUnlimited() bool {
	return q.queue.Len() == 0
}

// DepthReached returns true when current depth is greater or equals maxDepth
func (q *Workqueue) DepthReached() bool {
	return q.Depth() >= q.MaxDepth()
}

// QueueInfo returns the current workqueue info
func (q *Workqueue) QueueInfo() (info *WorkqueueInfo) {
	if q == nil {
		return nil
	}
	info = &WorkqueueInfo{
		Depth:        q.Depth(),
		Limit:        q.MaxDepth(),
		DepthReached: q.Depth() >= q.MaxDepth(),
	}
	return info
}

// BehaviourCond returns the pending behaviour evaluation flags for this queue item.
func (q *Workqueue) BehaviourCond() *BehaviourEval {
	if q == nil || q.evaluateCond == nil {
		return nil
	}
	return q.evaluateCond
}

// OnLimitCond reports whether onLimit when/or conditions should be evaluated by the informer.
func (q *Workqueue) OnLimitCond() bool {
	cond := q.BehaviourCond()
	if cond == nil {
		return false
	}
	return cond.OnLimit.Load()
}

// OnThresholdCond reports whether onThreshold when/or conditions should be evaluated by the informer.
func (q *Workqueue) OnThresholdCond() bool {
	cond := q.BehaviourCond()
	if cond == nil {
		return false
	}
	return cond.OnThreshold.Load()
}

// NeedsBehaviourEval reports true if there are pending behaviour conditions delegated to the informer.
func (q *Workqueue) NeedsBehaviourEval() bool {
	return q.OnLimitCond() || q.OnThresholdCond()
}

// MaxDepth returns the current maximum queue depth (0 = unlimited).
func (q *Workqueue) MaxDepth() int { return int(q.maxDepth.Load()) }
