# ork simulate

`ork simulate` runs the operator reconcile loop against a fake in-memory cluster. No Kubernetes cluster, no `kubectl`, no network.

---

## Why it exists

Traditionally, there are two options for testing: write unit tests that mock the Kubernetes client (which do not reflect real merge-patch semantics), or spin up a kind cluster and apply real CRs (which is slow and requires environment setup).

`ork simulate` takes a third path: it runs the same `GenericReconciler` that runs in production, but wires it to a fake in-memory Kubernetes store. This means:

- The template engine executes exactly as it would in production
- `when:` conditions are evaluated
- Status propagation runs
- `onCreate` and `onReconcile` blocks both execute, in order
- Steady state is detected when two consecutive cycles produce identical operations

The output tells you exactly what the operator would create, update, or delete — before you write a single CRD manifest or touch a cluster.

---

## The key property: same reconciler, no cluster

The distinction that makes `ork simulate` genuinely useful (not just a dry-run) is that it does not approximate what the reconciler does. It *is* the reconciler. Every code path that runs when a real CR lands in a real cluster runs identically in simulation.

This means simulation catches:

- Template expression errors (bad field references, type mismatches)
- Missing `when:` guards that cause unintended resource creation on first reconcile
- Status field propagation that references children not yet created
- `onReconcile` blocks that produce different output than `onCreate` when they should be identical

It does not catch:

- Admission webhook behavior (the fake cluster does not run webhooks)
- Actual Kubernetes API server behavior (merge-patch edge cases, field ownership)
- External HTTP responses — `external:` blocks produce empty responses in simulation

Use `ork simulate` to verify your templates are correct. Use `ork e2e` to verify your operator behaves correctly end-to-end against a real cluster.

---

## Running it

```bash
# Standard layout — katalog.yaml + cr.yaml in current directory
ork simulate

# Non-standard filenames
ork simulate -f my-operator.yaml --cr my-cr.yaml

# Simulate one CRD from a multi-CRD Katalog
ork simulate --crd website

# Run exactly 5 cycles (default is 10)
ork simulate --cycles 5
```

## Reading the output

```
Simulating website/my-site

  Cycle 1:
    + deployments/my-site
    + services/my-site
    ~ status/my-site

  Cycle 2:
    ~ status/my-site

  (cycles 3–10: identical)

  ✓ Steady state at cycle 3 in 189ms
```

`+` — resource created. `~` — resource updated. `-` — resource deleted.

`status/...` appearing on every cycle is expected — status fields are re-evaluated on every reconcile. Everything else should reach steady state within 2–3 cycles. If a resource keeps appearing as `+` after cycle 1, the reconciler is re-creating it instead of finding it stable — this usually means a template expression produces a different value on each evaluation (e.g., `now()` in a timestamp field).

Consecutive identical cycles are collapsed in the output. Steady state is noted but does not stop the run when `--cycles` is set explicitly.

---

## Workflow: write → simulate → cluster

The recommended development loop for a new operator:

1. Write the Katalog and a minimal CR
2. Run `ork simulate` — verify the right resources appear in cycle 1
3. Check that cycle 2 shows only `status/...` (no unexpected re-creation)
4. Adjust `when:` conditions, field references, or template expressions as needed
5. When simulation is clean, run against a real cluster with `ork run`

For anything more complex — state machines, dependencies between CRDs, admission webhooks — write an `ork e2e` spec that tests the full lifecycle. Simulation covers the template logic; E2E covers the system behavior.

---

## Relation to ork e2e

| | `ork simulate` | `ork e2e` |
|---|---|---|
| Requires cluster | No | Yes (kind or existing) |
| Runs real reconciler | Yes | Yes |
| Tests webhooks | No | Yes |
| Tests external calls | No (empty responses) | Yes |
| Tests health transitions | No | Yes |
| Speed | Milliseconds | Minutes |
| Best for | Template correctness | System correctness |

Use both. Simulate is the fast inner loop. E2E is the outer verification gate that runs before `ork registry push`.

---

→ See also: [`ork simulate` CLI reference](../../reference/cli/05-simulate.md), [`ork e2e`](../../reference/cli/08-e2e.md)
