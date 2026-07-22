# Hooks

Hooks run Go code during reconciliation alongside declarative templates.

There are two ways to use hooks. The hybrid pattern is recommended.

---

## Hybrid (recommended) {#hybrid}

Declare everything Orkestra handles well in the Katalog. Write Go only for what templates cannot express. Orkestra runs declared templates first, then the hook adds what's missing.

A common example: declare the `ServiceAccount` in the Katalog so Orkestra creates and drift-corrects it. The hook references it by the same naming convention — no coordination needed.

```yaml
operatorBox:
  reconciler:
    hooks:
      location: github.com/myorg/database-operator/hooks
      function: DatabaseHooks
      resources:
        - kind: StatefulSet
        - kind: Service

  onCreate:
    serviceAccounts:
      - name: "{{ .metadata.name }}-sa"   # Orkestra owns this
```

```go
func onReconcile(ctx context.Context, obj *apiv1.Database) error {
    // SA was declared in the Katalog — reference it by the same convention
    spec := orkstatefulset.Resolve(orktypes.StatefulSetTemplateSource{
        Name:               obj.Name,
        ServiceAccountName: obj.Name + "-sa",
        // ... rest of spec
    }, obj.Name)
    return orkstatefulset.Update(ctx, kube, obj, spec)
}
```

To run the hook before declared templates instead of after, set `runHooksFirst: true` in the hooks block. The default is `false` — declared templates run first.

---

## Hooks only {#hooks-only}

The hook manages all child resources in Go. No declared templates alongside it. Use when type-safe control over every resource is more important than keeping declarations in YAML, or when all resources depend on computed values that templates cannot express.

```yaml
operatorBox:
  reconciler:
    hooks:
      location: github.com/myorg/database-operator/hooks
      function: DatabaseHooks
      resources:
        - kind: StatefulSet
        - kind: Service
        - kind: CronJob
```

The hook creates, updates, and deletes all child resources directly via the `pkg/resources` library.

---

## Katalog reference

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

      operatorBox:
        reconciler:
          workers: 3
          resync: 30s
          hooks:
            location: github.com/myorg/database-operator/hooks
            version: v1.3.0
            fetch: true
            function: DatabaseHooks
            resources:
              - kind: StatefulSet
              - kind: Service
            args:
              readReplicaCount: 2
              backupEnabled: true
              replicationMode: async

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

`args` passes configuration from the Katalog into the hook at reconcile time. **String values support Go template expressions** — the GenericReconciler evaluates them against the current CR before the hook runs, so the hook sees fully-resolved values:

```yaml
args:
  readReplicaCount: 2                                       # static — passed through as-is
  backupEnabled: true                                       # static
  region: "{{ default \"us-east-1\" .spec.region }}"       # dynamic — evaluated per-CR
  database:
    engine: "{{ default \"postgres\" .spec.engine }}"       # dynamic — nested maps work too
```

```go
func onReconcile(ctx context.Context, obj *apiv1.Database) error {
    kube, _ := kubeclient.FromContext(ctx)
    replicas  := kube.Args().Int("readReplicaCount")   // 2
    region    := kube.Args().String("region")          // "eu-west-1" or default "us-east-1"
    db        := kube.Args().Sub("database")
    engine    := db.String("engine")                   // from spec or default "postgres"
    _ = replicas; _ = region; _ = engine
    return nil
}
```

The full note FuncMap is available in template args — `default`, `upper`, `lower`, and all other note functions work exactly as they do in `onCreate`/`onReconcile` templates.

For structured access, bind the whole map to a typed struct with `kube.Args().BindArgs(&cfg)`. See the [schema reference](../../reference/schema/02-katalog/04-operatorbox.md) for the full `args` API.

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