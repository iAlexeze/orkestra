# Orkestra Testing Guide

This document covers how tests are organized, how to run them, and what each
category is testing.

---

## How to run

```bash
make test-unit          # fast, no cluster — run this constantly
make test-race          # same but with Go's race detector — run before every PR
make test-integration   # needs envtest binaries (see below)
make test-all           # unit + integration
make test-coverage      # unit tests + HTML coverage report (coverage.html)
```

---

## Three categories of test

### 1. Unit tests — `pkg/**/*_test.go`

Unit tests live next to the code they test, inside each `pkg/` sub-package.
They run with `go test ./pkg/...` and require nothing outside the process:
no files, no cluster, no network.

**When to write one**: any pure function — validation logic, string manipulation,
state machines, math, filtering. If the function takes inputs and returns
outputs without talking to Kubernetes or the filesystem, it belongs here.

**What they cover today**:

| Package | Coverage | What is tested |
|---------|----------|----------------|
| `pkg/profiles` | 91.1% | Profile resolution and overlay |
| `pkg/health` | 77.8% | Admission stats, conversion stats, deletion/namespace protection stats |
| `pkg/note` | 72.9% | Note catalog lookups |
| `pkg/gateway/certmanager` | 71.6% | Certificate lifecycle |
| `pkg/runtime/queue` | 56.9% | Workqueue enqueue/dequeue/depth/shutdown |
| `pkg/registry/motif` | 51.4% | Motif loading, validation, expansion |
| `pkg/utils` | 47.8% | Template rendering, merge helpers, YAML formatting |
| `pkg/tools/generate` | 40.7% | Pure code-gen helpers (spec path extraction, type inference, alias dedup) |
| `pkg/gateway/api` | 37.7% | Gateway API auth and token resolution |
| `pkg/merger` | 25.6% | Katalog/Komposer merge, provider/security/notification accumulation |
| `pkg/metrics` | 24.7% | Prometheus counter/gauge wrappers |
| `pkg/tools/migrate` | 82.7% | Reconcile method migration to Orkestra constructor signature |
| `pkg/gateway/webhook` | 18.8% | Admission webhook routing |
| `pkg/registry` | 17.1% | OCI push/pull, reference resolution, artifact kind detection |
| `pkg/runtime/informer` | 15.0% | Namespace filter logic, GVK normalization |
| `pkg/runtime/reconciler` | 14.0% | Validation rule pipelines, rollback gate |
| `pkg/runtime/kordinator` | 10.9% | Parent-ready extraction, condition parsing |
| `pkg/kubeclient` | 1.3% | Context wiring (patch operations covered by integration tests) |

**Total unit coverage: ~20%** — the headline number is dragged down by large
packages (`provider/*`, `registry`) that contain almost
no testable pure logic; they are I/O and wiring. Every package that does contain
pure logic has meaningful coverage.

---

### 2. Integration tests — `tests/integration/`

Integration tests exercise behaviour that a unit test cannot reach: real file
I/O across multiple packages, or a real Kubernetes API server. They are guarded
by the `//go:build integration` build tag so `make test-unit` never runs them.

There are two flavours:

#### 2a. File-based integration tests

These run with no cluster at all — just Go and the filesystem. They test paths
that involve reading/writing real YAML files, loading multi-package pipelines,
or traversing real dependency graphs.

| Package | What it tests |
|---------|---------------|
| `tests/integration/activation/` | Informer factory lifecycle when a CRD is missing at startup then appears |
| `tests/integration/dependency/` | Topological sort order, cycle detection across the full katalog graph |
| `tests/integration/komposer/` | Merger loading Katalog and Komposer files from disk, upstream field accumulation (providers, security, notification), source deduplication |
| `tests/integration/reconciler/` | Validation rule pipelines for Deployment, Service, and Secret CRDs |

#### 2b. Envtest-based integration tests

These spin up a real in-process Kubernetes API server and etcd using
`sigs.k8s.io/controller-runtime/pkg/envtest`. The API server is real; the
cluster lives only for the duration of the test binary.

Use this category when you need to verify behaviour that only makes sense
against a real watch stream or a real patch endpoint — things that fake clients
get wrong.

| Package | What it tests |
|---------|---------------|
| `tests/integration/kubeclient/` | `PatchFinalizers`, `PatchLabels`, `PatchStatus` — verifies merge-patch semantics (idempotency, labels merge not replace, status isolation from spec) |
| `tests/integration/informer/` | Namespace filter wiring — verifies that events from a blocked namespace are silently dropped before reaching the queue, using a real Watch stream from envtest |

**Why envtest and not a fake client?**

Fake clients (like `k8s.io/client-go/kubernetes/fake`) implement a simplified
in-memory store. They do not enforce actual Kubernetes merge-patch semantics. A
merge-patch that silently duplicates finalizers or overwrites a label map would
pass with a fake client but fail against a real API server. Envtest catches
this class of bug.

**Running envtest tests** requires the `kube-apiserver` and `etcd` binaries.
Install once:

```bash
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
setup-envtest use 1.32.x --bin-dir ~/.envtest-bins
```

`make test-integration` runs `setup-envtest` automatically and sets
`KUBEBUILDER_ASSETS` for you.

---

### 3. E2E tests — `ork e2e`

E2E tests are declarative YAML documents (`kind: E2E`) executed by `ork e2e`.
They run `ork` as a real process against a real cluster and verify that
creating a CR causes the expected Deployments, Services, and health transitions.

**How it works:**

```
kind cluster → CRD apply → setup manifests → bundle generate+apply
  → ork helm install → CR apply → expectation polling → teardown
```

Teardown always runs — owned clusters are deleted unless `--keep-cluster` is
set; borrowed clusters (`--use-current` / `--cluster`) are cleaned up but not
destroyed.

**Running an E2E spec:**

```bash
ork e2e ./e2e.yaml                        # provisions a new kind cluster
ork e2e ./e2e.yaml --use-current          # uses the current kube context
ork e2e ./e2e.yaml --cluster my-ctx       # uses a specific named context
ork e2e ./e2e.yaml --keep-cluster         # skips cluster teardown (debug)
```

**Writing an E2E spec (`kind: E2E`):**

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: website-smoke
spec:
  katalog: ./katalog.yaml
  cr: ./website-cr.yaml
  cluster:
    provider: kind          # default
    name: ork-e2e           # default
  expect:
    - after: cr-applied
      timeout: 60s
      resources:
        - kind: Deployment
          name: website
          namespace: default
          fields:
            spec.replicas: 1
        - kind: Service
          name: website-svc
          namespace: default
```

Expectations are polled until they pass or the timeout expires. Each
`resources` entry can check field values, existence, and command exit codes.
Results are returned as a `*Result` with per-case timings — `ork push`
embeds them as OCI annotations.

E2E specs live alongside the Katalog they test, not under `tests/`.

---

## Directory layout

```
tests/
├── integration/
│   ├── activation/          file-based — CRD missing-then-appears lifecycle
│   ├── dependency/          file-based — topological sort, cycle detection
│   ├── health/              envtest  — webhook registration and cleanup
│   ├── informer/            envtest  — namespace filter drops blocked events
│   ├── komposer/            file-based — merger field accumulation
│   ├── kubeclient/          envtest  — patch merge-patch semantics
│   └── testenv/             shared envtest lifecycle (Start / Stop)
│
├── fixtures/
│   └── crds/
│       └── probe-crd.yaml   Probe CRD (integration.orkestra.io/v1) used by kubeclient + informer tests
│
└── helpers/
    ├── fake_kubeclient.go   Kubeclient backed by a fake clientset
    ├── fake_informer.go     Minimal informer stub for unit tests
    └── testutils.go         Shared assertion helpers
```

Per-package fixtures (living integration specs that run with `ork simulate` / `ork e2e`) live alongside the package they test:

| Package | Fixture path |
|---------|-------------|
| `pkg/runtime/reconciler` | `pkg/runtime/reconciler/fixture/` |
| `pkg/registry/e2e` | `pkg/registry/e2e/fixture/` |
| `pkg/children` | `pkg/children/fixtures/` |
| `pkg/gateway` | `pkg/gateway/*/fixture/` |
| `pkg/resources` | `pkg/resources/*/` |
| `pkg/note` | `pkg/note/fixture/` |
| `pkg/runtime/informer` | planned |
| `pkg/runtime/kordinator` | planned |
| `pkg/kubeclient` | planned |

---

## Writing a new test

### Unit test (pure logic)

Put it in the same package as the code being tested:

```
pkg/merger/merger.go
pkg/merger/merger_test.go   ← goes here
```

No build tag needed. Keep it fast — no sleeps, no goroutines unless testing
concurrency explicitly.

### File-based integration test

1. Create or pick a subdirectory under `tests/integration/`.
2. First line of every file: `//go:build integration`
3. Use `package <dir>_test` — black-box, import via public API.
4. Use `t.Cleanup` to remove temp files. Call `t.Helper()` on shared helpers.

### Envtest integration test

1. Create `tests/integration/<name>/suite_test.go` with a `TestMain`:

```go
//go:build integration

package myfeature_test

import (
    "os"
    "testing"
    "github.com/orkspace/orkestra/tests/integration/testenv"
    "k8s.io/client-go/rest"
)

var (
    testCfg *rest.Config
    testEnv *testenv.Env
)

func TestMain(m *testing.M) {
    testEnv = testenv.Start([]string{"../../fixtures/crds"})
    testCfg = testEnv.Config
    code := m.Run()
    testEnv.Stop()
    os.Exit(code)
}
```

2. Access the API server through `testEnv.Dynamic` (dynamic client) or
   `testEnv.Config` (REST config for custom clients like `kubeclient.NewForTesting`).

3. **Unstructured objects and the scheme**: when using `informer.ForListerWatcher`,
   the example object you pass must have its GVK set — the scheme resolves Kind
   from the object's metadata for unstructured types:

```go
exampleObj := &unstructured.Unstructured{}
exampleObj.SetGroupVersionKind(myGVK)
factory.ForListerWatcher(lw, exampleObj, ctx, opts)
```

4. CRD paths in `testenv.Start` are relative to the test package directory.
   From `tests/integration/<name>/` the fixtures path is `../../fixtures/crds`.

---

## Test exports pattern

When an integration test needs to reach unexported internals, add a
`test_exports.go` file inside the target package (no build tag — it compiles
into all builds but is tiny):

```go
// pkg/kubeclient/test_exports.go
package kubeclient

func NewForTesting(cfg *rest.Config, dyn dynamic.Interface, s *runtime.Scheme) *Kubeclient {
    return &Kubeclient{restConfig: cfg, dynamic: dyn, scheme: s}
}
```

Existing examples: [pkg/kubeclient/test_exports.go](../pkg/kubeclient/test_exports.go),
[pkg/merger/test_exports.go](../pkg/merger/test_exports.go),
[pkg/health/test_exports.go](../pkg/health/test_exports.go).

---

## CI

The CI pipeline runs `make test-unit` (fast gate, every commit) and
`make test-integration` (slower gate, pre-merge). The `//go:build integration`
tag guarantees the two jobs are fully isolated — there is no way for an
integration test to sneak into the unit run.

`sigs.k8s.io/controller-runtime` is a direct dependency in `go.mod` solely
for envtest. Production code (`pkg/`) uses raw `client-go` only.