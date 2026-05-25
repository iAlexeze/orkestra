# Status Conditions

`status.fields` entries accept an optional `when:` block. The field is written only when all conditions pass. Multiple entries for the same path are evaluated in order — the last one whose conditions pass writes the final value.

This is how Orkestra implements declarative state machines.

---

## Basic state machine

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
          equals: "Pending"

    - path: phase
      value: "Succeeded"
      when:
        - field: status.phase
          equals: "Running"
        - field: children.job.status.succeeded
          operator: gt
          value: "0"

    - path: phase
      value: "Failed"
      when:
        - field: children.job.status.failed
          operator: gt
          value: "0"
```

Transitions:
- First reconcile → `Pending` (status.phase not yet written)
- While Job runs → stays `Running`
- Job exits 0 → `Succeeded`
- Job exits non-zero → `Failed` (overrides any earlier write in the same cycle)

---

## First-reconcile detection with `notExists`

`notExists` passes when a field is absent or empty — which is true on the first reconcile before any status has been set.

```yaml
- path: phase
  value: "Pending"
  when:
    - field: status.phase
      operator: notExists
```

On every reconcile after the first, `status.phase` has a value and this entry is skipped.

---

## Multi-value matching with `in`

```yaml
- path: phase
  value: "Running/build"
  when:
    - field: status.phase
      operator: in
      value: "Pending,"     # "Pending" OR "" (first reconcile)
```

The trailing comma in the value list matches the empty string — covering both the `Pending` and `notExists` cases in one entry.

---

## Override semantics

Multiple entries for the same `path` are evaluated in declaration order. The **last** one whose conditions pass writes the final value.

Terminal states (`Failed`, `Succeeded`) should be declared last so they override in-progress states when conditions are met in the same reconcile cycle.

---

## Complete example output

```yaml
# kubectl get pipeline build-and-test -o yaml (status excerpt)
status:
  phase: Succeeded
  step: notify
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
```

The progression — `Pending` → `Running/build` → `Running/test` → `Running/notify` → `Succeeded` — is driven entirely by job completion events and the `when:` conditions above. No Go code. No phase annotation management.
