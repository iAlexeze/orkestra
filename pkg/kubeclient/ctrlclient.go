package kubeclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/utils"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
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
	if u, ok, reason := a.getFromStore(obj, key); ok {
		return runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, obj)
	} else {
		logger.Debug().
			Str("type", fmt.Sprintf("%T", obj)).
			Str("key", key.Namespace+"/"+key.Name).
			Str("reason", reason).
			Msg("ctrlclient.Get: cache miss — live API call")
	}
	return a.k.Get(ctx, key.Namespace, key.Name, obj)
}

// getFromStore attempts a cache-backed read. Returns (result, true, "") on hit;
// (nil, false, reason) on miss so the caller can log the reason without branching.
func (a *ctrlClientAdapter) getFromStore(obj runtime.Object, key sigs.ObjectKey) (*unstructured.Unstructured, bool, string) {
	fn := a.k.GetStoreFor()
	if fn == nil {
		return nil, false, "storeFor not wired"
	}
	gvks, _, err := a.k.Scheme().ObjectKinds(obj)
	if err != nil || len(gvks) == 0 {
		return nil, false, fmt.Sprintf("scheme cannot resolve GVK: %v", err)
	}
	gvk := gvks[0]
	store := fn(gvk)
	if store == nil {
		return nil, false, "no informer store for " + gvk.String()
	}
	storeKey := key.Name
	if key.Namespace != "" {
		storeKey = key.Namespace + "/" + key.Name
	}
	raw, exists, err := store.GetByKey(storeKey)
	if err != nil {
		return nil, false, "store.GetByKey error: " + err.Error()
	}
	if !exists || raw == nil {
		return nil, false, "key not in store"
	}
	if u, ok := raw.(*unstructured.Unstructured); ok {
		return u, true, ""
	}
	// Typed informers store the concrete Go type. Convert to unstructured so the
	// caller can FromUnstructured it into the target — same roundtrip, avoids a
	// direct type assertion that would only work for one specific type.
	rObj, ok := raw.(runtime.Object)
	if !ok {
		return nil, false, fmt.Sprintf("store item is %T, not runtime.Object", raw)
	}
	rawMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(rObj)
	if err != nil {
		return nil, false, "ToUnstructured: " + err.Error()
	}
	return &unstructured.Unstructured{Object: rawMap}, true, ""
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
	// List types have a "List" suffix in the kind; strip it to get the item GVK.
	gvk := gvks[0]

	if items, ok, reason := a.listFromStore(gvk, lo); ok {
		return a.assembleList(list, items)
	} else {
		logger.Debug().
			Str("gvk", gvk.String()).
			Str("namespace", lo.Namespace).
			Str("reason", reason).
			Msg("ctrlclient.List: cache miss — live API call")
	}

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

// listFromStore attempts a cache-backed list for the given list GVK. Returns
// (items, true) when the store is registered and all matching items are found.
// Falls back to false so the caller goes live. Label selector and namespace
// filtering match controller-runtime's delegating client behaviour.
//
// When a field selector is present, listFromStore tries a ByIndex lookup using
// the registered indexer for the GVK. If the field key matches a registered index
// name the query is served from the index. If no matching index is registered the
// call falls back to the live API (field-selector filtering in-process would return
// incorrect results for fields not covered by an index).
func (a *ctrlClientAdapter) listFromStore(listGVK schema.GroupVersionKind, lo *sigs.ListOptions) ([]*unstructured.Unstructured, bool, string) {
	fn := a.k.GetStoreFor()
	if fn == nil {
		return nil, false, "storeFor not wired"
	}
	elemKind := strings.TrimSuffix(listGVK.Kind, "List")
	if elemKind == listGVK.Kind {
		return nil, false, "not a list type"
	}
	elemGVK := schema.GroupVersionKind{Group: listGVK.Group, Version: listGVK.Version, Kind: elemKind}
	store := fn(elemGVK)
	if store == nil {
		return nil, false, "no informer store for " + elemGVK.String()
	}

	// Field selector — try index before full scan.
	if lo.FieldSelector != nil && !lo.FieldSelector.Empty() {
		reqs := lo.FieldSelector.Requirements()
		indexFn := a.k.GetIndexerFor()
		if indexFn == nil {
			return nil, false, "field selector present but indexerFor not wired"
		}
		indexer := indexFn(elemGVK)
		if indexer == nil {
			return nil, false, "field selector present but no indexer for " + elemGVK.String()
		}
		// Serve the first matchable requirement from the index; remaining
		// requirements are applied as post-filters.
		registeredIndexers := indexer.GetIndexers()
		for i, req := range reqs {
			if _, ok := registeredIndexers[req.Field]; !ok {
				continue
			}
			raws, err := indexer.ByIndex(req.Field, req.Value)
			if err != nil {
				return nil, false, "ByIndex error: " + err.Error()
			}
			remaining := append(reqs[:i:i], reqs[i+1:]...)
			var result []*unstructured.Unstructured
			for _, raw := range raws {
				u, ok := raw.(*unstructured.Unstructured)
				if !ok {
					return nil, false, fmt.Sprintf("index item is %T, not *Unstructured", raw)
				}
				if lo.Namespace != "" && u.GetNamespace() != lo.Namespace {
					continue
				}
				if lo.LabelSelector != nil && !lo.LabelSelector.Matches(labels.Set(u.GetLabels())) {
					continue
				}
				if !utils.MatchesFieldRequirements(u, remaining) {
					continue
				}
				result = append(result, u)
			}
			return result, true, ""
		}
		// No registered index covers any requirement — go live so the API applies the filter.
		return nil, false, "no registered index covers field selector " + lo.FieldSelector.String()
	}

	var sel labels.Selector
	if lo.LabelSelector != nil {
		sel = lo.LabelSelector
	} else {
		sel = labels.Everything()
	}

	var result []*unstructured.Unstructured
	for _, raw := range store.List() {
		u, ok := raw.(*unstructured.Unstructured)
		if !ok {
			return nil, false, fmt.Sprintf("store item is %T, not *Unstructured", raw)
		}
		if lo.Namespace != "" && u.GetNamespace() != lo.Namespace {
			continue
		}
		if !sel.Matches(labels.Set(u.GetLabels())) {
			continue
		}
		result = append(result, u)
	}
	return result, true, ""
}

// assembleList converts a slice of unstructured items into the typed list obj.
func (a *ctrlClientAdapter) assembleList(list sigs.ObjectList, items []*unstructured.Unstructured) error {
	rawItems := make([]interface{}, len(items))
	for i, u := range items {
		rawItems[i] = u.Object
	}
	listMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(list)
	if err != nil {
		return fmt.Errorf("ctrlclient List (cache): marshal list shell: %w", err)
	}
	listMap["items"] = rawItems
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
