.PHONY: build orkcc clean test test-unit test-race test-integration test-e2e test-all test-coverage test-coverage-text vet certs docs docs-build docs-serve site site-sync site-build site-start hugo-install generate-notes

# ── Configuration ────────────────────────────────────────────────────────────
ORKESTRA_DIR := .
CONTROL_CENTER_DIR := ./cmd/controlcenter
OUTPUT_DIR := $(HOME)/.orkestra/bin
BUILD_TAGS ?=

# Version stamping — reads from git; matches what CI/CD injects via ldflags.
GIT_VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
GIT_DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

ORK_LDFLAGS := -X github.com/orkspace/orkestra/pkg/version.Version=$(GIT_VERSION) \
               -X github.com/orkspace/orkestra/pkg/version.Commit=$(GIT_COMMIT) \
               -X github.com/orkspace/orkestra/pkg/version.Date=$(GIT_DATE)

CC_LDFLAGS  := -X github.com/orkspace/orkestra-cc/version.Version=$(GIT_VERSION) \
               -X github.com/orkspace/orkestra-cc/version.Commit=$(GIT_COMMIT) \
               -X github.com/orkspace/orkestra-cc/version.Date=$(GIT_DATE)

# ── Local build ───────────────────────────────────────────────────────────
generate-notes:
	@echo "Generating note catalog..."
	go run ./hack/generate-notes
	@echo "✅ Note catalog generated at pkg/note/catalog_generated.go"

ork:
	@echo "Building Orkestra..."
	@mkdir -p $(OUTPUT_DIR)
	cd $(ORKESTRA_DIR) && gofmt -w .
	cd $(ORKESTRA_DIR) && go build -ldflags "$(ORK_LDFLAGS)" -o $(OUTPUT_DIR)/ork ./cmd/orkestra
	@echo "✅ Orkestra built successfully"

orkcc:
	@echo "Building Orkestra Control Center..."
	@mkdir -p $(OUTPUT_DIR)
	cd $(CONTROL_CENTER_DIR) && gofmt -w .
	cd $(CONTROL_CENTER_DIR) && go build -ldflags "$(CC_LDFLAGS)" -o $(OUTPUT_DIR)/orkcc .
	@echo "✅ Orkestra Control Center built successfully"

build: ork orkcc
	@echo "✅ Both Orkestra and Control Center built successfully"

clean:
	@echo "Cleaning binaries..."
	rm -f $(OUTPUT_DIR)/ork
	rm -f $(OUTPUT_DIR)/orkcc
	@echo "✅ Clean complete"

# ── Install to system PATH (requires sudo) ───────────────────────────────────
install: build
	@echo "Installing to /usr/local/bin..."
	sudo cp $(OUTPUT_DIR)/ork /usr/local/bin/
	sudo cp $(OUTPUT_DIR)/orkcc /usr/local/bin/
	@echo "✅ Installed successfully"

# ── Uninstall from system ────────────────────────────────────────────────────
uninstall:
	@echo "Removing from /usr/local/bin..."
	sudo rm -f /usr/local/bin/ork
	sudo rm -f /usr/local/bin/orkcc
	@echo "✅ Uninstalled successfully"

# ── Add to PATH helper ───────────────────────────────────────────────────────
path-help:
	@echo "Add this to your ~/.bashrc or ~/.zshrc:"
	@echo "export PATH=\$$PATH:$(OUTPUT_DIR)"

# ── Docker Image Configuration ────────────────────────────────────────────────

# Default to short git commit, fallback to timestamp if git fails
# Fail explicitly if not in a Git repo
GIT_COMMIT := $(shell git rev-parse --short HEAD)
ifeq ($(GIT_COMMIT),)
  $(error "Not a Git repository. Set ORK_IMAGE and ORK_CC_IMAGE manually.")
endif

ORK_IMAGE ?= ghcr.io/orkspace/orkestra:$(GIT_COMMIT)
ORK_CC_IMAGE ?= ghcr.io/orkspace/orkestra-cc:$(GIT_COMMIT)

# Target architectures
ORK_AMD64_TARGET="ork-amd64"
ORK_ARM64_TARGET="ork-arm64"
ORK_CC_AMD64_TARGET="orkcc-amd64"
ORK_CC_ARM64_TARGET="orkcc-arm64"



# Path where the built binary lives
BIN := $(OUTPUT_DIR)/ork

# ── Linux Build Targets (for Docker) ──────────────────────────────────────────

ork-linux:
	@echo "Building Orkestra (Linux amd64)..."
	@mkdir -p $(OUTPUT_DIR)
	gofmt -w . && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-tags "$(BUILD_TAGS)" \
		-ldflags "$(ORK_LDFLAGS)" \
		-o $(OUTPUT_DIR)/ork ./cmd/orkestra
	@echo "✅ Linux Orkestra binary built: $(OUTPUT_DIR)/ork"

orkcc-linux:
	@echo "Building Orkestra Control Center (Linux amd64)..."
	@mkdir -p $(OUTPUT_DIR)
	cd $(CONTROL_CENTER_DIR) && gofmt -w . && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(OUTPUT_DIR)/orkcc .
	@echo "✅ Linux Control Center binary built: $(OUTPUT_DIR)/orkcc"

# ── Docker Build ──────────────────────────────────────────────────────────────

docker:
	$(MAKE) ork-linux BUILD_TAGS=runtime
	@echo "Building Docker image: $(ORK_IMAGE)"
	@cp $(OUTPUT_DIR)/ork ./$(ORK_AMD64_TARGET)
	
	docker build -t $(ORK_IMAGE) .
	@rm -f ./$(ORK_AMD64_TARGET)
	@echo "✔ Docker image built: $(ORK_IMAGE)"

docker-cc: orkcc-linux
	@echo "Building Docker image: $(ORK_CC_IMAGE)"
	cd $(CONTROL_CENTER_DIR) && cp $(OUTPUT_DIR)/orkcc ./$(ORK_CC_AMD64_TARGET)
	cd $(CONTROL_CENTER_DIR) && docker build -t $(ORK_CC_IMAGE) . && rm -rf ./$(ORK_CC_AMD64_TARGET)
	@echo "✔ Docker image built: $(ORK_CC_IMAGE)"

# ── Docker Push ───────────────────────────────────────────────────────────────

docker-push:
	@echo "Pushing Docker image: $(ORK_IMAGE)"
	docker push $(ORK_IMAGE)
	@echo "✔ Docker image pushed: $(ORK_IMAGE)"
	@echo "Pushing Docker image: $(ORK_CC_IMAGE)"
	docker push $(ORK_CC_IMAGE)
	@echo "✔ Docker image pushed: $(ORK_CC_IMAGE)"

# ── Docker Release (build + push) ─────────────────────────────────────────────

docker-release: docker docker-cc docker-push
	@echo "✔ Docker release complete: $(ORK_IMAGE)"

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
#   ./pkg/kordinator/...   — CRD health lifecycle
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
		./pkg/kordinator/... \
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
		./pkg/kordinator/... \
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
		./pkg/kordinator/... \
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
		./pkg/kordinator/... \
		./pkg/reconciler/... \
		-coverprofile=coverage.out -covermode=atomic -count=1
	@go tool cover -func=coverage.out | tail -5

# ── Docs (Hugo) ───────────────────────────────────────────────────────────────
# The Hugo site lives in website/ and renders the docs/ directory.
# Requires the hugo binary — install with: brew install hugo  or  snap install hugo

DOCS_PORT ?= 8191

docs:
	@if [ -z "$(HUGO)" ]; then echo "Hugo not found. Run: make hugo-install"; exit 1; fi
	@echo "Starting Hugo docs server at http://localhost:$(DOCS_PORT) ..."
	hugo server --source website --port $(DOCS_PORT) --bind 0.0.0.0 --disableFastRender --logLevel warn
	@echo "✅ Docs server stopped"

docs-build:
	@if [ -z "$(HUGO)" ]; then echo "Hugo not found. Run: make hugo-install"; exit 1; fi
	@echo "Building Hugo static site..."
	hugo --source website --minify
	@echo "✅ Hugo site built to website/public/"

docs-serve:
	@if [ -z "$(HUGO)" ]; then echo "Hugo not found. Run: make hugo-install"; exit 1; fi
	@echo "Serving production Hugo build on port $(DOCS_PORT)..."
	hugo server --source website --port $(DOCS_PORT) --bind 0.0.0.0 --renderStaticToDisk

# ── Hugo site (orkestra-site/) ────────────────────────────────────────────────
# The redesigned marketing + docs site.
# Requires hugo >= 0.120: brew install hugo  or  snap install hugo --channel=extended

SITE_DIR  := ./orkestra-site
SITE_PORT ?= 8565
HUGO      := $(shell which hugo 2>/dev/null)

# Install Hugo extended if not present (Linux/macOS)
.PHONY: hugo-install
hugo-install:
	@if [ -n "$(HUGO)" ]; then echo "Hugo already installed: $(HUGO)"; exit 0; fi; \
	OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	ARCH=$$(uname -m); \
	if [ "$$ARCH" = "x86_64" ]; then ARCH="amd64"; elif [ "$$ARCH" = "aarch64" ] || [ "$$ARCH" = "arm64" ]; then ARCH="arm64"; fi; \
	VER="0.147.0"; \
	if [ "$$OS" = "darwin" ]; then \
	  brew install hugo && exit 0; \
	fi; \
	URL="https://github.com/gohugoio/hugo/releases/download/v$${VER}/hugo_extended_$${VER}_$${OS}-$${ARCH}.tar.gz"; \
	echo "Installing Hugo v$${VER} from $$URL ..."; \
	curl -sSL "$$URL" | tar -xz -C /tmp hugo && sudo mv /tmp/hugo /usr/local/bin/hugo; \
	echo "✅ Hugo installed: $$(hugo version)"

site-sync:
	@echo "Syncing docs/ → orkestra-site/content/docs/ ..."
	@bash $(SITE_DIR)/scripts/sync-docs.sh
	@echo "✅ Docs synced"

site: site-sync
	@if [ -z "$(HUGO)" ]; then echo "Hugo not found. Run: make hugo-install"; exit 1; fi
	@echo "Starting Hugo site at http://localhost:$(SITE_PORT) ..."
	hugo server --source $(SITE_DIR) --port $(SITE_PORT) --bind 0.0.0.0 --disableFastRender

site-start: site

site-build: site-sync
	@if [ -z "$(HUGO)" ]; then echo "Hugo not found. Run: make hugo-install"; exit 1; fi
	@echo "Building Hugo site..."
	hugo --source $(SITE_DIR) --minify
	@echo "✅ Site built to orkestra-site/public/"

# ── Vet ───────────────────────────────────────────────────────────────────────
vet:
	@echo "Running go vet..."
	go vet ./...
