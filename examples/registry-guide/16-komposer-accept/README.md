# 16 — Komposer Accept: Mixed Lifecycle Concerns

Step 13 covered accepting a deprecated pattern. This step covers the case where your imports have different lifecycle concerns — one is deprecated, another is alpha. The mechanism is the same; the error messages differ.

## Files

| File | Purpose |
|------|---------|
| `katalog-webapp.yaml` | Deprecated Katalog — being replaced by v2.0.0 |
| `katalog-cache.yaml` | Alpha Katalog — API is settling |
| `komposer.yaml` | Komposer that imports both |
| `crd.yaml` | WebApp CRD |
| `crd-cache.yaml` | Cache CRD |

---

## What you see without acceptance

```text
✗  END OF LIFE
  This pattern reached end of life on 2026-06-01.
  v1.0.0 is end-of-life. Migrate to v2.0.0: add spec.healthPath to your CRs.
  Migrate to:  ghcr.io/myorg/katalogs/webapp@v2.0.0
  To acknowledge this import, add it to lifecycle.accept.patterns in your Komposer.

⚠  MATURITY WARNING
  cache-operator is alpha — experimental, breaking changes expected.
  To acknowledge this import, add it to lifecycle.accept.patterns in your Komposer.
```

Two different concerns, one place to acknowledge them:

```yaml
lifecycle:
  accept:
    patterns:
      - name: webapp-operator
      - name: cache-operator
```

```bash
ork validate -f komposer.yaml
```

---

## Next step

→ Back to [Registry Guide index](../README.md)
