# Multi-tenancy

One Orkestra runtime, multiple teams. Each Katalog declares `metadata.namespace` and the Control Center groups CRDs into separate panels per namespace. Workers, health tracking, informers, and reconcile loops are independent per CRD regardless of namespace.

## Namespace declaration

```yaml
metadata:
  name: payments
  namespace: fintech-team
```

Omitting `namespace` defaults to `"default"`.

## Cross-read access control

```yaml
crossAccess: false   # Katalog-level default

spec:
  crds:
    public-crd:
      crossAccess: true   # CRD-level override
    private-crd: {}       # inherits crossAccess: false
```

A `cross:` reference to an opted-out CRD returns `found: "false"` silently.

## Examples

| | |
|---|---|
| [01 — Basic namespacing](./01-basic-namespacing/README.md) | Two teams, separate CC panels |
| [02 — Access control](./02-cross-access-control/README.md) | Katalog-level `crossAccess: false` with CRD override |
| [03 — Shared platform](./03-shared-platform/README.md) | Platform infra consumed by application teams |

## Run all

From the root directory:

### Apply CRDs

```bash
kubectl apply -f 01-basic-namespacing/crd.yaml
kubectl apply -f 02-cross-access-control/crd.yaml
kubectl apply -f 03-shared-platform/crd.yaml
```

### Validate

```bash
ork validate
```

### Run

```bash
ork run
```

#### Start the Control Center:

```bash
ork control
```
