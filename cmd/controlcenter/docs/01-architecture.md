# 01 — Architecture

The Control Center is a single Go binary that runs an HTTP server. It holds one `ControlCenter` value in memory, which manages a set of connected Orkestra runtimes and serves all UI pages from embedded assets.

## Key types

| Type | File | Role |
|------|------|------|
| `ControlCenter` | `cc/controlcenter.go` | Central coordinator — holds instances, runs fetch loop, dispatches HTTP |
| `Instance` | `cc/controlcenter.go` | One connected Orkestra runtime: URL, Client, last-fetched Katalog, health |
| `Client` | `cc/client.go` | HTTP client for one runtime — all Orkestra API calls go through here |
| `Config` | `cc/controlcenter.go` | Startup config passed from `main.go` (refresh interval, log level, flags) |
| `ControlCenterKonfig` | `cc/konfig.go` | Environment-variable configuration read at process start |

## Boot sequence

```
main.go
  → NewControlCenterKonfig()         read env vars (PORT, ORK_URLS, …)
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
