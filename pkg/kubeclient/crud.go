package kubeclient

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// gvrFor derives the GroupVersionResource for obj using the scheme and mapper.
func (k *Kubeclient) gvrFor(obj runtime.Object) (*meta.RESTMapping, error) {
	gvks, _, err := k.scheme.ObjectKinds(obj)
	if err != nil {
		return nil, fmt.Errorf("unknown type %T: %w", obj, err)
	}
	gvk := gvks[0]
	mapping, err := k.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("no REST mapping for %s: %w", gvk, err)
	}
	return mapping, nil
}

// Get fetches the object identified by namespace/name into into.
// Derives the API group and resource from the Go type via the scheme and mapper.
func (k *Kubeclient) Get(ctx context.Context, namespace, name string, into sigs.Object) error {
	mapping, err := k.gvrFor(into)
	if err != nil {
		return err
	}

	var u *unstructured.Unstructured
	if namespace == "" {
		u, err = k.dynamic.Resource(mapping.Resource).Get(ctx, name, metav1.GetOptions{})
	} else {
		u, err = k.dynamic.Resource(mapping.Resource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return err
	}

	return runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, into)
}

// Create creates obj in the cluster.
// Derives the API group and resource from the Go type via the scheme and mapper.
func (k *Kubeclient) Create(ctx context.Context, obj sigs.Object) error {
	mapping, err := k.gvrFor(obj)
	if err != nil {
		return err
	}

	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return fmt.Errorf("convert to unstructured: %w", err)
	}
	u := &unstructured.Unstructured{Object: raw}
	ns := obj.GetNamespace()

	if ns == "" {
		_, err = k.dynamic.Resource(mapping.Resource).Create(ctx, u, metav1.CreateOptions{})
	} else {
		_, err = k.dynamic.Resource(mapping.Resource).Namespace(ns).Create(ctx, u, metav1.CreateOptions{})
	}
	return err
}

// Patch applies patch to obj in the cluster.
// The patch body and type are computed by calling patch.Data(obj).
// Use kubeclient.MergeFrom or kubeclient.StrategicMergeFrom to build the patch,
// or pass sigs.MergeFrom / sigs.StrategicMergeFrom directly — they satisfy Patch.
func (k *Kubeclient) Patch(ctx context.Context, obj sigs.Object, patch Patch) error {
	mapping, err := k.gvrFor(obj)
	if err != nil {
		return err
	}

	data, err := patch.Data(obj)
	if err != nil {
		return fmt.Errorf("compute patch: %w", err)
	}

	ns := obj.GetNamespace()
	name := obj.GetName()

	if ns == "" {
		_, err = k.dynamic.Resource(mapping.Resource).Patch(ctx, name, patch.Type(), data, metav1.PatchOptions{})
	} else {
		_, err = k.dynamic.Resource(mapping.Resource).Namespace(ns).Patch(ctx, name, patch.Type(), data, metav1.PatchOptions{})
	}
	return err
}
