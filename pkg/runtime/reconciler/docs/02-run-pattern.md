# 02 — The Runner Function Contract

> **Note:** Per-resource runner functions have moved to [`pkg/runtime/runners/`](../../runners/README.md). The canonical shape and full reference documentation are in [pkg/runtime/runners/docs/01-runner-contract.md](../../runners/docs/01-runner-contract.md). This page remains as a quick reference for readers navigating the reconciler docs.

Every runner file follows the same structure. This document describes each section in the order it appears.

## Canonical shape

```go
func runWidgets(
    ctx    context.Context,
    kube   kubeclient.KubeClient,
    resolver *orktmpl.Resolver,
    owner  domain.Object,
    srcs   []orktypes.WidgetTemplateSource,
    update bool,
    guard  func(ctx context.Context, obj domain.Object, ns string) bool,
) error {

    // ── Section A: activeNames pre-pass ──────────────────────────────────────
    activeNames := make(map[string]bool, len(srcs))
    for _, s := range srcs {
        if !orktypes.EvaluateConditions(resolver.Data(), s.Conditions, s.Or) {
            continue
        }
        n, _ := resolver.Resolve(s.Name)
        nsp, _ := resolver.Resolve(s.Namespace)
        if nsp == "" {
            nsp = owner.GetNamespace()
        }
        activeNames[nsp+"/"+n] = true
    }

    // ── Section B: main loop ──────────────────────────────────────────────────
    for i, src := range srcs {

        // B1. Evaluate conditions
        conditionPassed := orktypes.EvaluateConditions(resolver.Data(), src.Conditions, src.Or)

        // B2. Early name/namespace resolution
        name, _ := resolver.Resolve(src.Name)
        ns, _   := resolver.Resolve(src.Namespace)
        if ns == "" {
            ns = owner.GetNamespace()
        }

        // B3. Namespace guard
        if guard != nil && !guard(ctx, owner, ns) {
            continue
        }

        // B4. Condition failure path
        if !conditionPassed {
            if update || src.Reconcile {
                if !activeNames[ns+"/"+name] {
                    if err := orkwidget.DeleteIfOwned(ctx, kube, owner, name, ns); err != nil {
                        return fmt.Errorf("widgets[%d]: conditional cleanup: %w", i, err)
                    }
                }
            }
            logger.FromContext(ctx).Debug().
                Str("resource", "Widget").
                Int("index", i).
                Msg("conditions not met — skipping resource")
            continue
        }

        // B5. Resolve template expressions
        resolved, err := resolver.ResolveWidgetTemplate(src)
        if err != nil {
            return fmt.Errorf("widgets[%d]: %w", i, err)
        }

        // B6. Build registry spec and apply
        spec := orkwidget.Resolve(resolved, resolver.OwnerName())

        if update {
            if err := orkwidget.Update(ctx, kube, owner, spec); err != nil {
                return fmt.Errorf("widgets[%d].update: %w", i, err)
            }
        } else {
            if err := orkwidget.Create(ctx, kube, owner, spec); err != nil {
                return fmt.Errorf("widgets[%d].create: %w", i, err)
            }
            if src.Reconcile {
                if err := orkwidget.Update(ctx, kube, owner, spec); err != nil {
                    return fmt.Errorf("widgets[%d].reconcile: %w", i, err)
                }
            }
        }
    }
    return nil
}
```

## Section A — activeNames pre-pass

**Why it exists.** When two declarations target the same resource name but have mutually exclusive conditions (e.g. `typeOf: A` vs `typeOf: B`), the failing path's cleanup (`DeleteIfOwned`) would delete what the passing path just created. The pre-pass builds the set of `(ns/name)` pairs that have at least one passing condition. The main loop only calls `DeleteIfOwned` when the target is NOT in that set.

**What to include in the pre-pass.** Resolve name and namespace only. Do not call any Kubernetes API here. If the resource has a fallback name (like TLS secrets defaulting to `"orkestra-tls"` when no name is declared), apply the same fallback logic here.

**When to omit the pre-pass.** If your resource type never calls `DeleteIfOwned` in the condition-failure path (e.g. `runJobs` — jobs are terminal), you can omit Section A and the guard in B4. See [run_jobs.go](../run_jobs.go).

## Section B1 — Condition evaluation

Always call `orktypes.EvaluateConditions(resolver.Data(), src.Conditions, src.Or)`.

- `resolver.Data()` — the full CR data map including `.children.*`, `.external.*`, `.cross.*`. Do NOT pass the `owner` object directly — it does not have these injected fields.
- `src.Conditions` — the `when:` block (AND semantics).
- `src.Or` — the `or:` block (OR semantics). Both must be present on the type struct.

## Section B2 — Early name/namespace resolution

Resolve before the guard and before the condition-failure cleanup. `Resolve` is cheap and idempotent — the full `ResolveXxxTemplate` call in B5 resolves them again internally.

```go
name, _ := resolver.Resolve(src.Name)
ns, _   := resolver.Resolve(src.Namespace)
if ns == "" {
    ns = owner.GetNamespace()
}
```

The blank identifier on the error is intentional — template resolution errors on name/namespace are surfaced in B5 during full resolution.

## Section B3 — Namespace guard

```go
if guard != nil && !guard(ctx, owner, ns) {
    continue
}
```

Always nil-check. The guard is nil when the CRD has no namespace restrictions. Call it after namespace resolution so the actual target namespace is known.

## Section B4 — Condition failure path

Only attempt cleanup when `update || src.Reconcile` — on first create (update=false, reconcile=false) there is nothing to clean up.

Wrap `DeleteIfOwned` with `if !activeNames[ns+"/"+name]` to prevent the failing path from deleting a resource that the passing path owns.

## Section B5 — Template resolution

Call `resolver.ResolveXxxTemplate(src)`. This evaluates all `{{ }}` expressions in the source struct. Errors here are real — return them.

## Section B6 — Apply

Standard pattern:

```go
if update {
    // onReconcile — drift correction
    orkwidget.Update(...)
} else {
    // onCreate — idempotent create
    orkwidget.Create(...)
    if src.Reconcile {
        // reconcile: true shorthand — sync without a separate onReconcile block
        orkwidget.Update(...)
    }
}
```

Resources that are create-only (ServiceAccount, Job) skip the `update` branch entirely. See `run_serviceaccounts.go` and `run_jobs.go` for reference.

## Signature variations

| Runner | `update bool` | `guard` | Notes |
|--------|--------------|---------|-------|
| `runDeployments` | yes | yes | Full create/update |
| `runServices` | yes | yes | Full create/update |
| `runSecrets` | yes | yes | Extra: `once:`, `rotateAfter:`, `tls:`, `toNamespaces:` |
| `runConfigMaps` | yes | yes | Extra: `toNamespaces:`, `fromConfigMap:` |
| `runServiceAccounts` | yes | yes | Create-only (no update branch) |
| `runCronJobs` | yes | yes | Full create/update |
| `runJobs` | no | yes | Create-only, no activeNames pre-pass |

## Error format convention

```go
return fmt.Errorf("widgets[%d]: %w", i, err)       // resolution errors
return fmt.Errorf("widgets[%d].create: %w", i, err) // apply errors
return fmt.Errorf("widgets[%d].update: %w", i, err)
return fmt.Errorf("widgets[%d]: conditional cleanup: %w", i, err)
```

The `[%d]` index tells operators exactly which declaration in the YAML failed.

---

**Next →** [03 — The Registry Layer](03-registry-layer.md)
