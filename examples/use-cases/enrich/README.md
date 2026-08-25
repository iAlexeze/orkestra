# Enrich Examples

Three focused examples showing what `enrich` does, why it costs what it costs, and how conditional gates keep that cost near-zero in steady state.

| Example | What it teaches |
|---|---|
| [01 — Pod Health](01-pod-health/README.md) | `enrich: [pods]` — always-on pod count, readiness, crash detection in status |
| [02 — Warning Events](02-warning-events/README.md) | `enrich: [events]` with a conditional gate — zero API calls in steady state, warning details when degraded |
| [03 — Rollout Observer](03-rollout-observer/README.md) | `enrich: [replicasets]` with `or:` — replicaset data only during rollouts or debug mode |

All three share one CRD (`crd.yaml` at the root of this directory).

---

**Further reading:** [Enrich concept doc](https://orkestra.sh/docs/reference/schema/operatorbox/enrich)

---

## E2E

Run the full suite — all three enrich examples in one command:

```bash
ork e2e -f e2e.yaml
```

This runs [e2e.yaml](./e2e.yaml), which imports each sub-example's `e2e.yaml` and runs them sequentially in the same cluster. To run a single example:

```bash
cd 03-rollout-observer && ork e2e
```
