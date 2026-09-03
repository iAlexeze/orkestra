# Imports

`imports` is the block that makes a Komposer different from a Katalog. It declares one or more Katalog sources. All three source types can be combined in a single Komposer.

---

## `imports.files`

Load Katalogs from local paths, HTTPS URLs, or environment variable references.

```yaml
imports:
  files:
    # Local path — relative to the Komposer file
    - ./katalogs/website.yaml

    # HTTPS URL
    - https://raw.githubusercontent.com/myorg/repo/main/katalog.yaml

    # Environment variable reference
    - $MY_KATALOG_URL

    # Authenticated remote file
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

---

## `imports.helm`

Render a Helm chart and extract all `kind: Katalog` documents from the output.

```yaml
imports:
  helm:
    - repo: https://github.com/orkspace/charts.git
      chart: orkestra-katalog-example
      path: katalog-example
      version: v0.1.0
      valueFiles:
        - ./values/prod.yaml
      values:
        region: us-east-1
      # auth:
      #   type: bearer
      #   fromEnv: HELM_REPO_TOKEN
```

| Field | Required | Description |
|-------|----------|-------------|
| `repo` | yes | Helm chart repository URL. |
| `chart` | yes | Chart name within the repo. |
| `path` | no | Path inside the chart archive to the katalog file, if not at chart root. |
| `version` | yes | Chart version. Omit to use latest or a branch name. |
| `valueFiles` | no | Values files (local paths or remote URLs), applied in order. |
| `values` | no | Inline key-value overrides applied last. |
| `auth` | no | Repository credentials. |

Orkestra renders with `helm template` and scans the output for `kind: Katalog` documents. Non-Katalog resources in the chart output are ignored.

---

## `imports.registry`

Pull versioned Katalog patterns from the Orkestra Registry or any OCI-compatible registry.

```yaml
imports:
  registry:
    # OCI artifact — versioned and immutable
    - url: ghcr.io/orkspace/patterns/postgres@v1.0.0
      oci: true
      # auth:
      #   type: bearer
      #   fromEnv: ORK_REGISTRY_TOKEN

    # Orkestra Registry with branch-pinned patterns
    - url: $ORK_REGISTRY
      katalog:
        postgres:
          branch: main
        redis:
          branch: v7

    # Pull the upstream komposer.yaml instead of katalog.yaml
    - url: ghcr.io/myorg/patterns/platform@v2.0.0
      oci: true
      useKomposer: true
```

| Field | Description |
|-------|-------------|
| `url` | Registry URL, OCI reference, or `$ENV_VAR`. Also set via `ORK_REGISTRY` env var or `--registry` flag. |
| `oci` | `true` for OCI artifact references (`ghcr.io/...`). |
| `katalog` | Map of pattern name → options when using the Orkestra Registry URL form. |
| `katalog.<name>.branch` | Branch or tag to pull. Defaults to `main`. |
| `useKomposer` | Load the upstream `komposer.yaml` from the pattern instead of `katalog.yaml`. |
| `auth` | Registry credentials. |

### Pinning versions

Always pin OCI imports via `@version` for production:

```yaml
# Good — immutable
- url: ghcr.io/orkspace/patterns/postgres@v1.0.0
  oci: true

# Avoid in production — resolves to latest at run time
- url: ghcr.io/orkspace/patterns/postgres
  oci: true
```

---

## Combining all three sources

```yaml
imports:
  registry:
    - url: ghcr.io/orkspace/patterns/postgres@v1.0.0
      oci: true

  files:
    - ./website-katalog.yaml

  helm:
    - repo: https://github.com/orkspace/charts.git
      chart: orkestra-katalog-example
      version: v0.1.0
```

Resolution order is `registry → files → helm → spec.crds`. If a CRD name appears in two sources, that is a hard error. Use `spec.crds` for overrides, not re-definitions.

---

→ Next: [02-overrides.md](02-overrides.md) — `spec.crds` overrides and resolution rules  
→ Back: [Komposer index](index.md)
