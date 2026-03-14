// pkg/reconciler/generic.go
package reconciler

import (
	"context"
	"fmt"
	"slices"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type GenericReconciler[T domain.Object] struct {
	informer cache.SharedIndexInformer
	event    *event.Event
	kube     *kubeclient.Kubeclient
	hooks    domain.ReconcileHooks[T]
	newObj   func() T // factory — returns a zero-value T for type assertion
	crd      CRDInfo
}

type CRDInfo struct {
	Kind       string
	GVK        string
	GVR        schema.GroupVersionResource
	Finalizers []string
}

func NewGenericReconciler[T domain.Object](
	crd CRDInfo,
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
			// Programming error — hooks registered for wrong type.
			// Panic at startup, not silently at runtime.
			panic(fmt.Sprintf(
				"GenericReconciler[%T]: hooks type mismatch — got %T",
				newObj(), anyHooks,
			))
		}
		hooks = typed
	}

	return &GenericReconciler[T]{
		crd:      crd,
		informer: informer,
		event:    ev,
		kube:     kube,
		hooks:    hooks,
		newObj:   newObj,
	}
}

var _ domain.Reconciler = (*GenericReconciler[domain.Object])(nil)

func (r *GenericReconciler[T]) Reconcile(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return nil
	}

	ctx = logger.WithRequestID(ctx)
	ctx = logger.WithCRD(ctx, r.crd.GVK)
	ctx = logger.WithResource(ctx, key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid key %q: %w", key, err)
	}
	_ = namespace

	// Read from cache — never hits the API server
	raw, exists, err := r.informer.GetIndexer().GetByKey(key)
	if err != nil {
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

	// Always work on a deep copy — never mutate the cached object
	obj = obj.DeepCopyObject().(T)

	// Deletion path — obj.GetDeletionTimestamp() directly, no accessor needed
	if obj.GetDeletionTimestamp() != nil {
		logger.FromContext(ctx).Info().
			Str("name", obj.GetName()).
			Str("namespace", obj.GetNamespace()).
			Msgf("deletion handler called for %s", r.crd.GVK)

		r.event.Eventf(obj, corev1.EventTypeNormal, "Deleting",
			fmt.Sprintf("Deleting %s %s/%s", r.crd.GVK, obj.GetNamespace(), obj.GetName()))

		return r.handleDeletion(ctx, obj)
	}

	// Ensure finalizers are present before any reconcile logic
	if err := r.ensureFinalizers(ctx, obj); err != nil {
		r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.Kind+"FinalizerError",
			fmt.Sprintf("Failed to add finalizers: %v", err))
		return err
	}

	// Normal reconcile — call the hook if provided
	if r.hooks.OnReconcile != nil {
		if err := r.hooks.OnReconcile(ctx, obj); err != nil {

			logger.FromContext(ctx).Error().Err(err).
				Str("name", obj.GetName()).
				Str("namespace", obj.GetNamespace()).
				Msgf("reconciliation failed for %s", r.crd.GVK)

			r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.Kind+"ReconcileError",
				fmt.Sprintf("Failed to reconcile %s %s/%s: %v",
					r.crd.GVK, obj.GetNamespace(), obj.GetName(), err))

			return err
		}

		r.event.Eventf(obj, corev1.EventTypeNormal, r.crd.Kind+"Reconciled",
			fmt.Sprintf("Successfully reconciled %s %s/%s",
				r.crd.GVK, obj.GetNamespace(), obj.GetName()))
	}

	logger.FromContext(ctx).Info().
		Str("name", obj.GetName()).
		Str("namespace", obj.GetNamespace()).
		Msgf("reconciled %s", r.crd.GVK)

	return nil
}

func (r *GenericReconciler[T]) handleDeletion(ctx context.Context, obj T) error {
	// Call user hook first — cleanup external resources before finalizer is removed
	if r.hooks.OnDelete != nil {
		if err := r.hooks.OnDelete(ctx, obj); err != nil {
			r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.Kind+"DeleteError",
				fmt.Sprintf("Deletion hook failed: %v", err))
			return fmt.Errorf("deletion hook: %w", err)
		}
	}

	// Remove our finalizers — unblocks Kubernetes GC
	if err := r.removeFinalizers(ctx, obj); err != nil {
		r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.Kind+"FinalizerRemovalError",
			fmt.Sprintf("Failed to remove finalizers: %v", err))
		return err
	}

	r.event.Eventf(obj, corev1.EventTypeNormal, r.crd.Kind+"Deleted",
		fmt.Sprintf("Successfully deleted %s %s/%s",
			r.crd.GVK, obj.GetNamespace(), obj.GetName()))

	return nil
}

func (r *GenericReconciler[T]) ensureFinalizers(ctx context.Context, obj T) error {
	if len(r.crd.Finalizers) == 0 {
		return nil // no finalizers configured for this CRD
	}

	needsUpdate := false
	for _, f := range r.crd.Finalizers {
		if !ContainsFinalizer(obj, f) {
			needsUpdate = true
			break
		}
	}

	if !needsUpdate {
		return nil // all finalizers already present
	}

	// Add any missing finalizers
	newFinalizers := obj.GetFinalizers()
	for _, f := range r.crd.Finalizers {
		if !ContainsFinalizer(obj, f) {
			newFinalizers = append(newFinalizers, f)
		}
	}

	logger.Debug().
		Str("gvr", r.crd.GVR.String()).
		Str("name", obj.GetName()).
		Str("namespace", obj.GetNamespace()).
		Msgf("adding finalizers: %v -> %v", obj.GetFinalizers(), newFinalizers)

	r.event.Eventf(obj, corev1.EventTypeNormal, r.crd.Kind+"FinalizerAdded",
		fmt.Sprintf("Added finalizers to %s %s/%s",
			r.crd.GVK, obj.GetNamespace(), obj.GetName()))

	return r.kube.PatchFinalizers(ctx, obj, r.crd.GVR, newFinalizers)
}

func (r *GenericReconciler[T]) removeFinalizers(ctx context.Context, obj T) error {
	if len(obj.GetFinalizers()) == 0 {
		return nil
	}

	// Keep only finalizers that aren't ours
	newFinalizers := make([]string, 0, len(obj.GetFinalizers()))
	for _, f := range obj.GetFinalizers() {
		if !slices.Contains(r.crd.Finalizers, f) {
			newFinalizers = append(newFinalizers, f)
		}
	}

	// Nothing changed — all finalizers were foreign, nothing to remove
	if len(newFinalizers) == len(obj.GetFinalizers()) {
		return nil
	}

	logger.Debug().
		Str("gvr", r.crd.GVR.String()).
		Str("name", obj.GetName()).
		Str("namespace", obj.GetNamespace()).
		Msgf("removing finalizers: %v -> %v", obj.GetFinalizers(), newFinalizers)

	return r.kube.PatchFinalizers(ctx, obj, r.crd.GVR, newFinalizers)
}

// ── Finalizer helpers ─────────────────────────────────────────────────────────
// Exported so custom reconcilers can use them directly without going through
// the GenericReconciler.

// AddFinalizer adds the finalizer to obj if not already present.
// Returns true if the finalizer list was modified.
func AddFinalizer(o domain.Object, finalizer string) (updated bool) {
	if ContainsFinalizer(o, finalizer) {
		return false
	}
	o.SetFinalizers(append(o.GetFinalizers(), finalizer))
	return true
}

// RemoveFinalizer removes the finalizer from obj if present.
// Returns true if the finalizer list was modified.
func RemoveFinalizer(o domain.Object, finalizer string) (updated bool) {
	f := o.GetFinalizers()
	length := len(f)

	index := 0
	for i := range length {
		if f[i] == finalizer {
			continue
		}
		f[index] = f[i]
		index++
	}
	o.SetFinalizers(f[:index])
	return length != index
}

// ContainsFinalizer returns true if obj has the given finalizer.
func ContainsFinalizer(o domain.Object, finalizer string) bool {
	return slices.Contains(o.GetFinalizers(), finalizer)
}
