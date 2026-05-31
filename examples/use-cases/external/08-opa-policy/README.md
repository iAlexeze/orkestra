# External 08 — OPA Policy Enforcement

On every reconcile the operator submits the CR identity (name, namespace, image) to an OPA endpoint and gates the Deployment on the decision. A deny response surfaces in `status.phase` with the full OPA output for observability. No admission webhooks, no Go hooks — the policy check is declarative.

**What you learn:** enforcing organisational policy via an external decision service, surfacing structured denial reasons in status, and how `continueOnError: false` ensures the Deployment never exists without a passing check.

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

Dev server route: `POST /v1/data/:policy`
- Input `name` contains `"deny"` → `{"result": {"allow": false, "deny": true, "reason": "..."}}`
- Input `namespace` is `"forbidden"` → deny
- Everything else → `{"result": {"allow": true, "deny": false}}`

---

## Step 3 — Open the Control Center

```bash
ork control   # http://localhost:8081 → orkestra/orkestra
```

---

## Step 4 — Apply the denied CR

```bash
kubectl apply -f cr-denied.yaml
```

The CR name is `deny-my-app` — OPA denies it because the name contains `deny`.

Wait one reconcile (~30s):

```bash
kubectl get webapp deny-my-app -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: PolicyDenied
  policyDecision: '{"result":{"allow":false,"deny":true,"reason":"resource name contains a denied prefix — rename the CR"}}'
```

No Deployment created. The raw OPA response is in `policyDecision` — visible in the Control Center.

---

## Step 5 — Apply the allowed CR

```bash
kubectl apply -f cr.yaml
```

Wait for deployment rollout to complete, then run: 
```bash
kubectl get webapp my-app -o yaml | grep -A6 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  policyDecision: '{"result":{"allow":true,"deny":false,"reason":""}}'
```

---

## Step 6 — Patch to a denied name

```bash
# Not directly possible — name is immutable. Delete and re-create to test:
kubectl delete webapp my-app --ignore-not-found
kubectl apply -f cr-denied.yaml
```

Or observe that `cr-denied.yaml` stays in `PolicyDenied` every reconcile — OPA is checked on every cycle.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
