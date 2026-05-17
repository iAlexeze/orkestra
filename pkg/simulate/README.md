# pkg/simulate

`simulate` runs your operator's reconcile loop against an in-memory fake cluster — no Kubernetes required. Give it a Katalog and a CR file and it shows exactly which resources your operator creates, updates, or deletes, and when it converges.

```sh
ork simulate -f katalog.yaml --cr cr.yaml
```

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Read the output and understand what each line means | [docs/01-output.md](docs/01-output.md) |
| Understand steady state and how to tune cycles | [docs/02-steady-state.md](docs/02-steady-state.md) |
| Understand what simulate does not cover | [docs/03-limitations.md](docs/03-limitations.md) |
| Understand the internals (fake cluster, reactors, indexer) | [docs/04-internals.md](docs/04-internals.md) |
