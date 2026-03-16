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
	name string
	Queue         workqueue.TypedRateLimitingInterface[QueueItem]
	started       atomic.Bool
	maxQueueDepth int
}

func NewWorkqueue() *Workqueue {
	return &Workqueue{
		name: "default workqueue",
		Queue: workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[QueueItem]()),
	}
}

// enqueue adds the object's key to the workqueue
func (q *Workqueue) Enqueue(obj interface{}, gvk string) {
	// Handle tombstone (deleted objects)
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Error().Err(err).Msg("enqueue: failed to get key")
		return
	}

	q.Queue.Add(QueueItem{Key: key, GVK: gvk})
	logger.Debug().Msgf("Enqueued: %s, gvk: %s", key, gvk)
}

// Methods to implement the komponent interface
var _ domain.Komponent = (*Workqueue)(nil)

// Start is called by orkestra.Start() after all komoonents are registered
func (q *Workqueue) Start(ctx context.Context) error {
	logger.Debug().Msgf("right here in %s", q.name)
	
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

// Name returns the name of the default workqueue
func (q *Workqueue) Name() string {
	return q.name
}

// Depth returns the lenght of the default workqueue
// This is used by orkestra to determine the health of the workqueue
func (q *Workqueue) Depth() int {
	return q.Queue.Len()
}

// MaxQueueDepth returns the maximum queue deprh set for the queue
// This is also used by orjestra to determine the health of the workqueue
func (q *Workqueue) MaxQueueDepth() int { return q.maxQueueDepth }
