# Kubeclient — Developer Documentation

This package wraps all Kubernetes client concerns: REST config, typed clientset, dynamic client, and the `GenericClient` interface that the informer factory uses to build typed ListerWatchers.

## Documents

| File | What it covers |
|------|----------------|
| [01-genericclient.md](docs/01-genericclient.md) | The `GenericClient` interface, `Client` implementation, and the `ClientProvider` registration pattern |
| [02-dynamic.md](docs/02-dynamic.md) | `DynamicListerWatcher` — how dynamic (unstructured) CRDs build their ListerWatcher and how namespace scoping works |

Read [01-genericclient.md](docs/01-genericclient.md) when adding a new typed client. Read [02-dynamic.md](docs/02-dynamic.md) when working with dynamic CRDs or namespace-scoped watch streams.
