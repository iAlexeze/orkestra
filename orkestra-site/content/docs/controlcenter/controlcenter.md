---
title: "Controlcenter"
weight: 9
---

# Orkestra Control Center

## Overview

The Orkestra Control Center is an observability layer that provides unified visibility across multiple Orkestra runtime instances. It transforms operator management from a black-box experience into a transparent, data-rich interface where every component's health, performance, and behavior is visible at a glance.

Unlike traditional dashboards that require custom instrumentation, the Control Center derives all its data directly from the Orkestra runtime's native APIs. This means zero configuration, zero custom metrics, and zero additional code — just launch and observe.

![Control Center Architecture](docs/images/control-center-architecture.png)

---

## Starting the Control Center

The Control Center is built into the `ork` CLI, making it instantly available wherever Orkestra is installed.

### Basic Usage

```bash
# Start with default settings (port 8090, single instance on localhost:8080)
ork control start

# Specify a custom port
ork control start --port 9090

# Monitor multiple Orkestra instances
ork control start --urls "http://localhost:8080,http://localhost:8082,https://orkestra.prod.internal:8080"

# Combine options
ork control start --port 8090 --urls "http://localhost:8080,http://cluster2:8080" --refresh 30s
```

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-p, --port` | `8090` | Port to serve the Control Center on |
| `-u, --urls` | `http://localhost:8080` | Comma-separated list of Orkestra runtime URLs |
| `--refresh` | `10s` | Interval for fetching fresh Katalog data |
| `--log-level` | `info` | Log level (debug, info, warn, error) |
| `--version` | - | Show version information |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ORK_CC_PORT` | Override the default port |
| `ORK_CC_REFRESH` | Override refresh interval |
| `LOG_LEVEL` | Set log level |

---

## Architecture: Control Center vs Control Panel

The Control Center uses a two-level navigation structure that mirrors how platform teams actually work:

### Control Center (Landing Page)

The **Control Center** is the top-level view that aggregates all Katalogs from all configured Orkestra instances. Think of it as your command center — a single pane of glass showing every operator running across your entire infrastructure.

![Control Center Landing Page](docs/images/control-center-landing.png)

**What it shows:**
- Every Katalog discovered across all Orkestra instances
- Overall health status per Katalog
- Summary statistics (total CRDs, workers, resources)
- Quick navigation to each Katalog's Control Panel

**When to use the Control Center:**
- Getting a bird's-eye view of all operators
- Identifying which Katalogs are healthy vs degraded
- Navigating to a specific Katalog's detailed view

### Control Panel (Per-Katalog View)

Each Katalog has its own **Control Panel** — a dedicated dashboard that shows everything about that specific operator. This is where you drill down into the details.

![Katalog Control Panel](docs/images/katalog-control-panel.png)

**What it shows:**
- Katalog metadata (name, version, author, description)
- Platform health cards with CRD status breakdown
- Key metrics for this specific Katalog
- Grid of all CRDs with individual health cards
- Queue pressure, error rates, and uptime per CRD

**When to use a Control Panel:**
- Investigating a specific Katalog's health
- Monitoring CRD-level metrics
- Drilling into individual CRD details

### Relationship Between Center and Panels

```
┌─────────────────────────────────────────────────────────────────┐
│                      CONTROL CENTER                             │
│  (Aggregates ALL Katalogs from ALL instances)                   │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Katalog A   │  │ Katalog B   │  │ Katalog C   │             │
│  │ (Instance 1)│  │ (Instance 2)│  │ (Instance 1)│             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
└─────────┼────────────────┼────────────────┼─────────────────────┘
          │                │                │
          ▼                ▼                ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  CONTROL PANEL  │ │  CONTROL PANEL  │ │  CONTROL PANEL  │
│    Katalog A    │ │    Katalog B    │ │    Katalog C    │
│                 │ │                 │ │                 │
│ • CRD details   │ │ • CRD details   │ │ • CRD details   │
│ • Worker pools  │ │ • Worker pools  │ │ • Worker pools  │
│ • Queue depth   │ │ • Queue depth   │ │ • Queue depth   │
│ • RBAC rules    │ │ • RBAC rules    │ │ • RBAC rules    │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

---

## Understanding Health States

One of the most important concepts in the Control Center is the distinction between **Orkestra Health** and **Katalog Health**. They measure different things and have different implications.

### Orkestra Health (Runtime Health)

**Orkestra Health** indicates whether the Orkestra runtime itself is operational. This is about the infrastructure, not the workloads.

| State | Meaning | Action Required |
|-------|---------|-----------------|
| 🟢 Operational | The Orkestra runtime is running and able to serve requests | None |
| 🟡 Degraded | The runtime is experiencing issues (e.g., leader election problems) | Check Orkestra deployment |

Orkestra Health appears in the top-right of every Control Panel:

![Orkestra Health Indicator](docs/images/orkestra-health.png)

### Katalog Health (Workload Health)

**Katalog Health** indicates whether all CRDs within a Katalog are functioning correctly. This is about the operators themselves.

| State | Meaning | Action Required |
|-------|---------|-----------------|
| 🟢 Healthy | All CRDs are healthy and reconciling successfully | None |
| 🔴 Degraded | One or more CRDs are experiencing issues | Investigate individual CRDs |

### Why They Can Differ

It's entirely possible for Orkestra Health to be "Operational" while Katalog Health is "Degraded." This is normal and expected:

```
╔═══════════════════════════════════════════════════════════════════╗
║  Orkestra Runtime:  🟢 Operational                                ║
║  Katalog Health:    🔴 Degraded (2 degraded, 3 started)           ║
║                                                                   ║
║  The runtime is working perfectly. But some CRDs have issues.    ║
║  This tells you: the problem is in your CRD definitions or       ║
║  the resources they manage, not in Orkestra itself.              ║
╚═══════════════════════════════════════════════════════════════════╝
```

**This distinction is critical for troubleshooting:**
- Orkestra degraded → check the Orkestra deployment, network, certificates
- Only Katalog degraded → check your CRD definitions, templates, and the resources they create

---

## The Katalog Control Panel

When you select a Katalog from the Control Center, you enter its dedicated Control Panel. This is where you'll spend most of your time.

![Katalog Control Panel Annotated](docs/images/katalog-panel-annotated.png)

### Platform Health Cards

The top section shows a breakdown of all CRDs in this Katalog by their current state:

| Card | Color | Meaning |
|------|-------|---------|
| Healthy | Green | CRD is fully operational with no errors |
| Started | Blue | CRD has started but may have some issues (e.g., errors on first reconciles) |
| Pending | Yellow | CRD workers are waiting to start (dependencies not ready or CRD not yet installed) |
| Degraded | Red | CRD has persistent failures and needs attention |

Each card includes a progress bar showing its proportion of the total CRDs, giving you an immediate visual sense of overall health.

### Key Metrics Dashboard

Five key metrics provide a snapshot of the Katalog's operational state:

| Metric | Description |
|--------|-------------|
| CRDs Managed | Total CRDs in this Katalog, with count of healthy ones |
| Active Workers | Sum of all worker goroutines across all CRDs |
| Live Resources | Total custom resources in the informer cache |
| Katalog Health | Overall health status (Healthy/Degraded) |
| Orkestra Runtime | Runtime operational status |

### CRD Cards Grid

Each CRD is represented by a card showing its real-time state:

![CRD Card](docs/images/crd-card.png)

**What each card shows:**
- **CRD Name** — The resource kind this CRD manages
- **Status Badge** — Current state (Healthy, Started, Pending, Degraded)
- **Dependencies** — If this CRD depends on others (displayed as tags)
- **Workers** — Active vs total workers (e.g., "3/5" means 3 of 5 workers are running)
- **Queue Pressure** — Visual progress bar showing queue depth relative to maximum
- **Resources** — Number of custom resources currently in the informer cache
- **Error Rate** — Percentage of reconciliations that failed
- **Uptime** — How long the CRD has been running

**Queue Pressure Warnings:**
- 🔵 <50% — Normal operation
- 🟡 50-80% — Moderate pressure, monitor
- 🔴 >80% — High pressure, consider increasing workers

---

## CRD Detail View

Clicking "View details" on any CRD card takes you to the CRD Detail View — the most comprehensive view of a single CRD's operation.

![CRD Detail View](docs/images/crd-detail.png)

### Worker Pool Visualization

This is one of the most unique features of the Control Center. Every worker goroutine is shown as an individual card:

![Worker Pool](docs/images/worker-pool.png)

**Worker States:**
| Icon | State | Color | Meaning |
|------|-------|-------|---------|
| ⚡ | Processing | Blue (pulsing) | Currently reconciling an item |
| 💤 | Idle | Green | Waiting for work |
| ⛔ | Stopped | Red | Worker terminated (CRD deactivated) |

**Why this matters:**
- See if your workers are actually doing work or sitting idle
- Detect stuck workers (processing for too long)
- Understand utilization at a glance

For CRDs with many workers (e.g., 30+), the view shows the first 10 workers with a "Show all N workers" button to expand.

### Queue Visualization

The queue depth is shown as a progress bar with formatted numbers:

```
Queue Pressure: 15.2K / 50K  (30% full)
[████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░]
```

Large numbers are automatically formatted (e.g., "15.2K" instead of "15234") with tooltips showing exact values on hover.

### Runtime Health

This section shows the CRD's operational timeline:

| Field | Description |
|-------|-------------|
| Uptime | Total time the CRD has been registered |
| Started | When the CRD's workers were last started |
| Last Reconcile | When the last reconciliation completed |

**Consecutive Failures** and **Last Error** appear here when problems occur, helping you debug reconciliation issues.

### Version Conversion

If the CRD supports multiple API versions, this section shows conversion metrics:

| Metric | Description |
|--------|-------------|
| Total Requests | Number of conversion requests received |
| Success / Failure | Success and failure counts |
| Avg Latency | Average conversion time |
| P95 Latency | 95th percentile conversion time |

This helps identify performance issues in version conversion.

### Admission Webhooks

If admission webhooks are enabled, this section shows validation and mutation statistics:

| Metric | Description |
|--------|-------------|
| Total | Total admission requests |
| Allowed / Denied | Validation outcomes |
| Applied / Skipped | Mutation outcomes |
| Latency | Average and P95 response times |

### Dependencies

If this CRD depends on others, all dependencies are shown with their current health states:

![Dependencies](docs/images/dependencies.png)

Each dependency card shows:
- Dependency name (clickable to navigate directly)
- Current state (Healthy, Started, Pending, Degraded)
- Whether the condition is satisfied
- When the dependency was last checked

**This is critical for understanding cascading failures.** If a CRD is degraded but its dependencies are healthy, the problem is in the CRD itself. If dependencies are also degraded, fix them first.

### RBAC Permissions

A unique feature of Orkestra is that RBAC permissions are **derived from the Katalog**, not manually written. The CRD Detail View shows exactly what permissions this CRD requires:

![RBAC Permissions](docs/images/rbac-permissions.png)

**What it shows:**
- Total number of RBAC rules
- Summary of required permissions (e.g., "1 deployments, 1 services, 1 websites")
- Detailed table of API groups, resources, verbs, and descriptions
- Security note explaining that permissions are derived from the Katalog

**Why this matters:**
- Verify least-privilege security
- Audit what your CRDs can do
- Understand the blast radius of each operator

### Reconciler Configuration

The final section shows how the CRD is configured to reconcile:

| Field | Description |
|-------|-------------|
| Reconciler Type | Generic (declarative) or Custom (Go hooks) |
| Resync | How often to re-reconcile even without changes |
| Finalizers | Cleanup finalizers configured for this CRD |
| Hooks | Whether Go hooks are configured |
| Constructor | Whether a custom reconciler is used |

---

## Multi-Instance Monitoring

The Control Center can monitor multiple Orkestra runtimes simultaneously. This is configured via the `--urls` flag:

```bash
ork control start --urls "http://prod-cluster:8080,http://staging-cluster:8080,http://dev-cluster:8080"
```

![Multi-Instance](docs/images/multi-instance.png)

Each instance appears as a separate Katalog in the Control Center, allowing you to:

- Compare health across environments
- Monitor canary deployments alongside stable
- Troubleshoot cross-cluster issues from a single view

---

## Health Endpoints

The Control Center exposes its own health endpoints for orchestration:

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `/controlcenter/health` | Liveness probe | `{"status":"healthy"}` |
| `/controlcenter/ready` | Readiness probe | `{"status":"ready"}` when at least one backend is healthy |
| `/controlcenter/version` | Version info | `{"version":"1.0.0",...}` |

These are used by Kubernetes for liveness and readiness probes when deployed via Helm.

---

## Navigation Summary

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CONTROL CENTER                                    │
│                     /controlcenter                                          │
│                                                                             │
│  Shows all Katalogs from all configured Orkestra instances                 │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  Katalog A (healthy)    Katalog B (degraded)    Katalog C (healthy) │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ Click on Katalog
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CONTROL PANEL                                     │
│                     /controlcenter/katalog/{name}                           │
│                                                                             │
│  Shows all CRDs within a single Katalog                                     │
│                                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                         │
│  │  CRD Card   │  │  CRD Card   │  │  CRD Card   │                         │
│  │  (website)  │  │ (postgres)  │  │   (redis)   │                         │
│  └─────────────┘  └─────────────┘  └─────────────┘                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ Click "View details"
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CRD DETAIL                                        │
│                  /controlcenter/katalog/{name}/crd/{crd}                    │
│                                                                             │
│  Complete view of a single CRD:                                            │
│  • Worker pool (per-worker state)                                          │
│  • Queue visualization                                                     │
│  • Runtime health                                                          │
│  • Version conversion metrics                                              │
│  • Admission webhook stats                                                 │
│  • Dependencies with health status                                         │
│  • RBAC permissions                                                        │
│  • Reconciler configuration                                                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Troubleshooting Guide

### Control Center shows "No Katalogs found"

**Possible causes:**
- Orkestra runtime isn't running
- Wrong URL in `--urls` flag
- Network connectivity issue

**Solutions:**
1. Verify Orkestra runtime is running: `curl http://localhost:8080/health`
2. Check the URLs in your `--urls` flag
3. Ensure network connectivity between Control Center and runtimes

### Katalog shows "Degraded" but Orkestra is "Operational"

This is expected. The runtime is working, but your CRDs have issues:

1. Click into the Katalog Control Panel
2. Look for red "Degraded" badges on CRD cards
3. Click into the degraded CRD's detail view
4. Check "Last Error" and "Consecutive Failures" in Runtime Health
5. Fix the underlying issue (usually a template error or missing resource)

### Workers show as "Idle" but queue has items

This indicates workers are stuck. Possible causes:

1. A reconcile is taking too long
2. Deadlock in custom Go hooks
3. External API call not responding

**Check:**
- Are any workers showing as "Processing" for an unusually long time?
- Look at the "Last Reconcile" timestamp
- Check CRD logs for hung operations

### Queue depth is very high

High queue depth means reconciles are backing up:

1. Check worker utilization (are all workers processing?)
2. Consider increasing `workers` for this CRD
3. Check if reconciles are slow (look at reconcile duration metrics)
4. Look for error patterns causing requeues

### RBAC rules show 0

This means the runtime isn't providing RBAC information:

1. Ensure you're using a recent version of Orkestra (v1.0+)
2. Check that the CRD has declarative templates (onCreate, onReconcile, etc.)
3. Verify the runtime API is accessible

---

## Best Practices

### For Platform Teams

1. **Run the Control Center as a service** — Use the Helm chart to deploy it alongside your Orkestra runtimes
2. **Configure multiple URLs** — Monitor all your clusters from one place
3. **Set up alerts** — Watch for Katalog Health becoming degraded
4. **Use the RBAC view** — Audit permissions before deploying new CRDs

### For Developers

1. **Start locally** — `ork control start` while developing Katalogs
2. **Watch worker pools** — Ensure your reconciles aren't getting stuck
3. **Monitor queue depth** — If it grows, your reconciles may be too slow
4. **Check dependencies** — Use the dependency view to understand ordering issues

### For Security Teams

1. **Review RBAC in the UI** — No need to read YAML
2. **Verify least privilege** — Each CRD shows exactly what it needs
3. **Audit changes** — Use the Control Center to see permission drift
4. **Generate and review** — Run `ork generate rbac` to get the full ClusterRole

---

## Frequently Asked Questions

### Why does the Control Center show workers but not what they're doing?

The Control Center shows per-worker state (idle/processing) but not the specific resource being reconciled. This is intentional — exposing resource names could leak sensitive data. For detailed tracing, use the runtime's logs or distributed tracing.

### Can I control workers from the UI?

Not yet. The Control Center is read-only in v1.0. Worker scaling, CRD enable/disable, and other controls are planned for v2.0.

### How often does the Control Center refresh?

By default, every 10 seconds. You can change this with the `--refresh` flag. Shorter intervals give fresher data but increase load on the runtime.

### Does the Control Center store historical data?

No. v1.0 only shows current state. Historical trends and time-series graphs are planned for v2.0.

### Can I secure the Control Center with authentication?

Not in v1.0. Authentication, RBAC, and SSO are planned for v2.0. For production, run the Control Center behind an authenticating reverse proxy (e.g., oauth2-proxy, Cloudflare Access).

---

## Conclusion

The Orkestra Control Center transforms operator management from guesswork into science. Every component is visible. Every dependency is traceable. Every permission is auditable.

You no longer need to wonder:
- "Is my operator actually working?" — Look at the worker pool
- "Why is this reconcile failing?" — Check the Last Error
- "What permissions does this CRD need?" — See the RBAC table
- "Is this dependency causing my problem?" — Check the dependency health

This is observability by default, not by effort.

---

*Orkestra Control Center — Part of the Orkestra Project*
*MIT Licensed*