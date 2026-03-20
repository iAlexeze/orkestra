// pkg/kubeclient/dynamic.go
package kubeclient

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// DynamicListerWatcher implements cache.ListerWatcher using the dynamic client.
// Safe to construct before kube.Start() — k.dynamic is resolved at call time.
type DynamicListerWatcher struct {
	kube       *Kubeclient
	gvr        schema.GroupVersionResource
	namespace  string
	namespaced bool
}

// NewDynamicListerWatcher builds a cache.ListerWatcher for an unstructured CRD.
// Safe to call before kube.Start() — dynamic client is resolved lazily.
func (k *Kubeclient) NewDynamicListerWatcher(info CRDInfo) cache.ListerWatcher {
	return &DynamicListerWatcher{
		kube: k,
		gvr: schema.GroupVersionResource{
			Group:    info.Group,
			Version:  info.Version,
			Resource: info.Plural,
		},
		namespace:  info.Namespace,
		namespaced: info.Namespaced,
	}
}

// List implements cache.ListerWatcher.
// *unstructured.UnstructuredList satisfies runtime.Object — explicit cast needed
// because dynamic.ResourceInterface.List returns the concrete type, not the interface.
func (d *DynamicListerWatcher) List(opts metav1.ListOptions) (runtime.Object, error) {
	if d.namespaced { // If namespaced
		ns := d.namespace
		if ns == "" {
			ns = metav1.NamespaceAll
		}
		return d.kube.dynamic.Resource(d.gvr).Namespace(ns).List(context.Background(), opts)
	}

	// Cluster-scoped
	return d.kube.dynamic.Resource(d.gvr).List(context.Background(), opts)
}

// Watch implements cache.ListerWatcher.
func (d *DynamicListerWatcher) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	if d.namespaced { // If namespaced
		ns := d.namespace
		if ns == "" {
			ns = metav1.NamespaceAll
		}
		return d.kube.dynamic.Resource(d.gvr).Namespace(ns).Watch(context.Background(), opts)
	}

	// Cluster-scoped
	return d.kube.dynamic.Resource(d.gvr).Watch(context.Background(), opts)
}
