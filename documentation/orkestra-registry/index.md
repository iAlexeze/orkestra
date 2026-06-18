# Orkestra Registry

The Orkestra Registry is the distribution layer for operator patterns. Where traditional ecosystems distribute binaries, the registry distributes **behavior** — Katalogs, Motifs, and Komposers published as OCI artifacts that any Orkestra runtime can pull and interpret.

Patterns are versioned, immutable, and discoverable. A Katalog pattern pulled at `postgres:v14` behaves identically in every environment. Credentials use `~/.docker/config.json` — run `docker login` before pushing.

---

## CLI

| Command | Description |
|---------|-------------|
| `ork patterns` | Browse available patterns in the registry |
| `ork pull <name>:<version>` | Pull a pattern to the local cache |
| `ork inspect <name>:<version>` | Show metadata without downloading files |
| `ork push <name>:<version> <dir>` | Publish a pattern directory |

Override the default registries:

```bash
export ORK_REGISTRY=oci://myregistry.internal/patterns
export ORK_MOTIFS_REGISTRY=oci://myregistry.internal/motifs
```

---

## Pattern kinds

- [Motifs](01-motifs.md) — reusable resource primitives imported into Katalogs
- [Katalogs](02-katalogs.md) — complete operator declarations, one CRD or many
- [Komposers](03-komposers.md) — platform declarations that compose multiple Katalogs
- [E2E](04-e2e.md) — verification framework that gates publication
- [Simulate](05-simulate.md) — declarative reconciler assertions, no cluster required

---

## Where to go next

- **[Motifs](./01-motifs.md)** — reusable resource primitives
- **[Katalogs](./02-katalogs.md)** — publishing and pulling complete operator patterns
- **[Komposers](./03-komposers.md)** — platform-level composition across multiple Katalogs
- **[E2E](./04-e2e.md)** — gating publication with declarative verification
- **[Simulate](./05-simulate.md)** — fast reconciler assertions alongside your Katalog
