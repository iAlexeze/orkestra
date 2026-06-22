# 01 — cert-manager

A `SecurityConfig` CR. That is all your team creates.

Orkestra creates a self-signed `Issuer` and a cert-manager `Certificate` from it. cert-manager handles the issuance. The TLS Secret appears in the same namespace. Your team never sees the Issuer spec, the Certificate spec, the `secretName` naming convention, or the `renewBefore` duration format.

Same pattern as `00-argocd`. Different tool, different domain.

---

> **Before you start:** Update [katalog.yaml](katalog.yaml) — set `author:` to your name or org. cert-manager must be installed in the cluster.

---

## What the team creates

```yaml
apiVersion: ecosystem.demo.orkestra.io/v1alpha1
kind: SecurityConfig
metadata:
  name: webapp-cert
spec:
  domain: "api.myorg.io"
  renewBefore: "720h"
```

## What Orkestra creates

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: webapp-cert-issuer
  namespace: default
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: webapp-cert-cert
  namespace: default
spec:
  secretName: webapp-cert-tls
  issuerRef:
    name: webapp-cert-issuer
    kind: Issuer
  commonName: "api.myorg.io"
  dnsNames:
    - "api.myorg.io"
    - "*.api.myorg.io"
  renewBefore: "720h"
```

The `secretName` convention (`<name>-tls`) is enforced by the Katalog. Every certificate in your platform follows it — enforced, not documented.

---

## What the Katalog adds beyond creation

- **Validation**: `spec.domain` is required — applying without it is denied before cert-manager sees anything
- **Status propagation**: the Certificate's `Ready` condition flows back into the SecurityConfig CR's status via `hasStatus: true`
- **Naming convention**: `secretName` is always `<name>-tls`, Issuer is always `<name>-issuer` — no tribal knowledge required

---

## Step 1 — Install cert-manager

```bash
helm repo add jetstack https://charts.jetstack.io
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --version v1.13.6 \
  --set installCRDs=true \
  --wait
```

> **v1.13.6** — cert-manager v1.14+ uses `selectableFields` in its CRDs, which requires Kubernetes 1.31+. v1.13.6 is compatible with Kubernetes 1.28–1.30 (the default kind node versions).

---

## Step 2 — Simulate

```bash
ork simulate
```

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
kubectl get securityconfigs
kubectl get certificate webapp-cert-cert
kubectl get secret webapp-cert-tls
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

## Step 6 — Try the admission rule

Apply a SecurityConfig without a domain:

```bash
kubectl apply -f - <<EOF
apiVersion: ecosystem.demo.orkestra.io/v1alpha1
kind: SecurityConfig
metadata:
  name: bad-cert
  namespace: default
spec:
  renewBefore: "720h"
EOF
```

**With `ork run` (local, no gateway):** the CR is stored but Orkestra halts reconcile immediately — no Issuer, no Certificate, no Secret created. Check the status:

```bash
kubectl get securityconfig bad-cert -o yaml | grep -A6 "conditions:"
# - type: ValidationFailed  status: "True"  reason: DenyRuleViolation
#   message: "validation denied: field \"spec.domain\": spec.domain is required..."
```

Also visible in the Control Center **Conditions** tab.

**With the full Helm install + gateway** ([see root README](../README.md#running-the-examples)): rejected at apply time before it is stored.

---

## Step 7 — Push to the registry

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

---

## Step 8 — Inspect

```bash
ork inspect security-operator:0.1.0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next

[02 — Prometheus Operator](../02-prometheus/README.md) — `ServiceMonitor` and a `PrometheusRule` from `MonitoringConfig`.
