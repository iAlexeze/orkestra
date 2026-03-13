// pkg/queue/registry.go
package queue

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
)

type QueueRegistry struct {
	queues  map[string]*Workqueue // keyed by GVK string
	mu      sync.RWMutex
	started atomic.Bool
}

func NewQueueRegistry() *QueueRegistry {
	qr := &QueueRegistry{
		queues: make(map[string]*Workqueue),
	}

	qr.started.Store(false)
	return qr
}

func (r *QueueRegistry) Register(gvk string, maxQueueDepth int) *Workqueue {
	r.mu.Lock()
	defer r.mu.Unlock()
	wq := NewWorkqueue()
	r.queues[gvk] = wq
	wq.maxQueueDepth = maxQueueDepth
	return wq
}

func (r *QueueRegistry) For(gvk string) (*Workqueue, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wq, ok := r.queues[gvk]
	return wq, ok
}

func (r *QueueRegistry) Shutdown(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, wq := range r.queues {
		wq.Queue.ShutDown()
	}
}

// Methods
var _ domain.Komponent = (*QueueRegistry)(nil)

func (qr *QueueRegistry) Start(ctx context.Context) error {
	logger.Debug().Msgf("right here in queue registry with %v queues", len(qr.queues))

	qr.started.Store(true)
	return nil
}

func (qr *QueueRegistry) Started() bool { return qr.started.Load() }

func (qr *QueueRegistry) Name() string {
	return "queue registry"
}
