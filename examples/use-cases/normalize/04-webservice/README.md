# Normalize 04 — Full WebService

All normalize patterns in one operator. Apply `cr-simple.yaml` with a bare image, messy environment, and protocol-prefixed domain. Apply `cr-full.yaml` with clean explicit values. Both produce the same secrets, configmap, and deployments — because normalize ran first and every downstream resource read canonical values.

**What you learn:** Image normalization + string cleanup + resource defaults + composite field assembly (`internalName`) + `once:` secrets with `rotateAfter` + configmap from normalized spec + `forEach` backend deployments.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
✓ webservice
    kind: WebService
    group: demo.orkestra.io / version: v1 / plural: webservices
    mode: dynamic / workers: 3 / resync: 30s
```

---

## Step 2 — Simulate (optional, no cluster needed)

Run the reconciler against a fake in-memory cluster to see what normalize produces before applying to a real cluster:

```bash
ork simulate --cr cr-simple.yaml
```

```text
Simulating webservice/my-app

  Cycle 1:
    + secrets/my-app-production-acme-corp-api-key
    + secrets/my-app-production-jwt
    + configmaps/my-app-production-config
    + deployments/my-app-production
    + deployments/my-app-production-api
    + deployments/my-app-production-worker
    ~ status/my-app
  Cycle 2:
    ~ secrets/my-app-production-acme-corp-api-key
    ~ secrets/my-app-production-jwt
    ~ status/my-app
  Cycle 3:
    ~ status/my-app
  (cycles 4–10: identical)

  ✓ Steady state at cycle 4 in 350ms
```

**What this means:**
- The resource names in Cycle 1 already contain `production` — that is `internalName` computed by normalize from the messy `cr-simple.yaml` input (`" Production "` → `production`). Normalize ran before any template was evaluated.
- 2 secrets, 1 ConfigMap, and 3 Deployments (main + `api` + `worker` forEach backends) — all created from a single CR with bare, untidy inputs.
- `~ secrets` in Cycle 2 — the `once:` check runs every reconcile. Orkestra sees both secrets already exist and marks them as unchanged (`~` not `+`). They will not be regenerated.
- **Steady state at cycle 4** — one extra cycle compared to a simple operator, because `once:` takes a cycle to confirm the secret exists and skip rotation. From cycle 4 onward, no changes.
- If there is a template error — a typo in a field name, a missing normalize rule — simulate catches it here instead of on a live cluster.

---

## Step 3 — Start the operator

```bash
ork run
```

---

## Step 4 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081).

Select **webservice-operator**, then select the **WebService** CRD. Keep this tab open — you will watch every child resource appear as CRs are applied.

---

## Step 5 — Apply the simple CR

```bash
kubectl apply -f cr-simple.yaml
```

`cr-simple.yaml` has:
- `image: nginx` — bare, no tag, no registry
- `environment: " Production "` — messy
- `domain: "https://my-app.example.com/"` — protocol and trailing slash
- Two backends: `api`, `worker`
- No replicas, no resources

> The normalized image is `registry.internal/nginx:latest` — a fictional private registry. Deployments will show `ImagePullBackOff` and the phase will stay `Pending`. This is expected — the example demonstrates what normalize produces, not a running workload.

Switch to the Control Center. The `my-app` WebService appears. Click it, then click **top-right** to open child resources. You will see:

- **Secrets** — `my-app-production-acme-corp-api-key` and `my-app-production-jwt`
- **ConfigMap** — `my-app-production-config`
- **Deployments** — `my-app-production` (main), `my-app-production-api`, `my-app-production-worker` (backends)

Click each to inspect **Status**, **Labels**, **Events**, and **Conditions**.

Verify what normalize produced:

```bash
kubectl get webservice my-app -o yaml | grep -A15 "status:"
```

Expected:
```yaml
status:
  phase: Pending                            # ← registry.internal is fictional
  image: registry.internal/nginx:latest   # ← tag and registry added
  environment: production                  # ← trimmed, lowercase
  domain: my-app.example.com              # ← protocol and slash stripped
  internalName: my-app-production         # ← computed by normalize
  replicas: "1"                           # ← defaulted
  backendCount: "2"
  credentialsSecret: my-app-production-acme-corp-api-key
```

Check the ConfigMap was built from normalized values:

```bash
kubectl get configmap my-app-production-config -o yaml | grep -A10 "data:"
```

Expected:
```yaml
data:
  APP_DOMAIN: my-app.example.com
  APP_ENV: production
  APP_IMAGE: registry.internal/nginx:latest
  APP_REPLICAS: "1"
  ORG_ID: acme-corp
```

Check the backend Deployments:

```bash
kubectl get deployments
```

Expected (Deployments created — pods will show ImagePullBackOff since the registry is fictional):
```
NAME                        READY
my-app-production           0/1
my-app-production-api       0/1
my-app-production-worker    0/1
```

---

## Step 6 — Apply the full CR

```bash
kubectl apply -f cr-full.yaml
```

`cr-full.yaml` has clean inputs — fully-qualified image, lowercase environment, no protocol in domain — plus explicit replicas, resource requests, and three backends.

```bash
kubectl get webservice payments-service -o yaml | grep -A15 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  image: registry.internal/payments:2.1.3   # ← unchanged, already canonical
  environment: staging
  domain: payments-staging.example.com
  internalName: payments-service-staging
  replicas: "3"
  backendCount: "3"
```

In the Control Center, select `payments-service` and open child resources. You will see three backend Deployments — `processor`, `reconciler`, `reporter` — each created by the `forEach` loop. Click any of them to see the full resource detail: events, labels, status, conditions.

---

## What normalize did

| Field | `cr-simple.yaml` input | After normalize |
|---|---|---|
| `image` | `nginx` | `registry.internal/nginx:latest` |
| `environment` | `" Production "` | `production` |
| `domain` | `https://my-app.example.com/` | `my-app.example.com` |
| `internalName` | _(not in CR)_ | `my-app-production` |
| `replicas` | _(absent)_ | `1` |
| `resources.requests.cpu` | _(absent)_ | `100m` |

Every secret, configmap, and deployment name used `.spec.internalName` — assembled once in normalize, used everywhere.

---

## E2E

Run the full lifecycle in one command — applies the simple CR, asserts that:
- ConfigMap created with normalized values
- Secrets created with once semantics
-  Status internalName is normalized (environment trimmed and lowercased), then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
    - name: ConfigMap created with normalized values
      after: cr-applied
      timeout: 60s
      resources:
        - kind: ConfigMap
          name: my-app-production-config
          namespace: default

    - name: Secrets created with once semantics
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Secret
          name: my-app-production-acme-corp-api-key
          namespace: default
        - kind: Secret
          name: my-app-production-jwt
          namespace: default

    - name: Status internalName is normalized (environment trimmed and lowercased)
      after: cr-applied
      timeout: 60s
      commands:
        - run: kubectl get webservice my-app -o jsonpath='{.status.internalName}'
          outputContains: my-app-production
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
