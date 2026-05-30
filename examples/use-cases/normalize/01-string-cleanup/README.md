# Normalize 01 — String Cleanup

Users write environment names in any casing, domains with or without protocol, tenant names with trailing spaces. This operator accepts all of it and produces one canonical form downstream — without a single `toLower` call scattered across `onCreate` or status fields.

**What you learn:** `toLower`, `trimSpace`, `trimPrefix`, `trimSuffix`, `default` in `normalize.spec`. The difference between what the user wrote and what the operator sees.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
✓ tenant
    kind: Tenant
    group: demo.orkestra.io / version: v1 / plural: tenants
    mode: dynamic / workers: 2 / resync: 30s
```

---

## Step 2 — Start the operator

```bash
ork run
```

Orkestra applies `crd.yaml` and starts the operator. You will see the informer sync:

```
{"level":"info","message":"health server listening on :8080"}
{"level":"info","crd":"demo.orkestra.io/v1, Kind=Tenant","message":"informer synced"}
{"level":"info","message":"✅ All komponents started successfully"}
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081).

Select **tenant-operator** from the katalog list, then select the **Tenant** CRD. You will see an empty CRD view — no CRs yet. Keep this open.

---

## Step 4 — Apply the messy CR

```bash
kubectl apply -f cr-messy.yaml
```

This [CR](cr-messy.yaml) has:
- `name: "Acme Corp "` — trailing space
- `environment: " Production "` — leading and trailing spaces, uppercase
- `domain: "https://acme.example.com/"` — protocol prefix and trailing slash
- `tier: "PREMIUM"` — uppercase

Switch to the Control Center. The `acme` CR appears in the Tenant list. Click it, then click the **top-right button** to open child resources. You will see the `acme` Tenant resource.

Click the `acme` to inspect it. Look at:
- **Status** — the normalized values written by Orkestra
- **Events** — the reconcile event
- **Created resource** - `acme-config`

> Status is written after the first reconcile — allow ~5s, then run:

Run both of these and look at them side by side:

```bash
kubectl get tenant acme -o yaml | grep -A6 "spec:"
kubectl get tenant acme -o yaml | grep -A10 "status:"
```

```yaml
spec:                              │  status:
  name: "Acme Corp "               │    name: acme corp
  environment: " Production "      │    environment: production
  domain: https://acme.example.com/│    domain: acme.example.com
  tier: PREMIUM                    │    tier: premium
```

Same object. Same cluster. The left is what the user wrote — stored in etcd, untouched. The right is what every downstream template, validation rule, and child resource received. Normalize ran between them.

---

## Step 5 — Apply the clean CR

```bash
kubectl apply -f cr-clean.yaml
```

This CR has:
- `name: "globex"` — already lowercase
- `environment: "staging"` — already canonical
- `domain: "globex-staging.example.com"` — no protocol
- `tier` omitted — normalize defaults to `"standard"`

```bash
kubectl get tenant globex -o yaml | grep -A10 "status:"
```

Expected:
```yaml
status:
  phase: Active
  name: globex
  environment: staging
  domain: globex-staging.example.com
  tier: standard       # ← defaulted by normalize
```

Both CRs reconcile to the same ConfigMap shape. The operator never knew which format it received.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
