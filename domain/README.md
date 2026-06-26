# domain

`domain` defines the core interfaces that everything in Orkestra implements against. No business logic lives here — only the contracts that let the runtime, reconciler, gateway, and CLI coordinate without importing each other.

## What is here

| File | What it defines |
|------|----------------|
| [object.go](object.go) | `Object` and `ObjectList` — the minimal Kubernetes object interface used everywhere a resource must be both a `metav1.Object` and a `runtime.Object` |
| [domain.go](domain.go) | `Komponent` — the lifecycle interface for anything the manager starts and stops; `Reconciler` — the single-method contract every CRD reconciler satisfies |
| [health.go](health.go) | `Health` — the liveness and readiness contract, following Kubernetes probe semantics |
| [generic.go](generic.go) | `ReconcileHooks[T]`, `AnyReconcileHooks`, `ObjectHooks`, `HookBinder` — the typed hook system that lets users attach Go functions to a CRD's reconcile and delete events without giving up the declarative layer |

## Why a separate package

Every Orkestra package that participates in the runtime cycle needs to share these types. Putting them here keeps the import graph acyclic — `pkg/reconciler`, `pkg/webhook`, `pkg/kordinator`, and `cmd` can all import `domain` without pulling in each other.

## The hook type system

`ReconcileHooks[T]` is the user-facing API. Users write:

```go
domain.ReconcileHooks[*Database]{
    OnReconcile: func(ctx context.Context, obj *Database) error { ... },
    OnDelete:    func(ctx context.Context, obj *Database) error { ... },
}
```

Internally, `ReconcileHooks[T]` is adapted to `ObjectHooks` via `BindToObjectHooks()` — a type-erased form that the generic reconciler stores so it can serve both typed (`*Database`) and dynamic (`domain.Object`) paths from a single implementation.

See [generic.go](generic.go) for the full design rationale in the file header.
