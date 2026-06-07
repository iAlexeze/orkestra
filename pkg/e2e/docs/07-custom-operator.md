# Custom Operator Mode

`spec.customOperator: true` declares that this test uses its own operator — not Orkestra's reconcile loop. Bundle generation and Orkestra helm install/uninstall are skipped. Everything else runs unchanged.

## Activation

Declare it in the spec file — the file is the source of truth, there is no CLI flag:

```yaml
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
```

## What is skipped

| Step | Normal | `customOperator: true` |
|------|--------|-----------------------|
| Bundle generate + apply | ✓ | skipped |
| OCI import pre-pull | ✓ | skipped |
| Orkestra helm install | ✓ | skipped |
| Orkestra helm uninstall | ✓ | skipped |
| Everything else | ✓ | ✓ |

`spec.katalog` is optional when `customOperator: true`. The spec only needs `spec.cr` at minimum.

## Use cases

### Third-party operator smoke tests
Install cert-manager, FluxCD, Crossplane via `setup.helm`, apply a CR, assert what it creates:

```yaml
spec:
  customOperator: true
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
Two e2e files, same CRD, same assertions — one via Orkestra katalog, one via `customOperator`. When both pass, the implementations are equivalent:

```
my-operator/
├── e2e-orkestra.yaml   # spec.katalog: ./katalog.yaml
└── e2e-custom.yaml     # spec.customOperator: true + setup.helm: my-old-operator
```

### Universal test harness
`customOperator: true` exposes `ork e2e`'s assertion infrastructure — polling loops, count checks, command assertions, cleanup verification — to any operator framework: controller-runtime, Operator SDK, kube-rs, kopf.

## Example
See [`examples/use-cases/custom-operator/`](../../../examples/use-cases/custom-operator/README.md) for two runnable examples.
