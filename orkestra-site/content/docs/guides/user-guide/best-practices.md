---
title: "Best Practices"
weight: 32
---

# Best Practices for Orkestra Operators

This document collects lessons learned from building and running Orkestra operators in production. Follow these practices to build operators that are reliable, maintainable, and easy to operate.

---

## Choosing Between Katalog and Komposer

The first decision you'll make is whether to use a single Katalog or a Komposer. This choice depends entirely on how many sources you need to combine.

### Use Katalog When…

- You have a **single operator** — one CRD or multiple CRDs in one file
- You want a **single source of truth** — everything lives in one YAML file
- You don't need to merge multiple sources — no Helm charts, no external files
- You want the **fastest iteration loop** — edit → `ork run` → done

**Example:**

```yaml
# katalog.yaml — everything in one file
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
spec:
  crds:
    website:
      # ... config
    database:
      # ... config
```

### Use Komposer When…

- You have **multiple Katalogs** to combine into one runtime
- You want to **merge Helm charts** — Komposer can ingest Helm output
- You need **environment‑specific layering** — different values for dev, staging, prod
- You want **inline overrides** — patches for specific environments
- You want to consume **registry entries** (e.g., `platform-workflow@v2.45`)

**Example:**

```yaml
# komposer.yaml — combines multiple sources
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
sources:
  files:
    - ./katalogs/website.yaml
    - https://internal.company.com/crds/platform.yaml
  helm:
    - repo: ./charts
      chart: platform-crds
      version: 0.1.0
  registry:
    - postgres@v14
    - monitoring@v2
spec:
  crds:
    application:
      workers: 4   # override for production
```

### The Golden Rule

> **If you have one source → use Katalog.**
> **If you have more than one source → use Komposer.**

That's the entire decision. See 👉 **[Choosing Between Katalog and Komposer](./choosing-katalog-vs-komposer.md).**

---

## Katalog Design

### Start Simple, Grow as Needed

**Start with a single Katalog.** It's the simplest, fastest, cleanest path. Adopt Komposer only when your operator ecosystem grows.

```yaml
# Good — start here
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: my-operator
spec:
  crds:
    app:
      # ... minimal config
```

### Use Built‑in Resource Enrichment

Don't specify `group`, `version`, or `plural` for built‑in resources. Orkestra discovers them automatically.

```yaml
# Good — Orkestra knows what a Pod is
- name: pod-governance
  apiTypes:
    kind: Pod
```

```yaml
# Avoid — unnecessary
- name: pod-governance
  apiTypes:
    group: ""
    version: v1
    kind: Pod
    plural: pods
```

### Name CRDs Clearly

Use lowercase kebab‑case for CRD names. These appear in URLs and logs.

```yaml
# Good
- name: my-application
- name: database-backup
```

```yaml
# Avoid
- name: MyApplication
- name: database_backup
```

### Document Your Katalog

Add descriptions so others understand what each CRD does.

```yaml
- name: website
  description: |
    Manages a web application with Deployment and Service.
    Creates a Deployment with the specified image and replicas,
    and a Service only when exposePublicly is true.
```

### Keep CRDs Focused

One Katalog can contain multiple CRDs, but each CRD should have a clear purpose.

```yaml
# Good — related CRDs together
crds:
  database:
  backup:
  migration:
```

```yaml
# Avoid — unrelated CRDs
crds:
  website:
  database:
  monitoring:
  logging:
```

---

## Templating

### Use Templates for Everything That Varies

Hardcode only what's truly constant across all CRs.

```yaml
# Good — image and replicas come from the CR
deployments:
  - image: "{{ .spec.image }}"
    replicas: "{{ .spec.replicas }}"
    port: "80"  # Always 80
```

```yaml
# Avoid — hardcoded values that should vary
deployments:
  - image: "nginx:latest"
    replicas: "1"
```

### Provide Sensible Defaults

Don't require users to specify every field. Use defaults.

```yaml
# Good — replicas defaults to 1 if not specified
deployments:
  - replicas: "{{ .spec.replicas }}"
```

The GenericReconciler provides defaults automatically:

| Resource | Field | Default |
|----------|-------|---------|
| Deployment | name | `<cr-name>-deployment` |
| Deployment | replicas | `1` |
| Service | name | `<cr-name>-svc` |
| Service | type | `ClusterIP` |
| Secret | name | `<cr-name>-secret` |

### Use Conditional Creation Wisely

Use `when` blocks for optional resources, but keep conditions simple.

```yaml
# Good
services:
  - name: "{{ .metadata.name }}-public"
    when:
      - field: spec.exposePublicly
        equals: "true"
```

```yaml
# Avoid — too complex, better in a hook
services:
  - name: "{{ .metadata.name }}-public"
    when:
      - field: spec.environment
        equals: "production"
      - field: spec.region
        equals: "us-east-1"
      - field: spec.replicas
        min: 3
```

### Validate Templates with `--debug`

Always test templates with debug mode before running.

```bash
ork run --katalog katalog.yaml --debug
```

Look for resolved values:

```
DEBUG resolved template: image="{{ .spec.image }}" → "nginx:1.25"
DEBUG resolved template: replicas="{{ .spec.replicas }}" → "3"
DEBUG Conditions not met — skipping service
```

---

## Dependencies

### Declare Dependencies Explicitly

Don't rely on startup order. Declare `dependsOn` even if it seems obvious.

```yaml
# Good
- name: database
- name: application
  dependsOn:
    - database
```

```yaml
# Avoid — depends on order in the Katalog file
- name: database
- name: application
```

### Avoid Circular Dependencies

Orkestra detects cycles and refuses to start. Design your CRDs to avoid them.

```yaml
# Bad — cycle
- name: a
  dependsOn: [b]
- name: b
  dependsOn: [a]
```

### Keep Dependencies Shallow

Deep dependency chains make the system harder to reason about.

```yaml
# Acceptable
a → b → c
```

```yaml
# Avoid if possible
a → b → c → d → e → f
```

### Use Soft Dependencies for Optional Integrations

If a dependency is optional, the CRD should still start. The GenericReconciler handles this automatically — a missing CRD doesn't block startup, it just degrades health.

---

## Workers and Queues

### Set Workers Based on Expected Volume

| Workload | Workers | Reason |
|----------|---------|--------|
| Low volume (1-10 CRs) | 1-2 | Enough to process changes |
| Medium volume (10-100 CRs) | 3-5 | Handles bursts |
| High volume (100+ CRs) | 5-10 | Parallel processing |
| Very high volume (1000+ CRs) | 10-20 | Scale horizontally |

```yaml
- name: high-volume-crd
  workers: 10
  queue:
    maxQueueDepth: 5000
```

### Set Queue Depth Based on Volume and Recovery Time

Queue depth determines how many pending reconciliations can be stored.

```yaml
# For critical CRDs where you never want to drop events
queue:
  maxQueueDepth: 5000

# For non‑critical CRDs
queue:
  maxQueueDepth: 500
```

### Use Per‑CRD Queues for High‑Volume CRDs

High‑volume CRDs should have their own queue so they don't starve others.

```yaml
# High‑volume CRD — dedicated queue
- name: events
  queue:
    default: false
    maxQueueDepth: 5000
```

```yaml
# Low‑volume CRD — shared default queue
- name: config
  queue:
    default: true
```

### Tune Degrade Threshold

The degrade threshold controls how many consecutive failures before a CRD is marked degraded.

```yaml
# For critical CRDs — degrade quickly to alert
queue:
  degradeThreshold: 3

# For tolerant CRDs — allow more retries
queue:
  degradeThreshold: 10
```

Default: `5` (configured by `DEGRADE_THRESHOLD` env var)

---

## Observability

### Monitor Health Endpoints

Set up alerts on degraded CRDs.

```bash
# Check all CRDs
curl localhost:8080/katalog | jq '.crds[] | {name: .name, healthy: .healthy}'
```

```yaml
# Prometheus alert example
- alert: OrkestraCRDDegraded
  expr: controller_reconcile_total{result="error"} > 0
  for: 5m
```

### Watch Queue Depth

Queue depth indicates backlog. Set up alerts when it grows.

```promql
controller_queue_depth > 100
```

### Track Worker Utilization

Workers should be fully utilized during high load, idle during low load.

```promql
controller_workers_active
```

If workers are always at max, consider increasing `workers`.

### Use `ork status` for Quick Checks

```bash
ork status -w
```

This shows you live health, workers, queue depth, and resource counts across all CRDs.

---

## Testing

### Test with `ork validate` First

Always validate your Katalog before running.

```bash
ork validate --katalog katalog.yaml
```

This catches:
- Missing required fields
- Duplicate CRD names
- Circular dependencies
- Invalid references

### Test with `ork template`

Preview what Orkestra will do.

```bash
ork template --katalog katalog.yaml --graph
ork template --katalog katalog.yaml --json
```

### Test Missing CRD Scenarios

Test that your operator behaves correctly when CRDs aren't installed yet.

```bash
# Start Orkestra without CRD
ork run --katalog katalog.yaml

# Later, install CRD
kubectl apply -f crd.yaml

# Check that activation happened
curl localhost:8080/katalog/my-crd/health
```

### Test Dependency Scenarios

Test that dependents wait for dependencies.

```bash
# Install only dependent CRD first
kubectl apply -f dependent-crd.yaml

# Start Orkestra — dependent should be degraded
ork run --katalog katalog.yaml
curl localhost:8080/katalog/dependent/health  # Should be 503

# Install dependency
kubectl apply -f dependency-crd.yaml

# Wait — dependent should become healthy
```

### Test Komposer Merges

If you use Komposer, test that sources merge correctly.

```bash
# Validate the Komposer
ork validate --katalog komposer.yaml

# Preview the merged result
ork template --katalog komposer.yaml --json
```

---

## Deployment

### Use `ork init` to Scaffold

```bash
ork init my-operator
cd my-operator
```

This creates a clean workspace with examples and environment configuration.

### Set Environment Variables Appropriately

| Environment | Workers | Resync | Log Level |
|-------------|---------|--------|-----------|
| Development | 1-2 | 30s | debug |
| Staging | 2-3 | 1m | info |
| Production | 3-5 | 5m | warn |

### Run Multiple Replicas for HA

```yaml
# values.yaml
replicaCount: 2
leaderElection:
  enabled: true
```

### Configure Readiness Probe

The `/ready` endpoint returns 200 only when all CRDs are ready.

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Use Pod Anti‑Affinity for HA

Spread replicas across nodes for resilience.

```yaml
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app: orkestra
        topologyKey: kubernetes.io/hostname
```

### Inject Credentials via Secrets for Remote Sources

If your Komposer sources from private URLs, use Kubernetes Secrets.

```yaml
# values.yaml
extraEnvFrom:
  - secretRef:
      name: orkestra-katalog-creds
```

```bash
kubectl create secret generic orkestra-katalog-creds \
  --from-literal=GITHUB_TOKEN=ghp_xxxx \
  --from-literal=ARTIFACTORY_USER=svc-orkestra
```

---

## Common Pitfalls

### Pitfall: Hardcoded Namespaces

```yaml
# Bad — only works in default namespace
deployments:
  - namespace: "default"
```

```yaml
# Good — works in any namespace
deployments:
  - namespace: "{{ .metadata.namespace }}"
```

### Pitfall: Missing `reconcile: true`

If you want drift correction, you need `reconcile: true`.

```yaml
# Bad — no drift correction
deployments:
  - image: "{{ .spec.image }}"
```

```yaml
# Good — drift correction enabled
deployments:
  - image: "{{ .spec.image }}"
    reconcile: true
```

### Pitfall: Forgetting Owner References

Don't manage this yourself. Orkestra sets owner references automatically for all resources created from templates. If you write a custom reconciler, ensure you set them.

### Pitfall: Too Many Workers

Workers are cheap, but each worker adds a goroutine. Start with `workers: 3` and increase if queue depth is consistently high.

### Pitfall: Ignoring Missing CRDs

Orkestra handles missing CRDs gracefully, but you should still test that your operator works when CRDs appear later.

### Pitfall: Not Testing Shutdown

Always test that your operator shuts down cleanly.

```bash
# Start operator
ork run --katalog katalog.yaml &
ORK_PID=$!

# Wait a few seconds
sleep 5

# Send SIGTERM
kill $ORK_PID

# Check that it exits cleanly
wait $ORK_PID
```

### Pitfall: Hardcoding Credentials in Komposer

Never put credentials in your Komposer YAML. Always use environment variables and Kubernetes Secrets.

```yaml
# Bad — credentials in YAML
sources:
  files:
    - url: https://private.company.com/katalog.yaml
      auth:
        type: bearer
        token: "ghp_123456789"
```

```yaml
# Good — credentials from environment
sources:
  files:
    - url: https://private.company.com/katalog.yaml
      auth:
        type: bearer
        fromEnv: PRIVATE_TOKEN
```

---

## Summary

| Practice | Why |
|----------|-----|
| Start with Katalog, use Komposer when you have multiple sources | Simplicity first, scale when needed |
| Use built‑in enrichment | Let Orkestra discover what Kubernetes knows |
| Declare dependencies explicitly | Orkestra handles ordering |
| Set workers based on volume | Match resources to load |
| Monitor health and queue depth | Catch problems early |
| Test with `ork validate` | Fail fast |
| Test missing CRD scenarios | Verify self‑healing |
| Run multiple replicas for HA | Zero downtime on failure |
| Inject credentials via Secrets | Never hardcode secrets |

**Build operators that are simple, reliable, and self‑healing.** 🎼