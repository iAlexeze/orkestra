package reconciler

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	"github.com/ialexeze/orkestra/domain"
// 	"github.com/ialexeze/orkestra/pkg/event"
// 	"github.com/ialexeze/orkestra/pkg/kubeclient"
// 	orktypes "github.com/ialexeze/orkestra/pkg/types"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	"k8s.io/apimachinery/pkg/runtime/schema"
// 	"k8s.io/client-go/dynamic/fake"
// 	"k8s.io/client-go/tools/cache"
// 	"k8s.io/client-go/tools/record"
// )

// // testObject is a minimal domain.Object implementation for testing.
// type testObject struct {
// 	metav1.ObjectMeta
// 	Spec   map[string]interface{}
// 	Status map[string]interface{}
// }

// func (t *testObject) DeepCopyObject() runtime.Object {
// 	copy := &testObject{
// 		ObjectMeta: *t.ObjectMeta.DeepCopy(),
// 	}
// 	if t.Spec != nil {
// 		copy.Spec = make(map[string]interface{})
// 		for k, v := range t.Spec {
// 			copy.Spec[k] = v
// 		}
// 	}
// 	if t.Status != nil {
// 		copy.Status = make(map[string]interface{})
// 		for k, v := range t.Status {
// 			copy.Status[k] = v
// 		}
// 	}
// 	return copy.DeepCopyObject()
// }

// func (t *testObject) GetSpec() interface{} {
// 	return t.Spec
// }

// func (t *testObject) GetStatus() interface{} {
// 	return t.Status
// }

// func (t *testObject) SetSpec(spec interface{}) {
// 	if m, ok := spec.(map[string]interface{}); ok {
// 		t.Spec = m
// 	}
// }

// func (t *testObject) SetStatus(status interface{}) {
// 	if m, ok := status.(map[string]interface{}); ok {
// 		t.Status = m
// 	}
// }

// // fakeInformer implements a simple cache.Indexer for testing.
// type fakeInformer struct {
// 	store cache.Indexer
// }

// func (f *fakeInformer) AddIndexers(indexers cache.Indexers) error {
// 	return nil
// }

// func (f *fakeInformer) GetStore() cache.Store {
// 	return f.store
// }

// func (f *fakeInformer) GetController() cache.Controller {
// 	return nil
// }

// func (f *fakeInformer) Run(stopCh <-chan struct{}) {
// }

// func (f *fakeInformer) HasSynced() bool {
// 	return true
// }

// func (f *fakeInformer) LastSyncResourceVersion() string {
// 	return ""
// }

// func (f *fakeInformer) AddEventHandler(handler cache.ResourceEventHandler) {
// }

// func (f *fakeInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) {
// }

// func (f *fakeInformer) GetIndexer() cache.Indexer {
// 	return f.store
// }

// func newFakeInformer() *fakeInformer {
// 	return &fakeInformer{
// 		store: cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{}),
// 	}
// }

// func newTestObject(name, namespace string) *testObject {
// 	return &testObject{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      name,
// 			Namespace: namespace,
// 			Finalizers: []string{},
// 		},
// 		Spec:   make(map[string]interface{}),
// 		Status: make(map[string]interface{}),
// 	}
// }

// func TestGenericReconciler_Reconcile_NotFound(t *testing.T) {
// 	// Setup
// 	informer := newFakeInformer()
// 	recorder := record.NewFakeRecorder(100)
// 	eventRecorder := event.NewEvent(recorder)

// 	reconciler := NewGenericReconciler(
// 		CRDInfo{
// 			Kind: "Test",
// 			GVK:  "test.io/v1, Kind=Test",
// 			GVR:  schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "tests"},
// 		},
// 		informer,
// 		eventRecorder,
// 		nil, // kube
// 		nil, // hooks
// 		func() domain.Object { return newTestObject("", "") },
// 	)

// 	ctx := context.Background()

// 	// Test reconciling a non-existent object
// 	err := reconciler.Reconcile(ctx, "default/missing")
// 	assert.NoError(t, err, "reconcile should not error on missing object")
// }

// func TestGenericReconciler_Reconcile_Deletion(t *testing.T) {
// 	// Setup
// 	informer := newFakeInformer()
// 	obj := newTestObject("test-delete", "default")
// 	obj.DeletionTimestamp = &metav1.Time{Time: time.Now()}
// 	obj.Finalizers = []string{"test.io/finalizer"}
// 	require.NoError(t, informer.store.Add(obj))

// 	recorder := record.NewFakeRecorder(100)
// 	eventRecorder := event.NewEventWithRecorder(recorder)

// 	reconciler := NewGenericReconciler(
// 		CRDInfo{
// 			Kind: "Test",
// 			GVK:  "test.io/v1, Kind=Test",
// 			GVR:  schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "tests"},
// 			Finalizers: []string{"test.io/finalizer"},
// 		},
// 		informer,
// 		eventRecorder,
// 		nil, // kube
// 		nil, // hooks
// 		func() domain.Object { return newTestObject("", "") },
// 	)

// 	ctx := context.Background()

// 	err := reconciler.Reconcile(ctx, "default/test-delete")
// 	assert.NoError(t, err)

// 	// Verify finalizers were removed
// 	updated, exists, err := informer.store.GetByKey("default/test-delete")
// 	require.True(t, exists)
// 	require.NoError(t, err)
// 	updatedObj := updated.(*testObject)
// 	assert.Empty(t, updatedObj.Finalizers, "finalizers should be removed on deletion")
// }

// func TestGenericReconciler_Reconcile_AddsFinalizers(t *testing.T) {
// 	// Setup
// 	informer := newFakeInformer()
// 	obj := newTestObject("test-add-finalizer", "default")
// 	require.NoError(t, informer.store.Add(obj))

// 	recorder := record.NewFakeRecorder(100)
// 	eventRecorder := event.NewEventWithRecorder(recorder)

// 	reconciler := NewGenericReconciler(
// 		CRDInfo{
// 			Kind: "Test",
// 			GVK:  "test.io/v1, Kind=Test",
// 			GVR:  schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "tests"},
// 			Finalizers: []string{"test.io/finalizer"},
// 		},
// 		informer,
// 		eventRecorder,
// 		nil, // kube
// 		nil, // hooks
// 		func() domain.Object { return newTestObject("", "") },
// 	)

// 	ctx := context.Background()

// 	err := reconciler.Reconcile(ctx, "default/test-add-finalizer")
// 	assert.NoError(t, err)

// 	// Verify finalizers were added
// 	updated, exists, err := informer.store.GetByKey("default/test-add-finalizer")
// 	require.True(t, exists)
// 	require.NoError(t, err)
// 	updatedObj := updated.(*testObject)
// 	assert.Contains(t, updatedObj.Finalizers, "test.io/finalizer", "finalizer should be added")
// }

// func TestGenericReconciler_Reconcile_WithHooks(t *testing.T) {
// 	// Setup
// 	informer := newFakeInformer()
// 	obj := newTestObject("test-hooks", "default")
// 	require.NoError(t, informer.store.Add(obj))

// 	recorder := record.NewFakeRecorder(100)
// 	eventRecorder := event.NewEventWithRecorder(recorder)

// 	hookCalled := false
// 	hooks := domain.ReconcileHooks[*testObject]{
// 		OnReconcile: func(ctx context.Context, obj *testObject) error {
// 			hookCalled = true
// 			return nil
// 		},
// 	}

// 	reconciler := NewGenericReconciler(
// 		CRDInfo{
// 			Kind: "Test",
// 			GVK:  "test.io/v1, Kind=Test",
// 			GVR:  schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "tests"},
// 		},
// 		informer,
// 		eventRecorder,
// 		nil,
// 		hooks,
// 		func() domain.Object { return newTestObject("", "") },
// 	)

// 	ctx := context.Background()

// 	err := reconciler.Reconcile(ctx, "default/test-hooks")
// 	assert.NoError(t, err)
// 	assert.True(t, hookCalled, "OnReconcile hook should be called")
// }

// func TestGenericReconciler_Reconcile_WithDeclarativeTemplates(t *testing.T) {
// 	// Setup
// 	informer := newFakeInformer()
// 	obj := newTestObject("test-template", "default")
// 	obj.Spec["image"] = "nginx:latest"
// 	obj.Spec["replicas"] = int64(3)
// 	require.NoError(t, informer.store.Add(obj))

// 	// Create a fake dynamic client for testing
// 	scheme := runtime.NewScheme()
// 	dynamicClient := fake.NewSimpleDynamicClient(scheme)

// 	// Create a kubeclient with the fake client
// 	kube := &kubeclient.Kubeclient{}
// 	// Note: In a real test, you'd set up the kubeclient properly.
// 	// This is simplified for the example.

// 	recorder := record.NewFakeRecorder(100)
// 	eventRecorder := event.NewEventWithRecorder(recorder)

// 	reconciler := NewGenericReconciler(
// 		CRDInfo{
// 			Kind: "Test",
// 			GVK:  "test.io/v1, Kind=Test",
// 			GVR:  schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "tests"},
// 			ReconcilerConfig: orktypes.ReconcilerConfig{
// 				OnCreate: &orktypes.HookTemplates{
// 					Deployments: []orktypes.DeploymentTemplateSource{
// 						{
// 							Name:      "test-deployment",
// 							Image:     "{{ .spec.image }}",
// 							Replicas:  "{{ .spec.replicas }}",
// 							Namespace: "{{ .metadata.namespace }}",
// 						},
// 					},
// 				},
// 			},
// 		},
// 		informer,
// 		eventRecorder,
// 		kube,
// 		nil,
// 		func() domain.Object { return newTestObject("", "") },
// 	)

// 	ctx := context.Background()
// 	ctx = kubeclient.WithKubeclient(ctx, kube)

// 	err := reconciler.Reconcile(ctx, "default/test-template")
// 	assert.NoError(t, err)
// }

// func TestGenericReconciler_Reconcile_NoOp(t *testing.T) {
// 	// Setup
// 	informer := newFakeInformer()
// 	obj := newTestObject("test-noop", "default")
// 	require.NoError(t, informer.store.Add(obj))

// 	recorder := record.NewFakeRecorder(100)
// 	eventRecorder := event.NewEventWithRecorder(recorder)

// 	reconciler := NewGenericReconciler(
// 		CRDInfo{
// 			Kind: "Test",
// 			GVK:  "test.io/v1, Kind=Test",
// 			GVR:  schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "tests"},
// 			ReconcilerConfig: orktypes.ReconcilerConfig{
// 				// No hooks, no templates — no-op
// 			},
// 		},
// 		informer,
// 		eventRecorder,
// 		nil,
// 		nil,
// 		func() domain.Object { return newTestObject("", "") },
// 	)

// 	ctx := context.Background()

// 	err := reconciler.Reconcile(ctx, "default/test-noop")
// 	assert.NoError(t, err)
// }

// func TestGenericReconciler_handleDeletion_WithHook(t *testing.T) {
// 	// Setup
// 	informer := newFakeInformer()
// 	obj := newTestObject("test-delete-hook", "default")
// 	obj.Finalizers = []string{"test.io/finalizer"}
// 	require.NoError(t, informer.store.Add(obj))

// 	recorder := record.NewFakeRecorder(100)
// 	eventRecorder := event.NewEventWithRecorder(recorder)

// 	deleteCalled := false
// 	hooks := domain.ReconcileHooks[*testObject]{
// 		OnDelete: func(ctx context.Context, obj *testObject) error {
// 			deleteCalled = true
// 			return nil
// 		},
// 	}

// 	reconciler := NewGenericReconciler(
// 		CRDInfo{
// 			Kind: "Test",
// 			GVK:  "test.io/v1, Kind=Test",
// 			GVR:  schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "tests"},
// 			Finalizers: []string{"test.io/finalizer"},
// 		},
// 		informer,
// 		eventRecorder,
// 		nil,
// 		hooks,
// 		func() domain.Object { return newTestObject("", "") },
// 	)

// 	ctx := context.Background()

// 	// Call handleDeletion directly
// 	err := reconciler.handleDeletion(ctx, obj)
// 	assert.NoError(t, err)
// 	assert.True(t, deleteCalled, "OnDelete hook should be called")

// 	// Verify finalizers were removed
// 	updated, exists, err := informer.store.GetByKey("default/test-delete-hook")
// 	require.True(t, exists)
// 	require.NoError(t, err)
// 	updatedObj := updated.(*testObject)
// 	assert.Empty(t, updatedObj.Finalizers, "finalizers should be removed after deletion hook")
// }

// func TestGenericReconciler_handleDeletion_HookError(t *testing.T) {
// 	// Setup
// 	informer := newFakeInformer()
// 	obj := newTestObject("test-delete-error", "default")
// 	obj.Finalizers = []string{"test.io/finalizer"}
// 	require.NoError(t, informer.store.Add(obj))

// 	recorder := record.NewFakeRecorder(100)
// 	eventRecorder := event.NewEventWithRecorder(recorder)

// 	hooks := domain.ReconcileHooks[*testObject]{
// 		OnDelete: func(ctx context.Context, obj *testObject) error {
// 			return assert.AnError
// 		},
// 	}

// 	reconciler := NewGenericReconciler(
// 		CRDInfo{
// 			Kind: "Test",
// 			GVK:  "test.io/v1, Kind=Test",
// 			GVR:  schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "tests"},
// 			Finalizers: []string{"test.io/finalizer"},
// 		},
// 		informer,
// 		eventRecorder,
// 		nil,
// 		hooks,
// 		func() domain.Object { return newTestObject("", "") },
// 	)

// 	ctx := context.Background()

// 	err := reconciler.handleDeletion(ctx, obj)
// 	assert.Error(t, err, "deletion should return error when hook fails")

// 	// Verify finalizers were NOT removed
// 	updated, exists, err := informer.store.GetByKey("default/test-delete-error")
// 	require.True(t, exists)
// 	require.NoError(t, err)
// 	updatedObj := updated.(*testObject)
// 	assert.Contains(t, updatedObj.Finalizers, "test.io/finalizer", "finalizers should remain when deletion hook fails")
// }

// func TestGenericReconciler_ensureFinalizers_NoOpWhenNoneConfigured(t *testing.T) {
// 	// Setup
// 	informer := newFakeInformer()
// 	obj := newTestObject("test-no-finalizer-config", "default")
// 	require.NoError(t, informer.store.Add(obj))

// 	recorder := record.NewFakeRecorder(100)
// 	eventRecorder := event.NewEventWithRecorder(recorder)

// 	reconciler := NewGenericReconciler(
// 		CRDInfo{
// 			Kind: "Test",
// 			GVK:  "test.io/v1, Kind=Test",
// 			GVR:  schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "tests"},
// 			Finalizers: []string{}, // No finalizers configured
// 		},
// 		informer,
// 		eventRecorder,
// 		nil,
// 		nil,
// 		func() domain.Object { return newTestObject("", "") },
// 	)

// 	ctx := context.Background()

// 	err := reconciler.ensureFinalizers(ctx, obj)
// 	assert.NoError(t, err)

// 	// Verify no finalizers were added
// 	updated, exists, err := informer.store.GetByKey("default/test-no-finalizer-config")
// 	require.True(t, exists)
// 	require.NoError(t, err)
// 	updatedObj := updated.(*testObject)
// 	assert.Empty(t, updatedObj.Finalizers, "no finalizers should be added when none configured")
// }

// func TestGenericReconciler_ensureManagedLabel(t *testing.T) {
// 	// Setup
// 	informer := newFakeInformer()
// 	obj := newTestObject("test-label", "default")
// 	require.NoError(t, informer.store.Add(obj))

// 	recorder := record.NewFakeRecorder(100)
// 	eventRecorder := event.NewEventWithRecorder(recorder)

// 	reconciler := NewGenericReconciler(
// 		CRDInfo{
// 			Kind: "Test",
// 			GVK:  "test.io/v1, Kind=Test",
// 			GVR:  schema.GroupVersionResource{Group: "test.io", Version: "v1", Resource: "tests"},
// 		},
// 		informer,
// 		eventRecorder,
// 		nil,
// 		nil,
// 		func() domain.Object { return newTestObject("", "") },
// 	)

// 	ctx := context.Background()

// 	err := reconciler.ensureManagedLabel(ctx, obj)
// 	assert.NoError(t, err)

// 	// Verify managed label was added
// 	updated, exists, err := informer.store.GetByKey("default/test-label")
// 	require.True(t, exists)
// 	require.NoError(t, err)
// 	updatedObj := updated.(*testObject)
// 	assert.Equal(t, "orkestra", updatedObj.Labels["managed-by"], "managed-by label should be set to orkestra")
// 	assert.Equal(t, "test-label", updatedObj.Labels["orkestra-owner"], "orkestra-owner label should be set to object name")
// }
