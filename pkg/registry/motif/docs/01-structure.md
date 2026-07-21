# 01 — Motif YAML Structure

A Motif is a YAML file with `kind: Motif`. It declares a set of inputs and a set of Kubernetes resource templates. Callers bind values to the inputs and get back concrete resources.

## Minimal example

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Motif
metadata:
  name: postgres
  version: v1
  description: PostgreSQL stateful database

inputs:
  - name: image
    required: true
    description: Container image (e.g. postgres:16)

  - name: volumeSize
    default: "10Gi"
    description: PVC size

resources:
  statefulSets:
    - name: "{{ .metadata.name }}-db"
      image: "{{ index .inputs \"image\" }}"
      replicas: "1"
      storageSize: "{{ index .inputs \"volumeSize\" }}"
      mountPath: /var/lib/postgresql/data
```

## Top-level fields

| Field | Required | Notes |
|-------|----------|-------|
| `apiVersion` | ✓ | `orkestra.orkspace.io/v1` |
| `kind` | ✓ | Must be `Motif` — enforced by `StrictUnmarshal` + kind check |
| `metadata.name` | ✓ | Identifies the Motif in error messages and the registry |
| `metadata.version` | — | Semantic version; used when pulling from OCI registries |
| `metadata.description` | — | Human-readable summary |
| `inputs` | — | Declared input parameters (see below) |
| `resources` | ✓ | One or more resource blocks (see below) |

## Inputs

Each input declares one parameter the caller must or may supply:

```yaml
inputs:
  - name: image          # required — no default
    required: true

  - name: volumeSize     # optional — default used when caller omits it
    default: "10Gi"

  - name: user           # optional — no default, empty string when omitted
    description: Database user
```

Rules:
- `name` must be unique within the Motif.
- A `required: true` input must not have a `default`.
- An optional input without a `default` resolves to an empty string when not supplied.

Template expressions in `resources` reference inputs via `{{ index .inputs "name" }}`:

```yaml
image: "{{ index .inputs \"image\" }}"
```

The `.metadata.*` namespace in resource templates refers to the owner CR's metadata (name, namespace) — not the Motif's metadata.

## Resources

`resources` maps directly to `HookTemplates`. Every key is a resource type supported by the reconciler:

| Key | Kubernetes kind |
|-----|----------------|
| `statefulSets` | StatefulSet |
| `services` | Service |
| `configMaps` | ConfigMap |
| `secrets` | Secret |
| `persistentVolumeClaims` | PersistentVolumeClaim |
| `deployments` | Deployment |
| `serviceAccounts` | ServiceAccount |
| `roles` | Role |
| `roleBindings` | RoleBinding |

Any resource type that appears in `HookTemplates` is valid in a Motif's `resources` block.

→ Next: [02-loading.md](02-loading.md)
