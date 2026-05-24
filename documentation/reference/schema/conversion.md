# conversion

Enables multi-version CRD support via the Orkestra conversion webhook.
When declared, Orkestra registers a `/convert` endpoint and patches the CRD's caBundle automatically.

```yaml
conversion:
  storageVersion: v1          # version all objects are stored as internally
  updateCRD: true             # patch CRD caBundle on startup
  participant: false          # mark as participant without declaring paths

  paths:
    - from: v1alpha1
      to: v1
      spec:
        engine: "{{ .Spec.DatabaseEngine }}"
        replicas: "{{ .Spec.Size }}"

    - from: v1
      to: v1alpha1
      spec:
        databaseEngine: "{{ .Spec.Engine }}"
        size: "{{ .Spec.Replicas }}"
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `storageVersion` | yes | The version all objects are persisted as in etcd. All other versions are converted on read/write. |
| `updateCRD` | no | Automatically patch `spec.conversion.webhook.clientConfig.caBundle` on startup (default: `false`). |
| `participant` | no | Mark this CRD as a conversion participant in the hub-and-spoke pattern without declaring paths. Useful for the control center to add metrics to the storage version. |
| `paths` | no | Explicit conversion path declarations. |

## `paths`

Each path declares how to convert from one version to another.
Paths are bidirectional — declare both directions explicitly.

| Field | Required | Description |
|-------|----------|-------------|
| `from` | yes | Source API version (bare, e.g. `v1alpha1`) |
| `to` | yes | Target API version (bare, e.g. `v1`) |
| `spec` | yes | Output spec in the target version format. Supports Go templates against the source object. |

## How it works

1. Kubernetes calls the `/convert` endpoint when a version mismatch exists between stored and requested versions.
2. Orkestra evaluates the `spec` template for the matching `from`→`to` path.
3. The converted object is returned to Kubernetes.

`security.conversion.enabled: true` must be set for the `/convert` endpoint to be registered.


---
