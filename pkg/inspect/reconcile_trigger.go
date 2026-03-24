// pkg/inspect/reconcile_trigger.go
package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// ReconcileAnnotation is the annotation key written to trigger reconciliation.
// Orkestra's informer detects the metadata update and re-queues the object.
// The value is an RFC3339 timestamp — unique per trigger, readable in kubectl describe.
const ReconcileAnnotation = "orkestra.konductor.io/reconcile-at"

// TriggerResult holds the outcome of one reconcile trigger operation.
type TriggerResult struct {
	// Name — the CR name that was triggered
	Name string

	// Namespace — the CR namespace (empty for cluster-scoped)
	Namespace string

	// Error — non-nil if the trigger failed
	Error error
}

// TriggerReconcile patches the reconcile annotation on a single CR.
// The informer detects the metadata change and re-queues the object
// into the workqueue, causing Orkestra to reconcile it on the next loop.
//
// This is non-destructive — it only touches the annotation, never the spec.
func TriggerReconcile(
	ctx context.Context,
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	name string,
) error {
	patch := utils.BuildAnnotationPatch(ReconcileAnnotation, time.Now().UTC().Format(time.RFC3339))

	data, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshalling patch: %w", err)
	}

	var resource dynamic.ResourceInterface
	if namespace != "" {
		resource = client.Resource(gvr).Namespace(namespace)
	} else {
		resource = client.Resource(gvr)
	}

	_, err = resource.Patch(ctx, name, types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("patching %s/%s: %w", namespace, name, err)
	}

	return nil
}

// TriggerReconcileAll triggers reconciliation for every CR of a given CRD.
// Returns one TriggerResult per CR — failures are collected, not fatal.
// The caller decides how to handle partial failures.
func TriggerReconcileAll(
	ctx context.Context,
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string, // empty = all namespaces
) ([]TriggerResult, error) {
	var resource dynamic.ResourceInterface
	if namespace != "" {
		resource = client.Resource(gvr).Namespace(namespace)
	} else {
		resource = client.Resource(gvr)
	}

	list, err := resource.List(ctx, metav1.ListOptions{
		LabelSelector: konfig.LabelManaged + konfig.LabelManagedValue,
	})
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", gvr.Resource, err)
	}

	results := make([]TriggerResult, 0, len(list.Items))
	for _, item := range list.Items {
		result := TriggerResult{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		}
		result.Error = TriggerReconcile(ctx, client, gvr, item.GetNamespace(), item.GetName())
		results = append(results, result)
	}

	return results, nil
}
