# Kubeclient — Developer Documentation

This package wraps all Kubernetes client concerns: REST config, typed clientset, dynamic client, the `GenericClient` interface that the informer factory uses to build typed ListerWatchers, and the typed CRUD operations (`Get`, `Create`, `Patch`) used by constructor reconcilers.

## Documents

| File | What it covers |
|------|----------------|
| [01-genericclient.md](docs/01-genericclient.md) | The `GenericClient` interface, `Client` implementation, and the `ClientProvider` registration pattern |
| [02-dynamic.md](docs/02-dynamic.md) | `DynamicListerWatcher` — how dynamic (unstructured) CRDs build their ListerWatcher and how namespace scoping works |
| [03-crud.md](docs/03-crud.md) | `Get`, `Create`, `Patch` on `KubeClient` — typed object operations for constructor reconcilers, GVR derivation via scheme+mapper |
| [04-merge.md](docs/04-merge.md) | `MergeFrom` and `StrategicMergeFrom` — building `Patch` values without importing controller-runtime |
| [05-patch-helpers.md](docs/05-patch-helpers.md) | `PatchFinalizers`, `PatchLabels`, `PatchAnnotations`, `PatchStatus`, `PatchSpec` — generic reconciler mutation surface, plus context injection (`WithKubeclient` / `FromContext`) |

Read [01-genericclient.md](docs/01-genericclient.md) when adding a new typed client. Read [02-dynamic.md](docs/02-dynamic.md) when working with dynamic CRDs or namespace-scoped watch streams. Read [03-crud.md](docs/03-crud.md) and [04-merge.md](docs/04-merge.md) when writing or migrating a constructor reconciler. Read [05-patch-helpers.md](docs/05-patch-helpers.md) when working on the generic reconciler or hook functions that need to mutate objects.
