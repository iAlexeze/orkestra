# operatorBox.watch

`operatorBox.watch` declares secondary Kubernetes resources whose changes should re-enqueue the primary CR. Useful when a CR's reconcile outcome depends on objects it does not own — shared ConfigMaps, cluster-wide Nodes, Secrets managed by another operator.

No Go code is required. Orkestra creates a dynamic informer per entry and enqueues the relevant primary CR key(s) when an event fires.

---

## Declaration

```yaml
spec:
  crds:
    app:
      operatorBox:
        watch:
          - apiVersion: apps/v1
            kind: Deployment
            namespace: default
            on: [update]

          - apiVersion: v1
            kind: ConfigMap
            name: feature-flags
            namespace: config
            on: [update, delete]
            keyFrom:
              label: app.kubernetes.io/cr-owner

          - apiVersion: v1
            kind: Node
            on: [create, update, delete]
```

---

## `watch[]` fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | string | yes | API version of the watched resource, e.g. `apps/v1`, `v1`. |
| `kind` | string | yes | Kind of the watched resource, e.g. `Deployment`, `ConfigMap`. |
| `namespace` | string | no | Restrict the watch to this namespace. Omit for cluster-scoped or all-namespace watching. |
| `name` | string | no | Watch a single named instance. When set, the informer scopes to that object. |
| `on` | `[]string` | no | Event types to react to. Values: `create`, `update`, `delete`. Defaults to all three when omitted. |
| `keyFrom` | [WatchKeyFrom](#watchkeyfrom) | no | Override the default key-resolution strategy. See below. |
| `include` | string | no | Path to a YAML file whose `watch:` list replaces this entry. See [`include`](#include). |

Each `(apiVersion, kind, namespace)` combination must be unique across the `watch` list.

---

## `include`

An entry with only `include:` set is replaced in-place by the `watch:` list from the referenced file. All other fields on that entry are ignored. The path is resolved relative to the Katalog file.

```yaml
# katalog.yaml
operatorBox:
  watch:
    - include: ./shared-watches.yaml
    - apiVersion: v1
      kind: Secret
      name: api-credentials
      namespace: default
```

```yaml
# shared-watches.yaml
watch:
  - apiVersion: apps/v1
    kind: Deployment
  - apiVersion: v1
    kind: ConfigMap
    index:
      - name: metadata.labels.app
        field: metadata.labels.app
```

Multiple targets that share the same watched types can point to the same file instead of repeating entries:

```yaml
serve:
  target:
    v2-enabled:
      operatorBox:
        watch:
          - include: ./shared-watches.yaml
    v2-disabled:
      operatorBox:
        watch:
          - include: ./shared-watches.yaml
```

---

## Key resolution

When an event fires on a watched object, Orkestra resolves which primary CR(s) to enqueue using this order (first match wins):

1. **`keyFrom.label`** — a label on the watched object carries the primary CR key. Useful when the object is not owned by the primary CR but is labelled to indicate which CR it belongs to.

2. **`keyFrom.name`** — a fixed primary CR name declared in the watch entry. Useful for singleton or well-known CRs.

3. **ownerReference** — the watched object's `ownerReferences` contains an entry whose `apiVersion` and `kind` match the primary CRD. The referencing CR is enqueued by name.

4. **broadcast** — none of the above matched. All currently known primary CRs are enqueued. The right default for shared resources (cluster Nodes, shared ConfigMaps) that affect every CR equally.

---

## `WatchKeyFrom`

Overrides key resolution to steps 1 or 2 above. Exactly one of `label` or `name` must be set.

```yaml
watch:
  - apiVersion: v1
    kind: ConfigMap
    keyFrom:
      label: app.kubernetes.io/cr-owner   # OR
      name: my-singleton                   # but not both
      namespace: default                   # only with name
```

| Field | Type | Description |
|-------|------|-------------|
| `label` | string | Label key on the watched object whose value is the primary CR key (e.g. `namespace/name` or bare `name`). |
| `name` | string | Name of the primary CR to enqueue regardless of which watched object changed. |
| `namespace` | string | Namespace of the primary CR. Only meaningful with `name`. Has no effect when `label` is set. |

### Validation rules

- Exactly one of `label` or `name` must be set — both or neither is an error.
- `namespace` combined with `label` is rejected: label resolution reads the key from the object, so a namespace restriction has no meaning there.

---

## Interaction with `preReconcile.enqueueGate`

A watch-triggered enqueue goes through the same `preReconcile.enqueueGate` as a normal update enqueue. If the gate is configured with sentinels, those sentinels reflect the state of the **primary** CR at the time of re-enqueue, not the watched object.

---

## `preReconcile` gates and `failPolicy`

`preReconcile.enqueueGate` and `preReconcile.reconcileGate` both accept a `failPolicy:` field that controls what the gate does when it cannot evaluate — for example when an `external:` call fails or times out.

| Value | Behaviour |
|-------|-----------|
| `open` (default) | Evaluation failure passes the gate — the CR is enqueued / reconciled as if the gate was not declared. |
| `closed` | Evaluation failure holds the gate — the CR is dropped from the queue or held back from the reconciler. |

```yaml
operatorBox:
  preReconcile:
    reconcileGate:
      failPolicy: closed      # unknown state → hold back, do not reconcile
      external:
        - name: depHealth
          url: "{{ .spec.dependencyUrl }}/health"
      when:
        - field: external.depHealth.status
          equals: "200"
```

`open` is the safe default for `enqueueGate` — if you cannot evaluate, let the object through. `closed` is the right choice for `reconcileGate` when reconciling against an unknown dependency state is worse than missing a reconcile cycle.

### Validator warnings

`ork validate` emits a warning when:

1. `external:` calls are declared on a gate but `failPolicy` is omitted — the default is `open`, which may not be the intent for `reconcileGate`.
2. `failPolicy: closed` is declared but all `external:` calls have `continueOnError: true` — `continueOnError` suppresses call errors before they reach the gate, so `closed` will never trigger. Use `when: external.*.error` conditions instead.

---

## Validation

`ork validate` enforces:

- `apiVersion` and `kind` are present on every entry.
- `on:` values are one of `create`, `update`, `delete`.
- No two entries share the same `(apiVersion, kind, namespace)`.
- `keyFrom`, when present, has exactly one of `label` or `name`.
- `failPolicy`, when present, is one of `open`, `closed`.
