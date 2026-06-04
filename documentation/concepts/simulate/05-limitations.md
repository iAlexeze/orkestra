# Limitations

The fake cluster has no external connectivity and no real API server. Some operator features require a real cluster or a compiled user binary.

---

## Blocks that are inactive in simulation

| Block | Status | What you see |
|-------|--------|--------------|
| `external:` HTTP calls | Active by default — calls hit the real network; pass `--skip-external` to stub with empty 200 | Without `--skip-external`: real HTTP; with it: `external.*` fields are empty, note printed |
| `cross:` informer reads | Inactive — only the seeded CR is visible to the reconciler | If the cross-dependent CRD has a matching CR in the file it is simulated with a note ("cross: observation not executed") and `cross.*` fields are empty. If it has no matching CR, it is skipped entirely with a note ("no CR found — skipped") |

The output still shows whether the declarative layer (templates, status fields, `once:`, `forEach:`) is correct given absent data.

---

## Constructor: discarding event recorder

When a constructor is called from simulate, `ev event.Recorder` is a silent no-op — `event.Discard()`. The Kubernetes event recorder is not wired to a real API server, so all `Eventf` calls are discarded silently. No guards are needed; call `r.ev.Eventf(...)` normally.

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
