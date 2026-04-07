# Orkestra Health Subsystem

Orkestra exposes a lightweight, concurrency‑safe health server that provides
Kubernetes‑native liveness, readiness, and metrics endpoints. This subsystem is
independent from CRD‑level health and reflects the state of the operator itself.

---

## Overview

The health subsystem is implemented by `HealthServer`, a standalone HTTP server
that exposes:

- **`GET /health`** — Orkestra liveness probe  
- **`GET /ready`** — Orkestra readiness probe  
- **`GET /metrics`** — Prometheus metrics  
- **`/katalog/*`** — dynamically registered CRD endpoints

The server is started early in the operator lifecycle and remains active until
shutdown.

---

## Endpoints

### `GET /health` — Orkestra Liveness Probe

Returns:

- **200** when the operator is alive  
- **500** when the operator is unhealthy  

The operator becomes unhealthy when:

- `HealthServer.Unhealthy()` is called  
- shutdown begins  
- a fatal internal error is detected  

Example:

```json
{
  "status": "healthy",
  "service": "orkestra",
  "uptime": "12m",
  "started": "2026-03-20T04:00:00Z"
}
```

---

### `GET /ready` — Orkestra Readiness Probe

Returns:

- **200** when the operator is ready to process events  
- **503** when not ready  

The operator is *not ready* during:

- startup (before informer caches sync)  
- shutdown  
- degraded internal state  

Example:

```json
{
  "status": "ready",
  "service": "orkestra",
  "uptime": "12m",
  "started": "2026-03-20T04:00:00Z"
}
```

---

### `GET /metrics` — Prometheus Metrics

Exposes all registered Prometheus metrics, including:

- reconcile durations  
- reconcile failures  
- queue depth  
- informer lag  
- operator uptime  

This endpoint is automatically registered and requires no configuration.

---

## Lifecycle

### Startup

On `Start()`:

- server binds to the configured port  
- `/health`, `/ready`, `/metrics` are registered  
- CRD routes are registered before startup  
- `started = true`  
- `healthy = true`  
- `ready = false` (until dependency kordinator starts)

### Ready

The operator becomes ready when:

```go
HealthServer.SetReady()
```

This is called after:

- all informers sync  
- all reconcilers start  
- all CRD health trackers are initialized  

### Shutdown

On shutdown:

- `/ready` returns 503  
- `/health` returns 500  
- HTTP server shuts down gracefully  

---

## Dynamic Route Registration

The health server also hosts all CRD‑specific endpoints:

```
/katalog/{crd}
/katalog/{crd}/health
/katalog
```

All routes must be registered **before** `Start()`.

## Related Documentation

- [CRD Health](./crd-runtime-health.md)