# Rollback

When reconcile fails a configurable number of times, Orkestra enters a rollback
phase. It re-applies the previous desired state — captured before the failing
spec change was applied — and blocks normal reconciliation until the spec is
corrected.

Rollback is not transactional undo. It is re-declaration of the last known good
state. The existing Update functions handle idempotent re-application. No new
resource types are involved.

---

## How it works

```
User applies spec N (contains a bug)
  ↓
Reconcile fails 3 times (or N times within configured window)
  ↓
Rollback triggers
  ↓
Previous spec N-1 is re-applied via onRollback templates
  ↓
status.phase = "RolledBack" — normal reconcile blocked
  ↓
User applies spec N+1 (the fix)
  ↓
Generation changes → rollback cleared → normal reconcile resumes
```

The rollback phase exits only when the spec changes. Until then,
`onReconcile` and `onCreate` are blocked. Only `onRollback` runs.

---

## YAML

```yaml
spec:
  crds:
    website:
      operatorBox:
        default: true

        onCreate:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true

        rollback:
          trigger:
            consecutiveFailures: 3       # trigger after 3 failures
            withinDuration: 5m           # optional: only if all 3 within 5 minutes
          onRollback:
            deployments:
              - name: "{{ .metadata.name }}"
                image: "{{ .previous.spec.image }}"
                replicas: "{{ .previous.spec.replicas }}"
                reconcile: true
```

---

## Trigger configuration

```yaml
trigger:
  consecutiveFailures: 3
  withinDuration: 5m
```

**`consecutiveFailures`** — number of consecutive failures before rollback
activates. Default: 3.

**`withinDuration`** — optional. When set, all `consecutiveFailures` failures
must have occurred within this window. If failures are spread across a longer
period, rollback does not trigger.

Examples:
- `consecutiveFailures: 3` — any 3 consecutive failures → rollback
- `consecutiveFailures: 3, withinDuration: 5m` — only if all 3 happened within 5 minutes

Use `withinDuration` to avoid rolling back during transient cluster problems
(API server blip, temporary network partition) where a few failures over a long
period are normal.

---

## The `.previous.*` context

`onRollback` templates have access to `.previous.spec.*` — the spec the CR had
before the failing change was applied:

```yaml
onRollback:
  deployments:
    - name: "{{ .metadata.name }}"
      image: "{{ .previous.spec.image }}"      # image from before the bad change
      replicas: "{{ .previous.spec.replicas }}" # replicas from before the bad change
```

The previous spec is captured automatically before each spec change is applied.
It is stored in a compressed annotation on the CR:

```
orkestra.orkspace.io/previous-spec: <gzip+base64 encoded JSON>
```

You do not manage this annotation. Orkestra writes it on every generation change.

---

## Observability

When rollback is active, the CR status shows:

```yaml
status:
  phase: RolledBack
  conditions:
    - type: Ready
      status: "False"
      reason: RolledBack
      message: "rolled back to previous spec — fix the spec to resume reconciliation"
```

The Control Center shows:

```
website    RolledBack    3 consecutive failures
           Last error: <the error that triggered rollback>
```

The original error is preserved as the last error. The rollback reason is in
the condition message. Both are visible without `kubectl describe`.

---

## Exiting rollback

Apply a corrected spec. The generation changes. On the next reconcile cycle,
Orkestra detects the generation change, clears the rollback annotations, and
resumes normal reconciliation from the corrected spec.

No manual intervention. No annotation deletion. No operator restart.

---

## When rollback is not declared

If no `rollback:` block is declared in `operatorBox:`, behavior is unchanged.
Failed reconciles return an error, increment the failure counter, and requeue
with exponential backoff. The CRD is marked degraded at the configured threshold.
Rollback does not activate.

---

## What rollback does not do

**Rollback does not delete resources.** If the failing spec created a resource
that did not exist before, rollback will not delete it. It re-applies the
previous declaration. Resources that are now orphaned remain until the next
successful reconcile of the corrected spec cleans them up via owner reference
garbage collection.

**Rollback does not restore external state.** If a provider (AWS, MongoDB) made
changes before the failure, rollback does not reverse those changes. It only
re-applies the Kubernetes resources declared in `onRollback`.

**Rollback does not loop.** If `onRollback` itself fails, the error is logged
and the phase stays `RolledBack`. Orkestra does not attempt nested rollback.
Fix the spec to exit.