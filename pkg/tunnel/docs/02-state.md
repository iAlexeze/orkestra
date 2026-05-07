# 02 — Tunnel State

## Why persist state at all

Tunnel daemons are background processes that outlive the CLI command that started them. Without persistence, `ork tunnel status` would have no way to list running tunnels, and `ork tunnel stop` would not know which PID to kill. State also enables reuse: if the same app is deployed again, `Expose` can detect the existing live tunnel and skip starting a new one.

## State file

All tunnel entries are stored in a single JSON file:

```
~/.orkestra/tunnel-state.json
```

The file is a `map[name]State` — one entry per named tunnel. Names are user-visible identifiers like `"my-app"` or `"controlcenter"`. Permissions are `0600` (owner-only) because the file can contain port numbers and PIDs.

## The State struct

```go
type State struct {
    Name           string    // tunnel name — same as the map key
    Provider       string    // "cloudflared" or "ngrok"
    PID            int       // cloudflared or ngrok daemon PID
    PortForwardPID int       // kubectl port-forward PID (0 when port 80 was used)
    URL            string    // public URL, e.g. https://frog-dream.trycloudflare.com
    LocalPort      int       // local port the tunnel daemon forwards from
    StartedAt      time.Time // used for Uptime()
}
```

`PortForwardPID` is non-zero when the port-forward subprocess was started. Both `PID` and `PortForwardPID` must be killed when stopping the tunnel. If only the tunnel daemon is killed, the port-forward becomes a zombie.

## Reuse guard

`Expose` only reuses an existing state entry when **both** conditions hold:

```go
if existing.IsAlive() && isTCPListening(existing.LocalPort) {
    return existing.URL, nil
}
existing.Stop()
```

`IsAlive()` sends `Signal(0)` to the PID — a zero-signal that checks process existence without actually sending a signal. `isTCPListening` dials `127.0.0.1:<port>` with a 200 ms timeout. Both checks are necessary:

- **PID alive but port not listening**: the port-forward target (pod) was restarted; cloudflared is forwarding to a dead port. The tunnel shows as online but serves 502s.
- **Port listening but PID dead**: the previous state file is stale; a different process now owns the port. We should not reuse the URL from the state file.

A state entry that fails either check is treated as stale: `Stop()` is called (which terminates any remaining processes and deletes the state entry), then a fresh tunnel is started.

## Lifecycle functions

| Function | What it does |
|----------|--------------|
| `SaveTunnelState(name, state)` | Upserts one entry; creates the file if missing |
| `LoadTunnelState(name)` | Returns one entry, nil if not found |
| `LoadAllStates()` | Returns the full map; used by `ork tunnel status` |
| `RemoveTunnelState(name)` | Deletes one entry; removes the file if the map becomes empty |
| `RemoveAllStates()` | Deletes the entire state file |
| `state.Stop()` | SIGTERMs both PIDs and calls `RemoveTunnelState` |
| `state.IsAlive()` | `Signal(0)` check on tunnel PID |
| `state.Uptime()` | Human-readable duration since `StartedAt` |
