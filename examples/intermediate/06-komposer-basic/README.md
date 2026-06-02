# 06 — Basic Komposer

Two Katalogs. One Komposer. One runtime. This is how platform teams compose
operator behavior from multiple sources and apply environment-specific overrides
without touching the base definitions.

**What you learn:** Komposer sources, file composition, spec.crds overrides,
built-in kind governance alongside custom CRDs.

**Builds on:** [05 — When Conditions](../05-when-conditions/README.md)

---

## What is new

**Komposer** — instead of running `ork run --file`, you run
`ork run --file komposer.yaml`. The Komposer reads both source Katalogs,
merges their CRD entries, applies your overrides, and produces a single unified
configuration that Orkestra runs.

**Two Katalogs:**
- `website-katalog.yaml` — the website operator (custom CRD)
- `namespace-katalog.yaml` — namespace governance (built-in Kubernetes `Namespace` kind)

Both run in the same Orkestra instance. The namespace governance CRD watches
every `Namespace` in the cluster and warns when the `team` label is missing.

**`spec.crds` override** — the Komposer overrides `workers` and `resync` for
the website CRD without modifying `website-katalog.yaml`. The base Katalog
stays clean. The environment-specific tuning lives in the Komposer.

---

## Steps

### 1. Install the CRD

```bash
kubectl apply -f crd.yaml
```

### 2. Preview the merged configuration

Before running, see what the Komposer produces:

```bash
ork template --file komposer.yaml --yaml

# Use --json to view template result in JSON format
ork template --file komposer.yaml --json
```

You will see both CRD entries — `website` with `workers: 4` and `resync: 15s`
(overridden), and `namespace-governance` with `workers: 1` (from source).

### 3. Validate

```bash
ork validate --file komposer.yaml
```

Expected:
```
✓ website
    kind: Website
    group: demo.orkestra.io / version: v1alpha1 / plural: websites
    mode: dynamic / workers: 4 / resync: 15s   ← overridden values

✓ namespace-governance
    kind: Namespace → enriched from built-in registry
    group: "" / version: v1 / plural: namespaces / scope: Cluster
    mode: dynamic / workers: 1
```

### 4. Simulate (optional, no cluster needed)

Before running, verify what the operator would reconcile for a specific CR:

```bash
ork simulate --cr cr.yaml --crd website
```

```text
Simulating website/composed-site

  Cycle 1:
    + deployments/composed-site-deployment
    + services/composed-site-svc
    ~ status/composed-site
  Cycle 2:
    ~ status/composed-site
  (cycles 3–10: identical)

  ✓ Steady state at cycle 3 in 193ms
```

**What this means:**
- `--crd website` scopes the simulation to the Website CRD only — the namespace-governance CRD from the other Katalog is not simulated here. Use `--crd` to isolate one CRD at a time when a Komposer contains many.
- Cycle 1 creates a Deployment and a Service — the values come from the merged Komposer configuration, not either source Katalog alone. Worker count, resync, and any inline overrides in `komposer.yaml` are already baked in.
- **Steady state at cycle 3** — the Website CRD converges in three cycles. This is what you want to see before connecting the Komposer to a real cluster.

### 5. Start the operator

```bash
ork run --file komposer.yaml
```

> [!TIP]
> If you are watching the logs, you will already see orkestra working on existing namespaces and adding missing labels without creating a CR. This is the **Ongoing Validation** promise by Orkestra.

> P

### 5. Apply the CR

```bash
kubectl apply -f cr.yaml
```

### 6. Verify

```bash
kubectl get websites
kubectl get deployments | grep composed-site
kubectl get services | grep composed-site
```

### 7. See namespace governance in action

Create a namespace without a team label:

```bash
kubectl create namespace test-no-label
```

Check the `/katalog` endpoint for active warnings:

```bash
curl localhost:8080/katalog/namespace-governance | jq .
```

You will see the warn violation recorded for `test-no-label`.

Create a namespace with the required label and the warning clears:

```bash
kubectl label namespace test-no-label team=platform
```

### 8. See the composition

```bash
curl localhost:8080/katalog | jq '.crds[] | {name: .name, workers: .workers, resync: .resync}'
```

Both CRDs are running in the same Orkestra instance:
```json
{
    "name": "namespace-governance", 
    "workers": 1, 
    "resync": "15s"
}
{
    "name": "website", 
    "workers": 4, 
    "resync": "15s"
}
```

---

## The composition model

```
website-katalog.yaml      →  workers: 2, resync: 30s
                              ↘
komposer.yaml (override)  →  workers: 4, resync: 15s  ← override wins
                              ↗
namespace-katalog.yaml    →  namespace-governance (no override)
```

The base Katalog is reusable across environments. The Komposer is
environment-specific. This is the pattern: one Katalog per CRD behavior,
one Komposer per environment.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
