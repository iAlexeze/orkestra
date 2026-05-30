// pkg/queue/workqueue_test.go
package queue_test

import (
	"context"
	"testing"

	"github.com/orkspace/orkestra/pkg/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Construction ──────────────────────────────────────────────────────────────

func TestNewWorkqueue_IsNotNil(t *testing.T) {
	q := queue.NewWorkqueue()
	require.NotNil(t, q)
}

func TestNewWorkqueue_InitialState(t *testing.T) {
	q := queue.NewWorkqueue()

	assert.False(t, q.Started(), "queue must not report started before Start() is called")
	assert.Equal(t, 0, q.Depth(), "empty queue must have depth 0")
	assert.Equal(t, 0, q.MaxDepth(), "default max depth is 0 (unconfigured)")
}

func TestNewWorkqueue_Name(t *testing.T) {
	q := queue.NewWorkqueue()
	assert.Equal(t, "default workqueue", q.Name())
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

func TestWorkqueue_StartSetsStartedFlag(t *testing.T) {
	q := queue.NewWorkqueue()
	assert.False(t, q.Started())

	err := q.Start(context.Background())

	assert.NoError(t, err)
	assert.True(t, q.Started())
}

func TestWorkqueue_StartIsIdempotent(t *testing.T) {
	q := queue.NewWorkqueue()

	err1 := q.Start(context.Background())
	err2 := q.Start(context.Background())

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.True(t, q.Started())
}

func TestWorkqueue_ShutdownAfterStart(t *testing.T) {
	q := queue.NewWorkqueue()
	require.NoError(t, q.Start(context.Background()))

	assert.NotPanics(t, func() {
		q.Shutdown(context.Background())
	})
}

func TestWorkqueue_ShutdownBeforeStartDoesNotPanic(t *testing.T) {
	q := queue.NewWorkqueue()

	assert.NotPanics(t, func() {
		q.Shutdown(context.Background())
	})
}

// ── Depth ─────────────────────────────────────────────────────────────────────

func TestWorkqueue_DepthIsZeroInitially(t *testing.T) {
	q := queue.NewWorkqueue()
	assert.Equal(t, 0, q.Depth())
}

func TestWorkqueue_DepthAfterShutdownDoesNotPanic(t *testing.T) {
	q := queue.NewWorkqueue()
	q.Shutdown(context.Background())

	assert.NotPanics(t, func() {
		_ = q.Depth()
	})
}

// ── QueueRegistry ─────────────────────────────────────────────────────────────

func TestNewQueueRegistry_IsNotNil(t *testing.T) {
	r := queue.NewQueueRegistry()
	require.NotNil(t, r)
}

func TestQueueRegistry_InitialState(t *testing.T) {
	r := queue.NewQueueRegistry()

	assert.False(t, r.Started())
	assert.Equal(t, "queue registry", r.Name())
}

func TestQueueRegistry_StartSetsStartedFlag(t *testing.T) {
	r := queue.NewQueueRegistry()
	assert.False(t, r.Started())

	err := r.Start(context.Background())

	assert.NoError(t, err)
	assert.True(t, r.Started())
}

func TestQueueRegistry_RegisterAndRetrieve(t *testing.T) {
	r := queue.NewQueueRegistry()

	gvk := "demo.orkestra.io/v1alpha1, Kind=Website"
	r.Register(gvk, 100)

	q, ok := r.For(gvk)
	assert.True(t, ok, "registered GVK must be found")
	assert.NotNil(t, q, "registered queue must be retrievable by GVK")
}

func TestQueueRegistry_ForUnregisteredGVK(t *testing.T) {
	r := queue.NewQueueRegistry()

	q, ok := r.For("nonexistent/v1, Kind=Widget")
	assert.False(t, ok, "unregistered GVK must return false")
	assert.Nil(t, q, "unregistered GVK must return nil queue")
}

func TestQueueRegistry_DepthOfRegisteredQueue(t *testing.T) {
	r := queue.NewQueueRegistry()
	gvk := "demo.orkestra.io/v1alpha1, Kind=Website"
	r.Register(gvk, 50)

	depth := r.Depth(gvk)
	assert.Equal(t, 0, depth, "freshly registered queue must have depth 0")
}

func TestQueueRegistry_DepthOfUnregisteredGVK(t *testing.T) {
	r := queue.NewQueueRegistry()

	assert.NotPanics(t, func() {
		depth := r.Depth("unknown/v1, Kind=Nope")
		assert.Equal(t, 0, depth)
	})
}

func TestQueueRegistry_ShutdownAllQueues(t *testing.T) {
	r := queue.NewQueueRegistry()
	r.Register("gvk1", 10)
	r.Register("gvk2", 10)

	assert.NotPanics(t, func() {
		r.Shutdown(context.Background())
	})
}
