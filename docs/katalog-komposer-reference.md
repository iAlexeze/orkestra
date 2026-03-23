# Katalog and Komposer Reference

## Katalog

A **Katalog** declares CRDs and their behavior. It is the source of truth for what Orkestra manages.

### Required Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | string | Yes | Must be `orkestra.konductor.io/v1Alpha` |
| `kind` | string | Yes | Must be `Katalog` |
| `metadata.name` | string | Yes | Unique identifier for this Katalog |
| `spec.crds` | array | Yes | List of CRD entries (at least one) |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `metadata.description` | string | `""` | Human-readable description |
| `spec.finalizers` | array | `[]` | Katalog-level finalizers inherited by all CRDs |

### CRD Entry Fields (`spec.crds[]`)

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `name` | string | — | Yes | Unique identifier, lowercase kebab-case |
| `apiTypes.kind` | string | — | Yes | Resource kind (e.g., Pod, Website) |
| `apiTypes.group` | string | — | Yes* | API group (inferred for built-ins) |
| `apiTypes.version` | string | — | Yes* | API version (inferred for built-ins) |
| `apiTypes.plural` | string | — | Yes* | Plural resource name (inferred for built-ins) |
| `apiTypes.location` | string | — | No | Go import path for typed mode |
| `apiTypes.apiPath` | string | `/apis` | No | REST API path (`/api` for core types) |
| `enabled` | bool | `true` | No | Include in runtime |
| `namespaced` | bool | `true` | No | Scope: true = namespace, false = cluster |
| `namespace` | string | `""` | No | Target namespace (empty = all namespaces) |
| `workers` | int | `0` | No | Worker count (0 = use default) |
| `resync` | duration | `0` | No | Resync interval (0 = use default) |
| `dependsOn` | array | `[]` | No | Names of CRDs that must start first |
| `mode` | string | auto | No | `typed` or `dynamic` (auto-detected) |
| `critical` | bool | `false` | No | If true, CRD degradation degrades whole operator |
| `description` | string | `""` | No | Human-readable description |
| `reconciler.default` | bool | `true` | No | Use GenericReconciler |
| `reconciler.finalizers` | array | `[]` | No | Per-CRD finalizers (overrides Katalog-level) |
| `reconciler.onCreate` | object | `{}` | No | Templates for resource creation |
| `reconciler.onReconcile` | object | `{}` | No | Templates for drift correction |
| `reconciler.onDelete` | object | `{}` | No | Templates for cleanup |
| `reconciler.hooks` | object | — | No | Go hook function declaration |
| `reconciler.constructor` | object | — | No | Custom reconciler constructor |
| `reconciler.validation` | object | — | No | Validation rules (planned) |
| `reconciler.mutation` | object | — | No | Mutation rules (planned) |
| `queue.maxQueueDepth` | int | `0` | No | Max queue depth (0 = use default) |
| `queue.degradeThreshold` | int | `0` | No | Failures before degraded (0 = use default) |
| `endpoints.enabled` | bool | `true` | No | Disable all endpoints for this CRD |
| `endpoints.health` | bool | `true` | No | Disable `/health` endpoint |
| `endpoints.info` | bool | `true` | No | Disable `/info` endpoint |
| `endpoints.validation` | bool | `true` | No | Disable `/validation` endpoint |
| `labels` | array | `[]` | No | Static labels for all created resources |

\* For built-in Kubernetes resources (Pod, Deployment, Secret, etc.), `group`, `version`, and `plural` are auto‑discovered. Omit them and Orkestra enriches from the API server.

---

## Komposer

A **Komposer** composes multiple Katalogs into one runtime. It declares where to find CRD definitions.

### Required Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | string | Yes | Must be `orkestra.konductor.io/v1Alpha` |
| `kind` | string | Yes | Must be `Komposer` |
| `metadata.name` | string | Yes | Unique identifier for this Komposer |
| `sources` | object | Yes | At least one source must be declared |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `metadata.description` | string | `""` | Human-readable description |
| `spec.crds` | array | `[]` | Inline CRD overrides (win on name conflict) |

### Sources (`sources`)

| Field | Type | Description |
|-------|------|-------------|
| `files` | array | Local or remote Katalog files |
| `helm` | array | Helm charts that render Katalog templates |

### File Source Entry

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` / direct string | string | Yes | Local path, remote URL, or `$ENV_VAR` |
| `auth.type` | string | No | `bearer`, `github`, or `basic` |
| `auth.fromEnv` | string | For bearer/github | Environment variable with token |
| `auth.usernameFromEnv` | string | For basic | Environment variable with username |
| `auth.passwordFromEnv` | string | For basic | Environment variable with password |

**Simple form:** `sources.files: - ./path.yaml`  
**Authenticated form:**
```yaml
- url: https://private.com/katalog.yaml
  auth:
    type: bearer
    fromEnv: MY_TOKEN
```

### Helm Source Entry

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `repo` | string | Yes | Helm repo URL, local path, or Git URL |
| `chart` | string | Yes | Chart name (or path within repo for Git) |
| `version` | string | Yes* | Chart version (required for remote/Git) |
| `path` | string | No | Path within Git repo (for Git sources) |
| `valueFiles` | array | No | Local/remote values files or `$ENV_VAR` |
| `values` | object | No | Inline values (like `helm --set`) |

\* For local charts, version is optional.

---

## Inline CRD Overrides (Komposer Only)

CRDs declared in `spec.crds` on a Komposer are merged last and win on name conflict. Use for environment-specific adjustments.

All fields from the Katalog CRD entry are valid, with the same defaults.

---

## Example Katalog

```yaml
apiVersion: orkestra.konduktor.io/v1Alpha
kind: Katalog
metadata:
  name: website-katalog
  description: Manages Website CRD with Deployment and Service

spec:
  finalizers:
    - finalizer.demo.orkestra.io/website

  crds:
    - name: frontend
      enabled: true
      workers: 2
      resync: 30s
      dependsOn:
        - backend

      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites

      reconciler:
        default: true
        finalizers:
          - finalizer.demo.orkestra.io/website
        onCreate:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              port: "{{ .spec.port }}"
              reconcile: true
          services:
            - name: "{{ .metadata.name }}-svc"
              port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true

      queue:
        maxQueueDepth: 500

      description: Web application with Deployment and Service
```

---

## Example Komposer

```yaml
apiVersion: orkestra.konduktor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer
  description: Composes platform CRDs from multiple sources

sources:
  files:
    - ./katalogs/website.yaml
    - https://raw.github.com/myorg/crds/main/katalog.yaml
    - $SECURITY_KATALOG_URL

    - url: https://private.company.com/katalog.yaml
      auth:
        type: bearer
        fromEnv: PRIVATE_KATALOG_TOKEN

  helm:
    - repo: ./charts
      chart: platform-crds
      version: 0.1.0
      valueFiles:
        - ./values/production.yaml

spec:
  crds:
    - name: database
      workers: 4
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Database
        plural: databases
      reconciler:
        default: true
```