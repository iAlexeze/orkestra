# 05 — Patch Helpers and Context Injection

## Overview

The patch helpers (`PatchFinalizers`, `PatchLabels`, `PatchAnnotations`, `PatchStatus`, `PatchSpec`) are the generic reconciler's mutation surface. Like `Get`, `Create`, and `Patch` in [03-crud.md](03-crud.md), they derive the GVR internally from the object's Go type via the scheme and REST mapper — callers pass only the object and the data to patch.

## Context injection

```go
func WithKubeclient(ctx context.Context, kube KubeClient) context.Context
func FromContext(ctx context.Context) (KubeClient, bool)
```

`WithKubeclient` stores the `KubeClient` on the context before invoking hook and constructor functions. `FromContext` retrieves it. This is how hook functions receive the client without being wired up through the constructor.

```go
// In the reconciler before calling hooks:
ctx = kubeclient.WithKubeclient(ctx, kube)

// In a hook function:
kube, ok := kubeclient.FromContext(ctx)
if !ok {
    return fmt.Errorf("no kubeclient in context")
}
```

Constructor reconcilers receive `KubeClient` directly via the constructor function signature — they do not need `FromContext`.

## PatchFinalizers

```go
PatchFinalizers(ctx context.Context, obj runtime.Object, finalizers []string) error
```

Replaces the object's `metadata.finalizers` list with `finalizers` using a JSON Merge Patch. Only `metadata.finalizers` is sent in the patch body — the rest of the object is untouched. This avoids `resourceVersion` conflicts that would arise from patching the full object.

To add a finalizer:

```go
kube.PatchFinalizers(ctx, obj, append(obj.GetFinalizers(), "my.finalizer/cleanup"))
```

To remove all finalizers (unblock deletion):

```go
kube.PatchFinalizers(ctx, obj, nil)
```

## PatchLabels

```go
PatchLabels(ctx context.Context, obj runtime.Object, base, desired map[string]string) error
```

Transitions the object's labels from `base` to `desired` using a JSON Merge Patch. Keys in `base` that are absent in `desired` are set to `null` (the server deletes them). Keys in `desired` that differ from `base` are added or updated. Unchanged keys are omitted from the patch body.

`base` must be a snapshot of the labels **before** any in-memory mutations. Pass `nil` for brand-new objects.

```go
base := obj.GetLabels()    // snapshot before mutation
desired := maps.Clone(base)
desired["orkestra.io/managed-by"] = "orkestra"
kube.PatchLabels(ctx, obj, base, desired)
```

If `base` and `desired` are equal, the method returns nil without sending a request.

## PatchAnnotations

```go
PatchAnnotations(ctx context.Context, obj runtime.Object, annotations map[string]string) error
```

Merges `annotations` onto the object using a JSON Merge Patch. This is a **one-way merge**: keys in `annotations` are added or updated; keys absent from `annotations` are left unchanged. Keys are never deleted by this method.

This is intentional — Orkestra's annotation management (`managed-by`, `managed-since`) is write-once and never removes keys. Use `PatchLabels` with an explicit `base`/`desired` diff if key deletion is needed.

## PatchStatus

```go
PatchStatus(ctx context.Context, obj domain.Object, statusFields map[string]interface{}) error
```

Applies `statusFields` to the object's `/status` subresource using a JSON Merge Patch. The map is wrapped in `{"status": <statusFields>}` before sending. Fields not present in the map are left untouched.

```go
kube.PatchStatus(ctx, webapp, map[string]interface{}{
    "phase":    "Running",
    "endpoint": fmt.Sprintf("%s.%s.svc.cluster.local", webapp.Name, webapp.Namespace),
    "replicas": webapp.Spec.Replicas,
})
```

Requires the CRD to declare `subresources: status: {}`. Returns a 404 if the subresource is absent — treat this as non-fatal.

Uses merge patch (not strategic merge patch) because the API server has no built-in schema knowledge for CRD status fields. See the inline comment in `patch_status.go` for the full rationale.

## PatchSpec

```go
PatchSpec(ctx context.Context, obj domain.Object, specFields map[string]interface{}) error
```

Applies `specFields` to the object's spec using a JSON Merge Patch, wrapped as `{"spec": <specFields>}`. Same semantics as `PatchStatus` but for the spec subresource.

## GVR resolution

All patch helpers resolve the GVR the same way as `Get`, `Create`, and `Patch`:

1. `scheme.ObjectKinds(obj)` — maps the Go type to a GVK. For typed CRDs registered via `AddKnownTypeWithName`, this returns the override group declared in `apiTypes.group`, not the package's compiled-in `GroupVersion` constant.
2. `mapper.RESTMapping(gvk.GroupKind(), gvk.Version)` — the `DeferredDiscoveryRESTMapper` queries the API server's discovery endpoint to map the GVK to a GVR.

This means constructor reconcilers do not need to import or reference `GroupVersionResource` constants from their API type packages. The katalog's `apiTypes.group` and `apiTypes.plural` are the sole source of API identity.

---

**← Back to** [04 — Merge Patches](04-merge.md)
