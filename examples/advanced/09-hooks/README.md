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

Hooks are **additive** — the Katalog still owns what it can. In this example the `ServiceAccount` is declared in the Katalog and Orkestra creates it before the hook runs. The hook references it by the same naming convention:

```yaml
# katalog.yaml — Orkestra handles this
onCreate:
  serviceAccounts:
    - name: "{{ .metadata.name }}-sa"
```

```go
// hooks/database_hooks.go — hook references it by convention
ServiceAccountName: obj.Name + "-sa",
```

Declare in YAML what Orkestra handles well. Write Go for what it can't.

**What you learn:** how to build, validate, and ship a typed Orkestra extension
— your own operator binary with your business logic compiled in.

**Requirement:** `ork` CLI — install from [orkestra-install](https://github.com/orkspace/orkestra#getting-started)

---

## Step 1 — Files already in place

This pack includes a complete typed operator: a `Database` CRD with a
StatefulSet, Service, and optional backup CronJob. Examine the files:

- `api/v1alpha/database_types.go` – the Go structs for your CRD.
- `hooks/database_hooks.go` – the `OnReconcile` and `OnDelete` logic.
- `katalog.yaml` – declares the CRD and points to the hook location.
- `Makefile` – build, registry, validate, release targets.

---

## Step 2 — Generate the registry and entrypoint

Run the registry generator:

```bash
make registry
```

This executes:

```bash
ork generate registry --file katalog.yaml
```

It creates (or updates) two files:

- `pkg/typeregistry/zz_generated_typeregistry.go` – registers your Go types and hooks.
- `cmd/orkestra/main.go` – the entrypoint that imports the generated registry.

Both files are marked `DO NOT EDIT` – they are regenerated whenever you change the Katalog.

---

## Step 3 — See why you need a custom binary

The standard `ork` CLI does not know about your Go types. Validate it to see the
expected error:

```bash
ork validate
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

## Step 4 — Build your own operator binary

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

## Step 5 — Validate with your own binary

```bash
./ork validate
```

It should pass without errors. The debug output (from the generated registry)
will confirm that `ObjectRegistry` is populated.

---

## Step 6 — Run locally in Kind

>[!IMPORTANT]
> `ork run` with `--dev` flag spins up a local kind cluster, and deploys orkestra.
> Skip if you already have a cluster running.

Run your custom operator:

```bash
./ork run --dev
```

In another terminal, apply the custom resource:

Apply the CRD:

```bash
kubectl apply -f crd.yaml
```

```bash
kubectl apply -f cr.yaml
```

Watch the operator logs: you will see the `OnReconcile` hook firing. Verify
the created resources:

```bash
kubectl get statefulset,service,cronjob -n default
```

Observe the events:

```bash
kubectl get events --field-selector involvedObject.name=my-db
```

---

## Step 7 – Deploy to a cluster (production)

### Build and push your Docker image

```bash
make release IMAGE=yourregistry/your-operator:v1.0.0
```

### Generate the Orkestra bundle

```bash
./ork generate bundle -f katalog.yaml -o bundle.yaml
```

This creates a single YAML with Namespace, ServiceAccount, ClusterRole,
ClusterRoleBinding, and the ConfigMap containing your Katalog.

```bash
kubectl apply -f bundle.yaml
```

### Install Orkestra Helm chart with your image

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --set runtime.image.repository=yourregistry/your-operator \
  --set runtime.image.tag=v1.0.0 \
  --namespace orkestra-system \
  --wait --timeout 120s
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

## E2E

Typed operators require your own published image — `ork e2e` installs Orkestra via Helm and you must point it at your image so the deployed runtime includes your generated type registry.

**Step 1 — build and push your image:**

```bash
make docker push IMAGE_REPO=yourregistry/database-operator IMAGE_TAG=latest
```


**Important notes: (build-time security)**

- `make docker` builds a production‑only binary (tags `runtime`) – it cannot run developer commands like `validate` or `e2e`
- The binary is copied from `~/.orkestra/bin/runtime/ork` into the current directory for `docker build`, then removed
- Your local `./ork` (development CLI) is restored after the build – it remains unchanged and fully featured
- This round‑trip ensures your container image contains only the secure runtime, while your local environment keeps all developer tools

#### Try running a developer command
It will fail (as intended), only `ork run` succeeds.

```bash
~/.orkestra/bin/runtime/ork validate
# unknown command "validate" for "ork"
```

---

**Step 2 — run e2e with your image:**

```bash
ork e2e \
  --set runtime.image.repository=yourregistry/database-operator \
  --set runtime.image.tag=latest
```

The `--set` flags are are used by `ork e2e`, so the cluster runs your image instead of the default Orkestra image.

This spins up a kind cluster, deploys your custom Orkestra runtime, applies the CR, asserts your Go hooks fired and the expected resources exist, then tears down.

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Database deployment created
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        namespace: default
        ready: true

  - name: Deployment removed on delete
    after: cr-deleted
    timeout: 30s
    resources:
      - kind: Deployment
        namespace: default
        count: 0
```

---

## Cleanup

Stop the local operator with `Ctrl+C`, then:

```bash
kind delete cluster --name orkestra-playground
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