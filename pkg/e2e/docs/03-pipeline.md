# Run Pipeline

`runner.Run` executes these steps in order. Each step must succeed before the next begins.

```
 1. Capture original kubectl context  (restored via defer on any exit)
 2. Ensure cluster                    (create kind cluster, or use --cluster / --use-current)
 3. Check dependencies                (kubectl, helm, kind available)
 4. Apply CRD                         (from spec.crd or katalog crdFile entries)
 5. Pre-pull OCI imports              (skipped when customOperator: true)
 6. Generate + apply bundle           (skipped when customOperator: true)
 7. Apply setup files                 (spec.setup.apply, in order)
 8. Install setup Helm charts         (spec.setup.helm, in order)
 9. Wait for setup resources          (spec.setup.wait, in order)
10. Install or sync Orkestra          (skipped when customOperator: true — see below)
11. Run expectations                  (cr-applied / cr-deleted, in order)
12. Print results
13. Run imports                       (if any — see imports.md)
14. Delete cluster                    (unless --keep-cluster or --cluster was used)
15. Restore kubectl context           (defer — always runs)
```

## Orkestra install / sync (step 10)

**Fresh install** (Orkestra not present):
```
→ Installing Orkestra...
  ✓ Orkestra installed
→ Waiting for Orkestra to be ready...
  ✓ Orkestra runtime ready
```

**Already installed** (bundle update only — silent):
The runtime is restarted silently to pick up the new bundle. No messages are printed. The health check still runs to ensure the runtime is ready before the CR is applied.

**customOperator: true** — step 10 is skipped entirely. Your operator is responsible for reconciling the CR.

## Teardown

For non-owned clusters (--use-current, --cluster, --keep-cluster), teardown reverses every applied resource in order:

```
CR delete → Orkestra uninstall → bundle delete → setup (reverse) → CRDs
```

When running as a shared import (suite mode), the bundle is NOT deleted from the cluster — the next import will update it in place with its own bundle. The coordinator uninstalls Orkestra once after all imports complete.

## Bundle generation

The runner resolves `crdFile` references in the Katalog before generating the bundle. This embeds the CRD type information directly into the ConfigMap so the in-cluster Orkestra runtime doesn't need access to local files.

## Context restore

The original `kubectl` context is captured at the start of `Run` and restored via `defer` — so it's always restored regardless of whether the test passes, fails, or panics.
