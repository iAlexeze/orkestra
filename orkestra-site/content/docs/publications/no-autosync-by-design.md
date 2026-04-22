---
title: "No Autosync By Design"
weight: 65
---

# Orkestra Does Not Auto-Sync — And That’s the Point

*Orkestra Project — March 2026*

---

## Abstract

Most modern infrastructure tools default to continuous synchronization. If the source changes, the system updates. This model works well for stateless applications and fast-moving deployments.

Orkestra deliberately rejects this model.

Orkestra does not automatically re-fetch or re-apply your Katalog when a remote source changes. It only evaluates configuration at runtime start. After that, it focuses exclusively on one responsibility: reconciling Custom Resources in-cluster.

This is not a missing feature. It is a design decision rooted in how Orkestra defines production safety, API stability, and control over distributed systems.

---

## 1. The Industry Default: Continuous Sync

Tools like GitOps controllers, CI/CD pipelines, and deployment managers are built around a simple loop:

> Source changes → system detects → system applies

This works because the unit being deployed is typically:

* A stateless service
* A container image
* A short-lived configuration

Frequent updates are expected. Drift is undesirable. Automation is the goal.

But this model assumes something critical:

> That the source of truth is always safe to apply immediately.

For CRDs, this assumption breaks.

---

## 2. CRDs Are Not Deployments — They Are APIs

A Kubernetes Deployment can be rolled forward, rolled back, or replaced with minimal impact if designed correctly.

A Custom Resource Definition is different.

A CRD defines:

* An API contract
* A schema stored in etcd
* Long-lived state
* Behavioural guarantees through reconciliation

Changing a CRD is not a deploy. It is an API evolution.

Automatic updates in this context introduce risk:

* Schema drift
* Breaking field changes
* Incompatible reconciliation logic
* Silent behaviour changes across environments

In other words:

> Auto-syncing CRDs is equivalent to auto-deploying breaking API changes.

Orkestra refuses to do this.

---

## 3. Orkestra’s Model: Stability Over Reactivity

Orkestra evaluates all sources at startup:

* Local files
* Remote URLs
* Private APIs
* Helm charts
* OCI artifacts

It merges them into a single, validated runtime configuration.

Then it stops listening.

From that point forward:

> The runtime is fixed. Only the cluster state changes.

This creates a clear separation:

| Concern       | Responsibility            |
| ------------- | ------------------------- |
| Configuration | Resolved at startup       |
| Execution     | Continuous reconciliation |
| Updates       | Explicit human action     |

Orkestra does not poll GitHub.
It does not watch URLs.
It does not auto-reload.

Because none of those things are the problem it is trying to solve.

---

## 4. Purity of Input

A single Orkestra instance can compose multiple sources:

```yaml
sources:
  files:
    - https://raw.github.com/myorg/public-baseline/main/katalog.yaml

    - url: https://raw.githubusercontent.com/myorg/platform-crds/main/katalog.yaml
      auth:
        type: github
        fromEnv: GITHUB_TOKEN

    - url: https://api.security.myorg.io/orkestra/katalog.yaml
      auth:
        type: bearer
        fromEnv: SECURITY_API_TOKEN

    - url: https://artifactory.myorg.io/orkestra/compliance-katalog.yaml
      auth:
        type: basic
        usernameFromEnv: ARTIFACTORY_USER
        passwordFromEnv: ARTIFACTORY_PASSWORD

  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 3.2.1
```

This is not a single source of truth.
It is a *composition of truths*.

If Orkestra continuously tracked all of these:

* A transient network failure could partially update state
* A bad commit in one repo could corrupt the runtime
* A breaking change in one dependency could cascade silently

In a system managing multiple CRDs simultaneously:

> The correctness of the whole depends on the integrity of every input.

This is why Orkestra treats input as immutable during execution.

---

## 5. Production Is the Default, Not the Mode

Most systems treat production as a special case:

* “Enable safe deploy mode”
* “Add approval gates”
* “Turn on progressive rollout”

Orkestra inverts this.

Every run is treated as production.

Starting Orkestra is equivalent to a deployment:

* Sources are resolved
* Configuration is validated
* Runtime is constructed
* Reconciliation begins

Updating Orkestra is also a deployment:

* You restart with new inputs
* You consciously accept the change
* You control the timing

There is no background mutation.

---

## 6. Human-Controlled Evolution

Orkestra places the responsibility where it belongs:

> On the operator of the system.

If you want to update:

* You change a version
* You modify a Katalog
* You restart the runtime

Nothing happens implicitly.

This aligns with how APIs are evolved:

* Versioned deliberately
* Tested before rollout
* Rolled out with awareness of impact

Not triggered by a Git push.

---

## 7. What Would Auto-Sync Break?

If Orkestra auto-synced sources:

### 7.1 Non-deterministic runtime

Two identical deployments could behave differently depending on:

* When they last synced
* Which source updated first
* Network timing

### 7.2 Hidden failures

A remote change could:

* Introduce invalid configuration
* Break a conversion rule
* Remove required fields

Without any explicit deployment event.

### 7.3 Split-brain configuration

Multiple replicas could observe different states if sync timing differs.

### 7.4 Loss of auditability

You lose a clear answer to:

> “What version of the system is running right now?”

---

## 8. The Principle

Orkestra is built on a simple principle:

> Reconciliation should be continuous.
> Configuration should be intentional.

Everything else follows from this.

---

## 9. Conclusion

Orkestra does not auto-sync Katalogs because it is not a deployment tool.

It is a runtime for APIs.

Its responsibility is not to chase changes across distributed sources.
Its responsibility is to reconcile declared state reliably, predictably, and continuously.

By freezing configuration at startup and requiring explicit updates, Orkestra ensures:

* Deterministic behaviour
* Stable APIs
* Safe evolution
* Clear operational control

In a system where CRDs define long-lived state and business-critical behaviour, this is not conservatism.

It is correctness.
