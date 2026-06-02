# pkg/e2e

`e2e` runs a declarative end-to-end test against a real Kubernetes cluster. Give it a spec file and it orchestrates the full lifecycle — cluster creation, CRD apply, operator install, CR apply, expectation checking, and cleanup — the same way locally and in CI.

```bash
ork e2e -f e2e.yaml
ork e2e ./...                    # discover and run all *e2e.yaml files recursively
ork e2e ./examples/beginner/...  # scoped discovery
```

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand the spec file format and all fields | [docs/01-spec.md](docs/01-spec.md) |
| Understand how expectations are evaluated (resources, commands, polling) | [docs/02-expectations.md](docs/02-expectations.md) |
| Understand the full run pipeline (what happens in order) | [docs/03-pipeline.md](docs/03-pipeline.md) |
| Understand cluster lifecycle (kind, reuse, context restore, shared Orkestra) | [docs/04-cluster.md](docs/04-cluster.md) |
| Compose test suites with imports and the `wait:` field | [docs/05-imports.md](docs/05-imports.md) |
| Use `./...` discovery mode, `--wait`, `--skip`, `--dry-run` | [docs/06-discovery.md](docs/06-discovery.md) |
| Test any operator without Orkestra (`customOperator: true`) | [docs/07-custom-operator.md](docs/07-custom-operator.md) |
