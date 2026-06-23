# 00 — ArgoCD

An `App` CR. That is all your team creates.

Orkestra reads it, renders an ArgoCD Application from the template in the Katalog, and creates it in the `argocd` namespace. ArgoCD picks it up and begins syncing. Your team never sees the Application spec — not the `source.repoURL` format, not the `syncPolicy` structure, not the `project` field.

The Katalog implements the mapping once. Everyone benefits.

---

> **Before you start:** Update [katalog.yaml](katalog.yaml) — set `author:` to your name or org. ArgoCD must be installed in the cluster (`argocd` namespace). If running without a cluster, `ork simulate` works without it.

---

## What the team creates

```yaml
apiVersion: ecosystem.demo.orkestra.io/v1alpha1
kind: App
metadata:
  name: my-webapp
spec:
  repo: "github.com/myorg/services"
  path: "apps/webapp"
  branch: "main"
  targetNamespace: "production"
  labels:
    team: platform
```

## What Orkestra creates

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-webapp
  namespace: argocd
spec:
  project: default
  source:
    repoURL: "https://github.com/myorg/services"
    targetRevision: "main"
    path: "apps/webapp"
  destination:
    server: https://kubernetes.default.svc
    namespace: "production"
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

---

## What the Katalog adds beyond creation

- **Admission**: any `App` without `spec.labels.team` is denied at apply time — all apps must declare ownership before ArgoCD sees them
- **Deletion protection**: the App CR cannot be deleted accidentally — protection must be explicitly disabled first
- **Status propagation**: `syncStatus` and `health` from the ArgoCD Application flow back into the App CR's status via `hasStatus: true`

---

## Step 1 — Install ArgoCD

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

Wait for ArgoCD pods to be ready

---

## Step 2 — Simulate

```bash
ork simulate
```

Proves the ArgoCD Application is created in cycle 1. No cluster, no ArgoCD installation needed.

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
kubectl get apps
kubectl get application my-webapp -n argocd
```

---

## Step 5 — Control center

In a second terminal:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081) — the App CR and its ArgoCD Application child are both visible.

---

## Step 6 — Try the admission rule

Apply an App without a team label:

```bash
kubectl apply -f - <<EOF
apiVersion: ecosystem.demo.orkestra.io/v1alpha1
kind: App
metadata:
  name: bad-app
  namespace: default
spec:
  repo: "github.com/myorg/services"
  path: "apps/webapp"
  branch: "main"
  targetNamespace: "production"
EOF
```

**With `ork run` (local, no gateway):** the CR is accepted by Kubernetes and stored in etcd, but Orkestra halts reconcile immediately on the `deny` rule — no ArgoCD Application is ever created. Check the status:

```bash
kubectl get app bad-app -o yaml | grep -A5 "conditions:"
```

Expected:

```text
conditions:
  - lastTransitionTime: "2026-06-21T06:23:05Z"
    message: 'validation denied: field "metadata.labels.team": declare metadata.labels.team
      — all apps must declare ownership before they are deployed (got "")'
    observedGeneration: 1
    reason: ReconcileError
```

**With the full Helm install + gateway** ([see root README](../README.md#running-the-examples)): the webhook intercepts the request before it is stored and rejects it at apply time:

```
Error from server: admission webhook "orkestra-admission-validation.orkestra.io" denied the request:
All apps must declare a team label (spec.labels.team)
```

---

## Step 7 — Push to the registry

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

---

## Step 8 — Inspect

```bash
ork inspect app-operator:0.1.0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next

[01 — cert-manager](../01-cert-manager/README.md) — same pattern, certificates instead of GitOps.
