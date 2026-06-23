# 04 — Platform Stack

Four CRs. Four ecosystem tools. One runtime. Full admission enforcement and deletion protection.

In `00`–`03` you built four operators independently. Here a single Komposer composes all four under one control plane and adds the gateway — so the same validation rules that run at reconcile time also intercept `kubectl apply` before etcd storage.

---

> **Before you start:** Push the four operators from `00`–`03` to your registry first. Update the `imports.registry` URLs in [komposer.yaml](komposer.yaml) to match your published versions. Replace `ghcr.io/myorg` with your actual registry org.

---

## What the team creates

| CR | Orkestra creates | Tool |
|---|---|---|
| `App/my-webapp` | `Application/my-webapp` | ArgoCD |
| `SecurityConfig/webapp-cert` | `Issuer/webapp-cert-issuer` + `Certificate/webapp-cert-cert` | cert-manager |
| `MonitoringConfig/webapp-monitoring` | `ServiceMonitor/webapp-monitoring` + `PrometheusRule/webapp-monitoring` | Prometheus Operator |
| `Infra/webapp-db` | `PostgreSQLInstance/webapp-db` (once approved) | Crossplane |

---

## Step 1 — Install the ecosystem tools

**ArgoCD:**
```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl wait --for=condition=Available deployment/argocd-server -n argocd --timeout=120s
```

**cert-manager:**
```bash
helm repo add jetstack https://charts.jetstack.io
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --version v1.13.6 \
  --set installCRDs=true \
  --wait
```

> **v1.13.6** — cert-manager v1.14+ uses `selectableFields` in its CRDs, which requires Kubernetes 1.31+. v1.13.6 is compatible with Kubernetes 1.28–1.30 (the default kind node versions).

**Prometheus Operator (operator only — no Grafana, Alertmanager, or Prometheus instance):**
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

**Crossplane:**
```bash
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm upgrade --install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system --create-namespace \
  --wait
```

Apply the mock `PostgreSQLInstance` CRD (stands in for your org's Crossplane XRD + Composition in production):
```bash
kubectl apply -f ../03-crossplane/mock-crd.yaml
```

---

## Step 2 — Apply the CRDs

```bash
kubectl apply -f ../00-argocd/crd.yaml
kubectl apply -f ../01-cert-manager/crd.yaml
kubectl apply -f ../02-prometheus/crd.yaml
kubectl apply -f ../03-crossplane/crd.yaml
```

---

## Step 3 — Validate

Before generating or applying anything, validate to see exactly what will be authorized:

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs

ork pull -f komposer.yaml
ork validate
```

Then check the full RBAC and gateway declarations:

```bash
ork validate --full
```

The gateway section appears here because `security.webhooks.admission.enabled: true` and `security.deletionProtection.enabled: true` are declared in the Komposer. What you see in `--full` is exactly what gets applied to the cluster.

---

## Step 4 — Generate and apply the bundle

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
```

---

## Step 5 — Install Orkestra with Gateway

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set gateway.enabled=true \
  --wait --timeout 120s
```

Verify both runtime and gateway are running:

```bash
kubectl rollout status deployment/orkestra-runtime -n orkestra-system
kubectl rollout status deployment/orkestra-gateway -n orkestra-system
```

---

## Step 6 — Apply the CRs

```bash
kubectl apply -f cr-app.yaml
kubectl apply -f cr-security.yaml
kubectl apply -f cr-monitoring.yaml
kubectl apply -f cr-infra.yaml
```

Check what Orkestra created:

```bash
# ArgoCD
kubectl get app my-webapp -n argocd

# cert-manager
kubectl get certificate webapp-cert-cert
kubectl get secret webapp-cert-tls

# Prometheus Operator
kubectl get servicemonitor webapp-monitoring
kubectl get prometheusrule webapp-monitoring

# Crossplane (none yet — approved: false)
kubectl get postgresqlinstances
```

Approve the Infra CR when ready to provision:

```bash
kubectl patch infra webapp-db --type=merge -p '{"spec":{"approved":true}}'
kubectl get postgresqlinstances
```

---

## Step 7 — Admission enforcement

Apply an App without team ownership:

```bash
kubectl apply -f cr-denied.yaml
```

With the gateway active, the request is rejected synchronously before etcd storage:

```text
Error from server: admission webhook "validate.orkestra.orkspace.io" denied the request:
validation denied: All apps must declare team ownership (spec.labels.team)
```

The CR is never created. ArgoCD never sees it.

Before the gateway was active, this CR would have been stored — the reconciler would have caught the violation at reconcile time and written `ValidationFailed` to the status. Now the denial happens at `kubectl apply`.

---

## Step 8 — Deletion protection
All CRDs and CRs including the ones in the `custom:` block are now protected from deletion.

Try to delete any of the CRs:

```bash
kubectl delete -F cr-infra.yaml
kubectl delete -F cr-monitoring.yaml
```

A CRD in `custom:`
```bash
kubectl delete -f ../03-crossplane/mock-crd.yaml
```

```text
Error from server: admission webhook "protect.resources.orkestra.orkspace.io" denied the request:
[Orkestra Security] The resource is protected from deletion.
```

To proceed, disable protection on the CR first:

```bash
kubectl patch infra protected-db --type=merge \
  -p '{"metadata":{"annotations":{"orkestra.sh/deletion-protection":"false"}}}'
kubectl delete infra protected-db
```

---

## Step 9 — Control center

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081) — all four CRDs visible, admission events and deletion blocks alongside reconcile activity.

---

## Clean uninstall

Deletion protection blocks `helm uninstall` — Orkestra's own infrastructure is protected too. Disable it first.

In [komposer.yaml](komposer.yaml), set `security.deletionProtection.enabled: false`, then regenerate and reapply:

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
kubectl rollout restart deployment/orkestra-gateway -n orkestra-system
kubectl rollout status deployment/orkestra-gateway -n orkestra-system --timeout=60s
```

The gateway picks up the updated bundle, the housekeeper removes the deletion-protection webhook, and `helm uninstall` unblocks.

Then run cleanup:

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next

[05 — All-in-One](../05-all-in-one/README.md) — one `PlatformResource` CRD with a `workloadType` discriminator routing to all four tools.
