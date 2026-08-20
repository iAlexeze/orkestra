# Output — the rewritten file

## toclient mode (default)

`ork migrate` (without `--mode`) produces the minimum change needed to run your reconciler inside Orkestra. Only two things change:

### SetupWithManager is removed

Replaced with a comment:

```go
// SetupWithManager removed — Orkestra provides the informer, workqueue,
// worker pool, leader election, panic recovery, and metrics.
// Delete this file's main.go and scheme registration too.
```

### A constructor is injected

```go
func NewWebAppReconciler(kube kubeclient.Interface) domain.Reconciler {
    return domain.ReconcilerFrom(&WebAppReconciler{
        client: kubeclient.ToClient(kube),
    })
}
```

`kubeclient.ToClient` returns a `client.Client` — the same type your struct field already holds. `domain.ReconcilerFrom` adapts the `ctrl.Request` signature to Orkestra's interface. Your `Reconcile` method body is completely untouched.

---

## native mode (`--mode native`)

Full mechanical rewrite to idiomatic Orkestra style.

### Signature change

```go
// Before
func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)

// After
func (r *WebAppReconciler) Reconcile(ctx context.Context, key string) error
```

`key` is `namespace/name` — the same as `req.String()`. Orkestra calls this from its worker pool, which already manages concurrency, retries, and leader election.

### Return statements

Every `ctrl.Result` is collapsed:

```go
// Before
return ctrl.Result{}, err
return ctrl.Result{}, nil

// After
return err
return nil
```

`ctrl.Result{RequeueAfter: X}` cannot be collapsed mechanically — it is flagged:

```go
// TODO(ork migrate): RequeueAfter removed — Orkestra retries on non-nil error
return nil
```

Return an error to trigger a retry. Orkestra's backoff policy applies automatically.

### req.NamespacedName

When the body uses `req.NamespacedName`, the tool injects a key split at the top of `Reconcile` and replaces usages:

```go
// Injected at top of Reconcile body
parts := strings.SplitN(key, "/", 2)
namespace, name := parts[0], parts[1]

// Usage replaced
r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, webapp)
```

### r.Status().Update()

Flagged inline — Orkestra uses a different status API:

```go
nil /* TODO(ork migrate): replace with r.kube.PatchStatus(ctx, obj, GroupVersionResource, map[string]interface{}{...}) */
```

### TODO markers

All items that need human review are marked `// TODO(ork migrate):`. After migration:

```bash
grep -rn "TODO(ork migrate)" ./my-operator/
```

---

Next: [Generated files](02-generated-files.md)
