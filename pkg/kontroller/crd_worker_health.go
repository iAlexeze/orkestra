// pkg/kontroller/crd_worker_health.go
package kontroller

import "github.com/ialexeze/orkestra/pkg/metrics"

// Worker state constants
const (
	WorkerStateIdle       = "idle"
	WorkerStateProcessing = "processing"
	WorkerStateStopped    = "stopped"
)

// SetTotalWorkers sets the expected number of workers
func (h *CRDHealth) SetTotalWorkers(count int32) {
	h.totalWorkers.Store(count)
	metrics.SetWorkersTotal(h.gvk, float64(count))
}

// SetIdleWorkers sets the number of idle workers
func (h *CRDHealth) SetIdleWorkers(count int32) {
	h.idleWorkers.Store(count)
	if h.gvk != "" {
		metrics.SetWorkersIdle(h.gvk, float64(count))
	}
}

// SetProcessingWorkers sets the number of workers currently reconciling
func (h *CRDHealth) SetProcessingWorkers(count int32) {
	h.processingWorkers.Store(count)

	if h.gvk != "" {
		metrics.SetWorkersProcessing(h.gvk, float64(count))
	}
}

// GetTotalWorkers returns the expected total workers
func (h *CRDHealth) GetTotalWorkers() int32 {
	return h.totalWorkers.Load()
}

// MarkWorkerProcessing is called when a worker starts processing an item
// func (h *CRDHealth) MarkWorkerProcessing(workerID string) {
// 	old := h.processingWorkers.Add(1)
// 	h.workerStates.Store(workerID, WorkerStateProcessing)

// 	// Update metrics
// 	metrics.SetWorkersProcessing(h.gvk, float64(old+1))
// 	metrics.SetWorkersIdle(h.gvk, float64(h.GetTotalWorkers()-(old+1)))
// }

// MarkWorkerIdle is called when a worker finishes processing
// func (h *CRDHealth) MarkWorkerIdle(workerID string) {
// 	old := h.processingWorkers.Add(-1)
// 	h.workerStates.Store(workerID, WorkerStateIdle)

// 	// Update metrics
// 	metrics.SetWorkersProcessing(h.gvk, float64(old-1))
// 	metrics.SetWorkersIdle(h.gvk, float64(h.GetTotalWorkers()-(old-1)))
// }

// GetProcessingWorkers returns number of workers currently reconciling
func (h *CRDHealth) GetProcessingWorkers() int32 {
	return h.processingWorkers.Load()
}

// GetIdleWorkers returns number of workers waiting for work
func (h *CRDHealth) GetIdleWorkers() int32 {
	return h.GetTotalWorkers() - h.GetProcessingWorkers()
}

// GetActiveWorkers returns total workers (all are "active" if running)
// This is kept for API compatibility - active = total
func (h *CRDHealth) GetActiveWorkers() int32 {
	return h.GetTotalWorkers()
}

// GetWorkerStates returns a map of worker states for debugging
func (h *CRDHealth) GetWorkerStates() map[string]string {
	states := make(map[string]string)
	h.workerStates.Range(func(key, value interface{}) bool {
		states[key.(string)] = value.(string)
		return true
	})
	return states
}

// ResetWorkerCounts resets all worker counters (used during deactivation)
// func (h *CRDHealth) ResetWorkerCounts() {
// 	h.processingWorkers.Store(0)
// 	h.totalWorkers.Store(0)
// 	h.workerStates.Range(func(key, value interface{}) bool {
// 		h.workerStates.Store(key, WorkerStateStopped)
// 		return true
// 	})
// }

// MarkWorkerProcessing is called when a worker starts processing an item
func (h *CRDHealth) MarkWorkerProcessing(workerID string) {
	old := h.processingWorkers.Add(1)
	h.workerStates.Store(workerID, WorkerStateProcessing)

	// Update metrics
	if h.gvk != "" {
		metrics.SetWorkersProcessing(h.gvk, float64(old+1))
		metrics.SetWorkersIdle(h.gvk, float64(h.GetTotalWorkers()-(old+1)))
	}
}

// MarkWorkerIdle is called when a worker finishes processing
func (h *CRDHealth) MarkWorkerIdle(workerID string) {
	old := h.processingWorkers.Add(-1)
	h.workerStates.Store(workerID, WorkerStateIdle)

	// Update metrics
	if h.gvk != "" {
		metrics.SetWorkersProcessing(h.gvk, float64(old-1))
		metrics.SetWorkersIdle(h.gvk, float64(h.GetTotalWorkers()-(old-1)))
	}
}

// ResetWorkerCounts resets all worker counters (used during deactivation)
func (h *CRDHealth) ResetWorkerCounts() {
	h.processingWorkers.Store(0)
	// Don't reset totalWorkers - it should stay as configured
	h.workerStates.Range(func(key, value interface{}) bool {
		h.workerStates.Store(key, WorkerStateStopped)
		return true
	})

	if h.gvk != "" {
		metrics.SetWorkersProcessing(h.gvk, 0)
		metrics.SetWorkersIdle(h.gvk, 0)
	}
}
