# ── Typed Orkestra Operator ───────────────────────────────────────────────────
BINARY_NAME ?= ork
DEV_OUTPUT_DIR  ?= $(HOME)/.orkestra/bin
PROD_OUTPUT_DIR ?= $(HOME)/.orkestra/bin/runtime
KATALOG     ?= katalog.yaml

IMAGE_REPO  ?= myorg/web-app-reconciler
IMAGE_TAG   ?= latest
IMAGE       ?= $(IMAGE_REPO):$(IMAGE_TAG)

GOOS        ?= linux
GOARCH      ?= amd64
CGO_ENABLED  = 0

ORK_LDFLAGS := -X github.com/orkspace/orkestra/pkg/version.Version=$(GIT_VERSION) \
               -X github.com/orkspace/orkestra/pkg/version.Commit=$(GIT_COMMIT) \
               -X github.com/orkspace/orkestra/pkg/version.Date=$(GIT_DATE)

.PHONY: registry
registry:
	@[ -f go.mod.txt ] && mv go.mod.txt go.mod || true
	@[ -f go.sum.txt ] && mv go.sum.txt go.sum || true
	@find . -name "*.go" | xargs grep -l "^//go:build ignore$$" 2>/dev/null | while read f; do tail -n +3 "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f"; done || true
	ork generate registry --file $(KATALOG)

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

.PHONY: validate
validate:
	$(DEV_OUTPUT_DIR)/$(BINARY_NAME) validate -f $(KATALOG)

.PHONY: e2e
e2e:
	$(DEV_OUTPUT_DIR)/$(BINARY_NAME) e2e

.PHONY: docker
docker:
	@cp $(PROD_OUTPUT_DIR)/$(BINARY_NAME) ./$(BINARY_NAME)
	docker build -t $(IMAGE) .
	@rm -f ./$(BINARY_NAME)
