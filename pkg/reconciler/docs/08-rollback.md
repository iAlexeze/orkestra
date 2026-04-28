# 08 — Declarative Rollback

## What rollback does

When a CRD's reconcile fails a configurable number of times, Orkestra enters rollback mode. It re-applies the last known good spec — captured before the failing change was introduced — and blocks normal reconciliation until the operator corrects the spec.

Rollback is not transactional undo. It is idempotent re-application of the previous declaration via the same `Update` path used by `onReconcile`.

## YAML declaration

```yaml
operatorBox:
  rollback:
    trigger:
      consecutiveFailures: 3     # trigger after 3 consecutive failures
      withinDuration: 5m         # optional — only if all 3 happened within this window
    onRollback:
      deployments:
        - name: "{{ .previous.metadata.name }}"
          image: "{{ .previous.spec.image }}"
          replicas: "{{ .previous.spec.replicas }}"
          reconcile: true
      configMaps:
        - name: "{{ .previous.metadata.name }}-config"
          reconcile: true
```

`onRollback` uses the same resource grammar as `onReconcile`. The `.previous.*` context is hydrated from the spec snapshot annotation.

## The six-phase reconcile flow

Rollback adds two phases to `reconcileImpl`:

```
Phase 1 — Rollback gate
    isRollbackActive() → true:
        runRollback()     ← applies onRollback templates with .previous.*
        return            ← blocks normal reconcile until spec changes

Phase 2 — Mutation
Phase 3 — Validation

Phase 4 — Dispatch (hook / declarative templates / no-op)

Phase 5 — Rollback trigger check (on dispatch error)
    history.record()
    shouldRollback() → true:
        markRollbackActive()  ← writes RollbackGenerationAnnotation

Phase 6 — Spec snapshot (on dispatch success)
    snapshotSpec()         ← writes PreviousSpecAnnotation (gzip + base64)
    clearFailureHistory()
```

## Spec snapshotting (`snapshotSpec`)

Before each spec change is applied, `snapshotSpec` gzip-compresses and base64-encodes the current spec and writes it to the CR annotation `orkestra.orkspace.io/previous-spec`. This annotation is the source of truth for rollback templates.

Snapshotting only runs on the unstructured path (`*unstructured.Unstructured`). Typed reconcilers (Go hook path) are not snapshotted.

## Trigger evaluation (`shouldRollback`)

`RollbackTrigger.ShouldTrigger(failureTimes []time.Time)` checks whether the in-process failure history meets the declared threshold:

- **`consecutiveFailures` only** — triggers when the history has at least N entries, regardless of timing.
- **`consecutiveFailures` + `withinDuration`** — triggers only when all N failures occurred within the window. Failures older than the window are excluded.

The failure history (`rollbackHistory` map keyed by `"namespace/name"`) is in-process and resets on restart. The annotation survives restart and prevents a rollback-cleared state from being incorrectly re-entered.

## Rollback state (`isRollbackActive`)

Rollback state lives in the CR annotation `orkestra.orkspace.io/rollback-at-generation`, set to the generation at which rollback triggered. `isRollbackActive` returns true when this annotation equals the CR's current generation.

Exit condition: the user submits a spec change. The new generation differs from the annotation value → `isRollbackActive` returns false → normal reconciliation runs. On the first successful reconcile after rollback, Phase 6 detects the stale `RollbackGenerationAnnotation` and calls `clearRollback`, which removes both annotations and fires `rollbackClearFn` to update `CRDHealth`. `snapshotSpec` then writes a fresh `PreviousSpecAnnotation`.

## Running rollback templates (`runRollback`)

`runRollback` calls `resolver.WithPrevious(previousSpec)` to inject the decoded spec under `.previous.*`, then passes the resulting resolver to `runResourceGroup` with `update=true`. This is identical to `onReconcile` processing — the same runners, same idempotency guarantees.

When no `onRollback` block is declared, rollback still activates (blocks normal reconcile) but applies no resources.

## CRDHealth integration

`startCRDWorkers` injects two callbacks via `SetRollbackNotifiers`:

| Callback | Fired by | Effect on CRDHealth |
|----------|----------|---------------------|
| `onTrigger` | `markRollbackActive` | Increments `rollbackTotal`, sets `rollbackActive = true`, stores `rollbackLastAt` |
| `onClear` | `clearRollback` | Sets `rollbackActive = false` |

`CRDHealth.RollbackStats()` returns these values for the `/katalog/{crd}` response.

## Annotations

| Annotation | Value | Meaning |
|-----------|-------|---------|
| `orkestra.orkspace.io/previous-spec` | gzip+base64 JSON | Last successfully reconciled spec |
| `orkestra.orkspace.io/rollback-at-generation` | generation number | Generation at which rollback was triggered |

Both annotations are cleared by `clearRollback` when the spec is corrected.
