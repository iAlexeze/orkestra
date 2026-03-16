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
	name string
	queues  map[string]*Workqueue // keyed by GVK string
	mu      sync.RWMutex
	started atomic.Bool
}

// NewQueueRegistry returns a new queue registry
// At this stage only the queues map is created
// Register uses the map created to register all CRDs
// For returns a registered CRD
func NewQueueRegistry() *QueueRegistry {
	return &QueueRegistry{
		name: "queue registry",
		queues: make(map[string]*Workqueue),     // Create the map for per CRD registration
	}
}

// Register registers each CRD with their respective
// GVK and maximum queue depth
func (r *QueueRegistry) Register(gvk string, maxQueueDepth int) *Workqueue {
	r.mu.Lock()
	defer r.mu.Unlock()

	wq := NewWorkqueue()
	r.queues[gvk] = wq    // Register each CRD to a workqueue
	wq.maxQueueDepth = maxQueueDepth    // Set the maximum queue depth for this new queue per CRD
	
	return wq
}

// For returns the workqueue for a given GVK
func (r *QueueRegistry) For(gvk string) (*Workqueue, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.queues[gvk]
}

// Shutdown drains all registered workqueues
// This is called by orkestra.Shutdown() for graceful degradation
func (r *QueueRegistry) Shutdown(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, wq := range r.queues {
		if wq.Queue != nil {
			wq.Queue.ShutDown()
		}
	}
}

// Methods implementation of the Komponent interface
var _ domain.Komponent = (*QueueRegistry)(nil)

// Called by orkestra.Start() to start the workqueue registry
func (r *QueueRegistry) Start(ctx context.Context) error {
	logger.Debug().Msgf("right here in %s with %v queues", r.name, len(qr.queues))

	r.started.Store(true)
	return nil
}

// Started is a status check for all orkestra komponents
func (r *QueueRegistry) Started() bool { return r.started.Load() }

// Name returns the name of the queue registry
func (r *QueueRegistry) Name() string {
	return r.name
}
