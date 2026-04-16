// pkg/queue/registry.go
package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
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
	qr.queues[gvk] = wq                          // Register each CRD to a workqueue
	wq.maxQueueDepth.Store(int32(maxQueueDepth)) // Set the maximum queue depth for this new queue per CRD

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

// ShutdownQueue drains the queue of a given CRD
func (qr *QueueRegistry) ShutdownQueue(gvkStr string) error {
	qr.mu.Lock()
	defer qr.mu.Unlock()

	q, ok := qr.queues[gvkStr]
	if !ok {
		return fmt.Errorf("queue not found for %s", gvkStr)
	}

	q.Queue.ShutDown()
	delete(qr.queues, gvkStr)
	return nil
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
	count := len(qr.queues)
	logger.Debug().Str("name", qr.name).Int("queues", count).Msg("queue registry started")

	qr.started.Store(true)
	return nil
}

// Started is a status check for all orkestra komponents
func (qr *QueueRegistry) Started() bool { return qr.started.Load() }

// Name returns the name of the queue registry
func (qr *QueueRegistry) Name() string {
	return qr.name
}
