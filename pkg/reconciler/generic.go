// pkg/reconciler/generic.go
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/kordinator"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktmpl "github.com/ialexeze/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	corev1 "k8s.io/api/core/v1"
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
	katalogRegistry  *kordinator.ResourceKatalog
	providerRegistry orktypes.ProviderRegistry
	informer         cache.SharedIndexInformer
	event            *event.Event
	kube             *kubeclient.Kubeclient
	hooks            domain.ReconcileHooks[T]
	rc               orktypes.ReconcilerConfig
	newObj           func() T
	crd              orktypes.CRDEntry
}

func NewGenericReconciler[T domain.Object](
	crd orktypes.CRDEntry,
	informer cache.SharedIndexInformer,
	ev *event.Event,
	kube *kubeclient.Kubeclient,
	anyHooks domain.AnyReconcileHooks,
	newObj func() T,
	katalogRegistry *kordinator.ResourceKatalog,
	providerRegistry orktypes.ProviderRegistry,
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
		katalogRegistry:  katalogRegistry,
		providerRegistry: providerRegistry,
		crd:              crd,
		rc:               crd.ReconcilerConfig,
		informer:         informer,
		event:            ev,
		kube:             kube,
		hooks:            hooks,
		newObj:           newObj,
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
	ctx = logger.WithCRD(ctx, r.crd.GVKString())
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
	rawObj := obj.DeepCopyObject().(T)

	// Normalize before mutation/validation/template rendering ─────────────
	// Normalize + base resolver
	obj, resolver, err := r.applyNormalize(ctx, rawObj)
	if err != nil {
		return err
	}

	if obj.GetDeletionTimestamp() != nil {
		logger.FromContext(ctx).Info().
			Str("name", obj.GetName()).
			Msgf("deletion handler called for %s", r.crd.GVKString())

		r.event.Eventf(obj, corev1.EventTypeNormal, "Deleting",
			fmt.Sprintf("Deleting %s %s/%s", r.crd.GVKString(), obj.GetNamespace(), obj.GetName()))

		return r.handleDeletion(ctx, resolver, obj)
	}

	if !r.crd.RemoveFinalizers {
		if err := r.ensureFinalizers(ctx, obj); err != nil {
			r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.APITypes.Kind+"FinalizerError",
				fmt.Sprintf("Failed to add finalizers: %v", err))
			return err
		}
	} else {
		logger.FromContext(ctx).Debug().Msgf("removing finalizers for %s", obj.GetName())
		if err := r.removeFinalizers(ctx, obj); err != nil {
			r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.APITypes.Kind+"FinalizerRemovalError",
				fmt.Sprintf("Failed to remove finalizers: %v", err))
			return err
		}
		logger.FromContext(ctx).Debug().Msgf("finalizers removed for %s", obj.GetName())
	}

	// Ensure managed label and annotations — idempotent, like finalizer patching.
	// This is how ork reconcile knows what this operator instance manages.
	if err := r.ensureManagedLabel(ctx, obj); err != nil {
		return err
	}

	if err := r.ensureManagedAnnotations(ctx, obj, r.crd.KatalogName); err != nil {
		return err
	}

	// ── Step 5: Reconcile implementation ──────────────────────────────────────
	return r.reconcileImpl(ctx, resolver, obj)
}

// reconcileImpl dispatches to the correct reconcile implementation.
// Priority: Go hooks → declarative templates → no-op.
func (r *GenericReconciler[T]) reconcileImpl(ctx context.Context, resolver *orktmpl.Resolver, obj T) error {
	var err error

	// ── Reconcile-time mutation and validation ────────────────────────────────
	// Ordering respects MutationConfig.MutateFirst:
	//   false (default) — validate → mutate valid objects → reconcile
	//   true            — mutate first (apply defaults) → validate → reconcile
	//
	// Mutation failures are non-fatal: logged, reconcile continues.
	// Validation deny failures halt reconcile and return an error.

	// ── Reconcile-time mutation and validation ────────────────────────────────
	if r.crd.HasMutationRules() && r.crd.Mutation.MutateFirst {
		if mutErr := r.applyReconcileTimeMutation(ctx, resolver, obj); mutErr != nil {
			logger.FromContext(ctx).Warn().Err(mutErr).
				Str("name", obj.GetName()).
				Msg("reconcile mutation failed — continuing")
		}
	}

	if r.crd.HasValidationRules() {
		valResult := runValidation(obj, r.crd.Validation, r.crd.APITypes.Kind)

		// Warn violations: log and emit events but do NOT halt
		for _, w := range valResult.Warnings {
			logger.FromContext(ctx).Warn().
				Str("name", obj.GetName()).
				Str("crd", r.crd.GVKString()).
				Str("resource", obj.GetNamespace()+"/"+obj.GetName()).
				Str("field", w.Field).
				Str("message", w.Message).
				Msg("reconcile validation: warn")
			r.event.Eventf(obj, corev1.EventTypeWarning, "ValidationWarning",
				fmt.Sprintf("field %q: %s", w.Field, w.Message))
		}

		// Deny violations: halt reconcile
		if valResult.Deny {
			return valResult.DenialError()
		}
	}

	if r.crd.HasMutationRules() && !r.crd.Mutation.MutateFirst {
		if mutErr := r.applyReconcileTimeMutation(ctx, resolver, obj); mutErr != nil {
			logger.FromContext(ctx).Warn().Err(mutErr).
				Str("name", obj.GetName()).
				Msg("reconcile mutation failed — continuing")
		}
	}
	switch {
	case r.hooks.OnReconcile != nil:
		// Go hooks — user-provided, full type-safe access.
		// Requires: ork generate runtime to register in HookRegistry.
		err = r.hooks.OnReconcile(ctx, obj)

	case r.rc.OnCreate != nil || r.rc.OnReconcile != nil:
		// Declarative templates — interpreted at runtime.
		// Requires: nothing. ork generate runtime NOT needed.
		err = r.runTemplateReconcile(ctx, resolver, obj)

	default:
		// No-op — finalizers, events, metrics still handled above.
		logger.FromContext(ctx).Info().
			Str("name", obj.GetName()).
			Msgf("reconciled %s (no-op)", r.crd.GVKString())
		// Status still patched for no-op reconcilers
	}

	// Always patch status — best-effort, never fails reconcile.
	// Called with the outcome so Ready condition reflects reality.
	// Must run before the error return so Ready=False is written on failure.
	// r.updatedPatchStatus(ctx, obj, err)
	r.patchStatusWithChildren(ctx, obj, resolver, err) // Layer 3: read children only on success — no point reading

	if err != nil {
		logger.FromContext(ctx).Error().Err(err).
			Str("name", obj.GetName()).
			Msgf("reconciliation failed for %s", r.crd.GVKString())

		r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.APITypes.Kind+"ReconcileError",
			fmt.Sprintf("Failed to reconcile %s %s/%s: %v",
				r.crd.GVKString(), obj.GetNamespace(), obj.GetName(), err))
		return err
	}

	r.event.Eventf(obj, corev1.EventTypeNormal, r.crd.APITypes.Kind+"Reconciled",
		fmt.Sprintf("Successfully reconciled %s %s/%s",
			r.crd.GVKString(), obj.GetNamespace(), obj.GetName()))

	logger.FromContext(ctx).Info().
		Str("name", obj.GetName()).
		Msgf("reconciled %s", r.crd.GVKString())

	return nil
}

// namespaceAllowed returns true when the target namespace passes both the
// restricted and allowed namespace checks for this CRD.
// Called inside runResourceGroup before dispatching to each resource type.
func (r *GenericReconciler[T]) namespaceAllowed(
	ctx context.Context,
	obj domain.Object,
	targetNamespace string,
) bool {
	result := CheckNamespace(
		ctx,
		obj,
		targetNamespace,
		r.crd.RestrictedNamespaces,
		r.crd.AllowedNamespaces,
		r.crd.APITypes.Kind,
	)
	return result.Allowed
}

// handleDeletion runs cleanup then removes our finalizers.
// Finalizers are never removed on error — object stays protected until
// cleanup succeeds.
func (r *GenericReconciler[T]) handleDeletion(ctx context.Context, resolver *orktmpl.Resolver, obj T) error {
	switch {
	case r.hooks.OnDelete != nil:
		if err := r.hooks.OnDelete(ctx, obj); err != nil {
			r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.APITypes.Kind+"DeleteError",
				fmt.Sprintf("Deletion hook failed: %v", err))
			return fmt.Errorf("deletion hook: %w", err)
		}

	case r.rc.OnDelete != nil:
		if err := r.runTemplateOnDelete(ctx, resolver, obj); err != nil {
			r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.APITypes.Kind+"DeleteError",
				fmt.Sprintf("Template deletion failed: %v", err))
			return fmt.Errorf("template deletion: %w", err)
		}
	}

	if err := r.removeFinalizers(ctx, obj); err != nil {
		r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.APITypes.Kind+"FinalizerRemovalError",
			fmt.Sprintf("Failed to remove finalizers: %v", err))
		return err
	}

	r.event.Eventf(obj, corev1.EventTypeNormal, r.crd.APITypes.Kind+"Deleted",
		fmt.Sprintf("Successfully deleted %s %s/%s",
			r.crd.GVKString(), obj.GetNamespace(), obj.GetName()))

	return nil
}
