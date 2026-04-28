# Design: `rollBackOnError: true`

**Status:** Proposed  
**Author:** orkspace  
**Related:** `docs/runtime-manual/concepts/rollback.md`, `pkg/types/rollback.go`, `pkg/reconciler/rollback.go`

---

## Problem

The explicit `rollback:` block requires the author to redeclare every resource they already declared in `onCreate:` or `onReconcile:`, but with `.previous.spec.*` references instead of `.spec.*`:

```yaml
operatorBox:
  onCreate:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        reconcile: true

  rollback:
    trigger:
      consecutiveFailures: 3
    onRollback:
      deployments:
        - name: "{{ .metadata.name }}"
          image: "{{ .previous.spec.image }}"    # ← duplicated, just s/.spec/.previous.spec/
          replicas: "{{ .previous.spec.replicas }}"
          reconcile: true
```

This is the same pattern as `onCreate` + `onReconcile` before `reconcile: true` was introduced. Most rollback declarations are a mechanical translation of the `onCreate` block — identical structure, different dot path. Authors have to maintain both in sync.

For the common case — "if this breaks, put it back the way it was" — this is unnecessary ceremony.

---

## Proposal

Add `rollBackOnError: true` as a field on `operatorBox:`. When set, rollback activates automatically on the default trigger and re-applies the `reconcile: true` declarations from `onCreate:` and `onReconcile:` using the previous spec as the resolver base.

```yaml
operatorBox:
  rollBackOnError: true    # ← that's it

  onCreate:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        reconcile: true
```

No `rollback:` block. No `.previous.spec.*` references. No redeclaration.

---

## How it works

### Trigger

Uses the same defaults as the explicit block:

| Parameter | Default |
|---|---|
| `consecutiveFailures` | 3 |
| `withinDuration` | none (any 3 consecutive failures) |

### Template execution

When rollback activates, the runtime collects all resource declarations from `onCreate:` and `onReconcile:` that have `reconcile: true`. It runs those exact same templates, but builds the resolver with the **previous spec substituted as `.spec.*`**.

The author writes:

```yaml
image: "{{ .spec.image }}"
```

During normal reconcile — this resolves to the current spec.  
During `rollBackOnError` rollback — this resolves to the previous spec (the last known good state).

No template changes. No `.previous.*` syntax. The same declaration drives both normal reconciliation and rollback.

### Why substitute spec rather than inject `.previous.*`

The explicit `onRollback:` block gives full access to both the current spec (`.spec.*`) and the previous spec (`.previous.spec.*`) in the same template. That power is needed when rollback requires _comparing_ old and new state.

`rollBackOnError: true` targets the simpler case: "restore what was there before." The only correct rollback state is the previous spec. Substituting `.spec.*` with the previous spec directly means:

1. The author writes nothing new — the existing declarations are reused verbatim.
2. There is no risk of forgetting to update `.previous.spec.*` references when spec fields change.
3. The intent is unambiguous: rollback = re-apply previous state.

Context other than `spec` (`.metadata.*`, `.external.*`, `.cross.*`) is _not_ substituted — it comes from the current reconcile cycle. This is correct: the name, namespace, and external state are current; only the spec values are rewound.

---

## Interaction with explicit `rollback:` block

`rollBackOnError: true` and an explicit `rollback:` block can coexist. The rules are:

| `rollBackOnError` | `rollback.trigger` | `rollback.onRollback` | Effective behavior |
|---|---|---|---|
| `true` | not set | not set | Default trigger + derived templates |
| `true` | set | not set | Custom trigger + derived templates |
| `true` | not set | set | Default trigger + explicit templates (explicit wins) |
| `true` | set | set | Custom trigger + explicit templates (explicit wins) |
| not set | — | — | Current behavior unchanged |

When `rollback.onRollback` is declared alongside `rollBackOnError: true`, the explicit templates take precedence. This lets authors start with the shorthand and override only the resources that need custom rollback behavior — without having to abandon the shorthand entirely.

---

## YAML reference

**Minimal — zero config:**

```yaml
operatorBox:
  rollBackOnError: true

  onCreate:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        replicas: "{{ .spec.replicas }}"
        reconcile: true
    services:
      - name: "{{ .metadata.name }}-svc"
        port: "80"
        reconcile: true
```

**With custom trigger threshold:**

```yaml
operatorBox:
  rollBackOnError: true
  rollback:
    trigger:
      consecutiveFailures: 5
      withinDuration: 10m

  onCreate:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        reconcile: true
```

**With one resource overriding to explicit rollback templates:**

```yaml
operatorBox:
  rollBackOnError: true

  onCreate:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        reconcile: true
    secrets:
      - name: "{{ .metadata.name }}-creds"
        once: true           # ← not reconcile:true — excluded from derived rollback

  rollback:
    onRollback:
      secrets:
        - name: "{{ .metadata.name }}-creds"
          data:
            password: "{{ .previous.spec.defaultPassword }}"
```

---

## Scope of derived templates

Only resources with **`reconcile: true`** are included in the derived rollback. Resources without it are not eligible — they were created once and are not managed on every reconcile cycle, so re-applying them during rollback is not meaningful.

Resources with `once: true` (like generated secrets) are explicitly excluded — they should never be regenerated by rollback.

---

## Implementation sketch

The change touches three layers:

**1. `pkg/types/types.go` — add the field to `OperatorBoxConfig`:**

```go
type OperatorBoxConfig struct {
    // ... existing fields ...
    RollBackOnError bool          `yaml:"rollBackOnError,omitempty" json:"rollBackOnError,omitempty"`
    Rollback        *RollbackBlock `yaml:"rollback,omitempty" json:"rollback,omitempty"`
}
```

**2. `pkg/types/rollback.go` — add a helper that derives `onRollback` templates:**

```go
// DerivedRollback returns the effective rollback configuration when rollBackOnError:true
// is set. Merges explicit rollback.trigger and rollback.onRollback if declared.
// Returns nil when rollback is not configured at all.
func (c *OperatorBoxConfig) DerivedRollback() *RollbackBlock
```

This helper:
- Returns `nil` when neither `RollBackOnError` nor `Rollback` is set
- Collects all `reconcile: true` declarations from `OnCreate` and `OnReconcile` as the synthetic `OnRollback` when `RollBackOnError` is `true` and no explicit `OnRollback` is declared
- Merges explicit `Trigger` from `Rollback` if present

**3. `pkg/reconciler/rollback.go` — use `DerivedRollback` and run derived templates with previous-spec resolver:**

```go
// buildRollbackResolver returns a resolver where .spec.* resolves to
// the previous spec when rollBackOnError is active.
func buildRollbackResolver(base *orktmpl.Resolver, previousSpec map[string]interface{}) *orktmpl.Resolver
```

The resolver returned by `buildRollbackResolver` replaces the `spec` key in the data map with `previousSpec` before injecting `.previous.*`. This is a one-line change on top of `WithPrevious`.

**4. `pkg/types/methods.go` — update `HasRollbackRules`:**

```go
func (c *CRDEntry) HasRollbackRules() bool {
    return c.OperatorBox.Rollback != nil || c.OperatorBox.RollBackOnError
}
```

---

## What this does not change

- The existing `rollback:` block — fully backward compatible. `rollBackOnError` is opt-in.
- `runRollback` for the explicit path — unchanged.
- The trigger and annotation mechanism — unchanged.
- `.previous.*` context — still available in explicit `onRollback:` templates.
- Status and observability — `phase: RolledBack` and conditions remain identical regardless of which path triggered rollback.

---

## Analogy

| Shorthand | What it eliminates |
|---|---|
| `reconcile: true` on `onCreate` | Separate `onReconcile:` block that duplicates the same declarations |
| `rollBackOnError: true` on `operatorBox` | Separate `onRollback:` block that duplicates the same declarations with `.previous.*` |

Both follow the same principle: if the behavior is derivable from what is already declared, don't make the author declare it twice.
