# Control Center

The Orkestra Control Center is a web UI for monitoring one or more Orkestra runtime instances. It derives all its data directly from the runtime APIs — no instrumentation, no custom metrics, no extra configuration.

Start it with one command:

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081).

---

## What You Get

- **Control Center** — the global view. All Katalogs from all configured runtimes on one page.
- **Control Panel** — per-Katalog drill-down. CRD cards, worker pools, queue pressure, error rates.
- **CRD Detail** — per-CRD deep dive. Every worker's state, RBAC permissions, dependencies, admission metrics.
- **Resources** — live CR list for that CRD. The actual objects being reconciled.
- **CR Detail** — single CR view. Status fields, conditions, and child Kubernetes resources created by the reconciler, grouped by kind with ready state and replica counts.
- **[Generated Docs](./generated-docs.md)** — per-CRD documentation derived live from the running operator. Covers API shape, reconcile mode, operatorBox, RBAC, webhooks, protection, endpoints, and more.

---

## Starting

```bash
# Default: port 8081, points at localhost:8080
ork control

# Custom port
ork control --port 9090

# Multiple Orkestra runtimes
ork control --urls "http://cluster1:8080,http://cluster2:8080"

# Start with no preconfigured runtimes (add them from the UI)
ork control --ignore-default
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-p, --port` | `8081` | Port to serve the Control Center on |
| `-u, --urls` | `http://localhost:8080` | Comma-separated Orkestra runtime URLs |
| `--refresh` | `10s` | How often to fetch fresh data from runtimes |
| `--log-level` | `info` | Log level: debug, info, warn, error |
| `--ignore-default` | `false` | Start with no preconfigured runtime URLs |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ADMIN_USERNAME` | Login username | `orkestra` |
| `ADMIN_PASSWORD` | Login password | `orkestra` |
| `SESSION_SECRET` | Cookie signing secret | `dev-secret` |
| `ORK_CC_PORT` | Override port | `8081` |
| `ORK_CC_REFRESH` | Override refresh interval | `10s` |

Set `ADMIN_USERNAME`, `ADMIN_PASSWORD`, and `SESSION_SECRET` to non-default values in any non-local environment.

---

## Architecture

```text
Browser
   │
   ▼
Control Center (port 8081)
   │ aggregates from
   ├── Orkestra Runtime 1 (port 8080)
   ├── Orkestra Runtime 2 (port 8080)
   └── Orkestra Runtime N ...
```

Each runtime exposes `/katalog`, `/katalog/{crd}`, and `/katalog/{crd}/health`. The Control Center polls these and renders the results. It holds no state of its own.

---

## The Control Center as a developer portal

When a CRD entry has `serve.enabled: true` in its Katalog, the Control Center gains a `[+ Create]` button for that CRD:

```text
┌────────────────────────────────────────────────────┐
│  Application    3 CRs    ● Healthy    [+ Create]   │
└────────────────────────────────────────────────────┘
```

Clicking it opens a form generated directly from the CRD's OpenAPI schema and the `serve.fields` presentation hints declared in the Katalog — field labels, placeholders, and input order. No separate form builder. No schema duplication.

```yaml
spec:
  crds:
    application:
      serve:
        enabled: true
        fields:
          environment:
            label: "Environment"
            hint: "Production deployments require platform-team review"
            order: 1
          image:
            label: "Container Image"
            placeholder: "ghcr.io/myorg/myapp:v1.0.0"
            order: 2
```

Submitting the form posts to the gateway Gateway API. Every enforcement rule — admission, namespace protection, deletion protection — applies the same way as `kubectl apply`. The Control Center is one delivery path; CI pipelines, Terraform, and curl are others. The runtime does not distinguish between them.

→ [Internal Developer Platform concept](../../concepts/idp/index.md)
