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
	Scheme() *runtime.Scheme

	// Args returns the args declared under hooks.args or constructor.args in katalog.yaml.
	// Returns an empty Args map when no args were declared.
	Args() Args

	// WithArgs returns a copy of this KubeClient with the given args attached.
	// Used by the runtime to inject per-CRD args before calling a hook or constructor.
	WithArgs(args Args) KubeClient

	// ScopedFor evaluates any template expressions in the rawArgs using eval and
	// returns a copy of this KubeClient with the resolved args attached.
	// Called by GenericReconciler after building the resolver so hook authors see
	// fully-evaluated args without any extra wiring. Constructor authors call it
	// themselves using their own resolver.
	ScopedFor(eval func(string) (string, bool)) KubeClient

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
