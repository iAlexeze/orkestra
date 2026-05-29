# Typed Operators

Orkestra operators are declarative by default — no Go required. Typed operators are the escape hatch for the cases that genuinely need code.

A typed operator references compiled Go types. Everything else — informers, workqueues, drift correction, status, metrics — works exactly as it does for a pure YAML operator.

---

## When to use typed operators

Use a typed operator when your reconcile logic needs:

- **External calls the `external:` block cannot express** — the `external:` block handles HTTP calls declaratively (GET/POST, bearer tokens, response gating). Use hooks when you need non-HTTP protocols, SDK calls, database connections, or multi-step interactions that can't be expressed as "call a URL and gate on the response". If you need to provision a user inside PostgreSQL, call an AWS SDK, or run a gRPC call, that needs a hook.
- **Complex computed fields** — deriving values from multiple sources, running logic that template expressions cannot express
- **A fully custom reconciler** — integrating an existing operator's logic without rewriting it

If your operator only creates Kubernetes resources and applies rules, stay declarative.

!!! note "Templates already see the full spec"
    Template expressions have access to the complete CR — `{{ .spec.* }}`, `{{ .status.* }}`, `{{ .metadata.* }}` — regardless of whether `apiTypes.location` is set. The template resolver converts any object to `map[string]interface{}` before executing expressions. Setting `location` is for Go code only.

---

## The two patterns

→ [Hooks](./01-hooks.md) — add Go logic alongside declarative templates. The hook runs, then Orkestra applies `onCreate`/`onReconcile` templates as normal.

→ [Constructor](./02-constructor.md) — replace the reconciler entirely. Your Go code owns the full reconcile loop; declarative templates are not applied.

→ [Mixing all three](./03-mixed.md) — a declarative operator, a hooks operator, and a constructor operator composed into one runtime from a single Komposer.
