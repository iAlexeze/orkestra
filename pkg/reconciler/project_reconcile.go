package reconciler

import (
	"context"
	"fmt"

	projectTypev1 "github.com/ialexeze/multi-crd-controller/pkg/config/api/types/project/v1alpha1"
	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/event"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
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

	// Split the key into namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	// Get the object from the store
	obj, exists, err := r.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("failed to get object from store: %w", err)
	}

	if !exists {
		// Object was deleted
		logger.Info().Msgf("Project %s/%s has been deleted", namespace, name)
		// Perform any cleanup logic here
		return nil
	}

	// Type assert to project
	project, ok := obj.(*projectTypev1.Project)
	if !ok {
		return fmt.Errorf("expected *projectTypev1.Project, got %T", obj)
	}

	// Your reconciliation logic here
	logger.Info().Msgf("Reconciling project %s/%s (replicas: %d)",
		namespace, name, project.Spec.Replicas)

	// Example: Check if project needs finalizer
	if project.DeletionTimestamp != nil {
		// Handle deletion
		return r.handleDeletion(ctx, project)
	}

	// Normal reconciliation
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
	logger.Info().Msgf("Handling deletion for %s", project.Name)
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
