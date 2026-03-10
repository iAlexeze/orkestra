package event

import (
	"context"
	"sync"

	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	crderror "github.com/ialexeze/multi-crd-controller/pkg/config/pkg/error"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/kubeclient"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

type Event struct {
	name        string
	kube        *kubeclient.Kubeclient
	scheme      *runtime.Scheme
	broadcaster record.EventBroadcaster
	recorder    record.EventRecorder
	component   string
	stopped     bool           // Track state
	wg          sync.WaitGroup // Track in-flight events
	mu          sync.Mutex     // Protect shutdown
}

var _ domain.Component = (*Event)(nil)

func NewEvent(kube *kubeclient.Kubeclient) *Event {
	if kube.Scheme() == nil {
		utils.Exit(crderror.ErrSchemeNill)
	}

	return &Event{
		name:      "event handler",
		component: "multi-crd-controller",
		kube:      kube,
		scheme:    kube.Scheme(),
	}
}

func (r *Event) Start(ctx context.Context) error {
	// Check if context is cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	// Create event broadcaster
	r.broadcaster = record.NewBroadcaster(record.WithContext(ctx))
	r.broadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{
			Interface: r.kube.Clientset().CoreV1().Events(""),
		})

	// Create event recorder
	r.recorder = r.broadcaster.NewRecorder(
		r.scheme,
		corev1.EventSource{
			Component: r.component,
		})
	return nil
}

// Eventf - track in-flight events
func (e *Event) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return
	}
	e.wg.Add(1) // Track this event
	e.mu.Unlock()

	// Record in goroutine so we don't block
	go func() {
		defer e.wg.Done()
		if e.recorder != nil {
			e.recorder.Eventf(object, eventtype, reason, messageFmt, args...)
		}
	}()
}

func (e *Event) Shutdown(ctx context.Context) {
	logger.Info().Msgf("shutting down %s...", e.name)

	e.mu.Lock()
	e.stopped = true
	e.mu.Unlock()

	// Wait for in-flight events with timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info().Msg("all events flushed")
	case <-ctx.Done():
		logger.Warn().Msg("timeout waiting for events to flush")
	}

	if e.broadcaster != nil {
		e.broadcaster.Shutdown()
	}
}

func (r *Event) Name() string {
	return r.name
}

func (r *Event) Broadcaster() record.EventBroadcaster {
	return r.broadcaster
}

func (r *Event) Recorder() record.EventRecorder {
	return r.recorder
}
