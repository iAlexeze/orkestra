# ── ResourceProbe — pkg/resources integration fixture ──────────────────────────
BINARY_NAME ?= ork
DEV_OUTPUT_DIR  ?= $(HOME)/.orkestra/bin
PROD_OUTPUT_DIR ?= $(HOME)/.orkestra/bin/runtime
KATALOG     ?= katalog.yaml

IMAGE_REPO  ?= ghcr.io/orkspace/resource-probe
IMAGE_TAG   ?= latest
IMAGE       ?= $(IMAGE_REPO):$(IMAGE_TAG)

GOOS        ?= linux
GOARCH      ?= amd64
CGO_ENABLED  = 0

ORK_LDFLAGS := -X github.com/orkspace/orkestra/pkg/version.Version=$(GIT_VERSION) \
               -X github.com/orkspace/orkestra/pkg/version.Commit=$(GIT_COMMIT) \
               -X github.com/orkspace/orkestra/pkg/version.Date=$(GIT_DATE)

# ── registry ──────────────────────────────────────────────────────────────────
.PHONY: registry
registry:
	@[ -f go.mod.txt ] && mv go.mod.txt go.mod || true
	@[ -f go.sum.txt ] && mv go.sum.txt go.sum || true
	@find . -name "*.go" | xargs grep -l "^//go:build ignore$$" 2>/dev/null | while read f; do tail -n +3 "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f"; done || true
	ork generate registry --file $(KATALOG)

# ── build (development) ───────────────────────────────────────────────────────
.PHONY: build
build:
	@[ -f go.mod.txt ] && mv go.mod.txt go.mod || true
	@[ -f go.sum.txt ] && mv go.sum.txt go.sum || true
	@find . -name "*.go" | xargs grep -l "^//go:build ignore$$" 2>/dev/null | while read f; do tail -n +3 "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f"; done || true
	@mkdir -p $(DEV_OUTPUT_DIR)
	go mod tidy
	gofmt -w .
	go build \
		-ldflags "$(ORK_LDFLAGS)" \
		-o $(DEV_OUTPUT_DIR)/$(BINARY_NAME) ./cmd/orkestra
	@echo "✅ Development build: $(DEV_OUTPUT_DIR)/$(BINARY_NAME)"

# ── build-runtime (production) ────────────────────────────────────────────────
.PHONY: build-runtime
build-runtime:
	@[ -f go.mod.txt ] && mv go.mod.txt go.mod || true
	@[ -f go.sum.txt ] && mv go.sum.txt go.sum || true
	@find . -name "*.go" | xargs grep -l "^//go:build ignore$$" 2>/dev/null | while read f; do tail -n +3 "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f"; done || true
	@mkdir -p $(PROD_OUTPUT_DIR)
	go mod tidy
	gofmt -w .
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-tags "runtime" \
		-ldflags "$(ORK_LDFLAGS)" \
		-o $(PROD_OUTPUT_DIR)/$(BINARY_NAME) ./cmd/orkestra
	@echo "✅ Production runtime build: $(PROD_OUTPUT_DIR)/$(BINARY_NAME)"

# ── validate ──────────────────────────────────────────────────────────────────
.PHONY: validate
validate:
	$(DEV_OUTPUT_DIR)/$(BINARY_NAME) validate -f $(KATALOG)

# ── simulate ──────────────────────────────────────────────────────────────────
.PHONY: simulate
simulate:
	$(DEV_OUTPUT_DIR)/$(BINARY_NAME) simulate

# ── e2e ───────────────────────────────────────────────────────────────────────
.PHONY: e2e
e2e:
	$(DEV_OUTPUT_DIR)/$(BINARY_NAME) e2e

# ── docker / push / release ───────────────────────────────────────────────────
.PHONY: docker
docker: build-runtime
	@cp $(PROD_OUTPUT_DIR)/$(BINARY_NAME) ./$(BINARY_NAME)
	docker build -t $(IMAGE) .
	@rm -f ./$(BINARY_NAME)
	@echo "✅ Docker image built: $(IMAGE)"

.PHONY: push
push:
	docker push $(IMAGE)

.PHONY: release
release: docker push

# ── clean ─────────────────────────────────────────────────────────────────────
.PHONY: clean
clean:
	@rm -f $(DEV_OUTPUT_DIR)/$(BINARY_NAME)
	@rm -rf $(PROD_OUTPUT_DIR)
	@echo "✅ Removed all local builds"

# ── help ──────────────────────────────────────────────────────────────────────
.PHONY: help
help:
	@echo "  registry       generate type registry from Katalog"
	@echo "  build          compile full development CLI (all commands)"
	@echo "  build-runtime  compile production binary (only 'ork run')"
	@echo "  validate       run katalog validation (requires make build)"
	@echo "  simulate       run simulation tests (requires make build)"
	@echo "  e2e            run end-to-end tests"
	@echo "  docker         build-runtime + build Docker image"
	@echo "  push           push Docker image to registry"
	@echo "  release        docker + push"
	@echo "  clean          remove all local builds"
