# Admission Webhooks — Validation and Mutation

Orkestra registers a `ValidatingWebhookConfiguration` and a `MutatingWebhookConfiguration` that intercept every `CREATE` and `UPDATE` request for CRs managed by your Katalog. Bad CRs are rejected before they are stored. Incomplete CRs have defaults filled in before validation runs — so they pass rules they would otherwise fail.

**What you learn:** `security.webhooks.admission`, validation rules (`deny` / `warn`), mutation defaults, `mutateFirst`, status fields that mirror post-mutation values, webhook self-healing, and the Control Center resource inspector.

---

## How it works

```yaml
security:
  webhooks:
    admission:
      enabled: true
    cleanupOnShutdown: true
    failurePolicy: Ignore   # allow apply through if Orkestra is temporarily unreachable
```

Per-CRD rules for the `Platform` CRD:

```yaml
validation:
  rules:
    - field: spec.image
      prefix: "registry.internal/"
      action: deny        # hard block — wrong registry

    - field: spec.replicas
      greaterThan: 0
      action: deny        # hard block — must be at least 1

    - field: spec.environment
      operator: in
      value: "development,staging,production"
      action: deny        # hard block — unknown environment

    - field: spec.rateLimit
      operator: exists
      action: warn        # advisory — logged, not blocked

mutation:
  mutateFirst: true
  rules:
    - field: spec.replicas    default: "2"
    - field: spec.environment default: "development"
    - field: spec.rateLimit   default: "100"
```

At startup Orkestra:

1. Registers the `/admission` endpoint on its HTTPS server
2. Creates `ValidatingWebhookConfiguration` named `orkestra-admission-validation` — only because at least one validation rule is declared
3. Creates `MutatingWebhookConfiguration` named `orkestra-admission-mutation` — only because at least one mutation rule is declared
4. Adds rules covering CREATE and UPDATE for the `Platform` CRD group

If only validation rules were declared, only the `ValidatingWebhookConfiguration` would be created. If only mutation rules were declared, only the `MutatingWebhookConfiguration` would be created. Orkestra does not create a webhook configuration unless there are rules that require it.

### One declaration. Two enforcement points.

The same rules in the Katalog are enforced at two independent points:

1. **Admission time** (`security.webhooks.admission.enabled: true`) — the mutation webhook fires first (filling defaults), then the validation webhook fires (checking rules). A CR that fails a `deny` rule is rejected before it is stored; the reconciler never sees it.

2. **Reconcile time** (always, whether webhooks are enabled or not) — the reconciler re-applies mutation defaults and re-checks validation rules on every reconcile cycle. If a CR somehow bypassed the webhook (e.g., the webhook was absent during a restart window), reconcile halts the moment it encounters a `deny` violation — no Deployment, no Service, no child resource of any kind.

> **Without `security.webhooks.admission`:** only enforcement point 2 is active. A bad CR can be `kubectl apply`-ed and stored in etcd, but Orkestra will never act on it — the reconciler halts immediately on the first `deny` rule. Enabling admission webhooks adds the fast gate so the CR is never stored in the first place.

---

## Step 1 — Install the ork CLI

```bash
curl get.orkestra.sh | bash
ork version
```

---

## Step 2 — Validate the Katalog

```bash
ork validate -k katalog.yaml
```

Expected output:

```
Validating Katalog...

● platform kind: Platform / group: admission.orkestra.io / ...

1 CRD valid (0 built-in, 1 custom)
```

---

## Step 3 — Apply the CRD

```bash
kubectl apply -f crd.yaml
```

---

## Step 4 — Generate and apply the operator bundle

```bash
ork generate bundle -k katalog.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

---

## Step 5 — Install Orkestra

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --wait --timeout 120s
```

**TLS certificates:** Orkestra generates and rotates its own TLS certificate automatically. To supply your own: `--set tls.certFile=/path/to/tls.crt --set tls.keyFile=/path/to/tls.key`.

At startup you will see:

```
{"level":"info","message":"admission mutation webhook registered: orkestra-admission-mutation"}
{"level":"info","message":"admission validation webhook registered: orkestra-admission-validation"}
```

---

## Step 6 — Apply the valid CR

```bash
kubectl apply -f cr-valid.yaml
```

`cr-valid.yaml` declares all fields explicitly:

```yaml
spec:
  image: registry.internal/my-platform:v1.0
  replicas: 3
  environment: production
  rateLimit: 500
```

All validation rules pass. The mutation webhook sees that every field is already set and leaves them untouched.

Verify:

```bash
kubectl get platform my-platform
```

```
NAME          ENVIRONMENT   REPLICAS   PHASE     AGE
my-platform   production    3          Running   10s
```

---

## Step 7 — Try applying the bad CR

```bash
kubectl apply -f cr-bad.yaml
```

`cr-bad.yaml` uses an image from `docker.io` — not from `registry.internal/`:

```yaml
spec:
  image: docker.io/nginx:1.25   # ← fails the prefix rule
```

Expected:

```
Error from server: admission webhook "orkestra-admission-validation.orkestra.io" denied the request:
images must come from the internal registry (registry.internal/)
```

The CR is never stored. No Deployment is created. The reconciler never sees it.

---

## Step 8 — Apply the minimal CR and observe mutation

```bash
kubectl apply -f cr-mutated.yaml
```

`cr-mutated.yaml` declares only the required field:

```yaml
spec:
  image: registry.internal/my-platform:v1.0
  # replicas, environment, rateLimit intentionally omitted
```

The mutation webhook fires first and fills the defaults:

| Field | Declared | After mutation |
|-------|----------|----------------|
| `spec.replicas` | — | `2` |
| `spec.environment` | — | `development` |
| `spec.rateLimit` | — | `100` |

The validation webhook then fires and all rules pass (replicas=2 satisfies `greaterThan: 0`, environment=development satisfies the `in` rule).

Check the status to confirm exactly what was mutated:

```bash
kubectl get platform minimal-platform -o yaml
```

```yaml
status:
  phase: Running
  image: registry.internal/my-platform:v1.0
  environment: development
  replicas: 2
  rateLimit: 100
```

The status fields reflect the object **after mutation** — so the values shown are the values Orkestra is actually using, even when the CR did not declare them.

You can also use `kubectl get`:

```bash
kubectl get platform minimal-platform
```

```
NAME               ENVIRONMENT   REPLICAS   PHASE     AGE
minimal-platform   development   2          Running   15s
```

---

## Step 9 — Observe admission events in the Control Center

> **The Control Center is the best place to inspect what is happening.** Port-forward and open it:

```bash
kubectl port-forward svc/orkestra-cc -n orkestra-system 8081:8081
```

Open [http://localhost:8081](http://localhost:8081).

Navigate to **admission** → click the **Platform** CRD. In the **top-right panel** you will find:

- **Status tab** — live status fields for every Platform instance, showing post-mutation values
- **Conditions tab** — reconcile conditions, including any `warn`-level advisory messages (e.g., missing `rateLimit`)
- **Events tab** — a timestamped log of every admission decision: which CR was admitted, which was rejected, which fields were mutated, and the user identity behind each request

This gives you full audit visibility without `kubectl` — one place to see every Platform across all namespaces.

---

## Step 10 — Webhook self-healing

Orkestra's webhook controller watches the `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` it owns. If either is deleted manually, Orkestra recreates it within the configured sync interval (default 30 seconds; set `WEBHOOK_CONTROLLER_SYNC_INTERVAL` to any value in seconds — minimum 1):

```bash
kubectl delete validatingwebhookconfiguration orkestra-admission-validation
kubectl delete mutatingwebhookconfiguration orkestra-admission-mutation
```

Watch the webhooks recreated:

```bash
kubectl get validatingwebhookconfiguration -w
kubectl get mutatingwebhookconfiguration -w

# orkestra-admission-validation and orkestra-admission-mutation reappears
```

Both webhooks are back. Validation and mutation are restored without restarting the operator.

---

## Cleanup

`cleanupOnShutdown: true` removes both webhook configurations, the TLS Secret, and all other security resources automatically when Orkestra stops:

```bash
helm uninstall orkestra -n orkestra-system
```

After uninstall:
* Logs:
```json
{"level":"info","config":"orkestra-admission-validation","time":1777697002,"message":"webhook: ValidatingWebhookConfiguration unregistered"}
{"level":"info","config":"orkestra-admission-mutation","time":1777697002,"message":"webhook: MutatingWebhookConfiguration unregistered"}
{"level":"info","namespace":"orkestra-system","time":1777697002,"message":"tls secret removed on shutdown"}
{"level":"warn","time":1777697002,"message":"webhook server: offline"}
```

* Verify:
```bash
kubectl get validatingwebhookconfiguration orkestra-admission-validation
# Error from server (NotFound): ... — removed automatically

kubectl get mutatingwebhookconfiguration orkestra-admission-mutation
# Error from server (NotFound): ... — removed automatically
```

### Cleanup CR
```bash
chmod +x cleanup.sh && ./cleanup.sh
```

