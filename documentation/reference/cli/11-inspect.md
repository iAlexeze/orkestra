# ork inspect

Show metadata for a pattern version — registry, kind, digest, simulate status, E2E status, file list, and the exact import snippet — without downloading any files.

```bash
ork inspect <name>:<version>
ork inspect <name>:<version> --view <file>[,<file>]
```

`ork inspect` reads OCI annotations from the registry manifest. It does not download layers. Use it to check quality signals before deciding to pull or import.

---

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--motif` | `-m` | `false` | Resolve as a motif (uses `ORK_MOTIFS_REGISTRY`). |
| `--versions` | | `false` | List up to 10 tracked versions sorted by push date, newest first. |
| `--view` | | | Comma-separated list of files to print without pulling the full artifact. |

---

## Examples

```bash
# Inspect a katalog
ork inspect postgres:v1.0.0

# Inspect a motif
ork inspect web-service:v1.0.0 --motif

# Full OCI ref
ork inspect oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0

# List all tracked versions of a katalog
ork inspect postgres --versions

# List all tracked versions of a motif
ork inspect web-service --motif --versions

# Read the simulate assertions before importing
ork inspect postgres:v1.0.0 --view simulate.yaml

# Read the katalog and sample CR
ork inspect postgres:v1.0.0 --view katalog.yaml,cr.yaml
```

---

## Output

```text
postgres:v1.0.0
  Registry:    ghcr.io/orkspace/orkestra-registry/patterns/katalogs
  Kind:        Katalog
  Digest:      sha256:a3f1c8d20e4b7f9c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4
  Pushed:      2026-05-24T10:00:00Z
  Size:        14.2 KB

  Description: Declarative PostgreSQL deployment operator

  Tags:        postgres, database, statefulset
  Author:      orkspace
  License:     Apache-2.0
  Simulate:    ✓ Verified · 3 assertions · 3ms · tested 2 days ago
  E2E:         ✓ Verified · 48s · tested 2 days ago

  Files:
    katalog.yaml    5.1 KB
    crd.yaml        3.2 KB
    cr.yaml         512 B
    simulate.yaml   1.1 KB
    e2e.yaml        2.4 KB
    README.md       4.8 KB

To pull:
  ork pull postgres:v1.0.0

To import:
  imports:
    registry:
      - oci://ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v1.0.0
```

For a motif (`--motif`):

```text
To pull:
  ork pull web-service:v1.0.0 --motif

To import:
  imports:
    - motif: oci://ghcr.io/myorg/motifs/web-service:v1.0.0
```

With `--versions` (katalog):

```text
postgres  (3 versions)

  v1.2.0        ✓ Simulate · 3 assertions   ✓ E2E · 48s   ← latest
  v1.1.0        ✓ Simulate · 3 assertions   ✓ E2E · 51s
  v1.0.0        ✓ Simulate · 2 assertions   - Not verified
```

With `--versions --motif`:

```text
web-service  (2 versions)

  v1.1.0    ← latest
  v1.0.0
```

Versions are sorted by push date, newest first — matching the order shown in `ork patterns`. Floating tags (`latest`, `stable`) that point to the same digest as a versioned tag are deduplicated and do not appear as separate entries.

---

## Simulate status values

| Display | Meaning |
|---------|---------|
| `✓ Verified · N assertions · Xms` | `simulate.yaml` present, all assertions passed |
| `⊘ Skipped (pushed with --force or --no-simulate)` | Gates bypassed at push time |
| `⚠ No assertions` | `simulate.yaml` present but no `expect:` block |
| `- Not verified` | No `simulate.yaml` in artifact |

## E2E status values

| Display | Meaning |
|---------|---------|
| `✓ Verified · Xs · tested N ago` | All E2E expectations passed |
| `⊘ Skipped (pushed with --force or --no-e2e)` | E2E bypassed at push time |
| `- Not verified` | No `e2e.yaml` in artifact |

---

## --view: read files without a full pull

```bash
ork inspect postgres:v1.0.0 --view simulate.yaml
```

```text
--- simulate.yaml ---
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: postgres-sim
spec:
  katalog: ./katalog.yaml
  cr: ./cr.yaml
  cycles: 5
  expect:
    ...
```

Useful for reading the simulate assertions (the behavioral contract), the sample CR, or the katalog before committing to a pull. Only the requested files are downloaded — not the full artifact.

---

## Related

- [`ork pull`](./10-pull.md) — download the full artifact
- [`ork patterns`](./12-patterns.md) — browse what's in the registry
- [Consuming Patterns](../../registry-guide/02-consuming.md)
