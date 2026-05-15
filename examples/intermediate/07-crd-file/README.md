# 07 — CRD File

Declare the path to your CRD YAML inside the Katalog. When you run
`ork run` outside the cluster, Orkestra automatically applies the CRD
files before starting reconcile loops — no manual `kubectl apply` step needed.

**What you learn:** `crdFile` field in `spec.crds`, automatic CRD pre-application
in dev mode, managing multiple CRDs each with their own file.

**Builds on:** [06 — Basic Komposer](../06-komposer-basic/README.md)

---

## What is new

**`crdFile`** — each CRD entry in the Katalog can declare the path to its
CRD YAML:

```yaml
spec:
  crds:
    app:
      apiTypes:
        group: crdfile.orkestra.io
        version: v1alpha1
        kind: App
        plural: apps
      crdFile: ./crd-app.yaml
```

When `ork run` starts outside the cluster, it reads all `crdFile` paths and
runs `kubectl apply -f <path>` for each one before initialising any reconciler.
Your CRDs are guaranteed to exist before the operator begins watching for CRs.

This example has two CRDs — `App` and `Database` — each with its own file.

---

## Steps

### 1. Validate

```bash
ork validate --file katalog.yaml
```

Expected:
```
✓ app
    kind: App
    group: crdfile.orkestra.io / version: v1alpha1 / plural: apps
    crdFile: ./crd-app.yaml
    mode: dynamic / workers: 2 / resync: 30s

✓ database
    kind: Database
    group: crdfile.orkestra.io / version: v1alpha1 / plural: databases
    crdFile: ./crd-database.yaml
    mode: dynamic / workers: 1 / resync: 60s
```

### 2. Start the operator

```bash
ork run --file katalog.yaml
```

Watch the startup logs. You will see two lines like:

```
INF crdFile applied crd=app path=.../crd-app.yaml
INF crdFile applied crd=database path=.../crd-database.yaml
```

Both CRDs are installed automatically. No separate `kubectl apply -f crd-*.yaml`
step is needed.

### 3. Apply the CRs

In a second terminal:

```bash
kubectl apply -f cr-app.yaml
kubectl apply -f cr-database.yaml
```

### 4. Verify

```bash
kubectl get apps
kubectl get databases
```

Expected (after a moment):
```
NAME      IMAGE                PHASE
my-app    nginx:stable-alpine  Running

NAME          ENGINE    STORAGE   PHASE
my-database   postgres  5Gi       Running
```

### 5. Inspect the running state

```bash
curl localhost:8080/katalog | jq '.crds[] | {name: .name, workers: .workers, resync: .resync}'
```

Both CRDs are running in the same Orkestra instance.

---

## Without crdFile

Without this feature, the typical flow is:

```bash
kubectl apply -f crd-app.yaml
kubectl apply -f crd-database.yaml
ork run --file katalog.yaml
kubectl apply -f cr-app.yaml
```

With `crdFile`, the operator owns its CRD lifecycle in dev mode. The Katalog
is self-contained — you hand it off and it just works.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
