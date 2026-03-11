package queue

import (
	"context"

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
	Queue   workqueue.TypedRateLimitingInterface[QueueItem]
	started bool
}

func NewWorkqueue() *Workqueue {
	return &Workqueue{Queue: workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[QueueItem]())}
}

// enqueue adds the object's key to the workqueue
func (q *Workqueue) Enqueue(obj interface{}, gvk string) {
	// Handle tombstone (deleted objects)
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get key")
		return
	}

	q.Queue.Add(QueueItem{Key: key, GVK: gvk})
	logger.Debug().Msgf("Enqueued: %s, gvk: %s", key, gvk)
}

// Methods
var _ domain.Komponent = (*Workqueue)(nil)

func (q *Workqueue) Start(ctx context.Context) error {
	logger.Info().Msg("right here in queue")
	q.started = true
	return nil
}

func (q *Workqueue) Started() bool { return q.started }

func (q *Workqueue) Shutdown(ctx context.Context) {
	if q.Queue != nil {
		q.Queue.ShutDown()
	}
}

func (q *Workqueue) Name() string {
	return "queue"
}

func (q *Workqueue) Depth() int {
	return q.Queue.Len()
}
