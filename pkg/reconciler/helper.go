package reconciler

import (
	"context"
	"encoding/json"
	"fmt"

	mnsTypev1 "github.com/ialexeze/orkestra/api/types/managedNamespace/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

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

	restClient, err := r.kube.RestClientFor(mnsTypev1.APIPath, mnsTypev1.Group, mnsTypev1.Version)
	if err != nil {
		return fmt.Errorf("getting rest client: %w", err)
	}

	return restClient.Patch(types.MergePatchType).
		Resource(mnsTypev1.Plural). // ← must match spec.names.plural in the CRD YAML
		Name(mn.Name).
		SubResource("status"). // ← /status endpoint, not the main object
		Body(body).
		Do(ctx).
		Error()
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
