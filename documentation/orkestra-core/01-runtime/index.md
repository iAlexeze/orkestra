# Runtime

The Orkestra Runtime is the reconciliation engine. It is what turns a Katalog declaration into a running operator.

---

## What it does

The Runtime watches the Kubernetes API for the CRDs declared in your Katalog. When a CR is created, updated, or deleted, the Runtime reconciles it — creating child resources, resolving templates, patching status, enforcing drift correction, and managing the full resource lifecycle.

It exposes a `/katalog` HTTP API that the Control Center reads, and it calls the Gateway's `/notify` endpoint for outbound notifications.

---

## One binary, one operator

A Runtime instance corresponds to one Katalog. It owns the reconciliation of every CRD declared in that Katalog and nothing else. Multiple operators run as multiple Runtime instances — isolated, independently deployable, independently upgradeable.

---

## Production-ready by default

The Runtime is built for production from the first deployment:

**Leader election.** Only one Runtime replica processes work at a time. Failover is automatic and handled by Kubernetes lease coordination — no external dependency.

**Ordered startup.** CRD workers start in topological order. A worker for a CRD that depends on another does not start until its dependency is healthy. Crash loops cannot propagate downstream.

**Workqueue-backed processing.** Every CR update goes through a rate-limited, deduplicating workqueue. Bursts are absorbed, not dropped. Repeated failures back off automatically.

**Declarative drift correction.** The Runtime continuously reconciles declared state against live state. Resources mutated outside the operator converge back on every resync cycle.

**Graceful shutdown.** In-flight reconciles complete. Finalizers run. The process exits cleanly.

---

## Security-first design

The Runtime is deliberately minimal. What it does not do is as important as what it does.

- No TLS serving — that is the Gateway's responsibility
- No outbound HTTP to external systems — notifications are delegated to the Gateway
- No cluster-admin permissions — RBAC is scoped to exactly the resources declared in the Katalog
- No arbitrary code execution from configuration — Katalog templates are pure data transformation notes, not eval

**Validation boundary.** Before the Runtime starts a single reconciler, the Katalog passes through 22+ sequential validation checks — field constraints, uniqueness, dependency cycle detection, reconciler mode consistency, status types, service shapes, custom resource declarations, gateway requirements, and more. This runs on every cluster-bound operation: `ork run`, `ork generate bundle`, `ork validate`. Any invalid declaration fails immediately with an error, before anything touches the cluster.

YAML is strictly marshalled. Unknown keys are rejected at parse time — there is no silent ingestion of unexpected fields. The validation layer only ever sees a structurally valid, fully known input.

`ork generate bundle --for runtime` produces a ServiceAccount and ClusterRole scoped to exactly the resources your Katalog declares — nothing more.

---

## The shape it carries into the cluster

The security story is not about what the Runtime is configured to do — it is about what was compiled into it.

The Runtime image is built with a single Go build tag: `runtime`. This means only one CLI file is included: `run.go`. The entire developer toolchain — `ork init`, `ork notes`, `ork generate`, `ork push`, `ork pull`, `ork inspect`, `ork patterns`, `ork validate`, `ork template`, `ork diff`, `ork simulate` — is never compiled in. Not disabled. Not stripped. Never compiled.

The binary knows two things: how to run, and what version it is.

It runs on a `distroless/static` base image — no shell, no package manager, no OS utilities of any kind. If an attacker finds an exploitable path in the reconcile logic, they land in a container with a single-command binary and no environment to work with.

The surface area reduction is structural, not procedural. It cannot be misconfigured away.

The same principle applies to RBAC. `ork generate bundle --for runtime` produces a ServiceAccount and ClusterRole scoped to exactly the resources your Katalog declares — nothing more. The Runtime never receives permissions for the Gateway's resources (webhook configurations, cert secrets), and the Gateway never receives permissions for the Runtime's resources. Each component carries only the cluster permissions it actually needs.
