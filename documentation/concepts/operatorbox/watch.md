# Arbitrary Watch

An operatorBox reconciler normally runs when its own CR changes. `operatorBox.watch` extends this: declare secondary Kubernetes resources that, when they change, re-enqueue the primary CR for reconciliation.

No Go code is required. The feature is purely declarative.

---

## Why it exists

Many real operators react to resources they do not own:

- An app operator that reads a shared `ConfigMap` of feature flags — if the ConfigMap changes, every CR must reconcile.
- A database operator that watches `Nodes` — node capacity changes affect pod scheduling, so each database CR must re-evaluate.
- A workload operator that watches a `Secret` managed by cert-manager — when the Secret rotates, the operator must restart the relevant workload.

Without arbitrary watch, authors work around this by polling in a hook, or by adding a finalizer or ownerReference to an object they do not logically own. Both are fragile.

With `operatorBox.watch`, Orkestra manages the secondary informer and routes events back to the right primary CR.

---

## How it works

For each entry in `operatorBox.watch`, Orkestra starts a dynamic shared informer scoped to that resource type and optional namespace. Events that arrive during the initial cache sync (the List phase) are dropped — the same behavior as controller-runtime's `source.Kind`. Only events that arrive after sync trigger re-enqueues.

```yaml
operatorBox:
  watch:
    - apiVersion: v1
      kind: ConfigMap
      name: feature-flags
      namespace: config
      on: [update]
```

When the `config/feature-flags` ConfigMap is updated, the informer fires an `UpdateFunc`. Orkestra then resolves which primary CR(s) to enqueue and adds them to the workqueue.

---

## Key resolution

Orkestra resolves the primary CR key from the watched object using a four-step chain (first match wins):

### 1. `keyFrom.label`

The watched object carries a label whose value is the primary CR key. Use this when the object is not owned by the primary CR but is labelled to declare ownership.

```yaml
watch:
  - apiVersion: v1
    kind: Secret
    namespace: certs
    keyFrom:
      label: app.kubernetes.io/cr-owner
```

If the Secret has label `app.kubernetes.io/cr-owner: default/myapp`, then `default/myapp` is enqueued.

### 2. `keyFrom.name`

A fixed primary CR name. Use this for singleton patterns — a single well-known resource that, when it changes, always means a specific primary CR must reconcile.

```yaml
watch:
  - apiVersion: v1
    kind: ConfigMap
    name: global-config
    namespace: config
    keyFrom:
      name: my-operator
      namespace: default
```

Every update to `global-config` enqueues `default/my-operator`.

### 3. ownerReference

No `keyFrom` is set, but the watched object's `ownerReferences` list contains an entry whose `apiVersion` and `kind` match the primary CRD. The named owner is enqueued.

This is the most common case when the primary CR created the watched object and set an ownerReference. It mirrors controller-runtime's `EnqueueRequestForOwner`.

### 4. Broadcast

None of the above matched. Orkestra enqueues all currently known primary CRs of this type.

This is the right default for truly shared resources — cluster Nodes, cluster-wide ConfigMaps — where a change is relevant to every instance. It mirrors controller-runtime's `EnqueueRequestsFromMapFunc` with a "return all" mapper.

---

## Comparison with controller-runtime

| Pattern | controller-runtime | Orkestra |
|---|---|---|
| Watch owned objects | `Owns(T)` | ownerReference path (automatic) |
| Watch unowned objects, map to owner | `Watches(T, EnqueueRequestForOwner)` | `watch:` + ownerReference |
| Watch unowned objects, custom key | `Watches(T, EnqueueRequestsFromMapFunc)` | `watch:` + `keyFrom.label` or `keyFrom.name` |
| Watch cluster-wide resource, fan-out | `Watches(T, mapFunc that returns all)` | `watch:` (broadcast is the default fallback) |

The key difference: controller-runtime requires Go. Orkestra watch is declarative YAML — the informer, the event filter, and the key-resolution strategy are all expressed in the katalog.

---

## Event filtering with `on:`

By default, all three event types (`create`, `update`, `delete`) trigger re-enqueues. Restrict this with `on:`:

```yaml
watch:
  - apiVersion: apps/v1
    kind: Deployment
    on: [update]        # only updates re-enqueue; create and delete do not
```

---

## Interaction with `preReconcile.enqueueGate`

A watch-triggered enqueue goes through the same `preReconcile.enqueueGate` as any other update event. The gate sees the current state of the primary CR, not the watched object. Sentinels (`generationChanged`, `labelsChanged`, etc.) reflect the primary CR's own metadata delta from the previous reconcile, not the watched object's delta.

If you need to gate on the watched object's state, use a `when:` condition inside the reconcile template that reads the appropriate cross-CRD or external field.

---

## Schema reference

→ [operatorBox.watch schema](../../reference/schema/02-katalog/27-watch.md)
