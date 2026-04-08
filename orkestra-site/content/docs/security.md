---
title: "Security"
weight: 151
---

Here’s a **clean, tightened rewrite** of your Security doc that incorporates the new RBAC model and elevates it from “feature” → **core philosophy**.

I’ve kept your structure but sharpened the narrative and made the permission model feel intentional, not incidental.

---

# Security

Orkestra is designed for production from the start.
Security is not layered on later — it is part of the execution model.

This page covers:

* how Orkestra scopes permissions
* how to run with least privilege
* how TLS, webhooks, and external sources are secured

---

## The permission model

Orkestra does **not** require broad, wildcard permissions.

Instead, permissions are **derived directly from your Katalog**.

Every CRD you declare defines:

* what Orkestra watches
* what it creates
* what it updates

From that, Orkestra can compute the exact RBAC required.

This produces:

* `ServiceAccount`
* `ClusterRole`
* `ClusterRoleBinding`

All scoped **only to the resources your Katalog actually uses**.

---

## What Orkestra needs

Orkestra requires permissions in four areas — nothing more:

### 1. Managed CRDs

Read/write access to the resource types declared in your Katalog.

### 2. Leader election

```yaml
apiGroups: ["coordination.k8s.io"]
resources: ["leases"]
verbs: ["get", "create", "update"]
```

### 3. Event emission

```yaml
apiGroups: [""]
resources: ["events"]
verbs: ["create", "patch"]
```

### 4. Admission webhooks *(only if used)*

```yaml
apiGroups: ["admissionregistration.k8s.io"]
resources:
  - validatingwebhookconfigurations
  - mutatingwebhookconfigurations
verbs: ["get", "create", "update", "patch"]
```

If your Katalog does not define validation, mutation, or conversion rules,
these permissions are **not included**.

---

## From wildcard RBAC → precise RBAC

Traditional operators often ship with:

```yaml
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
```

This is simple — but unsafe.

Orkestra replaces this with **derived permissions**:

```yaml
- apiGroups: ["demo.orkestra.io", "platform.orkestra.io"]
  resources: ["websites", "applications"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

These rules:

* are generated automatically
* stay in sync with your Katalog
* shrink the blast radius if compromised

---

## Why Orkestra does this

Orkestra runs multiple CRDs in a single runtime.

That makes **permission accuracy critical**.

A wildcard role in a shared runtime would grant:

* unintended access across unrelated CRDs
* unnecessary privileges to external patterns
* increased risk in multi-tenant clusters

By deriving permissions from the Katalog:

* each deployment has a **bounded capability**
* external patterns cannot expand permissions silently
* security scales with composition

:::important
Rerun the `ork generate` command each time you add a new CRD, to update Orkestra permissions
:::

---

## Namespace restriction

Use `restrictedNamespaces` to prevent resource creation in sensitive areas:

```yaml
- name: website
  restrictedNamespaces:
    - kube-system
    - kube-*
    - "*-system"
```

Komposer-level restrictions apply globally and cannot be removed by CRDs.

---

## TLS certificates

Orkestra exposes a single HTTPS server (`:8443`) for:

* conversion webhooks
* validation webhooks
* mutation webhooks

All share **one certificate**.

### Options

| Approach     | Use case                 |
| ------------ | ------------------------ |
| Self-signed  | Development only         |
| cert-manager | Production (recommended) |
| External PKI | Enterprise environments  |

### Required SANs

```
DNS: orkestra.<namespace>.svc
DNS: orkestra.<namespace>.svc.cluster.local
```

---

## Credentials in sources

Authentication for external sources must use environment variables:

```yaml
auth:
  type: github
  fromEnv: GITHUB_TOKEN
```

In Kubernetes:

```yaml
env:
  - name: GITHUB_TOKEN
    valueFrom:
      secretKeyRef:
        name: orkestra-credentials
        key: github-token
```

---

## Admission webhook security

When enabled, Orkestra registers only the rules it needs.

* **FailurePolicy:** defaults to `Ignore`
* **Scope:** only CRDs with validation/mutation rules
* **TLS:** shared with conversion webhook

This avoids unnecessary API server calls and reduces failure impact.

---

## Supply chain security

### Binary verification

```bash
gpg --verify ork_linux_amd64.tar.gz.asc ork_linux_amd64.tar.gz
```

### Pattern pinning

```yaml
# Good
- url: ghcr.io/.../postgres@v1.2.0

# Avoid
- url: ghcr.io/.../postgres@latest
```

For Git:

```yaml
- url: https://github.com/myorg/registry@<commit-sha>
```

---

## Template safety

Katalog templates use Go `text/template`.

They:

* cannot execute commands
* cannot access the filesystem
* only read CR fields

---

## Logging and audit

* Structured logs via zerolog
* Per-reconcile visibility
* Kubernetes audit logs capture all API interactions

Integrate with your central logging system for alerting and traceability.

---

## Reporting vulnerabilities

Report privately with:

* reproduction steps
* logs
* impact assessment

Responsible disclosure protects users.

---

## 🔚 Closing note

Orkestra’s security model is simple:

> It only has the permissions required to do what your Katalog declares — nothing more.

That constraint is intentional.
It keeps the system predictable, auditable, and safe at scale.