# 02 — Based on Own Metrics

This example demonstrates **how the Orkestra autoscaler works**, what happens when you configure it, and how to tune it for your operator.

**What you learn:**

- How Orkestra reads and reacts to your own CRD metrics
- How autoscaling adjusts workers, queueDepth, and resync at runtime
- How the Control Center visualises autoscaler state and overrides
- How cooldown works and when baseline is restored

**Builds on:** [01 — Without Autoscaler](../01-without-autoscaler/README.md)
It reuses the same setup.

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
ork control
# username:password → orkestra
```

Visit: **http://localhost:8081**

---

## Run the Example

### 1. Start the operator

```bash
ork run
# Orkestra reads katalog.yaml, applies the CRD and cr.yaml, and starts the operator.
```

Watch the Control Center as resources are created:

- A **Deployment** and **Service** appear  
- **2 workers** stay mostly idle  
- Reconciliation happens every minute (`resync`)  
- Queue limit shows **100** (default)  
- A new section **Autoscaler Baseline** appears under *Worker Pool*  
  - Expand it to see the baseline configuration (exactly what you set in the katalog)

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
- Workers become fully busy  
- Once the queue stays above **80** for the `autoscale.interval` (15s), autoscaling activates  
- **Autoscaler Baseline** switches to **Autoscaler Override Active**  
- You’ll see:
  - queueDepth → **500**  
  - workers → **8**  
  - resync → **10s**  
  - **no items dropped**

In the *Worker Pool* section:

```
8 of 8 workers actively processing (scaled from 2)
```

### Expected logs

```json
{"level":"info","crd":"autoscale.orkestra.io/v1alpha1, Kind=Ingestor","workers":8,"message":"autoscaler: worker pool resized"}
{"level":"info","crd":"autoscale.orkestra.io/v1alpha1, Kind=Ingestor","queueDepth":500,"message":"autoscaler: queue depth limit updated"}
{"level":"info","crd":"autoscale.orkestra.io/v1alpha1, Kind=Ingestor","resync":10000,"message":"autoscaler: resync interval updated"}
{"level":"info","crd":"Ingestor","workers":8,"queueDepth":500,"resync":10000,"message":"autoscaler: override applied"}
```

---

## Reduce the Load

Bring the load down to 50:

```bash
./load.sh down 50
```

What happens:

- Queue depth begins to fall  
- Workers remain at **8** until the queue stays below **80%** for the entire `autoscale.cooldown` (2 minutes)  
- After cooldown, autoscaler restores the baseline

### Expected logs

```json
{"level":"info","crd":"autoscale.orkestra.io/v1alpha1, Kind=Ingestor","workers":2,"message":"autoscaler: worker pool resized"}
{"level":"info","crd":"autoscale.orkestra.io/v1alpha1, Kind=Ingestor","queueDepth":0,"message":"autoscaler: queue depth limit updated"}
{"level":"info","crd":"autoscale.orkestra.io/v1alpha1, Kind=Ingestor","resync":60000,"message":"autoscaler: baseline restored"}
```

This is how Orkestra performs **per‑OperatorBox autoscaling**:  
no restart, no redeployment — everything happens live at runtime based on your declarative katalog.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```