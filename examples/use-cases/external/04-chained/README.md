# External 04 — Chained Calls

Two calls run sequentially. The first fetches a short-lived auth token. The second uses that token in its `Authorization` header to call a protected API. The resolver is updated after each call — later calls can reference earlier results via template expressions in their `url:`, `token:`, or `body:` fields.

**What you learn:** sequential call ordering, referencing `{{ .external.<name>.body }}` from a prior call, `continueOnError: false` on required calls vs `true` on optional ones.

---

## Prerequisites

Set `CLIENT_SECRET` in the environment where `ork run` executes:

```bash
export CLIENT_SECRET="your-client-secret"
```

Replace `spec.serviceUrl` in `cr.yaml` with your API service base URL.

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
    mode: dynamic / workers: 2 / resync: 60s
```

---

## Step 2 — Start the operator

```bash
ork run
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **webapp-chained-calls**, then select the **WebApp** CRD.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

The operator:
1. Fetches a token from `{{ serviceUrl }}/auth/token`
2. Uses that token to call `{{ serviceUrl }}/resources/my-app`
3. Creates the Deployment only when both calls succeed

```bash
kubectl get webapp my-app -o yaml | grep -A8 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  lastExternalStatus: "200"
```

---

## How the chaining works

In the Katalog, call 2's `token:` field references the first call's result:

```yaml
token: "{{ .external.tokenFetch.body }}"
```

This works because Orkestra updates the resolver after each call — by the time call 2's fields are resolved, `.external.tokenFetch.body` is already populated. If call 1 fails (with `continueOnError: false`), the reconcile halts and call 2 never runs.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
