# Profiles 01 — Resource

One CR. Eight Deployments. Each gets a different CPU and memory budget from a profile name — no explicit `requests` or `limits` needed in the Katalog.

**What you learn:** `resources.profile`, what each preset expands to, how to pick the right profile for a workload.

---

## Profiles at a glance

| Profile | CPU request | CPU limit | Memory request | Memory limit | Use for |
|---|---|---|---|---|---|
| `tiny` | 25m | 100m | 64Mi | 128Mi | Sidecars, agents |
| `small` | 100m | 500m | 128Mi | 512Mi | Low-traffic APIs |
| `medium` | 250m | 1 | 256Mi | 1Gi | Standard web services |
| `large` | 500m | 2 | 512Mi | 2Gi | Data-intensive services |
| `burst` | 200m | 2 | 256Mi | 2Gi | Spiky workloads |
| `steady` | 300m | 600m | 256Mi | 512Mi | Stable workloads |
| `compute-heavy` | 1 | 2 | 512Mi | 1Gi | CPU-bound workers |
| `memory-heavy` | 250m | 500m | 1Gi | 2Gi | Caches, JVM apps |

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Simulate

```bash
ork simulate
```

---

## Step 3 — Start the runtime

```bash
ork run
```

---

## Step 4 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **service-resource-profiles**, then **Service**.

---

## Step 5 — Apply the CR

```bash
kubectl apply -f ../cr.yaml
```

Eight Deployments are created — one per profile. Click the `my-service` CR in the Control Center, then **top-right** to see all eight child Deployments.

> `phase` transitions to `Ready` once pods are running — allow ~15s after applying.
> If you are using a `1-node-kind-cluster`, some pods might be in `Pending` state due to limited resources.

Verify what each profile expanded to:

```bash
kubectl get deployments -o custom-columns=\
'NAME:.metadata.name,CPU-REQ:.spec.template.spec.containers[0].resources.requests.cpu,MEM-REQ:.spec.template.spec.containers[0].resources.requests.memory'
```

Expected:
```
NAME                       CPU-REQ   MEM-REQ
my-service-tiny            25m       64Mi
my-service-small           100m      128Mi
my-service-medium          250m      256Mi
my-service-large           500m      512Mi
my-service-burst           200m      256Mi
my-service-steady          300m      256Mi
my-service-compute-heavy   1         512Mi
my-service-memory-heavy    250m      1Gi
```

---

## Using a profile in your own Katalog

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    image: "{{ .spec.image }}"
    resources:
      profile: medium   # expands to cpu:250m/1, memory:256Mi/1Gi
```

---

## E2E

Run the full lifecycle in one command — applies the CR, asserts all eight profile Deployments are created with the correct CPU requests, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Eight profile Deployments created
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        name: my-service-tiny
        namespace: default
      - kind: Deployment
        name: my-service-large
        namespace: default

  - name: Tiny profile has correct CPU request
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get deployment my-service-tiny -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}'
        outputContains: 25m
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
