# Conditional Reconciliation

Conditional reconciliation lets you gate an entire reconcile cycle on conditions evaluated **before** the reconciler is called. When a CR does not satisfy the conditions, the reconciler is skipped entirely — the object is not re-queued, no resources are created or deleted, and the CRD's health state stays idle (not degraded).

This is distinct from resource-level conditions (`when:` on individual resources inside `onCreate`/`onReconcile`). Those conditions are evaluated inside the reconciler. `operatorBox.preReconcile.when` is evaluated by the kordinator before the reconciler is even invoked.

---

## Declaring a pre-reconcile gate

```yaml
spec:
  crds:
    app:
      apiTypes:
        group: apps.demo.io
        kind: App
        version: v1alpha1
      operatorBox:
        reconcile:
          when:
            - field: "{{ .spec.enabled }}"
              equals: "true"
        onReconcile:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              reconcile: true
```

With this configuration:
- A CR with `spec.enabled: true` → reconciler runs, Deployment is created/updated.
- A CR with `spec.enabled: false` → reconciler is skipped, Deployment is not created. No error recorded. CRD health state is `gated`.

---

## Gate semantics

| Property | Behavior |
|---|---|
| **Scope** | Per-CR object — each CR's conditions are evaluated independently |
| **Phase** | Before the reconciler is called — after dequeue, before `safeReconcile` |
| **On gate** | Item is dropped from the queue without re-queuing. No error. No status write. |
| **Health state** | `gated` — healthy (not degraded), idle, with the gate reason reported |
| **On next change** | If the CR is updated (e.g. `spec.enabled` flipped to `true`), the object is re-enqueued and the gate is re-evaluated |

---

## Condition types

All condition operators available in resource-level `when:` blocks are available here — `equals`, `contains`, `exists`, `gt`/`lt`, `in`, `regex`, and more. See the [full operator reference](../../reference/schema/02-katalog/06-when-conditions.md#operators).

`anyOf:` (OR semantics) is also supported alongside `when:` (AND semantics):

```yaml
reconcile:
  when:
    - field: "{{ .spec.enabled }}"
      equals: "true"
  anyOf:
    - field: "{{ .spec.environment }}"
      equals: "production"
    - field: "{{ .spec.environment }}"
      equals: "staging"
```

Both blocks must pass for reconciliation to proceed.

---

## Gated state in Control Center

When a CRD is gated, the Control Center displays a **purple** badge and the gate reason:

```text
● gated   spec.enabled is false
```

The state is separate from healthy and degraded — it is idle, not an error. It clears automatically the next time a reconcile succeeds (e.g. the CR is updated with a passing value).

---

## Difference from resource-level conditions

| | `operatorBox.preReconcile.when` | `onCreate` / `onReconcile` resource `when:` |
|---|---|---|
| **Evaluated by** | Kordinator (before reconciler) | Reconciler (inside reconcile loop) |
| **Scope** | Entire reconcile cycle | Individual resource |
| **Effect** | Reconciler never called | Resource is skipped; other resources still created |
| **Health on gate** | `gated` (idle) | No effect on health |
| **Error on gate** | None | None |
| **Re-queue** | No — waits for next CR change event | Normal re-queue |

Use pre-reconcile gates when: the entire operator should stay dormant until a condition is met (feature flag, license, environment). Use resource-level conditions when: most resources should be created but some are optional.

---

## Testing gates

### Simulate (with `--envtest`)

Use `absent:` to assert a resource was never created when the gate fires:

```yaml
# simulate-gated.yaml
crFiles:
  - cr-app-disabled.yaml   # spec.enabled: false
expect:
  crds:
    app:
      absent:
        - resource: deployments
```

Run with:

```bash
ork simulate -f simulate.yaml --envtest
```

### E2E

Assert the `/katalog` runtime endpoint reports `gated: true` after patching the CR:

```yaml
steps:
  - kubectl:
      patch:
        resource: apps
        name: my-app
        patch: '{"spec":{"enabled":false}}'
  - kubectl:
      port-forward:
        pod-selector: "app=orkestra-leader"
        port: 8080
      assert:
        path: /katalog
        jq: '.crds.app.gated'
        equals: "true"
```

---

## Try it

```bash
ork init --pack intermediate
cd 05-when-conditions/conditional-reconciliation
```

The pack includes an App CRD (gated by `spec.enabled`), a Route CRD (unconditional), a Simulate suite that tests both pass and gate scenarios, and a minimal E2E.
