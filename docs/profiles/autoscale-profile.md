# Autoscale Profile
*Operator scaling presets — workers, queues, and conditions.*

An autoscale profile is a named preset that expands into a complete `operatorBox.autoscale` block at Katalog load time.

Profiles are **relative**. They use the CRD’s declared `workers` and `queue.maxQueueDepth` as a **baseline**, then compute:

- **override workers**  
- **override queueDepth**  
- **thresholds** (as a % of the override queueDepth)  
- **interval + cooldown**  

The same profile behaves differently for an operator with 2 workers than for one with 20.

---

## Profiles

| Profile | Trigger | Workers Override | Queue Override | Interval | Cooldown |
|---|---|---|---|---|---|
| `burst` | queueDepth > 60% of override queueDepth | baseline × 4 | baseline × 10 | 5s | 30s |
| `steady` | queueDepth > 40% of override queueDepth AND workersBusy > 70% | baseline × 2 | baseline × 3 | 30s | 2m |
| `batch` | cron window 23:00 → 02:00 | baseline × 3 | baseline × 8 | 60s | 5m |
| `latency-sensitive` | P95 reconcile > 200ms | ⌈baseline × 2.5⌉ | — | 15s | 1m |
| `cost-optimized` | workersIdle > 60% AND queueDepth > 80% of override queueDepth | max(1, baseline × 0.5) | baseline × 0.5 | 30s | 10m |

---

## How profiles compute values

Given:

- baseline workers = `W`
- baseline queueDepth = `Q`
- maxQueueDepth = `M`
- profile multipliers = `WorkerMultiplier`, `QueueMultiplier`
- threshold percentage = `QueueThresholdPct`

Profiles compute:

### **1. Override queueDepth**
```
overrideQueueDepth = Q × QueueMultiplier
```

### **2. Threshold**
```
threshold = overrideQueueDepth × QueueThresholdPct
```

### **3. Override workers**
```
overrideWorkers = W × WorkerMultiplier
```

### **4. Interval + Cooldown**
Taken directly from the profile.

This ensures scaling happens **before** hitting the effective queue limit.

---

## Usage

```yaml
operatorBox:
  workers: 4
  queue:
    maxQueueDepth: 100
  autoscale:
    profile: steady
```

For `steady`, with baseline:

- workers = 4  
- queueDepth = 100  

The profile expands to:

- override workers = 4 × 2 = **8**
- override queueDepth = 100 × 3 = **300**
- threshold = 300 × 0.40 = **120**
- interval = 30s  
- cooldown = 2m  

Trigger:

```
queueDepth > 120 AND workersBusyPercent > 70
```

---

## Rules

**Profile or explicit — not both.**  
A `profile:` cannot coexist with manual autoscale fields (`interval:`, `cooldown:`, `conditions:`, `do:`).

```yaml
# Valid — profile only
autoscale:
  profile: burst

# Valid — explicit only
autoscale:
  interval: 10s
  cooldown: 1m
  conditions:
    when:
      - field: metrics.queueDepth
        greaterThan: "500"
  do:
    workers: 16

# Invalid — profile and explicit together
autoscale:
  profile: burst
  interval: 10s  # error: cannot mix profile and manual autoscale fields
```

**Unknown profiles fail fast.**  
An unrecognized profile name is a Katalog load error.

---

## Profile details

### `burst`
*React instantly to spikes.*

Aggressive scaling, short interval, short cooldown.

```
trigger:  queueDepth > 60% of override queueDepth
override: workers × 4, queueDepth × 10
timing:   interval 5s, cooldown 30s
```

---

### `steady`
*Smooth, predictable scaling.*

Requires both queue pressure and worker saturation.

```
trigger:  queueDepth > 40% of override queueDepth AND workersBusyPercent > 70
override: workers × 2, queueDepth × 3
timing:   interval 30s, cooldown 2m
```

---

### `batch`
*Scale for a nightly processing window.*

Time-triggered only.

```
trigger:  cron "0 23 * * *", duration 3h
override: workers × 3, queueDepth × 8
timing:   interval 60s, cooldown 5m
```

---

### `latency-sensitive`
*Keep reconcile latency low.*

Triggered by P95 latency, not queue depth.

```
trigger:  reconcileDurationP95Ms > 200
override: workers × 2.5 (ceiling)
timing:   interval 15s, cooldown 1m
```

---

### `cost-optimized`
*Minimize resource usage during low activity.*

Triggers when idle and queue is low.

```
trigger:  workersIdlePercent > 60 AND queueDepth > 80% of override queueDepth
override: max(1, workers × 0.5), queueDepth × 0.5
timing:   interval 30s, cooldown 10m
```

---

## Choosing a profile

1. **Sudden spikes?** → `burst`  
2. **Consistent load?** → `steady`  
3. **Nightly batch window?** → `batch`  
4. **Latency matters?** → `latency-sensitive`  
5. **Save resources?** → `cost-optimized`  

If none fit, write the autoscale block manually.