# Testing Operators

Orkestra provides a complete testing framework to ensure your operators work correctly. Tests are organized into three layers:

- **Unit tests** — test individual functions in isolation
- **Integration tests** — test components with fake Kubernetes clients
- **E2E tests** — test the full operator against a real cluster

---

## Testing Philosophy

Test for confidence, not coverage. Every test should answer a question you're afraid to answer without it:

```
What breaks? → Write a test.
What must work? → Write a test.
What did we fix? → Write a test.
```

---

## The Testing Pyramid

```
        ┌─────────────┐
        │    E2E      │  3-5 tests (slow, critical paths)
        │  (real K8s) │
        ├─────────────┤
        │ Integration │  10-20 tests (medium, component interactions)
        │  (fake K8s) │
        ├─────────────┤
        │    Unit     │  50-100 tests (fast, logic)
        │  (pure Go)  │
        └─────────────┘
```

---

## Unit Tests

Unit tests test single functions in isolation, with no external dependencies.

### Example: Testing CRDHealth

```go
// tests/unit/kordinator/crd_health_test.go
package kordinator_test

import (
    "testing"

    "github.com/orkspace/orkestra/pkg/kordinator"
    "github.com/stretchr/testify/assert"
)

func TestCRDHealth_RecordFailure(t *testing.T) {
    h := kordinator.NewCRDHealth("test")

    h.RecordFailure(fmt.Errorf("error"), 3)

    assert.Equal(t, int64(1), h.FailedReconciles())
    assert.Equal(t, int64(1), h.ConsecutiveFails())
    assert.False(t, h.IsHealthy())
}
```

### Running Unit Tests

```bash
make test-unit
# or
go test ./tests/unit/... -short
```

---

## Integration Tests

Integration tests use fake Kubernetes clients to test component interactions without a real cluster.

### Example: Testing Reconciler with Fake Client

```go
// tests/integration/reconciler_test.go
package reconciler_test

import (
    "context"
    "testing"

    "github.com/orkspace/orkestra/pkg/reconciler"
    "github.com/orkspace/orkestra/tests/helpers"
    "github.com/stretchr/testify/assert"
)

func TestGenericReconciler_CreatesDeployment(t *testing.T) {
    // Setup fake client
    fakeKube := helpers.NewFakeKubeclient()

    // Create CRD entry with Deployment template
    crd := &orktypes.CRDEntry{
        Name: "website",
        ReconcilerConfig: orktypes.ReconcilerConfig{
            OnCreate: &orktypes.HookTemplates{
                Deployments: []orktypes.DeploymentTemplateSource{
                    {Image: "nginx:latest", Replicas: "3"},
                },
            },
        },
    }

    // Create reconciler with fake dependencies
    reconciler := helpers.NewTestReconciler(crd, fakeKube)

    // Reconcile
    err := reconciler.Reconcile(context.Background(), "default/test-site")

    // Verify
    assert.NoError(t, err)

    deployment, err := fakeKube.Clientset.AppsV1().Deployments("default").Get(
        context.Background(), "test-deployment", metav1.GetOptions{})
    assert.NoError(t, err)
    assert.Equal(t, "nginx:latest", deployment.Spec.Template.Spec.Containers[0].Image)
}
```

### Running Integration Tests

```bash
make test-integration
# or
go test ./tests/integration/... -tags=integration
```

---

## E2E Tests

E2E tests run against a real Kubernetes cluster (using `kind`) to verify the entire system works.

### Example: Website E2E Test

```bash
#!/bin/bash
# tests/e2e/website/test.sh
set -e

echo "Testing: Website CRD → Deployment + Service"

# Install CRD
kubectl apply -f examples/website/website-crd.yaml

# Start Orkestra
/tmp/ork run --file examples/website/website-katalog.yaml &
ORK_PID=$!
sleep 5

# Apply CR
kubectl apply -f examples/website/website-cr.yaml
sleep 5

# Verify Deployment
kubectl get deployment test-website -n default
DEPLOYMENT_EXISTS=$?

# Verify Service
kubectl get service test-website-svc -n default
SERVICE_EXISTS=$?

# Cleanup
kill $ORK_PID

if [ $DEPLOYMENT_EXISTS -eq 0 ] && [ $SERVICE_EXISTS -eq 0 ]; then
    echo "✅ E2E test passed"
    exit 0
else
    echo "❌ E2E test failed"
    exit 1
fi
```

### Running E2E Tests

```bash
make test-e2e
# or
./tests/e2e/run.sh website
```

---

## Test Helpers

Orkestra provides helpers to make testing easier:

| Helper | Purpose |
|--------|---------|
| `NewFakeKubeclient()` | Creates a fake Kubernetes client |
| `NewFakeKubeclientWithObjects()` | Pre‑populated fake client |
| `NewTestReconciler()` | Reconciler with fake dependencies |
| `AssertNoError()` | Common assertion helper |
| `AssertError()` | Error assertion with message check |

### Example: Using Test Helpers

```go
func TestSomething(t *testing.T) {
    // Setup
    fakeKube := helpers.NewFakeKubeclientWithObjects(existingObjects...)
    reconciler := helpers.NewTestReconciler(crd, fakeKube)

    // Act
    err := reconciler.Reconcile(ctx, key)

    // Assert
    helpers.AssertNoError(t, err)
}
```

---

## Test Fixtures

Reusable test data lives in `tests/fixtures/`:

```
tests/fixtures/
├── katalogs/
│   ├── website.yaml
│   ├── dependencies.yaml
│   └── komposer.yaml
└── crds/
    ├── website-crd.yaml
    └── orkapp-crd.yaml
```

---

## CI Integration

GitHub Actions runs tests on every push and PR:

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  unit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make test-unit

  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make test-integration

  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: helm/kind-action@v1
      - run: make test-e2e
```

---

## What to Test First

| Priority | What | Why |
|----------|------|-----|
| **P0** | Website E2E | The "hello world" must work |
| **P0** | Missing CRD activation | Core value: CRDs appear later |
| **P0** | Dependency ordering | Core feature |
| **P1** | Health tracking | Critical for observability |
| **P1** | Katalog merge | Komposer correctness |
| **P1** | Reconciler creates resources | Templates work |
| **P2** | Drift correction | reconcile:true works |
| **P2** | Graceful shutdown | No panic on exit |
| **P3** | Metrics emission | Prometheus works |

---

## Next Steps

You've covered the entire operator lifecycle. Now you're ready to build your own operators with Orkestra.

👉 [Full Documentation Index →](../index.md)