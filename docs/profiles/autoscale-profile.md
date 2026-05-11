# Autoscale Profile
*Operator scaling presets — workers, queues, and conditions.*

An autoscale profile is a named preset that expands into a complete `operatorBox.autoscale` block at Katalog load time.

Autoscale profiles are relative. They use the CRD's declared `workers` and `queue.maxQueueDepth` as a **baseline** and derive thresholds, overrides, and timing from it. The same profile behaves differently for a low-throughput operator with 2 workers than for a high-throughput operator with 20.

---

## Profiles

| Profile | Trigger | Workers Override | Queue Override | Interval | Cooldown |
|---|---|---|---|---|---|
| `burst` | queueDepth > baseline × 3 | baseline × 4 | baseline × 10 | 5s | 30s |
| `steady` | queueDepth > baseline × 1.5 AND workers busy > 70% | baseline × 2 | baseline × 3 | 30s | 2m |
| `batch` | cron window 23:00 → 02:00 | baseline × 3 | baseline × 8 | 60s | 5m |
| `latency-sensitive` | P95 reconcile > 200ms | ⌈baseline × 2.5⌉ | — | 15s | 1m |
| `cost-optimized` | workers idle > 60% | max(1, baseline × 0.5) | baseline × 0.5 | 30s | 10m |

---

## Usage

Set `profile` inside the `autoscale:` block under `operatorBox:`:

```yaml
operatorBox:
  workers: 4
  queue:
    maxQueueDepth: 100
  autoscale:
    profile: steady
```

The profile expands using `workers: 4` and `maxQueueDepth: 100` as the baseline.

For `steady`, that produces:

- Trigger: queueDepth > 150 AND workersBusyPercent > 70
- Override: workers → 8, queueDepth → 300
- Interval: 30s, Cooldown: 2m

---

## Rules

**Profile or explicit — not both.**  
A `profile:` cannot coexist with manual `autoscale` fields (`interval:`, `cooldown:`, `conditions:`, `do:`) on the same CRD. Use one or the other.

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

Short intervals, aggressive overrides. Intended for operators that see sudden flood events — mass ingestion, fan-out operations, event storms.

```
trigger:  queueDepth > workers × 3
override: workers × 4, queueDepth × 10
timing:   interval 5s, cooldown 30s
```

Reverts quickly once the queue drains. Use when bursts are short-lived.

---

### `steady`
*Smooth, predictable scaling.*

Two-condition trigger prevents scaling on noise — both queue depth and worker saturation must rise together. Moderate overrides. Long cooldown.

```
trigger:  queueDepth > baseline × 1.5 AND workersBusyPercent > 70
override: workers × 2, queueDepth × 3
timing:   interval 30s, cooldown 2m
```

Good default for operators that serve consistent API load.

---

### `batch`
*Scale for a nightly processing window.*

Time-triggered via cron. Activates at 23:00, stays active for 3 hours, then reverts. No metric conditions.

```
trigger:  cron "0 23 * * *", duration 3h
override: workers × 3, queueDepth × 8
timing:   interval 60s, cooldown 5m
```

Use for ETL operators, nightly report generators, scheduled cleanup jobs.

---

### `latency-sensitive`
*Keep reconcile latency low.*

Triggers on P95 reconcile duration — not queue depth. Adds workers before the queue even builds. Only modifies worker count, not queue.

```
trigger:  reconcileDurationP95Ms > 200
override: workers × 2.5 (ceiling)
timing:   interval 15s, cooldown 1m
```

Use for operators where response time matters more than throughput — admission controllers, cert rotators, DNS operators.

---

### `cost-optimized`
*Minimize resource usage during low activity.*

Inverted logic — triggers when workers are mostly idle. Reduces both workers and queue depth. Long cooldown prevents flapping.

```
trigger:  workersIdlePercent > 60
override: max(1, workers × 0.5), queueDepth × 0.5
timing:   interval 30s, cooldown 10m
```

Use for operators that see highly variable load and you want to conserve resources during quiet periods.

---

## Choosing a profile

Ask these questions:

1. **Does this operator see sudden spikes?** → `burst`
2. **Does this operator run a scheduled nightly job?** → `batch`
3. **Does reconcile latency matter more than throughput?** → `latency-sensitive`
4. **Is the load consistent and predictable?** → `steady`
5. **Is the load low and I want to minimize footprint?** → `cost-optimized`

If none fit cleanly, write the autoscale block manually. Profiles are shortcuts, not constraints.

---

**Next →** [Probe Profile](./probe-profile.md)
