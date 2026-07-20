# Internal Tools

## Principle

The Control Center is the single pane of glass for your platform. Operators know what they create. The runtime tracks every resource. Internal Tools closes the last gap: where does a platform engineer go to act on what an operator built?

A tool registration is not a plugin. It is a pointer. The runtime is the plugin.

Traditional IDP platforms (Backstage, Port, Cortex) solve the plugin problem by shipping code — TypeScript components, custom data fetchers, versioned packages. The result is a plugin ecosystem with drift, maintenance burden, and bespoke UI work per integration. Orkestra's answer: register a URL and an API group. CC discovers the rest from the live resource list.

---

## Registration

Tools are registered in `values.yaml` (cluster-level tools) or via the `/katalog` response (operator-level tools that self-register). CC merges both at startup.

```yaml
# charts/orkestra/values.yaml
controlCenter:
  tools:
    - name: ArgoCD
      description: GitOps continuous delivery
      url: https://argocd.internal.example.com
      category: Deployment
      tags: [gitops, cd]
      owner: platform-team
      apiGroup: argoproj.io          # discovery key — links tool to live CRs

    - name: Grafana
      description: Metrics and dashboards
      url: https://grafana.internal.example.com
      category: Observability
      tags: [metrics, dashboards]
      owner: platform-team
      apiGroup: monitoring.coreos.com

    - name: Vault
      description: Secrets management
      url: https://vault.internal.example.com
      category: Security
      tags: [secrets]
      owner: security-team
      # no apiGroup — Vault manages secrets outside Kubernetes CRs
```

Operators that deploy ecosystem resources can self-register by including a `tools:` block in their katalog (flows through `/katalog` → CC aggregation, same path as `GatewayEndpoint` today).

### Go type

```go
// cc/types.go

type ToolRegistration struct {
    Name        string   `json:"name"`
    Description string   `json:"description,omitempty"`
    URL         string   `json:"url"`
    Category    string   `json:"category,omitempty"`  // Deployment, Observability, Security, Databases…
    Tags        []string `json:"tags,omitempty"`
    Owner       string   `json:"owner,omitempty"`      // team name
    APIGroup    string   `json:"apiGroup,omitempty"`   // discovery key: argoproj.io, cert-manager.io, …
}

type ToolStatus struct {
    Registration ToolRegistration
    Up           bool
    StatusCode   int
    LatencyMs    int64
    SSLDaysLeft  int      // -1 = HTTP-only (no SSL)
    SSLExpired   bool
    LastCheck    time.Time
    LastError    string
    UptimeBar    [24]bool // last 24 checks — rendered as sparkline
    ManagedCount int      // CRs discovered via APIGroup match
}
```

### ControlCenterKonfig

```go
// cc/konfig.go — add to existing struct
Tools []ToolRegistration `json:"tools,omitempty"`
```

Loaded from `ORK_TOOLS_CONFIG` env var pointing to a JSON file, or rendered directly from the Helm values ConfigMap that CC mounts.

---

## CR Discovery — the dependency chain

This is the core of the feature. Rather than statically linking a tool to a CRD, CC discovers the connection from the live resource list.

### How it works

The runtime tracks every resource created under each operator, including `custom:` resources declared in the katalog's `onCreate` block. Consider the `05-all-in-one` pattern:

```yaml
# A single PlatformResource CR creates one of:
#   argoproj.io/v1alpha1  Application        (workloadType: app)
#   cert-manager.io/v1    Certificate        (workloadType: cert)
#   monitoring.coreos.com/v1  ServiceMonitor  (workloadType: monitoring)
#   platform.demo.orkestra.io/v1alpha1  InfraClaim  (workloadType: infra)
```

Each of those downstream resources has an `apiVersion` containing an API group. CC scans the live CR list from the runtime, extracts the API group from each resource's GVK, and matches it against registered tools:

```
PlatformResource/my-app  →  Application/my-app  (argoproj.io)  →  ArgoCD tool
PlatformResource/my-cert →  Certificate/my-cert (cert-manager.io) →  Grafana tool (no match)
                                                                       cert-manager tool (match)
```

`ManagedCount` on `ToolStatus` is the count of live CRs whose API group matches `tool.APIGroup`. This is computed at each refresh cycle — no separate store, no annotation on CRs.

### Dependency chain view

On the CR detail page (`/controlcenter/katalog/{kat}/{crd}/{cr}`), if any of the resources created by that CR belong to a registered tool's API group, CC surfaces a "Managed by" strip:

```
This CR created:
  ● Application/my-app  →  [ArgoCD ↗]
  ● ServiceMonitor/my-app  →  [Grafana ↗]
```

Each entry links to the tool URL. If the tool supports deep linking (most do — ArgoCD: `/applications/{name}`, Grafana: `/d/{uid}`), the `deepLinkTemplate` field on `ToolRegistration` renders the full path:

```yaml
- name: ArgoCD
  url: https://argocd.internal.example.com
  apiGroup: argoproj.io
  deepLinkTemplate: "/applications/{{ .namespace }}/{{ .name }}"
```

---

## Health monitoring

CC pings each registered tool URL on `REFRESH_INTERVAL` (default 15s, already configurable). The check records:

- HTTP status code
- Response time (ms)
- SSL certificate expiry (days remaining)
- Whether the TLS handshake succeeded

### SSL states

| State | Threshold | Icon colour |
|---|---|---|
| Healthy | > 90 days | Green lock |
| Warning | 30–90 days | Amber lock |
| Critical | < 30 days | Red lock |
| Expired | 0 days | Red lock + strikethrough |
| HTTP only | no TLS | No lock icon (not alarming — internal tools may be HTTP) |

The uptime bar (`[24]bool`) holds the last 24 check results. Rendered as a 24-slot mini bar (3×10px per slot, green/red) in the card footer — no JS, pure CSS. Gives at-a-glance visibility into recent flakiness without storing history.

### Down alert

When any tool fails its health check, the Tools nav item in the sidebar shows a warning badge (`!`). This makes a downed tool visible from any page in CC without navigating to the Tools section. The badge uses the existing `cc-nav-badge` CSS class.

---

## UI

### Sidebar

Tools appears between Metrics and Documentation, gated on `HasTools`:

```html
{{ if .HasTools }}
<a href="/controlcenter/tools" class="cc-nav-item">
  <!-- wrench icon -->
  Tools
  {{ if .ToolsDown }}<span class="cc-nav-badge cc-nav-badge-warn">!</span>{{ end }}
</a>
{{ end }}
```

### Tool cards

Cards use the existing `cc-card` / `cc-card-body` / `cc-card-footer` structure — the same HTML as the katalog grid. No new CSS component needed.

```
┌─────────────────────────────────────┐
│ ArgoCD                    ● Healthy  │
│ [platform-team]  [Deployment]        │
│ GitOps continuous delivery           │
├─────────────────────────────────────┤
│ Latency    42ms                      │
│ SSL        🔒 87 days   ▓▓▓▓▓▓▓▓▓▓▓  │  ← uptime bar
│ Managed    3 resources  [View →]     │
│                                      │
│          [Open ArgoCD ↗]    [Create] │
└─────────────────────────────────────┘
```

- `View →` on the managed count links to the CRD grid filtered to the matching API group
- `[Create]` links to `/controlcenter/katalog/{linkedKatalog}/{linkedCRD}/idp` — the existing IDP form, no new handler
- `Open ↗` links to `tool.URL` (or rendered `deepLinkTemplate` when a name is known)

### Filter and search

Same `cc-filter-bar` pattern as the katalog grid — filter by category, tag, owner, health status. Search by name, URL, or owner.

### Empty state

When no tools are registered:

> Register your internal tools in `values.yaml → controlCenter.tools`. Each card links to the operator form that creates the resource — no plugin code needed.

---

## Route

`/controlcenter/tools` — handled by `cc.ServeHTTP` dispatch (same pattern as `/metrics`, `/docs`). No change to `routes.go`.

---

## What this is not

Tools is not a plugin system. It is a registry of access points for things the runtime already manages. The plugin — the code that creates, reconciles, and tracks state — is the Orkestra operator. The tool card is just the door to the downstream system that operator deploys to.

A [Create] button on a tool card is the full plugin story:
- The form is the existing IDP form (already built)
- The submission goes to the operator via the gateway Apply API (already built)
- The resource appears in the CRD grid (already tracked)
- The downstream tool reflects it via its own UI

Zero custom UI code per integration. The `cc-card` is the plugin container. The operator is the plugin.
