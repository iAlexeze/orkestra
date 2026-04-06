---
title: "Orkestra and GitOps: The Complete Integration Story"
weight: 50
description: "*Orkestra Project — March 2026*"
---

*Orkestra Project — March 2026*

---

## Abstract

GitOps is how serious platform teams deploy to Kubernetes. A Git repository is
the source of truth. A tool — ArgoCD or Flux — watches the repository and applies
changes to the cluster. The question "how does Orkestra fit into my GitOps
workflow?" is asked by almost every engineer who evaluates it seriously.

The answer is: Orkestra is not just compatible with GitOps — it is a natural
fit. The Katalog is a manifest. CRs are manifests. Both live in Git. Both are
applied by the GitOps tool. The operator runtime itself is deployed as a standard
Kubernetes Deployment. But the integration has details that are worth understanding
before designing the repository structure.

---

## 1. What goes where in Git

A GitOps repository for an Orkestra-managed platform has three distinct layers.

**Layer 1 — The operator runtime**

The Orkestra Deployment, its RBAC, and the Helm chart values that configure it.
This is the operator itself — the running process that watches CRDs.

```
platform-gitops/
  infrastructure/
    orkestra/
      namespace.yaml
      helmrelease.yaml    # or helm/orkestra/values.yaml
      secrets/
        tls-cert.yaml     # SealedSecret or ExternalSecret
```

**Layer 2 — The Katalog**

The operator definitions. This is what Orkestra reads at startup to know what
to manage. The Katalog is a ConfigMap in cluster (mounted into the Orkestra pod)
or a file reference in the Helm chart.

```
platform-gitops/
  operator-definitions/
    katalog.yaml          # all CRD declarations
    # or composed via Komposer:
    komposer.yaml         # sources + overrides
```

**Layer 3 — The CRs**

The application team resources. These are the objects Orkestra manages —
`Website`, `Database`, `PlatformNamespace`. They live alongside other application
manifests.

```
platform-gitops/
  applications/
    team-platform/
      website.yaml
      database.yaml
    team-api/
      website.yaml
```

The three layers are deployed independently and have different update frequencies.
The Orkestra runtime changes rarely — only on Orkestra version upgrades. The Katalog
changes when platform behavior changes — new CRDs, new validation rules, new
defaults. CRs change whenever application teams update their desired state.

---

## 2. Wave ordering in ArgoCD

The most important GitOps detail: Orkestra must be running before CRs are applied.
A `Website` CR applied before Orkestra's informer is started is handled correctly
(it is queued when the informer syncs), but the CRD must exist before `kubectl
apply` accepts the CR.

ArgoCD sync waves handle this:

```yaml
# infrastructure/orkestra/helmrelease.yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "0"    # Orkestra runtime: first
```

```yaml
# operator-definitions/katalog-configmap.yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"    # Katalog: after runtime is up
```

```yaml
# applications/team-platform/website.yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "2"    # CRs: after CRD and operator ready
```

Wave 0 deploys Orkestra and its RBAC. Wave 1 applies the Katalog ConfigMap
(which triggers an Orkestra reload or restart). Wave 2 applies the CRs that
Orkestra will reconcile.

**Flux equivalent:**

```yaml
# Flux HelmRelease for Orkestra
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: orkestra
spec:
  dependsOn:
    - name: cert-manager     # if using cert-manager for TLS
---
# Flux Kustomization for CRs
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: platform-crds
spec:
  dependsOn:
    - name: orkestra         # CRs after Orkestra is ready
```

---

## 3. The Katalog in Git

The Katalog is the platform team's primary artifact — the declaration of what
operators exist and how they behave. Treating it as code with the same practices
applied to application code pays dividends.

**One Katalog per environment:**

Different environments often need different operator configuration — more workers
in production, shorter resync in staging, debug logging in development.

```
operator-definitions/
  base/
    katalog.yaml           # shared CRD declarations
  overlays/
    development/
      komposer.yaml        # sources: [../../base/katalog.yaml], overrides: workers: 1
    staging/
      komposer.yaml        # sources: [../../base/katalog.yaml], overrides: workers: 2
    production/
      komposer.yaml        # sources: [../../base/katalog.yaml], overrides: workers: 8
```

The Komposer `spec.crds` block is the override mechanism — it wins on name
conflict over every source. Environment-specific tuning belongs there.

**`ork validate` in the CI pipeline:**

Every pull request that touches operator definitions should run `ork validate`.
This catches configuration errors — circular dependencies, unknown kinds, invalid
template expressions — before they reach the cluster.

```yaml
# .github/workflows/validate.yaml
name: Validate Katalog
on:
  pull_request:
    paths:
      - 'operator-definitions/**'
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install ork
        run: curl -sSL https://raw.githubusercontent.com/orkestra-sh/orkestra/main/install.sh | bash
      - name: Validate
        run: ork validate --katalog operator-definitions/overlays/production/komposer.yaml
```

`ork validate` requires no cluster connection. It validates structure, enriches
built-in kinds from the internal registry, detects dependency cycles, and checks
that registry sources have the five required files. All of this runs in CI without
a kubeconfig.

**The PR review model:**

A change to a validation rule — from `action: warn` to `action: deny` — shows
as a one-line diff in the pull request. The reviewer sees exactly what changed
and what its effect will be. This is the GitOps promise applied to operator
behavior: every change is visible, reviewed, and traceable.

---

## 4. ArgoCD health checks

ArgoCD uses health checks to determine when a resource is healthy before
proceeding to the next sync wave. Out of the box, ArgoCD understands the
`Ready` condition on pods but not on custom resources.

Since Orkestra writes a standard `Ready` condition to every managed CR after
every successful reconcile, ArgoCD can be configured to check it:

```lua
-- argocd-cm.yaml — resource.customizations.health
health.lua: |
  hs = {}
  if obj.status ~= nil then
    if obj.status.conditions ~= nil then
      for i, condition in ipairs(obj.status.conditions) do
        if condition.type == "Ready" and condition.status == "False" then
          hs.status = "Degraded"
          hs.message = condition.message
          return hs
        end
        if condition.type == "Ready" and condition.status == "True" then
          hs.status = "Healthy"
          return hs
        end
      end
    end
  end
  hs.status = "Progressing"
  hs.message = "Waiting for reconcile"
  return hs
```

With this health check, ArgoCD's application view shows green for healthy CRs,
yellow for progressing (newly applied, not yet reconciled), and red for degraded
(reconcile failed, Ready=False in status).

This integrates directly with Orkestra's Layer 1 status management — the `Ready`
condition is written automatically on every reconcile, no Katalog configuration
required.

---

## 5. Multi-environment strategy

The recommended structure for multi-environment GitOps with Orkestra:

```
platform-gitops/
  infrastructure/
    orkestra/               # runtime — version pinned
      values-base.yaml      # shared Helm values
      values-prod.yaml      # production overrides (replicas: 3, resources: ...)
      values-staging.yaml

  operator-definitions/
    katalog.yaml            # base CRD declarations
    production.komposer.yaml   # workers: 8, validation: deny
    staging.komposer.yaml      # workers: 3, validation: warn
    development.komposer.yaml  # workers: 1, all warnings

  applications/
    production/
      websites/
        my-site.yaml        # spec.environment: production
    staging/
      websites/
        my-site.yaml        # spec.environment: staging
```

The key design decision: **validation severity is environment-specific.**

In development, run all rules as `warn` — developers need fast feedback without
being blocked. In staging, run deny rules but with a short denial list covering
only security-critical rules. In production, full deny enforcement.

This is expressed not by duplicating validation rules but by using `ENABLE_ADMISSION_WEBHOOK=true`
only in production, and using `action: warn` in the base Katalog with environment
overlays promoting selected rules to `action: deny`.

---

## 6. The operator lifecycle in GitOps

**When the Katalog changes:**

The Katalog is a ConfigMap mounted into the Orkestra pod. When the ConfigMap
changes (new CRD added, rule modified, worker count changed), ArgoCD or Flux
detects the diff and applies it. The ConfigMap update triggers an Orkestra
restart (or, in a future version, a live reload). The new configuration takes
effect without manual intervention.

**When the Orkestra version changes:**

The Helm chart version is pinned in the GitOps repository. Upgrading Orkestra
is a version bump in the HelmRelease manifest — reviewed in a PR, applied by
ArgoCD, rolled out as a standard Deployment rollout. If the new version has a
breaking change in the Katalog schema, `ork validate` in CI catches it before
the PR merges.

**When a CRD is added:**

Adding a new CRD to the platform is:
1. Add the CRD YAML to `infrastructure/crds/`
2. Add the CRD entry to the Katalog
3. Add example CRs for testing
4. PR review + `ork validate` in CI
5. Merge — ArgoCD applies CRD, Komposer reloads, Orkestra starts managing the new type

From PR to running operator: one review cycle.

---

## 7. Handling secrets

The `auth` block in Komposer sources references environment variables — credentials
are never in YAML. In a GitOps context, inject them from Kubernetes Secrets:

```yaml
# infrastructure/orkestra/values.yaml (Helm)
extraEnv:
  - name: GITHUB_TOKEN
    valueFrom:
      secretKeyRef:
        name: orkestra-credentials
        key: github-token
```

The Secret is managed by ExternalSecrets Operator, SealedSecrets, or your
organisation's secret management tool — never committed as plaintext.

---

## 8. Summary: the GitOps-native operator platform

Orkestra integrates with GitOps without friction because its fundamental
artifacts — the Katalog, the Komposer, the CRs — are all YAML manifests.
They live in Git. They are diffable, reviewable, and auditable.

The complete integration provides:

- **Every operator behavior change is a PR** — reviewed, CI-validated, traceable
- **ArgoCD health checks from CR status** — no custom health logic needed
- **Environment-specific configuration** via Komposer overlays
- **CI validation** with `ork validate` — no cluster needed
- **Automated upgrade path** — Orkestra version bumps are Helm chart updates
- **Secrets never in Git** — env vars from Kubernetes Secrets, managed by your existing secret tooling

The result is an operator platform that behaves exactly like the rest of your
GitOps-managed infrastructure: git-driven, auditable, and consistent across
environments.
