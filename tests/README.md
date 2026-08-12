# Tests

## Run

```bash
make test-unit          # fast, no cluster — run this constantly
make test-race          # unit tests with Go's race detector — run before every PR
make test-integration   # envtest integration tests (setup-envtest auto-installed if missing)
make test-all           # unit + control-center + integration
make test-coverage      # unit tests + HTML coverage report (coverage.html)
```

---

## Testing tiers

| Tier | Where | Run with |
|------|-------|----------|
| Unit | `pkg/**/*_test.go` | `make test-unit` |
| Integration | `tests/integration/` | `make test-integration` |
| Simulate (fake) | `examples/` and alongside operators | `ork simulate` |
| Simulate (envtest) | `tests/simulate-envtest/` | `ork simulate --envtest` (binaries auto-downloaded) |
| E2E | alongside each Katalog (`e2e.yaml`) | `ork e2e` |

**Unit tests** live next to the code they test. No cluster, no files, no network — pure logic only.

**Integration tests** use either the filesystem or a real in-process API server (envtest). Guarded by `//go:build integration` so they never run during `make test-unit`.

**Simulate (fake)** runs the reconciler against an in-memory fake cluster — instant, catches template and expression errors.

**Simulate (envtest)** runs the same simulate.yaml against a real `kube-apiserver` + `etcd`. Catches CRD schema violations, status subresource enforcement, and real watch stream behaviour that fake clients miss. Binaries auto-downloaded to `~/.ork/envtest-bins` on first run.

**E2E** runs `ork` against a real cluster. Catches webhooks, pod scheduling, provider integrations — anything that requires a live environment.

→ See [Declarative Unit Testing](../documentation/concepts/simulate/index.md), [Declarative Integration Testing](../documentation/concepts/envtest/index.md), [Declarative End-to-End Testing](../documentation/concepts/e2e/index.md)

---

## Directory layout

```
tests/
├── integration/        envtest and file-based integration tests
│   ├── activation/     CRD missing-then-appears lifecycle
│   ├── dependency/     topological sort, cycle detection
│   ├── health/         webhook registration and cleanup
│   ├── informer/       namespace filter drops blocked events
│   ├── komposer/       merger field accumulation
│   ├── kubeclient/     patch merge-patch semantics
│   └── testenv/        shared envtest lifecycle (Start / Stop)
│
├── simulate-envtest/   simulate.yaml tests that run with --envtest
│
└── fixtures/
    └── crds/
        └── probe-crd.yaml   Probe CRD used by kubeclient + informer tests
```
