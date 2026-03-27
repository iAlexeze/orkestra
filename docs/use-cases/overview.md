# Use Cases

Orkestra is a declarative operator runtime. Every operator is a **Katalog** — a YAML file that defines what CRDs to manage and how to reconcile them.

This section shows what becomes possible when your operator is a file instead of a codebase.

---

## Categories

### Core Patterns
- Zero‑code operators  
- Platform namespace provisioning  
- Secret distribution  
- Multi‑CRD dependency ordering  

### Team & Org‑Level Patterns
- Centralised operator configuration (GitOps)  
- Multi‑team composition  
- Environment‑specific overrides (Komposer)  
- Helm‑driven operator configuration  

### Operational Patterns
- Progressive rollout  
- Disaster recovery  
- Air‑gapped environments  
- Observability  

### Advanced Patterns
- Registry‑powered operators  
- Multi‑version CRD conversion  
- Go hooks  
- Custom constructors  

!!! tip
    Each use case is self‑contained. You can read them in any order.
