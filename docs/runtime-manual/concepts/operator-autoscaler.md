# Operator Autoscaler

The Orkestra Operator Autoscaler dynamically adjusts an operatorbox's worker
count, queue depth, and resync interval based on runtime metrics, time windows,
and cron expressions. It is declared inside the `operatorBox:` block and runs
inside the same process as the operator.

---

## How it works

The autoscaler runs a ticker loop at the declared `interval`. On each tick it
evaluates the declared conditions. When conditions are met it applies the `do:`
overrides. When conditions are no longer met — and the `cooldown:` period has
elapsed — it restores the CRD's declared baseline. No external controller. No
separate resource. No revert block.

**The CRD's declared configuration is always the baseline.** Overrides are
temporary deviations from it. A restart always begins from the declared values.

---

## YAML

```yaml
spec:
  crds:
    website:
      workers: 4                # ← baseline (autoscaler always returns here)
      queue:
        maxQueueDepth: 100      # ← baseline
      resync: 120s              # ← baseline

      operatorBox:
        autoscale:
          interval: 15s   # how often to evaluate conditions
          cooldown: 2m    # how long conditions must be false before reverting

          conditions:
            # anyOf → OR
            anyOf:
              - time:
                  after: "08:00"
                  before: "17:00"
              - cron: "0 20 * * 1-5"   # nightly batch window
                duration: 4h

            # when → AND (combined with anyOf using AND)
            when:
              - field: metrics.queueDepth
                greaterThan: "200"

          do:
            workers: 12
            queueDepth: 1000
            resync: 20s
```

The combined logic is:
`(anyOf conditions pass) AND (when conditions pass) → apply do:`

When both `anyOf` and `when` are declared, both must pass. When only one is
declared, only that one is evaluated.

---

## Condition types

### Metric conditions

Reference live operatorbox metrics via the `metrics.*` namespace. Evaluated
without any API server or informer involvement.

```yaml
when:
  - field: metrics.workersBusyPercent
    greaterThan: "80"
  - field: metrics.queueDepth
    greaterThan: "500"
  - field: metrics.reconcileDurationP95Ms
    greaterThan: "200"
```

Available metric fields:

| Field | Description |
|---|---|
| `metrics.workersBusyPercent` | Percentage of workers actively reconciling |
| `metrics.workersIdlePercent` | Percentage of workers waiting for work |
| `metrics.queueDepth` | Current number of items in the queue |
| `metrics.reconcileDurationP95Ms` | P95 reconcile duration in milliseconds |
| `metrics.errorRatePercent` | Percentage of reconciles that failed in the last window |

### Clock conditions

Active when the current time is within the declared window. Both `after` and
`before` are optional — omit `before` for "after this time until midnight",
omit `after` for "until this time from midnight".

```yaml
anyOf:
  - time:
      after: "08:00"
      before: "17:00"
```

### Day-of-week conditions

Active on the specified days. Values are full English day names.

```yaml
anyOf:
  - dayOfWeek:
      in: ["Saturday", "Sunday"]

# or for weekdays only:
  - dayOfWeek:
      notIn: ["Saturday", "Sunday"]
```

### Cron conditions

A cron expression defines when a time window opens. `duration:` defines how
long it stays open. Without `duration:`, the window closes after one
`interval` tick — almost always not the intended behavior.

```yaml
anyOf:
  - cron: "0 8 * * 1-5"    # opens at 08:00 on weekdays
    duration: 9h            # active until 17:00
  - cron: "0 0 * * 0"      # opens at midnight on Sunday
    duration: 4h            # active until 04:00
```

Standard cron format: `minute hour dayOfMonth month dayOfWeek`

---

## Cooldown

The cooldown prevents oscillation when metrics fluctuate around the threshold.
Without cooldown, a queue depth that alternates between 195 and 205 around a
threshold of 200 would cause the autoscaler to apply and revert on alternating
ticks.

```yaml
autoscale:
  interval: 15s
  cooldown: 2m    # conditions must be continuously false for 2m before reverting
```

Cooldown applies only to the revert direction. Override application is immediate.

---

## What can be scaled

```yaml
do:
  workers: 12        # number of concurrent reconcile goroutines
  queueDepth: 1000   # maximum queue depth before backpressure
  resync: 20s        # how often all CRs are re-enqueued regardless of changes
```

All three fields are optional in `do:`. Declare only what needs to change.

---

## Scenarios

### Scale up under load

```yaml
autoscale:
  interval: 15s
  cooldown: 3m
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

### Business hours

```yaml
autoscale:
  interval: 60s
  cooldown: 10m
  conditions:
    anyOf:
      - time:
          after: "08:00"
          before: "17:00"
      - dayOfWeek:
          notIn: ["Saturday", "Sunday"]
  do:
    workers: 10
    resync: 30s
```

### Weekend scale-down

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

### Nightly batch window

```yaml
autoscale:
  interval: 60s
  cooldown: 5m
  conditions:
    anyOf:
      - cron: "0 23 * * *"
        duration: 3h
  do:
    workers: 20
    queueDepth: 5000
    resync: 10s
```

---

## Metrics

The autoscaler emits its own metrics alongside the operatorbox metrics:

| Metric | Description |
|---|---|
| `orkestra_autoscale_override_active{crd}` | 1 when an override is currently applied, 0 otherwise |
| `orkestra_autoscale_overrides_total{crd}` | Total number of times an override was applied |
| `orkestra_autoscale_restores_total{crd}` | Total number of times the baseline was restored |
| `orkestra_autoscale_workers_current{crd}` | Current effective worker count |
| `orkestra_autoscale_queue_depth_current{crd}` | Current effective queue depth limit |

These are visible in the Control Center per operatorbox and scrapable via
`/metrics`.

---

## Implementation notes

**Worker resizing** uses a resizable semaphore, not goroutine add/drain.
All worker goroutines run continuously. The semaphore gates how many may enter
the reconcile loop simultaneously. Increasing weight allows more through
immediately. Decreasing weight causes excess goroutines to block after
completing their current reconcile — in-flight work is never interrupted.

**Queue depth** is enforced via a token counter on the queue wrapper. When the
limit is reached, new items are dropped with a metric increment. The queue
depth limit is a soft ceiling for backpressure, not a hard capacity.

**Resync interval** is updated on the informer's resync ticker. The change
takes effect on the next tick after the override is applied.