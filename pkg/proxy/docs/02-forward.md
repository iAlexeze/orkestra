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
2. Establishes the portforward via `k8s.io/client-go/tools/portforward`
3. When the connection drops (pod restart, leader election) — detects the error, waits 2 seconds, and re-resolves from step 1
4. Exits cleanly when the context is cancelled (Ctrl+C)

The first successful connection prints `✓ Label  http://localhost:PORT   (pod-name)`. Subsequent reconnects print `[reconnected]`.

## Port conflict detection

Before opening any forward, `CheckPort` attempts to bind the local port. If the port is already in use it prints `✗ Label  port N in use — use --label-port to set an alternative` and returns an error. Other components are not affected.

## Clean shutdown

`RunAll` launches all goroutines then blocks on `<-ctx.Done()`. When the context is cancelled (Ctrl+C or SIGTERM), each goroutine's `stopChan` is closed, which causes `portforward.ForwardPorts()` to return, unwinding the goroutine.
