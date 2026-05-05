# pkg/tunnel

The tunnel package exposes local Kubernetes services to the public internet without touching DNS, firewalls, or cloud infrastructure. It is used by `ork deploy --expose` and `ork tunnel expose` to make an Orkestra app or the Control Center reachable from outside the cluster.

## What lives here

| File | Role |
|------|------|
| `provider.go` | `Provider` interface, auto-selection logic, shared helpers (`orkestaBinDir`, `binaryInPath`) |
| `cloudflare.go` | Cloudflare Quick Tunnels (trycloudflare.com) — no account required |
| `ngrok.go` | ngrok provider — requires a free-tier account and auth token |
| `expose.go` | High-level `Expose()` function, port resolution, `startPortForward`, reuse logic |
| `state.go` | Multi-tunnel daemon state persisted to `~/.orkestra/tunnel-state.json` |

## Quick reference

```go
// Expose an app (auto-detects port, falls back to kubectl port-forward)
url, err := tunnel.Expose(ctx, tunnel.ExposeOptions{
    Name:        "my-app",
    ServiceName: "my-app-orkestra-svc",
    Namespace:   "my-app-orkestra-ns",
    ServicePort: "8080",
})

// Expose the Control Center (always port-forwards, never uses ingress port 80)
url, err := tunnel.Expose(ctx, tunnel.ExposeOptions{
    Name:        "controlcenter",
    ServiceName: doktor.OrkestraControlCenter,   // "orkestra-cc"
    Namespace:   doktor.OrkestraNamespace,
    ServicePort: doktor.OrkestraControlCenterPort, // "8081"
    PortForward: true,
})
```

## Developer documentation

Full documentation is in [docs/](docs/README.md).

| I want to… | Go to |
|-----------|-------|
| Understand the Provider interface and how providers are selected | [01 — Providers](docs/01-providers.md) |
| Understand how tunnel state is persisted and reused | [02 — State](docs/02-state.md) |
| Understand the Expose() function and its options | [03 — Expose](docs/03-expose.md) |
| Understand how the local port is detected or forwarded | [04 — Port Detection](docs/04-port-detection.md) |
| Understand how tunnels survive parent process exit | [05 — Process Survival](docs/05-survival.md) |
