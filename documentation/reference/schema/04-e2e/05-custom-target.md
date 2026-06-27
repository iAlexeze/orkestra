# Custom Target Mode

`spec.custom.target` declares the runtime environment being tested when Orkestra is
not the operator. Bundle generation and Orkestra helm install/uninstall are skipped.
Everything else runs unchanged: cluster setup, setup manifests, assertions, and cleanup.

---

## Supported targets

| Value | Status | What it means |
|-------|--------|---------------|
| `kubernetes` | Supported | Your workload runs on Kubernetes. Orkestra manages the cluster lifecycle and assertions. Install your operator or chart via `setup.helm`. |
| `container` | Coming soon | Test a container image directly, without a cluster. |

---

## What runs, what is skipped

| Step | Normal | `custom.target: kubernetes` |
|------|--------|-----------------------------|
| Cluster setup | ✓ | ✓ |
| Setup manifests (`spec.setup`) | ✓ | ✓ |
| Bundle generate + apply | ✓ | **skipped** |
| Orkestra helm install | ✓ | **skipped** |
| OCI import pre-pull | ✓ | **skipped** |
| CRD apply (`spec.crd`) | ✓ | optional |
| CR apply (`spec.cr`) | ✓ | optional |
| Expectations + assertions | ✓ | ✓ |
| Orkestra helm uninstall | ✓ | **skipped** |
| CRD / setup cleanup | ✓ | ✓ |

`spec.katalog`, `spec.crd`, and `spec.cr` are all optional when `custom.target` is set.
For pure infrastructure tests (Helm charts, platform stacks), omit them entirely and
use `setup.apply` and `setup.helm` instead.

---

## The two patterns

### Pure Helm chart test (no CR)

When testing a Helm chart that does not involve a custom resource, omit `crd:` and
`cr:`. Apply any prerequisite manifests via `setup.apply`, install the chart via
`setup.helm`, use per-entry `wait:` to confirm each step before the next, and assert
everything with `after: setup-complete`.

```yaml
spec:
  custom:
    target: kubernetes

  setup:
    apply:
      - path: ./fixtures/config.yaml
        wait:
          - kind: ConfigMap
            name: my-config
            namespace: my-app
            timeout: 30s
    helm:
      - chart: ./
        release: my-app
        namespace: my-app
        createNamespace: true
        values:
          replicaCount: 2
        wait:
          - kind: Deployment
            name: my-app
            namespace: my-app
            ready: true
            timeout: 120s

  expect:
    - name: Deployment is ready
      after: setup-complete
      timeout: 30s
      resources:
        - kind: Deployment
          name: my-app
          namespace: my-app
          ready: true

    - name: Service is created
      after: setup-complete
      timeout: 30s
      resources:
        - kind: Service
          name: my-app
          namespace: my-app
```

`after: setup-complete` is also the default when `after:` is omitted — both forms are
equivalent.

### Operator with a CR

When testing an operator that reconciles a custom resource, apply the CRD and CR via
`setup.apply`, then install the operator via `setup.helm`. Use `after: setup-complete`
for infrastructure assertions and `after: cr-applied` / `after: cr-deleted` for
reconciliation assertions. The CR is applied as part of setup so the operator sees it
on first boot.

```yaml
spec:
  custom:
    target: kubernetes
  setup:
    apply:
      - path: ./crd.yaml
        wait:
          - kind: CustomResourceDefinition
            name: myresources.example.com
            timeout: 30s
      - ./cr.yaml
    helm:
      - repo: https://charts.example.com
        chart: my-operator
        namespace: my-operator-system
        createNamespace: true
        wait:
          - kind: Deployment
            name: my-operator
            namespace: my-operator-system
            ready: true
            timeout: 120s
  expect:
    - name: Operator is ready
      after: setup-complete
      timeout: 30s
      resources:
        - kind: Deployment
          name: my-operator
          namespace: my-operator-system
          ready: true
    - name: CR creates Deployment
      after: cr-applied
      timeout: 90s
      resources:
        - kind: Deployment
          name: my-app
          namespace: default
          ready: true
    - name: Cleanup verified
      after: cr-deleted
      timeout: 30s
      resources:
        - kind: Deployment
          name: my-app
          namespace: default
          count: 0
```

> When `cr:` is specified in spec, the runner manages its lifecycle — applying it
> before `after: cr-applied` blocks and deleting it before `after: cr-deleted` blocks.
> When the CR is applied via `setup.apply` instead, only `after: setup-complete`
> assertions run.

---

## Per-entry waits

Both `setup.apply` and `setup.helm` entries support an inline `wait:` list. The runner
blocks after each entry until all conditions pass before moving to the next. This gives
you ordered, incremental setup instead of a single global wait at the end.

```yaml
setup:
  apply:
    - namespace.yaml          # flat string — no wait
    - path: crd.yaml
      wait:
        - kind: CustomResourceDefinition
          name: myresources.example.com
          timeout: 30s
  helm:
    - repo: https://charts.example.com
      chart: cert-manager
      namespace: cert-manager
      createNamespace: true
      values:
        installCRDs: true
      wait:
        - kind: Deployment
          name: cert-manager
          namespace: cert-manager
          ready: true
          timeout: 120s
    - repo: https://charts.example.com
      chart: my-app
      namespace: my-app
      createNamespace: true
      wait:
        - kind: Deployment
          name: my-app
          namespace: my-app
          ready: true
          timeout: 120s
```

A global `setup.wait` list is also supported and runs after all apply and helm steps
complete. Use per-entry waits when order matters between steps, global wait for a final
readiness check that spans multiple resources.

---

## Lifecycle events

| Value | When it fires |
|-------|--------------|
| `setup-complete` | After all setup steps finish, before any CR is applied. Default when `after:` is omitted. |
| `cr-applied` | After `spec.cr` is applied to the cluster. Requires `spec.cr` to be set. |
| `cr-deleted` | After `spec.cr` is deleted from the cluster. Requires `spec.cr` to be set. |

---

## Use cases

### Self-test a Helm chart

Ship an `e2e.yaml` alongside the chart and run `ork e2e` in CI to verify every
release. No Orkestra operator required — the chart IS the thing under test.

```text
charts/my-chart/
├── Chart.yaml
├── templates/
├── values.yaml
├── e2e.yaml          ← spec.custom.target: kubernetes, setup.helm: chart: ./
└── fixtures/
    └── config.yaml
```

### Migration parity testing

Migrating from a Kubebuilder operator to Orkestra? Write two e2e files with identical
assertions — one for Orkestra, one for the legacy chart. When both pass, the migration
is verified.

```text
my-operator/
├── e2e-orkestra.yaml        # spec.katalog: ./katalog.yaml
└── e2e-legacy.yaml          # spec.custom.target: kubernetes
                             # setup.helm: my-old-operator-chart
                             # same assertions
```

### Third-party operator smoke tests

Test the behavior of operators you did not write — FluxCD, cert-manager, Crossplane,
ArgoCD — and assert the resources they create.

### Platform stack tests

Install multiple tools together and assert they interact correctly. No single CR; all
assertions are `after: setup-complete`.

---

→ Back: [04-imports](04-imports.md) | [Schema index](index.md)
