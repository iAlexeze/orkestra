// pkg/reconciler/generic.go
package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

type GenericReconciler[T runtime.Object] struct {
	informer  cache.SharedIndexInformer
	event     *event.Event
	kube      *kubeclient.Kubeclient
	gvk       string
	hooks     domain.ReconcileHooks[T]
	newObj    func() T // factory — returns a zero-value T for type assertion
	finalizer string
}

func NewGenericReconciler[T runtime.Object](
	gvk string,
	informer cache.SharedIndexInformer,
	ev *event.Event,
	kube *kubeclient.Kubeclient,
	anyHooks domain.AnyReconcileHooks,
	newObj func() T,
) *GenericReconciler[T] {

	// Recover typed hooks — nil anyHooks means empty hooks
	var hooks domain.ReconcileHooks[T]
	if anyHooks != nil {
		typed, ok := anyHooks.(domain.ReconcileHooks[T])
		if !ok {
			// This is a programming error — hooks were registered for wrong type
			panic(fmt.Sprintf(
				"GenericReconciler[%T]: hooks type mismatch — got %T",
				newObj(), anyHooks,
			))
		}
		hooks = typed
	}

	return &GenericReconciler[T]{
		gvk:       gvk,
		informer:  informer,
		event:     ev,
		kube:      kube,
		hooks:     hooks,
		newObj:    newObj,
		finalizer: "orkestra.io/finalizer",
	}
}

/*
The panic here is intentional — a hooks type mismatch is a wiring bug, not a runtime error. It should be caught immediately on startup, not silently produce wrong behavior.

---

### The full picture

User writes ProjectHooks()          → ReconcileHooks[*projectv1.Project]
                │
                ▼
Katalog entry wraps it              → HookFactory: func() AnyReconcileHooks
                │
                ▼
buildManager calls HookFactory()    → AnyReconcileHooks (type-erased)
                │
                ▼
NewGenericReconcilerFromHooks[T]    → type assertion back to ReconcileHooks[T]
                │                     panics at startup if wrong type
                ▼
GenericReconciler[*projectv1.Project] owns typed hooks
                │
                ▼
Reconcile() calls hooks.OnReconcile(ctx, obj *projectv1.Project)
*/

var _ domain.Reconciler = (*GenericReconciler[runtime.Object])(nil)

func (r *GenericReconciler[T]) Reconcile(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return nil
	}

	// Context enrichment — same as your ProjectReconciler
	ctx = logger.WithRequestID(ctx)
	ctx = logger.WithCRD(ctx, r.gvk)
	ctx = logger.WithResource(ctx, key)

	start := time.Now()
	defer func() {
		metrics.ReconcileDuration.
			WithLabelValues(r.gvk).
			Observe(time.Since(start).Seconds())
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid key %q: %w", key, err)
	}
	_ = namespace

	// Read from cache
	raw, exists, err := r.informer.GetIndexer().GetByKey(key)
	if err != nil {
		metrics.ReconcileTotal.WithLabelValues(r.gvk, "error").Inc()
		return fmt.Errorf("getting %q from store: %w", key, err)
	}

	if !exists {
		logger.FromContext(ctx).Info().Msgf("%s/%s not found — deleted", namespace, name)
		if r.hooks.OnNotFound != nil {
			return r.hooks.OnNotFound(ctx, key)
		}
		return nil
	}

	// Type assertion via factory — safe, no panic
	obj, ok := raw.(T)
	if !ok {
		return fmt.Errorf("expected %T, got %T", r.newObj(), raw)
	}

	// Work on a deep copy — never mutate the cache
	obj = obj.DeepCopyObject().(T)

	// Ensure TypeMeta is set — lister strips it
	// The caller is responsible for setting this via a hook or we do it via GVK lookup
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("getting accessor: %w", err)
	}

	// Deletion path
	if accessor.GetDeletionTimestamp() != nil {
		logger.FromContext(ctx).Info().
			Str("name", accessor.GetName()).
			Msgf("deletion handler called")
		return r.handleDeletion(ctx, obj, accessor)
	}

	// Ensure finalizer is present
	if err := r.ensureFinalizer(ctx, obj, accessor); err != nil {
		return err
	}

	// Normal reconcile — call the hook
	if r.hooks.OnReconcile != nil {
		if err := r.hooks.OnReconcile(ctx, obj); err != nil {
			metrics.ReconcileTotal.WithLabelValues(r.gvk, "error").Inc()

			logger.FromContext(ctx).Error().Err(err).
				Str("name", accessor.GetName()).
				Msgf("reconciliation failed for %s", r.gvk)
			return err
		}
	}

	logger.FromContext(ctx).Info().
		Str("name", accessor.GetName()).
		Msgf("reconciled %s", r.gvk)

	metrics.ReconcileTotal.WithLabelValues(r.gvk, "success").Inc()
	return nil
}

func (r *GenericReconciler[T]) handleDeletion(
	ctx context.Context,
	obj T,
	accessor metav1.Object,
) error {
	// Call user hook first — cleanup external resources
	if r.hooks.OnDelete != nil {
		if err := r.hooks.OnDelete(ctx, obj); err != nil {
			return fmt.Errorf("deletion hook: %w", err)
		}
	}

	// Remove finalizer — unblocks Kubernetes GC
	return r.removeFinalizer(ctx, obj, accessor)
}

func (r *GenericReconciler[T]) removeFinalizer(
	ctx context.Context,
	obj T,
	accessor metav1.Object,
) error {
	finalizers := accessor.GetFinalizers()
	for i := 0; i < len(finalizers); i++ {
		if finalizers[i] == r.finalizer {
			finalizers = append(finalizers[:i], finalizers[i+1:]...)
			break
		}
	}

	if len(finalizers) == 0 {
		return nil
	}

	accessor.SetFinalizers(finalizers)
	return r.kube.PatchFinalizers(ctx, obj, r.kube.Info.GroupVersion.WithResource(r.kube.Info.Plural), finalizers)
}

func (r *GenericReconciler[T]) ensureFinalizer(
	ctx context.Context,
	obj T,
	accessor metav1.Object,
) error {
	finalizers := accessor.GetFinalizers()
	for _, f := range finalizers {
		if f == r.finalizer {
			return nil // already present
		}
	}
	// Add and patch via crdClient
	finalizers = append(finalizers, r.finalizer)
	accessor.SetFinalizers(finalizers)
	return r.kube.PatchFinalizers(ctx, obj, r.kube.Info.GroupVersion.WithResource(r.kube.Info.Plural), finalizers)
}
