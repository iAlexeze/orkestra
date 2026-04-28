# ── Typed Orkestra Operator — Hooks ───────────────────────────────────────────
BINARY_NAME ?= ork
OUTPUT_DIR  ?= $(HOME)/.orkestra/bin
KATALOG     ?= katalog.yaml

IMAGE_REPO  ?= myorg/my-operator
IMAGE_TAG   ?= latest
IMAGE       ?= $(IMAGE_REPO):$(IMAGE_TAG)

GOOS        ?= linux
GOARCH      ?= amd64
CGO_ENABLED  = 0

# ── registry ──────────────────────────────────────────────────────────────────
# Generates pkg/runtime/zz_generated_runtime_registry.go.
# Re-run whenever you change apiTypes fields in katalog.yaml.
.PHONY: registry
registry:
	ork generate registry --katalog $(KATALOG)

# ── build ─────────────────────────────────────────────────────────────────────
.PHONY: build
build:
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -o $(OUTPUT_DIR)/$(BINARY_NAME) ./cmd/orkestra
	@echo "✅ $(OUTPUT_DIR)/$(BINARY_NAME)"

# ── validate ──────────────────────────────────────────────────────────────────
.PHONY: validate
validate:
	$(OUTPUT_DIR)/$(BINARY_NAME) validate -k $(KATALOG)

# ── docker / push / release ───────────────────────────────────────────────────
.PHONY: docker
docker: build
	@cp $(OUTPUT_DIR)/$(BINARY_NAME) ./$(BINARY_NAME)
	docker build -t $(IMAGE) .
	@rm -f ./$(BINARY_NAME)
	@echo "✅ $(IMAGE)"

.PHONY: push
push:
	docker push $(IMAGE)

.PHONY: release
release: docker push

# ── clean ─────────────────────────────────────────────────────────────────────
# Removes the local typed binary only. The standard ork CLI is unaffected.
.PHONY: clean
clean:
	@rm -f $(OUTPUT_DIR)/$(BINARY_NAME)
	@echo "✅ removed $(OUTPUT_DIR)/$(BINARY_NAME)"

# ── help ──────────────────────────────────────────────────────────────────────
.PHONY: help
help:
	@echo "  registry   generate pkg/runtime/zz_generated_runtime_registry.go"
	@echo "  build      compile the typed operator binary"
	@echo "  validate   run katalog validation with your binary"
	@echo "  release    build + push  IMAGE=repo/name:tag"
	@echo "  clean      remove local binary (standard ork unaffected)"