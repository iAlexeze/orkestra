# Core spec fields

The four required inputs that tell `ork e2e` what to test and where to run.

---

## `spec.katalog`

Path to the `kind: Katalog` file under test. Relative to the E2E file's location.

```yaml
spec:
  katalog: ./katalog.yaml
```

Optional when `spec.custom.target` is set — see [05-custom-target.md](05-custom-target.md).

---

## `spec.crd`

Path to the CRD YAML to apply before the test run. If the CRD is already installed in the target cluster, this is a no-op.

```yaml
spec:
  crd: ./crd.yaml
```

---

## `spec.cr`

Path to the Custom Resource YAML to apply at the start of the `cr-applied` phase.

```yaml
spec:
  cr: ./cr.yaml
```

---

## `spec.cluster`

Controls where the test runs.

```yaml
spec:
  cluster:
    provider: kind        # kind | existing
    name: ork-e2e
    reuse: false
```

| Field | Default | Description |
|-------|---------|-------------|
| `provider` | `kind` | `kind` creates a local kind cluster. `existing` uses the current kubeconfig context. |
| `name` | `ork-e2e` | Cluster name. For `kind`, the kind cluster name. For `existing`, used in output only. |
| `reuse` | `false` | When `true`, keeps the cluster after the test — useful for debugging failed runs. |

---

---

## `spec.custom`

Declares the target runtime when Orkestra is not the operator under test. Bundle generation and Orkestra helm install/uninstall are skipped. Everything else runs unchanged.

```yaml
spec:
  custom:
    target: kubernetes
  crd: ./crd.yaml
  cr: ./cr.yaml
  setup:
    helm:
      - repo: https://charts.example.com
        chart: my-operator
```

`target` must be `kubernetes`. `container` is reserved and not yet supported.

See [05-custom-target.md](05-custom-target.md) for full documentation and use cases.

---

## `imports`

A top-level list of other E2E files to run after this test completes. Used to compose test suites without a cluster-per-test overhead. See [04-imports.md](04-imports.md) for the full field reference.

```yaml
imports:
  - ./auth-e2e.yaml
  - ./rbac-e2e.yaml
  - path: ./infra-e2e.yaml
    freshCluster: true     # this one gets its own cluster
```

---

→ Next: [02-setup.md](02-setup.md) — prerequisite resources before the operator starts
