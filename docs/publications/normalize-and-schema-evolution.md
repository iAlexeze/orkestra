# Schema Evolution Without Webhooks: The Normalize Model

*Orkestra Project — April 2026*

---

## Abstract

Kubernetes API versioning imposes a specific burden on operator authors: when
a CRD's schema changes between versions, a conversion webhook must be deployed,
maintained, and kept reachable for every API server request involving the old
version. This infrastructure cost is proportional to the number of schema
changes, not to the complexity of the changes themselves. Orkestra introduces
a complementary model: the `normalize:` block, which transforms multiple valid
input shapes into a single canonical form before any reconcile logic runs. For
operators that accept flexible input formats within a single API version, this
eliminates the conversion webhook entirely. For operators that require
bidirectional version conversion, it simplifies the reconcile path by ensuring
that all downstream logic operates on a single normalized representation. This
paper describes the normalize model, its relationship to API conversion, and
the conditions under which each approach is appropriate.

---

## 1. Two distinct schema problems

API versioning and schema flexibility are related but distinct problems. Confusing
them leads to over-engineering in one direction and under-engineering in another.

**API versioning** is a contract problem. When a CRD exposes `v1` and `v2` APIs
simultaneously, Kubernetes must be able to convert any stored object between
versions on demand. A client requesting `v1` must receive a valid `v1` object
regardless of which version was used to create it. This requires bidirectional
conversion with round-trip fidelity guarantees. The conversion webhook is the
correct solution to this problem. Orkestra also solves this declaratively using
a crd `conversion:` block without webhook overhead.

**Schema flexibility** is a usability problem. Some fields naturally accept
multiple representations. A schedule can be expressed as a cron string
(`"*/5 * * * *"`) or as a structured object (`{minute: "*/5", hour: "*", ...}`).
A resource quantity can be expressed as a string (`"100m"`) or as a number
(`0.1`). These are not different API versions — they are different syntactic
sugar for the same semantic value. The user who writes the structured form and
the user who writes the string form both intend the same thing. The operator
should accept both without requiring the user to know which form is canonical.

Traditional operator frameworks have no direct answer to the schema flexibility
problem. Operators handle it with `if typeof(x) == "map"` branches scattered
throughout their reconcile code, or by restricting the API to one form and
forcing users to convert before submitting. In some cases, it leads to another API version.

---

## 2. The normalize block

The `normalize:` block in Orkestra's Katalog runs as the first step of the
operatorBox: reconcile pipeline — before mutation, validation, and template
rendering. It transforms fields in the CR's spec into a canonical form using
the same template `notes` available everywhere in the Katalog.

```yaml
operatorBox:
  normalize:
    spec:
      schedule: >
        {{ if typeMap .spec.schedule }}
          {{ cronFromMap .spec.schedule }}
        {{ else }}
          {{ cronNormalize .spec.schedule }}
        {{ end }}
```

After this block runs, `.spec.schedule` is always a plain cron string. Every
downstream step — mutation rules, validation rules, status field expressions,
onCreate templates, onReconcile templates — sees `"*/5 * * * *"`, not
`map[minute:*/5 hour:* dayOfMonth:* month:* dayOfWeek:*]`. The branching on
input format exists in exactly one place. The rest of the Katalog is clean.

The normalized form lives only in memory. The raw CR in etcd is untouched. When
the API server returns the CR to a client, the client sees the original form the
user submitted. Normalization is purely an internal reconcile concern.

---

## 3. Pipeline position and semantics

The normalize block occupies a specific position in the operatorBox: reconcile
pipeline:

```
informer cache → DeepCopy → normalize → mutation → validation → template rendering
```

This ordering is deliberate. Normalize runs on the raw spec — the exact form the
user submitted. Mutation then runs on the normalized spec, applying defaults to
fields that are absent or in their canonical empty state. Validation runs on the
normalized, mutated spec and can therefore make assumptions about field types.
Template rendering runs on the normalized, mutated, validated spec and operates
on a single canonical representation.

If normalize ran after mutation, fields added by mutation defaults would be
expressed in the pre-normalization form and might need to be re-normalized.
If normalize ran after validation, validation would need to handle all input
forms rather than just the canonical form. The current ordering is the only
one that produces clean semantics at each pipeline stage.

---

## 4. Normalize vs conversion: when to use each

The two mechanisms are not alternatives. They solve different problems and should
be used together where appropriate.

**Use normalize when:**

The CRD has a single API version, and multiple input formats are accepted for
convenience. The user writing `spec.schedule: "*/5 * * * *"` and the user writing
the structured form are submitting semantically identical requests. Both are using
the same API version. No Kubernetes-level conversion is needed.

The test: if removing the second input format would not require a CRD version bump
— because it was never an API contract, just a convenience — then normalize is
the right tool.

**Use conversion when:**

The CRD has published multiple API versions with different field names or
structures, and existing `kubectl apply` files reference the old version. Users
have version-pinned clients. The Kubernetes API server must serve requests at
the old version correctly. The contract is bidirectional: reads and writes at
both versions must work.

The test: if a client with `apiVersion: demo.orkestra.io/v1` in its YAML file
must continue to work after the schema changes, conversion is required.

**Use both when:**

The CRD has published `v1` (structured schedule) and `v2` (cron string), and
within `v2`, both representations are still accepted. The conversion webhook
handles the bidirectional `v1 ↔ v2` translation. The normalize block handles
the `v2 map → v2 string` transformation within the single stored version.
The onReconcile templates see only the normalized canonical string.

This combination, demonstrated in the CronJob pattern, produces the cleanest
reconcile code: the conversion block handles API versioning at the boundary,
and the normalize block handles representation flexibility in the interior.

---

## 5. The typeOf primitive

The normalize block in the CronJob example uses `typeMap`, a shorthand for `typeOf` note
that returns the runtime type of any value as a string: `"string"`, `"map"`,
`"slice"`, `"number"`, `"bool"`, or `"null"`.

```yaml
{{ if typeMap spec.schedule }} is equivalent to: {{ if eq (typeOf .spec.schedule) "map" }}
```

This is the same `typeOf` available in `when:` conditions:

```yaml
when:
  - field: spec.schedule
    operator: typeMap

# is equivalent to:

when:
  - field: spec.schedule
    operator: typeOf
    value: map
```

The dual availability — as a template function for value expressions and as a
condition operator for boolean branching — follows Orkestra's general design
principle: any mechanism available in one context is available in all contexts.
A user who learns `typeOf` for normalize can use it in status conditions without
learning a separate API.

The `len` function is analogous: it returns the element count of a map, slice,
or string, and is available in both template expressions and as a comparison
value in conditions.

---

## 6. Production evidence

The CronJob pattern with normalize was tested in the same cluster environment
as the conversion webhook version. The single-version CRD with normalize
produced identical behavior for both input formats:

A CR with `spec.schedule: "0 2 * * 1-5"` (string) reconciled correctly into
a `batch/v1 CronJob` with `spec.schedule: "0 2 * * 1-5"`.

A CR with `spec.schedule: {minute: "0", hour: "2", dayOfMonth: "*", month: "*",
dayOfWeek: "1-5"}` (map) reconciled correctly into the same `batch/v1 CronJob`
with `spec.schedule: "0 2 * * 1-5"`.

The status field `scheduleExpression` showed the normalized form in both cases.
The status field `scheduleFormat` showed `"string"` and `"map"` respectively,
demonstrating that the input format is observable even though the operator
processes only the normalized form.

The test for correctness: if `kubectl get cj` shows the same schedule for both
CRs, and both CRs reconcile without errors on every resync, normalize is working.

---

## 7. Schema evolution without migration

The normalize model has a consequence for API evolution that is worth stating
explicitly: within a single API version, schema evolution does not require
migration.

When a new field is added to the Katalog's `onReconcile` block, CRs that do
not have that field use `notExists` conditions or default values from mutation
rules. When the input format for an existing field is expanded to accept a new
representation, a normalize expression handles the new form. Old CRs continue
to work because the normalize expression handles the old form too.

This is fundamentally different from adding a field to a CRD schema, which is
a structural API change. Adding behavior to the Katalog that responds to a new
optional field is purely additive — existing CRs are unaffected and the Katalog
changes take effect on the next reconcile cycle without any object migration.

For operators built on the ConfigMap data-driven model — where a ConfigMap's
`data` map is the input surface rather than a typed CRD spec — this property
is absolute. ConfigMap data is schema-free by design. Every addition is
additive. No version is ever published. No migration is ever required. The
Katalog evolves; the input surface stays stable.

---

## 8. Conclusion

The normalize block fills a specific gap in the operator authoring toolkit.
It is not an alternative to conversion webhooks and does not replace them for
the problems they solve. It is the mechanism for handling representation
flexibility within a single API version — the case where multiple syntactic
forms express the same semantic intent, and the operator should accept all of
them without scattering format-detection branches throughout its reconcile code.

The normalize block makes the canonical form a property of the Katalog rather
than a property of every template expression that references the field. It
runs before mutation and validation, ensuring that all pipeline stages operate
on the same representation. It is the declarative equivalent of the input
normalization pattern that experienced operators implement manually in Go
hooks — expressed in one place, visible to any reader of the Katalog, and
automatically applied to every reconcile regardless of which code path the
object took to get there.
