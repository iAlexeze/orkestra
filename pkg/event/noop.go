package event

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// Recorder is the interface for emitting Kubernetes events.
// *Event implements this for real clusters.
// *NoopRecorder implements it for simulation — discards all events.
type Recorder interface {
	Eventf(obj runtime.Object, eventType, reason, messageFmt string, args ...interface{})
}

// NoopRecorder discards all events. Used in ork simulate.
type NoopRecorder struct{}

func (n *NoopRecorder) Eventf(runtime.Object, string, string, string, ...interface{}) {}

var _ Recorder = (*NoopRecorder)(nil)
