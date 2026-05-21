# Contributing to Orkestra

Orkestra is a declarative Kubernetes operator runtime. Operators are described as YAML — the runtime, gateway, and control center turn those descriptions into running reconciliation loops, webhook servers, and an observable UI.

This section is for people who want to extend, improve, or fix Orkestra itself.

---

## Where to go first

If you are new to the codebase, the best entry point is understanding how the project is laid out and which binary owns what. Start here:

→ [Codebase map](codebase-map.md) — which packages belong to the runtime, gateway, control center, and shared layer; where to read and where to start

Once oriented, pick the area you want to work in:

| Area | Guide |
|------|-------|
| Add a resource type to the registry | [orkestra-registry](contributing-registry.md) |
| Improve the control center UI | [control-center](contributing-controlcenter.md) |
| Add or improve a provider (AWS / GCP / Azure / databases) | [providers](contributing-providers.md) |
| Implement rollback | [rollback](contributing-rollback.md) |
| Add or improve an example pack | [examples](contributing-examples.md) |
| Add a note function to make operators more declarative (`pkg/note`) | [pkg/note README](../../pkg/note/README.md) |

---

## General contribution workflow

1. Fork the repository and create a branch from `main`.
2. Read the guide for the area you are changing.
3. Run `make ork` to rebuild after Go changes.
4. Run `go test ./...` to confirm nothing is broken.
5. Open a pull request with a short description of what and why.

Questions? Open a [GitHub Discussion](https://github.com/orkspace/orkestra/discussions).
