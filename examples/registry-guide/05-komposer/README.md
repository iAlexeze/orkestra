# 05 — Komposer: security, gateway, and deletion protection

A Komposer merges any number of Katalogs into a single Orkestra binary. This step adds the Gateway — enabling synchronous admission enforcement and deletion protection across both operators.

---

## Files

| File | Purpose |
|------|---------|
| `komposer.yaml` | Merges webapp, cache, and platform-app katalogs; enables gateway, admission, deletion protection |

---

> **Before you start:** If `ORK_REGISTRY` is not set, export it now (see [01-motifs](../01-motifs/README.md#push-to-the-registry)). Replace `myorg` with your actual registry path throughout this example in [`komposer.yaml`](komposer.yaml).



---

## What changes from step 04

In step 04, admission rules were enforced only at reconcile time — the CR was stored before the reconciler rejected it. Here, the Gateway registers webhook endpoints with the Kubernetes API server. `kubectl apply` is intercepted before etcd storage. A bad CR never reaches the cluster.

---

## Validate before you apply anything

Before generating or applying anything, run validate to see exactly what you are authorizing:

```bash
ork validate
```

```text
● cache
    kind: Cache / group: rkguide.demo / version: v1 / plural: caches / scope: Namespaced
    mode: dynamic / workers: 2 / resync: 30s
    protection: 🛡️  full (CRD + CRs)

● platformapp
    kind: PlatformApp / group: rkguide.demo / version: v1 / plural: platformapps / scope: Namespaced
    mode: dynamic / workers: 2 / resync: 30s
    protection: 🛡️  full (CRD + CRs)

● webapp
    kind: WebApp / group: rkguide.demo / version: v1 / plural: webapps / scope: Namespaced
    mode: dynamic / workers: 4 / resync: 30s
    protection: 🛡️  full (CRD + CRs)

3 CRDs valid (0 built-in, 3 custom)
```

Then run `--full`. This is the first step in the guide where the gateway section appears — because this is the first step where the gateway is enabled:

```bash
ork validate --full
```

```text
...

runtime
  rkguide.demo         webapps         get list watch create update patch delete
  rkguide.demo         caches          get list watch create update patch delete
  apps                 deployments     get list watch create update patch delete
  ...

gateway
  admissionregistration.k8s.io  validatingwebhookconfigurations  get list watch create update patch delete
  core                          secrets                          get list watch create update patch delete
  core                          namespaces                       get patch
  ...
```

Gateway gets exactly what the declared features need: `validatingwebhookconfigurations` because admission is enabled, `secrets` for TLS certificate storage, `namespaces` for deletion-protection labeling. Nothing else.

Runtime and gateway run as separate service accounts with separate ClusterRoles. A compromise of one cannot reach the other. Run `ork validate --full` before generating your bundle — what you see here is exactly what gets applied.

---

## Deploy with Gateway

**1. Apply all CRDs**

```bash
kubectl apply -f ../02-katalog-api/crd.yaml
kubectl apply -f ../03-katalog-cache/crd.yaml
kubectl apply -f ../04-katalog-platform/crd.yaml
```

**2. Generate and apply the bundle**

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
```

**3. Install Orkestra with Gateway**

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set gateway.enabled=true \
  --wait --timeout 120s
```

TLS certificates for the webhook server are generated and rotated automatically. You can supply your own by setting `TLS_CERT` and `TLS_KEY` in the Helm values.

**4. Verify both operators are running**

```bash
kubectl rollout status deployment/orkestra-runtime -n orkestra-system
kubectl rollout status deployment/orkestra-gateway -n orkestra-system
```

**5. Create instances of all three CRDs**

```bash
kubectl apply -f ../02-katalog-api/cr.yaml
kubectl apply -f ../03-katalog-cache/cr.yaml
kubectl apply -f ../04-katalog-platform/cr.yaml
```

---

## Admission enforcement — synchronous rejection

Apply the non-compliant CR from step 04:

```bash
kubectl apply -f ../04-katalog-platform/cr-denied.yaml
```

With the webhook active, the request is rejected before etcd storage:

```text
Warning: orkestra: field "metadata.labels.team": declare metadata.labels.team for cost attribution and alert routing
Error from server: error when creating "../04-katalog-platform/cr-denied.yaml": admission webhook "validate.orkestra.orkspace.io" denied the request: 

[Orkestra Validation] validation failed

PlatformApp/my-platform-app-denied/default was blocked due to the following policies:
 field "spec.image": image must be pulled from the internal registry (ghcr.io/orkspace/) (got: "docker.io/nginx:1.25")
```

The CR is never created. Nothing to clean up.

---

## Deletion protection

The same webhook intercepts every DELETE request on a protected resource — a CR, the CRD itself, or any Orkestra component. Try each:

```bash
# A CR
kubectl delete webapp my-webapp

# The CRD
kubectl delete crd webapps.rkguide.demo

# An Orkestra component
kubectl delete deployment orkestra-gateway -n orkestra-system
```

All three produce the same message:

```text
Error from server: admission webhook "protect.resources.orkestra.orkspace.io" denied the request:

[Orkestra Security] The resource is protected from deletion.

To remove protection:
- Set security.deletionProtection.enabled: false in the Katalog.
- Redeploy Orkestra Gateway.
- Retry the deletion.
```

Nothing is deleted. There is nothing to clean up.

---

## Inspect the live state

```bash
kubectl port-forward svc/orkestra-cc -n orkestra-system 8081:8081
# open http://localhost:8081
```

The Control Center shows reconcile metrics, webhook activity, and per-CRD health for both WebApp and Cache operators.

---

## Clean uninstall

Deletion protection is active for both CRDs. `kubectl delete` and `helm uninstall` are both blocked by the webhook — Orkestra's own infrastructure is protected too, so Helm cannot remove the deployment. You must disable protection first.

**1. Disable deletion protection**

In [`komposer.yaml`](komposer.yaml), set `security.deletionProtection.enabled: false`, then regenerate and reapply:

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
```

Before restarting, confirm both webhooks are present:

```bash
kubectl get validatingwebhookconfiguration
# NAME                           WEBHOOKS   AGE
# orkestra-admission-validation  1          5m
# orkestra-deletion-protection   2          5m
```

Restart the Gateway so it picks up the updated ConfigMap:

```bash
kubectl rollout restart deployment/orkestra-gateway -n orkestra-system
kubectl rollout status deployment/orkestra-gateway -n orkestra-system --timeout=60s
```

The new gateway pod starts the housekeeper, which reads the katalog, sees `deletionProtection.enabled: false`, and removes `orkestra-deletion-protection`. The admission webhook stays active — admission is still declared, so the housekeeper keeps it alive.

> **Note:** The housekeeper only runs when the gateway is on — which is exactly when it is needed. It manages the webhook configurations the gateway registered, and there is nothing to manage when there is no gateway.

Confirm the result:

```bash
kubectl get validatingwebhookconfiguration
# NAME                           WEBHOOKS   AGE
# orkestra-admission-validation  1          6m
```

The deletion-protection webhook is gone. Admission stays. The housekeeper removes exactly what the katalog no longer declares, and nothing else.

**2. Run cleanup**

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next step

→ [06-pattern-zoo](../06-pattern-zoo/README.md) — compose all seven official Orkestra Registry patterns in one Komposer
