package reconciler

import (
	"context"
	"fmt"
	"time"

	projectTypev1 "github.com/ialexeze/orkestra/api/types/project/v1alpha1"
	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type ProjectReconciler struct {
	informer cache.SharedIndexInformer
	event    *event.Event
}

func NewProjectReconciler(
	informer cache.SharedIndexInformer,
	event *event.Event,
) *ProjectReconciler {
	return &ProjectReconciler{
		informer: informer,
		event:    event,
	}
}

var _ domain.Reconciler = (*ProjectReconciler)(nil)

func (r *ProjectReconciler) ShutDown() {}

// Reconcile handles the actual business logic for a project
func (r *ProjectReconciler) Reconcile(ctx context.Context, key string) error {
	// Check if context is cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	// Log with context
	ctx = logger.WithRequestID(ctx)
	ctx = logger.WithCRD(ctx, "projects")
	ctx = logger.WithResource(ctx, key)

	start := time.Now()

	logger.FromContext(ctx).Info().Msg("starting reconciliation")
	// Always record duration
	defer func() {
		metrics.ReconcileDuration.
			WithLabelValues("project").
			Observe(time.Since(start).Seconds())
	}()

	// Split the key into namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	// Get the object from the store
	obj, exists, err := r.informer.GetIndexer().GetByKey(key)
	if err != nil {
		// Log metrics to prometheus
		metrics.ReconcileTotal.WithLabelValues("project", "error").Inc()

		return fmt.Errorf("failed to get object from store: %w", err)
	}

	if !exists {
		// Object was deleted
		logger.FromContext(ctx).Info().Msgf("Project %s/%s has been deleted", namespace, name)
		// Perform any cleanup logic here
		return nil
	}

	// Type assert to project
	project, ok := obj.(*projectTypev1.Project)
	if !ok {
		return fmt.Errorf("expected *projectTypev1.Project, got %T", obj)
	}

	// Your reconciliation logic here
	logger.FromContext(ctx).Info().Msgf("Reconciling project %s/%s (replicas: %d)",
		namespace, name, project.Spec.Replicas)

	// Example: Check if project needs finalizer
	if project.DeletionTimestamp != nil {
		// Handle deletion
		return r.handleDeletion(ctx, project)
	}

	// Normal reconciliation
	metrics.ReconcileTotal.WithLabelValues("project", "success").Inc()
	return r.reconcileNormal(ctx, project)
}

func (r *ProjectReconciler) reconcileNormal(ctx context.Context, project *projectTypev1.Project) error {
	// Add your business logic here
	// e.g., ensure dependent resources exist, update status, etr.

	if r.event.Recorder() != nil {
		r.event.Recorder().Eventf(
			project,
			corev1.EventTypeNormal,
			"ProjectReconciled",
			"%s project reconciled successfully", project.Name,
		)
	}
	logger.Debug().Msgf("Normal reconciliation for %s", project.Name)
	return nil
}

func (r *ProjectReconciler) handleDeletion(ctx context.Context, project *projectTypev1.Project) error {
	logger.FromContext(ctx).Info().Msgf("Handling deletion for %s", project.Name)
	// Add cleanup logic here
	// e.g., delete external resources, remove finalizers

	// Emit events
	if r.event.Recorder() != nil {
		r.event.Recorder().Eventf(
			project,
			corev1.EventTypeWarning,
			"ProjectDelete",
			"%s project deleted from %s namespace", project.Name, project.Namespace,
		)
	}
	return nil
}
