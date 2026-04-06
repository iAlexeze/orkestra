# Admission Webhook Test

Complete end-to-end test for Orkestra's validation and mutation admission webhooks.

## What this tests

| Test | File | Expected |
|---|---|---|
| A — mutation applies defaults | `cr-missing-defaults.yaml` | Created. Stored with `replicas=2`, `logLevel=info`, `managed-by=orkestra`. Two warnings. |
| C — deny rule at admission | `cr-bad-image.yaml` | **Rejected** by webhook. Error shown immediately. |
| D — warn rules only | `cr-warn-only.yaml` | Created. Two warnings printed. Active warnings on health API. |
| E — all rules pass | `cr-valid.yaml` | Created. No errors, no warnings. |
| F — numeric deny | `cr-too-many-replicas.yaml` | **Rejected** by webhook. |

---

## Prerequisites

- Kubernetes cluster (kind, minikube, or remote)
- `kubectl` configured
- Orkestra image built and accessible
- `openssl` installed (for cert generation)

---

## Step 1: Generate TLS certificates

```bash
chmod +x generate-certs.sh
./generate-certs.sh
```

This creates `certs/tls.crt` and `certs/tls.key` and applies the
`orkestra-tls` Secret to the `orkestra` namespace.

---

## Step 2: Install the CRD

```bash
kubectl apply -f website-crd.yaml
```

Verify:
```bash
kubectl get crd websites.demo.orkestra.io
```

---

## Step 3: Create the Katalog ConfigMap

```bash
kubectl create namespace orkestra --dry-run=client -o yaml | kubectl apply -f -

kubectl create configmap orkestra-katalog \
  --from-file=katalog.yaml=katalog.yaml \
  --namespace orkestra \
  --dry-run=client -o yaml | kubectl apply -f -
```

---

## Step 4: Deploy Orkestra with webhooks enabled

```bash
kubectl apply -f deployment.yaml
```

Wait for Orkestra to be ready:
```bash
kubectl rollout status deployment/orkestra-runtime -n orkestra
```

Check the logs — Orkestra should log webhook registration:
```bash
kubectl logs -n orkestra -l app=orkestra-runtime -f
```

Look for:
```
{"level":"info","message":"admission webhooks: /validate and /mutate registered on :8443"}
{"level":"info","crds":1,"config":"orkestra-validation","message":"webhook: ValidatingWebhookConfiguration registered"}
{"level":"info","crds":1,"config":"orkestra-mutation","message":"webhook: MutatingWebhookConfiguration registered"}
```

Verify the webhook configurations were created:
```bash
kubectl get validatingwebhookconfigurations orkestra-validation
kubectl get mutatingwebhookconfigurations orkestra-mutation
```

---

## Step 5: Run the tests

### Test E — valid CR (should succeed cleanly)

```bash
kubectl apply -f cr-valid.yaml
```

Expected:
```
website.demo.orkestra.io/valid-site created
```

Verify:
```bash
kubectl get website valid-site -o yaml
kubectl get deployments -l orkestra-owner=valid-site
kubectl get services -l orkestra-owner=valid-site
```

---

### Test C — bad image (should be rejected at admission)

```bash
kubectl apply -f cr-bad-image.yaml
```

Expected:
```
Error from server: admission webhook "validate.orkestra.konductor.io" denied the request:
[orkestra] validation failed: field "spec.image": image must be from the myorg registry
— use myorg/<n>:<tag> (got: "nginx:1.25")
```

The CR is **not** stored. Verify:
```bash
kubectl get website bad-image-site 2>&1
# Error from server (NotFound): websites.demo.orkestra.io "bad-image-site" not found
```

---

### Test F — too many replicas (should be rejected at admission)

```bash
kubectl apply -f cr-too-many-replicas.yaml
```

Expected:
```
Error from server: admission webhook "validate.orkestra.konductor.io" denied the request:
[orkestra] validation failed: field "spec.replicas": replicas cannot exceed 10
— contact platform team for exceptions (got: "15")
```

---

### Test A — missing defaults (should succeed, mutation applies defaults)

```bash
kubectl apply -f cr-missing-defaults.yaml
```

Expected (two warnings, then success):
```
Warning: orkestra: field "metadata.labels.team": all Website resources should declare a team owner
Warning: orkestra: field "spec.enabled": spec.enabled should be declared explicitly
website.demo.orkestra.io/defaults-site created
```

Now verify that mutation applied the defaults **to the stored object**:
```bash
kubectl get website defaults-site -o yaml
```

You should see in the stored spec:
```yaml
spec:
  image: myorg/nginx:1.25
  port: 8080
  replicas: 2        # ← set by mutation (was absent in cr-missing-defaults.yaml)
  logLevel: info     # ← set by mutation (was absent)
metadata:
  labels:
    managed-by: orkestra  # ← set by mutation override
```

If `replicas` is `2` and `logLevel` is `info` in the stored object, **mutation worked at admission time**.

---

### Test D — warn only (should succeed with warnings)

```bash
kubectl apply -f cr-warn-only.yaml
```

Expected:
```
Warning: orkestra: field "metadata.labels.team": all Website resources should declare a team owner
Warning: orkestra: field "spec.enabled": spec.enabled should be declared explicitly
website.demo.orkestra.io/warn-only-site created
```

Check the active warnings on the health API:
```bash
kubectl port-forward svc/orkestra 8080:8080 -n orkestra &

curl -s localhost:8080/katalog/website | jq '.validation.activeWarnings'
```

Expected output:
```json
[
  {
    "cr": "warn-only-site",
    "namespace": "default",
    "field": "metadata.labels.team",
    "message": "all Website resources should declare a team owner",
    "since": "2026-03-28T..."
  },
  {
    "cr": "warn-only-site",
    "namespace": "default",
    "field": "spec.enabled",
    "message": "spec.enabled should be declared explicitly",
    "since": "2026-03-28T..."
  }
]
```

Now fix the warnings — add the team label:
```bash
kubectl patch website warn-only-site --type=merge -p '{"metadata":{"labels":{"team":"platform"}},"spec":{"enabled":true}}'
```

Wait one reconcile cycle (up to 15s), then check again:
```bash
curl -s localhost:8080/katalog/website | jq '.validation.activeWarnings'
```

The warnings for `warn-only-site` should be gone — cleared by the reconciler
when it saw the fixed CR.

---

## Step 6: Check metrics

```bash
curl -s localhost:8080/metrics | grep controller_validation
```

Expected output:
```
controller_validation_total{crd="demo.orkestra.io/v1alpha1, Kind=Website",result="denied"} 2
controller_validation_total{crd="demo.orkestra.io/v1alpha1, Kind=Website",result="passed"} 2
controller_validation_total{crd="demo.orkestra.io/v1alpha1, Kind=Website",result="warned"} 2

controller_validation_violations_total{action="deny",crd="...",field="spec.image",rule="prefix"} 1
controller_validation_violations_total{action="deny",crd="...",field="spec.replicas",rule="max"} 1
controller_validation_violations_total{action="warn",crd="...",field="metadata.labels.team",rule="exists"} 2
controller_validation_violations_total{action="warn",crd="...",field="spec.enabled",rule="exists"} 2
```

And for mutation:
```bash
curl -s localhost:8080/metrics | grep controller_mutation
```

```
controller_mutation_total{crd="demo.orkestra.io/v1alpha1, Kind=Website"} 1
controller_mutation_applied_total{crd="...",field="spec.replicas",type="default"} 1
controller_mutation_applied_total{crd="...",field="spec.logLevel",type="default"} 1
controller_mutation_applied_total{crd="...",field="metadata.labels.managed-by",type="override"} 3
```

---

## What to look for in Orkestra logs

When you apply `cr-bad-image.yaml`:
```json
{"level":"info","kind":"Website","name":"bad-image-site","denials":1,"warnings":0,"message":"admission/validate: rejected"}
```

When you apply `cr-missing-defaults.yaml`:
```json
{"level":"info","kind":"Website","name":"defaults-site","field":"spec.replicas","was":"","now":"2","type":"default","message":"admission/mutate: rule applied"}
{"level":"info","kind":"Website","name":"defaults-site","field":"spec.logLevel","was":"","now":"info","type":"default","message":"admission/mutate: rule applied"}
{"level":"info","kind":"Website","name":"defaults-site","field":"metadata.labels.managed-by","was":"","now":"orkestra","type":"override","message":"admission/mutate: rule applied"}
{"level":"info","kind":"Website","name":"defaults-site","changes":3,"message":"admission/mutate: defaults applied"}
```

---

## Cleanup

```bash
kubectl delete -f cr-valid.yaml
kubectl delete -f cr-missing-defaults.yaml
kubectl delete -f cr-warn-only.yaml
kubectl delete -f deployment.yaml
kubectl delete validatingwebhookconfigurations orkestra-validation
kubectl delete mutatingwebhookconfigurations orkestra-mutation
kubectl delete -f website-crd.yaml
```
