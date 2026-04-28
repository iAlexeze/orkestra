# 09 — Typed Extensions (Go Hooks)

Orkestra's declarative Katalog can express the full lifecycle of a managed
resource — deployments, services, cronjobs, dependencies, conditions, status,
namespace protection, webhooks, and more. For most operators, a Katalog YAML
file is all you need.

Typed mode is for what comes after that line.

When your operator needs to provision an external resource (a cloud database, a
message queue, an ACL entry) - not yet supported by orkestra providers, coordinate multiple CRDs with shared mutable
state, call an existing Go SDK, or encode business rules that have no
declarative equivalent — you write a typed extension. You bring the logic.
Orkestra brings everything else: the informer, the workqueue, the worker pool,
the finalizer, status management, events, metrics, and the full security layer.

**What you learn:** how to build, validate, and ship a typed Orkestra extension
— your own operator binary with your business logic compiled in.

**Requirement:** `ork` CLI — install from [orkestra-install](https://github.com/orkspace/orkestra#getting-started)

---

## How it works

A typed extension is a separate Go module. It imports Orkestra as a library,
adds a blank import of its generated registry package, and compiles into its
own binary. The resulting binary is a fully self-contained Orkestra operator
that knows about your CRD types.

```
your module
├── api/v1alpha/
│   └── types.go          ← your CRD Go types
├── hooks/
│   └── database_hooks.go ← your reconcile and delete logic
├── pkg/runtime/
│   └── zz_generated_runtime_registry.go  ← generated, do not edit
├── cmd/orkestra/
│   └── main.go           ← imports _ "yourmodule/pkg/runtime"
├── katalog.yaml
├── Makefile
└── Dockerfile
```

---

## Step 1 — Files already in place

This pack includes a complete typed operator: a `Database` CRD with a
StatefulSet, Service, and optional backup CronJob. Examine the files:

- `api/v1alpha/database_types.go` – the Go structs for your CRD.
- `hooks/database_hooks.go` – the `OnReconcile` and `OnDelete` logic.
- `katalog.yaml` – declares the CRD and points to the hook location.
- `cmd/orkestra/main.go` – entrypoint (import disabled initially).
- `Makefile` – build, registry, validate, release targets.

---

## Step 2 — Generate the registry

The registry file wires your Go types into Orkestra's internal maps. Run:

```bash
make registry
```

This executes `ork generate registry --katalog katalog.yaml` and creates
`pkg/runtime/zz_generated_runtime_registry.go`. The file contains an `init()`
that populates `ObjectRegistry` and `HookRegistry` at startup.

---

## Step 3 — Enable the registry in main.go

Open `cmd/orkestra/main.go`. Uncomment the blank import line:

```go
import (
    "context"
    _ "github.com/workspace/orkestra-hooks-demo/pkg/runtime"  // <-- uncomment this
    // ...
)
```

This ensures the generated `init()` runs when your binary starts.

---

## Step 4 — See why you need a custom binary

The standard `ork` CLI does not know about your Go types. Validate it to see the
expected error:

```bash
ork validate -k katalog.yaml
```

Output:

```
addRuntimeObjects: no object constructor registered for demo.orkestra.io/v1alpha1, Kind=Database
```

This is fine — it tells you that the standard `ork` cannot run your typed
operator. Now remove any previous local custom binary (if built before):

```bash
make clean
```

---

## Step 5 — Build your own operator binary

Now build a binary that includes your generated registry:

```bash
make build
```

The binary is placed in `~/.orkestra/bin/ork` (the default `OUTPUT_DIR`). You will
use this binary for the rest of the tutorial. To make it easy, either:

- Add `$HOME/.orkestra/bin` to your PATH (if not already), or
- Call it directly as `~/.orkestra/bin/ork`.

For brevity, this guide will use `./ork` – you can copy the binary to the current
directory or adjust your PATH. Run `make build` again and then:

```bash
cp ~/.orkestra/bin/ork ./ork
```

Now you have a custom `./ork` binary that knows your CRD type.

---

## Step 6 — Validate with your own binary

```bash
./ork validate -k katalog.yaml
```

It should pass without errors. The debug output (from the generated registry)
will confirm that `ObjectRegistry` is populated.

---

## Step 7 — Run locally in Kind

Create a Kind cluster (if not already):

```bash
kind create cluster --name ork-typed
```

Apply the CRD:

```bash
kubectl apply -f crd.yaml
```

Run your custom operator:

```bash
./ork run -k katalog.yaml
```

In another terminal, apply the custom resource:

```bash
kubectl apply -f cr.yaml
```

Watch the operator logs: you will see the `OnReconcile` hook firing. Verify
the created resources:

```bash
kubectl get statefulset,service,cronjob -A
```

Observe the events:

```bash
kubectl get events --field-selector involvedObject.name=my-db
```

---

## Step 8 – Deploy to a cluster (production)

### Build and push your Docker image

```bash
make release IMAGE=yourregistry/your-operator:v1.0.0
```

### Generate the Orkestra bundle

```bash
./ork generate bundle -k katalog.yaml -o bundle.yaml
```

This creates a single YAML with Namespace, ServiceAccount, ClusterRole,
ClusterRoleBinding, and the ConfigMap containing your Katalog.

```bash
kubectl apply -f bundle.yaml
```

### Install Orkestra Helm chart with your image

```bash
helm repo add orkestra https://ialexeze.github.io/orkestra
helm install orkestra orkestra/orkestra \
  --set runtime.image.repository=yourregistry/your-operator \
  --set runtime.image.tag=v1.0.0 \
  --namespace orkestra-system \
  --create-namespace
```

### Apply the CRD and a custom resource

```bash
kubectl apply -f crd.yaml
kubectl apply -f cr.yaml
```

---

## What Orkestra provides, what you provide

| Concern | Orkestra | Your hook |
|---------|----------|-----------|
| Informer watching your CRD | ✓ | |
| Workqueue with dedup and backoff | ✓ | |
| Worker pool | ✓ | |
| Finalizer lifecycle | ✓ | |
| Kubernetes events | ✓ | |
| Ready condition (Layer 1 status) | ✓ | |
| Custom status fields (Layer 2) | ✓ via katalog | |
| Prometheus metrics | ✓ | |
| Deletion protection webhook | ✓ via katalog | |
| Namespace validation webhook | ✓ via katalog | |
| Reconcile logic | | ✓ |
| Delete logic | | ✓ |
| External API calls | | ✓ |
| Typed spec access | | ✓ |

---

## When to use typed mode

Reach for typed mode when the business logic cannot be expressed in a Katalog:

- Provisioning external resources that are not yet supported declaratively
- Calling an existing Go SDK or internal library
- Multi-resource coordination with shared mutable state across reconcile cycles
- Complex validation that goes beyond admission webhook rules
- Gradual migration of an existing kubebuilder/controller-runtime operator

For resource orchestration, status, dependencies, webhooks, and conditions — the
Katalog handles it without a line of Go.

---

## Cleanup

Stop the local operator with `Ctrl+C`, then:

```bash
kind delete cluster --name ork-typed
```

For the production deployment:

```bash
helm uninstall orkestra -n orkestra-system
kubectl delete -f bundle.yaml
```

---

## Next steps

- Read the `hooks/database_hooks.go` file – see how `OnReconcile` creates
  a StatefulSet, Service, and optional CronJob using OrkestraRegistry.
- Modify the Katalog to add status fields, webhooks, or dependencies.
- Turn this example into your own operator by replacing the API types and hooks.