# pkg/health

The health package owns the HTTP server that exposes Orkestra's lifecycle to Kubernetes. It has one responsibility: serve health and readiness probes so the cluster can gate traffic, restarts, and rolling updates correctly.

All webhook logic — admission, conversion, deletion protection, and namespace protection — lives in [`pkg/webhook`](../webhook/README.md).

## Responsibilities

| Concern | Where |
|---------|-------|
| Startup probe (`/startup`) | `handler.go` |
| Liveness probe (`/health`) | `handler.go` |
| Readiness probe (`/ready`) | `handler.go` |
| Prometheus metrics (`/metrics`) | `health.go` (via `promhttp.Handler`) |
| Katalog API routes (`/katalog/…`) | Registered externally by `cmd/internal/runtime_konstructor.go` |
| Conversion / admission stats types | `conversion_stats.go`, `admission_stats.go`, etc. |

## Server lifecycle

```
NewHealthServer(konfig)
    ↓
hs.Register(path, handler)   ← called by runtime_konstructor.go for all /katalog routes
    ↓
hs.Start(ctx)
    • binds HTTP port  → /startup /health /ready /metrics + /katalog routes
    ↓
hs.Shutdown(ctx)
    • marks not-ready, drains HTTP server
```

## Probe semantics

| Endpoint | Returns 200 when… |
|----------|-------------------|
| `/startup` | `SetStartupComplete()` has been called |
| `/health` | `healthy` atomic is true |
| `/ready` | `ready` atomic is true — false during boot and after shutdown begins |

`SetStartupComplete()` is called at the end of `Start()`. `Degraded()` and `SetReady()` are called by the dependency kordinator as informer caches sync and workers start.

## Stats types

The stats types (`ConversionStats`, `AdmissionStats`, `DeletionProtectionStats`, `NamespaceProtectionStats`, `WebhookStats`) live in this package because the kordinator reads them when building the `/katalog/{crd}` API response. The webhook package writes to these instances; the health package only defines the types and exposes read snapshots.

→ Next: [docs/01-probes.md](docs/01-probes.md)
