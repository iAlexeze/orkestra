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
	name    string
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
		name:   "queue registry",
		queues: make(map[string]*Workqueue), // Create the map for per CRD registration
	}
}

// Register registers each CRD with their respective
// GVK and maximum queue depth
func (qr *QueueRegistry) Register(gvk string, maxQueueDepth int) *Workqueue {
	qr.mu.Lock()
	defer qr.mu.Unlock()

	wq := NewWorkqueue()
	qr.queues[gvk] = wq              // Register each CRD to a workqueue
	wq.maxQueueDepth = maxQueueDepth // Set the maximum queue depth for this new queue per CRD

	return wq
}

// For returns the workqueue for a given GVK
func (qr *QueueRegistry) For(gvk string) (*Workqueue, bool) {
	qr.mu.RLock()
	defer qr.mu.RUnlock()

	wq, ok := qr.queues[gvk]
	if !ok {
		return nil, false
	}

	return wq, true
}

// Depth returns the queue depth for a given GVK
func (qr *QueueRegistry) Depth(gvk string) int {
	qr.mu.RLock()
	defer qr.mu.RUnlock()

	wq, ok := qr.queues[gvk]
	if !ok {
		return 0
	}

	if wq.Queue == nil {
		return 0
	}

	return wq.Depth()
}

// Shutdown drains all registered workqueues
// This is called by orkestra.Shutdown() for graceful degradation
func (qr *QueueRegistry) Shutdown(ctx context.Context) {
	qr.mu.Lock()
	defer qr.mu.Unlock()

	for _, wq := range qr.queues {
		if wq.Queue != nil {
			wq.Queue.ShutDown()
		}
	}
}

// Methods implementation of the Komponent interface
var _ domain.Komponent = (*QueueRegistry)(nil)

// Called by orkestra.Start() to start the workqueue registry
func (qr *QueueRegistry) Start(ctx context.Context) error {
	logger.Debug().Msgf("right here in %s with %v queues", qr.name, len(qr.queues))

	qr.started.Store(true)
	return nil
}

// Started is a status check for all orkestra komponents
func (qr *QueueRegistry) Started() bool { return qr.started.Load() }

// Name returns the name of the queue registry
func (qr *QueueRegistry) Name() string {
	return qr.name
}
