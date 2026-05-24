# Orkestra Registry

The Orkestra Registry is the distribution layer for operator patterns. Where traditional ecosystems distribute binaries, the registry distributes **behavior** — Katalogs, Motifs, and Komposers published as OCI artifacts that any Orkestra runtime can pull and interpret.

Patterns are versioned, immutable, and discoverable. A Katalog pattern pulled at `postgres:v14` behaves identically in every environment. Credentials use `~/.docker/config.json` — run `docker login` before pushing.

---

## CLI

| Command | Description |
|---------|-------------|
| `ork registry list` | Browse available patterns in the registry |
| `ork registry pull <name>:<version>` | Pull a pattern to the local cache |
| `ork registry info <name>:<version>` | Show metadata without downloading files |
| `ork registry push <name>:<version> <dir>` | Publish a pattern directory |

Override the default registries:

```bash
export ORK_REGISTRY=oci://myregistry.internal/patterns
export ORK_MOTIFS_REGISTRY=oci://myregistry.internal/motifs
```

---

## Pattern kinds

- [Motifs](motifs.md) — reusable resource primitives imported into Katalogs
- [Katalogs](katalogs.md) — complete operator declarations, one CRD or many
- [Komposers](komposers.md) — platform declarations that compose multiple Katalogs
- [E2E](e2e.md) — verification framework that gates publication
