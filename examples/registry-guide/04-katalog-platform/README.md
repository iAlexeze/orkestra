# 04 — Katalog: Platform admission policies

Extend the WebApp operator with platform admission policies by importing a second motif alongside the resource motif. The same Katalog now enforces image registry rules and replica bounds at admission time and on every reconcile cycle.

---

## Files

| File | Purpose |
|------|---------|
| `crd.yaml` | PlatformApp CRD schema |
| `katalog.yaml` | Imports web-service + platform-admission motifs |
| `cr.yaml` | Compliant CR — image from allowed registry, team label set |
| `cr-denied.yaml` | Non-compliant CR — image from public registry (will be rejected) |
| `simulate.yaml` | Gate: compliant CR creates Deployment + Service |
| `simulate-denied.yaml` | Gate: non-compliant CR is denied at reconcile time — no resources created |
| `e2e.yaml` | Gate: full cluster run — Deployment ready, Service present, cleanup verified |

---

> **Before you start:** If `ORK_MOTIFS_REGISTRY` and `ORK_REGISTRY` are not set from step 01, export them now (see [01-motifs](../01-motifs/README.md#push-to-the-registry)). Then run `ork patterns` to get the exact OCI references for `web-service` and `platform-admission` and update the imports in [katalog.yaml](katalog.yaml).

---

## Template, validate and simulate

```bash
ork template
ork validate
ork simulate
ork simulate -f simulate-denied.yaml
```

`simulate.yaml` proves the compliant CR creates a Deployment and Service. `simulate-denied.yaml` proves the non-compliant CR is denied at reconcile time — no resources are created and the reconciler surfaces the violation. This is the key property of step 04: the reconciler enforces policy on every cycle, not just at admission. The CR is stored, but nothing is provisioned until it complies.

Run `--full` to see the derived RBAC before generating your bundle:

```bash
ork validate --full
```

Only a `runtime` section appears — no `gateway`. Gateway RBAC is only generated when the gateway is enabled. That happens in the next step.

---

## How multiple motifs compose

A single CRD entry can import as many motifs as needed. Resources from all motifs are merged; admission rules from all motifs are union'd.

```yaml
imports:
  - motif: oci://ghcr.io/myorg/motifs/web-service:v1.0.0
    with: { ... }      # provides Deployment, Service, Ingress

  - motif: oci://ghcr.io/myorg/motifs/platform-admission:v1.0.0
    with:              # provides validation + mutation rules only
      allowedRegistry: "myregistry.example.com/"
      maxReplicas: "10"
```

The platform-admission motif has `resources: {}` — it contributes no Kubernetes resources, only policy. One motif owns infrastructure; another owns governance.

---

## Deploy

**1. Apply the CRD**

```bash
kubectl apply -f crd.yaml
```

**2. Generate and apply the bundle**

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
```

**3. Install Orkestra**

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --wait --timeout 120s
```

**4. Apply the compliant CR**

```bash
kubectl apply -f cr.yaml
kubectl get deploy
# NAME              READY   UP-TO-DATE   AVAILABLE   AGE
# my-platform-app   2/2     2            2           85s
```

---

## Observing the deny rule

Orkestra enforces admission rules at two points:

- **Apply time** — synchronous rejection via webhook (requires Gateway, introduced in step 05)
- **Reconcile time** — every reconcile cycle, before any child resource is created

This step runs without a Gateway, so the CR is admitted by Kubernetes. The Runtime then enforces the rules on the first reconcile and blocks resource creation.

```bash
kubectl apply -f cr-denied.yaml
# platformapp.rkguide.demo/my-platform-app-denied created

kubectl get deploy
# NAME              READY   UP-TO-DATE   AVAILABLE   AGE
# my-platform-app   2/2     2            2           2m6s
# (my-platform-app-denied has no Deployment — child resources were never created)
```

The violation is only visible in the Runtime logs. The runtime runs as two pods with leader election — reconcile logs come from whichever pod holds the lease:

```bash
kubectl get pods -n orkestra-system
# NAME                                READY   STATUS    RESTARTS   AGE
# orkestra-runtime-6d9f7b4c8-abc12   1/1     Running   0          5m
# orkestra-runtime-6d9f7b4c8-xyz99   1/1     Running   0          5m

kubectl logs -n orkestra-system <pod-name> | grep "my-platform-app-denied"
```

```json
{"level":"warn","field":"metadata.labels.team","message":"declare metadata.labels.team for cost attribution and alert routing","resource":"default/my-platform-app-denied","message":"reconcile validation: warn"}
{"level":"error","error":"validation denied: field \"spec.image\": image must be pulled from the internal registry (ghcr.io/orkspace/) (got \"docker.io/nginx:1.25\")","gvk":"rkguide.demo/v1, Kind=PlatformApp","key":"default/my-platform-app-denied","message":"reconcile failed"}
```

Two things happen in order: the missing `team` label triggers a **warn** (logged, not blocking), then the disallowed image triggers an **error** (reconcile fails, no resources created).
The control center also surfaces this. Port-forward and open the dashboard:

```bash
ork proxy
# open http://localhost:8081
```

`my-platform-app` shows as **Running**. `my-platform-app-denied` shows as **Pending** — the reconcile loop keeps retrying and failing. Click on the CRD in the dashboard to expand the detail view. The Last Error field shows the exact violation:

```
Last Error
validation denied: field "spec.image": image must be pulled from the internal registry (ghcr.io/orkspace/) (got "docker.io/nginx:1.25")
```

Admission webhook enforcement (synchronous rejection at `kubectl apply` time) is introduced in the next step when the Gateway is deployed.

---

## Push to the registry

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push .
```

Simulate and e2e run automatically before the artifact is published. Gate results are written as OCI annotations — visible to any consumer via `ork inspect platform-app-operator:v1.0.0`.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next step

→ [05-komposer](../05-komposer/README.md) — compose multiple Katalogs into one Runtime with Gateway and security features

