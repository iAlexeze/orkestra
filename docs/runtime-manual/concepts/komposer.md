# Komposer

A **Komposer** composes multiple Katalogs into one runtime. Where a Katalog
declares CRDs, a Komposer declares where to find them.

!!! tip "komposer is not a CRD"
    A komposer is not a CRD and [here is why](../../faqs/why-not-crds.md).

---

## A Simple Komposer?
```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer          # ← the distinction
metadata:
  name: platform-komposer
sources:
  files:
    - ./katalogs/website.yaml
    - ./katalogs/platform-namespace.yaml
  helm:
    - repo: ./charts
      chart: platform-crds
      version: 0.1.0
```

---

## Katalog vs Komposer

These are the only two valid document kinds in Orkestra. They have
completely different responsibilities and the distinction is enforced.

| | Katalog | Komposer |
|---|---|---|
| **Purpose** | Declare CRDs | Compose Katalogs |
| **Has `spec.crds`** | Yes | Only as inline overrides |
| **Has `sources`** | ❌ Error | ✅ Required |
| **Can source another Komposer** | — | ❌ Error |
| **Entry point for `ork run`** | Yes | Yes |

A Katalog with a `sources` block is an error:

```
"./katalog.yaml": kind Katalog cannot declare sources —
use kind: Komposer to compose multiple Katalogs
```

A Komposer sourcing another Komposer is an error:

```
"./komposer.yaml" sources.files["./other-komposer.yaml"]:
a Komposer cannot source another Komposer —
only Katalog files are valid sources
```

This keeps composition shallow and predictable. You always know exactly
what a source contains.

---

## Komposer structure

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: my-komposer
  description: Optional description

sources:
  files:
    - ./katalogs/project.yaml
    - https://raw.githubusercontent.com/myorg/crds/main/katalog.yaml
    - $REMOTE_KATALOG_URL

  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 1.2.0
      valueFiles:
        - ./values/production.yaml

# Optional — inline CRD overrides
# Merged last. Win on name conflict with any source.
spec:
  crds:
    website:
      workers: 4    # override the source definition for this environment
      ...
```

---

## File sources

File sources load Katalog YAML from local file paths or remote URLs.
Each file must be a valid Katalog (`kind: Katalog`). Files that are not
valid Katalog documents are silently skipped.

### Simple form

No authentication required. Works for local paths, public URLs, and
environment variable references.

```yaml
sources:
  files:
    - ./katalogs/project.yaml              # local relative path
    - /absolute/path/to/namespace.yaml     # local absolute path
    - https://raw.github.com/.../app.yaml  # public remote URL
    - $REMOTE_KATALOG_URL                  # environment variable
```

### Environment variables

Any entry starting with `$` is resolved via `os.Getenv` at startup.
The resolved value must be a valid file path or URL.

```yaml
sources:
  files:
    - $PLATFORM_KATALOG
    - $APP_KATALOG_URL
```

If the variable is not set or empty, Orkestra fails with a clear error
naming the variable.

---

## Authenticated file sources

Private Katalog files behind authentication use the struct form with an
`auth` block. Credentials are **always resolved from environment variables**
at startup — they never appear as literal values in the Komposer YAML.

```yaml
sources:
  files:
    # Simple form — no auth (existing behaviour unchanged)
    - ./katalogs/public.yaml

    # Authenticated form — private source
    - url: https://private.myorg.io/crds/platform-katalog.yaml
      auth:
        type: bearer
        fromEnv: PLATFORM_KATALOG_TOKEN
```

Both forms can be mixed freely in the same `sources.files` list.

### Bearer token

Use for any API that accepts `Authorization: Bearer <token>`.

```yaml
sources:
  files:
    - url: https://internal.company.com/crds/platform-katalog.yaml
      auth:
        type: bearer
        fromEnv: PLATFORM_KATALOG_TOKEN
```

Set the environment variable before running:

```bash
export PLATFORM_KATALOG_TOKEN=your-token-here
ork run --katalog komposer.yaml
```

In a Kubernetes deployment, inject it from a Secret:

```yaml
# Helm values
extraEnvFrom:
  - secretRef:
      name: orkestra-katalog-creds
```

```bash
kubectl create secret generic orkestra-katalog-creds \
  --namespace orkestra-system \
  --from-literal=PLATFORM_KATALOG_TOKEN=your-token-here
```

### GitHub token

Use for raw content from private GitHub repositories.
The token needs `repo` scope for private repository access.

```yaml
sources:
  files:
    - url: https://raw.githubusercontent.com/myorg/private-crds/main/katalog.yaml
      auth:
        type: github
        fromEnv: GITHUB_TOKEN
```

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
ork run --katalog komposer.yaml
```

`github` and `bearer` produce the same `Authorization: Bearer` header.
Use `github` when the token is a GitHub PAT — it makes the intent clear
to anyone reading the Komposer.

### Basic auth

Use for Artifactory, Nexus, and other corporate artifact stores.

```yaml
sources:
  files:
    - url: https://artifactory.company.com/orkestra/katalog.yaml
      auth:
        type: basic
        usernameFromEnv: ARTIFACTORY_USER
        passwordFromEnv: ARTIFACTORY_PASSWORD
```

```bash
export ARTIFACTORY_USER=svc-orkestra
export ARTIFACTORY_PASSWORD=xxxx
ork run --katalog komposer.yaml
```

### Mixing auth types

Multiple sources with different auth requirements in one Komposer:

```yaml
sources:
  files:
    # Public — no auth
    - https://raw.github.com/myorg/public-crds/main/katalog.yaml

    # Private GitHub repo
    - url: https://raw.githubusercontent.com/myorg/private-crds/main/katalog.yaml
      auth:
        type: github
        fromEnv: GITHUB_TOKEN

    # Internal API with bearer token
    - url: https://config.platform.myorg.io/crds/infra-katalog.yaml
      auth:
        type: bearer
        fromEnv: PLATFORM_KATALOG_TOKEN

    # Artifactory with basic auth
    - url: https://artifactory.company.com/orkestra/security-katalog.yaml
      auth:
        type: basic
        usernameFromEnv: ARTIFACTORY_USER
        passwordFromEnv: ARTIFACTORY_PASSWORD

    # Environment variable (resolved to a URL, then fetched)
    - $SECURITY_KATALOG_URL
```

### Injecting credentials in Kubernetes

Never put credential values in `values.yaml` or ConfigMaps committed to
source control. Always use Kubernetes Secrets.

```bash
kubectl create secret generic orkestra-katalog-creds \
  --namespace orkestra-system \
  --from-literal=GITHUB_TOKEN=ghp_xxxx \
  --from-literal=PLATFORM_KATALOG_TOKEN=bearer_yyyy \
  --from-literal=ARTIFACTORY_USER=svc-orkestra \
  --from-literal=ARTIFACTORY_PASSWORD=zzzz
```

```yaml
# Helm values
extraEnvFrom:
  - secretRef:
      name: orkestra-katalog-creds
```

All variables in the Secret become environment variables in the Orkestra
container at startup, where they are read by `auth.fromEnv`.

---

## Helm sources

Helm sources render a Helm chart and extract Katalog CRD definitions
from the rendered output. The chart must contain at least one template
with `kind: Katalog`.

```yaml
sources:
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 1.2.0                  # required — pin for reproducibility
      valueFiles:
        - ./values/production.yaml
        - https://config.io/vals.yaml
        - $ENVIRONMENT_VALUES
      values:
        environment: production       # inline overrides (like helm --set)
```

### Local charts

When `repo` is a file system path, Orkestra loads the chart directly.

```yaml
sources:
  helm:
    - repo: ./charts
      chart: platform-crds
      version: 0.1.0
      valueFiles:
        - ./values/local.yaml
```

### Git-based charts

When `repo` is a Git URL, Orkestra clones at `version` (branch, tag,
or commit hash) and loads the chart from `path` within the repo.

```yaml
sources:
  helm:
    - repo: https://github.com/myorg/helm-charts.git
      version: v1.2.0
      path: charts/platform-crds
      valueFiles:
        - ./values/production.yaml
```

### The Helm chart template

The chart renders a template with `kind: Katalog`. Orkestra extracts
CRD definitions from the rendered output.

```yaml
# charts/platform-crds/templates/katalog.yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: {{ .Release.Name }}-katalog
spec:
  crds:
    {{- range .Values.crds }}
    {{ .name }}:
      enabled: {{ .enabled | default true }}
      apiTypes:
        group: {{ $.Values.apiGroup }}
        version: {{ $.Values.apiVersion | default "v1alpha1" }}
        kind: {{ .kind }}
        plural: {{ .plural }}
      operatorBox:
        default: {{ .reconciler.default | default true }}
    {{- end }}
```

Version pinning is required for remote and Git sources. Omit it and
the same `ork run` may produce different behaviour as charts update.

---

## Inline overrides

CRDs declared in `spec.crds` on a Komposer are merged last and win on
name conflict. Use this to override a source definition for a specific
environment without forking the source.

```yaml
sources:
  files:
    - https://platform.internal/crds.yaml  # defines 'website' with workers: 2

spec:
  crds:
    # Override — production needs more workers
    website:
      workers: 4
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites
      operatorBox:
        default: true
```

---

## Merge rules

**Sources load in declaration order** — files first in `sources.files`
order, then helm in `sources.helm` order.

**Duplicate names across sources are errors:**

```
merger: duplicate CRD "website": defined in
  "file:./katalogs/website.yaml" and
  "url:https://remote/app.yaml"
```

**Inline `spec.crds` override silently** — intentional. This is the
override mechanism.

**Inline self-duplicates are errors:**

```
"./komposer.yaml" spec.crds: duplicate CRD "website" — each name must be unique
```

**Disabled CRDs survive the merge.** Disable a source CRD inline:

```yaml
spec:
  crds:
    legacy-resource:
      enabled: false
```

---

## CLI commands

All CLI commands that accept `--katalog` also accept a Komposer path.
The merger handles both transparently.

### `ork validate`

```bash
ork validate --katalog ./komposer.yaml
ork validate --katalog ./infra.yaml --katalog ./apps.yaml
ork validate --katalog $KATALOG_PATH
```

### `ork template`

```bash
ork template --katalog ./komposer.yaml
ork template --katalog ./komposer.yaml --graph
ork template --katalog ./komposer.yaml --json
```

### `ork generate runtime`

```bash
ork generate runtime --katalog ./komposer.yaml
ork generate runtime --katalog ./komposer.yaml --dry-run
```

### `ork run`

```bash
ork run --katalog ./komposer.yaml
ork run --katalog ./infra.yaml --katalog ./apps.yaml
```

---

## Complete examples

### Single Katalog — no Komposer needed

Most operators need only a Katalog. A Komposer is only needed when you
are composing definitions from multiple sources.

```yaml
# katalog.yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: website-katalog
spec:
  crds:
    website:
      enabled: true
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites
      operatorBox:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true
```

```bash
ork run --katalog ./katalog.yaml
```

### Multi-team composition

```yaml
# komposer.yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: platform-komposer

sources:
  files:
    - ./katalogs/website.yaml
    - ./katalogs/platform-namespace.yaml
    - https://raw.github.com/myorg/app-crds/main/katalog.yaml
    - $SECURITY_KATALOG_URL

spec:
  crds:
    application:
      workers: 4
      apiTypes:
        group: platform.myorg.io
        version: v1alpha1
        kind: Application
        plural: applications
        location: github.com/myorg/apis/application/v1alpha1
      operatorBox:
        default: true
        hooks:
          location: github.com/myorg/hooks
          function: ApplicationHooks
```

```bash
ork template --katalog ./komposer.yaml --graph
ork validate --katalog ./komposer.yaml
ork run --katalog ./komposer.yaml
```

### Enterprise — private sources with mixed auth

```yaml
# enterprise-komposer.yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: enterprise-komposer

sources:
  files:
    # Public shared baseline
    - https://raw.github.com/myorg/public-baseline/main/katalog.yaml

    # Private GitHub repo — platform CRDs
    - url: https://raw.githubusercontent.com/myorg/platform-crds/main/katalog.yaml
      auth:
        type: github
        fromEnv: GITHUB_TOKEN

    # Internal API — security team CRDs
    - url: https://api.security.myorg.io/orkestra/katalog.yaml
      auth:
        type: bearer
        fromEnv: SECURITY_API_TOKEN

    # Artifactory — compliance CRDs
    - url: https://artifactory.myorg.io/orkestra/compliance-katalog.yaml
      auth:
        type: basic
        usernameFromEnv: ARTIFACTORY_USER
        passwordFromEnv: ARTIFACTORY_PASSWORD

  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 3.2.1
      valueFiles:
        - ./values/enterprise-production.yaml

spec:
  crds:
    # Production environment override — more workers than the shared default
    application:
      workers: 10
      resync: 30s
      apiTypes:
        group: platform.myorg.io
        version: v1alpha1
        kind: Application
        plural: applications
      operatorBox:
        default: true
```

```bash
# All credentials injected from a Kubernetes Secret
kubectl create secret generic orkestra-katalog-creds \
  --namespace orkestra-system \
  --from-literal=GITHUB_TOKEN=ghp_xxxx \
  --from-literal=SECURITY_API_TOKEN=bearer_yyyy \
  --from-literal=ARTIFACTORY_USER=svc-orkestra \
  --from-literal=ARTIFACTORY_PASSWORD=zzzz

# Validate before deploying
ork validate --katalog enterprise-komposer.yaml
ork template --katalog enterprise-komposer.yaml --graph
```

### Helm chart + local Katalog + inline override

```yaml
# production-komposer.yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: production-komposer

sources:
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 2.1.0
      valueFiles:
        - ./values/base.yaml
        - ./values/production.yaml

  files:
    - https://shared.platform.io/common-crds.yaml

spec:
  crds:
    database:
      workers: 2
      resync: 5m
      apiTypes:
        group: platform.myorg.io
        version: v1alpha1
        kind: Database
        plural: databases
      operatorBox:
        default: false
        constructor:
          location: github.com/myorg/reconcilers
          function: NewDatabaseReconciler
```

---

## Adding new source types

Sources are implemented as functions on the Merger in `pkg/merger/`.
Adding a new source type is three steps.

**1. Add to `KatalogSources` in `pkg/types`:**

```go
type KatalogSources struct {
    Files []FileSource `yaml:"files,omitempty"`
    Helm  []HelmSource `yaml:"helm,omitempty"`
    S3    []S3Source   `yaml:"s3,omitempty"`   // ← new
}
```

**2. Add `loadS3Source` in `pkg/merger/`:**

```go
func (m *Merger) loadS3Source(src orktypes.S3Source) ([]orktypes.CRDEntry, error) {
    // fetch from S3, parse with parseKatalogDoc, return []CRDEntry
}
```

**3. Call it from `loadKomposer` in `pkg/merger/file.go`:**

```go
for i, s3Src := range doc.Sources.S3 {
    crds, err := m.loadS3Source(s3Src)
    if err != nil {
        return nil, fmt.Errorf("%q sources.s3[%d]: %w", path, i, err)
    }
    // dedup and append
}
```

**Current implementations:**

| Source | Declaration | Auth supported | Status |
|---|---|---|---|
| Local file | `sources.files: - ./path` | No (local) | ✅ |
| Remote URL | `sources.files: - https://...` | No | ✅ |
| Remote URL + bearer | `sources.files: - url: ... auth: type: bearer` | Yes | ✅ |
| Remote URL + GitHub token | `sources.files: - url: ... auth: type: github` | Yes | ✅ |
| Remote URL + basic auth | `sources.files: - url: ... auth: type: basic` | Yes | ✅ |
| Environment variable | `sources.files: - $VAR` | No | ✅ |
| Remote Helm repository | `sources.helm.repo: https://...` | No | ✅ |
| Local Helm chart | `sources.helm.repo: ./charts` | No | ✅ |
| Git-based Helm chart | `sources.helm.repo: https://...git` | No | ✅ |
| Inline `spec.crds` | `spec.crds: [...]` | No | ✅ |
| S3 / GCS / Azure Blob | `sources.s3` | — | Planned |
| Kubernetes ConfigMap | `sources.configMap` | — | Planned |


---

> For a complete list of all configurable options, see the **[Katalog and Komposer Reference](../../reference/katalog-komposer-reference.md)**.  
> For real‑world examples, see the **[Use Cases](../../use-cases/index.md)** documentation.