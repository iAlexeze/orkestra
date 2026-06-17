# Deprecation

Patterns in the registry are immutable at a version — `v1.0.0` cannot be modified after it is pushed. When a pattern is superseded, you deprecate the old version rather than delete it. Consumers on the old version see a warning and a migration path.

---

## Marking a pattern as deprecated

Add `deprecated: true` and a `deprecationMessage` to `metadata`:

```yaml
# motif.yaml — web-service v1.0.0 (deprecated)
apiVersion: orkestra.orkspace.io/v1
kind: Motif
metadata:
  name: web-service
  version: v1.0.0
  deprecated: true
  deprecationMessage: >
    Use web-service:v1.1.0 — adds configurable probe path and profile.
    Migration: add probeProfile and probePath to your Katalog's with: block.
    Both default to the v1.0.0 behavior.
```

Push the deprecation annotation:

```bash
ork push web-service:v1.0.0 ./deprecated/
```

This creates a new artifact at `v1.0.0` with the deprecation metadata. The previous artifact is not modified — `ork push` is always a new upload, never an in-place mutation.

---

## What consumers see

Consumers who inspect the deprecated version see the warning immediately:

```bash
ork inspect web-service:v1.0.0 --motif
```

```text
web-service:v1.0.0  ⚠ Deprecated
  Kind:       Motif
  Deprecated: true

  ⚠ Deprecation notice:
    Use web-service:v1.1.0 — adds configurable probe path and profile.
    Migration: add probeProfile and probePath to your Katalog's with: block.
    Both default to the v1.0.0 behavior.

To import:
  imports:
    - motif: oci://ghcr.io/myorg/motifs/web-service:v1.0.0   ← deprecated
```

Katalogs that import a deprecated Motif will surface the warning at `ork validate` and `ork simulate` time — before any CR is applied.

---

## The deprecation lifecycle

```
v1.0.0  ← active
          │
          │  v1.1.0 ships (backwards-compatible)
          ▼
v1.0.0  ← deprecated ("use v1.1.0")
v1.1.0  ← active
          │
          │  migration complete — all consumers on v1.1.0
          ▼
v1.0.0  ← retired (removed from index, not deleted)
v1.1.0  ← active
```

Retirement removes the pattern from `ork patterns` output but does not delete the OCI artifact. Consumers that pinned a digest can still pull. Consumers that use version tags get a "not found in index" message and should migrate.

---

## Deprecating an entire pattern

When a pattern is replaced by something conceptually different — not just a new version — mark the whole pattern as deprecated with a redirect:

```yaml
metadata:
  name: web-service
  version: v1.0.0
  deprecated: true
  deprecationMessage: >
    web-service is superseded by stateless-app:v1.0.0, which adds
    multi-port support and structured health check configuration.
    See: ork inspect stateless-app:v1.0.0
```

---

## Best practices

**Write the migration path in the deprecation message.** "Use v1.1.0" is not a migration path. "Add `probeProfile: standard` and `probePath: /health` to your `with:` block — both match v1.0.0 defaults" is.

**Give consumers time.** Publish the new version and deprecate the old one simultaneously — don't delete `v1.0.0` when `v1.1.0` ships. Platform teams may have change-freeze windows. The registry is the mechanism that lets them upgrade on their own schedule.

**Check for active importers before retiring.** `ork patterns` shows which patterns are imported by Komposers in the official registry. For your internal registry, check Komposers in your platform repository before retiring a pattern.

**Version the deprecation message itself.** If the migration path changes (e.g., `v1.2.0` ships and is even better than `v1.1.0`), push an updated deprecation message to `v1.0.0`:

```bash
# Update the deprecation message and re-push
ork push web-service:v1.0.0 ./deprecated/
```

---

## Try it

```bash
ork init --pack registry-guide
cd 09-deprecation

cat web-service-deprecated/motif.yaml    # deprecated annotation
ork push web-service:v1.0.0 ./web-service-deprecated/

ork inspect web-service:v1.0.0 --motif  # see the deprecation notice
ork patterns --motifs                    # deprecated patterns shown with ⚠
```
