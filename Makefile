.PHONY: test test-unit test-race test-integration test-e2e test-all test-coverage test-coverage-text vet certs

# ── Primary targets ───────────────────────────────────────────────────────────

# Default: vet + unit tests. Fast, no external dependencies.
test: vet test-unit

# ── Unit tests ────────────────────────────────────────────────────────────────
# Runs all pure-logic unit tests co-located with the packages they test.
#
# Includes:
#   ./pkg/health/...       — admission evaluation, stats, conversion logic
#   ./pkg/types/...        — admission, condition, and conversion types
#   ./pkg/metrics/...      — metric helper smoke tests
#   ./pkg/queue/...        — workqueue and registry tests
#   ./pkg/merger/...       — registry URL construction, source loading
#   ./pkg/katalog/...      — dependency graph, topological sort, cycle detection
#   ./pkg/kontroller/...   — CRD health lifecycle
#   ./pkg/reconciler/...   — validation rules, mutation patch building, namespace guard
#
# Excludes pkg/inspect/test (requires a live Kubernetes cluster).
#
#   -short  skips tests that guard slow/external work with t.Skip(testing.Short())
#   -count=1 disables result caching — tests always run fresh
test-unit:
	@echo "Running unit tests..."
	go test \
		./pkg/health/... \
		./pkg/types/... \
		./pkg/metrics/... \
		./pkg/queue/... \
		./pkg/merger/... \
		./pkg/katalog/... \
		./pkg/kontroller/... \
		./pkg/reconciler/... \
		-v -short -count=1

# ── Race detector ─────────────────────────────────────────────────────────────
# Same as test-unit but with Go's race detector enabled.
# Run this before every pull request. Catches concurrent map access,
# goroutine data races, and ring-buffer index races in the admission stats
# ring buffer, workqueue, and informer factory.
test-race:
	@echo "Running unit tests with race detector..."
	go test \
		./pkg/health/... \
		./pkg/types/... \
		./pkg/metrics/... \
		./pkg/queue/... \
		./pkg/merger/... \
		./pkg/katalog/... \
		./pkg/kontroller/... \
		./pkg/reconciler/... \
		-short -race -count=1

# ── Integration tests ─────────────────────────────────────────────────────────
# Requires a reachable Kubernetes cluster (from KUBECONFIG).
# Tests are guarded by the `integration` build tag so they never run
# during `make test-unit`.
test-integration:
	@echo "Running integration tests..."
	go test ./tests/integration/... -v -tags=integration -count=1

# ── End-to-end tests ──────────────────────────────────────────────────────────
# Full cluster lifecycle: deploys Orkestra, applies CRDs and CRs, checks
# health endpoints, verifies reconciliation, cleans up.
test-e2e:
	@echo "Running E2E tests..."
	./tests/e2e/run.sh website
	./tests/e2e/run.sh activation
	./tests/e2e/run.sh dependencies

# ── Full suite ────────────────────────────────────────────────────────────────
test-all: test-unit test-integration test-e2e

# ── Coverage ──────────────────────────────────────────────────────────────────
# Generates an HTML coverage report in coverage.html.
# Open it with: open coverage.html  (macOS) / xdg-open coverage.html (Linux)
#
# Uses the same package set as test-unit — excludes cluster-dependent tests.
test-coverage:
	@echo "Generating coverage report..."
	go test \
		./pkg/health/... \
		./pkg/types/... \
		./pkg/metrics/... \
		./pkg/queue/... \
		./pkg/merger/... \
		./pkg/katalog/... \
		./pkg/kontroller/... \
		./pkg/reconciler/... \
		-coverprofile=coverage.out -covermode=atomic -count=1
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

# Text summary of per-function coverage — useful in CI output.
test-coverage-text:
	@echo "Coverage summary..."
	go test \
		./pkg/health/... \
		./pkg/types/... \
		./pkg/metrics/... \
		./pkg/queue/... \
		./pkg/merger/... \
		./pkg/katalog/... \
		./pkg/kontroller/... \
		./pkg/reconciler/... \
		-coverprofile=coverage.out -covermode=atomic -count=1
	@go tool cover -func=coverage.out | tail -5

# ── Vet ───────────────────────────────────────────────────────────────────────
vet:
	@echo "Running go vet..."
	go vet ./...

# ── Certificates ─────────────────────────────────────────────────────────────
certs:
	@echo "Generating self-signed certificates for Orkestra..."
	@bash scripts/self-signed-certificates.sh
	@echo "Certificates generated in orkestrs-certs/"
