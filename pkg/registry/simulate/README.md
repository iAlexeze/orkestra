# pkg/registry/simulate

`simulate` runs your operator's reconcile loop against an in-memory fake cluster — no Kubernetes required. Give it a Katalog and a CR file and it shows exactly which resources your operator creates, updates, or deletes, and when it converges.

The recommended entry point is `simulate.yaml` — it records what your operator should produce so the run is repeatable and verifiable:

```sh
ork simulate                                   # auto-detects simulate.yaml, then katalog.yaml
ork simulate -f simulate.yaml                  # explicit — assert mode when expect: is set
ork simulate -f my-katalog.yaml --cr my-cr.yaml
ork simulate ./...                             # discovers all simulate.yaml files recursively
ork simulate ./... --skip vendor               # skip patterns during discovery
ork simulate --skip-external                   # stub external: HTTP calls
ork simulate --debug-ops                       # print all recorded ops with cycle numbers
ork simulate --dev-server                      # start mock dev server for external: examples
ork simulate init                              # generate simulate.yaml from observed cycle-1 ops
ork simulate init --suite                      # write a suite aggregator from discovered leaf files
```

## What works

| Feature | How |
|---------|-----|
| Declarative operators (templates, status, conditions) | Always — no binary needed |
| `external:` HTTP calls | Hits real network by default; pass `--skip-external` to stub |
| `cross:` CRD observation | Include sibling CRs separated by `---` in the CR file |
| Go hooks (`OnReconcile`, `OnDelete`) | Build your operator binary with `make registry && make build` |
| Custom constructors | Same binary requirement as hooks |
| Multi-CRD Katalogs | Multi-doc CR file — each CRD matched to its CR by `kind` |

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Read the output and understand what each line means | [docs/01-output.md](docs/01-output.md) |
| Understand steady state and how to tune cycles | [docs/02-steady-state.md](docs/02-steady-state.md) |
| Understand what simulate does not cover | [docs/03-limitations.md](docs/03-limitations.md) |
| Understand the internals (fake cluster, reactors, indexer, cross: wiring) | [docs/04-internals.md](docs/04-internals.md) |
