package queue

import (
	"context"
	"sync/atomic"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// For controller queueing
type QueueItem struct {
	Key string
	GVK string
}

type Workqueue struct {
	name          string
	Queue         workqueue.TypedRateLimitingInterface[QueueItem]
	started       atomic.Bool
	maxQueueDepth atomic.Int32 // 0 = unlimited; enforced atomically in Enqueue
}

func NewWorkqueue() *Workqueue {
	return &Workqueue{
		name:  "default workqueue",
		Queue: workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[QueueItem]()),
	}
}

// Enqueue adds the object's key to the workqueue.
// When a non-zero maxQueueDepth is set, new items are dropped (with a warning)
// once the queue is at or beyond that limit. Items already in the queue are
// not evicted — only incoming enqueues are rejected.
func (q *Workqueue) Enqueue(obj interface{}, gvk string) {
	// Handle tombstone (deleted objects)
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Error().Err(err).Str("gvk", gvk).Msg("enqueue: failed to get key")
		return
	}

	if limit := q.maxQueueDepth.Load(); limit > 0 && int32(q.Queue.Len()) >= limit {
		logger.Warn().
			Str("key", key).
			Str("gvk", gvk).
			Int32("limit", limit).
			Int("depth", q.Queue.Len()).
			Msg("enqueue: queue depth limit reached — item dropped")
		return
	}

	q.Queue.Add(QueueItem{Key: key, GVK: gvk})
	logger.Debug().Str("key", key).Str("gvk", gvk).Msg("enqueued")
}

// SetMaxQueueDepth adjusts the queue depth limit at runtime.
// 0 means unlimited. Safe to call concurrently with Enqueue.
func (q *Workqueue) SetMaxQueueDepth(n int) {
	q.maxQueueDepth.Store(int32(n))
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
	if q.Queue != nil {
		q.Queue.ShutDown()
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
		item, shutdown := q.Queue.Get()
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
	return q.Queue.Len()
}

// MaxQueueDepth returns the current maximum queue depth (0 = unlimited).
func (q *Workqueue) MaxQueueDepth() int { return int(q.maxQueueDepth.Load()) }
