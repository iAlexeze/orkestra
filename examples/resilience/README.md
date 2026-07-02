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
| [Admission Protection](01-admission-protection/README.md) | Runtime validation as a resilience layer. A bad CR degrades the operator after `failureThreshold` is exceeded. Patch it — the operator recovers automatically, no restart needed. |

---

## Running an example

```bash
ork init --pack resilience
cd resilience/safe-reconcile
ork run --dev
```

---

## E2E

Every example ships with a runnable `e2e.yaml`. Run the full pack in one command:

```bash
ork e2e -f resilience/e2e.yaml
```

Or run a single example:

```bash
cd safe-reconcile && ork e2e
cd 01-admission-protection && ork e2e
```

## Simulate

```bash
ork simulate -f resilience/simulate.yaml
```
