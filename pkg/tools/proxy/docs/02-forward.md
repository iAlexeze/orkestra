# Port forwarding

## ForwardTarget

Each component maps to a `ForwardTarget`:

```go
type ForwardTarget struct {
    Label     string // display name printed to the user
    Komponent string // "runtime", "control-center", or "gateway"
    Namespace string
    LocalPort int
    Scheme    string // "http" or "https"
    ViaLease  bool   // true for Runtime: resolve pod via Lease rather than service selector
}
```

## The reconnect loop

Each target runs in a dedicated goroutine via `forwardWithReconnect`. The loop:

1. Calls `resolveTarget` to find the pod to forward to (Lease for Runtime, service selector for CC/Gateway)
2. Calls `startForward` — sets up the SPDY tunnel and returns once the tunnel is ready (`readyChan` signalled), along with a `stop` func and a `done` channel
3. For `ViaLease` targets (Runtime): calls `probeReady`, which polls `scheme://localhost:port/health` until it returns 2xx or the context is cancelled. This guards against the SPDY handshake succeeding against a dead pod while the Lease transition is still in progress.
4. Prints `✓ Label  http://localhost:PORT  (pod-name)` once confirmed connected
5. Starts `watchPod` in a separate goroutine, which polls the pod every 3 seconds. When the pod is no longer Running, it calls `stop()` immediately — proactive disconnect detection without waiting for traffic to surface the broken tunnel.
6. Blocks on `<-done` until the forward drops
7. Prints `↺ Label  reconnecting...  (was pod-name)` and loops from step 1

If a target is not deployed (`FindService` returns nil), the goroutine prints `✗ Label  not deployed in <ns>` and exits. It does not retry — not-deployed is treated as a permanent state for the lifetime of the `ork proxy` invocation.

## Not-deployed exit

`RunAll` exits without waiting for Ctrl+C if every target reports not-deployed. Each goroutine signals a shared `notDeployed` channel on exit; a counter goroutine cancels the shared context when all signals are received.

## Port conflict detection

Before opening any forward, `CheckPort` attempts to bind the local port. If the port is already in use it prints `✗ Label  port N in use — use --label-port to set an alternative` and returns an error before any tunnel is opened.

## Clean shutdown

`RunAll` blocks on `<-ctx.Done()`. On Ctrl+C or SIGTERM the context is cancelled, which stops all `watchPod` goroutines, unblocks `probeReady`, and causes each `startForward` goroutine's context-linked stop to fire, unwinding all SPDY tunnels.
