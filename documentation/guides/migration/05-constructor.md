# Constructor — Lift and Change

You have a working controller-runtime operator. The reconcile logic is solid. What you want to remove is the machinery around it — the manager, the workqueue, the scheme registration, the leader election setup, the metrics. Not rewrite the operator from scratch.

Option 04 is the lift-and-change path. The reconcile logic moves across unchanged. Only the signature changes.

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/04-constructor-migration
```

---

## What you will learn

- The exact signature change from controller-runtime to Orkestra constructor
- What `default: false` means in the Katalog — the GenericReconciler is disabled, the constructor owns the loop
- What the runtime provides that `ctrl.NewManager` previously handled
- Why `ctrl.Result{RequeueAfter: X}` becomes a plain error return

---

## The signature change

Before:
```go
Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
```

After:
```go
Reconcile(ctx context.Context, key string) error`
```

`key` is `namespace/name` — the same as `req.String()`. Everything inside the method body stays unchanged.

The struct changes from an embedded `client.Client` to Orkestra's `informer`, `kube`, and `ev` fields. The constructor function (`NewWebAppReconciler`) is what the Katalog references. `SetupWithManager` is removed.

---

## What you removed

`ctrl.NewControllerManagedBy`, `SetupWithManager`, scheme registration, `main.go`, leader election setup. Orkestra provides all of it — informer, workqueue, worker pool, panic recovery, Prometheus metrics, health endpoints, leader election.

---

## `ork migrate` automates this

```bash
ork migrate ./controller/webapp_controller.go -o ./output
```

See [06 — ork migrate](./07-ork-migrate.md) or run option 06 in the pack.

---

## Try it

```bash
ork init --pack from-controller-runtime
cd from-controller-runtime/04-constructor-migration
# Follow steps in README
```

→ [05 — Constructor: Orkestra resources](./06-constructor-resources.md)
