# Apply Time Controls

## `serve.apply.overrides`

Controls apply-time behaviour for server-side apply and routing surface changes. These settings act as a **second line of defense** — they determine whether request-level overrides (`?overwrite=true` and `?override=true`) are honored.

Both settings default to `true` (allow overrides). Can be set at the CRD level (fallback for all targets) and per target.

```yaml
serve:
  enabled: true

  # CRD-level fallback for all targets
  apply:
    overrides:
      resourceConflict: true   # allow ?overwrite=true
      targetConflict: true     # allow ?override=true

  targets:
    staging:
      primary: true
      apply:
        overrides:
          targetConflict: false   # staging does NOT allow routing surface changes, even with ?override=true

    production:
      primary: false
      apply:
        overrides:
          resourceConflict: false # production does NOT allow force field changes, even with ?overwrite=true
```

**`resourceConflict`** — when `true` (default), callers can use `?overwrite=true` to force field ownership on server-side apply, equivalent to `--force-conflict`. When `false`, `?overwrite=true` is rejected and field conflicts are always surfaced as errors. Default: `true`.

**`targetConflict`** — when `true` (default), callers can use `?override=true` to change the routing surface (target/alias) of an existing CR. When `false`, `?override=true` is rejected and routing surface changes are always rejected with a conflict error. Default: `true`.

Resolution order per target:
1. Target-level (`serve.targets[<name>].apply.overrides`)
2. CRD-level (`serve.apply.overrides`)
3. Default (`true` — allow overrides)


## `serve.targets[<name>].fieldSelector`

Links full CRs to a target based on field values. When a CR matches ALL key-value pairs, it is automatically routed to that target — enabling per-target response config, tokens, permissions, and mode enforcement for full CR mode.

```yaml
serve:
  enabled: true
  targets:
    internal:
      fieldSelector:
        spec.workloadType: app
      modes:
        cr: false          # internal disallows full CRs
      apply:
        targetOverride: false
```

**`fieldSelector`** — a map of dot-notation field paths to values (max 3 per target). This is a true selector — like Service → Pod selection. Each target must have a unique selector. `ork validate` enforces uniqueness.

**Validation rules:**

| Rule | Description |
|------|-------------|
| **Maximum 3 selectors** | Each target can have at most 3 field selectors — keep it simple, avoid overlapping. |
| **Unique across targets** | No two targets can share the same `path:value` pair — routing must be deterministic. |
| **Dot-notation format** | Paths must be valid dot-notation paths (e.g., `spec.mealPlan`, not `.mealPlan`). |
| **Valid Kubernetes names** | Each path segment must be a valid Kubernetes qualified name. |
| **Non-empty values** | Values cannot be empty strings. |

**Warnings:**

| Condition | Warning |
|-----------|---------|
| CR mode disabled (`modes.cr: false`) and no `fieldSelector` | Target is unreachable via full CR mode — add field selectors or enable CR mode. |
| CR mode disabled globally (`serve.modes.cr: false`) and `fieldSelector` set | field selectors will have no effect — CR mode is disabled globally. |

```yaml
targets:
  internal:
    modes:
      cr: false          # full CR mode disabled
    # no fieldSelector → warning: target unreachable via full CR mode
```

```yaml
targets:
  internal:
    fieldSelector:
      spec.workloadType: app
    modes:
      cr: false          # full CR mode disabled, but fieldSelector provides a way in
    # ✅ no warning — fieldSelector routes CRs to this target
```

**The warnings are advisory.** `ork validate` does not block — it informs the platform team of potential misconfigurations.

**Examples**

Route CRs with `spec.workloadType: app` to `internal`, and enforce `cr: false`:

```yaml
serve:
  enabled: true
  targets:
    internal:
      modes:
        cr: false
      fieldSelector:
        spec.workloadType: app
      apply:
        targetOverride: false
```

Full CRs with `spec.workloadType: app` are routed to `internal`, and `cr: false` is enforced. The mode check uses the effective target, not the primary target.

**Validation errors:**

```bash
ork validate
✗ CRD "platRsc": target "internal" has 4 field selectors — maximum is 3
✗ CRD "platRsc": field selector "spec.workloadType=app" is used by both targets "internal" and "kitchen" — field selectors must be unique across targets
✗ CRD "platRsc": target "internal" has invalid field selector path ".workloadType": field selector path cannot start or end with a dot. Usage example: 'spec.mealPlan'
```

**Warnings:**

```bash
⚠ CRD "platRsc": target "internal" has fieldSelector but CR mode is disabled — fieldSelector will have no effect
```

## Quick Scan

### The Two-Layer Defense System

**Layer 1: Request-level (`?overwrite=true` or `?override=true`)**
- Callers can request to bypass conflicts per-request
- This is the **first line of defense**

**Layer 2: Configuration (`apply.overrides.resourceConflict` / `apply.overrides.targetConflict`)**
- CRD owners can **opt-out** of allowing these bypasses
- When set to `false`, even if the caller passes `?overwrite=true` or `?override=true`, the request is rejected
- This is the **second line of defense**

### Default Behavior

**`resourceConflict`**: 
- Default: `true` — allows `?overwrite=true` to force field conflicts
- When `false`: even `?overwrite=true` is rejected

**`targetConflict`**: 
- Default: `true` — allows `?override=true` to change routing surface
- When `false`: even `?override=true` is rejected
