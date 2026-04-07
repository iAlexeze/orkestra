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
applied by the GitOps tool.

---

## 1. What goes where in Git

A GitOps repository for an Orkestra-managed platform has three distinct layers.

**Layer 1 — The operator runtime**

The Orkestra Deployment, its RBAC, and the Helm chart values that configure it.

**Layer 2 — The Katalog**

The operator definitions. This is what Orkestra reads at startup to know what
to manage.

**Layer 3 — The CRs**

The application team resources. These are the objects Orkestra manages.

---

## 2. Wave ordering in ArgoCD

The most important GitOps detail: Orkestra must be running before CRs are applied.

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

---

## 3. The Katalog in Git

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

---

## 4. ArgoCD health checks

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
