# Constructor

A constructor replaces the GenericReconciler entirely. Your Go code owns the full reconcile loop — declarative templates are not applied when `default: false`.

Use a constructor when you are integrating an existing operator: pass Orkestra's `Kubeclient` and informer to the existing controller and it works without `controller-runtime`.

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

!!! note
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
    kube *kubeclient.Kubeclient,
    informer cache.SharedIndexInformer,
    ev *event.Event,
) domain.Reconciler {
    return &PipelineReconciler{kube: kube, informer: informer, event: ev}
}
```

Orkestra calls this function once at startup and uses the returned `domain.Reconciler` for all reconcile events on this CRD.

---

## Generate and build

After writing the Katalog, one command generates both `pkg/typeregistry/zz_generated_typeregistry.go` and `cmd/orkestra/main.go`:

```bash
ork generate registry --file katalog.yaml
go build ./cmd/orkestra
```

You write neither generated file. Re-run `ork generate registry` whenever you change `apiTypes`, `hooks`, or `constructor` declarations in the Katalog.

---

→ See the full working example: [`examples/advanced/10-constructor`](../../../examples/advanced/10-constructor/katalog.yaml)
