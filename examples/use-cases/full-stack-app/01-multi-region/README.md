# 01 — Multi-Region Deployment (forEach over a list)

One CR, one list, N Deployments. Orkestra iterates `spec.regions` and creates a Deployment per entry — same image, same replica count, only the name differs. No loop in Go, no per-region code.

**What you learn:** `forEach` over a list field. How `.item` carries the region name. Why a list is right when all targets share the same config — and where it falls short when they don't (see [multi-region-map](../../multi-region-map/README.md) for per-region replicas and ports).

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
✓ multi-region-app
    kind: MultiRegionApp
    group: advanced.orkestra.io / version: v1alpha1 / plural: multiregionapps
    mode: dynamic / workers: 2 / resync: 30s
```

---

## Step 2 — Start the operator

`ork run` reads the `crdFile` declared in `katalog.yaml`, applies the CRD to the cluster, and starts the runtime:

```bash
ork run
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **multi-region-app**, then select the **MultiRegionApp** CRD.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

The CR declares three regions as a flat list — no per-region config:

```yaml
spec:
  image: nginx:1.25
  regions:
    - us-east-1
    - eu-west-1
    - ap-southeast-1
  defaultReplicas: 1
```

Wait one reconcile (~30s).

```bash
kubectl get multiregionapp my-multi-region -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  regionsDeployed: "3"
```

---

## Step 5 — Check the Deployments

```bash
kubectl get deploy -l orkestra-owner=my-multi-region
```

Expected:
```
NAME                               READY   UP-TO-DATE   AVAILABLE
my-multi-region-ap-southeast-1    1/1     1            1
my-multi-region-eu-west-1         1/1     1            1
my-multi-region-us-east-1         1/1     1            1
```

Three Deployments from one CR. All the same image, all 1 replica. Notice there are no Services — this list-based forEach creates Deployments only, with no per-region port configuration.

---

## Step 6 — Add a region live

Edit the CR and add a fourth region:

```bash
kubectl edit multiregionapp my-multi-region
```

Add `ap-northeast-1` to the regions list. Orkestra reconciles immediately on the edit event — a fourth Deployment appears without any Katalog change.

---

## Step 7 — The list limitation

All three Deployments use the same port. There is no way to assign different ports or replica counts to individual regions using a list — every region gets identical config.

For per-region replicas and ports — where each region carries its own configuration — see [multi-region-map](../../multi-region-map/README.md). That example uses a map instead of a list and adds a real app you can port-forward to see each region individually.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRD, starts the operator, applies the CR, asserts three regional Deployments are created and ready, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Three regional Deployments ready
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        name: my-multi-region-us-east-1
        namespace: default
        ready: true
      - kind: Deployment
        name: my-multi-region-eu-west-1
        namespace: default
        ready: true
      - kind: Deployment
        name: my-multi-region-ap-southeast-1
        namespace: default
        ready: true
```

---

## Cleanup

```bash
kubectl delete multiregionapp my-multi-region --ignore-not-found
```
