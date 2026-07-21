# Custom Target Mode

`spec.custom.target` declares the runtime environment being tested when Orkestra is
not the operator. Bundle generation and Orkestra helm install/uninstall are skipped.
Everything else runs unchanged.

## Activation

Declare it in the spec file — the file is the source of truth, there is no CLI flag:

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
        namespace: my-operator-system
        createNamespace: true
```

Supported targets: `kubernetes`. `container` is coming soon.

## What is skipped

| Step | Normal | `custom.target: kubernetes` |
|------|--------|-----------------------------|
| Bundle generate + apply | ✓ | skipped |
| OCI import pre-pull | ✓ | skipped |
| Orkestra helm install | ✓ | skipped |
| Orkestra helm uninstall | ✓ | skipped |
| Everything else | ✓ | ✓ |

`spec.katalog` is optional when `custom.target` is set. The spec only needs `spec.cr` at minimum.

## Use cases

### Third-party operator smoke tests
Install cert-manager, FluxCD, Crossplane via `setup.helm`, apply a CR, assert what it creates:

```yaml
spec:
  custom:
    target: kubernetes
  cr: ./cr-certificate.yaml
  setup:
    helm:
      - repo: https://charts.jetstack.io
        chart: cert-manager
        values:
          installCRDs: true
    wait:
      - kind: Deployment
        name: cert-manager-webhook
        namespace: cert-manager
        ready: true
        timeout: 120s
```

### Migration parity testing
Two e2e files, same CRD, same assertions — one via Orkestra katalog, one via `custom.target`. When both pass, the implementations are equivalent:

```
my-operator/
├── e2e-orkestra.yaml   # spec.katalog: ./katalog.yaml
└── e2e-custom.yaml     # spec.custom.target: kubernetes + setup.helm: my-old-operator
```

### Universal test harness
`custom.target: kubernetes` exposes `ork e2e`'s assertion infrastructure — polling loops,
count checks, command assertions, cleanup verification — to any workload that runs on
Kubernetes: controller-runtime operators, Helm charts, raw manifests, third-party tools.

## Example

```bash
ork init --pack use-cases/custom-operator
cd custom-operator

# Follow the steps in the README.
```