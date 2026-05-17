# pkg/e2e

`e2e` runs a declarative end-to-end test against a real Kubernetes cluster. Give it a spec file and it orchestrates the full lifecycle — cluster creation, CRD apply, operator install, CR apply, expectation checking, and cleanup — the same way locally and in CI.

```bash
ork e2e -f e2e.yaml
```

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand the spec file format and all fields | [docs/01-spec.md](docs/01-spec.md) |
| Understand how expectations are evaluated (resources, commands, polling) | [docs/02-expectations.md](docs/02-expectations.md) |
| Understand the full run pipeline (what happens in order) | [docs/03-pipeline.md](docs/03-pipeline.md) |
| Understand cluster lifecycle (kind, reuse, context restore) | [docs/04-cluster.md](docs/04-cluster.md) |
