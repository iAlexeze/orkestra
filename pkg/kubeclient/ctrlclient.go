package kubeclient

import (
	"context"
	"fmt"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// ToClient wraps a kubeclient.Interface as a sigs.k8s.io/controller-runtime/pkg/client.Client.
// Constructor reconcilers migrated from controller-runtime can call ToClient(kube) and use
// the familiar client.Get / client.Patch / client.Status().Patch patterns without learning
// kubeclient's API. All operations delegate to the underlying Interface — the same dynamic
// client, scheme, and mapper that the declarative reconciler uses.
//
// Unsupported operations (DeleteAllOf, SubResource other than "status") return
// ErrNotSupported so the compile check passes and callers get a clear runtime signal.
func ToClient(k Interface) sigs.Client {
	return &ctrlClientAdapter{k: k}
}

type ctrlClientAdapter struct {
	k Interface
}

var _ sigs.Client = (*ctrlClientAdapter)(nil)

// ── Reader ────────────────────────────────────────────────────────────────────

func (a *ctrlClientAdapter) Get(ctx context.Context, key sigs.ObjectKey, obj sigs.Object, _ ...sigs.GetOption) error {
	return a.k.Get(ctx, key.Namespace, key.Name, obj)
}

func (a *ctrlClientAdapter) List(ctx context.Context, list sigs.ObjectList, opts ...sigs.ListOption) error {
	lo := &sigs.ListOptions{}
	for _, opt := range opts {
		opt.ApplyToList(lo)
	}

	gvks, _, err := a.k.Scheme().ObjectKinds(list)
	if err != nil {
		return fmt.Errorf("ctrlclient List: unknown type %T: %w", list, err)
	}
	// List types have a "List" suffix in the kind; strip it to get the item GVK
	gvk := gvks[0]
	mapping, err := a.k.Mapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("ctrlclient List: no REST mapping for %s: %w", gvk, err)
	}

	listOpts := metav1.ListOptions{}
	if lo.LabelSelector != nil {
		listOpts.LabelSelector = lo.LabelSelector.String()
	}
	if lo.FieldSelector != nil {
		listOpts.FieldSelector = lo.FieldSelector.String()
	}

	var ul *unstructured.UnstructuredList
	if lo.Namespace != "" {
		ul, err = a.k.DynamicClient().Resource(mapping.Resource).Namespace(lo.Namespace).List(ctx, listOpts)
	} else {
		ul, err = a.k.DynamicClient().Resource(mapping.Resource).List(ctx, listOpts)
	}
	if err != nil {
		return err
	}

	// Convert each item individually then re-assemble into the typed list.
	// FromUnstructured on ul.Object alone would restore list metadata but leave
	// Items empty — the dynamic client stores items as separate Unstructured values.
	items := make([]interface{}, len(ul.Items))
	for i := range ul.Items {
		out := map[string]interface{}{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(ul.Items[i].Object, &out); err != nil {
			return fmt.Errorf("ctrlclient List: convert item %d: %w", i, err)
		}
		items[i] = out
	}

	listMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(list)
	if err != nil {
		return fmt.Errorf("ctrlclient List: marshal list shell: %w", err)
	}
	// Copy list metadata from the server response, then set items.
	for k, v := range ul.Object {
		if k != "items" {
			listMap[k] = v
		}
	}
	listMap["items"] = items

	return runtime.DefaultUnstructuredConverter.FromUnstructured(listMap, list)
}

// ── Writer ────────────────────────────────────────────────────────────────────

func (a *ctrlClientAdapter) Create(ctx context.Context, obj sigs.Object, _ ...sigs.CreateOption) error {
	return a.k.Create(ctx, obj)
}

func (a *ctrlClientAdapter) Delete(ctx context.Context, obj sigs.Object, opts ...sigs.DeleteOption) error {
	do := &sigs.DeleteOptions{}
	for _, opt := range opts {
		opt.ApplyToDelete(do)
	}

	mapping, err := a.restMappingFor(obj)
	if err != nil {
		return err
	}

	delOpts := metav1.DeleteOptions{}
	ns := obj.GetNamespace()
	name := obj.GetName()

	if ns == "" {
		err = a.k.DynamicClient().Resource(mapping.Resource).Delete(ctx, name, delOpts)
	} else {
		err = a.k.DynamicClient().Resource(mapping.Resource).Namespace(ns).Delete(ctx, name, delOpts)
	}
	return err
}

func (a *ctrlClientAdapter) Update(ctx context.Context, obj sigs.Object, _ ...sigs.UpdateOption) error {
	mapping, err := a.restMappingFor(obj)
	if err != nil {
		return err
	}

	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return fmt.Errorf("ctrlclient Update: convert to unstructured: %w", err)
	}
	u := &unstructured.Unstructured{Object: raw}
	ns := obj.GetNamespace()

	if ns == "" {
		_, err = a.k.DynamicClient().Resource(mapping.Resource).Update(ctx, u, metav1.UpdateOptions{})
	} else {
		_, err = a.k.DynamicClient().Resource(mapping.Resource).Namespace(ns).Update(ctx, u, metav1.UpdateOptions{})
	}
	return err
}

func (a *ctrlClientAdapter) Patch(ctx context.Context, obj sigs.Object, patch sigs.Patch, _ ...sigs.PatchOption) error {
	// sigs.Patch == kubeclient.Patch (type alias), passes straight through.
	return a.k.Patch(ctx, obj, patch)
}

func (a *ctrlClientAdapter) DeleteAllOf(_ context.Context, _ sigs.Object, _ ...sigs.DeleteAllOfOption) error {
	return fmt.Errorf("ctrlclient: DeleteAllOf is not supported by the kubeclient adapter")
}

func (a *ctrlClientAdapter) Apply(_ context.Context, _ runtime.ApplyConfiguration, _ ...sigs.ApplyOption) error {
	return fmt.Errorf("ctrlclient: Apply (ApplyConfiguration) is not supported; use Patch with a SSA patch instead")
}

// ── StatusClient ──────────────────────────────────────────────────────────────

func (a *ctrlClientAdapter) Status() sigs.SubResourceWriter {
	return &ctrlStatusAdapter{k: a.k}
}

// ── SubResourceClientConstructor ─────────────────────────────────────────────

// SubResource returns a client for the named subresource. Only "status" is
// supported — Patch on the returned client routes to the /status subresource
// via the dynamic client. All other methods and all other subresource names
// return an error.
func (a *ctrlClientAdapter) SubResource(_ string) sigs.SubResourceClient {
	return &ctrlStatusAdapter{k: a.k}
}

// ── Scheme / Mapper / GVK ─────────────────────────────────────────────────────

func (a *ctrlClientAdapter) Scheme() *runtime.Scheme { return a.k.Scheme() }

func (a *ctrlClientAdapter) RESTMapper() apimeta.RESTMapper { return a.k.Mapper() }

func (a *ctrlClientAdapter) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	gvks, _, err := a.k.Scheme().ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return gvks[0], nil
}

func (a *ctrlClientAdapter) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	gvks, _, err := a.k.Scheme().ObjectKinds(obj)
	if err != nil {
		return false, err
	}
	mapping, err := a.k.Mapper().RESTMapping(gvks[0].GroupKind(), gvks[0].Version)
	if err != nil {
		return false, err
	}
	return mapping.Scope.Name() == apimeta.RESTScopeNameNamespace, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (a *ctrlClientAdapter) restMappingFor(obj runtime.Object) (*apimeta.RESTMapping, error) {
	gvks, _, err := a.k.Scheme().ObjectKinds(obj)
	if err != nil {
		return nil, fmt.Errorf("ctrlclient: unknown type %T: %w", obj, err)
	}
	mapping, err := a.k.Mapper().RESTMapping(gvks[0].GroupKind(), gvks[0].Version)
	if err != nil {
		return nil, fmt.Errorf("ctrlclient: no REST mapping for %s: %w", gvks[0], err)
	}
	return mapping, nil
}

// ── status adapter ────────────────────────────────────────────────────────────

// ctrlStatusAdapter implements sigs.SubResourceWriter for the /status subresource.
// Patch delegates to the dynamic client so the caller can use sigs.MergeFrom directly.
type ctrlStatusAdapter struct {
	k Interface
}

var _ sigs.SubResourceClient = (*ctrlStatusAdapter)(nil)

func (s *ctrlStatusAdapter) Get(_ context.Context, _ sigs.Object, _ sigs.Object, _ ...sigs.SubResourceGetOption) error {
	return fmt.Errorf("ctrlclient SubResource.Get is not supported")
}

func (s *ctrlStatusAdapter) Patch(ctx context.Context, obj sigs.Object, patch sigs.Patch, _ ...sigs.SubResourcePatchOption) error {
	gvks, _, err := s.k.Scheme().ObjectKinds(obj)
	if err != nil {
		return fmt.Errorf("ctrlclient Status.Patch: unknown type %T: %w", obj, err)
	}
	mapping, err := s.k.Mapper().RESTMapping(gvks[0].GroupKind(), gvks[0].Version)
	if err != nil {
		return fmt.Errorf("ctrlclient Status.Patch: no REST mapping for %s: %w", gvks[0], err)
	}

	data, err := patch.Data(obj)
	if err != nil {
		return fmt.Errorf("ctrlclient Status.Patch: compute patch: %w", err)
	}

	ns := obj.GetNamespace()
	name := obj.GetName()

	if ns == "" {
		_, err = s.k.DynamicClient().Resource(mapping.Resource).
			Patch(ctx, name, patch.Type(), data, metav1.PatchOptions{}, "status")
	} else {
		_, err = s.k.DynamicClient().Resource(mapping.Resource).Namespace(ns).
			Patch(ctx, name, patch.Type(), data, metav1.PatchOptions{}, "status")
	}
	return err
}

func (s *ctrlStatusAdapter) Create(_ context.Context, _ sigs.Object, _ sigs.Object, _ ...sigs.SubResourceCreateOption) error {
	return fmt.Errorf("ctrlclient Status.Create is not supported")
}

func (s *ctrlStatusAdapter) Update(ctx context.Context, obj sigs.Object, _ ...sigs.SubResourceUpdateOption) error {
	return s.Patch(ctx, obj, sigs.MergeFrom(obj.DeepCopyObject().(sigs.Object)))
}

func (s *ctrlStatusAdapter) Apply(_ context.Context, _ runtime.ApplyConfiguration, _ ...sigs.SubResourceApplyOption) error {
	return fmt.Errorf("ctrlclient Status.Apply is not supported")
}
