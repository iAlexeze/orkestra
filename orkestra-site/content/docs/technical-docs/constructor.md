---
title: "Constructor"
weight: 153
---

# Constructor

A constructor provides a fully custom reconciler for a CRD — replacing the `GenericReconciler` entirely. This is the escape hatch for use cases where the `GenericReconciler`'s lifecycle model does not fit.

{{< callout type="warning" title="Use hooks first" >}}
In the vast majority of cases, [hooks](./hooks.md) are the right choice.
Hooks give you full Go control over reconcile logic while Orkestra still
handles finalizers, events, metrics, health tracking, and leader election.
{{< /callout >}}

    Use a constructor only when you need to control the entire reconcile
    lifecycle — including finalizer management, event emission, and health
    reporting — yourself.

---

## When constructors are appropriate

- You have an existing operator reconciler from Kubebuilder or Operator SDK that you want to run inside Orkestra
- You need reconcile semantics that the `GenericReconciler` does not support (e.g. a state machine with explicit phase transitions stored in Status)
- You are migrating a complex operator to Orkestra incrementally and want the declarative features for new CRDs while keeping the existing reconciler for the complex one

---

## The NewReconcilerFunc signature

```go
// pkg/types/types.go
type NewReconcilerFunc func(
    ctx context.Context,
    entry CRDEntry,
    kube *kubeclient.Kubeclient,
) (domain.Reconciler, error)
```

The constructor is called once at CRD startup. It receives:
- `ctx` — the runtime context (cancelled on shutdown)
- `entry` — the full CRDEntry from the Katalog (workers, resync, APITypes, etc.)
- `kube` — the Kubernetes client wrapper

It must return a `domain.Reconciler`:

```go
// domain/reconciler.go
type Reconciler interface {
    // Reconcile is called for every work item from the CRD's workqueue.
    // key is "namespace/name" for namespaced resources, "name" for cluster-scoped.
    // Return nil on success. Return an error to requeue with backoff.
    Reconcile(ctx context.Context, key string) error
}
```

---

## Writing a constructor

```go
// pkg/reconcilers/database_reconciler.go
package reconcilers

import (
    "context"
    "fmt"

    "github.com/orkspace/orkestra/domain"
    orktypes "github.com/orkspace/orkestra/pkg/types"
    "github.com/orkspace/orkestra/pkg/kubeclient"
    apiv1 "github.com/myorg/api/database/v1alpha1"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

// NewDatabaseReconciler is the constructor function.
// Its signature must match orktypes.NewReconcilerFunc.
func NewDatabaseReconciler(
    ctx context.Context,
    entry orktypes.CRDEntry,
    kube *kubeclient.Kubeclient,
) (domain.Reconciler, error) {
    // Build any clients or state the reconciler needs
    dbClient, err := newDatabaseClient(ctx)
    if err != nil {
        return nil, fmt.Errorf("building database client: %w", err)
    }

    gvr := entry.GroupVersionResource  // already populated by Orkestra

    return &databaseReconciler{
        kube:     kube,
        dbClient: dbClient,
        gvr:      gvr,
    }, nil
}

// databaseReconciler implements domain.Reconciler.
type databaseReconciler struct {
    kube     *kubeclient.Kubeclient
    dbClient *DatabaseClient
    gvr      schema.GroupVersionResource
}

// Reconcile handles one work item from the queue.
// You are responsible for the full lifecycle here — finalizers, events, metrics.
func (r *databaseReconciler) Reconcile(ctx context.Context, key string) error {
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        return fmt.Errorf("invalid key %q: %w", key, err)
    }

    // Read from the informer cache — zero API calls
    obj, err := r.kube.DynamicClient().
        Resource(r.gvr).
        Namespace(namespace).
        Get(ctx, name, metav1.GetOptions{})
    if errors.IsNotFound(err) {
        // Object is gone — nothing to reconcile
        return nil
    }
    if err != nil {
        return fmt.Errorf("getting %s/%s: %w", namespace, name, err)
    }

    // Type assert to your typed struct if needed
    typed := &apiv1.Database{}
    if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, typed); err != nil {
        return fmt.Errorf("converting to typed: %w", err)
    }

    // Handle deletion
    if !typed.DeletionTimestamp.IsZero() {
        return r.handleDelete(ctx, typed)
    }

    // Handle creation/reconciliation
    return r.handleReconcile(ctx, typed)
}

func (r *databaseReconciler) handleDelete(ctx context.Context, obj *apiv1.Database) error {
    // External cleanup
    if err := r.dbClient.DropDatabase(ctx, obj.Spec.DatabaseName); err != nil {
        return fmt.Errorf("dropping database: %w", err)
    }

    // Remove finalizers to unblock deletion
    // You must do this yourself — GenericReconciler is not running
    patch := client.MergeFrom(obj.DeepCopy())
    obj.Finalizers = removeString(obj.Finalizers, "finalizer.database.io/cleanup")
    return r.kube.Client().Patch(ctx, obj, patch)
}

func (r *databaseReconciler) handleReconcile(ctx context.Context, obj *apiv1.Database) error {
    // Your reconcile logic here
    return nil
}
```

### Declaring in Katalog

```yaml
- name: database
  apiTypes:
    group: platform.myorg.io
    version: v1alpha1
    kind: Database
    plural: databases
    location: github.com/myorg/api/database/v1alpha1

  operatorBox:
    default: false       # ← must be false when using constructor
    constructor:
      location: github.com/myorg/reconcilers
      function: NewDatabaseReconciler
```

### Generate the registry

```bash
ork generate registry --file katalog.yaml
```

Generates the `ReconcilerRegistry` entry for `NewDatabaseReconciler`.

---

## What you own with a constructor

When `reconciler.default: false`, the `GenericReconciler` is not used. You own:

| Responsibility | You must implement |
|---|---|
| Finalizer add on create | Yes — or CRs will be deleted without cleanup |
| Finalizer removal on delete | Yes — or CRs will be stuck in terminating |
| Kubernetes events | Yes — `ork events` will show nothing |
| Prometheus metrics | Yes — `ork status` error rate will be blank |
| Health state updates | Yes — `/katalog/{crd}/health` will never degrade |
| `safeReconcile` panic recovery | Provided by Orkestra — wraps your `Reconcile` call |

{{< callout type="warning" title="safeReconcile is still active" >}}
Even with a custom constructor, Orkestra wraps your `Reconcile` method
in `safeReconcile`. A panic in your reconciler is caught, logged, and
returned as an error to the workqueue. This is non-negotiable — it protects
the other CRDs in the runtime.
{{< /callout >}}

---

## Incremental migration pattern

The constructor enables a clean migration path from an existing operator:

```go
// 1. Your existing operator reconciler — unchanged
func (r *ExistingReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
    // ... existing code ...
}

// 2. Adapter that maps domain.Reconciler → controller-runtime semantics
type reconcileAdapter struct {
    existing *ExistingReconciler
}

func (a *reconcileAdapter) Reconcile(ctx context.Context, key string) error {
    ns, name, _ := cache.SplitMetaNamespaceKey(key)
    result, err := a.existing.Reconcile(ctx, reconcile.Request{
        NamespacedName: types.NamespacedName{Namespace: ns, Name: name},
    })
    if err != nil {
        return err
    }
    if result.Requeue || result.RequeueAfter > 0 {
        // Map back to Orkestra's queue model
        return &requeueError{after: result.RequeueAfter}
    }
    return nil
}

// 3. Constructor wraps the adapter
func NewExistingReconcilerAdapter(ctx context.Context, entry orktypes.CRDEntry, kube *kubeclient.Kubeclient) (domain.Reconciler, error) {
    existing := &ExistingReconciler{/* your existing setup */}
    return &reconcileAdapter{existing: existing}, nil
}
```

This runs your existing reconciler inside Orkestra with no changes to its logic. Orkestra provides the informer, queue, worker pool, and the process lifecycle. Your code provides the reconcile function.

Over time, replace the existing reconciler with hooks or templates as you understand each piece.
