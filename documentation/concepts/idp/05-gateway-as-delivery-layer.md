# The Gateway as a Standalone Delivery Layer

The Orkestra Gateway API is a delivery surface, not a reconciliation engine. Its job ends the moment a CR lands in the cluster. What happens after — how resources are created, how drift is corrected, how state is reported — belongs entirely to whatever operator or reconciler owns that CRD.

This is a deliberate separation. The gateway does not need to know about your reconciler, and your reconciler does not need to know about the gateway.

---

## What the gateway owns

When a caller applies through the gateway:

1. **Token validation** — the request is authenticated and the token's permissions are checked against the CRD and alias token map.
2. **Field validation** — serve field constraints and CRD schema validation are enforced.
3. **Admission** — the admission webhook fires, running your validation and mutation rules.
4. **Provenance annotation** — `serve-target` and `serve-alias` are stamped on the CR before the SSA patch.
5. **CR delivery** — the CR is applied to the cluster via server-side apply.

The gateway's contract is complete at step 5. It returns `"accepted": true` when the CR is in the cluster. It does not wait for reconciliation, does not poll status, and does not manage the lifecycle of the resource beyond delivery.

---

## What the gateway does not own

- Reconciliation — creating child resources, managing dependencies, handling drift.
- Status reporting — the operator writes status; the gateway reads it back only if you add a `poll.field` to the serve response config.
- Deletion cascades, finalizers, or garbage collection.
- Any behavior that happens after the CR is applied.

The runtime — the Orkestra kordinator and operator controller — is a separate process. The gateway runs without it. You can deploy the gateway against a cluster where the runtime is not installed, and it will accept and deliver CRs to any operator that knows how to handle them.

---

## Running without the runtime

The gateway only needs:

- A Kubernetes cluster with the target CRDs installed.
- A `katalog.yaml` describing the CRDs and their serve configuration.
- Tokens configured in `gateway.api.auth`.

You do not need the Orkestra runtime, kordinator, or any Orkestra-specific controller. The gateway is compatible with any operator installed in your cluster — Argo CD, Crossplane, external-secrets, your own custom operator, or a plain CRD with no controller at all.

```bash
# Gateway running standalone — no runtime, no kordinator
ork proxy --for gateway     # exposes :8443

# Apply a CR to your cluster via the gateway
curl -s -X POST http://localhost:8443/api/v1/apply \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"target":"myapp","name":"payments","team":"platform","environment":"staging"}'
```

The gateway stamps the CR, validates it, and applies it. Argo CD or your operator picks it up and reconciles it. The gateway is done.

---

## Why this matters

Coupling reconciliation and delivery in a single process means that by adopting the delivery mechanism, you adopt the reconciliation model.

The Orkestra Gateway inverts this. You keep your existing reconciler — Argo CD, Flux, Crossplane, or something you built — and layer the gateway on top as a controlled intake surface. Developers interact with the gateway; the CRD is what your operator has always consumed.

The gateway adds:

- **Access control** — token-scoped permissions per CRD and alias.
- **Intent provenance** — every CR carries permanent annotations describing how it arrived.
- **Admission enforcement** — field validation, uniqueness checks, surface-gating.
- **Alias routing** — multiple caller surfaces on the same CRD, each with its own contract.
- **Response shaping** — callers get back only what their surface is allowed to see.

None of this requires changes to the operator. The operator sees a valid CR with the correct spec. The gateway's additions live in annotations and are transparent to existing controllers.

---

## The delivery boundary

```text
caller
  │
  ▼
Gateway API
  │  validates, annotates, applies
  ▼
Kubernetes CR
  │
  ▼
your operator / Argo CD / Flux / Crossplane
  │  reconciles
  ▼
cluster resources
```

The gateway owns the left side of this diagram. Everything to the right of the CR is the reconciler's domain.

---

## See also

- [Deliver, Don't Reconcile](../../foundations/08-deliver-dont-reconcile.md) — the foundational principle behind this design: when the gateway is enabled, every CR delivery passes through it regardless of source, and reconciliation is never its concern.
- [Gateway architecture](../../orkestra-core/02-gateway/index.md#standalone-deployment) — the gateway process can also run standalone as a pure admission and webhook layer without any operators deployed.
- [Aliases and provenance](04-aliases-and-provenance.md) — provenance annotations and alias routing
- [Token scoping](03-token-scoping.md) — per-CRD and per-alias token permissions
- [Schema reference — serve](../../reference/schema/02-katalog/20-serve.md)
