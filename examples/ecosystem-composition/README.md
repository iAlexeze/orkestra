# Ecosystem Composition

```bash
ork init --pack ecosystem-composition
```

This pack is about building an internal developer platform — a unified interface for your organisation on top of the tools you already run.

ArgoCD for GitOps. cert-manager for certificates. Crossplane for infrastructure. Prometheus Operator for monitoring. Each tool is good at what it does. Each tool also adds a CRD schema your team has to learn — its fields, its status format, its deletion behavior, its admission rules.

Ten tools. Ten mental models. Every new engineer spends weeks just learning the platform's API surface.

The answer is a unified internal interface: CRDs your organisation defines, in your organisation's vocabulary, with your organisation's admission rules. Orkestra makes those CRDs operational — it manages the translation to ecosystem tools, enforces correctness at admission time, and propagates status back to the internal CRD.

Your team creates an `App`. Orkestra creates the ArgoCD Application.
Your team creates a `SecurityConfig`. Orkestra creates the cert-manager Certificate.
Your team creates an `Infra`. Orkestra creates the Crossplane Claim.

The ecosystem tools keep running. They just stop being the interface your developers interact with.

**This is what an internal developer platform looks like without writing controllers.**

---

> **Before you start:** The katalogs in this pack import the `platform-admission` motif from your registry. Publish it first — or skip this step if you already did it in the registry guide.
>
> ```bash
> export ORK_REGISTRY=ghcr.io/myorg/katalogs
> export ORK_MOTIFS_REGISTRY=ghcr.io/myorg/motifs
> cd motifs && ork push ./platform-admission
> cd -
> ```
>
> **Already completed the registry guide?** The motif is already published — skip the step above.

---

## Running the examples

Each example's README walks through two modes:

**`ork run` (local, no gateway)** — runs the Orkestra reconciler in your terminal against your current cluster. No Helm install needed. Admission rules are enforced at reconcile time: a bad CR is stored in etcd but Orkestra immediately halts and sets a `ValidationFailed` condition — no child resource is ever created.

**Full Helm install with gateway** — installs Orkestra as a cluster deployment with the gateway enabled. Admission webhooks are registered: bad CRs are rejected by Kubernetes at `kubectl apply` time and never reach etcd.

To use the gateway path, install Orkestra with gateway enabled:

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set gateway.enabled=true \
  --wait --timeout 120s
```

Each katalog in this pack already declares `gateway.enabled: true` — the gateway is picked up automatically once Orkestra is running.

---

## Examples

| Example | Internal CRD | Ecosystem resource created |
|---------|-------------|--------------------------|
| [00 — ArgoCD](00-argocd/README.md) | `App` | `argoproj.io/v1alpha1 Application` |
| [01 — cert-manager](01-cert-manager/README.md) | `SecurityConfig` | `cert-manager.io/v1 Certificate` |
| [02 — Prometheus Operator](02-prometheus/README.md) | `MonitoringConfig` | `ServiceMonitor` + `PrometheusRule` |
| [03 — Crossplane](03-crossplane/README.md) | `Infra` | Crossplane Composite Claim |
| [04 — Platform Stack](04-platform-stack/README.md) | All four | All four composed, gateway admission + deletion protection |
| [05 — All-in-One](05-all-in-one/README.md) | `PlatformResource` | All four tools, one CRD, `workloadType` discriminator |
| [06 — IDP](06-idp/README.md) | `PlatformResource` | Same CRD, same operator — Control Center form replaces `kubectl apply` |

Work through them in order. Each example is self-contained. `04` builds on `00`–`03` and adds the full policy layer. `05` shows the trade-off between focused CRDs and a single unified CRD. `06` adds the Gateway API and shows the Control Center as a developer portal — read it after `05`.

---

## E2E

Every example ships with an `e2e.yaml`. The e2e installs the ecosystem tool via `setup.helm` so the test runs against a real ArgoCD or cert-manager — it does not mock anything.

```bash
cd 00-argocd && ork e2e
```

Run the full suite from root:

```bash
ork e2e -f e2e.yaml
```
