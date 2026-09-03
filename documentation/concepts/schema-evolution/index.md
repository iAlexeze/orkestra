# Schema Evolution

Schema evolution is the problem of changing what a CR looks like over time — without breaking the operators, clients, and stored objects that depend on the current shape.

Orkestra provides three approaches, each anchored to a different API layer. The choice is really a question of where in the stack the translation should live.

---

## Three approaches, three layers

**[Kubernetes API — `conversion.paths:`](./02-conversion-paths.md)**

The API server is the translation point. Orkestra Gateway handles `/convert`; the API server stores objects at one version and serves them at another. Multi-version CRD, bidirectional, lossless.

Use when external clients target a specific API version, or when live objects at v1 must remain readable as v1.

---

**[Runtime API — `normalize:`](./01-normalize.md)**

The reconciler is the translation point. The runtime normalises input format variation before reconcile runs. One CRD version. No webhook. No TLS.

Use when you control who creates the CRs and want to tolerate input shape variation without versioning overhead.

---

**[Gateway API — `serve.fields.values`](./03-serve-translation.md)**

The Gateway is the translation point. Callers submit a simplified intent through the Gateway API; `serve.fields.values` fans the fields out to the CRD's internal schema before the CR is written. One CRD version. No webhook. The CRD schema is an implementation detail — callers never see it.

Use when callers submit intents through the Gateway and should not be coupled to the CRD schema.

---

## At a glance

| | Kubernetes API | Runtime API | Gateway API |
|---|---|---|---|
| **Mechanism** | `conversion.paths:` | `normalize:` | `serve.fields.values` |
| **Translation point** | API server (`/convert`) | Reconciler | Gateway (before CR is written) |
| **CRD versions** | Two or more | One | One |
| **Caller sees CRD schema** | Yes | Yes | No |
| **Conversion webhook** | Yes — Orkestra Gateway's `/convert` | No | No |
| **TLS required** | Yes — auto-generated | No | No |
| **Gateway required** | Yes | No | Yes |
| **Objects in etcd** | One storage version, served in any declared version | One format | One format |

---

## Where to go next

- **[Kubernetes API — conversion.paths:](./02-conversion-paths.md)** — multi-version CRD with Orkestra as the conversion webhook
- **[Runtime API — normalize:](./01-normalize.md)** — input tolerance without versioning overhead
- **[Gateway API — serve.fields.values](./03-serve-translation.md)** — Gateway-layer translation; callers never see the CRD schema
