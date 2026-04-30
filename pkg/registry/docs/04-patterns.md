# 04 — Patterns and the Artifact Format

## Pattern directory layout

```
my-operator/
  katalog.yaml   ← operator declaration         (required)
  crd.yaml       ← CRD schema                   (required)
  README.md      ← human documentation          (optional)
  cr.yaml        ← example CR for the CRD       (optional)
```

`ValidateDirectory` enforces the required files and derives `PatternMeta` from the contents of `katalog.yaml`.

## Metadata derivation

Metadata is derived solely from `katalog.yaml`. The derivation runs in `ValidateDirectory`:

1. Parse `katalog.yaml` — extract `metadata.name`, `metadata.version`, `metadata.description`, `metadata.author`, `metadata.tags`, and required providers from `spec.providers`.
2. Validate: `name`, `version`, and `description` must all be non‑empty.  
   (`metadata.name` is already required by Katalog validation; `version` and `description` default to `latest` and a placeholder if absent.)

```go
meta, files, err := registry.ValidateDirectory("./my-operator/")
// meta.Name, meta.Version, meta.Description, meta.Tags, meta.Author
// files = ["katalog.yaml", "crd.yaml", "README.md", "cr.yaml"]  (present files in canonical order)
```

## Validation pipeline (on push)

Three layers of validation run before any bytes are sent to the registry:

| Layer | What is checked |
|-------|----------------|
| `ValidateDirectory` | Required files present; metadata non‑empty after derivation |
| `merger.New().Merge()` | Katalog YAML parses correctly; sources resolve |
| `katalog.ValidateConfig` | Full semantic validation: field types, GVK uniqueness, dependency graph, reconciler modes |
| `validateCRDFile` | YAML parses; `kind: CustomResourceDefinition`; `spec.group` and `spec.names.kind` present |

An empty `katalog.yaml` or an invalid CRD structure fails fast before any network call.

## Adding a new file type to the artifact

1. Add a `FileXxx` constant to `pattern.go`.
2. Add a case to `mediaTypeForFile` in `client.go` with an appropriate MIME type.
3. Add the file to the `optional` slice in `ValidateDirectory` (or `required` if it must always be present).

The file will be pushed as a layer with its media type and the `org.opencontainers.image.title` annotation set to the filename. `oras.Copy` on the pull side uses this annotation to write the file to the correct path in the cache directory.

## Publishing to the official registry

```bash
# 1. Authenticate
docker login ghcr.io

# 2. Push (validation runs automatically)
ork registry push my-operator:v1.0.0 ./my-operator/

# 3. Verify
ork registry info my-operator:v1.0.0
ork registry list
```

The index at `ghcr.io/orkspace/orkestra-registry/index:latest` is updated automatically after each push.

For the official `ghcr.io/orkspace/orkestra-registry` registry, patterns are published by opening a PR against `github.com/orkspace/orkestra-registry`. CI validates the pattern against a live kind cluster and pushes on merge.
