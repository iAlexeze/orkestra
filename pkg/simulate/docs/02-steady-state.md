# 02 — Steady state and cycles

Steady state means two consecutive cycles produced an identical sequence of ops — the operator has converged and no further mutations will occur.

```
✓ Steady state at cycle 3 in 173ms
```

If the simulation runs all requested cycles without converging:

```
~ Max cycles reached (10) in 3.2s
```

## Tuning cycles

`--cycles N` runs exactly N reconcile cycles regardless of steady state. Use it to see how your operator behaves over time:

```sh
ork simulate -f katalog.yaml --cr cr.yaml --cycles 20
```

A persistent loop of creates or updates in later cycles usually means a drift condition in `onReconcile` — the operator is always writing even when nothing changed.

## Common patterns

**Normal — creates then converges**

```
Cycle 1:
    + deployments/my-site
    + services/my-site-svc
    ~ status/my-site
Cycle 2:
    ~ status/my-site
(cycles 3–10: identical)

✓ Steady state at cycle 3
```

Cycle 1 creates both resources. Cycle 2 confirms they exist and only writes status.

**Cross-namespace source not found**

```
Cycle 1:
    ~ status/db-creds
    ✗ secrets[0].sync: reading source secret "database-credentials" from "platform": not found
(cycles 2–10: identical)

✓ Steady state at cycle 2
```

The fake cluster starts empty. Cross-namespace reads always return not-found. Use `ork e2e` for those flows.

**No ops in any cycle**

No cycle lines at all means `onCreate` was not reached, or all template conditions evaluated to false. Check that the CR has all required `spec` fields.

→ Next: [03-limitations.md](03-limitations.md)
