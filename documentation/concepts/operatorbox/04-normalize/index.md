# Normalize

`normalize` runs first in the reconcile pipeline — before mutation, validation, and every template expression in `onCreate`, `onReconcile`, and `status`. It rewrites the in-memory spec into a single canonical shape that all downstream stages rely on, without touching the stored CR in etcd.

**Accept many, work with one.** Users write their CR in whatever shape is natural to them. Normalize collapses that into one internal representation. Every rule, every template, every status field sees the normalized result — no branching, no defensive coding scattered across the Katalog.

---

## Declaration

```yaml
normalize:
  spec:
    fieldName: "{{ template expression }}"
```

Keys are dot-notation paths into `spec`. Values are template expressions evaluated against the raw CR — the object as it arrived from the API server, before any normalization of other fields. Results overwrite the corresponding fields in the in-memory spec copy. Fields not declared are left exactly as they are.

---

## Where normalize sits

```text
informer cache → DeepCopy → normalize → mutation → validation → reconcile
                                 ↑
                     Sees the raw CR. Rewrites in-memory.
                     Everything downstream sees the result.
```

See [Reconcile Pipeline](../01-reconcile-pipeline.md) for the full sequence.

---

## Try it

Four runnable examples, each focused on one pattern:

```bash
ork init --pack use-cases
cd normalize/01-string-cleanup            # toLower, trimSpace, domain cleanup
cd normalize/02-image-normalization       # bare image → registry.internal/name:tag
cd normalize/03-defaults-without-webhook  # default in normalize = mutation without gateway
cd normalize/04-webservice                # all patterns combined: secrets, configmap, forEach
ork run
```

Each example has two CRs — one messy, one clean — that both reconcile to identical downstream resources.

---

## Where to go next

- [Normalize vs Mutation](01-vs-mutation.md) — when to use each, and how `default` in normalize replaces webhook-based defaulting
- [Patterns](02-patterns.md) — format unification, string cleanup, type coercion, field assembly, and more
- [Reference](03-reference.md) — notes table, limitations, declaration syntax
