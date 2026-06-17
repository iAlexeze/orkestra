# ork patterns

List available patterns in the registry. Shows name, latest version, kind (Katalog or Motif), E2E status, tags, and description.

```bash
ork patterns [registry-url]
```

With no arguments, queries both `ORK_REGISTRY` (katalogs) and `ORK_MOTIFS_REGISTRY` (motifs). Pass an explicit registry URL to query a specific registry instead.

---

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--katalogs` | `-k` | `false` | Show only patterns with `kind: Katalog`. |
| `--motifs` | `-m` | `false` | Show only patterns with `kind: Motif`. |
| `--tag` | `-t` | | Filter by tag — shows only patterns tagged with this value. |

---

## Examples

```bash
# Browse all patterns in both registries
ork patterns

# Katalogs only
ork patterns --katalogs

# Motifs only
ork patterns --motifs

# Filter by tag
ork patterns --tag database
ork patterns --tag stateful --katalogs

# Query a specific registry
ork patterns oci://ghcr.io/myorg/katalogs
ork patterns oci://ghcr.io/myorg/motifs
```

---

## Output

```text
Orkestra Registry
─────────────────────────────────────────────────────────────────────────────
NAME              LATEST  KIND     E2E  TAGS                    DESCRIPTION
postgres          v1.0.0  Katalog  ✓    postgres, database,...  Declarative PostgreSQL...
redis             v1.0.0  Katalog  ✓    redis, cache,...        Declarative Redis...
kafka             v3.6.0  Katalog  ✓    kafka, messaging,...    Declarative Apache Kaf...
web-service       v1.1.0  Motif    -    web, deployment,...     Deployment + Service +...
data-store        v1.0.0  Motif    -    statefulset, storage    StatefulSet + PVC + cr...

5 pattern(s) — updated 3h ago
```

The `E2E` column:
- `✓` — artifact carries `io.orkestra.e2e.status=passed`
- `-` — no E2E result (Motifs, or Katalogs published without `e2e.yaml`)
- `✗` — E2E was run but failed, or force-pushed

---

## Registry resolution

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
export ORK_MOTIFS_REGISTRY=ghcr.io/myorg/motifs
```

Without these variables set, the command queries the official Orkestra registry. To query your own internal registry alongside the official one, set both variables.

---

## After browsing

```bash
# Inspect a pattern before importing
ork inspect postgres:v1.0.0

# Read the simulate assertions
ork inspect postgres:v1.0.0 --view simulate.yaml

# Pull
ork pull postgres:v1.0.0
```

---

## Related

- [`ork inspect`](./11-inspect.md) — full metadata and quality signals for one version
- [`ork pull`](./10-pull.md) — download a pattern
- [Consuming Patterns](../../registry-guide/02-consuming.md)
