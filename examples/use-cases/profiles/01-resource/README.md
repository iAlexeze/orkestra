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

## Step 2 — Start the operator

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

Open [http://localhost:8081](http://localhost:8081). Select **service-resource-profiles**, then **Service**.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f ../cr.yaml
```

Eight Deployments are created — one per profile. Click the `my-service` CR in the Control Center, then **top-right** to see all eight child Deployments.

> `phase` transitions to `Ready` once pods are running — allow ~15s after applying.

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

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
