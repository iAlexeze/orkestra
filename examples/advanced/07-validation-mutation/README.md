# 07 — Validation and Mutation

Admission-time policy without a webhook server. Defaults applied before
validation sees the object. The same rules enforced at admission and at
reconcile time from a single Katalog declaration.

**What you learn:** `validation` with `deny` and `warn`, `mutation` with
`default` and `override`, `mutateFirst`, two enforcement boundaries,
in-cluster deployment with `kubectl apply`.

**Builds on:** [06 — Basic Komposer](../../intermediate/06-komposer-basic/README.md)

---

## Two enforcement boundaries

Orkestra enforces the same declared rules at two points:

**Admission time** (`ENABLE_WEBHOOKS=true`) — synchronously during `kubectl apply`.
A `deny` rule rejects the request before the object is stored. The user sees
the error immediately in their terminal.

**Reconcile time** (always) — on every reconcile cycle. Catches CRs that
existed before the rules were added, and provides enforcement even when
webhooks are disabled.

Declare once. Enforced at both points.

---

## Mutation fires before validation

At admission time, the Kubernetes API server always calls the mutating webhook
before the validating webhook. This means:

1. Mutation runs → `spec.replicas` default `"2"` is applied if absent
2. Validation runs → `spec.replicas > 0` check now sees `"2"` — passes

`mutateFirst: true` in the Katalog replicates this ordering at reconcile time.

---

### 1. Create namespaces and the source Secret


## Step 1 — Deploy Orkestra in-cluster

```bash
# Install the CRD
kubectl apply -f crd.yaml
```

```bash
# Create the 'orkestra-system' namespace
kubectl apply -f namespace.yaml
```

### Apply the Katalog ConfigMap
```bash
kubectl apply -f orkestra-configmap.yaml
```

### Generate TLS certs for the webhook server.
For this example, generate self-signed ones:

```bash
# Generate certs (development only — use cert-manager in production)
chmod +x ../../installation/generate-certs.sh && \
../../installation/generate-certs.sh

# This creates a secret 'orkestra-tls' with certificates for webhook support
```

### Deploy Orkestra
```bash
kubectl apply -f ../../installation/install-webhook-support.yaml

# Wait for it to be ready
kubectl wait --for=condition=available deployment/orkestra \
  -n orkestra-system --timeout=60s
```

You should see the following endpoints as part of orkestra's startup

```text
Orkestra Endpoints:
- Health:   /health
- Ready:    /ready
- Metrics:  /metrics

Webhook Endpoints:
- Muatation:  /mutate
- Validation:  /validate
- Service Name: orkestra
- Service Namespace: orkestra-system
- Failure Policy: Ignore

- Conversion: /convert

Katalog Endpoints:
- Katalog:  /katalog
  - Website (website):  /katalog/website
  - Website (website):  /katalog/website/health
```

---

## Step 2 — Apply the valid CR

```bash
kubectl apply -f cr-valid.yaml
```

Note: `replicas` and `port` are not declared in the CR. Mutation will apply
defaults before reconcile creates resources.

```bash
# Mutation applied the defaults — spec now has replicas and port
kubectl get website my-site -o jsonpath='{.spec}'
# {"environment":"production","image":"myorg/nginx:1.25","port":"8080","replicas":"2"}
```

Check status:

```bash
kubectl get website my-site -o yaml | grep -A12 "status:"
```

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
  phase: Running
  environment: production
  observedReplicas: "2"
```

---

## Step 3 — Test the deny rule

```bash
kubectl apply -f cr-invalid.yaml
```

**With webhooks enabled:**
```
Error from server: admission webhook "validate.orkestra.konductor.io" denied the request:
[orkestra] validation failed: field "spec.image": images must come from the internal registry (myorg/)
```

The CR is never stored. Nothing to clean up.

**Without webhooks:**

The CR is stored, but reconcile halts:

```bash
kubectl get website bad-site -o jsonpath='{.status.conditions[0]}'
# {"type":"Ready","status":"False","reason":"ReconcileError",
#  "message":"validation denied: field \"spec.image\": images must come..."}
```

The Ready condition surfaces the validation error directly in status.

---

## Step 4 — Test the warn rule

Apply a CR without `spec.environment`:

```bash
kubectl apply -f - <<EOF
apiVersion: demo.orkestra.io/v1alpha1
kind: Website
metadata:
  name: warn-site
  namespace: default
spec:
  image: myorg/nginx:1.25
  replicas: 1
  port: 8080
  # environment not declared → warn rule fires
EOF
```

**With webhooks:** `kubectl apply` shows a warning but proceeds:
```
Warning: orkestra: field "spec.environment": declare spec.environment for better observability
website.demo.orkestra.io/warn-site created
```

**Mutation** applied `environment: development` as the default. The warn rule
then fired on the original (pre-mutation) object at admission time because it
checks before mutation? No — at admission time, mutation runs before validation.
The default is applied, so the warn rule actually does not fire for this CR
because `spec.environment` is now set to `development`.

The warn rule fires for CRs where `spec.environment` is genuinely absent and
no default exists — useful when `default` is deliberately not set for a field
you want to encourage but not require.

---

## Step 5 — Check the metrics

```bash
kubectl port-forward svc/orkestra 8080:8080 -n orkestra-system &

curl localhost:8080/katalog/website | jq '{
  "reconcileTotal": .reconcileTotal,
  "admission": .admission
}'
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
