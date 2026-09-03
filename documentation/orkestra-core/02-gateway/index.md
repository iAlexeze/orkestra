# Gateway

The Orkestra Gateway is the external-facing process. It owns every connection that crosses the cluster boundary — incoming intents, admission webhooks, conversion webhooks, and outbound notifications.

---

## What it does

**Serve API.** The Gateway is an intent runner — a delivery channel for CRs to the cluster that requires no Kubernetes knowledge on the caller's side. A caller submits a named set of fields; the Gateway resolves the target, checks token permissions, constructs the full CR, stamps provenance, runs admission rules, applies the CR to the cluster via server-side apply, and returns a shaped response. This is the primary delivery surface for Orkestra-managed operators. See [Serve API](./02-serve-api.md).

**Admission webhooks.** The Gateway serves the `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` endpoints that Kubernetes calls on resource create and update. It enforces deletion protection, namespace restrictions, and per-CRD validation and mutation rules. See [Admission](./01-admission.md).

**Conversion webhooks.** When a CRD has multiple versions, the Gateway converts between them at admission time, enabling zero-downtime schema evolution.

**Certificate management.** The Gateway generates, rotates, and patches TLS certificates for all webhook endpoints automatically. No cert-manager dependency required.

**Notifications.** When the Runtime determines that a notification event should fire, it calls the Gateway's `/notify` endpoint. The Gateway owns dispatch — Slack, email — and handles the delivery.

---

## Stateless and horizontally scalable

The Gateway holds no state. It does not watch resources, does not run reconcile loops, and does not participate in leader election. Every replica is identical and interchangeable.

Run two replicas for high availability. Run more under load. Kubernetes distributes webhook calls and serve requests across all of them.

---

## Running

The Gateway has two modes:

- **In-cluster** — deployed as a Kubernetes workload, serving TLS webhooks and the Serve API against real CRs
- **Local** — no cluster required; `ork serve play` runs the full intent chain in process, `ork gate` evaluates admission rules against a CR file

See [Running the Gateway](./03-running.md).

---

## Security design

The Gateway binary is built with a single Go build tag: `gateway`. The reconciler stack, workqueue, informer factory, and developer toolchain are never compiled in. A Gateway image cannot run a reconciler — the code is not there.

It runs on a `distroless/static` base image — no shell, no package manager, no OS utilities.

`ork generate bundle --for gateway` produces a ServiceAccount and ClusterRole scoped to webhook configuration management and certificate secrets only. It receives no permissions for any CRD managed by the Runtime.

---
## Where to go next

- [Serve API](./02-serve-api.md)
- [Admission](./01-admission.md)
- [Running the Gateway](./03-running.md)