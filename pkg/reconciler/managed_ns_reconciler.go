package reconciler

import (
	"context"
	"fmt"

	mnsTypev1 "github.com/ialexeze/multi-crd-controller/pkg/config/api/types/managedNamespace/v1alpha1"
	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/event"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/kubeclient"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type ManagedNamespaceReconciler struct {
	kube     *kubeclient.Kubeclient
	informer cache.SharedIndexInformer
	event    *event.Event
}

func NewManagedNamespaceReconciler(
	kube *kubeclient.Kubeclient,
	informer cache.SharedIndexInformer,
	event *event.Event,
) *ManagedNamespaceReconciler {
	return &ManagedNamespaceReconciler{
		kube:     kube,
		informer: informer,
		event:    event,
	}
}

var _ domain.Reconciler = (*ManagedNamespaceReconciler)(nil)

func (r *ManagedNamespaceReconciler) ShutDown() {}

// Reconcile is called for every ManagedNamespace event.
// key = "name" (cluster-scoped, no namespace prefix).
func (r *ManagedNamespaceReconciler) Reconcile(ctx context.Context, key string) error {
	// if err := ctx.Err(); err != nil {
	// 	return nil // context cancelled — clean exit
	// }
	// Check if context is cancelled

	if err := ctx.Err(); err != nil {
		return err
	}

	// Read from local cache
	obj, exists, err := r.informer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("failed to get object from store: %w", err)
	}

	if !exists || obj == nil {
		// Deleted — child resources cleaned up by owner references
		logger.Info().Str("name", key).Msg("ManagedNamespace deleted — nothing to do")
		return nil
	}

	// Type assert to ManagedNamespace
	mnRaw, ok := obj.(*mnsTypev1.ManagedNamespace)
	if !ok {
		return fmt.Errorf("expected *mnsTypev1.ManagedNamespace, got %T", obj)
	}

	// Always work on a deep copy — never mutate the cached object
	mn := mnRaw.DeepCopy()

	// Reconcile the Namespace this CR owns and child resources
	reconcileErr := r.reconcileNamespace(ctx, mn)

	// Set condition even in error
	if reconcileErr != nil {
		mn.Status.Phase = string(mnsTypev1.Failed)
		r.setCondition(
			mn,
			mnsTypev1.Error,
			metav1.ConditionFalse,
			"ReconcileError",
			reconcileErr.Error(),
		)
	} else {
		mn.Status.Phase = string(mnsTypev1.Active)
		r.setCondition(
			mn,
			mnsTypev1.Ready,
			metav1.ConditionTrue,
			"Reconciled",
			"All resources reconciled successfully",
		)
	}

	// Always patch status — even if reconcile failed
	// Status patch goes to /status endpoint — does not bump generation, no extra watch event
	if err := r.patchStatus(ctx, mn); err != nil {
		logger.Error().Err(err).Str("name", mn.Name).Msg("failed to patch status")
		// Non-fatal — log and continue. Status being stale is better than
		// masking the original reconcile error.
	}

	// Return original reconcile error
	if reconcileErr != nil {
		return fmt.Errorf("reconciling namespace for %q: %w", key, err)
	}

	logger.Info().
		Str("name", mn.Name).
		Str("team", mn.Spec.Team).
		Msg("reconciling ManagedNamespace")

	// Add more child reconcilers here as you build:
	// r.reconcileResourceQuota(ctx, mn)
	// r.reconcileNetworkPolicy(ctx, mn)
	// r.reconcileRoleBindings(ctx, mn)

	// Add event
	r.event.Recorder().
		Eventf(
			mn,
			corev1.EventTypeNormal,
			"Reconciled",
			"ManagedNamespace %s reconciled successfully", mn.Name,
		)

	logger.Info().
		Str("name", mn.Name).
		Msg("ManagedNamespace reconciled successfully")
	return nil
}

// reconcileNamespace ensures a Namespace exists with the correct labels.
// Idempotent — safe to call on every reconcile.
func (r *ManagedNamespaceReconciler) reconcileNamespace(ctx context.Context, mn *mnsTypev1.ManagedNamespace) error {
	// Check if context is cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	// Check if name exists
	if mn.Name == "" {
		return fmt.Errorf("ManagedNamespace name is empty")
	}
	nsName := mn.Name // or mn.Spec.Team

	existing, err := r.kube.Clientset().CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("getting namespace %q: %w", nsName, err)
		}

		// Doesn't exist — create it
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
				Labels: map[string]string{
					"managed-by":                   "kube-controller",
					"team":                         mn.Spec.Team,
					"platform.ialexeze.io/managed": "true",
				},
				// Owner reference — when ManagedNamespace is deleted,
				// k8s GC deletes this Namespace automatically.
				// Note: owner refs only work within same namespace for namespace-scoped
				// resources. For cluster-scoped Namespaces owned by cluster-scoped CRDs
				// this works correctly.
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: mnsTypev1.GroupVersion.String(),
						Kind:       "ManagedNamespace",
						Name:       mn.Name,
						UID:        mn.UID,
						Controller: boolPtr(true),
					},
				},
			},
		}

		if _, err := r.kube.Clientset().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("creating namespace %q: %w", nsName, err)
		}

		logger.Info().Str("namespace", nsName).Msg("namespace created")
		return nil
	}

	// Already exists — ensure labels are correct (drift detection)
	if existing.Labels["team"] != mn.Spec.Team {
		existing.Labels["team"] = mn.Spec.Team
		if _, err := r.kube.Clientset().CoreV1().Namespaces().Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("updating namespace %q labels: %w", nsName, err)
		}
		logger.Info().Str("namespace", nsName).Msg("namespace labels updated")
	}

	return nil
}

func boolPtr(b bool) *bool { return &b }
