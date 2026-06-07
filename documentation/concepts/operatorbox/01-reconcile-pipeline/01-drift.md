# Drift

Orkestra's drift model is explicit: it compares specific fields per resource type and only writes what changed. Understanding this prevents two common surprises — changes not being applied to existing resources, and unmanaged fields being overwritten unexpectedly.

---

## onCreate vs onReconcile

Every resource declaration lives under one of two hooks.

### `onCreate`

The resource is **created once** if it does not exist. If it already exists, the create is skipped — no comparison, no update, no error.

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
```

Use this for resources that should not change after initial provisioning — a database seed job, a one-time migration, a generated secret.

### `onReconcile`

The resource is **drift-corrected on every reconcile**. Orkestra reads the live object, compares the fields it manages, and writes a patch only when drift is detected.

```yaml
onReconcile:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      replicas: "{{ .spec.replicas }}"
```

Use this for resources that should track the CR's spec — the main Deployment, a ConfigMap reflecting live configuration.

### `reconcile: true` on an onCreate entry

A shorthand meaning "create it if absent, then drift-correct it on every reconcile." Equivalent to declaring the same entry under both `onCreate` and `onReconcile`.

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .spec.image }}"
      reconcile: true
```

---

## What gets drift-corrected

For Deployments, ReplicaSets, and StatefulSets, Orkestra compares these fields on every `onReconcile` cycle and patches only what changed:

| Field group | Corrected |
|---|---|
| `spec.replicas` | yes |
| Container image | yes |
| Container resources (CPU / memory) | yes |
| Environment variables (`env:`, `envFrom:`) | yes |
| Volumes and volume mounts | yes |
| Probes (liveness, readiness, startup) | yes |
| Container security context | yes |
| Pod security context | yes |
| Service account name | yes |
| Node selector | yes |
| Labels | yes |

If nothing in this set changed, no Kubernetes API call is made — the reconcile is effectively a no-op for that resource.

### Pods

Pods are largely immutable in Kubernetes — most fields cannot be updated in place. Orkestra handles Pod drift by **deleting and recreating** the Pod when any template-declared field changes. This is the correct behavior for standalone Pods; workload controllers (Deployment, StatefulSet) handle rolling updates for the pods they own.

### Jobs

Jobs are create-only. Once a Job exists Orkestra does not update it — the Job runs to completion and Kubernetes tracks its status. A completed Job is not re-created unless the CR is deleted and recreated.

---

## Deleted externally

If a resource managed by `onReconcile` (or `onCreate` with `reconcile: true`) is deleted outside Orkestra — by `kubectl delete`, a namespace cleanup, or another controller — Orkestra recreates it on the next reconcile. The `Update` path detects `NotFound` and falls through to `Create`.

Resources managed by plain `onCreate` (no `reconcile: true`) are **not** recreated automatically. They are created once and then left alone.

---

## Condition semantics: when a `when:` condition fails

### On `onCreate` — condition fails

The resource is **skipped**. If it already exists it is left untouched.

### On `onReconcile` — condition fails

Orkestra calls `DeleteIfOwned` — but only if the resource's name is **not covered by any other passing declaration** in the same resource group. This enables mutually exclusive phase-based resource declarations without leaving orphans:

```yaml
onReconcile:
  deployments:
    - name: "{{ .metadata.name }}"
      image: nginx:stable
      when:
        - field: status.channel
          equals: "stable"

    - name: "{{ .metadata.name }}"
      image: nginx:canary
      when:
        - field: status.channel
          equals: "canary"
```

When `status.channel = stable`: the first passes and is applied; the second fails but the name is covered by the first — **no deletion**. When `status.channel` is neither value: both fail and the name is uncovered — `DeleteIfOwned` removes the Deployment.

`DeleteIfOwned` only deletes resources carrying the `orkestra-owner` label matching the owning CR. Manually created resources with the same name are never touched.

---

## `once: true` — frozen after creation

Secrets declared with `once: true` are written once and never overwritten by subsequent reconciles, regardless of what the template would produce.

```yaml
onCreate:
  secrets:
    - name: "{{ .metadata.name }}-api-key"
      once: true
      data:
        key: "{{ randomHex 32 }}"
```

On the first reconcile the secret is created with a random value. Every subsequent reconcile finds it exists and skips. The value is stable across CR updates, operator restarts, and spec changes.

`once: true` with `rotateAfter:` adds time-based rotation. When the secret's age exceeds the declared duration it is deleted and recreated with a new value on the next reconcile.

---

## Owner references and garbage collection

Every resource Orkestra creates carries a Kubernetes owner reference pointing to the CR and an `orkestra-owner` label. When the CR is deleted, Kubernetes garbage collects owned resources automatically — no `onDelete` block needed for the common case.

`onDelete` templates are for resources that need explicit cleanup before the CR finalizer is removed: external registrations, DNS records, cross-namespace resources that owner references cannot cascade to.

