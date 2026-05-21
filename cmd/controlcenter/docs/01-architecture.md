# 01 — Architecture

The Control Center is a single Go binary that runs an HTTP server. It holds one `ControlCenter` value in memory, which manages a set of connected Orkestra runtimes and serves all UI pages from embedded assets.

## Key types

| Type | File | Role |
|------|------|------|
| `ControlCenter` | `cc/controlcenter.go` | Central coordinator — holds instances, runs fetch loop, dispatches HTTP |
| `Instance` | `cc/controlcenter.go` | One connected Orkestra runtime: URL, Client, last-fetched Katalog, health, optional gateway endpoint |
| `Client` | `cc/client.go` | HTTP client for one runtime — all Orkestra API calls go through here |
| `Config` | `cc/controlcenter.go` | Startup config passed from `main.go` (refresh interval, log level, flags) |
| `ControlCenterKonfig` | `cc/konfig.go` | Environment-variable configuration read at process start |

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8081` | HTTP port the control center listens on |
| `ORK_URLS` | — | Comma-separated base URLs of Orkestra runtime instances |
| `REFRESH_INTERVAL` | `15` | Background fetch interval in seconds |
| `LOG_LEVEL` | `info` | Log verbosity |
| `ENABLE_RUNTIME_MANAGER` | `true` | Show the runtime manager UI panel |
| `NO_LOGIN` | `false` | Disable the login page and allow unauthenticated access. |
| `PUBLIC_DEPLOYMENT` | `false` | Shorthand for public read-only mode: implies `NO_LOGIN=true` and `ENABLE_RUNTIME_MANAGER=false`. Individual vars override if set explicitly. |
| `ADMIN_USERNAME` | `orkestra` | Login username (unused when `NO_LOGIN=true`) |
| `ADMIN_PASSWORD` | `orkestra` | Login password (unused when `NO_LOGIN=true`) |
| `SESSION_SECRET` | `dev-secret` | HMAC secret for session tokens (unused when `NO_LOGIN=true`) |
| `GITHUB_CLIENT_ID` | — | Enable GitHub OAuth login |
| `GITHUB_CLIENT_SECRET` | — | GitHub OAuth secret |
| `IGNORE_DEFAULT` | `false` | Do not add `localhost:8080` when no URLs are provided |

## Boot sequence

```
main.go
  → NewControlCenterKonfig()         read env vars (PORT, ORK_URLS, NO_LOGIN, …)
  → LoadRuntimeStorage()             merge persisted URLs from ~/.orkestra/instances.json
  → cc.New(urls, Config{…})          create ControlCenter, one Instance per URL
      → go backgroundFetchLoop()     start background goroutine immediately
  → http.ListenAndServe(…, cc)       serve — ControlCenter is its own http.Handler
```

`New` returns before the first fetch completes. `cc.IsReady()` returns false until at least one Orkestra runtime responds successfully. The index page renders an empty-state view until then.

## Background fetch loop

```
backgroundFetchLoop()
  fetchAllKatalogs()           ← runs immediately on boot
  ticker (RefreshInterval)
    fetchAllKatalogs()         ← runs every N seconds
```

`fetchAllKatalogs` holds `cc.mu` (write lock) for the entire duration. It calls `inst.Client.FetchKatalog()` per instance, updates `inst.Katalog`, sets `inst.Status`, then releases the lock and calls `notifySubscribers()`.

The write lock during fetch is intentional: it ensures the UI never reads a partially updated state. Fetch is fast (one HTTP call per instance) and the refresh interval is 15 s by default, so contention is negligible.

## SSE live updates

Every browser page subscribes to `/controlcenter/sse`. The `ServeSSE` handler registers a buffered `chan struct{}` in `cc.subscribers` and blocks until the client disconnects.

After each `fetchAllKatalogs`, `notifySubscribers` sends a non-blocking signal to every registered channel. The browser-side JS receives the `data: update` event and does a targeted DOM refresh (via `/api/snapshot`) rather than a full page reload.

## Concurrency model

- `cc.mu sync.RWMutex` — all `Instance` reads use `RLock`, all writes (fetch, add/delete instance) use `Lock`.
- `cc.ready atomic.Bool` — read lock-free by health check and `handleIndex`.
- `cc.subscribers sync.Map` — SSE channels; `Store`/`Delete` are concurrent-safe.

→ Next: [02-routing.md](02-routing.md)
