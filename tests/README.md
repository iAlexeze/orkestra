# Orkestra Testing Strategy — A Complete Plan

This is your blueprint for testing Orkestra from first principles to production readiness.

---

## The Philosophy

**Test for confidence, not coverage.** Every test should answer a question you're afraid to answer without it.

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
        │  (real K8s) │  What: "Does the whole thing work?"
        ├─────────────┤
        │ Integration │  10-20 tests (medium, component interactions)
        │  (fake K8s) │  What: "Do components work together?"
        ├─────────────┤
        │    Unit     │  50-100 tests (fast, logic)
        │  (pure Go)  │  What: "Does each function do what it should?"
        └─────────────┘
```

---

## Phase 1: Foundation (Week 1)

### 1.1 Project Structure

```bash
tests/
├── unit/
│   ├── kordinator/
│   │   ├── crd_health_test.go
│   │   ├── dependency_test.go
│   │   └── activation_test.go
│   ├── katalog/
│   │   ├── merge_test.go
│   │   ├── validation_test.go
│   │   └── translator_test.go
│   ├── queue/
│   │   └── workqueue_test.go
│   └── utils/
│       ├── template_test.go
│       └── helpers_test.go
│
├── integration/
│   ├── reconciler/
│   │   ├── deployment_test.go
│   │   ├── service_test.go
│   │   └── secret_test.go
│   ├── dependency/
│   │   ├── ordering_test.go
│   │   └── cycle_detection_test.go
│   ├── activation/
│   │   └── missing_crd_test.go
│   └── komposer/
│       ├── merge_test.go
│       └── sources_test.go
│
├── e2e/
│   ├── website/
│   │   └── test.sh
│   ├── dependencies/
│   │   └── test.sh
│   ├── activation/
│   │   └── test.sh
│   └── komposer/
│       └── test.sh
│
├── fixtures/
│   ├── katalogs/
│   │   ├── website.yaml
│   │   ├── dependencies.yaml
│   │   └── komposer.yaml
│   └── crds/
│       ├── website-crd.yaml
│       └── orkapp-crd.yaml
│
├── helpers/
│   ├── fake_kubeclient.go
│   ├── fake_informer.go
│   └── testutils.go
│
├── Makefile
└── README.md
```

### 1.2 Test Helpers

Create reusable test helpers first:

```go
// tests/helpers/fake_kubeclient.go
package helpers

import (
    "context"
    "k8s.io/client-go/kubernetes/fake"
    "github.com/orkspace/orkestra/pkg/kubeclient"
)

func NewFakeKubeclient() *kubeclient.Kubeclient {
    return &kubeclient.Kubeclient{
        Clientset: fake.NewSimpleClientset(),
    }
}

func NewFakeKubeclientWithObjects(objects ...runtime.Object) *kubeclient.Kubeclient {
    return &kubeclient.Kubeclient{
        Clientset: fake.NewSimpleClientset(objects...),
    }
}
```

```go
// tests/helpers/testutils.go
package helpers

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func AssertNoError(t *testing.T, err error) {
    require.NoError(t, err)
}

func AssertError(t *testing.T, err error, msg string) {
    require.Error(t, err)
    assert.Contains(t, err.Error(), msg)
}
```

---

## Phase 2: Unit Tests (Week 1-2)

### 2.1 CRD Health Tests

```go
// tests/unit/kordinator/crd_health_test.go
package kordinator_test

import (
    "fmt"
    "testing"
    "time"
    
    "github.com/orkspace/orkestra/pkg/kordinator"
    "github.com/stretchr/testify/assert"
)

func TestCRDHealth_InitialState(t *testing.T) {
    h := kordinator.NewCRDHealth("test")
    
    assert.False(t, h.IsHealthy())
    assert.False(t, h.Started())
    assert.Equal(t, int64(0), h.TotalReconciles())
    assert.Equal(t, int64(0), h.FailedReconciles())
    assert.Equal(t, float64(0), h.ErrorRate())
}

func TestCRDHealth_RecordSuccess(t *testing.T) {
    h := kordinator.NewCRDHealth("test")
    
    h.RecordSuccess()
    
    assert.True(t, h.IsHealthy())
    assert.Equal(t, int64(1), h.TotalReconciles())
    assert.Equal(t, int64(0), h.FailedReconciles())
    assert.Equal(t, int64(0), h.ConsecutiveFails())
}

func TestCRDHealth_RecordFailure(t *testing.T) {
    h := kordinator.NewCRDHealth("test")
    
    h.RecordFailure(fmt.Errorf("something went wrong"), 3)
    
    assert.False(t, h.IsHealthy())
    assert.Equal(t, int64(1), h.TotalReconciles())
    assert.Equal(t, int64(1), h.FailedReconciles())
    assert.Equal(t, int64(1), h.ConsecutiveFails())
    assert.Equal(t, "something went wrong", h.LastError())
}

func TestCRDHealth_RecordFailureExceedsThreshold(t *testing.T) {
    h := kordinator.NewCRDHealth("test")
    
    // First failure — still healthy (threshold=3)
    h.RecordFailure(fmt.Errorf("error 1"), 3)
    assert.True(t, h.IsHealthy())
    
    // Second failure — still healthy
    h.RecordFailure(fmt.Errorf("error 2"), 3)
    assert.True(t, h.IsHealthy())
    
    // Third failure — becomes degraded
    h.RecordFailure(fmt.Errorf("error 3"), 3)
    assert.False(t, h.IsHealthy())
}

func TestCRDHealth_SuccessResetsConsecutiveFails(t *testing.T) {
    h := kordinator.NewCRDHealth("test")
    
    h.RecordFailure(fmt.Errorf("error"), 3)
    h.RecordFailure(fmt.Errorf("error"), 3)
    assert.Equal(t, int64(2), h.ConsecutiveFails())
    
    h.RecordSuccess()
    assert.Equal(t, int64(0), h.ConsecutiveFails())
    assert.True(t, h.IsHealthy())
}

func TestCRDHealth_ErrorRate(t *testing.T) {
    h := kordinator.NewCRDHealth("test")
    
    h.RecordSuccess()  // 1 total, 0 failed
    assert.Equal(t, float64(0), h.ErrorRate())
    
    h.RecordFailure(fmt.Errorf("error"), 3)  // 2 total, 1 failed
    assert.Equal(t, 0.5, h.ErrorRate())
    
    h.RecordFailure(fmt.Errorf("error"), 3)  // 3 total, 2 failed
    assert.InDelta(t, 0.666, h.ErrorRate(), 0.001)
}
```

### 2.2 Dependency Graph Tests

```go
// tests/unit/kordinator/dependency_test.go
package kordinator_test

import (
    "testing"
    
    "github.com/orkspace/orkestra/pkg/katalog"
    "github.com/orkspace/orkestra/pkg/orktypes"
    "github.com/stretchr/testify/assert"
)

func TestDependencyGraph_StartupOrder(t *testing.T) {
    crds := []orktypes.CRDEntry{
        {Name: "A", DependsOn: []string{}},
        {Name: "B", DependsOn: []string{"A"}},
        {Name: "C", DependsOn: []string{"B"}},
    }
    
    graph := katalog.NewDependencyGraph(crds)
    order := graph.StartupOrder()
    
    assert.Equal(t, []string{"A", "B", "C"}, order)
}

func TestDependencyGraph_ReverseShutdownOrder(t *testing.T) {
    crds := []orktypes.CRDEntry{
        {Name: "A", DependsOn: []string{}},
        {Name: "B", DependsOn: []string{"A"}},
        {Name: "C", DependsOn: []string{"B"}},
    }
    
    graph := katalog.NewDependencyGraph(crds)
    shutdownOrder := graph.ShutdownOrder()
    
    // Should be reverse of startup order
    assert.Equal(t, []string{"C", "B", "A"}, shutdownOrder)
}

func TestDependencyGraph_DetectsCycle(t *testing.T) {
    crds := []orktypes.CRDEntry{
        {Name: "A", DependsOn: []string{"B"}},
        {Name: "B", DependsOn: []string{"A"}},
    }
    
    graph := katalog.NewDependencyGraph(crds)
    err := graph.Validate()
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cycle")
}

func TestDependencyGraph_MissingDependency(t *testing.T) {
    crds := []orktypes.CRDEntry{
        {Name: "A", DependsOn: []string{"B"}},
    }
    
    graph := katalog.NewDependencyGraph(crds)
    err := graph.Validate()
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not found")
}

func TestDependencyGraph_GetDependents(t *testing.T) {
    crds := []orktypes.CRDEntry{
        {Name: "A", DependsOn: []string{}},
        {Name: "B", DependsOn: []string{"A"}},
        {Name: "C", DependsOn: []string{"A"}},
    }
    
    graph := katalog.NewDependencyGraph(crds)
    dependents := graph.GetDependents("A")
    
    assert.ElementsMatch(t, []string{"B", "C"}, dependents)
}
```

### 2.3 Katalog Merge Tests

```go
// tests/unit/katalog/merge_test.go
package katalog_test

import (
    "testing"
    
    "github.com/orkspace/orkestra/pkg/katalog"
    "github.com/orkspace/orkestra/pkg/orktypes"
    "github.com/stretchr/testify/assert"
)

func TestMerge_SingleSource(t *testing.T) {
    source1 := []orktypes.CRDEntry{
        {Name: "website", Workers: 2},
    }
    
    merger := katalog.NewMerger()
    result := merger.Merge(source1, nil)
    
    assert.Len(t, result, 1)
    assert.Equal(t, "website", result[0].Name)
    assert.Equal(t, 2, result[0].Workers)
}

func TestMerge_MultipleSources(t *testing.T) {
    source1 := []orktypes.CRDEntry{
        {Name: "website", Workers: 2},
    }
    source2 := []orktypes.CRDEntry{
        {Name: "database", Workers: 3},
    }
    
    merger := katalog.NewMerger()
    result := merger.Merge(source1, source2)
    
    assert.Len(t, result, 2)
}

func TestMerge_DuplicateNameError(t *testing.T) {
    source1 := []orktypes.CRDEntry{
        {Name: "website", Workers: 2},
    }
    source2 := []orktypes.CRDEntry{
        {Name: "website", Workers: 4},
    }
    
    merger := katalog.NewMerger()
    err := merger.MergeWithError(source1, source2)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "duplicate")
}

func TestMerge_InlineOverride(t *testing.T) {
    source := []orktypes.CRDEntry{
        {Name: "website", Workers: 2},
    }
    override := []orktypes.CRDEntry{
        {Name: "website", Workers: 4},
    }
    
    merger := katalog.NewMerger()
    result := merger.MergeWithOverride(source, override)
    
    assert.Len(t, result, 1)
    assert.Equal(t, 4, result[0].Workers)
}
```

---

## Phase 3: Integration Tests (Week 2-3)

### 3.1 Reconciler Tests

```go
// tests/integration/reconciler/deployment_test.go
package reconciler_test

import (
    "context"
    "testing"
    
    "github.com/orkspace/orkestra/pkg/kubeclient"
    "github.com/orkspace/orkestra/pkg/reconciler"
    "github.com/orkspace/orkestra/pkg/orktypes"
    "github.com/orkspace/orkestra/tests/helpers"
    "github.com/stretchr/testify/assert"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
                    {
                        Name:     "test-deployment",
                        Image:    "nginx:latest",
                        Replicas: "3",
                        Port:     "80",
                    },
                },
            },
        },
    }
    
    // Create reconciler
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

func TestGenericReconciler_UpdatesDeploymentOnDrift(t *testing.T) {
    fakeKube := helpers.NewFakeKubeclientWithObjects(
        // Pre-existing deployment with wrong image
        &appsv1.Deployment{
            ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
            Spec: appsv1.DeploymentSpec{
                Template: corev1.PodTemplateSpec{
                    Spec: corev1.PodSpec{
                        Containers: []corev1.Container{{Image: "old-image"}},
                    },
                },
            },
        },
    )
    
    crd := &orktypes.CRDEntry{
        Name: "website",
        ReconcilerConfig: orktypes.ReconcilerConfig{
            OnReconcile: &orktypes.HookTemplates{
                Deployments: []orktypes.DeploymentTemplateSource{
                    {Name: "test-deployment", Image: "nginx:latest", Replicas: "3", Reconcile: true},
                },
            },
        },
    }
    
    reconciler := helpers.NewTestReconciler(crd, fakeKube)
    err := reconciler.Reconcile(context.Background(), "default/test-site")
    
    assert.NoError(t, err)
    
    deployment, _ := fakeKube.Clientset.AppsV1().Deployments("default").Get(
        context.Background(), "test-deployment", metav1.GetOptions{})
    assert.Equal(t, "nginx:latest", deployment.Spec.Template.Spec.Containers[0].Image)
}
```

### 3.2 Activation Tests

```go
// tests/integration/activation/missing_crd_test.go
package activation_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/orkspace/orkestra/pkg/kordinator"
    "github.com/orkspace/orkestra/tests/helpers"
    "github.com/stretchr/testify/assert"
)

func TestActivation_MissingCRDAppearsLater(t *testing.T) {
    // Setup: CRD missing
    fakeKube := helpers.NewFakeKubeclient()
    katalog := helpers.NewTestKatalogWithMissingCRD("missing-crd")
    
    kordinator := kordinator.NewDependencyKordinator(fakeKube, katalog)
    
    // Start controller (CRD missing, should not start workers)
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go kordinator.RunOrDie(ctx)
    time.Sleep(2 * time.Second)
    
    // Verify workers not started
    assert.False(t, kordinator.IsStarted("missing-crd"))
    
    // Simulate CRD appearing
    helpers.InstallCRD(fakeKube, "missing-crd")
    
    // Wait for retry loop to detect and activate
    time.Sleep(5 * time.Second)
    
    // Verify workers started
    assert.True(t, kordinator.IsStarted("missing-crd"))
}
```

### 3.3 Komposer Tests

```go
// tests/integration/komposer/merge_test.go
package komposer_test

import (
    "testing"
    
    "github.com/orkspace/orkestra/pkg/katalog"
    "github.com/stretchr/testify/assert"
)

func TestKomposer_MergesFileSources(t *testing.T) {
    komposer := &katalog.Komposer{
        Sources: katalog.KomposerSources{
            Files: []string{
                "./fixtures/katalogs/website.yaml",
                "./fixtures/katalogs/platform.yaml",
            },
        },
    }
    
    result, err := katalog.MergeKomposer(komposer)
    assert.NoError(t, err)
    assert.Len(t, result.CRDs, 2)
}

func TestKomposer_MergesHelmSources(t *testing.T) {
    komposer := &katalog.Komposer{
        Sources: katalog.KomposerSources{
            Helm: []katalog.HelmSource{
                {
                    Repo:    "./fixtures/charts",
                    Chart:   "test-chart",
                    Version: "0.1.0",
                },
            },
        },
    }
    
    result, err := katalog.MergeKomposer(komposer)
    assert.NoError(t, err)
    assert.NotEmpty(t, result.CRDs)
}
```

---

## Phase 4: E2E Tests (Week 3-4)

### 4.1 E2E Test Framework

Create a reusable test harness:

```bash
#!/bin/bash
# tests/e2e/run.sh
set -e

TEST_NAME=$1
TIMEOUT=${2:-60}

echo "=== Running E2E Test: $TEST_NAME ==="

# Setup
kind create cluster --name orkestra-test
trap "kind delete cluster --name orkestra-test" EXIT

# Build Orkestra
go build -o /tmp/ork ./cmd/ork

# Run test
./tests/e2e/$TEST_NAME/test.sh

echo "=== E2E Test Passed ==="
```

### 4.2 Website E2E Test

```bash
#!/bin/bash
# tests/e2e/website/test.sh
set -e

echo "Testing: Website CRD → Deployment + Service"

# Install CRD
kubectl apply -f examples/website/website-crd.yaml

# Start Orkestra
/tmp/ork run --katalog examples/website/website-katalog.yaml &
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

# Verify Health Endpoint
curl -s localhost:8080/katalog/website/health | jq -e '.healthy == true'
HEALTH_OK=$?

# Verify Metrics
curl -s localhost:8080/metrics | grep controller_reconcile_total
METRICS_EXIST=$?

# Cleanup
kill $ORK_PID

if [ $DEPLOYMENT_EXISTS -eq 0 ] && [ $SERVICE_EXISTS -eq 0 ] && [ $HEALTH_OK -eq 0 ] && [ $METRICS_EXIST -eq 0 ]; then
    echo "✅ Website E2E test passed"
    exit 0
else
    echo "❌ Website E2E test failed"
    exit 1
fi
```

### 4.3 Activation E2E Test

```bash
#!/bin/bash
# tests/e2e/activation/test.sh
set -e

echo "Testing: Missing CRD appears after startup"

# Start Orkestra WITHOUT installing CRD
/tmp/ork run --katalog tests/fixtures/katalogs/missing-crd.yaml &
ORK_PID=$!
sleep 5

# Verify health shows degraded
curl -s localhost:8080/katalog/missing-crd/health | jq -e '.healthy == false'
HEALTH_DEGRADED=$?

# Install CRD
kubectl apply -f tests/fixtures/crds/missing-crd.yaml
sleep 10

# Verify health becomes healthy
curl -s localhost:8080/katalog/missing-crd/health | jq -e '.healthy == true'
HEALTH_HEALTHY=$?

# Cleanup
kill $ORK_PID
kubectl delete -f tests/fixtures/crds/missing-crd.yaml

if [ $HEALTH_DEGRADED -eq 0 ] && [ $HEALTH_HEALTHY -eq 0 ]; then
    echo "✅ Activation E2E test passed"
    exit 0
else
    echo "❌ Activation E2E test failed"
    exit 1
fi
```

### 4.4 Dependencies E2E Test

```bash
#!/bin/bash
# tests/e2e/dependencies/test.sh
set -e

echo "Testing: Dependency ordering"

# Install CRDs (A and B, where B depends on A)
kubectl apply -f tests/fixtures/crds/crd-a.yaml
kubectl apply -f tests/fixtures/crds/crd-b.yaml

# Start Orkestra
/tmp/ork run --katalog tests/fixtures/katalogs/dependencies.yaml &
ORK_PID=$!
sleep 5

# Check startup order (B should start only after A)
curl -s localhost:8080/katalog/crd-a/health | jq -e '.started == true'
A_STARTED=$?
curl -s localhost:8080/katalog/crd-b/health | jq -e '.started == true'
B_STARTED=$?

# Cleanup
kill $ORK_PID
kubectl delete -f tests/fixtures/crds/crd-a.yaml
kubectl delete -f tests/fixtures/crds/crd-b.yaml

if [ $A_STARTED -eq 0 ] && [ $B_STARTED -eq 0 ]; then
    echo "✅ Dependencies E2E test passed"
    exit 0
else
    echo "❌ Dependencies E2E test failed"
    exit 1
fi
```

---

## Phase 5: Test Automation (Week 4)

### 5.1 Makefile

```makefile
# Makefile
.PHONY: test test-unit test-integration test-e2e test-all

test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	go test ./tests/unit/... -v -short

test-integration:
	@echo "Running integration tests..."
	go test ./tests/integration/... -v -tags=integration

test-e2e:
	@echo "Running E2E tests..."
	./tests/e2e/run.sh website
	./tests/e2e/run.sh activation
	./tests/e2e/run.sh dependencies

test-all: test-unit test-integration test-e2e

test-coverage:
	@echo "Generating coverage report..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
```

### 5.2 GitHub Actions

```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  unit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Unit Tests
        run: make test-unit

  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Integration Tests
        run: make test-integration

  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Setup kind
        uses: helm/kind-action@v1
      - name: E2E Tests
        run: make test-e2e
```

---

## Phase 6: Test Fixtures

### 6.1 Sample Katalogs

```yaml
# tests/fixtures/katalogs/website.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: website-katalog
spec:
  crds:
    - name: website
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites
      operatorBox:
        default: true
        onCreate:
          deployments:
            - image: "nginx:latest"
              replicas: "3"
              reconcile: true
```

### 6.2 Sample CRDs

```yaml
# tests/fixtures/crds/website-crd.yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: websites.demo.orkestra.io
spec:
  group: demo.orkestra.io
  names:
    kind: Website
    plural: websites
    singular: website
  scope: Namespaced
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                image:
                  type: string
                replicas:
                  type: integer
```

---

## What Tests You Need for v1

| Priority | Test | Type | Purpose |
|----------|------|------|---------|
| **P0** | Website example works | E2E | The "hello world" must work |
| **P0** | Missing CRD activation | Integration | Core value: CRDs appear later |
| **P0** | Dependency ordering | Unit | Core feature |
| **P1** | Health tracking | Unit | Correct health transitions |
| **P1** | Katalog merge | Unit | Komposer correctness |
| **P1** | Reconciler creates resources | Integration | Templates work |
| **P2** | Drift correction | Integration | reconcile:true works |
| **P2** | Graceful shutdown | E2E | No panic on exit |
| **P3** | Metrics emission | Integration | Prometheus works |
| **P3** | Secret copy | Integration | FromSecret pattern |

---

## Success Criteria for v1

| Metric | Target |
|--------|--------|
| Unit test coverage | 70% on core packages |
| Integration tests | 10 passing |
| E2E tests | 3 passing (website, activation, dependencies) |
| CI time | < 5 minutes |
| E2E time | < 2 minutes per test |

---

## The Bottom Line

Start with the Website E2E test. That's your anchor. Then add unit tests for the logic you're most nervous about. Then add integration tests for the flows that broke during development.

**You don't need 100% coverage. You need confidence.** This plan gives you that. 🎼