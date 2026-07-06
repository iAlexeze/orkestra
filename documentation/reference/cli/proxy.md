# ork proxy

Forward ports for deployed Orkestra components to localhost.

```bash
ork proxy
ork proxy --for cc
ork proxy --for runtime,cc
ork proxy -n my-platform-ns
```

Inspired by `kubectl proxy`, `ork proxy` replaces manual `kubectl port-forward` invocations when working with a Helm-deployed Orkestra. It discovers each component by its `orkestra.orkspace.io/komponent` label, resolves the right pod, and keeps all forwards alive with automatic reconnection. For the Runtime, it resolves the elected leader via the `orkestra-konductor` Lease so requests always reach the pod with authoritative state.

---

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--for` | | *(all)* | Comma-separated list of components to forward. Omit to forward all three. |
| `--namespace` | `-n` | `orkestra-system` | Namespace where Orkestra is deployed |
| `--runtime-port` | | `8080` | Local port for the Runtime |
| `--cc-port` | | `8081` | Local port for the Control Center |
| `--gateway-port` | | `8443` | Local port for the Gateway |
| `--context` | | *(active context)* | Kubernetes context to use |

---

## Component names (`--for`)

| Value | Aliases | Component |
|-------|---------|-----------|
| `runtime` | `run` | Reconcilers, leader election, `/katalog` API |
| `cc` | `controlcenter`, `control-center` | Control Center web UI |
| `gateway` | `gw` | Gateway, admission webhooks |

---

## Examples

```bash
# Forward all three components
ork proxy

# Control Center only
ork proxy --for cc

# Runtime and Control Center, custom namespace
ork proxy --for runtime,cc -n my-platform-ns

# Use a non-default port for Runtime (e.g. avoid conflict)
ork proxy --runtime-port 9090

# Forward against a specific kubeconfig context
ork proxy --context staging-context
```

---

## What it does

1. Reads the active kubeconfig (respecting `--kubeconfig` and `--context`)
2. Discovers each selected component's Service by the `orkestra.orkspace.io/komponent` label
3. For Runtime: reads the `orkestra-konductor` Lease to resolve the elected leader pod
4. For CC and Gateway: selects a running pod from the Service's selector
5. Pre-checks that each local port is free — errors before opening any tunnel
6. Opens a SPDY port-forward per component and prints the local URL
7. Reconnects automatically if a pod is replaced (rollout, leader failover)
8. Exits cleanly on `Ctrl+C`

---

## Notes

- `ork proxy` requires a Helm-deployed Orkestra. For local development use `ork run` directly.
- The Runtime forward always targets the **leader pod**, not a random replica. This matters when inspecting live reconciler state or the `/katalog` API.
- Port conflicts are reported before any tunnel opens — use `--runtime-port`, `--cc-port`, or `--gateway-port` to remap.
- The Gateway forward uses plain HTTP on the local side; TLS is handled inside the cluster by the Gateway itself.
