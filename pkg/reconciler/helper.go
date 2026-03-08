package reconciler

import (
	"context"
	"encoding/json"
	"fmt"

	mnsTypev1 "github.com/ialexeze/multi-crd-controller/pkg/config/api/types/managedNamespace/v1alpha1"
	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/event"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/kubeclient"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	// "k8s.io/apimachinery/pkg/types"
)

type NewReconcilerFunc func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer, ev *event.Event) domain.Reconciler

func (r *ManagedNamespaceReconciler) patchStatus(
	ctx context.Context, mn *mnsTypev1.ManagedNamespace,
) error {
	// Check if context is cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	body, err := json.Marshal(map[string]interface{}{
		"status": map[string]interface{}{
			"phase":      mn.Status.Phase,
			"conditions": mn.Status.Conditions,
		},
	})
	if err != nil {
		return fmt.Errorf("marshalling status patch: %w", err)
	}

	logger.Debug().Msgf("patching status: %s", string(body))
	return nil
	// return r.kube.RestClient().Patch(types.MergePatchType).
	// 	Resource(mnsTypev1.NamePlural). // ← must match spec.names.plural in the CRD YAML
	// 	Name(mn.Name).
	// 	SubResource("status"). // ← /status endpoint, not the main object
	// 	Body(body).
	// 	Do(ctx).
	// 	Error()
}

// Helper to set conditions
func (r *ManagedNamespaceReconciler) setCondition(
	mn *mnsTypev1.ManagedNamespace,
	conditionType mnsTypev1.ManagedNamespaceConditionType,
	status metav1.ConditionStatus,
	reason string,
	message string,
) {
	now := metav1.Now()
	for i, c := range mn.Status.Conditions {
		if c.Type == string(conditionType) {
			// check whether any changes really happened
			// Only update LastTransitionTime if status actually changed
			if status != c.Status {
				mn.Status.Conditions[i].LastTransitionTime = now
			}
			mn.Status.Conditions[i].Status = status
			mn.Status.Conditions[i].Reason = reason
			mn.Status.Conditions[i].Message = message
			mn.Status.Conditions[i].ObservedGeneration = mn.Generation
			return
		}

	}
	// Not found? append new conditions
	mn.Status.Conditions = append(mn.Status.Conditions, metav1.Condition{
		Type:               string(conditionType),
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
		ObservedGeneration: mn.Generation,
	})

	// Alex notes:
	// ObservedGeneration on the condition tells observers
	// which version of the spec this condition reflects.
	// Always set it.
}
