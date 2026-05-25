# cmd/internal

Wiring layer for Orkestra. Assembles every komponent, threads dependencies, and hands the result to `pkg/orkestra` — the supervisor that calls `Start()` and `Stop()` in order.

Two public entrypoints:

- **`KonductRuntime`** — runtime. Reconcile loop, informers, leader election. Called by `ork run`.
- **`KonductGateway`** — gateway. TLS + admission webhooks, no reconcilers, no leader election. Called by `ork gate`. Cluster-only.

## Docs

- [01 — Overview](docs/01-overview.md)
- [02 — Runtime](docs/02-runtime.md)
- [03 — Gateway](docs/03-gateway.md)
- [04 — Security](docs/04-security.md)
