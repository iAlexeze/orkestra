# Ecosystem Composition Pack

Your platform already runs ArgoCD, cert-manager, Prometheus Operator, and Crossplane. This pack shows how to wrap each one with an Orkestra abstraction layer — internal CRDs your team already knows, backed by the ecosystem tools they use today.

Orkestra does not replace these tools. It reduces the interface to what your team actually needs to express.

```bash
ork init --pack ecosystem-composition
```

---

## The examples

| Directory | Wraps |
|-----------|-------|
| `00-argocd` | `App` → ArgoCD `Application` |
| `01-cert-manager` | `SecurityConfig` → cert-manager `Certificate` |
| `02-prometheus` | `MonitoringConfig` → `ServiceMonitor` + `PrometheusRule` |
| `03-crossplane` | `Infra` → Crossplane Composite Claim |
| `04-platform-stack` | All four in one Komposer |
| `05-policy-layer` | Shared admission motif across all four |

---

## What you get from every example

- Admission rules on the internal CRD — enforced before the ecosystem tool sees the request
- Status propagation back from the ecosystem resource to the internal CR
- Deletion protection — no accidental cascade
- A simulate proof and a full e2e that installs the real ecosystem tool and verifies the downstream resource was created

→ [Ecosystem Guide](../../guides/ecosystem/index.md) — concept per tool, e2e story, platform-stack composition
