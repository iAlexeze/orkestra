## Nested Spec Paths

### `serve.fields.path` 

By default, `serve.fields` maps field names directly to top-level `spec` paths. Use `path` to map a field to a nested location in the CRD `spec`.

```yaml
serve:
  fields:
    # Flat field — maps to spec.repository
    repository:
      label: "Repository"

    # Nested field — maps to spec.app.repository
    repository:
      path: app.repository
      label: "Repository"

    # Deeply nested — maps to spec.app.resources.cpu
    cpu:
      path: app.resources.cpu
      label: "CPU Request"
```

Callers submit flat field names — they don't need to know about nesting. The gateway maps the field to the correct location in the CRD.

```bash
curl -X POST /api/v1/apply \
  -d '{
    "target": "smartapp",
    "repository": "myorg/payments-api",  # → spec.app.repository
    "cpu": "500m"                         # → spec.app.resources.cpu
  }'
```

### Why Use `path`

| Without `path` | With `path` |
|----------------|-------------|
| Fields must match CRD structure | Fields are flat and caller-friendly |
| Callers must know nested paths | Callers submit simple field names |
| CRD evolution breaks callers | Gateway maps to new paths |
| UI fields show dot-paths | UI fields show clean names |

### Validation

`ork validate` enforces:

- **Unique paths** — no two fields can map to the same `spec` location
- **Valid format** — path segments must be valid Kubernetes names (alphanumeric, `_`, `-`, `.`)
- **No empty segments** — `app..repository` is rejected
- **No leading/trailing dots** — `.app.repository` is rejected

### Schema Validation (Not Yet Implemented)

Path existence in the CRD schema is not yet validated by `ork validate`. The platform team must verify that nested paths exist in the CRD spec. This will be added in a future release when OpenAPI schemas are loaded into the Katalog.

### Example

**CRD:**

```yaml
spec:
  app:
    repository: string
    image: string
    resources:
      cpu: string
      memory: string
  scaling:
    replicas: integer
    minReplicas: integer
    maxReplicas: integer
```

**Serve Config:**

```yaml
serve:
  fields:
    repository:
      path: app.repository
      label: "Repository"
    image:
      path: app.image
      label: "Container Image"
    cpu:
      path: app.resources.cpu
      label: "CPU Request"
    memory:
      path: app.resources.memory
      label: "Memory Request"
    replicas:
      path: scaling.replicas
      label: "Replicas"
```

**Caller Request:**

```json
{
  "target": "smartapp",
  "repository": "myorg/payments-api",
  "image": "ghcr.io/myorg/app:v1",
  "cpu": "500m",
  "memory": "512Mi",
  "replicas": 3
}
```

**Generated CR:**

```yaml
spec:
  app:
    repository: myorg/payments-api
    image: ghcr.io/myorg/app:v1
    resources:
      cpu: 500m
      memory: 512Mi
  scaling:
    replicas: 3
```

### Related

→ [`serve.fields`](20-serve.md#servefieldsname) — field configuration reference

→ [`serve labels/annotations`](20-serve.md#serve-labelsannotations) — labels and annotations as fields

→ [Target Mode API](../../../concepts/self-service/02-target-mode.md) — submitting fields instead of CRs