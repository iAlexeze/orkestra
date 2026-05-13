package kubeclient

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// NewForTesting builds a Kubeclient wired to the given REST config without
// going through Start(). Used exclusively by integration tests that provide
// their own envtest config.
func NewForTesting(cfg *rest.Config, dynamic dynamic.Interface, scheme *runtime.Scheme) *Kubeclient {
	return &Kubeclient{
		name:       "kubeclient-test",
		restConfig: cfg,
		dynamic:    dynamic,
		scheme:     scheme,
	}
}
