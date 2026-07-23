# Output — the rewritten file

`ork migrate` performs a mechanical rewrite of the `Reconcile` method. The output is structured so you can review and complete it — not run it as-is.

## Signature change

```go
// Before
func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

// After
func (r *WebAppReconciler) Reconcile(ctx context.Context, key string) error {
```

`key` is `namespace/name` — the same as `req.String()`. Orkestra calls this from its worker pool, which already manages concurrency, retries, and leader election.

## Return statements

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

## req.NamespacedName

When the body uses `req.NamespacedName`, the tool injects a key split at the top of `Reconcile` and replaces usages:

```go
// Injected at top of Reconcile body
parts := strings.SplitN(key, "/", 2)
namespace, name := parts[0], parts[1]

// Usage replaced
r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, webapp)
```

## SetupWithManager

The method is removed and replaced with a comment:

```go
// SetupWithManager removed — Orkestra provides the informer, workqueue,
// worker pool, leader election, panic recovery, and metrics.
// Delete this file's main.go and scheme registration too.
```

## r.Status().Update()

Flagged inline — Orkestra uses a different status API:

```go
nil /* TODO(ork migrate): replace with r.kube.PatchStatus(ctx, obj, GroupVersionResource, map[string]interface{}{...}) */
```

## TODO markers

All items that need human review are marked `// TODO(ork migrate):`. After migration:

```bash
grep -rn "TODO(ork migrate)" ./my-operator/
```
