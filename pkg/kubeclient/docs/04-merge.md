# 04 — Merge Patches

## Purpose

`MergeFrom` and `StrategicMergeFrom` are thin wrappers around `sigs.k8s.io/controller-runtime/pkg/client.MergeFrom` and `sigs.StrategicMergeFrom`, re-exported from this package so constructor reconcilers have a single import for all Orkestra client concerns.

`Patch` is a **type alias** for `sigs.Patch` — not a new type, not a wrapper. This means:

- `sigs.MergeFrom(existing.DeepCopy())` satisfies `kubeclient.Patch` directly.
- `sigs.StrategicMergeFrom(existing.DeepCopy())` satisfies `kubeclient.Patch` directly.
- `sigs.Apply(...)` (server-side apply) satisfies `kubeclient.Patch` directly.
- Any future patch type controller-runtime adds works in Orkestra without changes.

A reconciler migrating from controller-runtime does not change its patch construction lines at all. Only the method receiver changes: `r.client` → `r.kube`.

## The Patch type

```go
type Patch = sigs.Patch   // alias, not a new type
```

`sigs.Patch` is:

```go
type Patch interface {
    Type() types.PatchType
    Data(obj client.Object) ([]byte, error)
}
```

`Type` returns the patch media type. `Data` computes the patch body by diffing the base snapshot (captured at construction) against the mutated object. The result is sent directly to the dynamic client.

## MergeFrom — JSON Merge Patch (RFC 7396)

```go
func MergeFrom(base client.Object) Patch
```

Delegates to `sigs.MergeFrom`. Captures a JSON snapshot of `base` at call time. When `Data(modified)` is called, it computes a minimal JSON Merge Patch.

**Pattern:**

```go
patch := kubeclient.MergeFrom(existing.DeepCopy())
existing.Spec.Ports = desired.Spec.Ports
return kube.Patch(ctx, existing, patch)
```

`DeepCopy()` is critical — without it, mutating `existing` would also mutate `base`, producing an empty patch.

**When to use:** CRDs and any object where you control the entire field being patched. Merge patch replaces the patched key wholesale — it does not merge lists by a strategic key.

## StrategicMergeFrom — Strategic Merge Patch

```go
func StrategicMergeFrom(base client.Object) Patch
```

Delegates to `sigs.StrategicMergeFrom`. When `Data(modified)` is called, it computes a Strategic Merge Patch using the object's `patchMergeKey` struct tag annotations.

**Pattern:**

```go
patch := kubeclient.StrategicMergeFrom(existing.DeepCopy())
existing.Spec = desired.Spec
return kube.Patch(ctx, existing, patch)
```

**When to use:** Core Kubernetes types (`Deployment`, `DaemonSet`, `StatefulSet`, etc.) where the API server understands strategic merge patch annotations. In particular, list fields with `patchMergeKey` tags (e.g. containers merged by `name`, not replaced). For CRDs, use `MergeFrom` instead — the API server has no schema knowledge for custom types.

## Using sigs directly

Because `Patch` is a type alias, controller-runtime patch values work without any adapter:

```go
// These are equivalent — both satisfy kubeclient.Patch:
patch := kubeclient.MergeFrom(existing.DeepCopy())
patch := sigs.MergeFrom(existing.DeepCopy())

// Server-side apply also works:
patch := sigs.Apply
```

This is intentional. Orkestra does not rebuild what controller-runtime already provides. The constructor path is designed so that existing reconcile logic, including patch patterns, can be lifted in as-is.

## Choosing between the two

| | `MergeFrom` | `StrategicMergeFrom` |
|--|-------------|----------------------|
| Patch type | `MergePatchType` | `StrategicMergePatchType` |
| List merge | Replace | By merge key (`name`, etc.) |
| Works on CRDs | Yes | No — falls back to replace |
| Works on core types | Yes | Yes (preferred) |
| Underlying impl | `sigs.MergeFrom` | `sigs.StrategicMergeFrom` |

---

**← Back to** [03 — CRUD](03-crud.md) | **Next →** [05 — Patch Helpers and Context Injection](05-patch-helpers.md)
