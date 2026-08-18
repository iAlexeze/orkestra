# 14 — Lifecycle Maturity

Signal how stable a pattern is using `lifecycle.maturity`. This tells consumers — and the registry — whether a pattern is experimental, stabilising, or production-ready. Maturity alone is advisory; the `deprecation:` block within `lifecycle:` is enforced at both validate time and runtime startup — a deprecated pattern without acceptance blocks `ork run`.

## Files

| File | Purpose |
|------|---------|
| `katalog-alpha.yaml` | v0.1.0 — `maturity: alpha`, experimental |
| `katalog-beta.yaml` | v0.5.0 — `maturity: beta`, stabilising |
| `katalog-stable.yaml` | v1.0.0 — `maturity: stable`, production-ready |
| `crd.yaml` | WebApp CRD |

---

## The four maturity levels

```yaml
lifecycle:
  maturity: alpha    # experimental — breaking changes expected
```

```yaml
lifecycle:
  maturity: beta     # stabilising — API mostly settled
```

```yaml
lifecycle:
  maturity: stable   # production-ready — semantic versioning applies
```

```yaml
lifecycle:
  maturity: deprecated  # replaced — requires lifecycle.deprecation (see 09-deprecation)
```

When `maturity:` is omitted, `ork validate` treats the pattern as `stable` if `metadata.version >= 1.0.0`, otherwise `beta`.

---

## Validate all three

```bash
# alpha — passes, prints a non-fatal warning
ork validate -f katalog-alpha.yaml

# beta — passes, prints a lower-severity warning
ork validate -f katalog-beta.yaml

# stable — passes silently
ork validate -f katalog-stable.yaml
```

Alpha and beta validations pass — warnings are informational. The registry surfaces them as badges on `ork patterns` and `ork inspect` so consumers can see stability at a glance without running validate themselves.

---

## Graduating a pattern

Push each version with its maturity signal as the pattern stabilises:

```bash
# publish alpha
ork push ./katalog-alpha.yaml

# iterate — when stable, publish with updated maturity
ork push ./katalog-stable.yaml
```

Consumers who pinned `@v0.1.0` continue to see the alpha signal. Consumers who upgrade to `@v1.0.0` see stable. The registry does not retroactively change annotations on published versions.

---

## Next step

→ [15-lifecycle-compatibility](../15-lifecycle-compatibility/README.md) — declare which Kubernetes and Orkestra versions a pattern has been verified against
