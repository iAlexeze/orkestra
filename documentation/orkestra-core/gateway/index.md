# Gateway

The Orkestra Gateway is the external-facing process. It owns every connection that crosses the cluster boundary — admission webhooks, conversion webhooks, and outbound notifications. Nothing else runs inside it.

---

## What it does

**Admission webhooks.** The Gateway serves the `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` endpoints that Kubernetes calls on resource create and update. It enforces deletion protection, namespace restrictions, and per-CRD validation rules.

**Conversion webhooks.** When a CRD has multiple versions, the Gateway converts between them at admission time. This enables zero-downtime schema evolution.

**Certificate management.** The Gateway generates, rotates, and patches TLS certificates for all webhook endpoints automatically. No cert-manager dependency required.

**Notifications.** When the Runtime determines that a notification event should fire, it calls the Gateway's `/notify` endpoint. The Gateway owns dispatch — Slack, email — and handles throttling and delivery. The Runtime never touches external systems directly.

---

## Stateless and horizontally scalable

The Gateway holds no state. It does not watch resources, does not run reconcile loops, and does not participate in leader election. Every replica is identical and interchangeable.

Run two replicas for high availability. Run more under load. Kubernetes will distribute webhook calls across all of them.

---

## Standalone deployment

The Gateway can run without any Runtime. A standalone Gateway gives you deletion protection, admission webhooks, namespace protection, and auto-managed TLS on any cluster — even one with no Orkestra-managed CRDs.

```bash
ork validate -f examples/gateway/katalog.yaml
```

This is useful for platform teams that want the security layer on a cluster before any operators are deployed.

---

## Security-first design

The Gateway sits at a trust boundary — it is the only Orkestra component with a publicly reachable TLS port. Its design reflects that.

**TLS on every connection.** All webhook traffic is served over HTTPS. The Gateway manages certificate rotation automatically, with configurable rotation thresholds and validity periods.

**Failure policy control.** Each webhook endpoint can be configured with `failurePolicy: Fail` or `failurePolicy: Ignore`. Deletion protection defaults to `Fail` — a Gateway outage cannot be used to bypass protection. Admission validation defaults to `Ignore` — an outage degrades gracefully rather than blocking all resource operations.

**No outbound connections on the webhook path.** The Gateway only makes outbound calls for notification dispatch, which is isolated from the webhook serving path.

**Validation boundary.** The same 22-check validation pass that runs in the Runtime also runs in the Gateway at startup. Strict YAML marshalling rejects unknown keys at parse time. The Gateway never processes a Katalog that has not passed every check.

**Cleanup on shutdown.** Webhook configurations registered by the Gateway are removed when it exits cleanly. A cluster does not accumulate stale webhook registrations across deployments.

---

## The shape it carries into the cluster

The security story starts at compile time, before the image is built.

The Gateway image is built with a single Go build tag: `gateway`. This includes one CLI file: `gateway.go`. The reconciler stack, the workqueue, the informer factory, and the entire developer toolchain are never compiled in. A Gateway image literally cannot run a reconciler — the code is not there.

The binary knows two things: how to serve webhooks, and what version it is.

It runs on a `distroless/static` base image — no shell, no package manager, no OS utilities. There is nothing for an attacker to move with after a successful exploit.

**The Gateway also refuses to start outside a Kubernetes pod.** If you invoke `ork gateway` on your terminal, it exits immediately with a clear error. The gateway is not a tool. It is a workload. The binary enforces that distinction at startup.

The combination — minimal binary, distroless base, pod-only constraint — means the attack surface is not just small. It is structurally bounded.

The same principle extends to RBAC. `ork generate bundle --for gateway` produces a ServiceAccount and ClusterRole for the Gateway only — permissions to manage webhook configurations and certificate secrets, and nothing else. It does not receive permissions for any CRD managed by the Runtime. If the Gateway is somehow compromised, its blast radius is limited to what its own ServiceAccount can reach.

---

## Minimal footprint

The Gateway binary contains only the webhook server, the certificate manager, and the notification dispatcher. No reconciler framework, no workqueue, no informer factory.

A Gateway image is typically under 20 MB. It starts in under one second. Liveness and readiness probes are available immediately on startup.
