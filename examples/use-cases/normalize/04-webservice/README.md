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

Open [http://localhost:8081](http://localhost:8081).

Select **webservice-operator**, then select the **WebService** CRD. Keep this tab open — you will watch every child resource appear as CRs are applied.

---

## Step 4 — Apply the simple CR

```bash
kubectl apply -f cr-simple.yaml
```

`cr-simple.yaml` has:
- `image: nginx` — bare, no tag, no registry
- `environment: " Production "` — messy
- `domain: "https://my-app.example.com/"` — protocol and trailing slash
- Two backends: `api`, `worker`
- No replicas, no resources

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
  phase: Ready
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

Expected:
```
NAME                        READY
my-app-production           1/1    ← main
my-app-production-api       1/1    ← forEach backend
my-app-production-worker    1/1    ← forEach backend
```

---

## Step 5 — Apply the full CR

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

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
