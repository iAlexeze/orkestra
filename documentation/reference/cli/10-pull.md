# ork pull

Download a pattern from the OCI registry to the local cache. Subsequent commands (`ork simulate`, `ork e2e`) resolve the short name against the cache — no repeated network call.

```bash
ork pull [<name>:<version>]
ork pull -f <katalog-or-komposer.yaml>
```

---

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | | Pull all OCI imports declared in a katalog or komposer file. |
| `--out` | `-o` | | Extract the pulled artifact to this directory. |
| `--motif` | `-m` | `false` | Resolve as a motif (uses `ORK_MOTIFS_REGISTRY` instead of `ORK_REGISTRY`). |
| `--refresh` | | `false` | Force re-download even if the version is already cached. |

---

## Registry resolution

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
export ORK_MOTIFS_REGISTRY=ghcr.io/myorg/motifs
```

A short name (`postgres:v14`) resolves against `ORK_REGISTRY` (katalogs) or `ORK_MOTIFS_REGISTRY` (with `--motif`). A full OCI ref (`oci://ghcr.io/...`) is used as-is.

Authentication reads `~/.docker/config.json`. Run `docker login <registry>` first.

---

## Cache location

Downloaded artifacts land in `~/.orkestra/registry/<registry-host>/<name>/<version>/`. The directory contains all files from the artifact — `katalog.yaml`, `crd.yaml`, `simulate.yaml`, etc.

---

## Examples

```bash
# Pull a katalog
ork pull postgres:v1.0.0

# Pull a motif
ork pull web-service:v1.0.0 --motif

# Pull using a full OCI ref
ork pull oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0

# Pull all imports from a komposer file
ork pull -f komposer.yaml

# Pull all motif imports from a katalog file
ork pull -f katalog.yaml

# Extract to a local directory (for inspection or editing)
ork pull postgres:v1.0.0 -o ./local-postgres

# Force re-download
ork pull postgres:v1.0.0 --refresh
```

---

## Output

```text
Pulling postgres:v1.0.0
  → oci://ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres:v1.0.0
  ✓ Cached at ~/.orkestra/registry/ghcr.io/orkspace/orkestra-registry/patterns/katalogs/postgres/v1.0.0
```

With `--file`:

```text
Pulling imports from komposer.yaml...
  ✓ webapp-operator:v1.0.0
  ✓ cache-operator:v1.0.0
  ✓ postgres:v1.0.0

3 pattern(s) pulled.
```

---

## Run simulate on a pulled pattern

```bash
ork pull postgres:v1.0.0 -o ./postgres-local
cd ./postgres-local
ork simulate
# ✓ 3/3 assertions passed
```

The pulled artifact includes `simulate.yaml`. Running it locally verifies the behavioral proof the author published.

---

## Related

- [`ork inspect`](./11-inspect.md) — read metadata without downloading files
- [`ork push`](./09-push.md) — publish a pattern
- [Consuming Patterns](../../guides/registry/02-consuming.md)
