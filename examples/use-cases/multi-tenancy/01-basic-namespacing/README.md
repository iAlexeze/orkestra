# Multi-tenancy 01 — Basic namespacing

Two teams, one runtime. `platform-team` manages a Database CRD. `product-team` manages a Website CRD. Both Katalogs run in the same Orkestra process. The Control Center groups them into separate panels — one per namespace.

**What you learn:** declaring `metadata.namespace` on a Katalog, composing namespaced Katalogs with a Komposer, observing the namespace grouping in the Control Center.

---

## Step 1 — Validate

```bash
ork validate
```

Expected:
```
● database   kind: Database / group: multi-tenancy.orkestra.io / version: v1alpha1
● website    kind: Website  / group: multi-tenancy.orkestra.io / version: v1alpha1

2 CRDs valid
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

## Step 4 — Start the runtime

```bash
ork run
```

Both CRDs appear in the Control Center. Observe the **platform-team** panel showing `database` and the **product-team** panel showing `website` — separate sections, independent health.

---

## Step 5 — Apply the CRs

```bash
kubectl apply -f cr.yaml
```

Wait one reconcile cycle (~30s). Both CRs reach healthy state:

```bash
kubectl get databases,websites
```

```
NAME                                          PHASE     AGE
database.multi-tenancy.orkestra.io/main-db    Running   20s

NAME                                          PHASE     AGE
website.multi-tenancy.orkestra.io/storefront  Running   20s
```

Each CRD card appears under its team's panel in the Control Center. Health counts are tracked independently per namespace.

---

## E2E

Run the full lifecycle in one command — spins up a kind cluster, applies the CRDs, starts the operator, applies both CRs, asserts every expectation, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Database Deployment ready
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        name: main-db
        namespace: default
        ready: true
      - kind: Secret
        name: main-db-creds
        namespace: default

  - name: Website Deployment ready
    after: cr-applied
    timeout: 90s
    resources:
      - kind: Deployment
        name: storefront
        namespace: default
        ready: true

  - name: Cleanup verified
    after: cr-deleted
    timeout: 60s
    resources:
      - kind: Deployment
        name: main-db
        namespace: default
        count: 0
      - kind: Deployment
        name: storefront
        namespace: default
        count: 0
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
