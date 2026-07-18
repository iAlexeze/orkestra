# Contributing to the Control Center

The control center (`cmd/controlcenter`) is a lightweight Go web server with embedded HTML templates. It connects to one or more running Orkestra runtimes via their `/katalog` HTTP API and presents a read-only view of what is running.

---

## Current state

The control center currently surfaces:

| Page | Path | What it shows |
|------|------|---------------|
| Dashboard | `/` | All connected Katalogs — health, CRD count, worker count, resource count |
| Katalog panel | `/katalog/{name}` | CRD list for one Katalog, status per CRD |
| CRD detail | `/katalog/{name}/crd/{crd}` | CRD config, worker count, recent reconcile status |
| CR list | `/katalog/{name}/crd/{crd}/cr` | All custom resource instances |
| CR detail | `/katalog/{name}/crd/{crd}/cr/{ns}/{name}` | One CR — spec, status, children, events |
| Developer apps | `/katalog/{name}/app/{app}` | Developer Katalog view (orkdoctor path) |
| In-package docs | `/docs` | Rendered ork-docs for the connected operator |
| Metrics | `/metrics` | Metrics page — template exists, data not yet wired |
| Runtime manager | (conditional) | Add / remove / update connected runtime instances |

---

## Where contribution is needed

### Metrics page

The `/metrics` route renders `metrics.html` but currently passes `nil` data. The runtime exposes metrics via `pkg/metrics` (Prometheus counters and gauges on reconcile cycles, errors, webhook calls, queue depth). The work here is:

1. Add a metrics endpoint to the runtime's HTTP API (or query Prometheus directly).
2. Populate a `MetricsData` struct in `handleMetricsPage`.
3. Render charts or a summary table in `metrics.html`.

This is one of the highest-value additions — operators running in production need to see reconcile rates, error rates, and queue depth at a glance.

### CR detail — richer status

The CR detail page shows spec and status but does not yet show:
- Rollback state (is rollback active? how many consecutive failures?)
- Condition history (not just current conditions)
- Notification state (last sent, throttle window remaining)

All of this data is available from the runtime API — it needs to be threaded through `handleCRDetail` and rendered in `cr_detail.html`.

### CRD health timeline

The CRD detail page shows current health but not history. A small time-series chart of reconcile success/failure over the last hour would make degradation and recovery visible.

### Multi-instance diff view

When two runtime instances are connected, there is no view that compares them. A diff view showing which Katalogs / CRDs differ between instances would help multi-cluster operators.

### Gateway status

When a Katalog has a `gatewayEndpoint`, the control center knows the gateway URL from the `/katalog` response. It does not currently show webhook stats (admission calls, conversion calls, deletion protection blocks). The gateway exposes these on `/webhook-stats`.

### IDP mode — follow-on improvements

When `gateway.applyAPI.enabled: true`, the control center renders a **[+ Create]** button per CRD and a form generated from the CRD's OpenAPI schema (fetched from `GET {gatewayEndpoint}/api/v1/schema/{kind}`). The v1 form covers flat schemas only. Several improvements are open for contribution:

**OIDC authentication**

v1 uses static bearer tokens (`gateway.applyAPI.auth.tokens`). OIDC would let the CC use the user's existing session as the Apply API credential — no separate token needed. The gateway would validate the OIDC token against the configured issuer. See the Apply API spec for the auth block shape.

**Service account token review**

For in-cluster callers (CI pipelines running inside the cluster), the gateway can validate Kubernetes service account tokens via the `TokenReview` API. This removes the need for any static token configuration for in-cluster use cases.

**Nested schema rendering**

v1 flattens `type: object` and `type: array` spec fields to a JSON textarea. A richer renderer would:
- Expand `type: object` into an indented field group with one input per property
- Expand `type: array` into a repeatable entry group with add/remove controls

The form renderer lives in `cc/assets/` alongside the existing templates. The schema shape from `/api/v1/schema/{kind}` is the input.

---

## Development setup

The control center is a separate Go module at the root of `cmd/controlcenter/`.

```bash
# Run against a live runtime
ORK_RUNTIME_URLS=http://localhost:8080 go run ./cmd/controlcenter

# Run in NO_LOGIN mode (no auth, useful during local dev)
NO_LOGIN=true ORK_RUNTIME_URLS=http://localhost:8080 go run ./cmd/controlcenter
```

Templates live in `cmd/controlcenter/cc/assets/templates/`. They use Go `html/template` with a shared `_partials.html` for nav, header, and footer.

CSS is in `assets/static/css/style.css`. The control center supports light and dark themes via a `data-theme` attribute.

---

## Adding a new page

1. Add a handler function `func (cc *ControlCenter) handleMyPage(...)` in `controlcenter.go`.
2. Register the route in `ServeHTTP`.
3. Create `assets/templates/mypage.html` extending `_partials.html`.
4. Call `cc.renderTemplate(w, "mypage.html", data)`.

Keep handlers thin — data collection belongs in helper methods, not inline in the handler.
