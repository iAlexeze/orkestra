# 13 — Cross-Operator Communication · 03: Cross Cluster

Producer runs in **cluster-a**, Consumer runs in **cluster-b**. Consumer reads Producer's endpoint through cluster-a's external Orkestra URL — crossing a real cluster boundary with a single Katalog declaration.

**What you learn:** `make multi-cluster`, cross-cluster `source.endpoint`, how to expose an Orkestra API externally via NodePort, and the caching strategy to tolerate inter-cluster network latency.

**Builds on:** [02 — Cross Binary](../02-cross-binary/README.md)

---

## How it works

The only difference from `02-cross-binary` is that `source.endpoint` now points to a NodePort address in cluster-a instead of a cluster-internal service name. Orkestra's cross mechanism is network-agnostic — it does not care whether the endpoint is local or remote.

```yaml
source:
  endpoint: "http://$PRODUCER_ORKESTRA_URL/katalog/producer/cr/{{ .metadata.namespace }}/{{ .metadata.name }}"
  cacheFor: 30s
```

`$PRODUCER_ORKESTRA_URL` is the NodePort address of Orkestra running in cluster-a. Set it before applying the Consumer CR.

---

## Step 1 — Create two Kind clusters

```bash
make multi-cluster
```

This creates `orkestra-a` (Producer) and `orkestra-b` (Consumer).

---

## Step 2 — Apply the CRDs in both clusters

```bash
kubectl config use-context kind-orkestra-a
kubectl apply -f crd.yaml

kubectl config use-context kind-orkestra-b
kubectl apply -f crd.yaml
```

---

## Step 3 — Install Orkestra in cluster-a

```bash
kubectl config use-context kind-orkestra-a

helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --set service.type=NodePort \
  --set service.nodePort=30080 \
  --wait --timeout 120s
```

Note the NodePort address for later:

```bash
export PRODUCER_ORKESTRA_URL="$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}'):30080"
echo $PRODUCER_ORKESTRA_URL
```

---

## Step 4 — Apply the Producer CR in cluster-a

```bash
kubectl config use-context kind-orkestra-a
kubectl apply -f cr-producer.yaml
kubectl get producer my-producer
```

```
NAME          IMAGE                ENDPOINT           PHASE
my-producer   nginx:stable-alpine  10.96.45.12:8080   Running
```

---

## Step 5 — Install Orkestra in cluster-b

```bash
kubectl config use-context kind-orkestra-b

helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --wait --timeout 120s
```

---

## Step 6 — Apply the Consumer CR in cluster-b

Export the Producer Orkestra URL so Consumer's cross block can resolve it:

```bash
kubectl config use-context kind-orkestra-b

# Set the env var so Orkestra in cluster-b can reach cluster-a
kubectl set env deployment/orkestra \
  PRODUCER_ORKESTRA_URL="$PRODUCER_ORKESTRA_URL" \
  -n orkestra-system

kubectl apply -f cr-consumer.yaml
kubectl get consumer my-producer
```

```
NAME          IMAGE                PRODUCER ENDPOINT   PHASE
my-producer   nginx:stable-alpine  10.96.45.12:8080    Running
```

Consumer in cluster-b resolved Producer's endpoint from cluster-a.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
