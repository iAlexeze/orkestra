# 02 — Prometheus Operator

A `MonitoringConfig` CR. That is all your team creates.

Orkestra creates a `ServiceMonitor` and a `PrometheusRule` from it. Prometheus Operator picks them up. Scraping starts. Alerts are active. Your team declared `targetDeployment`, `port`, and an error threshold — they never wrote a PromQL expression or learned what a `namespaceSelector` is.

---

> **Before you start:** Update [katalog.yaml](katalog.yaml) — set `author:` to your name or org. Prometheus Operator (or kube-prometheus-stack) must be installed in the cluster. The `simulate` step works without a cluster regardless.

---

## What the team creates

```yaml
apiVersion: ecosystem.demo.orkestra.io/v1alpha1
kind: MonitoringConfig
metadata:
  name: webapp-monitoring
spec:
  targetDeployment: my-webapp
  port: "8080"
  team: platform
  errorRateThreshold: "0.05"
```

## What Orkestra creates

**ServiceMonitor** — tells Prometheus which pods to scrape:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: webapp-monitoring
spec:
  selector:
    matchLabels:
      app: my-webapp
  endpoints:
    - port: "8080"
      interval: "30s"
      path: /metrics
```

**PrometheusRule** — fires an alert when error rate exceeds the threshold:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: webapp-monitoring
spec:
  groups:
    - name: webapp-monitoring.rules
      rules:
        - alert: HighErrorRate
          expr: |
            rate(http_requests_total{job="my-webapp",status=~"5.."}[5m])
            /
            rate(http_requests_total{job="my-webapp"}[5m])
            > 0.05
          for: 5m
          labels:
            severity: warning
            team: platform
```

The PromQL expression is implemented once in the Katalog. Every team that creates a `MonitoringConfig` gets the same battle-tested alert logic — not a copy-pasted expression with a typo in the metric name.

---

## Step 1 — Install Prometheus Operator

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace \
  --set grafana.enabled=false \
  --set alertmanager.enabled=false \
  --set nodeExporter.enabled=false \
  --set kubeStateMetrics.enabled=false \
  --set prometheus.enabled=false \
  --set prometheusOperator.admissionWebhooks.enabled=false \
  --wait
```

This installs only the Prometheus Operator — the component that provides the `ServiceMonitor` and `PrometheusRule` CRDs. Grafana, Alertmanager, Prometheus itself, and exporters are disabled; they are not needed to demonstrate the operator pattern.

---

## Step 2 — Simulate

```bash
ork simulate
```

Both the ServiceMonitor and PrometheusRule appear in cycle 1.

---

## Step 3 — Apply the CRD

```bash
kubectl apply -f crd.yaml
```

---

## Step 4 — Run locally

```bash
ork run
```

Apply the CR:

```bash
kubectl apply -f cr.yaml
kubectl get monitoringconfigs
kubectl get servicemonitor webapp-monitoring
kubectl get prometheusrule webapp-monitoring
```

---

## Step 5 — Control center

In a second terminal:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081).

---

## Step 6 — Push to the registry

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

---

## Step 7 — Inspect

```bash
ork inspect monitoring-operator:0.1.0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next

[03 — Crossplane](../03-crossplane/README.md) — infrastructure provisioning from a simple `Infra` CR.
