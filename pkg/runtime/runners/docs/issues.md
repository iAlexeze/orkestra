# Known Issues — runners

## Surface cleanup does not cover hook-managed resources

`SweepOwnedNamespacedResources` and `SweepOwnedClusterScopedResources` find resources by the `orkestra-owner` label. This label is stamped automatically on resources created through the template engine (onCreate/onReconcile blocks). It is **not** stamped on resources created by Go hook code (`OnReconcile`, `OnDelete`), because the hook author writes plain Go — no automatic label injection happens.

As a result, when a CR switches from one per-target hook surface to another, the previous surface's hook-managed resources are invisible to the sweep and are not cleaned up.

**What the declaration already gives us:** `hooks.resources[].kind` is exactly the set of Kubernetes resource kinds the hook manages. This declaration is currently used for RBAC generation, webhook scope, and informer watches — but not for cleanup.

**The extension:** when `cleanupPreviousSurface` detects a target switch and the previous target's operatorBox declares `hooks.resources[]`, the sweep should include those specific kinds scoped to the previous target's ownerKey. If the hook did not label its resources, the fallback is name-based deletion (resources named after the CR in the previous target's namespace and kinds).

**Where to drive it:** `cleanupPreviousSurface` in `pkg/runtime/reconciler/run_surface_cleanup.go` has access to `r.crd`, which carries the previous target's operatorBox after `EffectiveOperatorBox(prevTarget)` is called. The hook's resource declarations are at `box.Reconciler.Hooks.Resources`.
