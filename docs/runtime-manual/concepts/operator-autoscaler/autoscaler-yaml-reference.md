# Autoscaler YAML Reference
*The complete schema for `operatorBox.autoscale`.*

The Operator Autoscaler is configured declaratively inside the `operatorBox:` block of a CRD’s Katalog entry. This document describes every field, its type, its behavior, and how it interacts with the runtime.

Autoscaling is optional. When omitted, the CRD runs with its declared baseline for its entire lifetime.

---

## Top‑level structure

```yaml
operatorBox:
  autoscale:
    interval: <duration>
    cooldown: <duration>
    conditions:
      anyOf: [<condition>, ...]
      when:  [<condition>, ...]
    do:
      workers: <int>
      queueDepth: <int>
      resync: <duration>
```

---

## 1. `interval:`
How often the autoscaler evaluates conditions.

**Type:** duration  
**Required:** yes  
**Examples:** `10s`, `30s`, `1m`

The autoscaler loop runs:

```
every interval:
    evaluate conditions
    apply overrides or restore baseline
```

Shorter intervals react faster but increase evaluation frequency.

---

## 2. `cooldown:`
How long conditions must remain **false** before the autoscaler restores the baseline.

**Type:** duration  
**Required:** no (default: `0s`)  
**Examples:** `2m`, `30s`, `5m`

Cooldown applies **only** to the revert direction.  
Overrides apply immediately.

---

## 3. `conditions:`
Defines when the autoscaler should apply overrides.

```yaml
conditions:
  anyOf: [ ... ]   # OR
  when:  [ ... ]   # AND
```

The combined logic is:

```
(anyOf empty OR anyOf passes)
AND
(when empty OR when passes)
```

If both blocks are present, **both must pass**.

---

## 4. Condition types

A condition is one of:

### 4.1 Metric condition
```yaml
field: metrics.queueDepth
greaterThan: "500"
```

Supported operators:

- `greaterThan`
- `lessThan`
- `equals`
- `notEquals`

Supported metric fields:

- `metrics.workersBusyPercent`
- `metrics.workersIdlePercent`
- `metrics.queueDepth`
- `metrics.reconcileDurationP95Ms`
- `metrics.errorRatePercent`

Unknown fields fail fast at Katalog load time.

---

### 4.2 Clock condition
```yaml
time:
  after: "08:00"
  before: "17:00"
```

Keys:

- `after:` — active from this time until midnight  
- `before:` — active from midnight until this time  

Both may be combined to form a bounded window.

---

### 4.3 Day‑of‑week condition
```yaml
dayOfWeek:
  in: ["Saturday", "Sunday"]
```

Keys:

- `in:` — active only on these days  
- `notIn:` — active on all days except these  

Valid day names:

```
Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday
```

---

### 4.4 Cron condition
```yaml
cron: "0 23 * * *"
duration: 3h
```

Keys:

- `cron:` — cron expression defining when the window opens  
- `duration:` — how long the window stays open  

Without `duration:`, the window lasts for one autoscaler tick.

Cron format:

```
minute hour dayOfMonth month dayOfWeek
```

---

## 5. `do:` — override block

Defines the values to apply when conditions are met.

```yaml
do:
  workers: <int>
  queueDepth: <int>
  resync: <duration>
```

All fields are optional.  
Only declared fields are overridden.

### 5.1 `workers:`**
Number of concurrent reconcile goroutines allowed.

**Type:** int  
**Required:** no  
**Example:** `12`

Uses a resizable semaphore.  
No goroutines are killed; scale‑down waits for in‑flight reconciles to finish.

---

### 5.2 `queueDepth:`
Maximum number of items allowed in the queue.

**Type:** int  
**Required:** no  
**Example:** `1000`

If the queue is already deeper than the new limit, no items are dropped.  
The limit only affects *new* enqueues.

---

### 5.3 `resync:`
How often all CRs are re‑enqueued regardless of changes.

**Type:** duration  
**Required:** no  
**Example:** `20s`

Overrides activate a dedicated resync goroutine.  
When reverted, the goroutine idles and the baseline resync takes over.

---

## 6. Baseline behavior

The CRD’s declared configuration is always the baseline:

```yaml
workers: 4
queue:
  maxQueueDepth: 100
resync: 120s
```

When conditions are false for the entire cooldown window, the autoscaler restores:

- baseline workers  
- baseline queue depth  
- baseline resync interval  

A restart of Orkestra always begins from the baseline.

---

## 7. Validation rules

At Katalog load time, Orkestra validates:

- metric field names  
- comparison operators  
- cron expressions  
- time formats  
- day‑of‑week values  
- duration formats  

Invalid configurations fail fast with a clear error message.

---

## 8. Examples

### Traffic‑based scaling
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
  do:
    workers: 10
    resync: 30s
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
