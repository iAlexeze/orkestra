package event

import (
	"context"

	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/kubeclient"
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
}

var _ domain.Component = (*Event)(nil)

func NewEvent(kube *kubeclient.Kubeclient) *Event {
	if kube.Scheme() == nil {
		panic("scheme cannot be nil")
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

func (r *Event) Shutdown(ctx context.Context) {
	if r.broadcaster != nil {
		r.broadcaster.Shutdown()
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
