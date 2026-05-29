# Katalog

!!! note "Katalog is not a Kubernetes CRD"
    `kubectl apply` will not work. Orkestra kinds are consumed by the `ork` CLI and runtime — not by the Kubernetes API server. [Your CRD is enough](../../../blog/02-your-crd-is-enough.md).

A Katalog declares one or more CRDs and defines how Orkestra manages them.
It is the **unit of operator definition**.

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1   # required
kind: Katalog                          # required

metadata:
  name: my-operator                    # required
  description: string                  # optional

spec:
  finalizers:                          # optional — applied to every CRD
    - platform.example.io/cleanup

  crds:                                # required — map of CRD entries by name
    <name>:                            # ← map key is the CRD name
      ...                              # → see crd-entry.md

security:                              # optional → see katalog-security.md
  ...

notification:                          # optional → see katalog-notification.md
  ...

providers:                             # optional → see katalog-providers.md
  - ...
```

## `metadata`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Unique identifier. Written as the `managed-by` annotation on all CRs. |
| `description` | no | Shown in the `/katalog` API response. |

## `spec.finalizers`

Katalog-level finalizers applied to every CRD in this Katalog.
Override per-CRD via [`operatorBox.finalizers`](04-operatorbox.md).

## `spec.crds`

A **map** — the key is the CRD name, the value is a `CRDEntry`.

```yaml
spec:
  crds:
    database:          # ← CRD name
      enabled: true
      apiTypes:
        group: apps.example.io
        version: v1alpha1
        kind: Database
        plural: databases
```

→ Full field reference: [crd-entry.md](02-crd-entry.md)

## See also

- [crd-entry.md](02-crd-entry.md)
- [katalog-security.md](10-katalog-security.md)
- [katalog-notification.md](11-katalog-notification.md)
- [katalog-providers.md](12-katalog-providers.md)
- [komposer.md](../03-komposer/index.md) — compose multiple Katalogs
