package intake

import (
	"context"

	"github.com/orkspace/orkestra/pkg/logger"
)

// workerPool runs submitted jobs on at most n concurrent goroutines,
// recovering from panics so one bad job can't take the pool down. Used for
// Slack's async apply-after-ack — Slack retries on a slow HTTP response, so
// a burst of commands must not translate into unbounded concurrent applies
// against the cluster.
type workerPool struct {
	sem chan struct{}
}

// newWorkerPool returns a pool that runs at most n jobs concurrently. n<=0
// is treated as 1 — a pool that runs nothing isn't useful.
func newWorkerPool(n int) *workerPool {
	if n <= 0 {
		n = 1
	}
	return &workerPool{sem: make(chan struct{}, n)}
}

// Submit runs fn on a pooled goroutine. Blocks the caller only long enough
// to acquire a slot, not until fn completes — callers past their own
// synchronous response (Slack's 3-second ack) are free to move on.
func (p *workerPool) Submit(ctx context.Context, fn func(context.Context)) {
	p.sem <- struct{}{}
	go func() {
		defer func() { <-p.sem }()
		defer func() {
			if r := recover(); r != nil {
				logger.FromContext(ctx).Error().
					Interface("panic", r).
					Msg("intake: worker pool job panicked")
			}
		}()
		fn(ctx)
	}()
}
