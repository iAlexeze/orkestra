# Custom Target Mode

`spec.custom.target` declares the runtime environment being tested when Orkestra is
not the operator. Bundle generation and Orkestra helm install/uninstall are skipped.
Everything else runs unchanged: cluster setup, CRD apply, setup manifests, CR apply,
assertions, and cleanup.

---

## Supported targets

| Value | Status | What it means |
|-------|--------|---------------|
| `kubernetes` | Supported | Your workload runs on Kubernetes. Orkestra manages the cluster lifecycle and assertions. Install your operator or chart via `setup.helm`. |
| `container` | Coming soon | Test a container image directly, without a cluster. |

---

## How to activate

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: my-operator-e2e

spec:
  custom:
    target: kubernetes
  crd: ./crd.yaml
  cr: ./cr.yaml
  setup:
    helm:
      - repo: https://charts.example.com
        chart: my-operator
        namespace: my-operator-system
        createNamespace: true
  expect:
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

---

## What runs, what is skipped

| Step | Normal | `custom.target: kubernetes` |
|------|--------|-----------------------------|
| Cluster setup | ✓ | ✓ |
| CRD apply (`spec.crd`) | ✓ | ✓ |
| Setup manifests (`spec.setup`) | ✓ | ✓ |
| Bundle generate + apply | ✓ | **skipped** |
| Orkestra helm install | ✓ | **skipped** |
| OCI import pre-pull | ✓ | **skipped** |
| CR apply | ✓ | ✓ |
| Expectations + assertions | ✓ | ✓ |
| Orkestra helm uninstall | ✓ | **skipped** |
| CRD / setup cleanup | ✓ | ✓ |

`spec.katalog` is optional when `custom.target` is set. If omitted, only `spec.crd`,
`spec.cr`, and `spec.setup` are used.

---

## The setup.helm pattern

Almost always paired with `setup.helm` to install the operator before the CR is
applied.

```yaml
spec:
  custom:
    target: kubernetes
  setup:
    helm:
      - repo: https://charts.jetstack.io
        chart: cert-manager
        namespace: cert-manager
        createNamespace: true
        version: v1.14.0
        values:
          installCRDs: true
    wait:
      - kind: Deployment
        name: cert-manager
        namespace: cert-manager
        ready: true
        timeout: 120s
```

---

## Use cases

### Migration parity testing

Migrating from a Kubebuilder operator to Orkestra? Write two e2e files — one with
Orkestra, one with `custom.target: kubernetes` and `setup.helm` pointing at your
existing binary — with identical assertions. When both pass, the migration is verified.

```text
my-operator/
├── e2e-orkestra.yaml          # spec.katalog: ./katalog.yaml
└── e2e-kubebuilder.yaml       # spec.custom.target: kubernetes
                               # setup.helm: my-old-operator-chart
                               # same assertions as e2e-orkestra.yaml
```

### Third-party operator smoke tests

Test the behavior of operators you did not write — FluxCD, cert-manager, Crossplane,
ArgoCD. Install via `setup.helm`, apply a CR, assert what the operator creates.

```yaml
spec:
  custom:
    target: kubernetes
  crd: ./certificate-crd.yaml
  cr: ./cr-certificate.yaml
  setup:
    helm:
      - repo: https://charts.jetstack.io
        chart: cert-manager
        # ...
  expect:
    - name: Certificate issued
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Secret
          name: my-tls-secret
          namespace: default
```

### Two-operator composition

Install two operators via `setup.helm`, apply CRs for both, assert they interact
correctly. Neither needs to be Orkestra.

```yaml
spec:
  custom:
    target: kubernetes
  setup:
    helm:
      - repo: https://operator-a.example.com
        chart: operator-a
      - repo: https://operator-b.example.com
        chart: operator-b
  cr: ./cr-combined.yaml
  expect:
    - name: Operator A and B outputs composed
      after: cr-applied
      timeout: 120s
      commands:
        - run: kubectl get compositeresource my-resource -o jsonpath='{.status.ready}'
          outputContains: "true"
```

---

## Example suite

```bash
ork init --pack use-cases/custom-operator
cd use-cases/custom-operator
```

| Example | What it shows |
|---------|---------------|
| `01-pure-custom` | `custom.target: kubernetes` with cert-manager installed via `setup.helm` |
| `02-side-by-side` | The same CR tested twice — once with Orkestra, once with a custom target — identical assertions |

→ See also: [Test anything that runs in Kubernetes](../../guides/e2e-universal.md)

---

→ Back: [04-imports](04-imports.md) | [Schema index](index.md)
