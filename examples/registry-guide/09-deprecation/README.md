# 09 — Deprecation

Mark a pattern as deprecated, surface warnings to consumers, and publish a
replacement that consumers can migrate to.

## Files

| File | Purpose |
|------|---------|
| `katalog.yaml` | v1.0.0 with `deprecation:` block set |
| `crd-v1.yaml` | WebApp v1 CRD (copied from 02-katalog-api) |

---

> **Before you start:** If `ORK_REGISTRY` is not set, export it now (see [01-motifs](../01-motifs/README.md#push-to-the-registry)). Replace `myorg` with your actual registry path in [katalog.yaml](katalog.yaml) and throughout this example.

---

## The deprecation block

Add `deprecation:` under `metadata:` to mark a pattern end-of-life:

```yaml
metadata:
  deprecation:
    migratedTo: ghcr.io/myorg/katalogs/webapp-operator:v2.0.0
    message: "v1.0.0 is end-of-life. Migrate to v2.0.0: add spec.healthPath."
```

This is written as OCI annotations when the pattern is pushed. Consumers see it in
three places:

- `ork inspect` — yellow warning block before the name/version line
- `ork patterns` — deprecated patterns prefixed with ⚠
- `ork validate` — warning on stderr when a Komposer pulls a deprecated pattern

## Publish the deprecation

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
ork patterns   # deprecated patterns are prefixed with ⚠
```

Re-pushing the same version tag with the deprecation annotation is enough —
consumers pulling `@v1.0.0` will see the warning immediately without any
changes to their Komposer.

## What consumers see

```bash
ork validate
# webapp-operator  This pattern ⚠ is deprecated.
#   Migrate to:  ghcr.io/ialexeze/katalogs/webapp@v2.0.0
#   Message:     v1.0.0 is end-of-life. Migrate to v2.0.0: add spec.healthPath to your CRs.
```

## Migration path

1. Consumers update their Komposer to reference `@v2.0.0`
2. Add `spec.healthPath` to any CRs that need a non-default probe path
3. Run `ork validate` — the deprecation warning disappears

## Next step

→ [10-hooks-katalog/README.md](../10-hooks-katalog/README.md) — typed Go operator: hooks, generate registry, build, publish
