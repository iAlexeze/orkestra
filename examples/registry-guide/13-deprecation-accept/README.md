# 13 — Accepting Deprecation

Step 12 covered the producer: marking a pattern deprecated and setting a timeline. This step covers the consumer: what happens when you import a deprecated pattern, and how to explicitly acknowledge it.

## Files

| File | Purpose |
|------|---------|
| `katalog.yaml` | Deprecated Katalog — the upstream pattern as published |
| `komposer.yaml` | Komposer that imports it; acceptance lives here |
| `crd-v1.yaml` | WebApp v1 CRD |

---

## The problem

`katalog.yaml` is deprecated. When your Komposer imports it, `ork validate` surfaces:

```text
✗  END OF LIFE
  This pattern reached end of life on 2026-06-01.
  v1.0.0 is end-of-life. Migrate to v2.0.0: add spec.healthPath to your CRs.
  Migrate to:  ghcr.io/myorg/katalogs/webapp@v2.0.0
  To acknowledge this import, add it to lifecycle.accept.patterns in your Komposer.
```

The pattern author declared the deprecation — they cannot accept it on your behalf. Acceptance is a consumer decision.

---

## The fix — accept at the Komposer level

Add the pattern name to `lifecycle.accept.patterns` on your [Komposer](./komposer.yaml):

```yaml
lifecycle:
  accept:
    patterns:
      - name: webapp-operator
```

That is the signal: you have read the deprecation, evaluated the migration path, and are choosing to continue using this version intentionally. `ork validate` passes and `ork run` starts.

```bash
ork validate -f komposer.yaml
```

---

## Scoping acceptance to a version range

If you are accepting a specific version window — not any future deprecated version of the same pattern — add a `version:` range:

```yaml
lifecycle:
  accept:
    patterns:
      - name: webapp-operator
        version: ">= 1.0.0, < 2.0.0"
```

Acceptance only applies when the imported version falls within that range. A newer deprecated version outside the range blocks again, forcing a conscious re-evaluation.

---

## Multiple deprecated imports

List each pattern by name. `ork validate` surfaces all unacknowledged names in one message so you can add them in a single pass:

```yaml
lifecycle:
  accept:
    patterns:
      - name: webapp-operator
      - name: old-data-store
```

---

## Next step

→ [14-lifecycle-maturity](../14-lifecycle-maturity/README.md) — signal pattern stability with alpha, beta, and stable maturity levels
