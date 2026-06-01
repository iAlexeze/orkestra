# Multi-tenancy 02 — Cross-read access control

`internal-team` declares `crossAccess: false` at the Katalog level — no other team can read any of its CRDs via `cross:`. The `ledger` CRD overrides back to `crossAccess: true` at the CRD level. `analytics-team` reads `ledger` successfully, but its `cross:` reference to `payment` returns `found: "false"` silently.

**What you learn:** Katalog-level `crossAccess: false`, CRD-level override, graceful degradation when a cross read is denied.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
● payment   kind: Payment / group: multi-tenancy.orkestra.io / version: v1alpha1
● ledger    kind: Ledger  / group: multi-tenancy.orkestra.io / version: v1alpha1
● report    kind: Report  / group: multi-tenancy.orkestra.io / version: v1alpha1

3 CRDs valid
```

---

## Step 2 — Apply the CRDs

```bash
kubectl apply -f crd.yaml
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081).

---

## Step 4 — Start the operator

```bash
ork run
```

Two namespace panels appear: **internal-team** (payment, ledger) and **analytics-team** (report).

---

## Step 5 — Apply the CRs

```bash
kubectl apply -f cr.yaml
```

Wait one reconcile cycle. Check the Report status:

```bash
kubectl get report daily-summary -o yaml | grep -A5 "status:"
```

Expected:
```yaml
status:
  phase: ready
  ledgerPhase: running
```

The Report reads `ledger` data (allowed) and reflects it in status. It cannot read `payment` — `cross.paymentState.found` returns `"false"` and the Report reconciles gracefully without it.

---

## Step 6 — Confirm payment is blocked

```bash
kubectl get payment checkout -o yaml | grep -A5 "status:"
```

The `report` CRD has no access to payment state. Its ConfigMap is created only from ledger data.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
