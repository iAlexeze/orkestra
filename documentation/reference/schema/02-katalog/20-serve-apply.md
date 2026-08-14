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

**Examples**

Disable all overrides globally:
```yaml
serve:
  enabled: true
  apply:
    overrides:
      resourceConflict: false
      targetConflict: false
```

Per-target override (only staging allows routing surface changes):
```yaml
serve:
  enabled: true
  apply:
    overrides:
      resourceConflict: false
      targetConflict: false

  targets:
    staging:
      primary: true
      apply:
        overrides:
          targetConflict: true    # only staging allows ?override=true

    production:
      primary: false
      # inherits CRD-level: resourceConflict: false, targetConflict: false
```

Disable force field conflicts globally, but allow routing surface changes:
```yaml
serve:
  enabled: true
  apply:
    overrides:
      resourceConflict: false   # nobody can force field conflicts
      targetConflict: true      # anyone can change routing surfaces
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

## Default Behavior

**`resourceConflict`**: 
- Default: `true` — allows `?overwrite=true` to force field conflicts
- When `false`: even `?overwrite=true` is rejected

**`targetConflict`**: 
- Default: `true` — allows `?override=true` to change routing surface
- When `false`: even `?override=true` is rejected
