# 06 — Full Stack (all patterns combined)

One CR, all five patterns at once. A `FullStackApp` creates 3 regional Deployments (`forEach`), a generated Secret (`once:`), a ConfigMap sourced from a database CR (`cross:`), all gated on a health check (`external:`), with a cleanup Job on terminal phases (`anyOf:`). This is the showcase — everything Orkestra can do in a single declaration.

**Prerequisite:** the `ManagedDatabase` from [03-cross-crd](../03-cross-crd/README.md) must be running — `FullStackApp` depends on it via `cross:`.

---

## Step 1 — Apply the CRD

```bash
kubectl apply -f crd.yaml
```

---

## Step 2 — Apply the database dependency

If not already running from example 03:

```bash
kubectl apply -f ../03-cross-crd/crd-database.yaml
kubectl apply -f ../03-cross-crd/database-cr.yaml
kubectl get manageddatabase my-app-db
# Wait until PHASE = Ready
```

---

## Step 3 — Validate

```bash
ork validate
```

Expected:
```
✓ full-stack-app
    kind: FullStackApp
    group: advanced.orkestra.io / version: v1alpha1 / plural: fullstackapps
    mode: dynamic / workers: 2 / resync: 30s
    dependsOn: managed-database
```

---

## Step 4 — Start the operator

`--dev-server` is required — the health check calls `/health` on every reconcile:

```bash
ork run --dev-server
```

---

## Step 5 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **full-stack-app**, then select the **FullStackApp** CRD.

---

## Step 6 — Apply the CR

```bash
kubectl apply -f cr.yaml
```

The CR covers all five patterns in one spec:

```yaml
spec:
  image: nginx:1.25
  regions: [us-east-1, eu-west-1, ap-southeast-1]   # forEach
  serviceUrl: http://localhost:9999                   # external:
  environment: production                             # ConfigMap value
  notify: "true"                                     # anyOf: notify trigger
```

---

## Step 7 — Watch the lifecycle

Phase walks forward as each condition is satisfied:

```bash
kubectl get fullstackapp my-app -w
# NAME     PHASE
# my-app   Pending
# my-app   WaitingForDatabase      ← cross: observed, database not yet Ready
# my-app   Ready                   ← health check passed, database Ready
```

Check everything that was created:

```bash
kubectl get deploy,secret,configmap -l orkestra-owner=my-app
```

Expected — for one CR:
```
deployment.apps/my-app-us-east-1      1/1
deployment.apps/my-app-eu-west-1      1/1
deployment.apps/my-app-ap-southeast-1 1/1
secret/my-app-creds                   (generated once, rotates after 90d)
configmap/my-app-config               (DB_HOST from database CR status)
```

---

## Step 8 — Status tells the full story

```bash
kubectl get fullstackapp my-app -o yaml | grep -A16 "status:"
```

Expected:
```yaml
status:
  phase: Ready
  regionsDeployed: "3"
  databaseEndpoint: my-app-db-postgres.default.svc.cluster.local:5432
  credentialsSecret: my-app-creds
```

Five patterns. One CR. Zero Go.

---

## Cleanup

```bash
kubectl delete fullstackapp my-app --ignore-not-found
kubectl delete manageddatabase my-app-db --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
```
