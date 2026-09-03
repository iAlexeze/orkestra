# 03 — CRUD: Get, Create, Patch

## Purpose

`Get`, `Create`, and `Patch` are typed object operations added to the `KubeClient` interface for constructor reconcilers. They mirror the `sigs.k8s.io/controller-runtime/pkg/client.Client` API surface, so a controller-runtime reconciler can be lifted into Orkestra with minimal changes.

The key difference from the generic reconciler's patch helpers (`PatchFinalizers`, `PatchLabels`, `PatchStatus` etc.) is that **the caller does not supply a GVR**. The GVR is derived from the Go type at call time using the scheme and REST mapper already stored on `Kubeclient`.

## Interface

```go
Get(ctx context.Context, namespace, name string, into client.Object) error
Create(ctx context.Context, obj client.Object) error
Patch(ctx context.Context, obj client.Object, patch Patch) error
```

All three accept `sigs.k8s.io/controller-runtime/pkg/client.Object`. Every typed Kubernetes struct (`*appsv1.Deployment`, `*corev1.Service`, custom CR types) implements this interface. The scheme must have the object's type registered (guaranteed for any type that appears in a katalog's constructor).

## GVR derivation (`gvrFor`)

```go
func (k *Kubeclient) gvrFor(obj runtime.Object) (*meta.RESTMapping, error) {
    gvks, _, err := k.scheme.ObjectKinds(obj)
    gvk := gvks[0]
    mapping, err := k.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
    return mapping, err
}
```

`scheme.ObjectKinds` returns all GVKs registered for the Go type. The first match is used. `mapper.RESTMapping` translates the GVK to a `RESTMapping` which includes the `GroupVersionResource` used for all dynamic client calls.

The mapper is a `restmapper.DeferredDiscoveryRESTMapper` (see `kubeclient.go`). It queries discovery lazily and caches the result. Call `RefreshMapper()` after creating CRDs to invalidate the cache.

## Get

```go
func (k *Kubeclient) Get(ctx context.Context, namespace, name string, into client.Object) error
```

1. Derives the REST mapping via `gvrFor(into)`.
2. Calls `k.dynamic.Resource(mapping.Resource).Namespace(ns).Get(ctx, name, ...)`.
   Cluster-scoped resources omit the `.Namespace()` call.
3. Converts the returned `*unstructured.Unstructured` to the typed object using
   `runtime.DefaultUnstructuredConverter.FromUnstructured`.

The returned error from the dynamic client is a `*k8s.io/apimachinery/pkg/api/errors.StatusError` — `errors.IsNotFound(err)` works as expected.

## Create

```go
func (k *Kubeclient) Create(ctx context.Context, obj client.Object) error
```

1. Derives the REST mapping via `gvrFor(obj)`.
2. Converts `obj` to `*unstructured.Unstructured` using
   `runtime.DefaultUnstructuredConverter.ToUnstructured`.
3. Calls `k.dynamic.Resource(mapping.Resource).Namespace(ns).Create(ctx, u, ...)`.

The namespace is read from `meta.Accessor(obj).GetNamespace()`. If the object has no namespace set (cluster-scoped), the `.Namespace()` call is omitted.

## Patch

```go
func (k *Kubeclient) Patch(ctx context.Context, obj client.Object, patch Patch) error
```

1. Derives the REST mapping via `gvrFor(obj)`.
2. Calls `patch.Data(obj)` to compute the patch body.
3. Calls `k.dynamic.Resource(mapping.Resource).Namespace(ns).Patch(ctx, name, patch.Type(), data, ...)`.

`Patch` is a type alias for `sigs.k8s.io/controller-runtime/pkg/client.Patch` — see [04-merge.md](04-merge.md). Values from `sigs.MergeFrom`, `sigs.StrategicMergeFrom`, and `sigs.Apply` all satisfy it directly.

## Usage in a constructor reconciler

```go
// Get → IsNotFound → Create, else patch.
// The only change from controller-runtime is the method receiver:
// r.client → r.kube. The patch lines are identical.

existing := &appsv1.Deployment{}
err := r.kube.Get(ctx, webapp.Namespace, webapp.Name, existing)
if errors.IsNotFound(err) {
    return r.kube.Create(ctx, desired)
}
if err != nil {
    return err
}
// kubeclient.StrategicMergeFrom delegates to sigs.StrategicMergeFrom.
// sigs.StrategicMergeFrom(existing.DeepCopy()) works here without any adapter.
patch := kubeclient.StrategicMergeFrom(existing.DeepCopy())
existing.Spec = desired.Spec
return r.kube.Patch(ctx, existing, patch)
```

The patch variables (`kubeclient.MergeFrom`, `sigs.MergeFrom`) are interchangeable — `Patch` is a type alias for `sigs.Patch`, not a new type. A controller-runtime reconciler can be lifted into Orkestra by changing the method receiver alone; existing patch construction lines do not need to move.

## Simulation

`FakeKubeclient` (in `pkg/registry/simulate`) stubs all three methods:

- `Get` records a `get` op and returns a NotFound error, so the reconciler always takes the `Create` path on every simulated cycle.
- `Create` and `Patch` record their ops and return nil.

This produces clean simulation output showing create operations without requiring real cluster state between cycles.

---

**← Back to** [02 — DynamicListerWatcher](02-dynamic.md) | **Next →** [04 — Merge Patches](04-merge.md)
