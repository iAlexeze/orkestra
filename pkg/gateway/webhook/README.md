# pkg/gateway/webhook

The webhook package owns Orkestra's entire HTTPS admission and conversion surface. It implements `domain.Komponent` so it is managed by the same lifecycle supervisor that manages all other runtime components.

A single `WebhookServer` binds one HTTPS listener and handles all webhook traffic:

- `/validate` — admission validation (deny or warn)
- `/mutate` — admission mutation (apply defaults)
- `/convert` — CRD version conversion
- `/deletion-protection` — block deletion of managed CRDs and Orkestra resources
- `/namespace-protection` — enforce allowed/restricted namespace rules
- `/strict-mode-protection` — block removal of the deletion-protection label (strict mode only)

The HTTPS server starts only when the Katalog declares at least one webhook capability. When no capabilities are declared, `Start()` is a no-op.

## Responsibilities

| Concern | Where |
|---------|-------|
| HTTPS server lifecycle | `webhook.go` |
| Webhook configuration registration | `registration.go` |
| Continuous reconciliation controller | `housekeeper.go` |
| Infrastructure security reconciliation | `infrastructure.go` |
| AdmissionReview types | `admission_review.go` |
| Validation and mutation rule evaluation | `admission_evaluation.go` |
| `/validate` and `/mutate` handlers | `admission.go` |
| Conversion logic | `conversion_logic.go` |
| `/convert` handler | `conversion.go` |
| `/deletion-protection` handler | `deletion_protection.go` |
| `/namespace-protection` handler | `namespace_protection.go` |
| `/strict-mode-protection` handler | `strict_mode_protection.go` |

## Separation from pkg/health

The `HealthServer` in `pkg/health` serves only HTTP probes and Prometheus metrics. The `WebhookServer` in this package serves only HTTPS webhook endpoints.

This separation allows the health server to start first — making `/ready` available immediately — while the webhook server starts after it, once the cluster-facing admission surface is needed.

## Lifecycle

```
NewWebhookServer(kubeClient, katalog, konfig)
    ↓
ws.SetCertManager(certMgr)           ← optional, only when certs were auto-generated
    ↓
ws.SetCertBundle(cert, key, ca, ...)  ← optional, only when certs were auto-generated
    ↓
WireWebhookHousekeeperInfra(ws, kube, kat, kfg)
    ↓                                 ← registers ConversionCRDPatcher + CRDWatcher
ws.Start(ctx)
    • resolves deletion/namespace/strict-mode protection state from Katalog
    • registers HTTPS endpoints based on declared capabilities
    • starts HTTPS server (skipped if no capabilities declared)
    • registers ValidatingWebhookConfiguration / MutatingWebhookConfiguration (in-cluster only)
    • starts housekeeper goroutine
    ↓
ws.Shutdown(ctx)
    • cancels controller goroutine
    • drains HTTPS server
    • removes webhook configurations (when cleanupOnShutdown: true)
    • removes TLS secret (when auto-generated and cleanupOnShutdown: true)
```

## TLS

Certificates are provisioned by `cmd/internal.ensureSecurity` before `Start()` is called. The paths are written to `konfig.Security().Webhooks.TLSCert` and `.TLSKey`. The webhook server reads them from konfig at construction time and uses them for the HTTPS server.

Three modes (handled by `ensureSecurity`, not this package):

1. **External** — user provides `TLS_CERT` / `TLS_KEY` env vars pointing to existing files.
2. **Self-signed** — Orkestra generates a self-signed bundle via `pkg/gateway/certmanager` and stores it in the `orkestra-tls` Secret.

The webhook server itself never generates certificates.

→ Next: [docs/01-architecture.md](docs/01-architecture.md)
