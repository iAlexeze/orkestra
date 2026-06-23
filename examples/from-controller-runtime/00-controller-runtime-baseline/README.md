# 00 — controller-runtime baseline

This works. You run it. It reconciles. You have been here.

A `WebApp` operator written the standard way: controller-runtime, a `Reconcile()` loop that creates a Deployment and a Service, and updates status. About 150 lines of Go across the controller and types — not counting Dockerfile, RBAC generation, Helm chart, and everything else that comes with shipping an operator.

This is the starting point. Nothing is wrong with it. The question is what it costs to maintain as requirements change and CRDs multiply.

---

## Files

| File | Purpose |
|------|---------|
| `api/v1alpha1/webapp_types.go` | CRD types — `WebAppSpec`, `WebAppStatus`, `WebApp`, `WebAppList`, `AddToScheme` |
| `api/v1alpha1/zz_generated.deepcopy.go` | Generated `DeepCopyObject` implementations — required by `runtime.Object` |
| `controller/webapp_controller.go` | `Reconcile()` — creates Deployment, creates Service, updates status |
| `main.go` | Manager setup, scheme registration, signal handling |
| `go.mod` | Module deps: controller-runtime v0.17, k8s.io v0.29 |
| `crd.yaml` | WebApp CRD schema |
| `cr.yaml` | Sample WebApp CR |
| `Dockerfile` | Multi-stage build — produces the controller binary image |
| `chart/` | Helm chart — ServiceAccount, ClusterRole, ClusterRoleBinding, Deployment |

---

## What you write from scratch, every time

- The Deployment spec — image, replicas, port, owner references
- The Service spec — selector, port mapping, owner references
- Status update — `Get` → mutate → `Status().Update()`
- Idempotency — `Get` → `IsNotFound` → `Create`, else `Update`
- Scheme registration in `main.go`
- RBAC markers on every method that touches a resource
- Dockerfile, Makefile, Helm chart, manifests

For one CRD, this is manageable. For five CRDs, it is a maintenance surface.

---

## Step 1 — Run locally

```bash
# Apply the CRD
kubectl apply -f crd.yaml

# Run the controller (requires Go toolchain and a kubeconfig)
go run ./main.go

# In another terminal, apply a CR
kubectl apply -f cr.yaml

# Verify
kubectl get webapps
kubectl get deployments
kubectl get services
```

---

## Step 2 — Build and push the image

> Replace `myorg` with your GitHub username or organisation in the commands below and in `chart/values.yaml`.

```bash
docker build -t ghcr.io/myorg/webapp-operator:0.1.0 .
docker push ghcr.io/myorg/webapp-operator:0.1.0
```

---

## Step 3 — Deploy to a cluster with Helm

> Same replacement — `myorg` appears in the `--set image.repository` flag and in `chart/values.yaml`.

```bash
kubectl apply -f crd.yaml

helm upgrade --install webapp-operator ./chart \
  --set image.repository=ghcr.io/myorg/webapp-operator \
  --set image.tag=0.1.0 \
  --namespace webapp-system \
  --create-namespace \
  --wait
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next

[01 — declarative](../01-declarative/README.md) — the same operator with zero Go.
