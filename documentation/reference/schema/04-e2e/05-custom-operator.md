# Custom Operator Mode

`spec.customOperator: true` declares that this test uses its own operator — not
Orkestra's reconcile loop. Bundle generation and Orkestra helm install/uninstall are
skipped. Everything else runs unchanged.

---

## How to activate

Declare it in the spec file. The file is the source of truth — there is no CLI flag.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: my-operator-e2e

spec:
  customOperator: true
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

| Step | Normal | `customOperator: true` |
|------|--------|-----------------------|
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

`spec.katalog` is optional when `customOperator: true`. If omitted, the test has
no katalog — only `spec.crd`, `spec.cr`, and `spec.setup` are used.

---

## The setup.helm pattern

Almost always paired with `setup.helm` to install the operator before the CR is
applied. The setup section installs the operator; assertions check what it creates.

```yaml
spec:
  customOperator: true
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

You are migrating from a Kubebuilder operator to Orkestra. The old operator is still
in production. Write two e2e files — one with Orkestra, one with `customOperator` +
`setup.helm` installing your existing binary — and use identical assertions in both.
When both pass, the migration is verified.

```text
my-operator/
├── e2e-orkestra.yaml        # spec.katalog: ./katalog.yaml
└── e2e-kubebuilder.yaml     # spec.customOperator: true
                             # setup.helm: my-old-operator-chart
                             # same assertions as e2e-orkestra.yaml
```

### Third-party operator smoke tests

Test the behavior of operators you did not write — FluxCD, cert-manager, Crossplane,
ArgoCD. Install via `setup.helm`, apply a CR, assert what the operator creates.

```yaml
spec:
  customOperator: true
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
  customOperator: true
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

### ork e2e as a universal test harness

The assertion infrastructure — polling loops, count checks, command assertions,
cleanup verification — is valuable regardless of which operator framework is under
test. `customOperator: true` exposes it to any team writing Kubernetes operators:
controller-runtime, Operator SDK, kube-rs, kopf, or anything else. They all get
the same CI harness as first-class Orkestra users.

---

## Example suite

For two runnable examples, run:

```bash
ork init --pack use-cases/custom-operator
cd use-cases/custom-operator
```

| Example | What it shows |
|---------|---------------|
| `01-pure-custom` | `customOperator: true` with a Kubebuilder-style operator installed via `setup.helm` |
| `02-side-by-side` | The same CR tested twice — once with Orkestra, once with a custom operator — identical assertions |

---

→ Back: [04-imports](04-imports.md) | [Schema index](index.md)
