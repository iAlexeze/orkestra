# Schema Evolution

Schema evolution is the problem of changing what a CR looks like over time — without breaking the operators, clients, and stored objects that depend on the current shape.

Orkestra provides two approaches. The choice comes down to one question: does the Kubernetes API server need to know there are two versions?

---

## Two approaches

**[`normalize:`](./01-normalize.md)** — the API server never sees two versions. The Katalog normalises input format variation before reconcile runs. One CRD version. No webhook. No TLS.

**[`conversion.paths:`](./02-conversion-paths.md)** — the API server stores objects at one version and serves them at another. Orkestra Gateway handles `/convert`. Multi-version CRD, bidirectional, lossless.

---

## Pick one

| | `normalize:` | `conversion.paths:` |
|---|---|---|
| **CRD versions** | One | Two or more |
| **Kubernetes sees** | One schema | Multi-version CRD spec |
| **Conversion webhook** | No | Yes — Orkestra Gateway's `/convert` |
| **TLS required** | No | Yes — auto-generated |
| **Objects in etcd** | One format | One storage version, served in any declared version |
| **Best for** | Internal operators, tolerating input shape variation | Public APIs, external clients targeting a specific version, migrating live objects |

If you control who creates the CRs and want to iterate quickly: `normalize:`.

If external clients target a specific API version, or you have live objects at v1 that **must** remain readable as v1: `conversion.paths:`.

---

## Where to go next

- **[normalize:](./01-normalize.md)** — input tolerance without versioning overhead
- **[conversion.paths:](./02-conversion-paths.md)** — multi-version API with Orkestra as the conversion webhook
