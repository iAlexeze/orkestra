---
title: "Testing Strategy"
weight: 164
---

# Orkestra Testing Strategy

*Version 1.0 — March 2026*

---

## Overview

Orkestra follows a three-tier test pyramid aligned with CNCF operator
project standards: unit tests at the base (fast, no dependencies), integration
tests in the middle (real Go + Kubernetes types, no live cluster), and end-to-end
tests at the top (live cluster, full reconciliation cycles).

```
         ┌─────────────────────────┐
         │      E2E Tests          │  cluster required
         │   tests/e2e/            │  slow, few, high confidence
         ├─────────────────────────┤
         │  Integration Tests      │  build tag: integration
         │  tests/integration/     │  kubernetes types, no cluster
         ├─────────────────────────┤
         │     Unit Tests          │  no external deps, < 1s
         │  pkg/*/  tests/unit/    │  fast, many, high isolation
         └─────────────────────────┘
```

---

## Guiding Principles

**Test the contract, not the implementation.**
Unit tests assert what a function does — its inputs, outputs, and side effects —
not how it does it. This allows internal refactoring without rewriting tests.

**No mocks for internal logic.**
Orkestra's business logic (rule evaluation, field resolution, stats accounting)
is pure: it takes inputs, returns outputs, and has no I/O. These functions are
tested directly with table-driven test cases. Mocks are reserved for components
that bridge Kubernetes API boundaries (e.g., fake clients in integration tests).

**Table-driven tests for combinatorial cases.**
All operator tests (`evaluateOneRule`, `resolveFieldPath`, `anyToString`) use
table-driven patterns. Adding a new operator variant means adding a table row,
not a new test function.

**Race detection is a first-class concern.**
`AdmissionStats` and `QueueRegistry` are shared state accessed concurrently.
The race detector (`make test-race`) is a mandatory pre-PR check.

**Tests live alongside the code they test.**
Pure logic tests (`admission_evaluation_test.go`, `admission_stats_test.go`)
are in `package health` inside `pkg/health/`. This gives direct access to
unexported functions without exported test shims. Integration-level tests
that cross package boundaries live in `tests/unit/`.

---

## Test Tiers

### Tier 1 — Unit Tests

**Location:** `pkg/health/`, `pkg/types/`, `pkg/metrics/`, `tests/unit/`

**What they cover:**

| Package | Test files | What is tested |
|---|---|---|
| `pkg/health` | `admission_evaluation_test.go` | `resolveFieldPath`, `setFieldPath`, `anyToString`, `evaluateOneRule` (all 9 operators), `evaluateValidationRules`, `applyMutationRules` (default/override/nested), `deepCopyMap`, `buildJSONPatch`, `gvrToKey` |
| `pkg/health` | `admission_stats_test.go` | `AdmissionStats` counters, latency tracking (min/max/avg), P95 ring buffer, concurrent writes, snapshot isolation |
| `pkg/health` | `conversion_test.go` | `applyConversion` up/down conversion, same-version no-op, missing path error, `ConversionRules.FindPath`, full round-trip |
| `pkg/types` | `admission_test.go` | `EffectiveAction`, `IsDeny/IsWarn`, `HasDenyRules/HasWarnRules`, `WebhookValidationEnabled/WebhookMutationEnabled`, `EffectiveOperations` |
| `pkg/types` | `types_test/restricted_test.go` | Restricted namespace matching and wildcard patterns |
| `pkg/metrics` | `metrics_test.go` | Every public helper function: smoke test (no panic, no double-registration) |
| `tests/unit/health` | `conversion_test.go` | External API: `ProcessConversionReviewForTest`, `ExportedApplyConversion` |
| `tests/unit/kordinator` | `crd_health_test.go` | `CRDHealth`: initial state, `RecordSuccess/Failure`, threshold degradation, error rate, consecutive fail reset |
| `tests/unit/queue` | `workqueue_test.go` | `Workqueue` and `QueueRegistry`: lifecycle (Start/Shutdown/Started), depth, registration, retrieval |
| `tests/unit/reconciler` | `conditions_test.go` | Condition evaluation: all operators, nested field traversal, multi-condition AND logic |

**How to run:**

```bash
make test-unit         # all unit tests, verbose
make test-race         # with Go race detector (run before every PR)
make test-coverage     # HTML coverage report → coverage.html
make test-coverage-text  # per-function coverage summary
```

**Properties:**
- No network calls, no file system writes, no Kubernetes cluster
- All tests complete in under 5 seconds on a laptop
- `-short` flag skips the concurrency stress test (`TestAdmissionStats_ConcurrentWrites`)
  to keep CI fast. Run without `-short` locally when changing concurrent code.

---

### Tier 2 — Integration Tests

**Location:** `tests/integration/`

**Build tag:** `integration` — never compiled during unit test runs

**What they cover:**
- Katalog parsing with real YAML fixtures
- Conversion registry roundtrips with real `ConversionRules`
- Merger composition with real source files
- Informer factory startup and cache sync with `envtest`

**How to run:**

```bash
make test-integration    # requires KUBECONFIG or envtest binary
```

Integration tests use `envtest` from `sigs.k8s.io/controller-runtime/pkg/envtest`
to spin up a local API server without a cluster. Fixtures are in `tests/fixtures/`.

---

### Tier 3 — End-to-End Tests

**Location:** `tests/e2e/`

**What they cover:**
- Full operator lifecycle: apply CRD → apply CR → observe reconciliation → delete CR → verify cleanup
- Health endpoint responses at each state transition
- Admission webhook interception: valid CR created, bad CR rejected, mutations applied to stored object
- `ork status` output reflects actual cluster state

**How to run:**

```bash
make test-e2e           # requires a running cluster with Orkestra deployed
./tests/e2e/run.sh website
./tests/e2e/run.sh activation
./tests/e2e/run.sh dependencies
```

E2E tests are the authoritative tests for admission webhook behavior. Unit tests
verify rule evaluation logic in isolation; E2E tests verify that the webhook
registration, TLS handshake, Kubernetes API server interception, and response
handling all work together.

---

## What Is Explicitly Not Unit-Tested

The following are tested at integration or E2E level only:

| Component | Why not unit-tested |
|---|---|
| `HealthServer` HTTP handlers | Require `httptest.Server`, admission registry, and a wired HealthServer — integration concern |
| Informer factory | Requires `cache.SharedIndexInformer` and a real client or `envtest` |
| `pkg/inspect` discovery | Requires a live Kubernetes discovery API |
| Full katalog parsing | Requires YAML fixtures and enrichment of full CRD entries |
| Webhook registration (`RegisterWebhooks`) | Requires `admissionregistration.k8s.io` API group — integration concern |

---

## Coverage Targets

| Area | Target | Rationale |
|---|---|---|
| `pkg/health` evaluation | ≥ 85% | Pure logic; every branch is testable |
| `pkg/health` stats | ≥ 90% | Ring buffer and percentile code is safety-critical |
| `pkg/types` admission | ≥ 95% | Public contract types; every method tested |
| `pkg/metrics` helpers | ≥ 80% | Smoke tests confirm wiring; value accuracy is E2E |
| `tests/unit/kordinator` | ≥ 80% | CRDHealth is core state machine |
| `tests/unit/reconciler` | ≥ 75% | Condition evaluation is critical path |

Run `make test-coverage-text` to see the current per-function summary.

---

## CI Integration

Recommended GitHub Actions pipeline:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version-file: go.mod }

      - name: Vet
        run: make vet

      - name: Unit tests
        run: make test-unit

      - name: Race detector
        run: make test-race

      - name: Coverage
        run: make test-coverage-text

  integration:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - name: Integration tests
        run: make test-integration
```

Unit tests and race detection run on every push and PR.
Integration tests run only on push to `main` to avoid cluster provisioning overhead on every PR.
E2E tests are run manually before releases.

---

## Writing New Tests

### For a new validation operator

1. Add a table row to `TestEvaluateOneRule` in `pkg/health/admission_evaluation_test.go`
2. Add a row to `TestResolveValidationOperator` if a new shorthand was added

### For a new metric

1. Add a `TestRecord<Name>_DoesNotPanic` test in `pkg/metrics/metrics_test.go`
2. Verify the label set matches the counter definition

### For a new stats field (e.g., adding a `MutationWarned` counter)

1. Add unit tests for the new `Record...` method in `pkg/health/admission_stats_test.go`
2. Add the field to the `TestGetStats_CountersReflectedInSnapshot` assertion
3. Add to the concurrency test `TestAdmissionStats_ConcurrentWrites`

### For a new CRD health state

1. Update `TestCRDHealth_InitialState` in `tests/unit/kordinator/crd_health_test.go`
2. Add a test for the new transition path

### For a new admission rule type (beyond validation/mutation)

1. Create `pkg/health/<ruletype>_evaluation_test.go` in `package health`
2. Follow the table-driven pattern from `admission_evaluation_test.go`
3. Add integration tests in `tests/integration/` if the rule type touches Kubernetes types

---

## Test File Naming Convention

| Pattern | Purpose |
|---|---|
| `pkg/<pkg>/<source>_test.go` with `package <pkg>` | White-box unit test — same package, accesses unexported |
| `pkg/<pkg>/<source>_test.go` with `package <pkg>_test` | Black-box unit test — external, only exported API |
| `tests/unit/<pkg>/<test>_test.go` | Cross-package unit test — typically tests public API surface |
| `tests/integration/<pkg>/<test>_test.go` | Integration test — `// +build integration` tag |
| `tests/e2e/<name>/` | End-to-end test shell scripts with kubectl |

---

*This document should be updated whenever a new test tier, coverage target,
or architectural boundary changes.*
