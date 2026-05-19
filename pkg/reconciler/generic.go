// pkg/reconciler/generic.go
package reconciler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/autoscaler"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kordinator"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/notification"
	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orkqueue "github.com/orkspace/orkestra/pkg/queue"
	orktypes "github.com/orkspace/orkestra/pkg/types"
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
//   - Finalizer/Annotation/Label management
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
//
// Type parameter PTR:
//
// PTR must be a pointer to the concrete CR struct (e.g. *Database).
// This matches Kubernetes informer semantics: the informer stores pointer values
// so the type assertion raw.(PTR) in reconcileCore succeeds only for pointer types.
// When used through the dynamic registry path in konstructor.go, PTR is inferred
// as domain.Object (the interface), which also satisfies the constraint and is safe
// because the informer cache always holds the correct underlying concrete type.
// See pkg/reconciler/ptr_hooks.go for the full design rationale.
type GenericReconciler[PTR domain.Object] struct {
	katalogRegistry   *kordinator.ResourceKatalog
	crdHealthRegistry map[string]*kordinator.CRDHealth
	providerRegistry  orktypes.ProviderRegistry
	providerStats     providerStatsRecorder
	informer          cache.SharedIndexInformer
	event             event.Recorder
	kube              kubeclient.KubeClient
	// hooks holds type-erased, domain.Object-parameterized callbacks built at
	// construction time from the user's ReconcileHooks[PTR]. Stored as
	// ObjectHooks rather than ReconcileHooks[PTR] so the reconciler remains
	// compatible with the runtime registry path (PTR = domain.Object interface).
	hooks       domain.ObjectHooks
	operatorBox orktypes.OperatorBoxConfig
	newObj      func() PTR
	crd         orktypes.CRDEntry
	kat         *katalog.Katalog

	// workerSem gates concurrent reconcile execution. All worker goroutines run
	// continuously; the semaphore controls how many may be in Reconcile simultaneously.
	// Resized at runtime by the autoscaler when autoscale: is declared.
	workerSem *autoscaler.ResizableSemaphore

	// autoMetrics holds live operatorbox runtime metrics for autoscale evaluation.
	autoMetrics *autoscaler.AutoMetrics

	// autoscaler is non-nil when operatorBox.autoscale is declared.
	autoscaler *autoscaler.Autoscaler

	// queue is the per-CRD workqueue, injected by startCRDWorkers after construction.
	// Used by SetQueueDepthLimit and the resync goroutine.
	queue *orkqueue.Workqueue

	// resyncNs holds the current resync interval in nanoseconds.
	// 0 means the resync goroutine is idle (informer handles baseline resync).
	// Written by SetResyncInterval; read by the resync goroutine.
	resyncNs atomic.Int64

	// Notification
	notifStack *notification.NotificationStack

	// rollbackHistory tracks per-CR failure timestamps for window-based rollback triggers.
	// Key: "namespace/name". Guarded by rollbackMu.
	rollbackHistory map[string]*rollbackFailureHistory
	rollbackMu      sync.Mutex

	// spawnWorker is injected by kordinator after construction. Called by ResizeWorkers
	// when scaling up to start additional goroutines matching the new semaphore capacity.
	// nil when autoscale is not declared or kordinator hasn't injected it yet.
	spawnWorker func()

	// rollbackNotifier is injected by kordinator after construction. Called when
	// rollback is triggered or cleared so CRDHealth can track rollback stats.
	rollbackTriggerFn func()
	rollbackClearFn   func()
}

// NewGenericReconciler constructs a GenericReconciler for the given CRD.
//
// PTR must be a pointer to the concrete CR type (e.g. *Database). When called
// from the runtime registry path in konstructor.go, PTR is inferred as
// domain.Object (the interface) — this is also valid because the constraint
// domain.Object is satisfied and the informer stores the correct concrete type.
//
// anyHooks, if non-nil, must implement domain.HookBinder. Every
// domain.ReconcileHooks[T] value satisfies HookBinder automatically via its
// BindToObjectHooks() method. Passing any other type panics at startup.
func NewGenericReconciler[PTR domain.Object](
	crd orktypes.CRDEntry,
	informer cache.SharedIndexInformer,
	ev event.Recorder,
	kube kubeclient.KubeClient,
	anyHooks domain.AnyReconcileHooks,
	newObj func() PTR,
	katalogRegistry *kordinator.ResourceKatalog,
	crdHealthRegistry map[string]*kordinator.CRDHealth,
	providerRegistry orktypes.ProviderRegistry,
	providerStats providerStatsRecorder,
	kat *katalog.Katalog,
) *GenericReconciler[PTR] {

	// Adapt the user's strongly-typed ReconcileHooks[PTR] to the type-erased
	// ObjectHooks stored on the reconciler. BindToObjectHooks wraps each hook
	// in a closure that performs obj.(PTR) at call time — safe because the
	// informer cache only ever holds objects of the type it was built for.
	var hooks domain.ObjectHooks
	if anyHooks != nil {
		binder, ok := anyHooks.(domain.HookBinder)
		if !ok {
			panic(fmt.Sprintf(
				"NewGenericReconciler[%T]: hooks value must implement domain.HookBinder "+
					"(got %T) — use domain.ReconcileHooks[*YourType]{...} or a type that "+
					"wraps one and forwards BindToObjectHooks()",
				newObj(), anyHooks,
			))
		}
		hooks = binder.BindToObjectHooks()
	}

	workers := crd.Workers
	if workers <= 0 {
		workers = 1
	}
	sem := autoscaler.NewResizableSemaphore(workers)
	autoMet := autoscaler.NewAutoMetrics(sem)

	r := &GenericReconciler[PTR]{
		katalogRegistry:   katalogRegistry,
		crdHealthRegistry: crdHealthRegistry,
		providerRegistry:  providerRegistry,
		providerStats:     providerStats,
		crd:               crd,
		operatorBox:       crd.OperatorBox,
		informer:          informer,
		event:             ev,
		kube:              kube,
		hooks:             hooks,
		newObj:            newObj,
		workerSem:         sem,
		autoMetrics:       autoMet,
		rollbackHistory:   make(map[string]*rollbackFailureHistory),
		kat:               kat,
	}

	if crd.AutoscaleEnabled() {
		baseline := orktypes.AutoscaleBaseline{
			Workers:       workers,
			MaxQueueDepth: crd.Queue.MaxQueueDepth,
			Resync:        crd.Resync,
		}
		r.autoscaler = autoscaler.NewAutoscaler(
			crd.APITypes.Kind,
			crd.OperatorBox.Autoscale,
			baseline,
			r,
			autoMet,
		)
	}

	// Wire notification: GatewayNotifier when a gateway endpoint is configured;
	// DirectNotifier otherwise (standalone SMTP/Slack dispatch on the runtime).
	if kat != nil && crd.IsNotificationEnabled() {
		var notifier notification.Notifier
		if ep := kat.GatewayEndpoint(); ep != "" {
			notifier = notification.NewGatewayNotifier(ep)
		} else {
			notifier = notification.NewDirectNotifier(kat)
		}
		r.notifStack = notification.NewNotificationStack(kat, notifier)
	}

	return r
}

var _ domain.Reconciler = (*GenericReconciler[domain.Object])(nil)

// Reconcile dispatches to the correct reconcile implementation.
// Order:
//  1. Conditional provisioning (when blocks) — handled by runTemplateReconcile
//  2. Go hooks → Declarative templates → No-op (through reconcileImpl())
//
// The semaphore gates concurrent execution — when an autoscaler is active it
// can reduce effective concurrency below the goroutine count without stopping goroutines.
func (r *GenericReconciler[PTR]) Reconcile(ctx context.Context, key string) error {
	if err := r.workerSem.Acquire(ctx); err != nil {
		return err // context cancelled while waiting for a concurrency slot
	}
	start := time.Now()
	err := r.reconcileCore(ctx, key)
	r.workerSem.Release()
	r.autoMetrics.RecordReconcile(time.Since(start), err != nil)
	return err
}

func (r *GenericReconciler[PTR]) reconcileCore(ctx context.Context, key string) error {
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

	obj, ok := raw.(PTR)
	if !ok {
		return fmt.Errorf("type assertion failed: expected %T, got %T", r.newObj(), raw)
	}
	rawObj := obj.DeepCopyObject().(PTR)

	// Normalize before mutation/validation/template rendering ─────────────
	// Normalize + base resolver
	obj, resolver, err := r.applyNormalize(ctx, rawObj)
	if err != nil {
		return err
	}

	// ──────────────────────────────────────────────────────────────────────────────
	// GVK FIX: typed objects from the informer cache may arrive without a valid
	// GroupVersionKind on the very first reconcile after a CR is created.
	// This occurs because the watch event from the API server sometimes omits
	// the TypeMeta fields (`apiVersion`, `kind`). The subsequent reconcile loop
	// (e.g., after an operator restart) or a full list operation does include them.
	//
	// The effect: without a correct GVK, owner references created by hooks or
	// registry functions become invalid, causing API server rejections like
	//
	//   metadata.ownerReferences.apiVersion: Invalid value: "": version must not be empty
	//
	// The fix: set the missing GVK using the known values from the CRD entry,
	// which Orkestra parsed during startup from the Katalog. This ensures every
	// typed object presented to hooks and child‑resource creation carries a
	// complete TypeMeta.
	//
	// See: https://github.com/orkspace/orkestra/issues/85
	// ──────────────────────────────────────────────────────────────────────────────
	if obj.GetObjectKind().GroupVersionKind().Empty() {
		obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   r.crd.APITypes.Group,
			Version: r.crd.APITypes.Version,
			Kind:    r.crd.APITypes.Kind,
		})
	}
	// Check if resource is being deleted
	if obj.GetDeletionTimestamp() != nil {
		logger.FromContext(ctx).Info().
			Str("name", obj.GetName()).
			Msgf("deletion handler called for %s", r.crd.GVKString())

		r.event.Eventf(obj, corev1.EventTypeNormal, "Deleting",
			fmt.Sprintf("Deleting %s %s/%s", r.crd.GVKString(), obj.GetNamespace(), obj.GetName()))

		return r.handleDeletion(ctx, resolver, obj)
	}

	// Namespace guard — skip reconcile for CRs in restricted or non-allowed namespaces.
	// Deletion is always allowed so finalizers can be removed; this guard runs after
	// the deletion-timestamp check so deleting CRs are never blocked.
	if r.crd.HasNamespaceRules() {
		result := CheckNamespace(ctx, obj, obj.GetNamespace(), r.crd.RestrictedNamespaces, r.crd.AllowedNamespaces, r.crd.APITypes.Kind)
		if !result.Allowed {
			logger.FromContext(ctx).Debug().
				Str("name", obj.GetName()).
				Str("namespace", obj.GetNamespace()).
				Str("reason", result.Reason).
				Msg("reconcile: skipping CR in restricted/non-allowed namespace")
			return nil
		}
	}

	// Ensure finalizers
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
	// Labels
	if err := r.ensureManagedLabel(ctx, obj); err != nil {
		return err
	}

	// Annotations
	if err := r.ensureManagedAnnotations(ctx, obj, r.crd.KatalogName); err != nil {
		return err
	}

	// ── Step 5: Reconcile implementation ──────────────────────────────────────
	return r.reconcileImpl(ctx, resolver, obj)
}

// reconcileImpl dispatches to the correct reconcile implementation.
// Priority: Go hooks → declarative templates → no-op.
//
// Rollback phase order:
//  1. Rollback gate  — if rollback is active, re-apply previous spec and return
//  2. Snapshot       — on success, capture current spec as rollback baseline
//  3. Mutation/validation
//  4. Reconcile dispatch
//  5. Failure trigger check — record failure; trigger rollback if threshold met
//  6. Status patch
func (r *GenericReconciler[PTR]) reconcileImpl(ctx context.Context, resolver *orktmpl.Resolver, obj PTR) error {
	var err error

	// ── Phase 1: Rollback gate ────────────────────────────────────────────────
	if isRollbackActive(obj) {
		logger.FromContext(ctx).Info().
			Str("name", obj.GetName()).
			Msg("rollback: active — blocking normal reconcile")
		if rbErr := r.runRollback(ctx, resolver, obj); rbErr != nil {
			logger.FromContext(ctx).Error().Err(rbErr).
				Str("name", obj.GetName()).
				Msg("rollback: failed to re-apply previous state")
		}
		r.patchStatusWithChildren(ctx, obj, resolver, fmt.Errorf("rollback active"))
		return nil // do not propagate — stays in rollback loop
	}

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
		// Requires: ork generate registry to register in HookRegistry.
		err = r.hooks.OnReconcile(ctx, obj)

	case r.operatorBox.OnCreate != nil || r.operatorBox.OnReconcile != nil:
		// Declarative templates — interpreted at runtime.
		// Requires: nothing. ork generate registry NOT needed.
		// The returned resolver carries cross/external/git data for status evaluation.
		resolver, err = r.runTemplateReconcile(ctx, resolver, obj)

	default:
		// No-op — finalizers, events, metrics still handled above.
		logger.FromContext(ctx).Info().
			Str("name", obj.GetName()).
			Msgf("reconciled %s (no-op)", r.crd.GVKString())
		// Status still patched for no-op reconcilers
	}

	// ── Phase 5: Rollback trigger check ─────────────────────────────────────
	if err != nil && r.crd.HasRollbackRules() {
		key := obj.GetNamespace() + "/" + obj.GetName()
		h := r.getFailureHistory(key)
		derived := r.crd.OperatorBox.DerivedRollback()
		h.record(derived.Trigger.EffectiveConsecutiveFailures())
		if r.shouldRollback(len(h.times), h) {
			logger.FromContext(ctx).Warn().
				Str("name", obj.GetName()).
				Msg("rollback: threshold reached — marking rollback active")
			if markErr := r.markRollbackActive(ctx, obj); markErr != nil {
				logger.FromContext(ctx).Error().Err(markErr).Msg("rollback: failed to mark active")
			}
		}
	}

	// ── Phase 6: Snapshot + rollback cleanup ─────────────────────────────────
	if err == nil && !r.crd.HasRollbackRules() {
		// If a prior rollback cycle resolved (user corrected spec, generation
		// advanced), clear the stale RollbackGenerationAnnotation and notify
		// CRDHealth. snapshotSpec re-writes PreviousSpecAnnotation immediately after.
		annots := obj.GetAnnotations()
		if annots[orktypes.RollbackGenerationAnnotation] != "" {
			if clrErr := r.clearRollback(ctx, obj); clrErr != nil {
				logger.FromContext(ctx).Warn().Err(clrErr).
					Str("name", obj.GetName()).
					Msg("rollback: failed to clear stale rollback annotation — continuing")
			}
		}
		if snapErr := r.snapshotSpec(ctx, obj); snapErr != nil {
			logger.FromContext(ctx).Warn().Err(snapErr).
				Str("name", obj.GetName()).
				Msg("rollback: failed to snapshot spec — continuing")
		}
		r.clearFailureHistory(obj.GetNamespace() + "/" + obj.GetName())
	}

	// Inject live runtime metrics into the resolver so status.fields templates
	// can reference .metrics.queueDepth, .metrics.workers, .metrics.autoscaleActive, etc.
	metricsMap := r.autoMetrics.AsMap()
	if r.autoscaler != nil {
		if snap := r.autoscaler.Snapshot(); snap != nil {
			metricsMap["autoscaleActive"] = snap.OverrideActive
		}
	} else {
		metricsMap["autoscaleActive"] = false
	}
	resolver = resolver.WithMetrics(metricsMap)

	// Inject live runtime health into the resolver so status.fields templates
	// can reference .health.healthy, .health.state, .health.uptime,
	// .health.totalReconciles, .health.lastError, etc.
	//
	// This surfaces the operatorbox health endpoint directly into templates,
	// enabling CR status fields to show live reconcile health, uptime,
	// dependency health, and error information without any API calls.
	if h, ok := r.crdHealthRegistry[r.crd.GVK().String()]; ok {
		resolver = resolver.WithHealth(h.HealthAsMap())
	}

	// Always patch status — best-effort, never fails reconcile.
	// Called with the outcome so Ready condition reflects reality.
	// Must run before the error return so Ready=False is written on failure.
	r.patchStatusWithChildren(ctx, obj, resolver, err)

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
func (r *GenericReconciler[PTR]) namespaceAllowed(
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
func (r *GenericReconciler[PTR]) handleDeletion(ctx context.Context, resolver *orktmpl.Resolver, obj PTR) error {
	switch {
	case r.hooks.OnDelete != nil:
		if err := r.hooks.OnDelete(ctx, obj); err != nil {
			r.event.Eventf(obj, corev1.EventTypeWarning, r.crd.APITypes.Kind+"DeleteError",
				fmt.Sprintf("Deletion hook failed: %v", err))
			return fmt.Errorf("deletion hook: %w", err)
		}

	case r.operatorBox.OnDelete != nil:
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
