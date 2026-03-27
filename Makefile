# Makefile
.PHONY: test test-unit test-integration test-e2e test-all certs

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


certs:
	@echo "Generating self-signed certificates for Orkestra..."
	@bash scripts/self-signed-certificates.sh
	@echo "Certificates generated in orkestrs-certs/"
