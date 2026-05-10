# 13 — Dependencies · 03: Cross Cluster

Database lives in **cluster-a** (infrastructure cluster). App lives in **cluster-b** (application cluster) and will not start until Database in cluster-a is healthy — across a real network boundary.

**What you learn:** Cross-cluster dependency ordering, NodePort exposure of the Orkestra API, and how `source.cacheFor` prevents excessive inter-cluster calls during high-frequency resyncs.

**Builds on:** [02 — Cross Binary](../02-cross-binary/README.md)

---

## How it works

App's cross block uses `source.endpoint` with a URL that crosses the cluster boundary:

```yaml
source:
  endpoint: "http://$DATABASE_ORKESTRA_URL/katalog/database/cr/{{ .metadata.namespace }}/{{ .metadata.name }}"
  cacheFor: 30s
```

`dependsOn: database: healthy` is checked against `cross.database.status.phase`. Once the remote returns `phase: Running`, App's reconcile loop begins. Results are cached for 30 s.

---

## Step 1 — Create two Kind clusters

```bash
make multi-cluster
```

Creates `orkestra-a` (Database) and `orkestra-b` (App).

---

## Step 2 — Apply CRDs in both clusters

```bash
kubectl config use-context kind-orkestra-a && kubectl apply -f crd.yaml
kubectl config use-context kind-orkestra-b && kubectl apply -f crd.yaml
```

---

## Step 3 — Install Orkestra in cluster-a (Database)

```bash
kubectl config use-context kind-orkestra-a

helm repo add orkestra https://orkspace.github.io/orkestra
helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --set service.type=NodePort \
  --set service.nodePort=30080 \
  --wait --timeout 120s

export DATABASE_ORKESTRA_URL="$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}'):30080"
echo "Database Orkestra URL: $DATABASE_ORKESTRA_URL"
```

---

## Step 4 — Apply Database CR in cluster-a

```bash
kubectl config use-context kind-orkestra-a
kubectl apply -f cr-database.yaml
kubectl get database my-database
```

```
NAME          IMAGE               ENDPOINT                         PHASE
my-database   postgres:16-alpine  my-database.default.svc:5432     Running
```

---

## Step 5 — Install Orkestra in cluster-b (App)

```bash
kubectl config use-context kind-orkestra-b

helm install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --wait --timeout 120s

kubectl set env deployment/orkestra \
  DATABASE_ORKESTRA_URL="$DATABASE_ORKESTRA_URL" \
  -n orkestra-system
```

---

## Step 6 — Apply App CR in cluster-b

```bash
kubectl config use-context kind-orkestra-b
kubectl apply -f cr-app.yaml
```

App starts as `Pending` while the dependency check calls cluster-a. Once the cached response confirms `phase: Running`, App moves to `Running`:

```bash
kubectl get app my-database
```

```
NAME          IMAGE                DB ENDPOINT                      PHASE
my-database   nginx:stable-alpine  my-database.default.svc:5432     Running
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
