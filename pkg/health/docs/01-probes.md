# 01 — Probes

`HealthServer` exposes three Kubernetes probe endpoints over HTTP.

## Startup probe — `/startup`

Returns `200` once `SetStartupComplete()` has been called, `503` before that.

Kubernetes uses this to gate liveness and readiness checks. Without a startup probe, a slow-booting operator fails liveness before it ever runs.

```yaml
startupProbe:
  httpGet:
    path: /startup
    port: 8080
  failureThreshold: 30
  periodSeconds: 5
```

## Liveness probe — `/health`

Returns `200` when the controller is healthy, `500` when not.

The `healthy` atomic is set to `true` on `Start()` and never returns to `false` in normal operation. If the process becomes unhealthy (unrecoverable internal state), call `Unhealthy()` — Kubernetes will restart the pod.

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 15
  periodSeconds: 10
```

## Readiness probe — `/ready`

Returns `200` when the controller is ready to serve traffic, `503` during startup and shutdown.

`SetReady()` / `Degraded()` toggle the ready state. The reconciler calls `Degraded()` when informer caches are not yet synced.

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

## Response body

All three endpoints return JSON:

```json
{ 
  "status": "healthy", 
  "service": "my-operator", "uptime": "2h3m", "started": "2025-01-01T10:00:00Z" }
```

The `uptime` field is the duration since `Start()` was called, rounded to the nearest second.

→ Next: [02-webhooks.md](02-webhooks.md)
