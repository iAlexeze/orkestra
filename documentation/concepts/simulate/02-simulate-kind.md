# simulate.yaml

`simulate.yaml` declares what your operator should produce — which resources, in which cycle — so every run is repeatable and verifiable without a cluster.

```bash
ork simulate          # auto-detects simulate.yaml in the current directory
ork simulate -f simulate.yaml
```

---

## Schema

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: database-sim
  description: Verify hooks fire and StatefulSet + CronJob created in cycle 1

spec:
  katalog: ./katalog.yaml
  cr: ./cr.yaml
  cycles: 5            # default 10 when omitted
  skipExternal: false  # stub external: HTTP calls (default: hit real network)

  expect:
    steady: true       # reconciler must stabilise within cycles
    noErrors: true     # any reconcile error fails the run
    ops:
      - cycle: 1
        verb: create
        resource: statefulsets
      - cycle: 1
        verb: create
        resource: cronjobs
        name: my-db-backup   # optional: assert a specific resource name
```

**`expect:` is optional.** Without it, simulate runs in op-print mode — prints every op recorded per cycle, always exits 0. Add `expect:` when you know the sequence and want it enforced.

---

## expect.ops rules

Each rule asserts that at least one recorded op in the declared cycle matches all non-empty fields.

| Field | Required | Description |
|-------|----------|-------------|
| `cycle` | yes | Reconcile cycle the op must occur in |
| `verb` | yes | `create`, `update`, `delete`, or `patch` |
| `resource` | yes | Kubernetes resource type (`statefulsets`, `deployments`, …) |
| `name` | no | Assert a specific resource name |
| `count` | no | Minimum number of matching ops (default: 1) |

`verb` is validated by `ork validate` — any value outside the four above is an error.

---

## Multi-CRD: expect.crds

For katalogs with multiple CRDs, `expect.crds` mirrors the `spec.crds` map from the katalog. Top-level `expect` fields are defaults; a per-CRD entry overrides for that CRD only.

```yaml
spec:
  katalog: ./katalog.yaml
  cr: ./cr.yaml         # multi-doc YAML — engine matches each doc to its CRD by kind

  expect:
    noErrors: true      # default: applies to all CRDs
    crds:
      database:
        steady: true
        ops:
          - cycle: 1
            verb: create
            resource: statefulsets
      website:
        steady: true
        ops:
          - cycle: 1
            verb: create
            resource: deployments
```

---

## Aggregator form

A `simulate.yaml` can aggregate other simulate specs as a suite. Useful for multi-operator examples.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: advanced-suite

imports:
  - ./09-hooks/simulate.yaml
  - ./10-constructor/simulate.yaml
```

An aggregator has no `spec:`. Each imported file runs independently; the suite fails if any file fails.

---

## Output

### With expect — assert mode

```text
Simulating database/my-db

  Cycle 1:
    + statefulsets/my-db
    + services/my-db-svc
    + cronjobs/my-db-backup
    ~ status/my-db
  Cycle 2:
    ~ status/my-db
  (cycles 3–5: identical)

  ✓ Steady state at cycle 2 in 4ms

  ✓ steady
  ✓ noErrors
  ✓ create statefulsets (cycle 1)
  ✓ create cronjobs/my-db-backup (cycle 1)

  PASS
```

### Without expect — op-print mode

Same cycle output, no assertions, always exits 0.

---

## Validation

```bash
ork validate -f simulate.yaml
```

Reports missing required fields, verbs outside the allowed set, and imports that don't exist on disk:

```text
Validating Simulate...

  ✓ metadata.name: database-sim
  ✓ spec.katalog: ./katalog.yaml (found)
  ✓ spec.cr: ./cr.yaml (found)
  ⚠ spec.cycles: not set — defaulting to 10
  ✓ expect.ops: 2 rule(s)
```

---

## vs. e2e.yaml

| | `simulate.yaml` | `e2e.yaml` |
|---|---|---|
| Cluster required | no | yes (kind) |
| Speed | <1s | 2–5 min |
| What is asserted | reconciler ops by cycle | Kubernetes resources in a live cluster |
| `expect:` | ops, steady, noErrors | resource kind/name/ready/count |
| When to use | reconciler behavior | integration truth |
