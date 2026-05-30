---
title: "Security"
weight: 5
---

Orkestra treats security as a structural property of the system, not a configuration option you bolt on later. Every layer — the binary, the permissions, the admission gates, the runtime enforcement — is designed so that the default posture is the safe one.

This page is the entry point. Each section below links to a dedicated deep-dive. Read them in order if you are new to Orkestra's security model; jump to any section if you already know what you are looking for.

---

## How Orkestra is secured

Orkestra's security model has five interlocking layers, each described in its own document:

| Layer | What it protects | Document |
|-------|-----------------|----------|
| **Binary surface** | Production containers cannot run developer commands | [Binaries & build tags](01-binaries.md) |
| **Permissions** | The operator only has the rights it needs, derived from your Katalog | [RBAC](02-rbac.md) |
| **Admission control** | Bad CRs are rejected before they reach etcd | [Admission webhooks](03-admission.md) |
| **Deletion protection** | CRs and the operator itself cannot be accidentally deleted | [Deletion protection](04-deletion-protection.md) |
| **Namespace isolation** | CRs are confined to the namespaces you allow | [Namespace protection](05-namespace-protection.md) |
| **Validation pipeline** | Strict parsing and multi-stage validation before anything runs | [Validation pipeline](06-validation-pipeline.md) |
| **Pod security** | Workload containers run with hardened security contexts | [Pod security](07-pod-security.md) |

These layers are independent and can be enabled in any combination. You do not need all five to get value from any one of them.

---

## The two deployment modes

Understanding Orkestra's deployment modes is essential before reading any security document, because some protections behave differently depending on which mode you are running.

### Runtime mode (full)

```text
┌────────────────────────────────────────────┐
│  Kubernetes cluster                        │
│                                            │
│  ork run   ←── watches CRDs, reconciles    │
│  ork gate  ←── webhooks, TLS, admission    │
└────────────────────────────────────────────┘
```

The Orkestra runtime (`ork run`) reconciles your CRDs. The Gateway (`ork gate`) handles all webhook traffic. Both run in the cluster; they communicate through the Kubernetes API server.

In this mode the reconciler continuously enforces the label invariants declared in your Katalog: it adds the deletion-protection label when protection is enabled, removes it when a CRD opts out, and corrects any manual drift. The system is self-healing.

### Gateway-only mode (standalone)

```text
┌────────────────────────────────────────────┐
│  Kubernetes cluster                        │
│                                            │
│  ork gate  ←── webhooks, TLS, admission    │
│  (no runtime reconciler)                   │
└────────────────────────────────────────────┘
```

Only the Gateway is deployed. Admission webhooks, deletion protection, and namespace protection all work. But there is no reconciler, so label enforcement is not automatic: **you are responsible for applying the correct labels to your resources yourself**. See [Deletion protection — gateway-only mode](04-deletion-protection.md#gateway-only-mode) for what this means in practice.

This mode is useful when you bring your own operator or a cluster with Kubernetes resources and only want Orkestra's admission and protection layer.

You might also need this mode when you are testing locally (deploy gateway via Helm, run `ork run` in your terminal — the gateway continues to honour its contract, `ork run` becomes its companion runtime, no blockers).

---

## Minimal cluster access surface

Only two commands connect to and actively manage a real Kubernetes cluster:

| Command | What it does in the cluster |
|---------|-----------------------------|
| `ork run` | Applies CRDs, starts reconcilers, watches and manages resources |
| `ork gate` | Registers webhooks, processes admission requests |

Every other command — `ork validate`, `ork simulate`, `ork template`, `ork notes`, `ork generate`, `ork init`, `ork plan --bundle` — runs entirely offline. No cluster credentials are read, no API server calls are made.

This minimal surface is intentional. If a command does not need the cluster, it does not touch it. The [Validation pipeline](./06-validation-pipeline.md) covers the full offline path.

---

## Enforcement at two independent points

For every security rule declared in the Katalog, Orkestra enforces it at two independent places:

```bash
kubectl apply
    │
    ▼
[Admission webhook]   ← fast gate: rejects before etcd
    │ (passes)
    ▼
  etcd (stored)
    │
    ▼
[Reconciler]          ← backstop: re-checks before acting
    │ (passes)
    ▼
  Child resources created
```

Both layers must fail independently before a rule is violated. In gateway-only mode, only the webhook layer is active.

---

## Validating your security posture

Before deploying, you can inspect the security posture of every CRD in your Katalog with:

```bash
ork validate
```

The output shows each CRD's protection level at a glance:

```text
● app
    kind: App / group: security.orkestra.io / version: v1alpha1
    strict-protection: 🔓 CRD only (CRs not protected)

● database
    kind: Database / group: security.orkestra.io / version: v1alpha1
    protection: 🛡️  full (CRD + CRs)

⚠ cache
    kind: Cache / group: security.orkestra.io / version: v1alpha1
    strict-protection: ⚠ CRs only (CRD not protected - see warning)
    warning: protectCRs=true is ineffective once the CRD is deleted.
             Consider setting protectCRD=true if you intend to protect instances permanently.
```

Warnings surface misconfigurations before you ship — not after. `ork validate` is the fastest way to audit what your Katalog actually enforces.

To review all permissions that will be requested for your CRDs and enabled features in the katalog, run:

```bash
ork validate --full
```

---

## Testing security end-to-end

Every security example ships with an `e2e.yaml` file. Run the full scenario in an example directory with a single command:

```bash
ork e2e
```

The declarative E2E format lets you describe what should be allowed, what should be blocked, and assert that both are true — without writing test code.

For all scenarios, run:
```bash
ork init --pack security
```

---

## Vulnerabilities

Report security issues privately with reproduction steps, relevant logs, and an impact assessment. Responsible disclosure is appreciated and protects all users.

---

## Where to go next

- **[Binaries](./05-binaries.md)** — verifying Orkestra release artifacts
- **[RBAC](./02-rbac.md)** — generating and scoping ClusterRoles
- **[Admission Control](./01-admission.md)** — deny/warn rules at admission time
- **[Deletion Protection](./04-deletion-protection.md)** — preventing accidental CR and CRD deletion
- **[Namespace Protection](./03-namespace-protection.md)** — admission and runtime enforcement
- **[Validation Pipeline](./06-validation-pipeline.md)** — strict parsing, offline validation, and minimal cluster access
- **[Pod Security](./07-pod-security.md)** — container and pod security context profiles
