---
title: "Conditional Status Fields"
weight: 50
description: "**How declarative state machines work in Orkestra.**"
---

**How declarative state machines work in Orkestra.**

This document is a maintainer reference. It explains the implementation of
`when:` conditions on `status.fields` entries — the primitive that makes
declarative state machines possible.

---

## The problem this solves

Before this feature, `status.fields` wrote unconditionally after every
successful reconcile. A field declared as:

```yaml
status:
  fields:
    - path: phase
      value: "Running"
```

always wrote `"Running"` to `status.phase` regardless of what the CR's
current state was. This made status fields useful for reporting current
observed state, but not for driving state transitions.

A state machine requires writing different values to the same field
depending on current state. The only way to do this before was a Go
constructor that owned the full reconcile loop.

---

## The feature

`status.fields` entries now accept an optional `when:` block. The field is
written only when all conditions in the block pass. Multiple entries for the
same path are evaluated in order — the last one whose conditions pass wins.

```yaml
status:
  fields:
    - path: phase
      value: "Pending"
      when:
        - field: status.phase
          operator: notExists    # only on first reconcile

    - path: phase
      value: "Running"
      when:
        - field: status.phase
          equals: "Pending"      # only when currently Pending
```

---

## Implementation

The implementation is in `pkg/reconciler/run_status.go`.

### `resolveStatusFields`

The entry point. Called with the ordered list of `StatusField` entries and
the resolver (which includes the full CR map plus `.children.*`).

```go
func resolveStatusFields(
    ctx context.Context,
    obj domain.Object,
    resolver *orktmpl.Resolver,
    fields []orktypes.StatusField,
) (map[string]interface{}, []string)
```

For each field:
1. If `field.When` is non-empty, call `evaluateStatusConditions`
2. If conditions fail, `continue` — skip this field entirely
3. If conditions pass (or absent), resolve `field.Value` via the resolver
4. Write to the result map via `setNestedStatus`

The result map is later merged into the status patch.

### `evaluateStatusConditions`

Evaluates all conditions in a `when:` block with AND semantics. Calls
`evaluateOneStatusCondition` for each condition. Returns `false` as soon
as any condition fails.

### `evaluateOneStatusCondition`

Evaluates a single `Condition` against the object map. The object map is
obtained from `resolver.Data()` — it includes:

- `.spec.*` — all spec fields from the CR
- `.status.*` — current status fields (from the last observed state in the informer cache)
- `.metadata.*` — name, namespace, labels, annotations, generation
- `.children.*` — child resources read after the reconcile (keyed by lowercase kind)

The operators supported:

| Operator | Meaning | Use case |
|---|---|---|
| `exists` | field is non-empty | optional fields that drive conditionals |
| `notExists` | field is absent or empty | first-reconcile detection |
| `equals` | exact string match | phase transition gating |
| `notEquals` | string mismatch | |
| `contains` | substring match | |
| `hasPrefix` | starts with | phase prefix matching |
| `hasSuffix` | ends with | |
| `gt` | numeric greater-than | child succeeded/failed count |
| `lt` | numeric less-than | |
| `in` | membership in comma-separated list | multi-phase matching |

### `resolveNestedField`

Navigates a dot-notation path through the object map. `"status.phase"` returns
`objMap["status"]["phase"]`. Returns `""` when any segment is missing — this
is the `notExists` case.

```go
func resolveNestedField(m map[string]interface{}, path string) string
```

This function is separate from the template resolver's field access because
it operates on the raw map without template expression evaluation. It must
handle nil at every level without panicking.

### `setNestedStatus`

Writes a value at a dot-notation path in the result map. Creates intermediate
maps as needed. The standard `setNestedPatch` pattern.

---

## The override semantics

Multiple entries for the same `path` are evaluated in declaration order.
The last one whose conditions pass writes the final value.

This means terminal states should be declared **last**:

```yaml
status:
  fields:
    - path: phase
      value: "Running/build"     # written early — conditions easy to pass
      when:
        - field: status.phase
          operator: in
          value: "Pending,"

    - path: phase
      value: "Succeeded"         # written later — overrides Running/build
      when:
        - field: status.phase
          equals: "Running/notify"
        - field: children.job.status.succeeded
          operator: gt
          value: "0"

    - path: phase
      value: "Failed"            # written last — overrides everything
      when:
        - field: children.job.status.failed
          operator: gt
          value: "0"
```

In a single reconcile cycle, if the `Failed` conditions pass, `"Failed"` is
written — even if an earlier entry wrote `"Running/build"`. The map is built
entry-by-entry and the last write wins. This is correct: a Job failure should
always drive to `Failed` regardless of what the current phase says.

---

## The children dependency

The `when:` conditions on status fields frequently reference `.children.job.status.succeeded`.
This works because `ReadChildren` is called before `resolveStatusFields`:

```
reconcile cycle:
  1. runTemplateReconcile (create/update resources)
  2. ReadChildren         (read child resource state into children map)
  3. resolver.SetChildren (inject children into resolver context)
  4. resolveStatusFields  (evaluate conditions, resolve values, build patch)
  5. PatchStatus          (write to Kubernetes)
```

The children map is keyed by lowercase Kubernetes Kind:
- `children["deployment"]` ← first Deployment declared
- `children["deployments"]` ← map of all Deployments by name
- `children["cronjob"]` ← first CronJob declared
- `children["job"]` ← first Job declared
- etc.

The singular key is a shorthand for the single-child common case. The plural
key is the full map for multi-child cases.

---

## The `notExists` operator

`notExists` is the most important operator for state machines. It detects
that a field has not yet been written — specifically, the first reconcile
before any status has been set.

```go
case orktypes.ConditionNotExists:
    return fieldVal == "" || fieldVal == "<no value>"
```

`resolveNestedField` returns `""` when a path does not exist in the map.
Go's `text/template` with `missingkey=zero` returns `"<no value>"` for
absent keys (though the resolver strips this to `""`). Both cases are
treated as `notExists`.

This is why the first-reconcile detection pattern works:

```yaml
- path: phase
  value: "Pending"
  when:
    - field: status.phase
      operator: notExists
```

On the first reconcile, `status.phase` is absent from the informer's cached
object. `resolveNestedField` returns `""`. `notExists` passes. `"Pending"` is
written.

On every subsequent reconcile, `status.phase` has a value. `notExists` fails.
This entry is skipped. The next entry whose conditions pass writes the phase.

---

## The `in` operator with empty string

The `in` operator with a comma-separated value allows matching multiple states
including the empty case:

```yaml
- field: status.phase
  operator: in
  value: "Pending,"   # "Pending" OR "" (first reconcile)
```

This is a shorthand for "Pending or notExists". The trailing comma produces
an empty string in the split list which matches `fieldVal == ""`.

Implementation:

```go
case orktypes.ConditionIn:
    values := strings.Split(expected, ",")
    for _, v := range values {
        if strings.TrimSpace(v) == fieldVal {
            return true
        }
    }
    if fieldVal == "" {
        for _, v := range values {
            if strings.TrimSpace(v) == "" {
                return true
            }
        }
    }
    return false
```

---

## Adding a new operator

1. Add the constant to `pkg/types/condition_operators.go`
2. Add the case to `evaluateOneStatusCondition` in `run_status.go`
3. Add the case to `evaluateValidationRule` in `run_validation.go` (same operators, same semantics)
4. Add the shorthand to the `Condition` struct if appropriate
5. Document in `docs/concepts/notes.md` and this file

The operators are shared between validation rules and status condition
evaluation. When you add an operator, add it in both places.

---

## Testing

The declarative state machine can be tested without a running cluster:

```bash
# Install the CRD
kubectl apply -f examples/phases/crd.yaml

# Start Orkestra locally
ork run --katalog examples/phases/katalog.yaml

# Apply both CRs (success and failure paths)
kubectl apply -f examples/phases/cr.yaml

# Watch phase progression in real time
kubectl get pipelines -w
```

Expected output:
```
NAME               PHASE            STEP     AGE
build-and-test     Pending                   0s
build-and-test     Running/build    build    2s
build-and-test     Running/test     test     8s
build-and-test     Running/notify   notify   14s
build-and-test     Succeeded                 18s
failing-pipeline   Pending                   0s
failing-pipeline   Running/build    build    2s
failing-pipeline   Failed                    8s
```

The `failing-pipeline` drives to `Failed` when its build Job exits non-zero.
The `build-and-test` pipeline completes all three steps and drives to `Succeeded`.
Both are governed by the same Katalog — no code written.
