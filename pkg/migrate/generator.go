// pkg/migrate/generator.go
//
// Generates the Orkestra scaffolding files that accompany a migrated reconciler:
// katalog.yaml, simulate.yaml, e2e.yaml, and go.mod.
//
// All generated files are stubs with TODO markers. The user fills in
// CRD details (group, kind, location) and resource assertions.
package migrate

import (
	"fmt"
	"strings"
)

// Files holds all generated file contents keyed by filename.
type Files struct {
	Katalog    string
	Simulate   string
	E2E        string
	GoMod      string
	Makefile   string
	Dockerfile string
}

// Options controls what the generator emits.
type Options struct {
	// ModulePath is the Go module path of the migrated operator (e.g. github.com/myorg/my-operator).
	// Used in go.mod and as a hint in katalog.yaml location fields.
	ModulePath string

	// OperatorName is the kebab-case name for the operator (e.g. webapp-operator).
	// Derived from ReceiverType if not set.
	OperatorName string

	// OrkVersion is the Orkestra CLI version (from pkg/version.Short()).
	// Written into go.mod as the orkestra require version.
	OrkVersion string
}

// Generate produces all scaffolding files from a Rewrite result.
func Generate(res *Result, opts Options) Files {
	if opts.OperatorName == "" {
		opts.OperatorName = toKebab(res.ReceiverType)
	}
	if opts.ModulePath == "" {
		opts.ModulePath = "github.com/myorg/" + opts.OperatorName
	}

	crdName := strings.TrimSuffix(opts.OperatorName, "-operator")
	crdName = strings.TrimSuffix(crdName, "-reconciler")
	constructorFn := "New" + res.ReceiverType

	return Files{
		Katalog:    generateKatalog(res, opts, crdName, constructorFn),
		Simulate:   generateSimulate(opts, crdName),
		E2E:        generateE2E(opts, crdName),
		GoMod:      generateGoMod(opts),
		Makefile:   generateMakefile(opts),
		Dockerfile: generateDockerfile(),
	}
}

func generateKatalog(res *Result, opts Options, crdName, constructorFn string) string {
	return fmt.Sprintf(`# Schema reference: https://orkestra.sh/docs/reference/schema/katalog/
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: %s
  author: myorg  # TODO(ork migrate): set your name or org
  version: 0.1.0
  description: >
    Migrated from controller-runtime. The constructor lifted from %s
    runs inside Orkestra's reconcile loop — informer, workqueue, worker pool,
    leader election, and metrics are provided by the runtime.
  tags:
    - migration
    - constructor
    - typed

spec:
  crds:
    %s:
      apiTypes:
        group: TODO  # TODO(ork migrate): set your CRD group (e.g. apps.myorg.io)
        version: v1alpha1
        kind: TODO   # TODO(ork migrate): set your CRD kind (e.g. WebApp)
        plural: TODO # TODO(ork migrate): set the plural (e.g. webapps)
        object: TODO
        objectList: TODOList
        location: %s/api/v1alpha1  # TODO(ork migrate): adjust to your API types package
        alias: apiv1alpha1

      allowedNamespaces:
        - default

      operatorBox:
        # default: false — the GenericReconciler is not used.
        # Your constructor owns the full reconcile loop.
        default: false

        constructor:
          location: %s/%s  # TODO(ork migrate): adjust to your reconciler package
          function: %s
          resources:
            - kind: TODO  # TODO(ork migrate): list every resource kind your operator manages
`, opts.OperatorName, res.PkgName, crdName, opts.ModulePath, opts.ModulePath, res.PkgName, constructorFn)
}

func generateSimulate(opts Options, crdName string) string {
	return fmt.Sprintf(`# Schema reference: https://orkestra.sh/docs/reference/schema/simulate/
apiVersion: orkestra.orkspace.io/v1
kind: Simulate
metadata:
  name: %s-sim
  description: >
    Verify resources are created in cycle 1 — no cluster needed.
    TODO(ork migrate): fill in the resource kinds your operator creates.

spec:
  katalog: ./katalog.yaml
  cr: ./cr.yaml
  cycles: 3

  expect:
    steady: true
    noErrors: true
    ops:
      - cycle: 1
        verb: create
        resource: TODO  # TODO(ork migrate): e.g. deployments, services, configmaps
      # - cycle: 1
      #   verb: create
      #   resource: services
`, opts.OperatorName)
}

func generateE2E(opts Options, crdName string) string {
	return fmt.Sprintf(`# Schema reference: https://orkestra.sh/docs/reference/schema/e2e/
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: %s-e2e
  description: >
    End-to-end test for %s.
    TODO(ork migrate): fill in your CRD kind, CR name, and assertions.

spec:
  katalog: ./katalog.yaml
  crd: ./crd.yaml
  cr: ./cr.yaml

  cluster:
    provider: kind
    name: ork-e2e
    reuse: false

  expect:
    - name: CR created
      after: cr-applied
      timeout: 30s
      resources:
        - kind: TODO  # TODO(ork migrate): set your CRD kind
          name: TODO  # TODO(ork migrate): set your CR name from cr.yaml
          namespace: default

    - name: Status written
      after: cr-applied
      timeout: 60s
      commands:
        - run: kubectl get TODO my-cr -o jsonpath='{.status.phase}'  # TODO(ork migrate): adjust
          outputContains: Running

    - name: Resources created
      after: cr-applied
      timeout: 90s
      resources:
        - kind: TODO  # TODO(ork migrate): e.g. Deployment
          name: TODO
          namespace: default
          ready: true

    - name: Cleanup verified
      after: cr-deleted
      timeout: 30s
      resources:
        - kind: TODO  # TODO(ork migrate): your managed resource
          name: TODO
          namespace: default
          count: 0
`, opts.OperatorName, opts.OperatorName)
}

func generateGoMod(opts Options) string {
	orkVer := opts.OrkVersion
	if orkVer == "" || orkVer == "dev" {
		orkVer = "v0.0.0  // TODO(ork migrate): replace with the published Orkestra version"
	}
	return fmt.Sprintf(`module %s

go 1.22

require (
	github.com/orkspace/orkestra %s
	k8s.io/api v0.29.3
	k8s.io/apimachinery v0.29.3
	k8s.io/client-go v0.29.3
)

// Run: go mod tidy
// to resolve all indirect dependencies.
`, opts.ModulePath, orkVer)
}

func generateMakefile(opts Options) string {
	return fmt.Sprintf(`# ── Typed Orkestra Operator ───────────────────────────────────────────────────
BINARY_NAME ?= ork
DEV_OUTPUT_DIR  ?= $(HOME)/.orkestra/bin
PROD_OUTPUT_DIR ?= $(HOME)/.orkestra/bin/runtime
KATALOG     ?= katalog.yaml

IMAGE_REPO  ?= myorg/%s
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
	@if [ -f $(DEV_OUTPUT_DIR)/$(BINARY_NAME) ]; then cp $(DEV_OUTPUT_DIR)/$(BINARY_NAME) ./$(BINARY_NAME); fi

.PHONY: push
push:
	docker push $(IMAGE)

.PHONY: release
release: docker push

.PHONY: clean
clean:
	@rm -f $(DEV_OUTPUT_DIR)/$(BINARY_NAME)
	@rm -rf $(PROD_OUTPUT_DIR)

.PHONY: help
help:
	@echo "  registry       generate type registry from Katalog"
	@echo "  build          compile full development CLI"
	@echo "  build-runtime  compile production binary (only 'ork run')"
	@echo "  validate       run katalog validation"
	@echo "  e2e            run end-to-end tests"
	@echo "  docker         build-runtime + Docker image"
	@echo "  push           push Docker image"
	@echo "  release        docker + push"
	@echo "  clean          remove all local builds"
`, opts.OperatorName)
}

func generateDockerfile() string {
	return `FROM gcr.io/distroless/static-debian12:nonroot
COPY ork /usr/local/bin/ork
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/ork"]
`
}

// toKebab converts a PascalCase type name to kebab-case.
// "WebAppReconciler" → "webapp-reconciler"
func toKebab(s string) string {
	if s == "" {
		return "my-operator"
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('-')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}
