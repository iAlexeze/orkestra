# External 04 — Chained Calls

Two calls run sequentially. The first fetches a short-lived auth token. The second uses that token in its `Authorization` header to call a protected API. The resolver is updated after each call — later calls can reference earlier results via template expressions in their `url:`, `token:`, or `body:` fields.

**What you learn:** sequential call ordering, referencing `{{ .external.<name>.body }}` from a prior call, `continueOnError: false` on required calls vs `true` on optional ones.

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

## Step 2 — Start the runtime

`--dev-server` starts a mock HTTP server on `:9999` — no auth service or secret needed. It handles the full chain: `POST /auth/token` returns `dev-token-abc123`, then `GET /resources/:name` returns a resource stub:

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

Open [http://localhost:8081](http://localhost:8081). Select **webapp-chained-calls**, then select the **WebApp** CRD.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f cr-dev.yaml
```

The operator:
1. Fetches a token from `/auth/token`
2. Uses that token to call `/resources/my-app`
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

Check the Deployment was created:

```bash
kubectl get deploy
# NAME      READY   UP-TO-DATE   AVAILABLE
# my-app    1/1     1            1
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

---

## E2E

Run the full lifecycle — deploys the mock dev server, starts the operator, applies the CR, asserts the Deployment is created and status is Ready after both auth chain calls succeed, then tears down:

```bash
ork e2e --dev-server
```

CRs use the in-cluster address defined in [cr-e2e.yaml](./cr-e2e.yaml). This runs everything in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Deployment created after full auth chain succeeds
    after: cr-applied
    resources:
      - kind: Deployment
        name: my-app
        ready: true
  - name: Status phase is Ready
    after: cr-applied
    commands:
      - run: kubectl get webapp my-app -o jsonpath='{.status.phase}'
        outputContains: Ready
```
