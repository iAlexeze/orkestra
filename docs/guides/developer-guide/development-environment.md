# Development Environment

This section describes how to set up a development environment for Orkestra. It is modeled after Argo CD’s development workflow but tailored specifically for Orkestra’s architecture and tooling.

---

## Prerequisites

To develop Orkestra, you need the following tools installed:

- Go 1.22 or later  
- Docker or containerd  
- Make  
- Git  
- kubectl  
- A local Kubernetes cluster (kind, k3d, microk8s, or Docker Desktop)  
- controller‑gen (for typed CRDs)  
- yq (optional, for YAML manipulation)  

Orkestra is developed and tested primarily on Linux and macOS, but Windows with WSL2 is also supported.

---

## Repository Structure

A typical Orkestra repository layout:

```
cmd/
  ork/                 # CLI entrypoint
  orkestra-runtime/    # Runtime entrypoint

pkg/
  runtime/             # Core runtime logic
  registry/            # Registry loading and katalog resolution
  komposer/            # Komposer engine
  reconciler/          # Generic reconciler and hooks
  api/                 # Typed CRD definitions (optional)
  metrics/             # Prometheus metrics
  health/              # Health endpoints

scripts/
  dev/                 # Developer scripts
  certs/               # TLS generation helpers

tests/
  unit/
  integration/
  e2e/
```

This structure keeps the runtime, CLI, and extension points cleanly separated.

---

## Setting Up a Local Kubernetes Cluster

You can use any local cluster, but the recommended options are:

### kind (recommended)

```
kind create cluster --name ork-dev
```

### k3d

```
k3d cluster create ork-dev
```

### microk8s

```
microk8s enable dns storage
```

After creating the cluster, verify connectivity:

```
kubectl get nodes
```

---

## Building the Runtime

Clone the repository:

```
git clone https://github.com/orkestra/orkestra.git
cd orkestra
```

Build the CLI:

```
make ork
```

Build the runtime:

```
make runtime
```

Both binaries will be placed in `bin/`.

---

## Running Orkestra Locally

Orkestra can run outside the cluster and connect to your local Kubernetes context.

```
./bin/orkestra-runtime \
  --katalog ./examples/katalog.yaml --debug
```

This starts:

- Informers  
- Work queues  
- Worker pools  
- Conversion webhook server  
- Health endpoints  
- Metrics server  

You can now apply CRDs and CRs to your local cluster and observe reconciliation.

---

## Hot Reloading Katalogs

During development, you may want to reload katalogs without restarting the runtime.

```
ork template --katalog katalog.yaml --graph
```

Orkestra does not automatically reload katalogs at runtime, but the CLI provides fast validation and preview tools.

---

## Generating runtime registry

Typed CRDs, Go hooks, and custom constructors require runtime generation:

```
ork generate registry --katalog katalog.yaml
go mod tidy
```

This generates:

- Scheme registration  
- Hook registry entries  
- Reconciler constructors  

Dynamic CRDs do not require generation.

---

## Running Tests

### Unit Tests

```
make test-unit
```

### Integration Tests

```
make test-integration
```

### End‑to‑End Tests

Requires a running cluster:

```
make test-e2e
```

### Full Test Suite

```
make test-all
```

---

## Debugging

### Logs

Run the runtime with debug logging:

```
./bin/orkestra-runtime --debug
```

### Health Endpoints

```
curl localhost:8080/katalog/<crd>/health
```

### Metrics

```
curl localhost:8080/metrics
```

### Kubernetes Events

```
kubectl get events --sort-by=.lastTimestamp
```

---

## Development Workflow Summary

1. Create or modify katalogs  
2. (Optional) Add typed CRDs or hooks  
3. Run `ork generate registry`  
4. Build the runtime  
5. Start a local cluster  
6. Run Orkestra locally  
7. Apply CRDs and CRs  
8. Observe reconciliation  
9. Run tests  
10. Submit a pull request  
