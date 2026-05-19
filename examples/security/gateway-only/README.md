# Gateway-Only: Deletion and Namespace Protection

This example demonstrates running only the Orkestra gateway process — no runtime reconciler — to protect arbitrary namespaces from accidental deletion. No CRDs are managed; the gateway registers admission webhooks that guard the `production` and `staging` namespaces and blocks deletion of any protected resource.

**What you learn:** `--no-runtime` bundle generation, separating the gateway ServiceAccount/ClusterRole from the runtime, and protecting namespaces without deploying a full operator.

---

## How it works

The Katalog declares zero CRDs but enables `deletionProtection` and `namespaceProtection`. When the gateway starts it:

1. Creates a `ValidatingWebhookConfiguration` for deletion protection
2. Labels the Orkestra namespace so the webhook's `ObjectSelector` matches
3. Blocks DELETE requests for the listed namespaces (`production`, `staging`)

Because there is no runtime, no reconciler runs and no ConfigMap is needed for CRD definitions.

---

## Steps

### Step 1 — Validate

```bash
ork validate -f katalog.yaml
```

### Step 2 — Generate and apply the bundle

```bash
ork generate bundle -f katalog.yaml --no-runtime | kubectl apply -f -
```

`--no-runtime` excludes the runtime ServiceAccount and ClusterRole. The gateway still gets its own `orkestra-gateway` ServiceAccount and ClusterRole with the minimal permissions needed to manage webhook configurations and TLS secrets.

### Step 3 — Deploy the gateway (runtime disabled)

```bash
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set runtime.enabled=false \
  --set gateway.enabled=true \
  --wait \
  --timeout 120s
```

### Step 4 — Verify protection is active

Try deleting a protected namespace — the request should be denied:

```bash
kubectl delete ns production
# Error from server: admission webhook "orkestra-deletion-protection.orkestra.io" denied the request
```

### Step 5 — Cleanup

To remove protection, disable deletion protection in `katalog.yaml`, regenerate the bundle, and restart the gateway:

```bash
# Edit katalog.yaml: set deletionProtection.enabled: false and namespaceProtection.enabled: false
ork generate bundle -f katalog.yaml --no-runtime | kubectl apply -f -
kubectl rollout restart deployment/orkestra-gateway -n orkestra-system
```
