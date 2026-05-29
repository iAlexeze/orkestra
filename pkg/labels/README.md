# pkg/labels

`labels` owns two things:

1. **All label, annotation, and finalizer constants** used by the Orkestra control plane — stable string identifiers shared across the runtime, CLI, generators, komposers, Gateway, and admission webhooks.

2. **The `Manager`** — a stateless helper that applies the standard label invariants to any Kubernetes object based on Katalog configuration. Previously part of the reconciler, it lives here so the admission layer and any future consumer can use it without importing reconciler internals.

The package is dependency-free except for `domain.Object` and `k8s.io/apimachinery`. Safe to import from any layer.

## What lives here

| File | Role |
|------|------|
| `labels.go` | All label keys, annotation keys, finalizer keys, selector helpers, and label set constructors |
| `manager.go` | `Manager` — applies managed, deletion-protection, and strict-mode-exempt labels to a domain.Object in memory |

## Key constants

| Constant | Value | Purpose |
|----------|-------|---------|
| `DeletionProtectionLabel` | `orkestra.io/deletion-protection` | Marks resources the deletion-protection webhook blocks from being deleted |
| `StrictModeExemptKey` | `orkestra.io/strict-mode-exempt` | Signals that a CRD has opted out of strict-mode; the strict-mode webhook allows label removal when present |
| `ManagedKey` | `orkestra.orkspace.io/managed` | Identifies resources actively managed by Orkestra |
| `OrkestraOwner` | `orkestra-owner` | Identifies the owning CR name |
| `AnnotationManagedBy` | `orkestra.orkspace.io/managed-by` | Records which operator instance manages the resource |
| `AnnotationManagedSince` | `orkestra.orkspace.io/managed-since` | Records when Orkestra first took ownership (write-once) |
| `FinalizerOrkestra` | `orkestra.orkspace.io/finalizer` | Finalizer applied to CRs for cleanup hooks |

## Quick usage

```go
// Construct a Manager once per reconcile cycle
mgr := labels.NewManager(labels.Config{
    DeletionProtectionEnabled: kat.IsDeletionProtectionEnabled(),
})

// Apply all label invariants in memory
mgr.EnsureManagedLabel(obj)
mgr.EnsureDeletionProtectionLabel(obj, shouldProtect)
mgr.EnsureStrictModeExemptLabel(obj, effectiveStrict)

// Caller persists to the API server
kube.PatchLabels(ctx, obj, gvr, serverLabels, obj.GetLabels())
```

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand every label and annotation constant and who uses them | [01 — Label reference](docs/01-label-reference.md) |
| Understand how the Manager works and the two-phase protection removal | [02 — Manager](docs/02-manager.md) |
