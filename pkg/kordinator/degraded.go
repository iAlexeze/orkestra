// pkg/kordinator/degraded.go
package kordinator

func (k *Kontroller) IsDegraded(gvk string) bool {
	wq, _ := k.queueRegistry.For(gvk)

	if wq.Depth() > wq.MaxQueueDepth() {
		return true
	}

	// Check if error rate is too high
	if k.errorRate(gvk) > 0.1 {
		return true
	}

	return false
}
