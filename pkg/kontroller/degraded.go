// pkg/kontroller/degraded.go
package kontroller

func (c *Controller) IsDegraded(gvk string) bool {
	wq, _ := c.queueRegistry.For(gvk)

	if wq.Depth() > wq.MaxQueueDepth() {
		return true
	}

	// Check if error rate is too high
	if c.errorRate(gvk) > 0.1 {
		return true
	}

	return false
}
