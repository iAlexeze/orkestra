# External 03 — Image Signing

The signing call only fires when `spec.image` changes. After a successful sign, `status.signedImage` is set to the current image — the next reconcile sees they match and skips the call entirely. No annotations, no counters: a status field and a `when:` condition do all the work.

**What you learn:** the idiomatic "call once per spec field change" pattern, how `status.*` fields drive `when:` conditions on external calls, why this beats `once: true`.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
✓ webapp
    kind: WebApp
    group: demo.orkestra.io / version: v1 / plural: webapps
    mode: dynamic / workers: 2 / resync: 30s
```

---

## Step 2 — Start the operator

`--dev-server` starts a mock HTTP server on `:9999` — no signing service or token needed. It responds to `POST /sign` with `{"signed":true}`, ignoring the body and any auth header:

```bash
ork run --dev-server
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **webapp-image-signing**, then select the **WebApp** CRD.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

The operator signs `nginx:1.25`, sets `status.signedImage`, and creates the Deployment.

```bash
kubectl get webapp my-app -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  signedImage: nginx:1.25
  lastExternalStatus: "200"
```

---

## Step 5 — Observe skip on next reconcile

Wait 30s for the next resync. Check the operator logs — no sign call is made because `status.signedImage == spec.image`.

---

## Step 6 — Change the image

```bash
kubectl patch webapp my-app --type=merge -p '{"spec":{"image":"nginx:1.26"}}'
```

The operator detects `signedImage != spec.image`, signs the new image, and updates the Deployment. Only this one reconcile makes the signing call.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
