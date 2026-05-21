# ── Typed Orkestra Operator — Hooks ───────────────────────────────────────────
BINARY_NAME ?= ork
OUTPUT_DIR  ?= $(HOME)/.orkestra/bin
KATALOG     ?= katalog.yaml

IMAGE_REPO  ?= myorg/my-operator
IMAGE_TAG   ?= latest
IMAGE       ?= $(IMAGE_REPO):$(IMAGE_TAG)
BUILD_TAGS	?=

GOOS        ?= linux
GOARCH      ?= amd64
CGO_ENABLED  = 0

ORK_LDFLAGS := -X github.com/orkspace/orkestra/pkg/version.Version=$(GIT_VERSION) \
               -X github.com/orkspace/orkestra/pkg/version.Commit=$(GIT_COMMIT) \
               -X github.com/orkspace/orkestra/pkg/version.Date=$(GIT_DATE)

# ── registry ──────────────────────────────────────────────────────────────────
# Generates pkg/typeregistry/zz_generated_typeregistry.go.
# Re-run whenever you change apiTypes fields in katalog.yaml.
.PHONY: registry
registry:
	ork generate registry --file $(KATALOG)

# ── build ─────────────────────────────────────────────────────────────────────
.PHONY: build
build:
	@mkdir -p $(OUTPUT_DIR)
	go mod tidy
	gofmt -w . && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-tags "$(BUILD_TAGS)" \
		-ldflags "$(ORK_LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME) ./cmd/orkestra
	@echo "✅ $(OUTPUT_DIR)/$(BINARY_NAME)"

# ── validate ──────────────────────────────────────────────────────────────────
.PHONY: validate
validate:
	$(OUTPUT_DIR)/$(BINARY_NAME) validate -f $(KATALOG)

# ── docker / push / release ───────────────────────────────────────────────────
.PHONY: docker
docker:
	$(MAKE) build BUILD_TAGS=runtime
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
	@echo "  registry   generate pkg/typeregistry/zz_generated_typeregistry.go"
	@echo "  build      compile the typed operator binary"
	@echo "  validate   run katalog validation with your binary"
	@echo "  release    build + push  IMAGE=repo/name:tag"
	@echo "  clean      remove local binary (standard ork unaffected)"