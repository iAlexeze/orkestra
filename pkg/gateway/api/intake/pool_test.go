package intake

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_RunsJobs(t *testing.T) {
	p := newWorkerPool(2)
	var ran int32
	var wg sync.WaitGroup
	wg.Add(3)
	for i := 0; i < 3; i++ {
		p.Submit(context.Background(), func(ctx context.Context) {
			defer wg.Done()
			atomic.AddInt32(&ran, 1)
		})
	}
	wg.Wait()
	if ran != 3 {
		t.Errorf("ran = %d, want 3", ran)
	}
}

func TestWorkerPool_CapsConcurrency(t *testing.T) {
	const limit = 2
	p := newWorkerPool(limit)

	var current, max int32
	var mu sync.Mutex
	var wg sync.WaitGroup
	release := make(chan struct{})

	// Submit from their own goroutines: Submit blocks its caller until a
	// slot is free, and with more jobs than the pool's limit, some of these
	// calls won't get a slot until the first batch finishes below — so the
	// submitting goroutine can't be the same one that later closes release.
	const jobs = 6
	wg.Add(jobs)
	for i := 0; i < jobs; i++ {
		go p.Submit(context.Background(), func(ctx context.Context) {
			defer wg.Done()
			n := atomic.AddInt32(&current, 1)
			mu.Lock()
			if n > max {
				max = n
			}
			mu.Unlock()
			<-release
			atomic.AddInt32(&current, -1)
		})
	}

	// Give the first `limit` jobs time to start and block on `release`.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if max > limit {
		t.Errorf("observed max concurrency = %d, want <= %d", max, limit)
	}
}

func TestWorkerPool_ZeroOrNegativeTreatedAsOne(t *testing.T) {
	p := newWorkerPool(0)
	if cap(p.sem) != 1 {
		t.Errorf("cap(sem) = %d, want 1 for n<=0", cap(p.sem))
	}
}

func TestWorkerPool_PanicDoesNotBlockSubsequentJobs(t *testing.T) {
	p := newWorkerPool(1)
	var wg sync.WaitGroup
	wg.Add(2)

	p.Submit(context.Background(), func(ctx context.Context) {
		defer wg.Done()
		panic("boom")
	})

	var secondRan int32
	p.Submit(context.Background(), func(ctx context.Context) {
		defer wg.Done()
		atomic.StoreInt32(&secondRan, 1)
	})

	wg.Wait()
	if secondRan != 1 {
		t.Error("a panic in one job should not prevent the next job from running")
	}
}
