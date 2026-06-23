# Resilience Pack

Operators that stay running — through panics, cascading failures, bad CRs, and degraded dependencies.

```bash
ork init --pack resilience
```

---

## Examples

| Example | What it teaches |
|---------|-----------------|
| [Safe Reconcile](safe-reconcile/README.md) | Panic isolation in the worker pool. A nil pointer in a typed hook is caught, logged, and recovered — the operator stays running and other CRDs are unaffected. |

---

## Running an example

```bash
ork init --pack resilience
cd resilience/safe-reconcile
ork run --dev
```

---

## E2E

Every example ships with a runnable `e2e.yaml`:

```bash
cd safe-reconcile && ork e2e
```
