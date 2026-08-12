# Conditionals

Conditionals are the logic layer in Orkestra. They let you express *when* something should happen — without writing Go code.

A conditional is a `when:` or `anyOf:` block attached to a resource, a status field, or a hook. Orkestra evaluates it on every reconcile. If the condition passes, the block runs. If it fails, the block is skipped cleanly — no error, no partial state.

---

## `when:` — AND semantics

All conditions must pass. If any one fails, the block is skipped.

```yaml
services:
  - name: "{{ .metadata.name }}-public"
    type: LoadBalancer
    when:
      - field: spec.exposePublicly
        equals: "true"
      - field: spec.environment
        equals: "production"
```

This service is created only when `exposePublicly` is true **and** `environment` is `production`.

---

## `anyOf:` — OR semantics

At least one condition must pass.

```yaml
services:
  - name: "{{ .metadata.name }}-svc"
    anyOf:
      - field: spec.environment
        equals: "production"
      - field: spec.forceExpose
        NotEquals: "true"
```

The service is created when either condition is true.

---

## Condition operators

String matching (`equals`, `contains`, `prefix`, `suffix`, `regex`), existence (`exists`, `notExists`), numeric comparison (`gt`/`lt`/`gte`/`lte`/`between`), and set membership (`in`/`notIn`) are all available, each with a same-named shorthand field so you rarely need `operator:`/`value:` explicitly — see the [full operator reference](../../reference/schema/02-katalog/06-when-conditions.md#operators).

---

## Where conditionals work

| Location | Effect |
|---|---|
| `operatorBox.reconcile.when` | Entire reconcile cycle is skipped when conditions fail — CRD reports `gated` |
| `onCreate`, `onReconcile`, `onDelete` resources | Resource is created/updated/deleted only when conditions pass |
| `status.fields` entries | Status field is written only when conditions pass |
| Motifs | Conditions apply inside motif templates the same way |

## Where to go next

- [Resource Conditions](01-resource-conditions.md) — conditional creation for `onCreate`, `onReconcile`, `onDelete`
- [Async Reconciliation](02-async-reconciliation.md) — multi-phase workflows using `when:` gates
- [Status Conditions](03-status-conditions.md) — state machines via `when:` on `status.fields`
- [Conditional Reconciliation](04-conditional-reconciliation.md) — pre-reconcile gates via `operatorBox.reconcile.when`

---

## Try it

```bash
ork init --pack intermediate
cd 05-when-conditions
```

Follow the README — one CRD, three tiers, different resources at each tier, zero Go code.
