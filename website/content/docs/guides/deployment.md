---
title: "Deploying Orkestra"
weight: 1
description: "This document covers every deployment path — from a quick local test to a"
---

This document covers every deployment path — from a quick local test to a
production multi-cluster setup. Choose the path that matches your situation.

---

## Installation

### macOS (Homebrew)

```bash
brew tap orkspace/tap
brew install ork
```

### Linux / macOS (curl)

```bash
curl -sSL https://get.orkestra.sh | bash
```

### Options

```bash
# Review before running
curl -sSL https://get.orkestra.sh -o install.sh
less install.sh
bash install.sh

# Pin to a specific version
curl -sSL https://get.orkestra.sh | ORK_VERSION=v0.1.1 bash

# Install to a custom directory
curl -sSL https://get.orkestra.sh | ORK_INSTALL_DIR=~/.local/bin bash
```

### Verify the binary (recommended)

Every release is GPG-signed. To verify the binary before running it:

```bash
# Import the Orkestra public key (one time)
curl -sSL https://github.com/orkspace/orkestra/releases/download/v0.1.1/orkestra-public-key.asc | gpg --import

# Download the binary and its signature
curl -sSLO https://github.com/orkspace/orkestra/releases/download/v0.1.1/ork_linux_amd64.tar.gz
curl -sSLO https://github.com/orkspace/orkestra/releases/download/v0.1.1/ork_linux_amd64.tar.gz.asc

# Verify
gpg --verify ork_linux_amd64.tar.gz.asc ork_linux_amd64.tar.gz
# gpg: Good signature from "Orkestra Releases <releases@orkestra.io>"
```

### Confirm installation

```bash
ork version
```

## Quick local test

The fastest way to run Orkestra. No cluster deployment needed — the `ork`
binary is the operator.

### Requirements

You only need two things:

- **A Kubernetes cluster** (1.28+).  
  Works with [kind](https://kind.sigs.k8s.io/), [minikube](https://minikube.sigs.k8s.io/), [k3s](https://k3s.io/), or a managed cluster ([EKS](https://aws.amazon.com/eks/), [GKE](https://cloud.google.com/kubernetes-engine/), [AKS](https://azure.microsoft.com/en-us/products//kubernetes-service/)).

- **The [ork](#installation) CLI** – installed via Homebrew or the install script.

Orkestra automatically discovers your cluster from your kubeconfig — no extra setup required.

---

### Run your first Operator
```bash
# Scaffold a project
ork init my-operator && cd my-operator

# Apply the example CRD
kubectl apply -f examples/website/website-crd.yaml

# Run
ork run --file examples/website/website-katalog.yaml
```

Use this path for development, demos, and validating a Katalog before
deploying it to a cluster.

---

## Helm deployment

Helm is the recommended path for running Orkestra on a cluster.

### Add the chart repository

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm repo update
```

### Install with defaults

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace
```

This installs Orkestra with a starter Katalog. Replace it with your own
before running any real CRDs.

### Install with your own Katalog

```bash
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --values my-values.yaml
```

```yaml
# my-values.yaml
katalog:
  inline: |
    apiVersion: orkestra.orkspace.io/v1
    kind: Katalog
    metadata:
      name: my-katalog
    spec:
      crds:
        - name: website
          apiTypes:
            group: demo.orkestra.io
            version: v1alpha1
            kind: Website
            plural: websites
          operatorBox:
            onCreate:
              deployments:
                - image: "{{ .spec.image }}"
                  replicas: "{{ .spec.replicas }}"
                  reconcile: true
```

### Verify the deployment

```bash
kubectl get pods -n orkestra-system
# NAME                        READY   STATUS    RESTARTS
# orkestra-7d9b4c8f6d-xkj2p   1/1     Running   0
# orkestra-7d9b4c8f6d-m9r3t   1/1     Running   0

curl http://$(kubectl get svc orkestra -n orkestra-system -o jsonpath='{.spec.clusterIP}'):8080/ready
# {"ready":true}

curl http://$(kubectl get svc orkestra -n orkestra-system -o jsonpath='{.spec.clusterIP}'):8080/katalog | jq
```

---

## Managing your Katalog

There are three ways to provide the Katalog to a Helm-deployed Orkestra.

### Option 1 — Inline in values (simplest)

Embed the Katalog YAML directly in your `values.yaml`. The chart creates
a ConfigMap from it. Good for small operators and getting started.

```yaml
katalog:
  inline: |
    apiVersion: orkestra.orkspace.io/v1
    kind: Katalog
    ...
```

**Updating:** change `values.yaml` and run `helm upgrade`. The ConfigMap
updates and the Deployment rolls automatically.

### Option 2 — External ConfigMap (recommended for GitOps)

Create the ConfigMap separately — managed by ArgoCD, Flux, or kubectl —
and tell Helm to use it:

```yaml
katalog:
  existingConfigMap: platform-katalog
  configMapKey: katalog.yaml
```

This decouples the Katalog lifecycle from the Helm release. The Katalog
can be updated without touching the Helm chart at all.

```bash
# Update the Katalog
kubectl apply -f my-katalog-configmap.yaml -n orkestra-system

# Restart Orkestra to pick up the change
kubectl rollout restart deployment/orkestra -n orkestra-system
```

### Option 3 — Remote URL (Komposer)

Point a Komposer at a remote Katalog URL. Orkestra fetches it at startup.
This is how platform teams distribute shared CRD definitions across clusters.

```yaml
# komposer-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: platform-komposer
  namespace: orkestra-system
data:
  katalog.yaml: |
    apiVersion: orkestra.orkspace.io/v1
    kind: Komposer
    metadata:
      name: platform-komposer
    sources:
      files:
        - https://config.platform.myorg.io/crds/platform-katalog.yaml
    spec:
      crds: []
```

---

## Multi-CRD operators with Komposer

When your operator manages more than one CRD, use a Komposer to compose
definitions from multiple Katalogs. This is the recommended pattern for
platform teams.

### Why Komposer over a large Katalog

A large Katalog with ten CRD entries becomes hard to maintain. Different
teams own different CRDs. The website CRD is owned by the application team.
The namespace CRD is owned by the platform team. The database CRD is owned
by the data team.

With Komposer, each team maintains their own Katalog. The platform's Komposer
composes them:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: platform-komposer
sources:
  files:
    # Each team's Katalog — they own these files independently
    - https://raw.github.com/myorg/app-crds/main/katalog.yaml
    - https://raw.github.com/myorg/platform-crds/main/katalog.yaml
    - https://raw.github.com/myorg/data-crds/main/katalog.yaml
    # Security team provides theirs via environment variable
    - $SECURITY_KATALOG_URL

spec:
  crds:
    # Production environment override — more workers than the shared default
    - name: application
      workers: 8
      apiTypes:
        group: platform.myorg.io
        version: v1alpha1
        kind: Application
        plural: applications
      operatorBox:
        default: true
```

### Dependency ordering across Komposer sources

Dependencies declared in any source Katalog are respected across the full
merged set. If the application Katalog declares `dependsOn: [project]` and
the project Katalog comes from a different source, Orkestra resolves it
correctly after merging.

```yaml
# app-katalog.yaml — owned by application team
spec:
  crds:
    - name: application
      dependsOn: [project]   # project comes from platform-katalog.yaml

# platform-katalog.yaml — owned by platform team
spec:
  crds:
    - name: project
      dependsOn: []
```

After merging, `project` starts first. `application` waits.

### Preview the merged result before deploying

Always validate and preview before applying:

```bash
ork validate --file komposer.yaml

ork template --file komposer.yaml --graph
# Dependency Graph:
# project
# application
#   └─ project

ork template --file komposer.yaml --json | jq '.[].name'
# "project"
# "application"
```

---

## Remote Katalog sources

Orkestra resolves sources at startup. Any change to a remote Katalog takes
effect the next time Orkestra starts — either from a pod restart or a
Deployment rollout.

### Public URLs

```yaml
sources:
  files:
    - https://raw.githubusercontent.com/myorg/crds/main/katalog.yaml
```

### Environment variables

```yaml
sources:
  files:
    - $PLATFORM_KATALOG_URL
    - $SECURITY_KATALOG_URL
```

Set the environment variables in the Helm values:

```yaml
extraEnv:
  - name: PLATFORM_KATALOG_URL
    value: https://config.platform.myorg.io/katalog.yaml
  - name: SECURITY_KATALOG_URL
    valueFrom:
      secretKeyRef:
        name: katalog-urls
        key: security
```

### Helm chart sources

```yaml
sources:
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 2.1.0
      valueFiles:
        - ./values/production.yaml
```

---

## Authenticated remote sources

Private Katalog files behind authentication require credentials. Orkestra
supports per-source authentication via environment variables.

### Bearer token

```yaml
sources:
  files:
    - url: https://internal.company.com/crds/platform-katalog.yaml
      auth:
        type: bearer
        fromEnv: PLATFORM_KATALOG_TOKEN
```

```yaml
# Helm values — inject the token from a Kubernetes Secret
extraEnvFrom:
  - secretRef:
      name: katalog-auth-tokens
```

```bash
# Create the Secret
kubectl create secret generic katalog-auth-tokens \
  --namespace orkestra-system \
  --from-literal=PLATFORM_KATALOG_TOKEN=your-token-here
```

### GitHub token (private repos)

```yaml
sources:
  files:
    - url: https://raw.githubusercontent.com/myorg/private-crds/main/katalog.yaml
      auth:
        type: github
        fromEnv: GITHUB_TOKEN
```

GitHub sends a `Authorization: Bearer <token>` header automatically when
the `type: github` auth is used. The token needs `repo` scope for private
repositories.

### Basic auth

```yaml
sources:
  files:
    - url: https://artifactory.company.com/orkestra/katalog.yaml
      auth:
        type: basic
        usernameFromEnv: ARTIFACTORY_USER
        passwordFromEnv: ARTIFACTORY_PASSWORD
```

### Injecting credentials securely

Never put credentials in values files that are committed to source control.
Use Kubernetes Secrets and inject via `extraEnvFrom`:

```bash
kubectl create secret generic orkestra-katalog-creds \
  --namespace orkestra-system \
  --from-literal=PLATFORM_KATALOG_TOKEN=ghp_xxxx \
  --from-literal=SECURITY_KATALOG_TOKEN=bearer_yyyy \
  --from-literal=ARTIFACTORY_USER=svc-orkestra \
  --from-literal=ARTIFACTORY_PASSWORD=xxxx
```

```yaml
# values.yaml
extraEnvFrom:
  - secretRef:
      name: orkestra-katalog-creds
```

---

## Production checklist

Before running Orkestra in production:

**Cluster**
- [ ] CRDs applied to the cluster (`kubectl apply -f my-crd.yaml`)
- [ ] Namespace created (`kubectl create ns orkestra-system`)
- [ ] RBAC reviewed — Orkestra needs cluster-wide watch permissions

**Helm values**
- [ ] `replicaCount: 2` or higher for HA
- [ ] `leaderElection.enabled: true`
- [ ] `pdb.enabled: true` with `minAvailable: 1`
- [ ] Resource requests and limits set appropriately
- [ ] `config.logLevel: warn` or `error` (not `debug` in production)

**Katalog**
- [ ] Validated with `ork validate --file <path>`
- [ ] Dependency graph reviewed with `ork template --graph`
- [ ] Remote source URLs verified and accessible from the cluster
- [ ] Auth credentials injected via Kubernetes Secrets (not hardcoded)

**Observability**
- [ ] Prometheus scrape configured for `/metrics`
- [ ] Health probes reaching `/health` and `/ready`
- [ ] Alerts on `controller_reconcile_total{result="error"}`

---

## High availability

Orkestra uses leader election for HA. Run at least
two replicas — only one leads, the others keep warm informer caches.
On leadership loss, a follower takes over in milliseconds.

```yaml
# Helm values for HA
replicaCount: 3

leaderElection:
  enabled: true
  leaseDuration: 15s
  renewDeadline: 10s
  retryPeriod: 2s

pdb:
  enabled: true
  minAvailable: 2

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values: [orkestra]
        topologyKey: kubernetes.io/hostname
```

The `requiredDuringScheduling` anti-affinity prevents two Orkestra pods
from landing on the same node. If a node goes down, one pod survives.

---

## GitOps with ArgoCD or Flux

### ArgoCD

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: orkestra
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://orkspace.github.io/orkestra
    chart: orkestra
    targetRevision: 0.1.0
    helm:
      valuesFiles:
        - values/production.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: orkestra-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

Manage the Katalog as a separate ArgoCD Application pointing at a ConfigMap
in your infrastructure repo. This separates Orkestra lifecycle from Katalog
lifecycle — upgrade Orkestra without touching CRD definitions, and update
CRD definitions without touching the Orkestra deployment.

### Flux

```yaml
# HelmRelease
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: orkestra
  namespace: orkestra-system
spec:
  interval: 1h
  chart:
    spec:
      chart: orkestra
      version: "0.1.0"
      sourceRef:
        kind: HelmRepository
        name: orkestra
        namespace: flux-system
  valuesFrom:
    - kind: ConfigMap
      name: orkestra-values
```

```yaml
# HelmRepository
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: orkestra
  namespace: flux-system
spec:
  interval: 1h
  url: https://orkspace.github.io/orkestra
```

---

## Multi-cluster

Each cluster runs its own Orkestra instance. There is no cross-cluster
coordination — Orkestra manages CRDs within one cluster only.

The Komposer pattern handles environment-specific configuration cleanly:

```
clusters/
  production/
    values.yaml         # production overrides (workers: 8, logLevel: warn)
    komposer.yaml       # points to shared sources + prod overrides
  staging/
    values.yaml         # staging overrides (workers: 4, logLevel: info)
    komposer.yaml       # same sources, different overrides
  development/
    values.yaml         # dev overrides (workers: 2, logLevel: debug)
    komposer.yaml       # same sources, different overrides
```

```yaml
# clusters/production/komposer.yaml
sources:
  files:
    - https://shared.platform.myorg.io/crds/katalog.yaml
spec:
  crds:
    - name: application
      workers: 8    # production needs more workers
      ...
```

```yaml
# clusters/development/komposer.yaml
sources:
  files:
    - https://shared.platform.myorg.io/crds/katalog.yaml
spec:
  crds:
    - name: application
      workers: 2    # development is fine with 2
      ...
```

The shared Katalog is defined once. Each environment overrides only what
differs. The merge rules ensure the inline override always wins.

---

## Upgrading

### Upgrade the Helm chart

```bash
helm repo update
helm upgrade orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --values my-values.yaml
```

This performs a rolling update — new pods start before old ones stop.
Leader election ensures continuous operation throughout.

### Upgrade the Katalog only

If you only changed the Katalog (not the Helm chart version):

```bash
# Update the ConfigMap
kubectl apply -f my-katalog-configmap.yaml -n orkestra-system

# Restart to pick up the change
kubectl rollout restart deployment/orkestra -n orkestra-system

# Watch the rollout
kubectl rollout status deployment/orkestra -n orkestra-system
```

### Verify after upgrade

```bash
# Check pods are healthy
kubectl get pods -n orkestra-system

# Check health API
curl localhost:8080/katalog | jq '.crds[].name'

# Check for any reconcile errors
curl localhost:8080/metrics | grep 'controller_reconcile_total{.*result="error"}'
```