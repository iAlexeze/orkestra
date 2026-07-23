# pkg/runtime/informer

This directory implements Orkestra's shared informer factory: the layer between the Kubernetes API server watch stream and the per-CRD work queues.

## Documents

| File | What it covers |
|------|----------------|
| [01-architecture.md](docs/01-architecture.md) | Factory lifecycle, event routing, how one informer per CRD is created and started |
| [02-namespace-filter.md](docs/02-namespace-filter.md) | Three-tier namespace filtering — from cluster-scoped ListerWatcher down to reconciler safety net |
| [03-listerwatch.md](docs/03-listerwatch.md) | How ListerWatchers are built for typed and dynamic CRDs, Tier 1 namespace scoping |

Read them in order the first time. When debugging event routing or namespace restriction, jump directly to [02-namespace-filter.md](docs/02-namespace-filter.md).
