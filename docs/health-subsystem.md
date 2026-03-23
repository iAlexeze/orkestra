# Orkestra Health Subsystem

Orkestra exposes a lightweight, concurrency‑safe health server that provides
Kubernetes‑native liveness, readiness, and metrics endpoints. This subsystem is
independent from CRD‑level health and reflects the state of the operator itself.

---

## Overview

The health subsystem is implemented by `HealthServer`, a standalone HTTP server
that exposes:

- **`GET /healthz`** — Orkestra liveness probe  
- **`GET /readyz`** — Orkestra readiness probe  
- **`GET /metrics`** — Prometheus metrics  
- **`/katalog/*`** — dynamically registered CRD endpoints

The server is started early in the operator lifecycle and remains active until
shutdown.

---

## Endpoints

### `GET /healthz` — Orkestra Liveness Probe

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

### `GET /readyz` — Orkestra Readiness Probe

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
- `/healthz`, `/readyz`, `/metrics` are registered  
- CRD routes are registered before startup  
- `started = true`  
- `healthy = true`  
- `ready = false` (until dependency kontroller starts)

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

- `/readyz` returns 503  
- `/healthz` returns 500  
- HTTP server shuts down gracefully  

---

## Dynamic Route Registration

The health server also hosts all CRD‑specific endpoints:

```
/katalog/{crd}
/katalog/{crd}/health
/katalog
```

These are registered via:

```go
hs.Register("/katalog/"+crdName+"/health", BuildCRDHealthHandler(...))
hs.Register("/katalog/"+crdName, BuildCRDInfoHandler(...))
hs.Register("/katalog", BuildKatalogHandler(...))
```

All routes must be registered **before** `Start()`.

More on CRD health [here](./crd-runtime-health.md)
