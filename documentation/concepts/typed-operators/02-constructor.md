# Constructor

A constructor replaces the GenericReconciler entirely. Your Go code owns the full reconcile loop — declarative templates are not applied when `default: false`.

Use a constructor when:

- **Migrating an existing controller-runtime operator** — change the `Reconcile` signature from `(ctx, req) (Result, error)` to `(ctx, key string) error`, remove the manager setup, and register the constructor in the Katalog. The informer, workqueue, worker pool, leader election, metrics, and panic recovery are all provided by Orkestra. Your reconcile logic is unchanged.
- **Running a custom state machine** — when the reconcile loop itself is stateful and not easily expressed as declarative templates with `when:` conditions.

For new operators, prefer [hooks in hybrid mode](./01-hooks.md#hybrid). Only reach for a constructor when you need to own the full loop.

---

## Katalog

```yaml
spec:
  crds:
    pipeline:
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Pipeline
        plural: pipelines
        object: Pipeline
        objectList: PipelineList
        location: github.com/myorg/pipeline-operator/api/v1alpha1

      workers: 5
      resync: 10s

      operatorBox:
        default: false   # disable GenericReconciler; constructor owns everything

        constructor:
          location: github.com/myorg/pipeline-operator/reconciler@v2.0.0
          function: NewPipelineReconciler
          resources:
            - kind: Job
              group: batch
              version: v1
              plural: jobs
```

`default: false` tells Orkestra not to use the GenericReconciler. The constructor at `location` provides the complete reconcile implementation.

The `@version` suffix in `location` is shorthand for the `version:` field — `location: github.com/myorg/reconciler@v2.0.0` is equivalent to declaring `version: v2.0.0` separately. Both forms are accepted; the `@` shorthand keeps the declaration compact.

`fetch` controls whether `ork generate registry` adds the module to your project:

| `fetch` | Behaviour |
|---|---|
| `false` (default) | Module must already be in `go.mod`. `ork generate registry` wires it without modifying dependencies. |
| `true` | `ork generate registry` runs `go get <location>@<version>` automatically, adding or updating the module in `go.mod` and `go.sum`. |

Use `fetch: true` when pulling the constructor from a remote module you have not yet added to the project. Use `fetch: false` (or omit it) when the module is already a local dependency.

`resources` declares what Kubernetes resources the constructor manages — required for RBAC generation.

!!! note Constructor Ownership
    `onCreate`, `onReconcile`, `onDelete`, `hooks`, and `status.fields` are all ignored when `default: false`. The constructor owns status management directly.

---

## Constructor function

The `function` value must be an exported function with this signature:

```go
package reconciler

import (
    "github.com/orkspace/orkestra/pkg/kubeclient"
    "github.com/orkspace/orkestra/pkg/event"
    "github.com/orkspace/orkestra/domain"
    "k8s.io/client-go/tools/cache"
)

func NewPipelineReconciler(
    kube kubeclient.Kubeclient,
    informer cache.SharedIndexInformer,
    ev *event.Event,
) domain.Reconciler {
    return &PipelineReconciler{kube: kube, informer: informer, event: ev}
}
```

Orkestra calls this function once at startup and uses the returned `domain.Reconciler` for all reconcile events on this CRD.

---

## Two styles of reconcile implementation

### Lift and change signature

The minimal migration from controller-runtime. Change the `Reconcile` signature and remove the manager setup. Your resource management logic stays exactly as it was.

**Before (controller-runtime)**:
```go
func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    existing := &appsv1.Deployment{}
    err := r.Get(ctx, req.NamespacedName, existing)
    if errors.IsNotFound(err) {
        return ctrl.Result{}, r.Create(ctx, desired)
    }
    patch := client.MergeFrom(existing.DeepCopy())
    existing.Spec = desired.Spec
    return ctrl.Result{}, r.Patch(ctx, existing, patch)
}
```

**After (Orkestra constructor)**:
```go
func (r *WebAppReconciler) Reconcile(ctx context.Context, key string) error {
    // key == req.String() — same content, no other change to this method
    existing := &appsv1.Deployment{}
    err := r.kube.Get(ctx, namespace, name, existing)
    if errors.IsNotFound(err) {
        return r.kube.Create(ctx, desired)
    }
    patch := client.MergeFrom(existing.DeepCopy())
    existing.Spec = desired.Spec
    return r.kube.Patch(ctx, existing, patch)
}
```

What you removed: `ctrl.NewManager`, `SetupWithManager`, scheme registration in `main.go`. Orkestra provides the informer, workqueue, worker pool, leader election, metrics, and panic recovery. You kept all the business logic.

---

### With Orkestra resources

Replace the manual Get / IsNotFound / Create / Patch pattern with the `pkg/resources` library. Each resource type exports four functions: `Create`, `Update`, `Delete`, `DeleteIfOwned`.

**Before (manual)**:
```go
existing := &appsv1.Deployment{}
err := r.kube.Get(ctx, namespace, name, existing)
if errors.IsNotFound(err) {
    return r.kube.Create(ctx, desired)
}
patch := client.MergeFrom(existing.DeepCopy())
existing.Spec = desired.Spec
return r.kube.Patch(ctx, existing, patch)
```

**After (Orkestra resources)**:
```go
import orkdeploy "github.com/orkspace/orkestra/pkg/resources/deployments"

spec := orkdeploy.Resolve(orktypes.DeploymentTemplateSource{
    Name:      obj.GetName(),
    Namespace: obj.GetNamespace(),
    Image:     typedObj.Spec.Image,
    Replicas:  fmt.Sprintf("%d", typedObj.Spec.Replicas),
}, obj.GetName())

return orkdeploy.Update(ctx, kube, obj, spec)
```

`Update` handles: create if absent, patch if drifted, owner references, system labels (`managed-by: orkestra`, `orkestra-owner: <cr-name>`). No conditional logic in your reconciler.

To remove a resource only if this CR owns it:

```go
orkdeploy.DeleteIfOwned(ctx, kube, obj, name, namespace)
```

`DeleteIfOwned` is a no-op if the resource does not exist or is owned by a different CR.

---

## Generate and build

After writing the Katalog, one command generates both `pkg/typeregistry/zz_generated_typeregistry.go` and `cmd/orkestra/main.go`:

```bash
ork generate registry --file katalog.yaml
go build ./cmd/orkestra
```

You write neither generated file. Re-run `ork generate registry` whenever you change `apiTypes`, `hooks`, or `constructor` declarations in the Katalog.

---

To try the full working example:

```bash
ork init --pack advanced
cd 10-constructor
```

Follow the steps in the README