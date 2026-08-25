# Lifecycle

The `lifecycle:` block is a policy annotation on any Katalog or Komposer. `lifecycle.maturity` is advisory — tooling (`ork validate`, `ork inspect`, the registry index) surfaces it as a signal. The `deprecation:` block within `lifecycle:` is enforced: `ork validate` warns, and `ork run` blocks startup if a deprecated pattern has not been accepted. The Gateway enforces the same gate on apply.

---

## Maturity

`lifecycle.maturity` signals how stable a pattern's API surface is:

| Value | Meaning |
|-------|---------|
| `alpha` | Experimental — breaking changes expected between versions |
| `beta` | Stabilising — API is mostly settled, minor breakage possible |
| `stable` | Production-ready — semantic versioning applies |
| `deprecated` | Replaced — requires `lifecycle.deprecation` |

```yaml
lifecycle:
  maturity: alpha
```

`ork validate` prints a non-fatal warning for `alpha` and `beta`. `stable` and an absent `maturity` produce no output. `deprecated` without a `lifecycle.deprecation` block is an error.

---

## Deprecation

Patterns in the registry are immutable at a version — `v1.0.0` cannot be modified after it is pushed. When a pattern is superseded, you deprecate the old version rather than delete it. Consumers on the old version see a warning and a migration path.

### Marking a pattern as deprecated

The `lifecycle.deprecation` block is the primary signal — `maturity: deprecated` is optional when the block is present. Setting both is explicit but redundant:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: webapp-operator
  version: v1.0.0

lifecycle:
  maturity: deprecated
  deprecation:
    migratedTo: ghcr.io/myorg/katalogs/webapp@v2.0.0
    message: "v1.0.0 is end-of-life. Migrate to v2.0.0: add spec.healthPath to your CRs."
    timeline:
      from: "2025-01-01"
      to: "2026-06-01"
```

Push the deprecation:

```bash
ork push webapp-operator:v1.0.0 ./deprecated/
```

`ork push` is always a new upload — the previous artifact is not modified.

### What consumers see

```bash
ork inspect webapp-operator:v1.0.0
```

```text
⚠  This pattern is deprecated.
  Migrate to:  ghcr.io/myorg/katalogs/webapp@v2.0.0
  Note:        v1.0.0 is end-of-life. Migrate to v2.0.0: add spec.healthPath to your CRs
  EOL:         2026-06-01
```

Komposers that import a deprecated Katalog surface the warning at `ork validate` and `ork simulate` time — before any CR is applied.

### The state machine

```text
v1.0.0  ← active
          │
          │  v2.0.0 ships (migration target)
          ▼
v1.0.0  ← ⚠ deprecated ("migrate to v2.0.0")
v2.0.0  ← active
          │
          │  migration complete — all consumers on v2.0.0
          ▼
v1.0.0  ← retired (removed from index, not deleted)
v2.0.0  ← active
```

Retirement removes the pattern from `ork patterns` output but does not delete the OCI artifact. Consumers that pinned a digest can still pull. Consumers that use version tags get a "not found in index" message.

### The timeline block

`timeline.from` and `timeline.to` are advisory — they document the deprecation window so consumers can plan. `ork validate` surfaces them as part of the deprecation warning.

```yaml
lifecycle:
  deprecation:
    timeline:
      from: "2025-01-01"   # when deprecation was announced
      to: "2026-06-01"     # end-of-life: pattern will be retired after this date
```

### Best practices

**Write the migration path in the message.** "Use v2.0.0" is not a migration path. "Add `spec.healthPath: /health` to your CRs — it was implicit in v1.0.0 but is required in v2.0.0" is.

**Give consumers time.** Publish the new version and deprecate the old one simultaneously — don't retire `v1.0.0` the day `v2.0.0` ships. Platform teams may have change-freeze windows.

**Check for active importers before retiring.** `ork patterns` shows which patterns are imported by Komposers in the official registry. For internal registries, check Komposers in your platform repository before retiring.

---

## Compatibility

`lifecycle.compatibility` declares the Kubernetes and Orkestra versions the pattern has been verified against. Both fields accept [Masterminds/semver](https://github.com/Masterminds/semver) range syntax — the same library Helm uses.

```yaml
lifecycle:
  maturity: stable
  compatibility:
    kubernetes: ">=1.28"       # verified against Kubernetes 1.28 and above
    orkestra: ">=0.7.0"        # verified against Orkestra 0.7.0 and above
```

`ork validate` checks that the range string is valid semver syntax.

Common range forms:

```text
>=1.28            at least 1.28
>=1.28, <1.33     between 1.28 and 1.33 exclusive
^1.28             >=1.28, <2.0
~1.28             >=1.28, <1.29
```

---

## Accept — Komposer level

A Komposer that imports Katalogs with lifecycle concerns (deprecated, alpha, or beta) uses `lifecycle.accept.patterns` to acknowledge each import by name:

```yaml
lifecycle:
  accept:
    patterns:
      - name: webapp-operator   # accepts all lifecycle concerns for this import
        author: myorg           # optional — disambiguates if the name is not unique
      - name: cache-operator
        author: myorg
```

Each entry covers the full lifecycle state of that Katalog — deprecated, pre-stable maturity, or both. Without an entry, `ork validate` warns for every unacknowledged import.

`lifecycle.accept.patterns` is only valid on Komposers. Declaring it on a Katalog is an error.

### Scoping an acceptance to a specific version

`version` is an optional semver range that scopes the acceptance to a specific imported version. When set, `ork validate` warns if the imported version no longer matches — keeping the Komposer tidy as imports graduate:

```yaml
lifecycle:
  accept:
    patterns:
      - name: webapp-operator
        author: myorg
        version: "=1.0.0"   # only accepts this specific deprecated version
```

Without `version`, the entry accepts any version of that pattern (current default behaviour).

### The author field

`author` is optional. When set, an entry only matches imports whose `metadata.author` equals that value. Use it when two registries publish a Katalog with the same name:

```yaml
patterns:
  - name: webapp-operator
    author: myorg       # only the myorg version, not a same-named pattern from another publisher
```

When `author` is omitted, any publisher's pattern with that name is accepted.

---

## Platform policy

`policy:` is a platform-tier top-level block distinct from `lifecycle:`. Where `lifecycle:` is about what a pattern signals, `policy:` is about what a Komposer allows.

```yaml
policy:
  lifecycle:
    minMaturity: beta   # alpha imports are errors, not warnings
```

`policy.lifecycle.minMaturity` sets a floor: any import whose maturity falls below the declared level is an error at `ork validate` time instead of a warning. Valid floors are `alpha`, `beta`, and `stable`. (`deprecated` is not a valid floor — deprecated imports require explicit `lifecycle.accept.patterns` regardless of any floor.)

`policy:` is structured as `policy.<area>.*` so other policy categories — security, registry, user-defined — can grow alongside `lifecycle:` without flattening into one block.

---

## Full schema

```yaml
lifecycle:
  maturity: alpha | beta | stable | deprecated   # optional; deprecation block alone implies deprecated
  deprecation:                                    # presence implies deprecated; block alone is sufficient
    migratedTo: <oci-ref>                         # where to migrate
    message: "<human-readable migration note>"    # required
    timeline:
      from: "YYYY-MM-DD"                          # when deprecation was announced
      to: "YYYY-MM-DD"                            # EOL date
  compatibility:                                  # optional
    kubernetes: "<semver-range>"
    orkestra: "<semver-range>"
  accept:                                         # Komposer only
    patterns:
      - name: <katalog-name>
        author: <author>                          # optional — disambiguates when name is not unique
        version: "<semver-range>"                 # optional — scope acceptance to a specific version

policy:                                           # optional; Komposer-level platform enforcement
  lifecycle:
    minMaturity: alpha | beta | stable            # imports below this floor are errors (not warnings)
```

---

## Try it

```bash
ork init --pack registry-guide
cd 09-deprecation          # mark a pattern deprecated and migrate consumers
cd 13-deprecation-accept   # katalog-level and komposer-level acceptance
cd 14-lifecycle-maturity   # alpha, beta, stable in one directory
cd 15-lifecycle-compatibility  # kubernetes and orkestra ranges
cd 16-komposer-accept      # komposer accepting deprecated + alpha imports
```
