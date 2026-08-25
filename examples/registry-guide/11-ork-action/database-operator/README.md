# 10 — Typed operator: Go hooks

Build a typed Go operator using Orkestra hooks. The Database CRD drives a StatefulSet, Service, and optional backup CronJob — all created from type-safe Go structs rather than YAML templates.

---

## Files

| File | Purpose |
|------|---------|
| `katalog.yaml` | Database Katalog with `operatorBox.hooks` declaration |
| `crd.yaml` | Database CRD (group: `rkguide.demo`) |
| `cr.yaml` | Sample Database CR — postgres 14, 10Gi, backup enabled |
| `simulate.yaml` | Gate: StatefulSet + Service + CronJob created on cycle 1 |
| `e2e.yaml` | Full cluster test: create CR, verify resources, delete CR |
| `api/v1alpha1/database_types.go` | Go type for the Database CRD |
| `hooks/database_hooks.go` | Typed reconcile hooks |
| `go.mod` | Module file
| `go.sum` | Checksums |
| `Makefile` | Registry generation, build, docker, push targets |
| `Dockerfile` | Distroless runtime image |
| `cleanup.sh` | Remove CRD and CR from cluster |

---

> **Before you start:** If `ORK_REGISTRY` is not set, export it now (see [01-motifs](../01-motifs/README.md#push-to-the-registry)). Replace `myorg` with your actual registry path throughout this example.

---

## Why typed hooks?

Declarative templates work for most patterns. Hooks are the escape hatch when logic is not expressible in templates

Orkestra still provides the informer, workqueue, worker pool, finalizer management, status management, and metrics. The hook only provides the business logic.

---

## Workflow

**1. Generate the type registry**

```bash
make registry
```

Runs `ork generate registry --file katalog.yaml` and generates `pkg/typeregistry/zz_generated_typeregistry.go`. Re-run whenever `apiTypes` fields in `katalog.yaml` change.

**2. Build the development binary**

```bash
make build
```

Compiles a full CLI binary with all commands. Output: `~/.orkestra/bin/ork`.

**3. Validate and simulate**

```bash
make validate
make simulate
```

**4. Build and push the production image**

```bash
export IMAGE_REPO=ghcr.io/myorg/database-operator
export IMAGE_TAG=v1.0.0
make release
```

`make release` compiles with the `runtime` build tag (no validate/simulate/e2e commands), builds the distroless image, and pushes it.

**5. Update [values.yaml](values.yaml) with your image**

The e2e gate runs automatically during push and needs to pull your custom runtime image. Update [values.yaml](values.yaml) to point to the image you just built:

```yaml
runtime:
  image:
    repository: ghcr.io/myorg/database-operator
    tag: v1.0.0
```

This is the declarative equivalent of `--set runtime.image.repository=... --set runtime.image.tag=...` used in the [advanced typed examples](../../advanced/09-hooks/README.md). The file travels with the artifact so any consumer running e2e gets the correct image automatically.

**6. Push the katalog pattern to the registry**

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

Simulate and e2e run automatically before the artifact and its dependencies are published.

**7. Confirm the published artifact**

```bash
ork inspect database-operator:v1.0.0
```

Expected output (excerpt):

```text
  Typed:       ✓ hooks · requires custom runtime
  Runtime:     v0.7.6

  Files:
    katalog.yaml                   1.9 KB
    crd.yaml                       1.2 KB
    README.md                      4.2 KB
    cr.yaml                        220 B
    e2e.yaml                       947 B
    simulate.yaml                  650 B
    go.mod                         11.2 KB
    go.sum                         65.5 KB
    Makefile                       6.2 KB

To pull:
  ork pull database-operator:v1.0.0

To import:
  imports:
    registry:
      - oci://ghcr.io/myorg/katalogs/database-operator:v1.0.0
```

`Typed: ✓ hooks` confirms the hooks annotation was written. `Runtime: v0.7.6` is read from `go.mod` and tells consumers which Orkestra version the operator was built against. The `To import:` block is the exact entry a downstream Komposer needs to pull and run this operator.

---

## Deploy in-cluster

```bash
kubectl apply -f crd.yaml

ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml

helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set runtime.image.repository=ghcr.io/myorg/database-operator \
  --set runtime.image.tag=v1.0.0 \
  --wait --timeout 120s

kubectl apply -f cr.yaml
kubectl get database my-database
```

---

## How the hook wires up

```yaml
operatorBox:
  hooks:
    location: github.com/orkspace/orkestra-registry-guide/hooks
    function: DatabaseHooks
    alias: dbhooks
    managedResources:
      - kind: StatefulSet
      - kind: Service
      - kind: CronJob
```

- `location` — Go import path for the hooks package
- `function` — exported function that returns `domain.AnyReconcileHooks`
- `resources` — list of Kubernetes kinds managed by the hook, used for RBAC generation
- `ork generate registry` reads these and writes the type registry

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next step

→ [11-typed-komposer](../11-typed-komposer/README.md) — combine a declarative katalog and a typed operator in one Komposer
