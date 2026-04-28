# 01 — Architecture

## Key types

| Type | File | Role |
|------|------|------|
| `WebhookServer` | `webhook.go` | Central struct implementing `domain.Komponent`. Owns the HTTPS server, all handlers, registration, and the controller. |
| `WebhookRegistrationOptions` | `registration.go` | Options for creating webhook configurations (service name, port, CA bundle path, etc.). |
| `NamespaceRules` | `namespace_protection.go` | Per-CRD allow/restrict namespace sets. |

## Boot sequence

```
konstructor.go
  → webhook.NewWebhookServer(kubeClient, katalog, konfig)
      • resolves enablement flags from Katalog
      • initializes registries (admission, conversion)
      • initializes stats instances
      • precomputes protection data (CRD names, namespace rules)
  → ws.SetCertManager(certMgr)       ← if auto-generated certs
  → ws.Start(ctx)
      • sets deletionProtection / namespaceProtection atomic flags
      • (skips if !IsRunningInCluster())
      • registers HTTPS endpoints based on Katalog capabilities
      • starts HTTPS server in goroutine
      • registers webhook configurations in goroutine (best-effort)
      • starts webhookController goroutine (if IsWebhookControllerEnabled)
```

`WebhookServer` starts after `HealthServer` so `/ready` is already live when webhook registration runs.

## Concurrency model

- All atomic booleans (`deletionProtection`, `namespaceProtection`) are written once in `Start()` and read by handlers concurrently.
- Registration and controller goroutines run independently — they share only `kubeClient` (thread-safe) and `katalog` (read-only after construction).
- The controller receives a context derived from `Start(ctx)`. When `Shutdown()` calls `cancel()`, the controller goroutine exits on the next `select`.

## File organisation

Each webhook type is self-contained:

```
webhook.go             — WebhookServer struct + Start/Shutdown/Name/Started
registration.go        — webhook configuration CRUD (ValidatingWebhookConfiguration etc.)
controller.go          — periodic reconciliation loop
admission_review.go    — AdmissionReview / AdmissionRequest / AdmissionResponse types
admission_evaluation.go — validation and mutation rule evaluation logic
admission.go           — /validate and /mutate HTTP handlers
conversion_logic.go    — applyConversion + field resolution helpers
conversion.go          — /convert HTTP handler
deletion_protection.go — /deletion-protection HTTP handler
namespace_protection.go — /namespace-protection HTTP handler + NamespaceRules type
```

→ Next: [02-handlers.md](02-handlers.md)
