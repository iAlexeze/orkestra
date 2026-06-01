# Multi-Region Map — forEach over a map

**Builds on:** [full-stack-app/01-multi-region](../full-stack-app/01-multi-region/README.md) — list forEach, same config for all regions.
Here each region carries its own replica count and port — this is what map forEach enables.

---

You have one application and you want to run it in three regions, each with its own replica count and port. Normally that means a reconciler loop in Go, building one Deployment and Service per region. Here it is a twelve-line `forEach` block in a Katalog — Orkestra expands it at reconcile time.

The key property: **adding or changing a region requires editing only the CR** — no Katalog change, no redeployment of Orkestra. Step 6 demonstrates this live.

**What you learn:** `forEach` over a map field. How `.item` carries the map key (the region name) and `.value.*` carries the per-region data. How a single CR entry becomes N child resources automatically.

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

```bash
ork run
```

Orkestra registers `crd.yaml` and starts the operator. You will see:

```
{"level":"info","message":"health server listening on :8080"}
{"level":"info","crd":"advanced.orkestra.io/v1alpha1, Kind=MultiRegionApp","message":"informer synced"}
{"level":"info","message":"✅ All komponents started successfully"}
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081).

Select **multi-region-demo** from the katalog list, then select the **MultiRegionApp** CRD. You will see an empty view — no CRs yet. Keep this open.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

The [CR](cr.yaml) declares three regions as a map. Each key is a region name; each value carries that region's specific config:

```yaml
spec:
  image: ghcr.io/orkspace/orkestra-demo:multi-region-map
  defaultReplicas: 1
  defaultPort: 8080
  regions:
    us-east-1:
      replicas: 3
      port: 8080
    eu-west-1:
      replicas: 1
      port: 8081
    ap-southeast-1:
      replicas: 2
      port: 8080
```

Switch to the Control Center. The `my-multi-region` CR appears in the list. Click it, then click the **top-right button** to open child resources. You will see **6 children** — one Deployment and one Service per region.

> The Katalog's `forEach` block iterates `spec.regions` once per key. `.item` is the region name, `.value.replicas` and `.value.port` are the per-region overrides. Where a region omits a field, the `or` fallback picks up `spec.defaultReplicas` or `spec.defaultPort`.

---

## Step 5 — Check what got created

```bash
kubectl get deployments,services -l orkestra-owner=my-multi-region
```

Expected (within a few seconds):

```
NAME                                                READY   UP-TO-DATE   AVAILABLE
deployment.apps/my-multi-region-ap-southeast-1     2/2     2            2
deployment.apps/my-multi-region-eu-west-1          1/1     1            1
deployment.apps/my-multi-region-us-east-1          3/3     3            3

NAME                                                    TYPE        PORT(S)
service/my-multi-region-ap-southeast-1-svc             ClusterIP   8080/TCP
service/my-multi-region-eu-west-1-svc                  ClusterIP   8081/TCP
service/my-multi-region-us-east-1-svc                  ClusterIP   8082/TCP
```

One CR entry → 6 Kubernetes resources. Replica counts and ports match exactly what you declared in the map.

Check status on the CR itself:

```bash
kubectl get multiregionapp my-multi-region
```

```
NAME               PHASE   REGIONS   AGE
my-multi-region    Ready             30s
```

---

## Step 6 — Port-forward and visit each region

Open three terminals and forward to each region's Service:

```bash
# Terminal 1 — us-east-1
kubectl port-forward svc/my-multi-region-us-east-1-svc 8000:8080

# Terminal 2 — eu-west-1
kubectl port-forward svc/my-multi-region-eu-west-1-svc 8001:8081

# Terminal 3 — ap-southeast-1
kubectl port-forward svc/my-multi-region-ap-southeast-1-svc 8002:8082
```

Visit each in your browser:

| URL | Shows |
|---|---|
| [http://localhost:8000](http://localhost:8000) | `Region: us-east-1` |
| [http://localhost:8001](http://localhost:8001) | `Region: eu-west-1` |
| [http://localhost:8002](http://localhost:8002) | `Region: ap-southeast-1` |

Each pod only knows its own region because Orkestra injected `REGION={{ .item }}` and `PORT={{ or .value.port .spec.defaultPort }}` at reconcile time. The container image is identical across all three — no region-specific build, no env-file per region.

---

## Step 7 — Change a region live

Scale `eu-west-1` up and shift its port:

```bash
kubectl edit multiregionapp my-multi-region
```

Change:

```yaml
eu-west-1:
  replicas: 3
  port: 9000
```

Orkestra reconciles immediately on the edit event. Watch the deployment scale:

```bash
kubectl get deployment my-multi-region-eu-west-1 -w
```

```
NAME                             READY   UP-TO-DATE   AVAILABLE
my-multi-region-eu-west-1       1/1     1            1
my-multi-region-eu-west-1       1/1     1            1
my-multi-region-eu-west-1       3/3     3            3
```

Check the Service port updated too:

```bash
kubectl get svc my-multi-region-eu-west-1-svc
```

```
NAME                              TYPE        CLUSTER-IP   PORT(S)
my-multi-region-eu-west-1-svc    ClusterIP   ...          9000/TCP
```

No Katalog change. No redeployment of Orkestra. The map in the CR is the single source of truth — edit the map, the cluster reflects it.

---

## How the forEach works

The Katalog deployment block:

```yaml
deployments:
  - name: "{{ .metadata.name }}-{{ .item }}"
    image: "{{ .spec.image }}"
    replicas: "{{ or .value.replicas .spec.defaultReplicas }}"
    port: "{{ or .value.port .spec.defaultPort }}"
    env:
      - name: REGION
        value: "{{ .item }}"
      - name: PORT
        value: "{{ or .value.port .spec.defaultPort }}"
    forEach:
      field: spec.regions
      as: item
```

When `spec.regions` is a **map**, Orkestra iterates once per key-value pair:

| Variable | Value for `eu-west-1` entry |
|---|---|
| `.item` | `eu-west-1` |
| `.value.replicas` | `1` |
| `.value.port` | `8081` |
| `.spec.defaultReplicas` | `1` (fallback if `.value.replicas` absent) |
| `.spec.defaultPort` | `8080` (fallback if `.value.port` absent) |

The Service block uses the same `forEach` — so each Deployment gets exactly one matching Service, selector already wired, port already set.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRD, starts the operator, applies the CR, asserts all six regional resources are created with the correct replica counts, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Three Deployments created
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

  - name: us-east-1 has 3 replicas
    after: cr-applied
    timeout: 90s
    commands:
      - run: kubectl get deployment my-multi-region-us-east-1 -o jsonpath='{.spec.replicas}'
        outputContains: "3"
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
