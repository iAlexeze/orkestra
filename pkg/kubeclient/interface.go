package kubeclient

import (
	"context"

	"github.com/orkspace/orkestra/domain"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// KubeClient is the interface every registry function depends on.
// *Kubeclient satisfies this with real clients.
// *simulate.FakeKubeclient satisfies this with k8s.io/client-go/kubernetes/fake.
type KubeClient interface {
	Clientset() kubernetes.Interface
	DynamicClient() dynamic.Interface
	Mapper() meta.RESTMapper
	RestConfig() *rest.Config

	// CRUD — typed object operations for constructor reconcilers.
	// Accepts sigs.k8s.io/controller-runtime/pkg/client.Object so reconcilers
	// migrated from controller-runtime compile without changes to their call sites.
	// GVR is derived from the Go type via the scheme and mapper; callers do not
	// need to specify it explicitly.
	Get(ctx context.Context, namespace, name string, into sigs.Object) error
	Create(ctx context.Context, obj sigs.Object) error
	Patch(ctx context.Context, obj sigs.Object, patch Patch) error

	// Patch helpers — used by the generic reconciler for finalizer, label,
	// annotation, and status updates. Implementations must be idempotent.
	PatchFinalizers(ctx context.Context, obj runtime.Object, finalizers []string) error
	PatchLabels(ctx context.Context, obj runtime.Object, base, desired map[string]string) error
	PatchAnnotations(ctx context.Context, obj runtime.Object, annotations map[string]string) error
	PatchStatus(ctx context.Context, obj domain.Object, statusFields map[string]interface{}) error
}

// Compile check — *Kubeclient must satisfy this.
var _ KubeClient = (*Kubeclient)(nil)
