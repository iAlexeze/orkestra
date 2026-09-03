# 03 — Imports

Imports are only valid on `kind: Komposer` documents. They are resolved in a fixed order:

```
1. imports.registry  (loadRegistrySource)
2. imports.files     (loadImportFileWithAuth)
3. imports.helm      (loadHelmSource)
4. spec.crds         (inline — merged last, wins on conflict)
```

## File imports

```yaml
imports:
  files:
    - ./katalogs/local.yaml              # local path (no auth)
    - https://public.url/katalog.yaml    # remote URL (no auth)
    - $MY_KATALOG_URL                    # environment variable reference

    # Authenticated form
    - url: https://private.url/katalog.yaml
      auth:
        type: bearer
        fromEnv: MY_TOKEN
```

- Environment variable references (`$VAR`) are resolved via `resolveEnvVar`.
- Auth credentials are resolved from environment variables by `fileSrc.Auth.Resolve()`.
- The loaded file must be `kind: Katalog` — a Komposer cannot import another Komposer.

## Helm imports

```yaml
imports:
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 1.2.0
      valueFiles:
        - ./values/prod.yaml
      values:
        region: us-east-1
```

The Helm chart is rendered with `helm template`. The merger then scans the rendered output for templates with `kind: Katalog` and loads them.

## Registry imports

```yaml
imports:
  registry:
    - url: $ORK_REGISTRY
```

Fetches CRD definitions from the Orkestra registry. The registry URL can be set via an environment variable or the `--registry` CLI flag.

## Adding a new import type

1. Add a field to `KatalogSources` in `pkg/types/katalog.go`.
2. Implement a `loadXxxSource(src XxxSource) (map[string]CRDEntry, error)` function in a new `pkg/merger/xxx.go` file.
3. Add a step in `loadKomposer` in `file.go` following the existing pattern (loop, dedup, accumulate top-level fields).
4. Update this document and `README.md`.
