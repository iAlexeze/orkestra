# Komposer

A Komposer is a Katalog with an `imports` block. The separate `kind` is intentional — it keeps composition explicit and prevents collapsing imported operator specs into a single undifferentiated Katalog.

A Komposer cannot import another Komposer — only `kind: Katalog` files are valid import targets.

## Wire format

```yaml
apiVersion: orkestra.orkspace.io/v1   # required
kind: Komposer                         # required

metadata:
  name: platform                       # required

imports:
  files:
    - ...
  helm:
    - ...
  registry:
    - ...

# Optional inline overrides — merged last, win on name conflict
spec:
  crds:
    <name>:
      ...

# Same top-level fields as Katalog
security:
  ...
notification:
  ...
providers:
  - ...
```

## `imports.files`

```yaml
imports:
  files:
    - ./katalogs/local.yaml
    - https://raw.githubusercontent.com/org/repo/main/katalog.yaml
    - $MY_KATALOG_URL

    # Authenticated
    - url: https://private.example.com/katalog.yaml
      auth:
        type: bearer
        fromEnv: MY_TOKEN
```

| Field | Description |
|-------|-------------|
| `url` | Local path, HTTPS URL, or `$ENV_VAR` reference. |
| `auth.type` | `bearer`, `basic`, or `github`. |
| `auth.fromEnv` | Environment variable holding the token or password. |

## `imports.helm`

```yaml
imports:
  helm:
    - repo: https://charts.example.com
      chart: platform-crds
      version: 1.2.0
      valueFiles:
        - ./values/prod.yaml
      values:
        region: us-east-1
```

| Field | Required | Description |
|-------|----------|-------------|
| `repo` | yes | Helm chart repository URL. |
| `chart` | yes | Chart name within the repo. |
| `version` | yes | Chart version. Pin this for reproducibility. |
| `valueFiles` | no | Values files (local paths or remote URLs). |
| `values` | no | Inline key-value overrides. |

Orkestra renders with `helm template` and scans for `kind: Katalog` documents in the output.

## `imports.registry`

```yaml
imports:
  registry:
    - url: $ORK_REGISTRY
      katalog:
        postgres:
          branch: main
        redis:
          branch: v7

    - url: oci://ghcr.io/myorg/patterns/postgres:v14
      oci: true
```

| Field | Description |
|-------|-------------|
| `url` | Registry URL or `$ENV_VAR`. Also set via `ORK_REGISTRY` env var or `--registry` flag. |
| `oci` | `true` for OCI artifact references. |
| `useKomposer` | Load the upstream `komposer.yaml` instead of `katalog.yaml`. |

## Resolution order

```
1. imports.registry
2. imports.files
3. imports.helm
4. spec.crds   (inline — wins on name conflict)
```

CRD names must be unique across all imports. A duplicate name is a hard error.

## Top-level field accumulation

| Field | Merge strategy |
|-------|----------------|
| `security` | Komposer's setting wins on conflict. |
| `notification.teams` | Union of all imports; Komposer wins on duplicate team name. |
| `providers` | Union; deduplicated by name. |

---
