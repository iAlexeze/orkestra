# ork doctor + ork doctor deploy --expose — Design Document

*Orkestra Project — April 2026*

---

## Overview

This document covers two related features:

1. **`ork doctor` expansions** — notification detection, dependency auto-install,
   git/license metadata, kind cluster without requiring Go.

2. **`ork doctor deploy --expose`** — instant public URL for local and dev clusters
   via a managed tunnel daemon.

Both features serve the same goal: a developer with a Dockerfile and a `.env`
reaches a running, observable, shareable production deployment with as close to
zero configuration as possible.

---

## Part 1: ork doctor expansions

### 1.1 Rename: doctor → doctor

All references to `ork doctor` become `ork doctor` (k-theme, consistent with
Kubernetes, Katalog, Komposer, Kordinator). The old name is kept as a hidden
alias for backward compatibility through v1.

```bash
ork doctor              # replaces: ork doctor
ork doctor init         # replaces: ork doctor init
```

### 1.2 Git metadata and license extraction

`ork doctor init` reads two fields from the project before generating the Katalog:

**Author** — from git config `user.email`:
```bash
git config user.email   # → developer@example.com
```

**License** — from the LICENSE file in the project root:
- Detect by filename: `LICENSE`, `LICENSE.md`, `LICENSE.txt`
- Extract identifier from first line: `MIT License` → `MIT`, `Apache License 2.0` → `Apache-2.0`
- If no LICENSE file: `meta.license` is omitted

These populate Katalog metadata:

```yaml
metadata:
  name: my-app
  description: Orkestra HA deployment for my-app
  author: developer@example.com   # from git config user.email
  license: MIT                    # from LICENSE file
```

The developer contributes nothing extra. The Katalog is self-describing from
information the project already contains.

### 1.3 Notification detection and --notify-me

`ork doctor` scans the `.env` file for notification credentials. When found,
it suggests the `--notify-me` flag at the end of its output:

**Detected patterns:**

| Pattern | Channel |
|---|---|
| `SMTP_HOST`, `SMTP_USER`, `SMTP_PASS` | Email |
| `SLACK_WEBHOOK_URL`, `SLACK_BOT_TOKEN` | Slack |

**Detection output** (shown only when credentials are found):

```
  ✓ Slack credentials found (SLACK_WEBHOOK_URL)

💡 Run 'ork doctor init --notify-me' to get deployment alerts in Slack
```

**`--notify-me` behavior:**

Adds a `notification:` block to the generated Katalog using the credentials
found in `.env`. The developer's git `user.email` becomes the default email
recipient. Slack channels are inferred from the webhook URL's description if
available, otherwise defaults to a sensible channel name.

```yaml
notification:
  enabled: true
  defaults:
    interval: 15m
    slackWebhookUrl: "{{ env.SLACK_WEBHOOK_URL }}"
  teams:
    developer:
      email: ["developer@example.com"]   # from git config user.email
      slack: ["#deployments"]
      interval: 5m
      message: >
        {{ .metadata.name }} in {{ .metadata.namespace }}:
        {{ conditionMessage .children.deployment "Available" }}
```

**Notification conditions** — added automatically to operatorBox:

```yaml
onReconcile:
  when:
    - field: metrics.errorRatePercent
      greaterThan: "5"
      notify: [developer]     # fires when error rate exceeds 5%

    - field: "{{ replicasReady .children.deployment }}"
      equals: "false"
      notify: [developer]     # fires when replicas are not all ready
```

The developer receives Slack/email alerts:
- When their deployment is failing (error rate > 5%)
- When pods are not ready (crash loop, OOM, image pull failure)
- Automatically silenced after 15 minutes if the issue persists (prevents spam)

**Important:** The SMTP and Slack credentials stay in the cluster Secret —
they are read from the `.env` and included in the generated Secret bundle,
same as any other secret. They are not hardcoded in the Katalog. The
notification block references them via `env.*` template expressions which
resolve at runtime.

### 1.4 Dependency auto-install

`ork doctor` checks for required tools and installs missing ones automatically.
Each installation is shown as a status line.

**Tool dependency matrix:**

| Tool | Required for | Install method |
|---|---|---|
| `kubectl` | All deployments | Download binary to `~/.orkestra/bin/` |
| `helm` | All deployments | Download binary to `~/.orkestra/bin/` |
| `kind` | `--dev` flag | Download binary to `~/.orkestra/bin/` |
| `docker` | Build + push | Not auto-installed — required, user must install |
| `metrics-server` | HPA | `helm install metrics-server` in cluster |
| `ingress-nginx` | Host/Ingress | `helm install ingress-nginx` in cluster |

**`kubectl` and `helm`** are downloaded as static binaries — no package manager,
no sudo required. Binaries go to `~/.orkestra/bin/`. The deploy commands add
this directory to their child process PATH automatically.

**`kind`** is also a static binary. `ork doctor deploy --dev` downloads it to
`~/.orkestra/bin/kind`. **Go is not required.** The earlier design required
`go install kind` — this is replaced with a direct binary download. Most
developers who want a local cluster do not have Go installed.

**`docker`** is explicitly not auto-installed. The developer's Docker setup
involves credentials, daemon configuration, and potentially Desktop licensing.
If docker is not found, `ork doctor` exits with a clear message:

```
  ✗ Docker not found — install Docker Desktop or Docker Engine first
    https://docs.docker.com/get-docker/
```

**`metrics-server`** is installed when HPA is enabled (default). Detected by
checking if `metrics.k8s.io/v1beta1` API group is available:

```bash
kubectl api-resources --api-group=metrics.k8s.io 2>/dev/null | grep -q nodes
```

If not available and HPA is in the Katalog:
```
  → Installing metrics-server (required for HPA)...
  ✓ Metrics server ready
```

**`ingress-nginx`** is installed when `host:` is set in `.orkestra/cr.yaml`.
Already documented in the previous design — included here for completeness.

**Installation output** during `ork doctor deploy`:

```
Checking dependencies...
  ✓ docker 27.1.0
  → kubectl not found — installing...
  ✓ kubectl v1.31.0 installed (~/.orkestra/bin/kubectl)
  ✓ helm v3.16.0
  ✓ Cluster: my-cluster (1 node)
  → metrics-server not found — installing...
  ✓ Metrics server ready
```

### 1.5 `ork doctor deploy --dev` — kind cluster without Go

```bash
ork doctor deploy --dev --registry ghcr.io/myorg
```

When `--dev` is specified:

1. Check if `kind` is in PATH or `~/.orkestra/bin/kind`
2. If not found: download the kind binary for the current OS/arch
3. Check if a kind cluster already exists (`kind get clusters`)
4. If not: `kind create cluster --name orkestra-dev`
5. Set kubectl context to `kind-orkestra-dev`
6. Continue with normal deploy flow

The kind cluster creation is one-time. Subsequent `ork doctor deploy --dev` calls
detect the existing cluster and skip creation.

```
  → Creating local cluster (kind)...
  ✓ Cluster ready: kind-orkestra-dev
```

**Go is not required.** Kind is distributed as a static binary. Downloaded
from `https://github.com/kubernetes-sigs/kind/releases/` directly.

---

## Part 2: ork doctor deploy --expose

### 2.1 Goal

After `ork doctor deploy`, the application is running in the cluster. On a local kind
cluster, it is only accessible via localhost port-forwarding. On a remote
cluster with no public Ingress IP, the same problem exists.

`--expose` starts a background tunnel daemon that creates a public HTTPS URL
pointing at the application. The URL survives the deploy command's exit.

```bash
ork doctor deploy --dev --expose
#   ✓ App live at https://abc123.ngrok.io
#   ✓ Tunnel: running (ork tunnel stop to end)
```

### 2.2 Tunnel provider selection

Two providers are supported in v1. Default selection is automatic based on
what is installed:

| Provider | Account required | Free tier | Auto-detect |
|---|---|---|---|
| `cloudflared` | No | Unlimited (trycloudflare.com) | Yes |
| `ngrok` | Yes (free) | 40 req/min | Yes |

**Provider priority:** cloudflared first (no account), ngrok second.

The developer can override with `--tunnel-provider`:
```bash
ork doctor deploy --expose --tunnel-provider ngrok
```

**Cloudflare Tunnel (default):**
```bash
cloudflared tunnel --url http://localhost:80
```
No account. No token. Anonymous URL at `*.trycloudflare.com`. HTTPS included.
No rate limits on the free tier. The URL changes on every tunnel start.

**ngrok:**
```bash
ngrok http 80 --log=stdout --log-format=json
```
Requires an ngrok account and auth token (free). Rate-limited on free tier.
Stable URL possible with paid account (out of scope for v1).

### 2.3 Tunnel as a background daemon

The tunnel must outlive the `ork doctor deploy` command. It runs as a detached
background process.

**Daemon lifecycle:**

```
ork doctor deploy --expose
  ↓
Start tunnel daemon (detached process)
Write PID + URL to ~/.orkestra/tunnel-state.json
Print URL
Exit deploy command

↓ (daemon continues running)

ork tunnel status     → shows URL, provider, uptime
ork tunnel stop       → kills daemon, removes state file
```

**`~/.orkestra/tunnel-state.json`:**

```json
{
  "provider": "cloudflared",
  "pid": 12345,
  "url": "https://abc123.trycloudflare.com",
  "localPort": 80,
  "startedAt": "2026-04-19T10:23:00Z"
}
```

`ork doctor deploy --expose` checks for an existing running daemon before starting
a new one:
- If running and pointing at the same local port → print existing URL, skip start
- If running but stale (PID dead) → clean up and start fresh
- If not running → start new daemon

### 2.4 Local port detection

The tunnel forwards to the ingress controller's local port. Detection order:

1. `kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath={.spec.ports[?(@.name=="http")].nodePort}` → kind NodePort
2. Port-forward if NodePort is not available: `kubectl port-forward -n ingress-nginx svc/ingress-nginx-controller 8080:80` → use 8080
3. Direct service port-forward if no ingress: `kubectl port-forward -n <ns> svc/<name>-svc <localPort>:<port>`

For kind clusters, the NodePort is the most reliable path. `ork doctor deploy --dev`
creates the kind cluster with the ingress controller already configured to
expose NodePort 80.

### 2.5 CLI commands

**`ork doctor deploy --expose`** — start tunnel as part of deploy:
```bash
ork doctor deploy --registry ghcr.io/myorg --expose
ork doctor deploy --dev --expose                         # kind + tunnel
ork doctor deploy --dev --expose --tunnel-provider ngrok # explicit provider
ork doctor deploy --dev --expose --tunnel-token $NGROK_TOKEN  # non-interactive
```

**`ork tunnel status`** — show current tunnel state:
```bash
ork tunnel status

  Provider: cloudflared
  URL: https://abc123.trycloudflare.com
  Local: http://localhost:80
  Uptime: 23m
  Status: running
```

**`ork tunnel stop`** — stop the tunnel daemon:
```bash
ork tunnel stop

  ✓ Tunnel stopped
```

**`ork tunnel restart`** — stop and start fresh (new URL):
```bash
ork tunnel restart

  ✓ Tunnel restarted
  ✓ New URL: https://xyz789.trycloudflare.com
```

### 2.6 Token storage

ngrok tokens are stored in `~/.orkestra/tunnel.yaml` at permissions 0600:

```yaml
providers:
  ngrok:
    authToken: "2abc3def..."
```

Cloudflared requires no token storage (anonymous tunnels).

On first use with ngrok and no stored token:
```
ngrok auth token required. Get yours at https://dashboard.ngrok.com/get-started/your-authtoken
Token: [prompt]
  ✓ Token saved to ~/.orkestra/tunnel.yaml
```

### 2.7 Deploy output with --expose

```
Building my-app...
  → ghcr.io/myorg/my-app:a3f5c2b
  ✓ Built (18s)
  ✓ Pushed

Applying to cluster...
  ✓ Bundle applied
  ✓ Image: ghcr.io/myorg/my-app:a3f5c2b

Waiting for deployment...
  ✓ 2/2 pods ready

Starting tunnel...
  ✓ Tunnel: https://abc123.trycloudflare.com (cloudflared)

  App:     https://abc123.trycloudflare.com
  Status:  Ready
  Image:   ghcr.io/myorg/my-app:a3f5c2b
  Commit:  a3f5c2b

  Control Center → ork control start
  Tunnel        → ork tunnel status
  Stop tunnel   → ork tunnel stop
```

### 2.8 Provider interface

```go
// pkg/tunnel/provider.go

type Provider interface {
    // Name returns the provider name for display and config.
    Name() string

    // Available returns true when the provider binary is installed
    // or can be downloaded.
    Available() bool

    // Install downloads and installs the provider binary if missing.
    Install(ctx context.Context) error

    // Authenticate configures credentials if required.
    // No-op for providers that don't require authentication.
    Authenticate(ctx context.Context, token string) error

    // Start starts the tunnel and returns the public URL.
    // The tunnel runs as a background daemon; Start returns once
    // the URL is available (not when the tunnel exits).
    Start(ctx context.Context, localPort int) (url string, pid int, err error)

    // Stop kills the tunnel process.
    Stop(pid int) error
}
```

Implementations:
- `pkg/tunnel/cloudflare.go` — `cloudflaredProvider`
- `pkg/tunnel/ngrok.go` — `ngrokProvider`

### 2.9 Security considerations

- Tokens stored at `~/.orkestra/tunnel.yaml` with 0600 permissions
- Tunnel processes run as child processes owned by the user
- No shared tokens — each developer uses their own ngrok account
- Cloudflare anonymous tunnels expose nothing beyond what the running service exposes
- `ork tunnel stop` kills the process cleanly — no lingering tunnels on exit
- Tunnel state file is cleaned up on stop and on stale PID detection

---

## Part 3: Developer example pack

### Pack name: `developer`

Focused on local-to-production deployment. No Kubernetes knowledge required.

```bash
ork init my-app --pack developer
```

### Examples

| Example | What it demonstrates |
|---|---|
| `01-single-project` | Zero to running: one app, one command |
| `02-multi-project` | Two services, shared cluster, internal DNS |
| `03-deletion-protection` | Security block, accidental deletion prevention |
| `04-notify-me` | Slack/email alerts on deployment failures |
| `05-rollback` | Bad deploy → instant rollback to previous image |

### Positioning within the pack hierarchy

```
beginner      → learn Kubernetes operators
intermediate  → multi-resource patterns
advanced      → hooks, constructors, registries
use-cases     → full-stack platform patterns
developer     → local to production, no Kubernetes required
```

The developer pack is the only pack that assumes no Kubernetes knowledge.
It is the entry point for the third Orkestra audience. Every example uses
`ork doctor`, `ork doctor deploy`, and the ConfigMap-as-CRD pattern. No custom CRDs.
No Go. No Helm charts written by the developer.

---

## Part 4: Full developer flow (revised)

```
Installation
  brew install orkspace/tap/ork     # or curl install script
  # kubectl, helm, kind installed automatically on first use

New project
  cd my-app/
  ork doctor
  # → detects language, port, .env, Slack credentials
  # → suggests --notify-me

  ork doctor init --notify-me
  # → generates .orkestra/katalog.yaml with notification block
  # → generates .orkestra/cr.yaml

Local deploy
  ork doctor deploy --dev --expose
  # → downloads kind if needed, creates cluster
  # → builds image, deploys to kind cluster
  # → installs metrics-server, ingress-nginx automatically
  # → starts cloudflared tunnel
  # → prints: https://abc123.trycloudflare.com

Second project (same cluster)
  cd my-api/
  ork doctor init
  ork doctor deploy --dev --expose
  # → detects existing kind cluster, skips creation
  # → registers my-api in ~/.orkestra/deploy/komposer.yaml
  # → Orkestra picks up new CRD, no restart
  # → new tunnel URL for my-api

Something goes wrong
  # Slack: "my-app: error rate 12% — reconcile failing"
  ork doctor deploy rollback
  # → previous image restored, rolling update, 2/2 pods ready

Production deploy
  ork doctor deploy --registry ghcr.io/myorg
  # → same commands, same experience, real cluster
```

---

## Implementation order

1. `ork doctor` rename with alias (30 min)
2. Git metadata extraction (1h)
3. License extraction (1h)
4. SMTP/Slack detection + `--notify-me` (3h)
5. Dependency auto-install: kubectl, helm (2h)
6. Kind binary download without Go (2h)
7. Metrics-server auto-install (1h)
8. Ingress-nginx auto-install (already implemented)
9. Tunnel provider interface (2h)
10. Cloudflared provider (3h)
11. ngrok provider (2h)
12. Tunnel daemon + state file (3h)
13. `ork tunnel` subcommands (2h)
14. `ork doctor deploy --expose` integration (2h)
15. Developer example pack (4h)

Total: ~28h focused work.