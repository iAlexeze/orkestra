# Prometheus

## MonitoringConfig → ServiceMonitor + PrometheusRule

`02-prometheus` wraps Prometheus Operator. One `MonitoringConfig` CR creates two resources: a `ServiceMonitor` that tells Prometheus what to scrape, and a `PrometheusRule` with an alert that fires when the error rate exceeds the threshold. The developer declares a threshold — Orkestra writes the PromQL.

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/02-prometheus
```

---

## One CR, two resources

This is the first example where one internal CR maps to multiple ecosystem resources. The pattern is the same — declare the internal CRD in your vocabulary, implement the mapping in the Katalog:

```yaml
spec:
  targetDeployment: my-webapp
  port: "8080"
  team: platform
  errorRateThreshold: "0.05"
```

What Orkestra creates:

**ServiceMonitor** — scrape configuration:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
spec:
  selector:
    matchLabels:
      app: my-webapp
  endpoints:
    - port: "8080"
      interval: "30s"
      path: /metrics
```

**PrometheusRule** — alert expression:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
spec:
  groups:
    - rules:
        - alert: HighErrorRate
          expr: |
            rate(http_requests_total{job="my-webapp",status=~"5.."}[5m])
            /
            rate(http_requests_total{job="my-webapp"}[5m])
            > 0.05
```

The developer wrote `errorRateThreshold: "0.05"`. They did not write the PromQL expression, the rate window, or the metric label selectors. The platform team wrote those once in the Katalog.

---

## Why this matters

PromQL is expressive. It is also opaque to developers who are not SREs. Common failure modes:

- Wrong label selectors — alert fires for everything or nothing
- Wrong rate window — too short produces noise, too long misses incidents
- Inconsistent metric naming — alert does not fire because the team named their counter differently

With the mapping:
- The label selector is derived from `spec.targetDeployment` — always matches the right pods
- The rate window is fixed at `5m` by the platform
- The metric names are enforced by convention

The SRE team writes the PromQL template once. Every team that creates a `MonitoringConfig` gets a working, correctly-labelled alert without writing a single PromQL expression.

---

## Try it

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/02-prometheus
# Follow steps in README
```

→ [03 — Crossplane](./04-crossplane.md) — infrastructure provisioning with an approval gate.
