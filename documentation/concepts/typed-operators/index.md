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

## The patterns

→ [Hooks — hybrid](./01-hooks.md#hybrid) **(recommended)** — declare everything Orkestra handles well in the Katalog; write Go only for what templates cannot express. Orkestra runs declared templates first, then the hook.

→ [Hooks — hooks only](./01-hooks.md#hooks-only) — the hook manages all child resources in Go. Use when type-safe control over every resource matters more than keeping declarations in YAML.

→ [Constructor](./02-constructor.md) — replace the reconciler entirely. Your Go code owns the full reconcile loop; declared templates are not applied. Use when migrating an existing controller-runtime operator or running a custom state machine.

→ [Mixing all three](./03-mixed.md) — a declarative operator, a hooks operator, and a constructor operator composed into one runtime from a single Komposer.

→ [Migrating from controller-runtime](./05-migration.md) — have a working controller-runtime operator? The `from-controller-runtime` pack shows the same operator expressed five ways, and `ork migrate` automates the constructor path.
