# 1. Multi‑dimensional autoscaling (beyond workers/queue/resync)
Right now you scale:

- workers  
- queueDepth  
- resync  

But OperatorBox already has more knobs:

- provider concurrency  
- external call rate limits  
- IPC throughput  
- state machine pacing  
- reconcile batching  
- retry backoff  
- drift correction frequency  

Autoscaling can eventually tune **all of these**.

Imagine:

```yaml
do:
  provider.aws.s3.maxConcurrent: 20
  ipc.rateLimit: 200
  stateMachine.stepDelay: 0s
```

Operators that *reshape their entire execution model* under load.

---

## 2. Autoscaler Profiles (pre‑built behaviors)
You’ll eventually have presets like:

- **burst**  
- **steady**  
- **batch**  
- **latency‑sensitive**  
- **cost‑optimized**  

Users will write:

```yaml
autoscale:
  profile: burst
```

And Orkestra will apply a curated set of overrides and conditions.

This is how Kubernetes HPA → VPA → KEDA evolved, but you’ll do it cleaner.

---

## 3. Cross‑operator autoscaling (DONE)
Operator A can scale based on Operator B’s pressure.

Example:

```yaml
conditions:
  when:
    - field: cross.database.metrics.queueDepth
      greaterThan: "500"
```

This is only possible because OperatorBox already supports **cross‑operator IPC**.

Imagine a database operator scaling up when the application operator is overwhelmed.  
Or a pipeline operator slowing down when a downstream operator is saturated.

This is *ecosystem‑level autoscaling*.

---

## 4. Autoscaler‑driven provider scaling
Providers (AWS, MongoDB, Redis, etc.) already expose metrics and operations.

Autoscaling can eventually do:

```yaml
do:
  providers.aws.s3.scaleUp: true
```

Or:

```yaml
do:
  providers.mongodb.clusterSize: 5
```

Operators scaling *infrastructure* based on runtime pressure.

This is where Orkestra becomes a **control plane for cloud resources**, not just Kubernetes objects.

---

## 5. Predictive autoscaling (trend‑based)
Because Orkestra has:

- queue depth over time  
- reconcile duration over time  
- error rate over time  
- worker utilization over time  

…it can detect trends.

Imagine:

- “Queue depth is rising steadily → scale early”  
- “Reconcile duration is degrading → preemptively increase workers”  
- “Error rate spikes every day at 9am → scale before the spike”  

This is autoscaling that *anticipates* load instead of reacting to it.

---

## 6. Autoscaler as a state machine
Autoscaling itself can become declarative and multi‑step:

```yaml
autoscale:
  phases:
    - name: warmup
      when:
        - field: metrics.queueDepth
          greaterThan: "200"
      do:
        workers: 8

    - name: burst
      when:
        - field: metrics.queueDepth
          greaterThan: "1000"
      do:
        workers: 20
        queueDepth: 5000
```

Operators that scale in **stages**, not just on/off.

---

## 7. Autoscaler UI in Control Center
You’ll eventually visualize:

- live worker count  
- queue depth over time  
- autoscaler decisions  
- condition evaluations  
- override history  
- baseline vs override diffs  

A full autoscaler dashboard.

This will make Orkestra feel like a **living organism**.

---

## 8. Autoscaler‑driven normalization
Imagine autoscaling that adjusts *schema normalization* behavior:

```yaml
do:
  normalize.spec.schedule: aggressive
```

Or:

```yaml
do:
  normalize.enabled: false
```

Operators that change how they interpret CRs under load.

---

## 9. Autoscaler‑driven drift correction
Drift correction frequency can scale too:

```yaml
do:
  driftCorrection.interval: 5s
```

During outages or chaos, the operator becomes more vigilant.  
During calm periods, it relaxes.

---

## 10. Autoscaler as a platform primitive
Eventually, autoscaling becomes so fundamental that:

- every operator uses it  
- every provider uses it  
- every pipeline uses it  
- every state machine uses it  
- every cross‑operator relationship uses it  

Autoscaling becomes the **runtime’s adaptive layer**.

Orkestra becomes a **self‑optimizing operator operating system**.