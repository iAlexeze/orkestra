# 01 — Reading the output

```text
Simulating website/hello-website

  Cycle 1:
    + deployments/hello-website-deployment
    ~ status/hello-website
  Cycle 2:
    ~ status/hello-website
  (cycles 3–10: identical)

  ✓ Steady state at cycle 3 in 173ms
```

**Header** — `Simulating <crd>/<cr-name>`: which CRD and CR instance is being simulated.

**Cycle N** — one pass of the reconcile loop. Cycles with no visible ops are skipped. Consecutive identical cycles are collapsed to `(cycles X–Y: identical)`.

**Op icons:**

| Icon | Verb | Meaning |
|------|------|---------|
| `+` (green) | `create` | Resource was created for the first time |
| `~` (yellow) | `update` / `patch` | Existing resource was updated |
| `-` (red) | `delete` | Resource was deleted |
| `✗` (red) | error | Reconcile returned an error this cycle |

**`status/...`** appears every cycle — the reconciler always writes a status update. This is expected and not an error.

If a resource is both created and patched in the same cycle (happens with `reconcile: true`), only `+` is shown — the create wins.

→ Next: [02-steady-state.md](02-steady-state.md)
