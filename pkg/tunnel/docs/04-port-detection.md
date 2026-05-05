# 04 — Port Detection

## The problem

A tunnel daemon needs a local port to forward from. Kubernetes services are not directly reachable from localhost on the host machine without either:

- Host port mapping (e.g. the kind ingress controller mapping host port 80)
- `kubectl port-forward` creating a localhost proxy

Picking the wrong local endpoint leads to 502/530 errors even when the tunnel itself is healthy.

## Port resolution priority

```
resolveLocalPort(name, opts)
  │
  ├── ServiceName != "" && Namespace != ""?
  │     → startPortForward(...)  → deterministic port 19000-19999, pfPID
  │     "direct service path — bypasses ingress, survives pod restarts"
  │
  ├── isTCPListening(80)?        → return 80, pfPID=0
  │     "no service info — fall back to host-mapped ingress port"
  │
  └── error: "no service reachable on port 80"
```

Port-forward is always preferred when `ServiceName` and `Namespace` are provided. This is the case for all Orkestra-deployed apps and the Control Center.

**Why port-forward is preferred over port 80**: port 80 on the host connects through the full ingress stack (kind-ingress → ingress controller → service → pod). If the backend pod restarts or the ingress becomes temporarily unavailable, port 80 stays "listening" (the ingress controller itself is still up). cloudflared cannot detect the broken origin and keeps serving 502s. The reuse guard (`isTCPListening(80)`) also returns true, so no fresh tunnel is started — the broken state persists until manual restart.

Port-forward connects directly to the K8s service, which routes to whatever pod is currently running. When the port-forward dies, `isTCPListening(localPort)` immediately returns false, triggering a fresh tunnel on the next `Expose` call.

Port 80 is retained only as a last-resort fallback when neither `ServiceName` nor `Namespace` is provided.

## Why NodePort is not used

The ingress controller's NodePort (e.g. 32352) is reachable **inside the Docker network** only. From the host machine, `127.0.0.1:32352` is not the kind cluster's node — it is the host itself. `isTCPListening(nodePort)` would return false, and even if it returned true (due to a coincidental listener), it would be the wrong service.

NodePort was removed from port detection entirely. Only port 80 (host-port-mapped) or a port-forward proxy is used.

## kubectl port-forward

When port 80 is not available, `startPortForward` starts a detached `kubectl port-forward` subprocess:

```go
cmd := exec.Command(
    "kubectl", "port-forward",
    "-n", namespace,
    "svc/"+serviceName,
    fmt.Sprintf("%d:%s", localPort, servicePort),
)
cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
```

`Setsid: true` places the subprocess in its own session. Without this, when the CLI process exits, the kernel delivers SIGHUP to the process group, killing the port-forward and severing the tunnel's local endpoint. See [05 — Process Survival](05-survival.md).

After starting the subprocess, the code polls `isTCPListening(localPort)` for up to 8 seconds with 150 ms intervals. It also checks `proc.Signal(syscall.Signal(0))` on each iteration to detect early exits (pod not running, service has no endpoints, namespace does not exist).

If the port is not bound within 8 seconds but the process is still alive, the function returns anyway. cloudflared will retry the origin until it becomes available.

## Deterministic local port

Multiple apps can be tunneled simultaneously. Each needs a unique local port so the port-forward proxies do not conflict.

`portForName` maps a tunnel name to a stable port in the range 19000–19999 using a polynomial hash:

```go
func portForName(name string) int {
    h := 0
    for _, c := range name {
        h = h*31 + int(c)
    }
    if h < 0 {
        h = -h
    }
    return 19000 + (h % 1000)
}
```

The same name always produces the same port. If two tunnel names happen to hash to the same port (unlikely in practice), the second `startPortForward` will fail because the port is already bound — which surfaces as a clear error rather than silent data corruption.

## isTCPListening

```go
func isTCPListening(port int) bool {
    conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
    if err != nil {
        return false
    }
    conn.Close()
    return true
}
```

200 ms is long enough to distinguish "nothing listening" from "listener present but slow to accept". It is used both for port 80 detection and for polling the port-forward proxy readiness.
