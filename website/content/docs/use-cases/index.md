# Use Cases

Orkestra is a declarative operator runtime. Every operator is a **[Katalog](../runtime-manual/concepts/katalog.md)** — a YAML file that defines what CRDs to manage and how to reconcile them.

This section shows what becomes possible when your operator is a file instead of a codebase.

---

## Categories

### Core Patterns
- [Zero‑code operators](./zero-code-operators.md)  
- [Platform namespace provisioning](./platform-namespace-provisioning.md)
- [Secret distribution](./secret-distribution.md)
- [Multi‑CRD dependency ordering](./dependency-ordering.md)

### Team & Org‑Level Patterns
- [Centralised operator configuration (GitOps)](./centralized-configuration.md)
- [Multi‑team composition](./multi-team-composition.md)
- [Environment‑specific overrides (Komposer)](./environment-overrides.md)
- [Helm‑driven operator configuration](./helm-driven-operators.md)

### Operational Patterns
- [Progressive rollout](./progressive-rollout.md)
- [Disaster recovery](./disaster-recovery.md)
- [Air‑gapped environments](./air-gapped.md)
- [Observability](./observability.md)

### Advanced Patterns
- [Registry‑powered operators](./registry.md)
- [Multi‑version CRD conversion](./conversion.md)
- [Go hooks](./hooks.md)
- [Custom constructors](./custom-constructors.md)

:::tip
Each use case is self‑contained. You can read them in any order.
:::
