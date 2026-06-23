# 09 — Pointer Type Parameter and the ObjectHooks Adapter

## The problem in one sentence

Users write `domain.ReconcileHooks[*Database]` (pointer type, matching Kubernetes
conventions), but the runtime registry passes those hooks through a type-erased
path that resolves the reconciler's type parameter to `domain.Object` (an
interface). A direct type assertion fails at startup with a panic:

```text
hooks type mismatch — got domain.ReconcileHooks[*Database]
```

---

## Why pointer types are correct

Kubernetes informers store and return pointer values. Every item retrieved from
an informer cache is a `*Database`, not a `Database`. The whole ecosystem —
`controller-runtime`, `kubebuilder`, typed client-go — uses pointer receivers and
pointer type arguments for CR types. Users writing:

```go
domain.ReconcileHooks[*Database]{OnReconcile: myFunc}
```

are following the correct convention.

---

## Why Go generics make this hard

Go generics are **invariant**. `ReconcileHooks[*Database]` and
`ReconcileHooks[domain.Object]` are unrelated types even though `*Database`
implements `domain.Object`. A direct type assertion:

```go
anyHooks.(domain.ReconcileHooks[PTR])  // PTR = domain.Object
```

fails at runtime when the value is `ReconcileHooks[*Database]`.

---

## Why not two type parameters?

The constraint:

```go
type GenericReconciler[S any, PTR interface{ *S; domain.Object }] struct { ... }
```

enforces at compile time that `PTR` is a pointer to a concrete struct. It cannot
be satisfied by `domain.Object` (an interface — not `*S` for any `S`).

The runtime registry path in `runtime_konstructor.go` infers `PTR = domain.Object`
because its `newObj` factory is typed `func() domain.Object`. Making it supply
concrete types would require a typed factory per CRD in the registry — a much
larger API change with no benefit for the dynamic template path where hooks are
always nil anyway.

---

## The ObjectHooks adapter

`GenericReconciler[PTR]` stores `domain.ObjectHooks` rather than
`domain.ReconcileHooks[PTR]`:

```go
type ObjectHooks struct {
    OnReconcile func(ctx context.Context, obj Object) error
    OnDelete    func(ctx context.Context, obj Object) error
    OnNotFound  func(ctx context.Context, key string) error
}
```

The adapter is built **once** at construction time through the `domain.HookBinder`
interface:

```go
binder, ok := anyHooks.(domain.HookBinder)
hooks = binder.BindToObjectHooks()
```

`BindToObjectHooks()` wraps each hook in a closure that performs `obj.(T)` before
calling the typed function:

```go
oh.OnReconcile = func(ctx context.Context, obj domain.Object) error {
    return fn(ctx, obj.(T))   // T = *Database
}
```

Every `domain.ReconcileHooks[T]` value satisfies `HookBinder` automatically via
the generic method — no user action required.

---

## Why the assertion is safe

- The informer is constructed for a single concrete type.
- Every object in its cache IS that type, stored as `interface{}`.
- `reconcileCore` already asserts `raw.(PTR)` before passing `obj` downstream,
  so by the time the hook closure runs, the object is guaranteed to be the right
  type.

If the wrong type somehow ends up in the store the assertion panics with a clear
`interface conversion` message, which is the correct fail-fast behavior.

---

## End-to-end call path

```text
1. User writes DatabaseHooks() returning ReconcileHooks[*Database]
       │
2. Generated registry puts it in HookRegistry[GVK]
       │
3. runtime_konstructor.go: anyHooks = crd.OperatorBox.HookFactory()
                            = ReconcileHooks[*Database]
       │
4. NewGenericReconciler[domain.Object](..., anyHooks, func() domain.Object{...})
       │
5. anyHooks.(domain.HookBinder)  ← succeeds: ReconcileHooks[T] has BindToObjectHooks()
       │
6. BindToObjectHooks() → ObjectHooks with obj.(*Database) closures
       │
7. reconcileCore: raw.(domain.Object) → obj (underlying type *Database)
       │
8. r.hooks.OnReconcile(ctx, obj)
       │
9. closure: obj.(*Database) ← succeeds, user receives *Database ✓
```

---

## Implementing a custom hook wrapper

If you need middleware around hooks (logging, metrics, error translation),
implement `domain.HookBinder`:

```go
type LoggingHooks[T domain.Object] struct {
    Inner domain.ReconcileHooks[T]
}

// Satisfy domain.AnyReconcileHooks
func (h LoggingHooks[T]) isHooks() {}   // unexported, must be in domain package

// Satisfy domain.HookBinder
func (h LoggingHooks[T]) BindToObjectHooks() domain.ObjectHooks {
    oh := h.Inner.BindToObjectHooks()

    if innerReconcile := oh.OnReconcile; innerReconcile != nil {
        oh.OnReconcile = func(ctx context.Context, obj domain.Object) error {
            log.Info("reconciling", "name", obj.GetName())
            return innerReconcile(ctx, obj)
        }
    }
    return oh
}
```

> **Note:** `isHooks()` is an unexported method on `domain.AnyReconcileHooks`.
> Third-party packages cannot implement it. Embed `domain.ReconcileHooks[T]`
> or keep your wrapper in the same module as the hooks declaration.

---

## Summary

| Decision | Reason |
|---|---|
| Type param named `PTR` not `T` | Signals pointer expectation at every call site |
| `ObjectHooks` stored, not `ReconcileHooks[PTR]` | Bridges typed user hooks and the type-erased runtime registry |
| `HookBinder` interface, not direct assertion | Works for any `ReconcileHooks[T]` regardless of what `T` is |
| Single type parameter kept | Keeps `runtime_konstructor.go` compatible; two-param form would break the dynamic registry path |
