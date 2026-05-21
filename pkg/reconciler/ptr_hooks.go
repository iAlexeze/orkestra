// pkg/reconciler/ptr_hooks.go
//
// ── Design note: pointer type parameter and the ObjectHooks adapter ───────────
//
// WHY PTR, NOT T
// ──────────────
// GenericReconciler uses the type parameter name PTR rather than the
// conventional T to make the pointer expectation visible at every call site.
// Kubernetes informers store and return pointer values: when you create an
// informer for Database objects for example, every item retrieved from its cache is a
// *Database. The reconciler type-asserts the raw interface{} value from the
// cache with raw.(PTR), which succeeds only when PTR is a pointer type.
//
// Users therefore write:
//
//	domain.ReconcileHooks[*Database]{OnReconcile: myFunc}
//
// not:
//
//	domain.ReconcileHooks[Database]{OnReconcile: myFunc}  // wrong
//
// This matches the pattern used throughout the Kubernetes ecosystem:
// controller-runtime, kubebuilder, and all typed client-go code work with
// pointer receivers and pointer type arguments for the same reason.
//
// THE MISMATCH PROBLEM
// ────────────────────
// Go generics are invariant: ReconcileHooks[*Database] and
// ReconcileHooks[domain.Object] are unrelated types even though *Database
// implements domain.Object. A direct type assertion:
//
//	anyHooks.(domain.ReconcileHooks[PTR])
//
// fails at runtime whenever the caller provides ReconcileHooks[*Database]
// but PTR is inferred as domain.Object — which is exactly what happens in
// the runtime registry path inside runtime_konstructor.go, where the concrete type
// is not known at compile time.
//
// WHY NOT TWO TYPE PARAMETERS
// ───────────────────────────
// The constraint PTR interface{ *S; domain.Object } (with a second parameter
// S any) enforces at compile time that PTR is a pointer to a concrete struct.
// However, it cannot be satisfied by domain.Object (an interface), so the
// runtime registry path in runtime_konstructor.go — which infers PTR = domain.Object —
// would fail to compile. Updating runtime_konstructor.go to supply concrete types would
// require the registry to carry a separate typed factory per CRD, a much larger
// change with no benefit for the dynamic template path where no hooks exist.
//
// THE OBJECTHOOKS ADAPTER
// ───────────────────────
// Instead of storing domain.ReconcileHooks[PTR] on the reconciler, we store
// domain.ObjectHooks — a plain struct whose function fields are parameterised
// on domain.Object. The adapter is built once in NewGenericReconciler via the
// domain.HookBinder interface:
//
//	binder, ok := anyHooks.(domain.HookBinder)
//	hooks = binder.BindToObjectHooks()
//
// Every domain.ReconcileHooks[T] value satisfies HookBinder automatically
// through its BindToObjectHooks() method. That method wraps each hook in a
// closure that performs obj.(T) before delegating to the typed function:
//
//	oh.OnReconcile = func(ctx context.Context, obj domain.Object) error {
//	    return fn(ctx, obj.(T))   // T = *Database at runtime
//	}
//
// The assertion is safe because:
//   - The informer is always constructed for a single concrete type.
//   - Every item returned from its cache IS that type, even when retrieved as
//     domain.Object (or interface{}).
//   - GenericReconciler already asserts raw.(PTR) earlier in reconcileCore,
//     so obj arriving at the hook is guaranteed to be of the right type.
//
// END-TO-END CALL PATH FOR TYPED HOOKS
// ──────────────────────────────────────
//
//  1. User writes:
//     func DatabaseHooks() domain.AnyReconcileHooks {
//     return domain.ReconcileHooks[*Database]{OnReconcile: onReconcile}
//     }
//
//  2. Generated registry registers DatabaseHooks in HookRegistry.
//
//  3. runtime_konstructor.go calls HookFactory() → gets ReconcileHooks[*Database].
//
//  4. NewGenericReconciler receives it as domain.AnyReconcileHooks.
//     PTR is inferred as domain.Object (from func() domain.Object newObj).
//
//  5. anyHooks.(domain.HookBinder) succeeds because ReconcileHooks[*Database]
//     has BindToObjectHooks() via the generic method.
//
//  6. BindToObjectHooks() builds ObjectHooks with closures: obj.(*Database).
//
//  7. At reconcile time, reconcileCore does raw.(*Database or domain.Object)
//     — whichever PTR resolves to — producing a domain.Object value whose
//     underlying type is *Database.
//
//  8. r.hooks.OnReconcile(ctx, obj) invokes the closure from step 6.
//     obj.(*Database) succeeds. The user's typed function receives *Database. ✓
//
// CUSTOM HOOK WRAPPERS
// ────────────────────
// If you wrap ReconcileHooks[T] in your own struct (e.g. to add middleware),
// implement domain.HookBinder by forwarding to the inner hooks:
//
//	type LoggingHooks[T domain.Object] struct {
//	    Inner domain.ReconcileHooks[T]
//	}
//	func (h LoggingHooks[T]) isHooks() {}
//	func (h LoggingHooks[T]) BindToObjectHooks() domain.ObjectHooks {
//	    oh := h.Inner.BindToObjectHooks()
//	    inner := oh.OnReconcile
//	    oh.OnReconcile = func(ctx context.Context, obj domain.Object) error {
//	        log.Info("reconciling", "name", obj.GetName())
//	        return inner(ctx, obj)
//	    }
//	    return oh
//	}
package reconciler
