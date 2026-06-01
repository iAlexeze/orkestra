# External 03 — Image Signing

The signing call only fires when `spec.image` changes — and only when the image has not already been signed or rejected. After a successful sign, `status.signedImage` is written and subsequent reconciles skip the call entirely. After a rejection, `status.rejectedImage` is written and the call is also skipped — no repeated API calls for an image the signing service has already refused. The Deployment is only created or updated once the image is confirmed signed.

**What you learn:** gating expensive API calls with status fields and `when:` conditions, how rejection state prevents unnecessary retries, how the Deployment gate enforces the signing requirement, and how status surfaces the full picture without writing Go code.

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

`--dev-server` starts a mock HTTP server on `:9999`. `POST /sign` accepts any image except `nginx:not-secure`, which it rejects with 403 — simulating a signing policy violation. No `IMAGE_SIGNING_TOKEN` needed:

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

## Step 4 — Apply the reject CR

Start with an image the signing service refuses:

```bash
kubectl apply -f cr-reject.yaml
```

Wait one reconcile (~30s). The operator calls `POST /sign` with `nginx:not-secure` — the dev server returns 403.

**Check status:**

```bash
kubectl get webapp my-app -o yaml | grep -A12 "status:"
```

Expected:
```yaml
status:
  phase: SigningRejected
  rejectedImage: nginx:not-secure
  lastSigningStatus: "403"
  lastSigningError: "expected status 200, got 403"
```

**Check whether a Deployment was created:**

```bash
kubectl get deploy
```

Expected: nothing. The Deployment gate (`when: signedImage == spec.image`) blocked creation — the image was never confirmed safe, so the operator never touched the cluster resource.

**Check the metrics** to see how many times the signing service was called:

```bash
curl -s localhost:8080/metrics | grep external_call
```

You will see exactly **1 call** — the initial reconcile that hit the 403. The status patch from that reconcile wrote `rejectedImage` before the next reconcile ran, so the gate was already closed. Every subsequent reconcile — including resyncs — skips the call entirely. This is the point of the pattern: the gate does the work so the API doesn't have to.

**After the next resync** — the signing call is NOT retried. The operator reconciles without making any API calls and stays in `SigningRejected`. A 5xx would have left the gate open and retried naturally — but 4xx is a definitive policy decision, so the call is permanently suppressed for this image.

---

## Step 5 — Fix the image

Patch the CR to a trusted image:

```bash
kubectl patch webapp my-app --type=merge -p '{"spec":{"image":"nginx:1.25"}}'
```

Now `spec.image != status.rejectedImage` AND `spec.image != status.signedImage` — both gates are open. The signing call fires. The dev server returns 200.

`status.signedImage` is written. The Deployment is created.

```bash
kubectl get webapp my-app -o yaml | grep -A12 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  signedImage: nginx:1.25
  rejectedImage: nginx:not-secure
  lastSigningStatus: "200"
  lastSigningError: ""
```

```bash
kubectl get deploy
# NAME      READY   UP-TO-DATE   AVAILABLE
# my-app    1/1     1            1
```

---

## Step 6 — Observe skip on next reconcile

Wait 30s for the next resync. `signedImage == spec.image` — the signing call is skipped. No API call, no churn. Check the operator logs to confirm.

---

## Step 7 — Change the image

```bash
kubectl patch webapp my-app --type=merge -p '{"spec":{"image":"nginx:1.26"}}'
```

`spec.image` differs from both `signedImage` and `rejectedImage` — signing fires for the new image. On success, `signedImage` updates and the Deployment rolls to `nginx:1.26`. Only this reconcile makes the signing call.

---

## How the gates work

```
reconcile fires
  └── signedImage == spec.image?     → skip call entirely, Deployment is up to date
  └── rejectedImage == spec.image?   → skip call entirely, phase stays SigningRejected
  └── neither?                       → call signing service
        ├── 200 → write signedImage, create/update Deployment, phase = Ready
        ├── 4xx → write rejectedImage, Deployment untouched, phase = SigningRejected
        │         next reconcile skips the call (gate is closed)
        └── 5xx → signing service unavailable, phase = SigningUnavailable
                  next reconcile retries (gate stays open — transient failure)
```

The reconcile loop is the retry mechanism. 4xx closes the gate because it is a policy decision — the same image will always be refused. 5xx leaves the gate open because the service may recover. No annotations, no finalizers, no custom Go state machine.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## E2E

Run the full lifecycle — deploys the mock dev server, starts the operator, applies the CR with `nginx:1.25`, asserts the Deployment is created and `status.signedImage` is set, then tears down:

```bash
ork e2e --dev-server
```

CRs use the in-cluster address defined in [cr-e2e.yaml](./cr-e2e.yaml). This runs everything in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Deployment created after image signing succeeds
    after: cr-applied
    resources:
      - kind: Deployment
        name: my-app
        ready: true
  - name: Status records verified image
    after: cr-applied
    commands:
      - run: kubectl get webapp my-app -o jsonpath='{.status.signedImage}'
        outputContains: nginx:1.25
```
