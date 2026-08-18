# 09 — Deprecation

Mark a pattern as deprecated, signal a migration path, and set a timeline. This is the producer side — the next step ([13-deprecation-accept](../13-deprecation-accept/README.md)) covers the consumer side.

## Files

| File | Purpose |
|------|---------|
| `katalog.yaml` | v1.0.0 with `lifecycle.deprecation:` block and timeline |
| `crd-v1.yaml` | WebApp v1 CRD |

---

> **Before you start:** If `ORK_REGISTRY` is not set, export it now (see [01-motifs](../01-motifs/README.md#push-to-the-registry)). Replace `myorg` with your actual registry path.

---

## The lifecycle block

`lifecycle:` is a top-level field alongside `metadata:` and `spec:`. It carries policy for tooling — the runtime ignores it.

```yaml
lifecycle:
  maturity: deprecated
  deprecation:
    migratedTo: ghcr.io/myorg/katalogs/webapp@v2.0.0
    message: "v1.0.0 is end-of-life. Migrate to v2.0.0: add spec.healthPath to your CRs."
    timeline:
      from: "2025-01-01"    # warn from this date
      to: "2026-06-01"      # EOL on this date
```

`ork validate` enforces that `maturity: deprecated` requires a `deprecation:` block. Without one, validation fails.

## Publish the deprecation

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork validate
ork push .
ork patterns   # deprecated patterns are prefixed with ⚠
```

## What consumers see

```bash
ork inspect webapp-operator:v1.0.0
# ⚠ This pattern is deprecated.
#   Migrate to: ghcr.io/myorg/katalogs/webapp@v2.0.0
#   Message:    v1.0.0 is end-of-life. Migrate to v2.0.0: add spec.healthPath.
#   EOL:        2026-06-01
```

`ork validate` on a Komposer that imports this pattern surfaces the same warning — before any CR is applied.

## Next step

→ [13-deprecation-accept](../13-deprecation-accept/README.md) — accept running a deprecated pattern as a consumer
