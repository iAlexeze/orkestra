# Ecosystem Composition

ArgoCD for GitOps. cert-manager for certificates. Crossplane for infrastructure. Prometheus Operator for monitoring. Each tool is good at what it does. Each tool also adds a CRD schema your team has to learn — its fields, its status format, its deletion behavior, its mandatory fields.

Orkestra gives your organisation a unified internal interface on top of all of them. Your team creates an `App`. Orkestra creates the ArgoCD `Application`. The ecosystem tool keeps running — it just stops being the interface your developers interact with.

Define the internal CRD once. Enforce admission rules once. Propagate status back once. Everyone uses the vocabulary your organisation chose, not the vocabulary of every tool you adopted.

This is not a new concept — it is what a good platform team does. Orkestra makes it operational without writing controllers.

---

## The pack

```bash
ork init --pack ecosystem-composition
```

---

## What you will learn

- The abstraction layer pattern — internal CRD → ecosystem resource
- Admission rules on internal CRDs before any ecosystem tool sees the request
- Status propagation from ecosystem resources back to internal CRDs
- Composing all four operators into one runtime with Komposer
- Testing the full chain — from your abstraction down to the ecosystem resource — with `ork e2e`

---

## The examples

| Directory | Internal CRD | Manages |
|-----------|-------------|---------|
| [00-argocd](./01-argocd.md) | `App` | ArgoCD `Application` |
| [01-cert-manager](./02-cert-manager.md) | `SecurityConfig` | cert-manager `Certificate` |
| [02-prometheus](./03-prometheus.md) | `MonitoringConfig` | `ServiceMonitor` + `PrometheusRule` |
| [03-crossplane](./04-crossplane.md) | `Infra` | Crossplane Composite Claim |
| [04-platform-stack](./05-platform-stack.md) | All four | All four, one Komposer, gateway + deletion protection |
| [05-all-in-one](./06-all-in-one.md) | `PlatformResource` | All four tools, one CRD, `workloadType` discriminator |

---

## Try it

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/00-argocd
# Follow steps in README
```

→ [The pattern](./00-the-pattern.md) — concepts behind the abstraction layer
