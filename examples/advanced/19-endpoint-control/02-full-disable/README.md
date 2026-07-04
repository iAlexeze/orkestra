# 19 — Endpoint Control · 02: Full Disable

Suppressing all per-CRD HTTP endpoints while the operator continues to reconcile.

**What you learn:** `endpoints.enabled: false`, what "disabled" means at the HTTP layer vs the reconcile layer, and how the top-level `/katalog` summary remains intact.

---

## How it works

The `credential` operator generates a 32-character token on first reconcile (`once: true`) and rotates it every 720 hours. None of this is visible via HTTP. `/katalog/credential`, `/katalog/credential/health`, and `/katalog/credential/cr` are not registered — they return 404.

```yaml
endpoints:
  enabled: false
```

The operator appears in the top-level `/katalog` count but is not drillable from the Control Center or via curl.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:

```
● credential   kind: Credential / group: endpoint-control.orkestra.io / version: v1alpha1

1 CRD valid
```

---

## Step 2 — Simulate

```bash
ork simulate
```

Verifies the Secret is created on cycle 1.

---

## Step 3 — Start the runtime

```bash
ork run
```

`crFiles` in the katalog applies the CR automatically. Keep this terminal open.

---

## Step 4 — Observe endpoint behaviour

Top-level — still reachable, CRD appears in count:

```bash
curl -s localhost:8080/katalog | jq .total
# → 1
```

Per-CRD endpoint — returns 404:

```bash
curl localhost:8080/katalog/credential
# → 404 page not found

curl localhost:8080/katalog/credential/health
# → 404 page not found
```

The Secret exists and holds the generated token:

```bash
kubectl get secret api-creds-token -o jsonpath='{.data.token}' | base64 -d && echo
```

---

## Step 5 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The `credential` CRD appears in the list showing its runtime health state badge alongside a muted `⊘` icon and an `⊘ info endpoint disabled` notice in place of the usual metrics. Clicking **View Details** shows a clean disabled page:

> **Endpoints disabled** — All HTTP endpoints for this CRD have been disabled via `endpoints: enabled: false` in the Katalog. The operator is running and reconciling normally — its API surface is intentionally hidden.

The Resources tab shows **No resources found** — the CR list endpoint is not reachable, so no CRs are displayed even if they exist.

---

## Step 6 — Run the E2E

```bash
ork e2e
```

Asserts the Secret is created and both `/katalog/credential` and `/katalog/credential/health` return 404.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
