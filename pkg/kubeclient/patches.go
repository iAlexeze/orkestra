package kubeclient

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// PatchFinalizers replaces the object's finalizer list with finalizers by
// sending a minimal JSON Merge Patch that touches only metadata.finalizers.
// Sending only the changed field avoids resourceVersion conflicts that would
// arise from patching the full object.
func (k *Kubeclient) PatchFinalizers(
	ctx context.Context,
	obj runtime.Object,
	gvr schema.GroupVersionResource,
	finalizers []string,
) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("getting accessor: %w", err)
	}

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": finalizers,
		},
	}

	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshalling finalizer patch: %w", err)
	}

	namespace := accessor.GetNamespace()
	name := accessor.GetName()

	if namespace == "" {
		// Cluster-scoped resource
		_, err = k.dynamic.Resource(gvr).Patch(
			ctx,
			name,
			types.MergePatchType,
			body,
			metav1.PatchOptions{},
		)
	} else {
		// Namespace-scoped resource
		_, err = k.dynamic.Resource(gvr).Namespace(namespace).Patch(
			ctx,
			name,
			types.MergePatchType,
			body,
			metav1.PatchOptions{},
		)
	}

	return err
}

// PatchLabels transitions the object's labels from base to desired by sending
// a JSON Merge Patch. Keys present in base but absent in desired are set to
// null so the server deletes them. Keys in desired that differ from base are
// added or updated. Unchanged keys are omitted from the patch body.
//
// base must be a snapshot of the labels as they exist on the server immediately
// before any in-memory mutations are applied (the controller-runtime MergeFrom
// pattern). Pass nil for base when the object is brand-new and has no labels.
func (k *Kubeclient) PatchLabels(
	ctx context.Context,
	obj runtime.Object,
	gvr schema.GroupVersionResource,
	base, desired map[string]string,
) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("getting accessor: %w", err)
	}

	// Build the label map for the merge patch body.
	// null  → server deletes the key (keys present in base but removed from desired)
	// value → server adds or updates the key
	// omit  → server leaves the key unchanged
	labelPatch := make(map[string]interface{})
	for key := range base {
		if _, ok := desired[key]; !ok {
			labelPatch[key] = nil
		}
	}
	for key, val := range desired {
		if base[key] != val {
			labelPatch[key] = val
		}
	}
	if len(labelPatch) == 0 {
		return nil
	}

	body, err := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{"labels": labelPatch},
	})
	if err != nil {
		return fmt.Errorf("marshalling label patch: %w", err)
	}

	ns := accessor.GetNamespace()
	name := accessor.GetName()

	if ns == "" {
		_, err = k.dynamic.Resource(gvr).Patch(
			ctx, name, types.MergePatchType, body, metav1.PatchOptions{},
		)
	} else {
		_, err = k.dynamic.Resource(gvr).Namespace(ns).Patch(
			ctx, name, types.MergePatchType, body, metav1.PatchOptions{},
		)
	}

	return err
}

// PatchAnnotations merges annotations onto the object by sending a JSON Merge
// Patch containing only the desired annotation keys. Keys present in annotations
// are added or updated on the server; keys absent from the patch are left
// unchanged (not deleted).
//
// This is intentionally a one-way merge: Orkestra's annotation management
// only ever adds keys (managed-by and managed-since are write-once and never
// removed). If key deletion is ever needed, mirror the base/desired pattern
// used by [PatchLabels].
func (k *Kubeclient) PatchAnnotations(
	ctx context.Context,
	obj runtime.Object,
	gvr schema.GroupVersionResource,
	annotations map[string]string,
) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("getting accessor: %w", err)
	}

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": annotations,
		},
	}

	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshalling annotation patch: %w", err)
	}

	ns := accessor.GetNamespace()
	name := accessor.GetName()

	if ns == "" {
		_, err = k.dynamic.Resource(gvr).Patch(
			ctx, name, types.MergePatchType, body, metav1.PatchOptions{},
		)
	} else {
		_, err = k.dynamic.Resource(gvr).Namespace(ns).Patch(
			ctx, name, types.MergePatchType, body, metav1.PatchOptions{},
		)
	}

	return err
}
