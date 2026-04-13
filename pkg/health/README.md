# pkg/health

The health package owns the runtime servers that expose Orkestra to both Kubernetes probes and external webhook traffic.

A single `HealthServer` binds two listeners:

- **HTTP** — probe endpoints (`/startup`, `/health`, `/ready`) and Prometheus metrics (`/metrics`).
- **HTTPS** — webhook endpoints (`/convert`, `/validate`, `/mutate`, `/deletion-protection`), started only when the Katalog declares conversion paths, admission rules, or deletion protection.

## Responsibilities

| Concern | Where |
|---------|-------|
| Lifecycle probes | `handler.go` — startup / health / ready handlers |
| Conversion webhook | `conversion.go`, `conversion_logic.go`, `conversion_stats.go` |
| Admission webhook (validate + mutate) | `admission_handlers.go`, `admission_evaluation.go`, `admission_stats.go` |
| Deletion protection webhook | `deletion_protection_handler.go` |
| Webhook registration / cleanup | `webhook_registration.go` |
| Security context | `security.go` |
| Admission rule registry | `admission_registry.go` |

## Server lifecycle

```
NewHealthServer(kubeClient, katalog, konfig)
    ↓
hs.Start(ctx)
    • binds HTTP  → /startup /health /ready /metrics
    • binds HTTPS → /convert /validate /mutate /deletion-protection  (conditional)
    • registers ValidatingWebhookConfiguration / MutatingWebhookConfiguration (in-cluster only)
    ↓
hs.Shutdown(ctx)
    • drains HTTP and HTTPS servers
    • removes ValidatingWebhookConfiguration / MutatingWebhookConfiguration
    • removes deletion protection webhook
```

## Probe semantics

| Endpoint | Returns 200 when… |
|----------|-------------------|
| `/startup` | `SetStartupComplete()` has been called |
| `/health` | `healthy` atomic is true (never flips false in normal operation) |
| `/ready` | `ready` atomic is true — false during boot and after shutdown begins |

## Webhook routing

Routes are registered on the HTTPS mux **before** the server goroutine starts. Order is enforced by the `Start` method — callers cannot register routes after `Start()`.

The HTTPS server only starts if at least one route is needed. This avoids requiring TLS credentials when the Katalog declares no webhooks.

## Deletion protection

The `/deletion-protection` endpoint receives DELETE admission reviews from Kubernetes. The handler:

1. Decodes the `AdmissionReview` request.
2. Checks `isProtectedCRD(name)` — filters against `ProtectedCRDNames()` from the Katalog.
3. Denies if the CRD is managed by this Katalog; allows all others through.

Two-level filtering keeps the webhook rule broad (intercepts all CRD deletions) while the handler narrows to only the CRDs this operator owns.

→ Next: [docs/01-probes.md](docs/01-probes.md)
