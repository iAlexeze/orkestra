package kubeclient

import (
	"context"

	"github.com/orkspace/orkestra/domain"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// KubeClient is the interface every registry function depends on.
// *Kubeclient satisfies this with real clients.
// *simulate.FakeKubeclient satisfies this with k8s.io/client-go/kubernetes/fake.
type KubeClient interface {
	Clientset() kubernetes.Interface
	DynamicClient() dynamic.Interface
	Mapper() meta.RESTMapper

	// Patch helpers — used by the reconciler for finalizer, label,
	// annotation, and status updates. Implementations must be idempotent.
	PatchFinalizers(ctx context.Context, obj runtime.Object, gvr schema.GroupVersionResource, finalizers []string) error
	PatchLabels(ctx context.Context, obj runtime.Object, gvr schema.GroupVersionResource, base, desired map[string]string) error
	PatchAnnotations(ctx context.Context, obj runtime.Object, gvr schema.GroupVersionResource, annotations map[string]string) error
	PatchStatus(ctx context.Context, obj domain.Object, gvr schema.GroupVersionResource, statusFields map[string]interface{}) error
}

// Compile check — *Kubeclient must satisfy this.
var _ KubeClient = (*Kubeclient)(nil)
