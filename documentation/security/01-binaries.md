# Binaries & build tags

Orkestra ships as three distinct compiled binaries from a single codebase. Go build tags separate them at compile time — not at runtime, not behind a flag. Code that is not compiled in cannot be exploited.

---

## The three binaries

| Binary | Entry point | What it does |
|--------|-------------|-------------|
| **ork** (developer CLI) | `ork run`, `ork generate`, `ork init`, … | Everything. Used locally, never in production. |
| **ork** (runtime) | `ork run` only | Watches CRDs and reconciles. No generation surface. |
| **ork-gateway** | `ork gate` only | Serves webhooks, TLS, admission. No reconcile surface. |

All three binaries are compiled from the same Go source files. The `runtime` and `gateway` build tags strip out the commands that are not relevant to each role.

---

## Why this matters

A traditional operator binary in production exposes everything the developer CLI can do. An attacker who can exec into the container — or who can exploit a vulnerability in the running process — can potentially run `generate rbac`, `init`, `template`, or any other command the binary offers.

Orkestra's approach: the production runtime binary knows only how to run. The production gateway binary knows only how to gate. Neither can generate RBAC bundles, scaffold operators, enumerate registered CRDs, or exfiltrate Katalog definitions. The code does not exist in the binary.

This is a structural guarantee, not a permissions check.

---

## Build tag reference

### `//go:build runtime`

Files tagged `runtime` are **included only in the runtime binary**. This is primarily `cmd/cli/run.go` — the `ork run` command itself.

### `//go:build !runtime`

Files tagged `!runtime` are **excluded from the runtime binary** but included in the developer CLI. All developer-only commands (`generate`, `init`, `validate`, `template`, `diff`, `simulate`, …) use this tag.

### No tag

Files with no build tag are compiled into all three binaries. Core types, the logger, domain models, and shared utilities live here.

### `//go:build gateway`

The gateway binary is compiled separately (`cmd/gateway/`). It receives only the webhook server, TLS management, and admission handlers.

---

## Command matrix

| Command | Developer CLI | Runtime binary | Gateway binary |
|---------|:---:|:---:|:---:|
| `ork run` | ✓ | ✓ | — |
| `ork gate` | — | — | ✓ |
| `ork version` | ✓ | — | — |
| `ork generate bundle` | ✓ | — | — |
| `ork generate rbac` | ✓ | — | — |
| `ork generate crd` | ✓ | — | — |
| `ork validate` | ✓ | — | — |
| `ork init` | ✓ | — | — |
| `ork template` | ✓ | — | — |
| `ork diff` | ✓ | — | — |
| `ork simulate` | ✓ | — | — |
| `ork e2e` | ✓ | — | — |
| `ork notes` | ✓ | — | — |

---

## Distroless container image

Both production binaries run in a distroless base image:

```dockerfile
FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETARCH
COPY ork-${TARGETARCH} /usr/local/bin/ork

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/ork"]
```

Distroless images contain no shell, no package manager, no curl, no tar, no standard Unix utilities. If an attacker gains code execution in the container, they have a single static binary and nothing else to pivot with. There is no `sh`, no `bash`, no `python`, no way to download additional tools.

The image runs as a non-root user (UID 65532) and is not given any Linux capabilities.

---

## Pod security context

The [Helm chart](https://artifacthub.io/packages/helm/orkestra/orkestra) applies a hardened security context to every pod by default:

```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 1000
  capabilities:
    drop:
      - ALL
```

- **`runAsNonRoot`** — the kernel refuses to start the container if the image would run as root.
- **`readOnlyRootFilesystem`** — the container cannot write to its own filesystem. No log files, no temp files, no config mutations at runtime.
- **`allowPrivilegeEscalation: false`** — the process cannot gain more privileges than it started with (no setuid, no `sudo`).
- **`capabilities: drop: ALL`** — every Linux capability is dropped. The process has only the default ambient set (effectively none for a static binary).
- **`seccompProfile: RuntimeDefault`** — the container runtime's built-in seccomp profile filters syscalls to the set a typical server process needs.

These are on by default. You do not need to configure anything to get them.

---

## Development workflow with gateway-only

For local development, you do not need to run the full stack inside Kubernetes. The recommended workflow is:

1. Deploy the Gateway only using Helm (set `runtime.enabled: false`):

   ```bash
   helm upgrade --install orkestra orkestra/orkestra \
     --set runtime.enabled=false \
     --set gateway.enabled=true \
     -n orkestra-system
   ```

2. Run the runtime locally in your terminal:

   ```bash
   ork run --file katalog.yaml
   ```

The Gateway continues serving webhooks to the cluster. `ork run` connects as a companion runtime. Admission, deletion protection, and namespace protection all work exactly as they would in production. There are no blockers — you can iterate on your Katalog and test the full security contract locally without rebuilding or redeploying the Gateway.

### Runtime ↔ Gateway communication

The runtime and gateway do not talk to each other directly. The runtime is configured with the Gateway's base URL:

```yaml
# katalog.yaml
gateway:
  endpoint: "http://orkestra-gateway.orkestra-system.svc:8080"
```
OR
```yaml
# values.yaml
runtime:
  gatewayEndpoint: "http://orkestra-gateway.orkestra-system.svc:8080"
```


The runtime includes this URL in its `/katalog` API response. When the Control Center queries the runtime for state, it reads the gateway endpoint from that response and calls the Gateway directly to fetch webhook statistics (admission decisions, deletion protection events, strict-mode blocks). The runtime never proxies this traffic — it just advertises where the Gateway lives so the Control Center can pull from both sources and merge the view.

---
