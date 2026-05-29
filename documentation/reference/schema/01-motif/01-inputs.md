# Inputs

`inputs` is an ordered list of named parameters the Motif exposes to its consumers. Each input can be required or optional with a default. Inside the Motif's resource templates, inputs are referenced as `{{ .inputs.<name> }}`.

---

## Declaration

```yaml
inputs:
  - name: image
    description: PostgreSQL image (e.g. postgres:16)
    required: true

  - name: replicas
    description: Number of StatefulSet replicas
    default: "1"

  - name: volumeSize
    description: PVC storage size
    default: "10Gi"

  - name: resourceProfile
    description: CPU/memory profile (small, standard, compute-heavy)
    default: "standard"

  - name: enableUI
    description: Deploy the pgAdmin management UI (true/false)
    default: "true"
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Input identifier. Referenced as `{{ .inputs.<name> }}` inside resource templates. |
| `description` | no | Shown in `ork registry inspect` and registry UI. |
| `required` | no | When `true`, the importer must supply this input in `with:`. Missing required inputs fail at startup — not at reconcile time. |
| `default` | no | Value used when `with:` does not supply this input. Only valid when `required` is `false` or omitted. |
| `type` | no | Type hint: `string`, `int`, `bool`. Reserved for future enforcement; not validated today. |

---

## Using inputs in templates

Inside resource blocks, inputs are available as `{{ .inputs.<name> }}`.

```yaml
resources:
  statefulSets:
    - name: "{{ .metadata.name }}-postgres"
      image: "{{ .inputs.image }}"
      replicas: "{{ .inputs.replicas }}"
      resources:
        profile: "{{ .inputs.resourceProfile }}"
      volumeClaimTemplates:
        - name: data
          storageSize: "{{ .inputs.volumeSize }}"
          mountPath: /var/lib/postgresql/data
```

Template context inside a Motif:

| Variable | Value |
|----------|-------|
| `{{ .metadata.name }}` | Name of the CR being reconciled — not the Motif itself. |
| `{{ .metadata.namespace }}` | Namespace of the CR being reconciled. |
| `{{ .inputs.<name> }}` | Bound value for this input from the importer's `with:`. |
| `{{ .spec.* }}` | CR spec fields — available but discouraged. Prefer passing them via `with:`. |

---

## Conditional inputs

Use inputs to gate resources via `when:`:

```yaml
inputs:
  - name: enableUI
    default: "false"
    description: Deploy the management UI (true/false)

resources:
  deployments:
    - name: "{{ .metadata.name }}-admin"
      image: "{{ .inputs.pgAdminImage }}"
      when:
        - field: inputs.enableUI
          equals: "true"

  services:
    - name: "{{ .metadata.name }}-admin-svc"
      port: 80
      type: LoadBalancer
      when:
        - field: inputs.enableUI
          equals: "true"
```

When `enableUI` is `"false"` (or the default), neither the Deployment nor the Service is created.

---

## Binding at import time

The Katalog supplies values for each input in `with:`. Values are Go templates evaluated against the CR being reconciled.

```yaml
imports:
  - motif: oci://ghcr.io/orkspace/patterns/motifs/postgres:v0.1.0
    with:
      image: "{{ .spec.postgresImage }}"
      volumeSize: "{{ .spec.storage | default \"10Gi\" }}"
      replicas: "{{ .spec.replicas | default \"1\" }}"
      enableUI: "false"
```

Binding rules:

- A required input not in `with:` → validation error at startup.
- An optional input not in `with:` → the input's `default` is used.
- `with:` values are Go templates evaluated in the CR reconcile context.
- Literal values do not need to be templates: `enableUI: "false"` is valid.

---

→ Next: [02-resources.md](02-resources.md) — resource blocks, template context, onCreate, status, admission
