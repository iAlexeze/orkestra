# Multi-Region Map Demo — Per-Region Config with forEach

This example deploys a real web application into multiple regions from a single
CR. Each region gets its own Deployment and Service, with independent replica
count and port configured directly in the CR — no Go code, no Helm chart, no
templating engine.

This is the **map forEach** pattern. Unlike the list forEach example (which
deploys the same config everywhere), here every region entry is a map with
its own `replicas` and `port`. Orkestra expands them at reconcile time.

---

## What the app shows

The demo app reads two environment variables Orkestra injects from the CR:

| Variable | Source in CR | Purpose |
|---|---|---|
| `REGION` | map key (`us-east-1`, `eu-west-1`, ...) | shown in the page |
| `PORT` | `regions.<name>.port` or `defaultPort` | port the app listens on |

After port-forwarding to any region's Service you will see:

```
Orkestra Multi-Region Demo
Region: ap-southeast-1
This instance is running in ap-southeast-1.
```

Each region's pod shows its own name. The replicas and port are different
per-region — Orkestra reads them from the CR map and wires everything up.

---

## How the map forEach works

The CR declares regions as a map instead of a list:

```yaml
spec:
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

The Katalog template uses `.item` for the region name and `.value.*` for the
per-region fields:

```yaml
deployments:
  - name: "{{ .metadata.name }}-{{ .item }}"
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

Orkestra expands this into three fully-resolved Deployment declarations before
a single API call is made. The `or` fallback means a region entry can omit
`replicas` or `port` and the CR-level defaults apply.

---

## Step 1 — Apply the CRD

```bash
kubectl apply -f examples/multi-region-map/crd.yaml
```

This registers `MultiRegionApp` in your cluster. Check it appeared:

```bash
kubectl get crd multiregionapps.advanced.orkestra.io
```

---

## Step 2 — Run Orkestra

Choose the path that matches your setup.

### Option A — Dev mode (quickest)

Runs Orkestra directly against your current kubeconfig. No cluster deployment needed.

```bash
ork run -f examples/multi-region-map/katalog.yaml
```

### Option B — Helm deployment (recommended for shared clusters)

If Orkestra is already installed via Helm, generate the least-privilege RBAC,
ServiceAccount, and Katalog ConfigMap for this example:

```bash
# Full bundle — RBAC + ServiceAccount + ConfigMap
ork generate bundle \
  -f examples/multi-region-map/katalog.yaml \
  -o /tmp/multi-region-bundle.yaml

kubectl apply -f /tmp/multi-region-bundle.yaml
```

Or generate each piece separately:

```bash
# ConfigMap only — if RBAC is already applied
ork generate configmap \
  -f examples/multi-region-map/katalog.yaml \
  -o /tmp/multi-region-configmap.yaml

# RBAC only
ork generate bundle --rbac \
  -f examples/multi-region-map/katalog.yaml \
  -o /tmp/multi-region-rbac.yaml
```

## Step 3 — Open the Control Center
### If running with Helm
```bash
kubectl port-forward svc/orkestra-cc -n orkestra-system 8081:8081 &
```

Open **http://localhost:8081** and select **multi-region-demo**. You will see
the `multi-region-app` operatorBox in `started` state — no CRs yet, so no
reconciles have run.

---

### If running directly
Open the Control Center in a second terminal:

```bash
ork control start             # serves on http://localhost:8081 by default
# username:password → orkestra
ork control start --port 9090 # or on a custom port
# username:password → orkestra
```

Open **http://localhost:8081** (or whichever port you passed to `ork control start`).
Select **multi-region-demo**. You will see
the `multi-region-app` operatorBox in `started` state — no CRs yet, so no
reconciles have run.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f examples/multi-region-map/cr.yaml
```

The CR declares three regions with distinct replica counts and ports:

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

---

## Step 5 — Watch reconciliation

Orkestra reconciles immediately. Watch the resources appear:

```bash
kubectl get deployments,services -l orkestra-owner=my-multi-region
```

Expected output (within a few seconds):

```
NAME                                          READY   UP-TO-DATE   AVAILABLE
deployment.apps/my-multi-region-ap-southeast-1   2/2     2            2
deployment.apps/my-multi-region-eu-west-1        1/1     1            1
deployment.apps/my-multi-region-us-east-1        3/3     3            3

NAME                                              TYPE        CLUSTER-IP   PORT(S)
service/my-multi-region-ap-southeast-1-svc   ClusterIP   ...          8080/TCP
service/my-multi-region-eu-west-1-svc        ClusterIP   ...          8081/TCP
service/my-multi-region-us-east-1-svc        ClusterIP   ...          8080/TCP
```

Replica counts match exactly what you declared in the CR map.
`eu-west-1` runs on port `8081`. All others on `8080`.

In the **Control Center**:

1. Click **multi-region-app** → **View Resources**
2. Click `my-multi-region`
3. **Children** section shows all 6 child resources (3 Deployments + 3 Services)
4. **Status** section shows `phase: Ready` once the first replica is up

---

## Step 6 — Port-forward and see each region

Open three terminals and forward to each region's Service:

```bash
# Terminal 1 — ap-southeast-1 (port 8080)
kubectl port-forward svc/my-multi-region-ap-southeast-1-svc 8001:8080

# Terminal 2 — eu-west-1 (port 8081)
kubectl port-forward svc/my-multi-region-eu-west-1-svc 8002:8081

# Terminal 3 — us-east-1 (port 8080)
kubectl port-forward svc/my-multi-region-us-east-1-svc 8003:8080
```

Visit each in your browser:

| URL | Shows |
|---|---|
| http://localhost:8001 | `Region: ap-southeast-1` |
| http://localhost:8002 | `Region: eu-west-1` |
| http://localhost:8003 | `Region: us-east-1` |

Each pod only knows its own region because Orkestra injected `REGION={{ .item }}`
and `PORT={{ or .value.port .spec.defaultPort }}` at reconcile time. The app
itself has no region-specific logic — it just reads the environment.

---

## Step 7 — Change a region live

Edit the CR to scale `eu-west-1` up and shift its port:

```bash
kubectl edit multiregionapp my-multi-region
```

Change:
```yaml
eu-west-1:
  replicas: 3
  port: 9000
```

Orkestra reconciles within `resync: 30s` (or immediately on the edit event).
Watch the deployment scale and the Service port update:

```bash
kubectl get deployment my-multi-region-eu-west-1 -w
kubectl get svc my-multi-region-eu-west-1-svc
```

No Katalog change. No redeployment of Orkestra. The operator reflected the
new desired state from the CR map automatically.

---

## What Orkestra replaced here

Without Orkestra you would have written a reconciler that:

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var app v1alpha1.MultiRegionApp
    r.Get(ctx, req.NamespacedName, &app)

    for region, config := range app.Spec.Regions {
        deploy := buildDeployment(app.Name, region, config.Replicas, config.Port, app.Spec.Image)
        if err := r.createOrUpdate(ctx, deploy); err != nil {
            return ctrl.Result{}, err
        }
        svc := buildService(app.Name, region, config.Port)
        svc.Spec.Selector = map[string]string{"app": app.Name + "-" + region}
        if err := r.createOrUpdate(ctx, svc); err != nil {
            return ctrl.Result{}, err
        }
    }
    return ctrl.Result{}, nil
}
```

That is ~80 lines of Go including type definitions, owner references, drift
detection, and label wiring. The Katalog replaces all of it with the
`forEach` map expansion — and adds drift correction, status management, and
reconcile observability for free.
