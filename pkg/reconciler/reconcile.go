package reconciler

import (
	"fmt"

	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/event"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/kubeclient"
	"k8s.io/client-go/tools/cache"
)

type NewReconcilerFunc func(
	kube *kubeclient.Kubeclient,
	inf cache.SharedIndexInformer,
	ev *event.Event,
) domain.Reconciler

var registry = map[string]NewReconcilerFunc{}

func RegisterReconcilers() map[string]NewReconcilerFunc {
	return map[string]NewReconcilerFunc{
		"project": func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer, ev *event.Event) domain.Reconciler {
			return NewProjectReconciler(inf, ev)
		},

		"managednamespace": func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer, ev *event.Event) domain.Reconciler {
			return NewManagedNamespaceReconciler(kube, inf, ev)
		},
	}
}

func Get(name string) (NewReconcilerFunc, error) {
	fn, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("reconciler not registered: %s", name)
	}
	return fn, nil
}
