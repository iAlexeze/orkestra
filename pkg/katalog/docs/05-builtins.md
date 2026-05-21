# 05 — Built-in Type Handling

The built-in kind registry lives in [`pkg/children`](../../children/README.md) — not in `pkg/katalog`.

`pkg/katalog` imports `pkg/children` to enrich CRD entries and generate RBAC rules:

- `children.LookupBuiltIn` — resolves group/version/plural for kind-only declarations
- `children.AllBuiltInKinds` — populates error messages with known kind names
- `children.BuiltInMeta` — reads per-kind flags at validation time to stamp `IgnoreStatusPatch` and `IgnoreObservedGeneration` onto each `CRDEntry`
- `children.GVRForBuiltIn` — resolves GVRs for RBAC rule generation
- `children.AllBuiltInKindDefs` — iterates all entries to emit RBAC rules per resource type

In the hot reconcile path, `r.crd.SkipStatusSubresource()` and `r.crd.SkipObservedGeneration()` read pre-computed flags stamped at boot — no registry lookup at reconcile time.

For the full reference on the built-in registry — `BuiltInKind` struct, accessor functions, GVR variables, and how to add a new built-in kind — see [pkg/children/docs/05-builtins.md](../../children/docs/05-builtins.md).

→ Back to: [README.md](../README.md)
