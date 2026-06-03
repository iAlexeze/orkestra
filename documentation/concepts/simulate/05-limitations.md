# Limitations

The fake cluster has no external connectivity and no real API server. Some operator features require a real cluster or a compiled user binary.

---

## Blocks that are inactive in simulation

These blocks do not cause a skip — simulate runs everything else and emits a note.

| Block | Status | What you see |
|-------|--------|--------------|
| `external:` HTTP calls | Inactive — no outbound network | `external.*` fields are empty; `when:` conditions on them evaluate as unmet |
| `cross:` informer reads | Inactive — only one CR seeded | `cross.*` fields are empty; dependent resources with `when:` on cross fields are not created |

The output for these files still shows whether the declarative layer (templates, status fields, `once:`, `forEach:`) is correct given absent data.

---

## Constructor: nil event recorder

When a constructor is called from simulate, `*event.Event` is `nil`. The Kubernetes event recorder is not wired — there is no real API server to record to.

If your constructor calls `r.ev.Record(...)` or similar without guarding, it will panic. Guard all event recorder calls:

```go
if r.ev != nil {
    r.ev.Record(ctx, obj, corev1.EventTypeNormal, "Reconciled", "pipeline ready")
}
```

---

## Standard binary fallback

When running from the standard `ork` binary (not a custom operator binary), `HookRegistry` and `ReconcilerRegistry` are empty for your custom types. Hook bodies and constructor reconcile loops do not execute. Simulate falls back to `GenericReconciler` and runs the status layer only.

Build and use your own `ork` binary to get full hook and constructor simulation:

```bash
make registry && make build
ork simulate --cr cr.yaml
```

---

## What simulate cannot cover

| Feature | Use instead |
|---------|-------------|
| Admission webhook validation/mutation | `ork e2e` |
| Real pod scheduling and readiness | `ork e2e` |
| Watch events triggering reconciles | `ork e2e` |
| Actual merge-patch / SSA semantics | `ork e2e` |
| Cross-namespace secret reads | `ork e2e` |
| Provider blocks (AWS, MongoDB) | `ork e2e` |

For anything in the right column, provision a kind cluster and run `ork e2e`.

---

→ Back to: [ork simulate](index.md)
