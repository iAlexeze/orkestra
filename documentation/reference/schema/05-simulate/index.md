# Simulate

!!! note "Simulate is not a Kubernetes CRD"
    `kubectl apply` will not work. Orkestra kinds are consumed by the `ork` CLI and runtime — not by the Kubernetes API server. [Your CRD is enough](/blog/your-crd-is-enough/).

A `Simulate` Pattern declares what your operator should produce — which resources, in which cycle — and verifies it by running the reconciler against a fake in-memory cluster. No Kubernetes. No Docker. Sub-second.

```text
Motif     — smallest reusable unit
    ↓
Katalog   — operator declaration
    ↓
Komposer  — platform declaration
    ↓
Simulate  — in-memory reconciler verification
```

`ork validate` checks the structure. `ork simulate` runs it.

---

## Wire format

### With `expect:` — assert mode

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Simulate

metadata:
  name: my-operator-sim
  description: Verify Deployment and Service created in cycle 1

spec:
  katalog: ./katalog.yaml
  cr: ./cr.yaml
  cycles: 5
  skipExternal: false

  expect:
    noErrors: true
    ops:
      - cycle: 1
        verb: create
        resource: deployments
        name: my-app
    # absent:   # ops that must NOT appear — fill in for failure-path coverage
    #   - cycle: 1
    #     verb: create
    #     resource: deployments
    steady: true

```

### Without `expect:` — op-print mode

Same spec, no `expect:` block. Runs the reconciler, prints every op recorded per cycle, always exits 0. Used for exploration before you know the expected sequence.

### Aggregator form

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Simulate

metadata:
  name: my-suite

imports:
  - ./01-basic/simulate.yaml
  - ./02-with-hooks/simulate.yaml
```

No `spec:`. Each imported file runs independently. The suite fails if any file fails.

---

## Field reference

### `metadata`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Name of this simulate run — used in output headers |
| `description` | no | Free-text description shown in output and registry |

### `spec`

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `katalog` | yes | | Path to the `katalog.yaml` (or `komposer.yaml`) |
| `cr` | yes | | Path to the CR YAML file. Multi-doc YAML supported — each doc matched to its CRD by `kind`. Extra docs of the CRD-under-test's OWN kind aren't reconciled — they're seeded as pre-existing instances so `operator: unique` (and similar reconcile-time checks that list other instances) can be simulated. Only the first doc of that kind is actually reconciled |
| `cycles` | no | `10` | Maximum number of reconcile cycles to run |
| `skipExternal` | no | `false` | Stub all `external:` HTTP calls with an empty 200 response instead of hitting the real network |

### `spec.expect`

Optional. When omitted the run is in op-print mode — no assertions, always exits 0.

| Field | Type | Description |
|-------|------|-------------|
| `steady` | bool | Reconciler must reach steady state within `cycles` |
| `steadyAt` | int | Steady state must be reached by this cycle number (≤ value) |
| `noErrors` | bool | Any reconcile error fails the run |
| `ops` | `[]OpRule` | Ops that must appear — see below |
| `absent` | `[]OpRule` | Ops that must NOT appear — use for failure-path assertions |
| `crds` | `map[string]expect` | Per-CRD overrides for multi-CRD Katalogs. Top-level fields are defaults; a named entry overrides for that CRD only |

### `OpRule`

Used in both `ops` and `absent`.

| Field | Required | Description |
|-------|----------|-------------|
| `cycle` | yes | Reconcile cycle the op must (or must not) occur in |
| `verb` | yes | `create`, `update`, `delete`, or `patch` |
| `resource` | yes | Kubernetes resource type: `deployments`, `statefulsets`, `configmaps`, … |
| `name` | no | Assert a specific resource name |
| `count` | no | Minimum number of matching ops (default: ≥1). Ignored in `absent` |

---

## Validation

```bash
ork validate -f simulate.yaml
```

Catches: missing required fields, verbs outside the four allowed values, `imports` paths that don't resolve on disk.

---

## Scaffold with `ork simulate init`

`ork simulate init` runs the reconciler once, captures the cycle-1 create ops, and writes a `simulate.yaml` pre-filled with `expect:` rules — including a commented `absent:` block seeded with the first observed resource type.

```bash
ork simulate init           # writes simulate.yaml
ork simulate init --dry-run # prints to stdout, nothing written
ork simulate init --force   # overwrites existing simulate.yaml
```

→ [Scaffolding Tests](../../../concepts/simulate/03-init.md)

---

## Try it

```bash
ork init --pack beginner
cd beginner/02-with-serviceaccount
ork simulate
```

---

## See also

- [Katalog schema](../02-katalog/01-top-level.md) — the Katalog under simulation
- [simulate concept docs](../../../concepts/simulate/index.md) — how simulate works, aggregators, hooks and constructors
- [ork simulate CLI reference](../../cli/05-simulate.md)
