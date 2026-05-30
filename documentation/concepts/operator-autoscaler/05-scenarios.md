# Autoscaler Scenarios
*Real‑world patterns for scaling operatorBox:es.*

This document provides practical examples of how to use the Operator Autoscaler to adapt operator performance to real‑world workloads. Each scenario demonstrates a different combination of metric‑based, time‑based, cron‑based, and **cross‑operator‑based** conditions.

All examples assume the CRD declares a baseline:

```yaml
workers: 4
queue:
  maxDepth: 100
resync: 120s
```

The autoscaler always returns to these values when conditions are no longer met.

---

## 1. Scale Up Under Load
Increase workers and queue depth when the operator is under pressure.

```yaml
autoscale:
  interval: 15s
  cooldown: 2m

  conditions:
    when:
      - field: metrics.queueDepth
        greaterThan: "500"
      - field: metrics.workersBusyPercent
        greaterThan: "75"

  do:
    workers: 16
    queueDepth: 2000
```

**Behavior:**

- Overrides apply immediately when queue depth > 500 *and* workers are > 75% busy  
- Baseline is restored only after both conditions remain false for 2 minutes  

This is the most common autoscaling pattern.

---

## 2. Cross‑Operator Scaling
Scale based on the load of another operator.

```yaml
autoscale:
  interval: 20s
  cooldown: 1m

  conditions:
    when:
      - field: cross.db.metrics.queueDepth
        greaterThan: "500"
      - field: cross.db.metrics.workersBusyPercent
        greaterThan: "70"

  do:
    workers: 12
    queueDepth: 1500
```

**Behavior:**

- The operator scales up only when the *database operator* is under pressure  
- Enables upstream/downstream coordination without out-of-band signaling

---

## 3. Business Hours Scaling
Increase workers during the day, reduce them at night.

```yaml
autoscale:
  interval: 60s
  cooldown: 10m

  conditions:
    anyOf:
      - time:
          after: "08:00"
          before: "17:00"

  do:
    workers: 10
    resync: 30s
```

**Behavior:**

- Active from 08:00 → 17:00  
- Outside that window, baseline is restored after 10 minutes  
- Useful for operators that process daytime traffic or business workflows  

---

## 4. Weekend Scale‑Down
Reduce workers on weekends to conserve resources.

```yaml
autoscale:
  interval: 60s
  cooldown: 30m

  conditions:
    anyOf:
      - dayOfWeek:
          in: ["Saturday", "Sunday"]

  do:
    workers: 2
    queueDepth: 50
    resync: 300s
```

**Behavior:**

- Active only on Saturday and Sunday  
- Baseline restored on Monday after 30 minutes  
- Ideal for workloads with predictable weekday peaks  

---

## 5. Nightly Batch Window
Scale up for heavy batch processing at night.

```yaml
autoscale:
  interval: 60s
  cooldown: 5m

  conditions:
    anyOf:
      - cron: "0 23 * * *"   # 23:00 every day
        duration: 3h         # active until 02:00

  do:
    workers: 20
    queueDepth: 5000
    resync: 10s
```

**Behavior:**

- Window opens at 23:00  
- Window closes at 02:00  
- Overrides apply instantly  
- Baseline restored after 5 minutes of inactivity  

This is ideal for ETL, compaction, or nightly reconciliation workloads.

---

## 6. Error‑Rate‑Triggered Scaling
Scale up when the operator is failing too often.

```yaml
autoscale:
  interval: 20s
  cooldown: 1m

  conditions:
    when:
      - field: metrics.errorRatePercent
        greaterThan: "10"

  do:
    workers: 12
    resync: 15s
```

**Behavior:**

- If >10% of reconciles fail in the last window, scale up  
- Helps operators recover from transient provider failures or API throttling  

---

## 7. Latency‑Driven Scaling
Scale when reconciles become slow.

```yaml
autoscale:
  interval: 30s
  cooldown: 2m

  conditions:
    when:
      - field: metrics.reconcileDurationP95Ms
        greaterThan: "250"

  do:
    workers: 14
```

**Behavior:**

- If P95 reconcile duration exceeds 250ms, increase concurrency  
- Useful for operators with expensive reconcile logic  

---

## 8. Hybrid: Business Hours + Load
Scale only during business hours *and* only when under load.

```yaml
autoscale:
  interval: 30s
  cooldown: 5m

  conditions:
    anyOf:
      - time:
          after: "08:00"
          before: "17:00"

    when:
      - field: metrics.queueDepth
        greaterThan: "200"

  do:
    workers: 12
```

**Behavior:**

- Time window must be active  
- Queue depth must exceed 200  
- Both conditions must pass  

This is a precise, cost‑efficient scaling pattern.

---

## 9. Aggressive Burst Scaling
Scale aggressively for short bursts of traffic.

```yaml
autoscale:
  interval: 5s
  cooldown: 30s

  conditions:
    when:
      - field: metrics.queueDepth
        greaterThan: "1000"

  do:
    workers: 32
    queueDepth: 10000
```

**Behavior:**

- Reacts within 5 seconds  
- Restores baseline after 30 seconds of calm  
- Ideal for spiky workloads  

---

## 10. Multi‑Window Scaling
Different scaling behaviors at different times.

```yaml
autoscale:
  interval: 30s
  cooldown: 2m

  conditions:
    anyOf:
      - cron: "0 8 * * 1-5"   # morning
        duration: 2h
      - cron: "0 18 * * 1-5"  # evening
        duration: 3h

  do:
    workers: 18
    queueDepth: 3000
```

**Behavior:**

- Two independent windows  
- Either window activates the override  
- Useful for morning and evening traffic waves  

---

## 11. Minimalist Scaling
Scale only one parameter.

```yaml
autoscale:
  interval: 20s
  cooldown: 1m

  conditions:
    when:
      - field: metrics.queueDepth
        greaterThan: "150"

  do:
    workers: 8
```

**Behavior:**

- Only workers change  
- Queue depth and resync remain at baseline  
