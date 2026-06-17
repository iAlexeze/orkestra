# 08 — Audit trail: bad actor detection

The Orkestra registry records who pushed every pattern version and when.
This example shows how to inspect the audit trail and detect unexpected pushes.

---

> **Before you start:** Replace `ghcr.io/myorg` with your actual registry path throughout this example.

---

## Inspect a pattern

```bash
# Full metadata for a specific version
ork inspect ghcr.io/myorg/katalogs/webapp-operator:v1.1.0

# List all versions with push timestamps
ork inspect ghcr.io/myorg/katalogs/webapp-operator --versions
```

`ork inspect` shows:

```
  Name:        webapp-operator
  Version:     v1.1.0
  Author:      myorg
  Pushed:      2026-06-11T14:23:01Z
  Digest:      sha256:abc123...
  Tags:        web, stateless, ingress
```

## Detecting an unexpected push

If a version appears that was not expected — wrong author, wrong timestamp, or
an unknown digest — the pattern may have been tampered with or pushed by an
unauthorised account.

Steps to investigate:

1. Compare the digest against the CI artifact:
   ```bash
   ork inspect ghcr.io/myorg/katalogs/webapp-operator:v1.1.0 | grep Digest
   ```

2. Pull the pattern and diff against the known-good source:
   ```bash
   ork pull ghcr.io/myorg/katalogs/webapp-operator:v1.1.0 -o /tmp/pulled
   diff -r ./web-service-v1.1.0/ /tmp/pulled/
   ```

3. If compromised, deprecate the version immediately (see `09-deprecation`) and
   push a clean replacement.

## Preventing bad pushes

- Require CI to sign every push with a registry token scoped to the CI identity
- Pin versions in all Komposers — a `latest` tag is unpinnable and unauditable
- Use `ork inspect` in CI to assert no unexpected versions exist after a release

## Next step

→ [09-deprecation/README.md](../09-deprecation/README.md) — mark a pattern deprecated and migrate consumers