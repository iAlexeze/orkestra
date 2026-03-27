# Case Study: Managing a Multi-Version Website CRD with Incompatible Schemas

This case study demonstrates how Orkestra handles a CRD with two incompatible versions—without writing conversion webhooks, without deploying separate services, and without managing TLS certificates manually.

---

## The Scenario

You manage a `Website` CRD that has evolved over time:

- **v1alpha1** (original): simple, with `image`, `replicas`, and `port` fields
- **v1** (new): adds `autoscaling` configuration, but **incompatibly** — v1alpha1 has no concept of autoscaling

This is exactly the CronJob problem from the Kubebuilder tutorial, but for a Website resource.

---

## Step 1: The CRD with Two Incompatible Versions
Examine the CRD: [multi-version-website](./multi-version-website-katalog.yaml)

**The incompatibility:** v1alpha1 has no `autoscaling` field. When converting from v1alpha1 to v1, we must provide a default. When converting from v1 to v1alpha1, we must drop the field.

---

## Step 2: Generate a Self‑Signed Certificate

Since Orkestra's conversion webhook requires HTTPS, we generate a self‑signed certificate for development:

```bash
# Generate CA key and certificate
openssl genrsa -out ca.key 2048
openssl req -new -x509 -days 365 -key ca.key -subj "/CN=Orkestra CA" -out ca.crt

# Generate server key and certificate signing request
openssl genrsa -out server.key 2048
openssl req -new -key server.key -subj "/CN=orkestra.orkestra.svc" -out server.csr

# Sign with our CA
openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key -set_serial 01 -out server.crt

# Convert to base64
cat ca.crt | base64 -w0 && echo

# Create a Kubernetes secret with the certificate
kubectl create secret tls orkestra-tls \
  --cert=server.crt \
  --key=server.key \
  --namespace orkestra
```

---

## Step 3: The Katalog with Declarative Conversion Rules

---

## Step 4: Deploy Orkestra with Conversion Enabled

```bash
# Mount the TLS secret
kubectl create secret tls orkestra-tls \
  --cert=server.crt \
  --key=server.key \
  --namespace orkestra

# Deploy Orkestra with conversion enabled
kubectl apply -f - <<EOF
EOF
```

---

## Step 5: Create Resources in Both Versions

### v1alpha1 Resource (Simple)

```yaml
# website-v1alpha1.yaml
```

### v1 Resource (With Autoscaling)

```yaml
```

---

## Step 6: Apply and Observe

```bash
# Apply the CRD
kubectl apply -f crd.yaml

# Start Orkestra
kubectl apply -f deployment.yaml

# Apply both versions
kubectl apply -f website-v1alpha1.yaml
kubectl apply -f website-v1.yaml
```

---

## What Happens Behind the Scenes

### When you apply the v1alpha1 resource:

1. **API server receives v1alpha1** → validates against v1alpha1 schema
2. **API server calls Orkestra's conversion webhook** at `https://orkestra:8443/convert`
3. **Orkestra reads the Katalog** → finds conversion rules for Website
4. **Orkestra applies the `from` path** for v1alpha1 → v1:
   - `image: "nginx:1.25"` (from spec)
   - `replicas: 2` (from spec)
   - `port: 80` (from spec)
   - `autoscaling: {enabled: false, minReplicas: 1, maxReplicas: 1}` (default)
5. **Orkestra returns the converted v1 resource** to the API server
6. **API server stores the v1 resource** in etcd
7. **Orkestra's v1 reconciler receives the resource** (as v1) and creates:
   - Deployment (with 2 replicas)
   - HorizontalPodAutoscaler (disabled, but created)

### When you apply the v1 resource:

1. **API server receives v1** → validates against v1 schema
2. **No conversion needed** (storage version)
3. **API server stores the v1 resource** directly
4. **Orkestra's v1 reconciler receives the resource** and creates:
   - Deployment (with 5 replicas)
   - HorizontalPodAutoscaler (enabled, min 3, max 10)

### When you query the v1alpha1 resource:

```bash
# This returns the resource as v1alpha1, with autoscaling dropped
ork get website my-blog --version v1alpha1
```

Output:
```yaml
apiVersion: demo.orkestra.io/v1alpha1
kind: Website
metadata:
  name: my-blog
spec:
  image: nginx:1.25
  replicas: 2
  port: 80
  # autoscaling is gone — as expected
```

---

## The Result

| What | Traditional Approach | Orkestra Approach |
|------|---------------------|-------------------|
| **Conversion code** | Write Go conversion functions | Write YAML in Katalog |
| **Webhook server** | Deploy separate pod, service, TLS | Same pod as Orkestra |
| **TLS certificates** | Manage with cert-manager | One‑time self‑signed for dev, or user‑provided for prod |
| **Version handling** | Reconciler must know both versions | Separate reconciler configs per version |
| **User sees** | Storage version (v1) with autoscaling default | Original version (v1alpha1) without autoscaling |
| **Upgrade path** | Complex, requires webhook updates | Update Katalog, restart Orkestra |

---

## Summary

This case study demonstrates:

1. **Declarative conversion** — users define conversion rules in YAML, not Go
2. **No separate webhook** — Orkestra's existing HTTPS server handles `/convert`
3. **Version‑aware reconciliation** — different templates for different versions
4. **Version‑preserving CLI** — `ork get --version` shows the original resource
5. **Zero‑code version management** — the entire multi‑version operator is defined in a single Katalog

**This is the "your CRD is enough" philosophy applied to versioning.** 🎼