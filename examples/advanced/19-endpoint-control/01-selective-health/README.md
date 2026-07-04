# 19 — Endpoint Control · 01: Selective Health

Disabling a single endpoint while keeping the rest of the live API accessible.

**What you learn:** `endpoints.health: false`, why you might hide health state from external consumers, and how to assert a 404 on a disabled endpoint in e2e using the `statusCode:` assertion.

---

## How it works

The `payment` operator runs normally — creates a Deployment on reconcile, writes status. The health endpoint (`/katalog/payment/health`) is not registered. Info (`/katalog/payment`) and the CR list (`/katalog/payment/cr`) remain up.

```yaml
endpoints:
  health: false
```

`consecutiveFails`, `lastError`, `uptime`, and `state` are not reachable via HTTP. The Control Center shows the CRD in the list but the health badge is absent.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:

```
● payment   kind: Payment / group: endpoint-control.orkestra.io / version: v1alpha1

1 CRD valid
```

---

## Step 2 — Simulate

```bash
ork simulate
```

Verifies the Deployment is created on cycle 1 without a cluster.

---

## Step 3 — Start the runtime

```bash
ork run
```

`crdFile` registers the CRD and `crFiles` applies the CR — no separate apply step needed. Keep this terminal open.

---

## Step 4 — Observe reconciliation

In a second terminal, watch the Deployment come up:

```bash
kubectl get deployments
```

---

## Step 5 — Observe endpoint behaviour

Info endpoint — still reachable:

```bash
curl -s localhost:8080/katalog/payment | jq .name
# → "payment"
```

Health endpoint — returns 404:

```bash
curl localhost:8080/katalog/payment/health
# → 404 page not found
```

CR list — still reachable:

```bash
curl -s localhost:8080/katalog/payment/cr | jq .crd
# → "payment"
```

---

## Step 6 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The `payment` CRD appears in the list with its health state badge (Healthy/Started/etc.) derived from the runtime's internal state, plus a muted `⊘` icon indicating the health endpoint is not publicly reachable. Opening the CRD detail shows the Runtime Health section as `⊘ Health endpoint disabled — endpoints: health: false`, while Resources remain visible since the info endpoint is still active.

---

## Step 7 — Run the E2E

```bash
ork e2e
```

Asserts that `/katalog/payment/health` returns 404 and `/katalog/payment` returns 200 with the correct payload.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
