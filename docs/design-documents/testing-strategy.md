# Testing Strategy — Orkestra v1

*April 2026 · authoritative reference for all new test work*

---

## Current state

```
Total coverage (make test-unit):  12.7%
Packages with no tests at all:    12 of 29 measured
```

Per-package breakdown, highest to lowest:

| Package | Coverage | Test files | Notes |
|---|---|---|---|
| `pkg/labels` | 100% | 1 | Pure functions, fully covered |
| `pkg/certmanager` | 71.4% | 1 | Fake k8s client, black-box |
| `pkg/note` | 67.1% | 19 | Table-driven, white-box |
| `pkg/queue` | 56.9% | 1 | Concurrent state machine |
| `pkg/health` | 38.8% | 2 | Admission stats + constructor |
| `pkg/webhook` | 27.2% | 4 | Fake admission review objects |
| `pkg/metrics` | 24.7% | 1 | Smoke tests only |
| `pkg/generate` | 21.0% | 2 | Namespace + katalog generation |
| `pkg/katalog` | 16.8% | 3 | Dependency graph, builtins |
| `pkg/kordinator` | 7.2% | 1 | CRDHealth state machine |
| `pkg/reconciler` | 7.0% | 5 | Validation, mutation, conditions |
| `pkg/merger` | 4.3% | 2 | Registry + file loading |
| `pkg/types` | 2.9% | 2 | Admission types |
| `pkg/autoscaler` | 0% | 0 | |
| `pkg/event` | 0% | 0 | |
| `pkg/informer` | 0% | 0 | |
| `pkg/konductor` | 0% | 0 | |
| `pkg/konfig` | 0% | 0 | |
| `pkg/kubeclient` | 0% | 0 | |
| `pkg/logger` | 0% | 0 | |
| `pkg/notification` | 0% | 0 | |
| `pkg/orkestra` | 0% | 0 | |
| `pkg/orkestra-registry/*` | 0% | 0 | 18 sub-packages |
| `pkg/provider/*` | 0% | 0 | 7 sub-packages |
| `pkg/runtime` | 0% | 0 | |
| `pkg/utils` | 0% | 0 | |
| `pkg/version` | 0% | 0 | |

The 0% packages are not all equal — some have no tests because they are genuinely
hard to test (cluster-dependent I/O), others because no one has written tests yet
for pure logic that is completely testable. This document distinguishes those cases
and prescribes what to do for each.

---

## Package taxonomy

Every package in Orkestra falls into one of four categories. The category determines
the right testing approach, not the coverage number alone.

### Category A — Pure logic

No I/O, no Kubernetes API, no network. Input in, output out. These should have
high unit-test coverage because there are no constraints preventing it.

**Packages:** `pkg/note`, `pkg/labels`, `pkg/types`, `pkg/utils` (subset),
`pkg/reconciler` (validation and mutation rules), `pkg/katalog` (dependency graph,
rule evaluation), `pkg/version`

**Target coverage: ≥ 80%**

Test with: table-driven unit tests in `*_test.go` alongside the source.
White-box (`package foo`) is fine when the tested unit is an unexported helper;
black-box (`package foo_test`) is preferred when testing the exported API.
No fake clients, no `envtest`, no test clusters.

---

### Category B — Kubernetes-wrapping

Calls Kubernetes API but its behaviour is fully determined by the API response.
The business logic is in this package; Kubernetes is just a persistence layer.

**Packages:** `pkg/certmanager`, `pkg/webhook`, `pkg/generate`, `pkg/kubeclient`,
`pkg/kordinator` (CRDHealth), `pkg/health`, `pkg/metrics`, `pkg/queue`,
`pkg/orkestra-registry/*`

**Target coverage: ≥ 65%**

Test with: `k8s.io/client-go/kubernetes/fake`.
Fake clients cover the full API surface without a cluster; they are the right tool
for asserting that your package correctly calls Get, Create, Patch, and Update
and handles 404 / conflict errors. Anything that requires watching (informers,
leader election) is integration territory.

Do NOT test with real clients in unit tests. A fake client that mis-behaves
compared to a real API server is caught by integration tests, not here.

---

### Category C — Cluster-dependent

Correct behaviour requires a running API server: watching real events, informer
cache sync, webhook TLS handshake, leader election, or applying CRs and waiting
for status updates.

**Packages:** `pkg/informer`, `pkg/orkestra`, `pkg/konductor`, `pkg/runtime`,
the reconcile-loop entry points in `pkg/kordinator` and `pkg/reconciler`

**Target coverage: not measured by unit tests**

Test with: `envtest` in `tests/integration/` (no cluster, real API server binary)
or real cluster with orkestra-action in `.github/e2e/`. Unit tests for pure helpers inside these
packages are still encouraged but not the primary test vehicle.

---

### Category D — Thin wrappers / adapters

Wraps an external library or standard library with no business logic of its own.
Testing the wrapper tests the dependency, not Orkestra code.

**Packages:** `pkg/logger`, `pkg/event`, `pkg/notification`, `pkg/konfig`
(env-var loading), `pkg/provider/*` (database/cloud SDK wrappers)

**Target coverage: not a v1 priority**

Test with: a single smoke test confirming the constructor does not panic and the
interface is satisfied. Do not write elaborate mocks for external cloud SDKs.
Integration or contract tests belong in a dedicated `tests/provider/` directory
and are out of scope for v1.

---

## The three-tier pyramid

```
              ┌──────────────────────────┐
              │     E2E (Tier 3)         │  real cluster
              │                          │  few, slow, high confidence
              ├──────────────────────────┤
              │  Integration (Tier 2)    │  //go:build integration
              │  tests/integration/      │  no cluster, real types
              ├──────────────────────────┤
              │    Unit (Tier 1)         │  make test-unit
              │  pkg/*/                  │  fast, many, isolated
              └──────────────────────────┘
```

Each tier has a distinct contract. Do not push tests down to a lower tier just
to hit a coverage number, and do not push tests up to a higher tier because the
lower tier was hard to set up.

### Tier 1 — Unit tests

- Location: `pkg/<pkg>/<file>_test.go`
- Build constraint: none (always compiled)
- Cluster required: no
- External deps: none (`fake` clients are acceptable)
- Speed: entire suite in under 10 seconds
- Command: `make test-unit` → `go test ./pkg/... -short -count=1`

Table-driven tests are the default pattern. Each test function covers one
behaviour; each table row covers one input variant. If a test requires more
than ~20 lines of setup, extract a builder helper (`makeWebsite(...)`,
`buildUnstructured(...)`).

### Tier 2 — Integration tests

- Location: `tests/integration/<topic>/`
- Build constraint: `//go:build integration` (first line of every file)
- Cluster required: no (uses `envtest` or real YAML fixtures)
- Speed: under 60 seconds
- Command: `make test-integration` → `go test ./tests/integration/... -tags=integration`

Use for: behaviour that spans multiple packages, real file I/O, real YAML parsing,
real dependency-graph traversal, or anything that requires the Kubernetes type
system but not a running API server. The existing `tests/integration/` README
gives the precise inclusion criteria.

### Tier 3 — E2E tests

- Location: `.github/workflows/e2e-*.yml`
- Cluster required: yes (`kind` cluster created in CI)
- Speed: minutes (triggered manually via `workflow_dispatch`)
- Tool: `orkspace/orkestra-action` + `ork init`

E2E tests run as GitHub Actions workflows. Each workflow targets one numbered
example from a beginner or advanced pack:

```
e2e-beginner-pack-01.yml  →  examples/beginner/01-hello-website
e2e-beginner-pack-02.yml  →  examples/beginner/02-...
e2e-advanced-pack-09.yml  →  examples/advanced/09-hooks
```

The pattern for every E2E workflow:

1. **Spin up a kind cluster** via `helm/kind-action`
2. **`orkestra-action` with `init: true`** — runs `ork init` on the target example;
   outputs katalog path, CRD path, bundle file, namespace
3. **Validate** — confirms the katalog is structurally valid before applying
4. **Generate and apply** — bundle or registry generated, CRD and bundle applied
5. **Install Orkestra** — Helm chart installed into the cluster
6. **Apply CR** — `kubectl apply -f ${{ steps.ork.outputs.cr_path }}`
7. **Assert** — verify expected resources exist (`kubectl wait`, `kubectl get`)
8. **Delete CR** — verify cleanup removes all managed resources
9. **Debug on failure** — operator logs posted to step summary

For advanced examples requiring custom binaries (typed hooks, extensions), the
workflow additionally builds the binary, pushes a Docker image, and passes the
custom image reference to the Helm chart.

Use for: full operator lifecycle (apply CRD → apply CR → reconcile → delete →
cleanup), hook execution, admission webhook interception, and any behaviour that
requires Orkestra's reconcile loop to run against a real API server.

---

## The katalog approach in pkg/note — evaluation

`pkg/note/example/` contains a live Katalog (`katalog.yaml`) and a companion
CR (`cr.yaml`) that exercise every Kubernetes-family note by applying them to
real child objects and inspecting the status fields. This is a **good pattern**
for the specific problem it solves, but it is not the right pattern for all
packages and should not be generalised.

### Why it exists

The Kubernetes-family notes (`kube_replica.go`, `kube_container.go`,
`kube_job.go`, `kube_service.go`, `kubernetes.go`) take `map[string]interface{}`
from the dynamic client as input. Unit tests construct these maps by hand. Real
API responses differ from hand-crafted maps in subtle ways: extra metadata fields,
different numeric types after JSON round-tripping, absent optional fields. The
katalog example catches regressions that unit tests cannot because it exercises
the full path from API server → dynamic client → `map[string]interface{}` → note
function → status update.

### When it is appropriate

Use a katalog-style integration fixture for any note or function that:
1. Consumes data originating from a live Kubernetes object, AND
2. Has tripped over the hand-crafted-map vs real-API-response gap before.

This is currently only true of the kubernetes-family notes. Pure-logic notes
(`math`, `strings`, `conditional`, `cron`, `random`, `collections`) do not need
it — unit tests are sufficient and complete for them.

### When it is not appropriate

Do not replicate this pattern for packages whose core logic is not note-function
evaluation. The katalog approach is a niche e2e tool, not a general-purpose
integration test framework. For packages like `pkg/certmanager`, `pkg/webhook`,
or `pkg/queue`, fake clients and `envtest` cover the equivalent gap at lower cost
and without a real cluster dependency.

### Summary

| Package type | Right test vehicle |
|---|---|
| Pure-logic notes | Unit tests only |
| Kubernetes-family notes | Unit tests + katalog example (e2e) |
| Kubernetes-wrapping packages | Unit tests with fake client |
| Cluster-dependent packages | `envtest` in tests/integration/ |
| Full operator behaviour | tests/e2e/ scripts |

---

## Coverage targets for v1

These are the targets to reach before tagging v1.0. They are not arbitrary
percentages — each target is set at the level where meaningful new regressions
are caught and the test maintenance burden is proportional.

| Package | Current | v1 target | Why |
|---|---|---|---|
| `pkg/note` | 67% | 80% | Pure logic; every branch is testable |
| `pkg/labels` | 100% | 100% | Already there |
| `pkg/certmanager` | 71% | 80% | Core TLS lifecycle |
| `pkg/queue` | 57% | 75% | Concurrent state machine |
| `pkg/health` | 39% | 70% | Admission stats + constructor |
| `pkg/webhook` | 27% | 70% | Admission rule evaluation |
| `pkg/types` | 3% | 80% | Public contract types |
| `pkg/reconciler` | 7% | 70% | Validation + mutation rules |
| `pkg/katalog` | 17% | 65% | Dependency graph + builtins |
| `pkg/merger` | 4% | 60% | Registry loading |
| `pkg/kordinator` | 7% | 70% | CRDHealth state machine |
| `pkg/generate` | 21% | 60% | Namespace + katalog generation |
| `pkg/metrics` | 25% | 50% | Smoke tests; registration wiring |
| `pkg/utils` | 0% | 60% | Pure helpers — no excuse |
| `pkg/version` | 0% | 30% | Smoke test for ldflags wiring |
| `pkg/kubeclient` | 0% | 50% | Fake client for patch/apply logic |
| `pkg/orkestra-registry/*` | 0% | 50% | Registry object builders |
| `pkg/logger` | 0% | skip | Thin wrapper |
| `pkg/konfig` | 0% | skip | Env-var loader |
| `pkg/provider/*` | 0% | skip | External SDK wrappers |
| Cluster-dependent packages | — | skip | Covered at integration/e2e |

---

## Priority order for new test work

1. **`pkg/types`** — Public contract used by every package. 3% is unacceptable.
   All exported methods on `AdmissionConfig`, `ValidationRule`, `MutationRule`
   should have table-driven unit tests. No fakes needed.

2. **`pkg/reconciler`** — Validation and mutation rule pipelines are the core
   operator feature. The test files exist but cover only 7% because they use
   white-box access to a few functions. Expand to cover all operators, all
   mutation modes, all error paths.

3. **`pkg/utils`** — ~900 lines of pure functions, zero tests. Pure Category A.
   There is no reason for 0% here.

4. **`pkg/webhook`** — Admission evaluation, conversion, and registration all
   have test files but at 27%. The evaluation logic (the rule pipeline) is
   pure and should reach 80%+.

5. **`pkg/merger` and `pkg/katalog`** — Both have test files but low coverage
   because only happy paths are exercised. Add error paths and edge cases.

6. **`pkg/kubeclient`** — All the patch and apply helpers can be tested with
   `fake.NewSimpleClientset()`. No cluster required.

7. **`pkg/orkestra-registry/*`** — 18 sub-packages, all 0%. Each sub-package
   builds resource objects (Deployment, Service, etc.). These are pure builders
   that return typed structs. Table-driven unit tests with field assertions.

---

## Patterns and conventions

### Table-driven tests

```go
func TestEvaluateRule(t *testing.T) {
    tests := []struct {
        name    string
        rule    ValidationRule
        obj     *unstructured.Unstructured
        want    bool
        wantErr bool
    }{
        {"equals — match",   ruleEq("v1"), obj("spec.version", "v1"), true, false},
        {"equals — no match", ruleEq("v1"), obj("spec.version", "v2"), false, false},
        {"missing field",     ruleEq("v1"), obj("spec.other", "v1"),  false, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := evaluateRule(tt.obj, tt.rule)
            if (err != nil) != tt.wantErr {
                t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Fake Kubernetes client

```go
func TestEnsureCertificate_Idempotent(t *testing.T) {
    cs := fake.NewSimpleClientset()
    mgr := certmanager.New(cs)
    ctx := context.Background()

    spec := certmanager.CertificateSpec{ServiceName: "ork", Namespace: "default"}
    first, _ := mgr.EnsureCertificate(ctx, spec)
    second, _ := mgr.EnsureCertificate(ctx, spec)

    if first.CertPEM != second.CertPEM {
        t.Error("expected idempotent: second call must return same cert")
    }
}
```

### Using `t.Helper()` in shared builders

```go
func buildCR(t *testing.T, spec map[string]interface{}) *unstructured.Unstructured {
    t.Helper()
    return &unstructured.Unstructured{Object: map[string]interface{}{
        "apiVersion": "demo.orkestra.io/v1",
        "kind":       "Website",
        "metadata":   map[string]interface{}{"name": "test", "namespace": "default"},
        "spec":       spec,
    }}
}
```

### Race detection

Any package with shared mutable state must pass `make test-race`. Current
candidates: `pkg/queue`, `pkg/health`, `pkg/kordinator`, `pkg/metrics`.

```bash
make test-race    # run before every PR that touches concurrent code
```

### Numeric type precision in tests

`pkg/note` math functions pass results through `nativeNumber()` which converts
whole-number float64 results to int64. When writing expected values:

```go
// Wrong — 1.5 + 2.5 = 3.0 (float64), but nativeNumber converts it
{"float + float", 1.5, 2.5, 4.0, false}

// Correct
{"float + float", 1.5, 2.5, int64(4), false}
```

When comparing slice or map values with `==`, use `reflect.DeepEqual` — the
`==` operator panics on non-comparable types.

---

## What does NOT need tests

- **Generated code** (`catalog_generated.go`) — test the generator, not the output.
- **Main packages** (`cmd/orkestra/`, `cmd/cli/`) — test via E2E or acceptance tests.
- **Thin logging wrappers** — testing that `log.Info` was called is not valuable.
- **One-liner accessor methods** with no logic — `func (h *CRDHealth) Name() string { return h.name }`.
