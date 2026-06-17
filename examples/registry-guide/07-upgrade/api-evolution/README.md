# API Evolution

When the CRD field layout itself changes — not just the implementation underneath — consumers need a migration path. `spec.port` moving into `spec.expose.port` is a breaking change for anyone who has CRs in the cluster or pipelines writing v1 fields.

Two strategies, pick one:

| Strategy | When to use |
|----------|-------------|
| [with-webhooks](with-webhooks/README.md) | Bidirectional v1↔v2 conversion via Orkestra Gateway — existing CRs are readable and writable in both versions |
| [without-webhooks](without-webhooks/README.md) | `normalize` collapses both field layouts at reconcile time — no multi-version CRD, no webhook |

Both are self-contained. No dependency on [implementation-evolution](../implementation-evolution/README.md).

---

→ Back: [07-upgrade](../README.md)
