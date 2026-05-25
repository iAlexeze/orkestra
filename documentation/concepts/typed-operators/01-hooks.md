# Hooks

Hooks run Go code during reconciliation alongside declarative templates. The hook runs first, then Orkestra applies any `onCreate`/`onReconcile` templates.

---

## Katalog

```yaml
spec:
  crds:
    database:
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Database
        plural: databases
        object: Database
        objectList: DatabaseList
        location: github.com/myorg/database-operator/api/v1alpha1

      workers: 3
      resync: 30s

      operatorBox:
        default: true   # keep the GenericReconciler; hooks are additive

        hooks:
          location: github.com/myorg/database-operator/hooks
          version: v1.3.0
          fetch: true
          function: DatabaseHooks
          resources:
            - kind: StatefulSet
            - kind: Service

        # declarative templates still apply after the hook
        status:
          fields:
            - path: phase
              value: "Running"
```

`apiTypes.location` tells Orkestra to deliver `*apiv1.Database` to your hook function instead of `domain.Object`. Set `object` and `objectList` to the Go type names at that path.

`version` pins the module version. `fetch` controls whether `ork generate registry` adds it to your project:

| `fetch` | Behaviour |
|---|---|
| `false` (default) | Module must already be in `go.mod`. `ork generate registry` wires it without modifying dependencies. |
| `true` | `ork generate registry` runs `go get <location>@<version>` automatically, adding or updating the module in `go.mod` and `go.sum`. |

For private modules with `fetch: true`, set `GOPRIVATE` and ensure credentials are available before running `ork generate registry`.

`resources` declares what Kubernetes resources the hook manages — required for RBAC generation.

---

## Hook function

The `function` value must be an exported function that returns `domain.AnyReconcileHooks`:

```go
package hooks

import (
    "context"
    "github.com/orkspace/orkestra/domain"
    apiv1 "github.com/myorg/database-operator/api/v1alpha1"
)

func DatabaseHooks() domain.AnyReconcileHooks {
    return domain.ReconcileHooks[*apiv1.Database]{
        OnReconcile: onReconcile,
        OnDelete:    onDelete,
    }
}

func onReconcile(ctx context.Context, obj *apiv1.Database) error {
    // obj.Spec.Engine, obj.Spec.Storage — all fields accessible
    // call external APIs, compute status, manage resources
    return nil
}

func onDelete(ctx context.Context, obj *apiv1.Database) error {
    // clean up external resources before the finalizer is removed
    return nil
}
```

`domain.ReconcileHooks[T]` has three optional fields: `OnReconcile`, `OnDelete`, `OnNotFound`. Implement only what you need. `T` must be a pointer type — `*apiv1.Database`, not `apiv1.Database`.

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
cd 09-hooks
```

Follow the steps in the README