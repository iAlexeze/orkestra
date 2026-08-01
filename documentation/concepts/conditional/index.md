# Conditionals

Conditionals are the logic layer in Orkestra. They let you express *when* something should happen — without writing Go code.

A conditional is a `when:` or `anyOf:` block attached to a resource, a status field, or a hook. Orkestra evaluates it on every reconcile. If the condition passes, the block runs. If it fails, the block is skipped cleanly — no error, no partial state.

---

## Where conditionals work

| Location | Effect |
|---|---|
| `onCreate`, `onReconcile`, `onDelete` resources | Resource is created/updated/deleted only when conditions pass |
| `status.fields` entries | Status field is written only when conditions pass |
| Motifs | Conditions apply inside motif templates the same way |

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

## Where to go next

- [Resource Conditions](resource-conditions/) — conditional creation for `onCreate`, `onReconcile`, `onDelete`
- [Async Reconciliation](async-reconciliation/) — multi-phase workflows using `when:` gates
- [Status Conditions](status-conditions/) — state machines via `when:` on `status.fields`

---

## Try it

```bash
ork init --pack intermediate
cd 05-when-conditions
```

Follow the README — one CRD, three tiers, different resources at each tier, zero Go code.
