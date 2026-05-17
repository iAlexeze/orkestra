# 03 — What simulate does not cover

The fake cluster starts empty and has no external connectivity. Some operator features require a real cluster or a compiled user binary.

## Current gaps

| Feature | Status | Alternative |
|---------|--------|-------------|
| Cross-namespace reads (secrets, configmaps) | ✗ fake cluster is empty | `ork e2e` |
| External HTTP calls (`external:`) | ✗ not executed | `ork e2e` |
| Git hooks | ✗ not executed | `ork e2e` |
| Provider blocks (AWS, MongoDB) | ✗ not executed | `ork e2e` |
| Admission webhook validation | ✗ not registered | `ork e2e` |
| Watch events / informer cache updates | ✗ only the `--cr` file is seeded | `ork e2e` |
| Real rollout timing (pod readiness) | ✗ Deployments are marked ready immediately | `ork e2e` |
| Go hooks (typed reconcile hooks) | ✗ nil hook binder | future work |
| Reconcile resource registry | ✗ nil katalog registry | future work |

For anything in the "use `ork e2e`" column, provision a real kind cluster and apply the CR against the actual operator runtime.

## Future work: Go hook wiring

`simulate` currently passes `nil` for the hook binder and the `ResourceKatalog` registry. This means typed Go hooks (`OnCreate`, `OnUpdate`, `OnDelete`) and constructor-based reconciliation are silently skipped.

**The path to wiring hooks:**

1. The user runs `ork generate registry` to produce the Go registry (already required for typed operators).
2. The user builds their operator binary (`make ork` or similar).
3. `simulate` calls `reconciler.NewGenericReconciler` with the user's compiled hook binder instead of `nil`.

The blocker is linking: Go hooks are compiled into the user's binary, not ork's. The options are:
- **Plugin mode**: build the user's hooks as a `plugin.so` and load it at runtime (complex, OS-specific).
- **Embedded simulate**: user imports `pkg/simulate` in their own test binary and calls `simulate.Run()` directly, passing their hooks.
- **Subprocess**: simulate shells out to the user's binary with a `--simulate` flag that the user wires.

For the advanced hooks example: if you build the operator binary with hooks compiled in, you can call `simulate.Run()` directly from a Go test in the operator repo, passing real hooks. This is the recommended path today.

**What the resource registry enables:**

Passing a real `*kordinator.ResourceKatalog` would allow simulate to track cross-CRD resource ownership and surface registry-level conflicts — useful for multi-CRD Katalogs.

→ Next: [04-internals.md](04-internals.md)
