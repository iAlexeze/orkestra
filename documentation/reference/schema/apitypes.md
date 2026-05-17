# apiTypes

Declares the Kubernetes API group, kind, and version of the CRD being managed.
Every CRDEntry must have `apiTypes`.

```yaml
apiTypes:
  group: apps.example.io     # required
  version: v1alpha1          # required
  kind: Database             # required
  plural: databases          # required

  # Typed mode — required when location is set
  location: github.com/example/operator/api/v1alpha1
  object: Database
  objectList: DatabaseList
  alias: dbv1                # auto-derived if omitted

  apiPath: /apis             # almost always leave empty
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `group` | yes | Kubernetes API group (e.g. `platform.example.io`) |
| `version` | yes | API version (e.g. `v1alpha1`) |
| `kind` | yes | Resource Kind, as it appears in YAML (e.g. `Database`) |
| `plural` | yes | Lowercase plural resource name used for REST URLs (e.g. `databases`) |
| `location` | typed mode | Fully-qualified Go import path for the API types package |
| `object` | typed mode | Go type name for a single CR instance |
| `objectList` | typed mode | Go type name for the CR list |
| `alias` | no | Go import alias — auto-derived from the last two path segments if omitted |
| `apiPath` | no | REST API path prefix. Leave empty (`/apis` default). Override to `/api` only for core Kubernetes types (Pod, ConfigMap, etc.). |

## Dynamic mode vs typed mode

**Dynamic mode** — omit `location`, `object`, and `objectList`. Orkestra uses `unstructured.Unstructured` for all CR operations. Simpler to start with; no code generation required.

```yaml
apiTypes:
  group: apps.example.io
  version: v1alpha1
  kind: Database
  plural: databases
```

**Typed mode** — set `location`, `object`, and `objectList`. Orkestra uses your generated Go types. Required for Go hooks and constructor.

```yaml
apiTypes:
  group: apps.example.io
  version: v1alpha1
  kind: Database
  plural: databases
  location: github.com/example/operator/api/v1alpha1
  object: Database
  objectList: DatabaseList
```

Run `ork generate registry` after setting typed mode to produce the Go registry that wires the scheme.

---

→ Next: [operatorbox.md](operatorbox.md)
