# 03 — The Expose Function

## Overview

`Expose` is the primary entry point for the tunnel package. It handles the full lifecycle: detect an existing live tunnel → reuse it, or select a provider → install if needed → resolve the local port → start the daemon → wait for a live URL → persist state.

```
Expose(ctx, opts)
  │
  ├── LoadTunnelState(name)
  │     └── IsAlive() && isTCPListening(localPort)?  → return cached URL
  │
  ├── Select() or SelectByName()
  │
  ├── p.Install(ctx)          if !p.Available()
  ├── p.Authenticate(ctx)     if opts.Token != ""
  │
  ├── resolveLocalPort()      if opts.LocalPort == 0
  │     ├── port 80 listening?  → use 80 (ingress path)
  │     └── startPortForward()  → deterministic port 19000–19999
  │
  ├── p.Start(ctx, localPort)
  │     └── blocks until URL is DNS-live (see doc 05)
  │
  └── SaveTunnelState(name, state)
        └── return url
```

## ExposeOptions

```go
type ExposeOptions struct {
    // Name identifies the tunnel in the state map ("my-app", "controlcenter").
    // Falls back to ServiceName when empty.
    Name string

    // Provider is the explicit provider name ("cloudflared" or "ngrok").
    // Empty means auto-select.
    Provider string

    // Token is passed to Provider.Authenticate when non-empty.
    Token string

    // LocalPort overrides auto-detection when non-zero.
    LocalPort int

    // ServiceName is the Kubernetes service for port-forwarding
    // (e.g. "my-app-orkestra-svc", "orkestra-cc").
    ServiceName string

    // Namespace is the target namespace for the service port-forward.
    Namespace string

    // ServicePort is the container port on the service (default "80").
    ServicePort string

    // PortForward, when true, skips port 80 detection and always starts a
    // kubectl port-forward. Use for services never behind an ingress
    // (e.g. the Orkestra Control Center).
    PortForward bool
}
```

## Naming convention

The `Name` field is the key in the state map and is shown in `ork tunnel status` output. Keep names short and stable — they determine the deterministic local port via `portForName` (see [04 — Port Detection](04-port-detection.md)).

Callers set these names:

| Caller | Name | Why |
|--------|------|-----|
| `exposeApp()` in deploy.go | `<appName>` (e.g. `"my-app"`) | One tunnel per deployed app |
| `exposeControlCenter()` in deploy.go | `"controlcenter"` | Fixed — there is only one CC |
| `tunnelExposeCmd` in tunnel.go | user-provided or `"controlcenter"` | User selects which tunnel to manage |

## Port selection

When `ServiceName` and `Namespace` are provided, `Expose` always uses `kubectl port-forward` to reach the service directly. Port 80 (host-mapped ingress) is never used. See [04 — Port Detection](04-port-detection.md) for the full reasoning.

The `PortForward` flag is kept for compatibility but has no effect when `ServiceName` is set — port-forward is the default for all calls that supply service information.

Port 80 is only used as a last resort when neither `ServiceName` nor `Namespace` is provided. This path exists for manual or scripted tunnels where the caller only knows a port.
