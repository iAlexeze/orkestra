# 05 — Patch Helpers and Context Injection

## Overview

The patch helpers (`PatchFinalizers`, `PatchLabels`, `PatchAnnotations`, `PatchStatus`, `PatchSpec`) are the generic reconciler's mutation surface. Unlike the CRUD methods in [03-crud.md](03-crud.md), they require the caller to supply the GVR explicitly. This is intentional: the generic reconciler always has the GVR available from the CRD config, and passing it explicitly avoids a scheme+mapper lookup on every reconcile call.

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
PatchFinalizers(ctx context.Context, obj runtime.Object, gvr schema.GroupVersionResource, finalizers []string) error
```

Replaces the object's `metadata.finalizers` list with `finalizers` using a JSON Merge Patch. Only `metadata.finalizers` is sent in the patch body — the rest of the object is untouched. This avoids `resourceVersion` conflicts that would arise from patching the full object.

To add a finalizer:

```go
kube.PatchFinalizers(ctx, obj, gvr, append(obj.GetFinalizers(), "my.finalizer/cleanup"))
```

To remove all finalizers (unblock deletion):

```go
kube.PatchFinalizers(ctx, obj, gvr, nil)
```

## PatchLabels

```go
PatchLabels(ctx context.Context, obj runtime.Object, gvr schema.GroupVersionResource, base, desired map[string]string) error
```

Transitions the object's labels from `base` to `desired` using a JSON Merge Patch. Keys in `base` that are absent in `desired` are set to `null` (the server deletes them). Keys in `desired` that differ from `base` are added or updated. Unchanged keys are omitted from the patch body.

`base` must be a snapshot of the labels **before** any in-memory mutations. Pass `nil` for brand-new objects.

```go
base := obj.GetLabels()    // snapshot before mutation
desired := maps.Clone(base)
desired["orkestra.io/managed-by"] = "orkestra"
kube.PatchLabels(ctx, obj, gvr, base, desired)
```

If `base` and `desired` are equal, the method returns nil without sending a request.

## PatchAnnotations

```go
PatchAnnotations(ctx context.Context, obj runtime.Object, gvr schema.GroupVersionResource, annotations map[string]string) error
```

Merges `annotations` onto the object using a JSON Merge Patch. This is a **one-way merge**: keys in `annotations` are added or updated; keys absent from `annotations` are left unchanged. Keys are never deleted by this method.

This is intentional — Orkestra's annotation management (`managed-by`, `managed-since`) is write-once and never removes keys. Use `PatchLabels` with an explicit `base`/`desired` diff if key deletion is needed.

## PatchStatus

```go
PatchStatus(ctx context.Context, obj domain.Object, gvr schema.GroupVersionResource, statusFields map[string]interface{}) error
```

Applies `statusFields` to the object's `/status` subresource using a JSON Merge Patch. The map is wrapped in `{"status": <statusFields>}` before sending. Fields not present in the map are left untouched.

```go
kube.PatchStatus(ctx, webapp, apiv1.GroupVersionResource, map[string]interface{}{
    "phase":    "Running",
    "endpoint": fmt.Sprintf("%s.%s.svc.cluster.local", webapp.Name, webapp.Namespace),
    "replicas": webapp.Spec.Replicas,
})
```

Requires the CRD to declare `subresources: status: {}`. Returns a 404 if the subresource is absent — treat this as non-fatal.

Uses merge patch (not strategic merge patch) because the API server has no built-in schema knowledge for CRD status fields. See the inline comment in `patch_status.go` for the full rationale.

## PatchSpec

```go
PatchSpec(ctx context.Context, obj domain.Object, gvr schema.GroupVersionResource, specFields map[string]interface{}) error
```

Applies `specFields` to the object's spec using a JSON Merge Patch, wrapped as `{"spec": <specFields>}`. Same semantics as `PatchStatus` but for the spec subresource.

## Why explicit GVR here, but not in Get/Create/Patch?

The generic reconciler (`run_template_reconcile.go`, `run_customresource.go`) calls these helpers in the hot path. The GVR is already known from the CRD config registered at startup — deriving it from the scheme+mapper on every call would add latency for no benefit.

Constructor reconcilers use `Get`/`Create`/`Patch` (see [03-crud.md](03-crud.md)), which derive the GVR automatically. They are called less frequently and the added convenience outweighs the lookup cost.

---

**← Back to** [04 — Merge Patches](04-merge.md)
