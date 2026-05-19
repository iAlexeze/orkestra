# 03 — Gateway

The gateway is a minimal Orkestra process that handles TLS certificate management
and serves admission and conversion webhooks. It does not run reconcilers, hold
informer caches, or compete for the konductor lease.

## What the gateway is

Admission webhooks and conversion webhooks are served over HTTPS. The gateway
owns this surface: it starts the webhook server, loads or generates TLS
certificates, and serves `/validate`, `/mutate`, and `/convert` endpoints.

The health server also runs in the gateway so that Kubernetes readiness probes
work independently of the runtime.

## Why no leader election

Webhook servers are stateless. Every request is self-contained — the webhook
receives an `AdmissionReview` object, evaluates it against the Katalog rules,
and returns a decision. There is no in-memory state that needs to be consistent
across replicas.

This means any number of gateway replicas can run simultaneously. Kubernetes
distributes webhook requests across replicas through the standard `Service`
load-balancing mechanism. If a replica dies, another handles the next request
without any coordination step.

## Cluster-only

The gateway exits immediately if it is not running inside a Kubernetes pod.
`utils.IsRunningInCluster()` checks for the service account token that Kubernetes
injects into every pod. Outside a cluster there is no meaningful webhook endpoint
to serve (no Kubernetes API server to register with), and `ensureSecurity` would
fail without cluster credentials.

For local development, use `ork run` instead.

## Komponent start order

```
1. HealthServer   — /ready and /livez, available from first second of startup
2. WebhookServer  — HTTPS on :8443, /validate, /mutate, /convert
3. Kubeclient     — already started during wiring, managed for Stop()
```

`ensureSecurity` runs before any komponent starts. It generates or loads TLS
certificates and writes the cert and key paths into `kfg` so the webhook server
picks them up when it binds its HTTPS listener. See [04-security.md](04-security.md)
for the full security wiring sequence.

→ Back: [02-runtime.md](02-runtime.md)
→ Next: [04-security.md](04-security.md)
