# 05 — All-in-One

This example shows the alternative to focused CRDs — one `PlatformResource` CRD that handles all four ecosystem integrations via a `spec.workloadType` discriminator.

---

## What you will learn

- How `when:` conditions on `custom:` entries route a single CRD to different ecosystem resources
- How a single motif import enforces organisation-wide admission rules across all workload types
- The trade-offs between this approach and focused CRDs (one per tool)

---

## Step 1 — Install the ecosystem tools

```bash
# ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# cert-manager
helm repo add jetstack https://charts.jetstack.io
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --set installCRDs=true

# Crossplane
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm upgrade --install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system --create-namespace
```

---

## Run the example

```bash
ork validate
kubectl apply -f crd.yaml

# Deploy an ArgoCD Application via PlatformResource
kubectl apply -f cr-app.yaml
kubectl get platformresources

# Deploy a Certificate via PlatformResource
kubectl apply -f cr-cert.yaml
kubectl get platformresources
```

---

## How it works

`spec.workloadType` determines which ecosystem resource is created:

| `workloadType` | Creates |
|---|---|
| `app` | ArgoCD `Application` |
| `cert` | cert-manager `Certificate` |
| `monitoring` | Prometheus `ServiceMonitor` |
| `infra` | Crossplane `InfraClaim` |

The `platform-admission` motif enforces `spec.team` presence and organisation-wide defaults for all types from a single import.

---

## Trade-offs

See [06-all-in-one.md](https://orkestra.sh/docs/guides/ecosystem/all-in-one) in the guide for the full comparison: focused CRDs vs. unified CRD, and when to choose each.

---

## Cleanup

```bash
kubectl delete -f cr-app.yaml
kubectl delete -f cr-cert.yaml
kubectl delete -f crd.yaml
```
