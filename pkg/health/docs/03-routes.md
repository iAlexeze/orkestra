# 03 — HTTP Routes

The `HealthServer` serves two categories of HTTP routes.

## Built-in routes (registered in `Start()`)

| Path | Handler | Description |
|------|---------|-------------|
| `GET /startup` | `startupHandler` | Returns 200 after `SetStartupComplete()` |
| `GET /health` | `healthHandler` | Returns 200 when healthy |
| `GET /ready` | `readyHandler` | Returns 200 when ready |
| `GET /metrics` | `promhttp.Handler()` | Prometheus scrape endpoint |

## External routes (registered via `Register()`)

All `/katalog/…` routes are registered by `cmd/internal/runtime_konstructor.go` before `Start()` is called, using `hs.Register(path, handler)`. This keeps the health package decoupled from the kordinator and katalog packages.

```go
// In runtime_konstructor.go:
hs.Register("/katalog/"+crdName+"/health", kordinator.BuildCRDHealthHandler(...))
hs.Register("/katalog/"+crdName,           kordinator.BuildCRDInfoHandler(...))
hs.Register("/katalog/"+crdName+"/cr",     kordinator.BuildCRListHandler(...))
hs.Register("/katalog/"+crdName+"/cr/",    kordinator.BuildCRDetailAndEventsHandler(...))
hs.Register("/katalog/"+crdName+"/raw",    kordinator.BuildCRDRawHandler(...))
hs.Register("/katalog/"+crdName+"/enriched", kordinator.BuildCRDEnrichedHandler(...))
hs.Register("/katalog/raw",     kordinator.BuildRawKatalogHandler(...))
hs.Register("/katalog/enriched", kordinator.BuildEnrichedKatalogHandler(...))
hs.Register("/katalog",         kordinator.BuildKatalogHandler(...))
```

The `Register()` method wraps each handler in the `logRoutesMiddleware` when debug logging is enabled, adding request duration to each log line.

## Webhook routes

Webhook routes (`/validate`, `/mutate`, `/convert`, `/deletion-protection`, `/namespace-protection`) are served by `pkg/webhook.WebhookServer` on a separate HTTPS port. See [`pkg/webhook/README.md`](../../webhook/README.md).
