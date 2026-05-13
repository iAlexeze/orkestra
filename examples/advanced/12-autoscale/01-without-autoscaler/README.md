# 01 — Without Autoscaler

This example demonstrates **baseline operator behaviour in Orkestra when autoscaling is disabled**.  
It mirrors how traditional Kubernetes operators behave under load:

- You see **queue management** in action  
- You observe **Control Center** metrics and worker activity  
- You learn what happens when the reconcile queue exceeds its **queueDepth limit** (default: 100)

---

## Prerequisites

- Ork CLI  
- Kubernetes cluster (Kind works great)

Install Ork CLI:

```bash
curl get.orkestra.sh | bash
```

Create a Kind cluster and run this example:

```bash
ork run -f katalog --dev
```

Start the Control Center:

```bash
ork control start
```

Visit: **http://localhost:8081**

---

## Run the Example

### 1. Apply the CRD

```bash
kubectl apply -f crd.yaml
```

### 2. Apply the CR

```bash
kubectl apply -f cr.yaml
```

Watch the Control Center as resources are created:

- A **Deployment** and **Service** appear  
- **2 workers** stay mostly idle  
- Reconciliation happens every minute - `resync`
- Queue limit shows **100** (default) when you click the CRD

> Note:
> You can override the default queue limit by setting `queue.maxQueueDepth`. 

---

## Load the Ingestor

Now let’s overload the operator with **150 Ingestor resources**.

```bash
./load.sh 150
```

Observe in the Control Center:

- Queue depth rises  
- Workers stay busy  
- Once the queue reaches **100**, new items are **dropped** (default behaviour)

### Expected logs

```json
{"level":"warn","key":"default/ingestor-123","gvk":"autoscale.orkestra.io/v1alpha1, Kind=Ingestor","limit":100,"depth":100,"message":"enqueue: queue depth limit reached — item dropped"}
{"level":"warn","key":"default/ingestor-124","gvk":"autoscale.orkestra.io/v1alpha1, Kind=Ingestor","limit":100,"depth":100,"message":"enqueue: queue depth limit reached — item dropped"}
{"level":"warn","key":"default/ingestor-125","gvk":"autoscale.orkestra.io/v1alpha1, Kind=Ingestor","limit":100,"depth":100,"message":"enqueue: queue depth limit reached — item dropped"}
```

Control Center will show:

- **100/100**  
- **99/100**  

No errors — the operator simply drops excess events once the queue is full.

This is **normal behaviour** for operators **without autoscale**.

---

## What You Learned

- How Orkestra handles queue pressure without autoscaling  
- How workers behave under load  
- How the Control Center visualises queue depth and worker activity  
- Why items get dropped when queueDepth is exceeded  

In the **next example**, you’ll enable autoscaling and see how Orkestra prevents dropped items by dynamically scaling workers and adjusting queueDepth.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

> This will take a few seconds to drain the queue and finalize cleanup.