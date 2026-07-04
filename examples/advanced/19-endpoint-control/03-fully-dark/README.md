# 19 — Endpoint Control · 03: Fully Dark

The asymmetric cross-CRD pattern — both operators try to read each other, but only the dark one succeeds.

**What you learn:** `crossAccess: false` controls *inbound* reads only. `keyrotation` is fully dark (`endpoints: enabled: false`, `crossAccess: false`) yet successfully reads `auditlog` state. `auditlog` is fully visible and attempts to read `keyrotation` — `crossAccess: false` blocks it, returning empty. The dark operator sees its sibling; its sibling cannot see it.

---

## How it works

Two CRDs in the same Katalog, both with a `cross:` block pointing at each other:

| CRD | crossAccess | endpoints | Reads sibling | Result |
|---|---|---|---|---|
| `keyrotation` | `false` | `enabled: false` | `auditlog` | ✓ succeeds — auditlog is open |
| `auditlog` | default (`true`) | default (all enabled) | `keyrotation` | ✗ denied — `crossAccess: false` |

`keyrotation` reads `auditlog`'s `status.auditRef` and reflects it in its own `status.auditRef`. `auditlog` tries to read `keyrotation`'s phase — the read resolves to empty and the ConfigMap records `keyRef: access-denied`.

Neither `crossAccess: false` nor `endpoints: enabled: false` affects reconciliation — both operators reconcile normally.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:

```
● auditlog     kind: AuditLog / group: endpoint-control.orkestra.io / version: v1alpha1
● keyrotation  kind: KeyRotation / group: endpoint-control.orkestra.io / version: v1alpha1

2 CRDs valid
```

---

## Step 2 — Simulate

```bash
ork simulate
```

The CR file passed to the simulator is [`crs/cr-keyrotation-audit.yaml`](./crs/cr-keyrotation-audit.yaml) — a combined multi-doc with both the KeyRotation and AuditLog CRs. This gives the simulator both peers so that each CRD's `cross:` block has a target to resolve.

Because the simulator runs each CRD independently, the simulate uses `expect.crds` to scope each assertion to the right CRD:

```yaml
expect:
  steady: true
  noErrors: true
  crds:
    keyrotation:
      ops:
        - cycle: 1
          verb: create
          resource: secrets
          name: platform-key-key
    auditlog:
      ops:
        - cycle: 1
          verb: create
          resource: configmaps
          name: platform-key-log
```

---

## Step 3 — Start the runtime

```bash
ork run
```

`crdFile` registers the CRDs and `crFiles` applies the CRs — no separate apply step needed. Keep this terminal open.

---

## Step 4 — Observe both sides

**`keyrotation` reads `auditlog` successfully.** After both operators have reconciled at least once:

```bash
kubectl get keyrotation platform-key
# NAME           PHASE     AUDITREF                    AGE
# platform-key   rotated   audit/platform-key active   ...
```

**`auditlog` tries to read `keyrotation` — denied:**

```bash
kubectl get auditlog platform-key
# NAME           PHASE    AUDITREF                    KEYREF          AGE
# platform-key   active   audit/platform-key active   access-denied   ...
```

The ConfigMap entry confirms it too:

```bash
kubectl get configmap platform-key-log -o jsonpath='{.data.keyRef}' && echo
# → access-denied
```

> `status.auditRef` on `keyrotation` is recomputed on every reconcile — it populates once `auditlog` has set its own status after its first cycle.

---

## Step 5 — Observe endpoint behaviour

`keyrotation` — all endpoints return 404:

```bash
curl localhost:8080/katalog/keyrotation
# → 404 page not found

curl localhost:8080/katalog/keyrotation/health
# → 404 page not found
```

`auditlog` — fully accessible:

```bash
curl -s localhost:8080/katalog/auditlog | jq .name
# → "auditlog"

curl -s localhost:8080/katalog/auditlog/health | jq .state
# → "healthy"
```

Both CRDs appear in the top-level count:

```bash
curl -s localhost:8080/katalog | jq .total
# → 2
```

---

## Step 6 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

`auditlog` is fully visible — info, health badge, CR list. `keyrotation` shows its runtime health state with a `⊘` icon and an `⊘ info endpoint disabled` notice; clicking View Details shows the endpoints-disabled page. The dark operator ran, read its sibling, and wrote a Secret — all without exposing a single endpoint.

---

## Step 7 — Run the E2E

```bash
ork e2e
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
