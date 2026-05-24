# Komposers

A Komposer is a platform declaration that composes multiple Katalogs into a single runtime. It can be published to the registry so platform teams can distribute a complete control plane as one versioned artifact.

## Publishing a Komposer pattern

A Komposer pattern directory follows the same shape as a Katalog:

```text
platform/
  komposer.yaml   # required — the Komposer declaration
  e2e.yaml        # optional — E2E expectations
  README.md       # optional
```

```bash
ork registry push platform:v2 ./patterns/platform/
```

The registry detects `komposer.yaml` as the primary file and records the pattern kind accordingly.

## Importing a Komposer pattern

Consumers use `useKomposer: true` to tell Orkestra to load `komposer.yaml` from the artifact instead of `katalog.yaml`:

```yaml
imports:
  registry:
    - url: ghcr.io/myorg/patterns/platform:v2
      oci: true
      useKomposer: true
```

This is how a platform team distributes a complete, pre-composed control plane: one reference, one version, one runtime.

## Overriding on import

The consuming Komposer's inline `spec.crds` always wins. Environment-specific overrides require no changes to the upstream pattern:

```yaml
imports:
  registry:
    - url: platform:v2
      useKomposer: true
spec:
  crds:
    database:
      workers: 10    # production override
```
