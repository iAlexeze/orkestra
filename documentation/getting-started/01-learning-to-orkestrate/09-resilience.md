# Resilience Pack

Operators that stay running through panics, bad input, and degraded state.

```bash
ork init --pack resilience
```

---

## The examples

| Directory | What it demonstrates |
|-----------|----------------------|
| `safe-reconcile` | Panic isolation — a nil pointer in a typed hook is caught by `safeReconcile`, logged with a stack trace, and re-queued with backoff. Other CRDs keep reconciling. The process never crashes. |
| `01-admission-protection` | Runtime validation as a resilience layer — a bad CR degrades the operator after `failureThreshold` is exceeded. Patch the CR and the operator recovers automatically. No restart needed. |

---

## What every example shows

- Orkestra stays **Operational** at the runtime level even when individual operators are **Degraded**
- The Control Center shows the exact failure — consecutive fail count, last error, stack trace (for panics)
- Recovery is automatic — when the root cause is fixed, the operator moves from Degraded → Pending → Healthy without intervention

---

## Run the full suite

```bash
ork e2e -f resilience/e2e.yaml
```

Or simulate without a cluster:

```bash
ork simulate -f resilience/simulate.yaml
```

→ [Resilience examples](../../../examples/resilience/)
