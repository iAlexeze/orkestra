package simulate

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// Op is one recorded cluster operation.
type Op struct {
	Cycle     int
	Verb      string // "create", "update", "delete", "get", "patch"
	Resource  string // "deployments", "services", etc.
	Namespace string
	Name      string
	At        time.Time
}

// fakeShared holds the mutable, mutex-guarded state shared across all
// WithArgs/ScopedFor copies of a FakeKubeclient.
type fakeShared struct {
	mu           sync.Mutex
	ops          []Op
	currentCycle int
}

// FakeKubeclient implements kubeclient.KubeClient using k8s fakes.
// All operations are recorded in Ops() for simulation output.
type FakeKubeclient struct {
	clientset kubernetes.Interface
	dynamic   dynamic.Interface
	mapper    meta.RESTMapper
	scheme    *runtime.Scheme
	shared    *fakeShared

	rawArgs map[string]interface{}
	args    kubeclient.Args
}

// dynamicObjects seeds the fake dynamic client's tracker at construction —
// this must happen at construction, not via Tracker().Add() afterward,
// because fakedynamic.NewSimpleDynamicClient derives its GVR→ListKind
// mapping from the objects it's given; List() calls for a GVR that wasn't
// present at construction time fail even if the object is added later.
// Used to make an in-progress-reconcile CR (and any pre-existing sibling
// instances from a multi-doc CR file) visible to reconcile-time checks that
// list other instances of the same CRD, like operator: unique.
func NewFakeKubeclient(scheme *runtime.Scheme, dynamicObjects ...runtime.Object) *FakeKubeclient {
	f := &FakeKubeclient{shared: &fakeShared{}}

	cs := fake.NewClientset()
	// PrependReactor intercepts every operation and records it before the
	// default object-tracker reactor handles it. AddReactor appends to the
	// chain AFTER the tracker, so it is never reached — PrependReactor is
	// required here.
	cs.Fake.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		f.shared.mu.Lock()
		f.shared.ops = append(f.shared.ops, Op{
			Cycle:     f.shared.currentCycle,
			Verb:      verbFromAction(action),
			Resource:  action.GetResource().Resource,
			Namespace: action.GetNamespace(),
			Name:      nameFromAction(action),
			At:        time.Now(),
		})
		f.shared.mu.Unlock()
		return false, nil, nil
	})

	f.clientset = cs
	f.scheme = scheme

	dyn := fakedynamic.NewSimpleDynamicClient(scheme, dynamicObjects...) // For custom + seeded pre-existing instances
	// PrependReactor intercepts every operation and records it before the
	// default object-tracker reactor handles it. AddReactor appends to the
	// chain AFTER the tracker, so it is never reached — PrependReactor is
	// required here.
	dyn.Fake.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		f.shared.mu.Lock()
		f.shared.ops = append(f.shared.ops, Op{
			Cycle:     f.shared.currentCycle,
			Verb:      verbFromAction(action),
			Resource:  action.GetResource().Resource,
			Namespace: action.GetNamespace(),
			Name:      nameFromAction(action),
			At:        time.Now(),
		})
		f.shared.mu.Unlock()
		return false, nil, nil
	})
	f.dynamic = dyn
	f.mapper = &fakeMapper{}

	return f
}

func (f *FakeKubeclient) Clientset() kubernetes.Interface  { return f.clientset }
func (f *FakeKubeclient) DynamicClient() dynamic.Interface { return f.dynamic }
func (f *FakeKubeclient) Mapper() meta.RESTMapper          { return f.mapper }
func (f *FakeKubeclient) RestConfig() *rest.Config         { return nil }
func (f *FakeKubeclient) Scheme() *runtime.Scheme          { return f.scheme }

func (f *FakeKubeclient) Args() kubeclient.Args {
	if f.args != nil {
		return f.args
	}
	if f.rawArgs != nil {
		return kubeclient.Args(f.rawArgs)
	}
	return kubeclient.Args{}
}

func (f *FakeKubeclient) WithArgs(args kubeclient.Args) kubeclient.KubeClient {
	cp := *f
	cp.rawArgs = map[string]interface{}(args)
	cp.args = nil
	return &cp
}

func (f *FakeKubeclient) ScopedFor(eval func(string) (string, bool)) kubeclient.KubeClient {
	cp := *f
	if len(f.rawArgs) == 0 {
		return &cp
	}
	cp.args = kubeclient.ResolveArgsMap(f.rawArgs, eval)
	return &cp
}

// AdvanceCycle increments the cycle counter. Call between simulated reconciles.
func (f *FakeKubeclient) AdvanceCycle() {
	f.shared.mu.Lock()
	defer f.shared.mu.Unlock()
	f.shared.currentCycle++
}

// Ops returns all recorded operations in order.
func (f *FakeKubeclient) Ops() []Op {
	f.shared.mu.Lock()
	defer f.shared.mu.Unlock()
	result := make([]Op, len(f.shared.ops))
	copy(result, f.shared.ops)
	return result
}

// OpsForCycle returns operations from one reconcile cycle.
func (f *FakeKubeclient) OpsForCycle(cycle int) []Op {
	f.shared.mu.Lock()
	defer f.shared.mu.Unlock()
	var result []Op
	for _, op := range f.shared.ops {
		if op.Cycle == cycle {
			result = append(result, op)
		}
	}
	return result
}

// MarkDeploymentReady advances simulated state — marks a Deployment as Available.
// Call after the first reconcile cycle to allow the reconciler to progress
// through "Deploying" → "Ready" state transitions.
func (f *FakeKubeclient) MarkDeploymentReady(namespace, name string) {
	// The fake clientset stores objects in an object tracker.
	// We patch the Deployment's status directly.
	dep, err := f.clientset.AppsV1().Deployments(namespace).Get(
		context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return
	}
	if dep.Spec.Replicas != nil {
		dep.Status.AvailableReplicas = *dep.Spec.Replicas
		dep.Status.ReadyReplicas = *dep.Spec.Replicas
		dep.Status.Replicas = *dep.Spec.Replicas
	}
	f.clientset.AppsV1().Deployments(namespace).UpdateStatus(
		context.Background(), dep, metav1.UpdateOptions{})
}

// verbFromAction returns "apply" for Server-Side Apply patch operations,
// "patch" for all other patch types, and the raw verb for everything else.
func verbFromAction(action k8stesting.Action) string {
	if pa, ok := action.(k8stesting.PatchAction); ok {
		if pa.GetPatchType() == k8stypes.ApplyPatchType {
			return "apply"
		}
	}
	return action.GetVerb()
}

// nameFromAction extracts the resource name from a test action.
func nameFromAction(action k8stesting.Action) string {
	switch a := action.(type) {
	case k8stesting.GetAction:
		return a.GetName()
	case k8stesting.CreateAction:
		obj := a.GetObject()
		if acc, ok := obj.(metav1.Object); ok {
			return acc.GetName()
		}
	case k8stesting.UpdateAction:
		obj := a.GetObject()
		if acc, ok := obj.(metav1.Object); ok {
			return acc.GetName()
		}
	case k8stesting.DeleteAction:
		return a.GetName()
	}
	return ""
}

// fakeMapper satisfies meta.RESTMapper for the simulation context.
// Returns a fixed mapping for standard resource types.
type fakeMapper struct{}

func (m *fakeMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	version := "v1"
	if len(versions) > 0 && versions[0] != "" {
		version = versions[0]
	}
	scope := meta.RESTScopeNameNamespace
	if gk.Kind == "Namespace" || gk.Kind == "ClusterRole" || gk.Kind == "ClusterRoleBinding" {
		scope = meta.RESTScopeNameRoot
	}
	return &meta.RESTMapping{
		Resource: schema.GroupVersionResource{
			Group:    gk.Group,
			Version:  version,
			Resource: strings.ToLower(gk.Kind) + "s",
		},
		GroupVersionKind: gk.WithVersion(version),
		Scope:            fakescope(scope),
	}, nil
}

func (m *fakeMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}
func (m *fakeMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	return nil, nil
}
func (m *fakeMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	return input, nil
}
func (m *fakeMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	return []schema.GroupVersionResource{input}, nil
}
func (m *fakeMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	mapping, err := m.RESTMapping(gk, versions...)
	if err != nil {
		return nil, err
	}
	return []*meta.RESTMapping{mapping}, nil
}
func (m *fakeMapper) ResourceSingularizer(resource string) (string, error) {
	return strings.TrimSuffix(resource, "s"), nil
}

type fakescope string

func (f fakescope) Name() meta.RESTScopeName { return meta.RESTScopeName(f) }
func (f fakescope) String() string           { return string(f) }

// Compile check — *FakeKubeclient must satisfy kubeclient.KubeClient.
var _ kubeclient.KubeClient = (*FakeKubeclient)(nil)

// CRUD stubs — record operations. Get always returns NotFound so the reconciler
// takes the Create path on every simulated cycle, producing visible create ops.

func (f *FakeKubeclient) Get(_ context.Context, namespace, name string, into sigs.Object) error {
	f.shared.mu.Lock()
	f.shared.ops = append(f.shared.ops, Op{
		Cycle:     f.shared.currentCycle,
		Verb:      "get",
		Resource:  resourceNameFromObject(into),
		Namespace: namespace,
		Name:      name,
		At:        time.Now(),
	})
	f.shared.mu.Unlock()
	return fakeNotFound(name)
}

func (f *FakeKubeclient) Create(_ context.Context, obj sigs.Object) error {
	f.shared.mu.Lock()
	f.shared.ops = append(f.shared.ops, Op{
		Cycle:     f.shared.currentCycle,
		Verb:      "create",
		Resource:  resourceNameFromObject(obj),
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
		At:        time.Now(),
	})
	f.shared.mu.Unlock()
	return nil
}

func (f *FakeKubeclient) Patch(_ context.Context, obj sigs.Object, _ kubeclient.Patch) error {
	f.shared.mu.Lock()
	f.shared.ops = append(f.shared.ops, Op{
		Cycle:     f.shared.currentCycle,
		Verb:      "patch",
		Resource:  resourceNameFromObject(obj),
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
		At:        time.Now(),
	})
	f.shared.mu.Unlock()
	return nil
}

// fakeNotFound returns an error that satisfies k8s.io/apimachinery/pkg/api/errors.IsNotFound.
func fakeNotFound(name string) error {
	return k8serrors.NewNotFound(schema.GroupResource{}, name)
}

func resourceNameFromObject(obj runtime.Object) string {
	t := fmt.Sprintf("%T", obj)
	if idx := strings.LastIndex(t, "."); idx >= 0 {
		t = t[idx+1:]
	}
	return strings.ToLower(t) + "s"
}

// Patch stubs — record operations but perform no real mutations.
// The fake dynamic client handles the underlying object storage.

func (f *FakeKubeclient) PatchFinalizers(_ context.Context, obj runtime.Object, finalizers []string) error {
	f.shared.mu.Lock()
	f.shared.ops = append(f.shared.ops, Op{
		Cycle:    f.shared.currentCycle,
		Verb:     "patch",
		Resource: "finalizers",
		Name:     nameFromRuntimeObject(obj),
		At:       time.Now(),
	})
	f.shared.mu.Unlock()
	return nil
}

func (f *FakeKubeclient) PatchLabels(_ context.Context, obj runtime.Object, base, desired map[string]string) error {
	if stringMapsEqual(base, desired) {
		return nil
	}
	f.shared.mu.Lock()
	f.shared.ops = append(f.shared.ops, Op{
		Cycle:    f.shared.currentCycle,
		Verb:     "patch",
		Resource: "labels",
		Name:     nameFromRuntimeObject(obj),
		At:       time.Now(),
	})
	f.shared.mu.Unlock()
	if mo, ok := obj.(metav1.Object); ok {
		mo.SetLabels(desired)
	}
	return nil
}

func (f *FakeKubeclient) PatchAnnotations(_ context.Context, obj runtime.Object, annotations map[string]string) error {
	f.shared.mu.Lock()
	f.shared.ops = append(f.shared.ops, Op{
		Cycle:    f.shared.currentCycle,
		Verb:     "patch",
		Resource: "annotations",
		Name:     nameFromRuntimeObject(obj),
		At:       time.Now(),
	})
	f.shared.mu.Unlock()
	// Persist to the in-memory object so subsequent cycles see the update
	// and the idempotency guard in ensureManagedAnnotations skips the patch.
	if mo, ok := obj.(metav1.Object); ok {
		mo.SetAnnotations(annotations)
	}
	return nil
}

func (f *FakeKubeclient) PatchStatus(_ context.Context, obj domain.Object, _ map[string]interface{}) error {
	f.shared.mu.Lock()
	f.shared.ops = append(f.shared.ops, Op{
		Cycle:    f.shared.currentCycle,
		Verb:     "patch",
		Resource: "status",
		Name:     obj.GetName(),
		At:       time.Now(),
	})
	f.shared.mu.Unlock()
	return nil
}

func stringMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func nameFromRuntimeObject(obj runtime.Object) string {
	if acc, ok := obj.(interface{ GetName() string }); ok {
		return acc.GetName()
	}
	return ""
}
