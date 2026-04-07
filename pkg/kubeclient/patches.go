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

// Patch Finalizers
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

	// Build a minimal merge patch — only touch finalizers
	// Never send the full object — avoids resourceVersion conflicts
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

// Patch Labels
func (k *Kubeclient) PatchLabels(
	ctx context.Context,
	obj runtime.Object,
	gvr schema.GroupVersionResource,
	labels map[string]string,
) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("getting accessor: %w", err)
	}

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": labels,
		},
	}

	body, err := json.Marshal(patch)
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
