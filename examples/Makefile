# Shared Makefile for Orkestra examples.
# Copied into each pack by `ork init` — every generated project gets this.

CLUSTER_NAME  ?= orkestra-playground
CLUSTER_A     ?= orkestra-a
CLUSTER_B     ?= orkestra-b
KATALOG       ?= katalog.yaml
E2E_FILE      ?= e2e.yml
LOAD_CRD      ?= ingestors                 # override per example: make load LOAD_CRD=loaders
LOAD_COUNT    ?= 50                        # number of CRs to create
LOAD_NS       ?= default

.PHONY: help cluster multi-cluster clean validate test e2e load load-clean

help:
	@echo ""
	@echo "  make cluster        Create a local Kind cluster ($(CLUSTER_NAME))"
	@echo "  make multi-cluster  Create two clusters for cross-cluster examples ($(CLUSTER_A), $(CLUSTER_B))"
	@echo "  make clean          Remove CRs, CRDs, RBAC and delete the Kind cluster"
	@echo "  make validate       Validate the Katalog with ork validate"
	@echo "  make test           Alias for validate"
	@echo "  make e2e            Run the example end-to-end (requires $(E2E_FILE))"
	@echo "  make load           Flood the queue with $(LOAD_COUNT) CRs (LOAD_CRD=$(LOAD_CRD))"
	@echo "  make load-clean     Delete all generated load CRs"
	@echo ""

cluster:
	@./setup-kind.sh create $(CLUSTER_NAME)

multi-cluster:
	@./setup-kind.sh create $(CLUSTER_A)
	@./setup-kind.sh create $(CLUSTER_B)

clean:
	@if [ -f cleanup.sh ]; then bash cleanup.sh; fi
	@./setup-kind.sh delete $(CLUSTER_NAME)
	@./setup-kind.sh delete $(CLUSTER_A) 2>/dev/null || true
	@./setup-kind.sh delete $(CLUSTER_B) 2>/dev/null || true

validate test:
	ork validate -f $(KATALOG)

e2e:
	@if [ -f $(E2E_FILE) ]; then \
		ork e2e -k $(E2E_FILE); \
	else \
		echo "No $(E2E_FILE) found — check the example README for manual steps."; \
	fi

load:
	@./load.sh create $(LOAD_CRD) $(LOAD_COUNT) $(LOAD_NS)

load-clean:
	@./load.sh delete $(LOAD_CRD) $(LOAD_NS)
