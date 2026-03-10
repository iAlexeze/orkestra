// pkg/controller/degraded.go
package controller

func (c *Controller) IsDegraded(gvk string) bool {
	// Check if queue depth is too high
	if c.wq.Depth() > c.maxQueueDepth {
		return true
	}

	// Check if error rate is too high
	if c.errorRate(gvk) > 0.1 {
		return true
	}

	c.hs.Degraded()
	return false
}
