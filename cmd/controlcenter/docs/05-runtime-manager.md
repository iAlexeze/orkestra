# 05 — Runtime Manager and Configuration

## Environment variables

All configuration is read from the environment at startup by `cc/konfig.go`. There are no config files — the process is configured entirely through env vars.

| Variable | Default | Type | Description |
|----------|---------|------|-------------|
| `PORT` | `8081` | string | HTTP listen port |
| `ORKESTRA_URLS` | `""` | comma-separated | Orkestra runtime URLs to connect to at boot |
| `REFRESH_INTERVAL` | `15` | seconds (int) | How often to poll each runtime |
| `LOG_LEVEL` | `info` | string | Log verbosity (debug, info, warn, error) |
| `IGNORE_DEFAULT` | `false` | bool | If true, do not add `http://localhost:8080` when no URLs are given |
| `ENABLE_RUNTIME_MANAGER` | `true` | bool | Show the runtime manager UI (add/edit/delete instances) |

`ORKESTRA_URLS` is split on commas. Leading and trailing spaces around each URL are trimmed.

## ENABLE_RUNTIME_MANAGER

When `ENABLE_RUNTIME_MANAGER=false`, the runtime manager UI is hidden:
- The sidebar link to "Manage Runtimes" is not rendered.
- The top-bar "Add Runtime" button is not rendered.
- The runtime manager drawer is not rendered.
- The `manageRuntime.js` script is not loaded.

The flag is passed from `main.go` through `cc.Config` and made available to the index template via `IndexData.EnableRuntimeManager`. The template guards each element with `{{ if .EnableRuntimeManager }}`.

The REST API (`/api/instances`) is **not** guarded by this flag — the guard is UI-only. If you need to disable the API itself, add the flag check in `ServeHTTP` before dispatching to the instance handlers.

## Instance persistence

Runtime URLs added through the UI are persisted to `~/.orkestra/instances.json` so they survive process restarts. The file format is:

```json
{"urls": ["http://runtime-a:8080", "http://runtime-b:8080"]}
```

`LoadRuntimeStorage` is called in `cc.New` before the first fetch. Persisted URLs are merged with URLs passed via `ORKESTRA_URLS` — the union is used. Duplicate URLs (by string equality after normalization) are deduplicated.

`SaveRuntimeStorage` is called on every add, update, or delete via the runtime manager UI. It overwrites the file with the current `cc.urls` slice.

URL normalization (`normalizeURL`) adds `http://` if no scheme is present and strips a trailing `/`.

## Runtime manager operations

All mutations go through `cc/manage_runtime.go`:

| Operation | Method | What it does |
|-----------|--------|-------------|
| Add | `AddInstance(rawURL)` | Normalizes URL, checks for duplicate, adds to `cc.instances` and `cc.urls`, persists, then fetches the new instance's Katalog in a background goroutine |
| Delete | `DeleteInstance(rawURL)` | Removes from `cc.instances` and `cc.urls`, persists |
| Update | `UpdateInstance(oldURL, newURL)` | Deletes old, adds new (preserves last Katalog data in the background fetch) |

All three methods hold `cc.mu` (write lock) for their duration. After the lock is released, the next background fetch tick will pick up the changed instance set.

## Adding a new configuration flag

1. Add the env var read in `handleEnvVars()` in `cc/konfig.go`.
2. Add the field to `ControlCenterKonfig`.
3. Pass it into `cc.Config` in `main.go`.
4. Add the field to `cc.Config` in `cc/controlcenter.go`.
5. Thread it to wherever it is needed (handler, view-model, template).
6. If it is a UI feature gate, add it to the relevant `*Data` struct and wrap the template block with `{{ if .FieldName }}`.
