# External 06 — SBOM and Signature Verification

Two chained calls run once per image change: first fetch the SBOM vulnerability report, then verify the cosign signature. The Deployment is only created after both pass. A vulnerable or unsigned image writes `status.rejectedImage` and suppresses retries until `spec.image` changes.

**What you learn:** chaining two calls where the second is gated on the first, handling two distinct rejection reasons (vulnerability vs missing signature), and how a single `rejectedImage` gate closes both call paths.

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Start the operator

```bash
ork run --dev-server
```

Dev server routes used:
- `GET /sbom/:image` — `nginx:vulnerable` returns `critical:3, high:12, clean:false`. `nginx:unknown` → 404. All others clean.
- `POST /cosign/verify` — `nginx:unsigned` → 403. All others verified.

---

## Step 3 — Open the Control Center

```bash
ork control   # http://localhost:8081 → orkestra/orkestra
```

---

## Step 4 — Apply the unsigned CR

```bash
kubectl apply -f cr-unsigned.yaml
```

Wait one reconcile (~30s). The SBOM is clean for `nginx:unsigned` — call 1 passes. Cosign returns 403 — call 2 rejects.

```bash
kubectl get webapp my-app -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: SignatureRejected
  rejectedImage: nginx:unsigned
  lastError: "expected status 200, got 403"
```

No Deployment. `rejectedImage` is written — both calls are suppressed until `spec.image` changes.

You should see exactly one call in the logs:

```json
{
  "level":"warn","request_id":"45a61565-1222-46bf-8d56-b1ae13e3fc9c",
  "crd":"demo.orkestra.io/v1, Kind=WebApp","resource":"default/my-app",
  "call":"cosign","url":"http://localhost:9999/cosign/verify",
  "error":"HTTP 403","time":1780238206,"message":"external call failed"
}
```

---

## Step 5 — Apply the vulnerable CR

```bash
kubectl apply -f cr-vulnerable.yaml
```

Wait one reconcile. The SBOM returns `clean:false` for `nginx:vulnerable` — call 1 produces a body the cosign gate rejects. The cosign call is never made.

```bash
kubectl get webapp my-app -o yaml | grep -A6 "status:"
```

Expected:
```yaml
status:
  phase: VulnerabilityRejected
  rejectedImage: nginx:vulnerable
```

No Deployment. `rejectedImage` is written via the SBOM body condition — no second call needed.

---

## Step 6 — Fix the image

```bash
kubectl patch webapp my-app --type=merge -p '{"spec":{"image":"nginx:1.25"}}'
```

`spec.image` differs from `rejectedImage`. Both gates open. SBOM returns clean, cosign returns 200. `verifiedImage` is written. Deployment is created.

```bash
kubectl get webapp my-app -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  verifiedImage: nginx:1.25
  rejectedImage: nginx:vulnerable
```

---

## How the gates work

```
reconcile fires
  └── verifiedImage == spec.image?    → skip both calls, Deployment up to date
  └── rejectedImage == spec.image?    → skip both calls, phase unchanged
  └── neither?
        └── Call 1: GET /sbom/:image
              ├── clean:false → write rejectedImage, skip cosign, phase = VulnerabilityRejected
              └── clean:true  → proceed to cosign
                    └── Call 2: POST /cosign/verify
                          ├── 200 → write verifiedImage, create Deployment, phase = Ready
                          ├── 4xx → write rejectedImage, no Deployment, phase = SignatureRejected
                          └── 5xx → transient, gate stays open, next reconcile retries
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
