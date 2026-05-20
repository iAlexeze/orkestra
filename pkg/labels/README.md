# pkg/labels

`labels` defines all label, annotation, and finalizer constants used by the Orkestra control plane. It is intentionally dependency-free and safe to import from any layer of the system — runtime, CLI, generators, komposers, motifs.

Nothing in this package performs logic. It only provides stable string identifiers and a few helpers for constructing label sets and selectors.

## Key constants

| Constant | Value | Purpose |
|----------|-------|---------|
| `DeletionProtectionLabel` | `orkestra.io/deletion-protection` | Marks resources the deletion-protection webhook protects |
| `Managed` | `orkestra.orkspace.io/managed` | Resources actively managed by Orkestra |
| `OrkestraOwner` | `orkestra-owner` | Label value identifying the owning CR name |
| `AnnotationManagedBy` | `orkestra.orkspace.io/managed-by` | Which operator instance manages the CR |
| `AnnotationManagedSince` | `orkestra.orkspace.io/managed-since` | When Orkestra first took ownership |
| `FinalizerOrkestra` | `orkestra.orkspace.io/finalizer` | Finalizer applied to CRs for cleanup hooks |

## Label set helpers

```go
// Standard labels applied to every Orkestra control-plane resource.
// Includes deletion-protection so the webhook protects Orkestra's own resources.
labels.OrkestraBaseLabels() map[string]string

// Add deletion-protection to an existing label map (non-destructive copy).
labels.WithDeletionProtection(m) map[string]string
```

## Selector helpers

```go
// LabelSelector matching orkestra.io/deletion-protection=true
// Used when building webhook NamespaceSelector or ObjectSelector.
labels.DeletionProtectionSelector() *metav1.LabelSelector

// LabelSelector matching all Orkestra control-plane resources.
labels.OrkestraResourceSelector() *metav1.LabelSelector
```
