// pkg/reconciler/generic.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// GenericReconciler manages the full lifecycle of one CRD.
//
// This file is a pure dispatcher. It owns:
//   - Context enrichment
//   - Cache reads
//   - Deletion routing
//   - Finalizer/Annotation/Labele management
//   - Template interpretation
//   - Reconcile priority (Go hooks → declarative templates → no-op)
//   - Event firing and logging
//
// Resource-specific logic lives in separate files:
//
//	run_deployments.go    — Deployment create/update
//	run_services.go       — Service create/update
//	run_secrets.go        — Secret create/copy/sync
//	run_configmaps.go     — ConfigMap create/copy/sync
//	run_serviceaccounts.go — ServiceAccount create
//	run_jobs.go           — Job create (onDelete cleanup)
//	run_cronjobs.go       — CronJob create/update
//
// Adding a new resource type:
//  1. Add a file run_<resource>.go with a runXxx() function
//  2. Call it from runTemplateReconcile() and/or runTemplateOnDelete()
//  3. Add the field to orktypes.HookTemplates
//     That is all — generic.go does not change.
type GenericReconciler[T domain.Object] struct {
	informer cache.SharedIndexInformer
	event    *event.Event
	kube     *kubeclient.Kubeclient
	hooks    domain.ReconcileHooks[T]
	rc       orktypes.ReconcilerConfig
	newObj   func() T
	crd      CRDInfo
}

type CRDInfo struct {
	Kind             string
	GVK              string
	GVR              schema.GroupVersionResource
	Finalizers       []string
	ReconcilerConfig orktypes.ReconcilerConfig
	Operator         string
}

func NewGenericReconciler[T domain.Object](
	crd CRDInfo,
	informer cache.SharedIndexInformer,
	ev *event.Event,
	kube *kubeclient.Kubeclient,
	anyHooks domain.AnyReconcileHooks,
	newObj func() T,
) *GenericReconciler[T] {

	var hooks domain.ReconcileHooks[T]
	if anyHooks != nil {
		typed, ok := anyHooks.(domain.ReconcileHooks[T])
		if !ok {
			panic(fmt.Sprintf(
				"GenericReconciler[%T]: hooks type mismatch — got %T",
				newObj(), anyHooks,
			))
		}
		hooks = typed
	}

	return &GenericReconciler[T]{
		crd:      crd,
		rc:       crd.ReconcilerConfig,
		informer: informer,
		event:    ev,
		kube:     kube,
		hooks:    hooks,
		newObj:   newObj,
	}
}

var _ domain.Reconciler = (*GenericReconciler[domain.Object])(nil)

// Reconcile dispatches to the correct reconcile implementation.
// Order:
//  1. Conditional provisioning (when blocks) — handled by runTemplateReconcile
//  2. Go hooks → Declarative templates → No-op (through reconcileImpl())
func (r *GenericReconciler[T]) Reconcile(ctx context.Context, key string) error {
	ctx = kubeclient.WithKubeclient(ctx, r.kube)
	if err := ctx.Err(); err != nil {
		return err
	}

	ctx = logger.WithRequestID(ctx)
	ctx = logger.WithCRD(ctx, r.crd.GVK)
	ctx = logger.WithResource(ctx, key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid key %q: %w", key, err)
	}
	_ = namespace

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

	obj, ok := raw.(T)
	if !ok {
		return fmt.Errorf("type assertion failed: expected %T, got %T", r.newObj(), raw)
	}
	obj = obj.DeepCopyObject().(T)

	if obj.GetDeletionTimestamp() != nil {
		logger.FromContext(ctx).Info().
			Str("name", obj.GetName()).
			Msgf("deletion handler called for %s", r.crd.GVK)

		r.event.Eventf(obj, corev1.EventTypeNormal, "Deleting",
			fmt.Sprintf("Deleting %s %s/%s", r.crd.GVK, obj.GetNamespace(), obj.GetName()))

		return r.handleDeletion(ctx, obj)
	}

	if err := r.ensureFinalizers(ctx, obj); err != nil {
		r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.Kind+"FinalizerError",
			fmt.Sprintf("Failed to add finalizers: %v", err))
		return err
	}

	// Ensure managed label — idempotent, like finalizer patching.
	// This is how ork reconcile knows what this operator instance manages.
	if err := r.ensureManagedLabel(ctx, obj); err != nil {
		return err
	}

	if err := r.ensureManagedAnnotations(ctx, obj, r.crd.Operator); err != nil {
		return err
	}

	// ── Step 5: Reconcile implementation ──────────────────────────────────────
	return r.reconcileImpl(ctx, obj)
}

// reconcileImpl dispatches to the correct reconcile implementation.
// Priority: Go hooks → declarative templates → no-op.
func (r *GenericReconciler[T]) reconcileImpl(ctx context.Context, obj T) error {
	var err error

	switch {
	case r.hooks.OnReconcile != nil:
		// Go hooks — user-provided, full type-safe access.
		// Requires: ork generate runtime to register in HookRegistry.
		err = r.hooks.OnReconcile(ctx, obj)

	case r.rc.OnCreate != nil || r.rc.OnReconcile != nil:
		// Declarative templates — interpreted at runtime.
		// Requires: nothing. ork generate runtime NOT needed.
		err = r.runTemplateReconcile(ctx, obj)

	default:
		// No-op — finalizers, events, metrics still handled above.
		logger.FromContext(ctx).Info().
			Str("name", obj.GetName()).
			Msgf("reconciled %s (no-op)", r.crd.GVK)
		// Status still patched for no-op reconcilers
	}

	// Always patch status — best-effort, never fails reconcile.
	// Called with the outcome so Ready condition reflects reality.
	// Must run before the error return so Ready=False is written on failure.
	// r.updatedPatchStatus(ctx, obj, err)
	r.patchStatusWithChildren(ctx, obj, err) // Layer 3: read children only on success — no point reading

	if err != nil {
		logger.FromContext(ctx).Error().Err(err).
			Str("name", obj.GetName()).
			Msgf("reconciliation failed for %s", r.crd.GVK)

		r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.Kind+"ReconcileError",
			fmt.Sprintf("Failed to reconcile %s %s/%s: %v",
				r.crd.GVK, obj.GetNamespace(), obj.GetName(), err))
		return err
	}

	r.event.Eventf(obj, corev1.EventTypeNormal, r.crd.Kind+"Reconciled",
		fmt.Sprintf("Successfully reconciled %s %s/%s",
			r.crd.GVK, obj.GetNamespace(), obj.GetName()))

	logger.FromContext(ctx).Info().
		Str("name", obj.GetName()).
		Msgf("reconciled %s", r.crd.GVK)

	return nil
}

// handleDeletion runs cleanup then removes our finalizers.
// Finalizers are never removed on error — object stays protected until
// cleanup succeeds.
func (r *GenericReconciler[T]) handleDeletion(ctx context.Context, obj T) error {
	switch {
	case r.hooks.OnDelete != nil:
		if err := r.hooks.OnDelete(ctx, obj); err != nil {
			r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.Kind+"DeleteError",
				fmt.Sprintf("Deletion hook failed: %v", err))
			return fmt.Errorf("deletion hook: %w", err)
		}

	case r.rc.OnDelete != nil:
		if err := r.runTemplateOnDelete(ctx, obj); err != nil {
			r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.Kind+"DeleteError",
				fmt.Sprintf("Template deletion failed: %v", err))
			return fmt.Errorf("template deletion: %w", err)
		}
	}

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

// ── Template dispatch ─────────────────────────────────────────────────────────
// runTemplateReconcile and runTemplateOnDelete are the only places in this file
// that know which resource types exist. Adding a new resource type means adding
// one line here and one new run_<resource>.go file. generic.go changes no further.

// runTemplateReconcile interprets onCreate and onReconcile blocks.
// Each resource type is handled by its own run_xxx() function.
func (r *GenericReconciler[T]) runTemplateReconcile(ctx context.Context, obj domain.Object) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not found in context")
	}

	resolver, err := orktmpl.NewResolver(ctx, obj)
	if err != nil {
		return fmt.Errorf("building resolver: %w", err)
	}

	// ── onCreate ─────────────────────────────────────────────────────────────
	if t := r.rc.OnCreate; t != nil {
		if err := runDeployments(ctx, kube, resolver, obj, t.Deployments, false); err != nil {
			return err
		}
		if err := runServices(ctx, kube, resolver, obj, t.Services, false); err != nil {
			return err
		}
		if err := runSecrets(ctx, kube, resolver, obj, t.Secrets, false); err != nil {
			return err
		}
		if err := runConfigMaps(ctx, kube, resolver, obj, t.ConfigMaps, false); err != nil {
			return err
		}
		if err := runServiceAccounts(ctx, kube, resolver, obj, t.ServiceAccounts); err != nil {
			return err
		}
		if err := runCronJobs(ctx, kube, resolver, obj, t.CronJobs, false); err != nil {
			return err
		}
	}

	// ── onReconcile ──────────────────────────────────────────────────────────
	if t := r.rc.OnReconcile; t != nil {
		if err := runDeployments(ctx, kube, resolver, obj, t.Deployments, true); err != nil {
			return err
		}
		if err := runServices(ctx, kube, resolver, obj, t.Services, true); err != nil {
			return err
		}
		if err := runSecrets(ctx, kube, resolver, obj, t.Secrets, true); err != nil {
			return err
		}
		if err := runConfigMaps(ctx, kube, resolver, obj, t.ConfigMaps, true); err != nil {
			return err
		}
		if err := runCronJobs(ctx, kube, resolver, obj, t.CronJobs, true); err != nil {
			return err
		}
		// ServiceAccounts don't drift — no onReconcile needed
	}

	return nil
}

// runTemplateOnDelete interprets the onDelete block.
// Currently handles Jobs — the primary onDelete use case.
// Owner references handle all other resource cleanup automatically.
func (r *GenericReconciler[T]) runTemplateOnDelete(ctx context.Context, obj domain.Object) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not found in context")
	}

	resolver, err := orktmpl.NewResolver(ctx, obj)
	if err != nil {
		return fmt.Errorf("building resolver: %w", err)
	}

	if t := r.rc.OnDelete; t != nil {
		if err := runJobs(ctx, kube, resolver, obj, t.Jobs); err != nil {
			return err
		}
	}

	return nil
}
