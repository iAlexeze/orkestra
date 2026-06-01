# Multi-tenancy

Three focused examples showing how Katalog namespaces scope CRDs by team within a single runtime and how `crossAccess` controls which teams can read each other's CR state.

| Example | What it teaches |
|---|---|
| [01 — Basic namespacing](01-basic-namespacing/README.md) | Two teams, one runtime — Control Center renders a separate panel per namespace |
| [02 — Cross-read access control](02-cross-access-control/README.md) | `crossAccess: false` closes a Katalog; one CRD overrides back to open |
| [03 — Shared platform](03-shared-platform/README.md) | Platform infra CRDs expose endpoints; application teams read them via `cross:` |

Each example is self-contained with its own `crd.yaml`, `cr.yaml`, `komposer.yaml`, and `cleanup.sh`. Run any example in isolation from its subfolder, or run them all at once with the root `komposer.yaml`.

---

**Further reading:** [Multi-tenancy concept doc](https://orkestra.sh/docs/concepts/operatorbox/multi-tenancy/)
