# Integration Tests

Integration tests verify behaviour that requires either real file I/O and
multi-package coordination, or a real Kubernetes API server. Unit tests
(`make test-unit`) never cover these paths.

## Running

```bash
make test-integration
```

`setup-envtest` must be installed once:

```bash
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
```

The Makefile calls `setup-envtest` automatically to locate (or download) the
right API-server binaries before running the tests.

## Two kinds of integration test

### 1. File-based — no cluster required

These tests use real temp files, real dependency graphs, or real validation
pipelines. `envtest` is not involved.

| Package | What it covers |
|---------|----------------|
| `activation/` | CRD health lifecycle — appears, disappears, reappears |
| `dependency/` | Topological sort and cycle detection across the katalog graph |
| `komposer/` | Merger loading Katalog/Komposer files, source composition, field accumulation |
| `reconciler/` | Validation rule pipelines for deployment, service, and secret CRDs |

### 2. Envtest-based — embedded API server

These tests spin up a real in-process Kubernetes API server (`sigs.k8s.io/controller-runtime/pkg/envtest`)
and exercise behaviour that only makes sense against a real watch stream or
a real API patch endpoint.

| Package | What it covers |
|---------|----------------|
| `kubeclient/` | `PatchFinalizers`, `PatchLabels`, `PatchStatus` — merge-patch semantics against a real API server |
| `informer/` | Namespace filter wiring — verifies that blocked-namespace events are dropped from the queue before being enqueued, using a real Watch stream |

`KUBEBUILDER_ASSETS` must point to a directory containing `kube-apiserver`
and `etcd` binaries. `make test-integration` sets this automatically via
`setup-envtest`.

## Structure

```
tests/integration/
├── activation/       file-based
├── dependency/       file-based
├── komposer/         file-based
├── reconciler/       file-based
├── kubeclient/       envtest — kubeclient patch operations
├── informer/         envtest — namespace filter + queue wiring
├── testenv/          shared envtest lifecycle (Start / Stop)
└── README.md
```

`tests/fixtures/crds/` — CRD manifests installed into envtest for tests that
create custom resources (`Probe` CRD used by kubeclient and informer tests).

## CRD fixtures

```
tests/fixtures/crds/
├── probe-crd.yaml    integration.orkestra.io/v1 Probe — used by kubeclient + informer tests
├── orkapp-crd.yaml
└── website-crd.yaml
```

## What counts as an integration test

| Criterion | Integration | Unit |
|-----------|-------------|------|
| Spans multiple packages | Yes | No — stays in one package |
| Writes real temp files | Yes (file-based) | No |
| Requires an API server | Yes (envtest-based) | No |
| Requires a live remote cluster | No | No |
| Requires network | No | No |

Tests that need a live remote cluster (cloud provisioning, real ingress, DNS)
belong in `tests/e2e/`.

## Writing a new integration test

### File-based

1. Pick the right subdirectory — or create one for a new domain.
2. Add `//go:build integration` as the **first line** (before `package`).
3. Use `package <dir>_test` — black-box; import via the public API.
4. Avoid global state that leaks between tests. Use `t.Cleanup` for temp files.
5. Call `t.Helper()` on shared helpers so failure lines point to the caller.

### Envtest-based

1. Create `tests/integration/<name>/suite_test.go` with a `TestMain` that calls
   `testenv.Start(crdPaths)` and `testenv.Stop()`.
2. Point `crdPaths` to `../../fixtures/crds` (relative to the test package dir).
3. Access the API server via `testEnv.Dynamic` (dynamic client) or
   `testEnv.Config` (REST config for custom clients).
4. Use `kubeclient.NewForTesting(testCfg, testEnv.Dynamic, scheme)` to build a
   `Kubeclient` without going through `Start()`.
5. Use `informer.SharedInformerFactory(nil, testCfg, ...)` with
   `ForListerWatcher` for informer tests. Pass an example object with its GVK
   set (`obj.SetGroupVersionKind(gvk)`) — the scheme reads Kind from
   unstructured objects directly.

## Test helpers and test exports

Some tests need access to unexported internals. Add a `test_exports.go` file
inside the target package (no build tag — compiled into all builds, but tiny):

```go
// pkg/kubeclient/test_exports.go
package kubeclient

func NewForTesting(cfg *rest.Config, dyn dynamic.Interface, s *runtime.Scheme) *Kubeclient {
    return &Kubeclient{restConfig: cfg, dynamic: dyn, scheme: s}
}
```

Existing examples: `pkg/kubeclient/test_exports.go`, `pkg/merger/test_exports.go`,
`pkg/health/test_exports.go`.

## CI

Integration tests run in a separate job after unit tests pass. `setup-envtest`
is installed as part of the CI setup step. The `//go:build integration` tag
ensures these tests are never accidentally included in `make test-unit`.
