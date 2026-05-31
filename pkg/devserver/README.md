# pkg/devserver

A lightweight mock HTTP server for local development. Started via `ork run --dev-server`, it binds on `:9999` and serves all endpoints used by the `external:` examples — no real upstream services, no internet, no tokens required.

## Starting it

```bash
ork run --dev-server
```

The server starts before the Orkestra runtime and prints its endpoint list in the startup banner. It runs for the lifetime of the process.

## Endpoints

| Route | Method | Used by |
|---|---|---|
| `/health` | GET | 01-health-gate (healthy), full-stack gated-app |
| `/status/200` | GET | 01-health-gate (httpbin-compatible) |
| `/status/503` | GET | 01-health-gate (degraded CR) |
| `/config/:name` | GET | 02-config-inject |
| `/sign` | POST | 03-image-signing |
| `/auth/token` | POST | 04-chained (first call) |
| `/resources/:name` | GET | 04-chained (second call) |
| `/flags/:name` | GET | full-stack gated-app (feature flags) |

All responses are `application/json`. All handlers return immediately — no sleep, no blocking, no state.

See [docs/01-endpoints.md](docs/01-endpoints.md) for full request/response shapes.

## Dev CRs

Each external example ships a `cr-dev.yaml` (or `cr-dev-healthy.yaml` / `cr-dev-degraded.yaml` for the health gate) that points `spec.serviceUrl` at `http://localhost:9999`. Apply them instead of the production CRs when running locally.

## Extending

Add new routes in `handlers.go` and register them in `registerHandlers`. The dev server is designed to grow as new external examples are added — each new example pattern gets its endpoint here.
