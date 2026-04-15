# Autoscaler Condition Engine
*How Orkestra evaluates autoscale conditions.*

The autoscaler condition engine determines **when** an OperatorBox should scale.  
It evaluates a combination of **metric conditions**, **cross‑operator metrics**, **clock windows**, **day‑of‑week rules**, and **cron expressions**, using the same declarative condition DSL used throughout Orkestra.

Conditions are evaluated on every autoscaler tick (`interval:`).  
If the declared logic resolves to **true**, the autoscaler applies the `do:` overrides.  
If the logic resolves to **false** for the entire `cooldown:` period, the autoscaler restores the CRD’s baseline.

---

## Condition structure

Autoscale conditions are expressed using two blocks:

```yaml
conditions:
  anyOf:   # OR
  when:    # AND
```

The combined logic is:

```
(anyOf is empty OR anyOf evaluates to true)
AND
(when is empty OR when evaluates to true)
```

If both blocks are present, **both must pass**.  
If only one block is present, only that block is evaluated.

---

## 1. Metric conditions (`metrics.*`)

Metric conditions reference **live operatorBox metrics** for the current CRD.  
These values come directly from the autoscaler’s in‑memory metrics — no API calls, no informers.

```yaml
when:
  - field: metrics.queueDepth
    greaterThan: "500"
  - field: metrics.workersBusyPercent
    greaterThan: "75"
```

Supported metric fields:

| Field | Description |
|---|---|
| `metrics.workersBusyPercent` | Percentage of workers actively reconciling |
| `metrics.workersIdlePercent` | Percentage of workers waiting for work |
| `metrics.queueDepth` | Current number of items in the queue |
| `metrics.reconcileDurationP95Ms` | P95 reconcile duration in milliseconds |
| `metrics.errorRatePercent` | Percentage of reconciles that failed in the last window |

Unknown metric fields fail fast at Katalog load time.

---

## 2. Cross‑operator metric conditions (`cross.<alias>.metrics.*`)

Cross‑operator metric conditions reference **runtime metrics from another operator**, accessed through Orkestra’s cross‑operator IPC layer.

This allows an operator to scale based on the load, pressure, or health of another operator.

```yaml
when:
  - field: cross.db.metrics.queueDepth
    greaterThan: "500"
  - field: cross.db.metrics.workersBusyPercent
    greaterThan: "70"
```

Cross‑operator metrics expose the same fields as local metrics:

| Field | Description |
|---|---|
| `cross.<alias>.metrics.queueDepth` | Queue depth of the referenced operator |
| `cross.<alias>.metrics.workersBusyPercent` | Busy worker percentage of the referenced operator |
| `cross.<alias>.metrics.workersIdlePercent` | Idle worker percentage |
| `cross.<alias>.metrics.reconcileDurationP95Ms` | P95 reconcile duration |
| `cross.<alias>.metrics.errorRatePercent` | Error rate |

If the referenced operator is not found, the metrics block is omitted and the condition evaluates to **false**.

---

## 3. Clock conditions

Clock conditions define a time window using `after:` and/or `before:`.

```yaml
anyOf:
  - time:
      after: "08:00"
      before: "17:00"
```

Rules:

- `after:` → active from this time until midnight  
- `before:` → active from midnight until this time  
- Using both defines a bounded window  
- Times use the system’s local timezone  

---

## 4. Day‑of‑week conditions

Day‑of‑week conditions activate on specific days.

```yaml
anyOf:
  - dayOfWeek:
      in: ["Saturday", "Sunday"]
```

Supported keys:

- `in:` — active only on these days  
- `notIn:` — active on all days except these  

Valid day names:

```
Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday
```

---

## 5. Cron conditions

Cron expressions define **when a time window opens**.  
The optional `duration:` defines how long the window stays open.

```yaml
anyOf:
  - cron: "0 8 * * 1-5"
    duration: 9h
```

Without `duration:`, the window lasts for one autoscaler tick.

Cron format:

```
minute hour dayOfMonth month dayOfWeek
```

---

## 6. Combined logic

The autoscaler evaluates conditions in this order:

1. **Evaluate anyOf (OR)**  
   - empty → true  
   - any entry true → pass  
   - all false → fail  

2. **Evaluate when (AND)**  
   - empty → true  
   - all entries must be true  

3. **Final result**  
   ```
   final = anyOf_passes AND when_passes
   ```

If `final == true` → apply `do:` overrides.  
If `final == false` for the entire `cooldown:` → restore baseline.

---

## 7. Examples

### Metrics + cross‑operator metrics

```yaml
conditions:
  when:
    - field: metrics.queueDepth
      greaterThan: "300"
    - field: cross.db.metrics.queueDepth
      greaterThan: "500"
```

### Cron‑driven batch window

```yaml
conditions:
  anyOf:
    - cron: "0 23 * * *"
      duration: 3h
  when:
    - field: metrics.workersBusyPercent
      greaterThan: "60"
```

### Weekend scale‑down

```yaml
conditions:
  anyOf:
    - dayOfWeek:
        in: ["Saturday", "Sunday"]
```

---

## 8. Validation

At Katalog load time, Orkestra validates:

- all `metrics.*` fields  
- all `cross.*.metrics.*` fields  
- all cron expressions  
- all time formats  
- all day‑of‑week values  

Invalid conditions fail fast with a clear error message.

---

## 9. Runtime behavior

- Conditions are evaluated every `interval:` tick  
- Overrides apply immediately  
- Restoration requires `cooldown:` consecutive false evaluations  
- All evaluations are in‑memory and O(1)  
- No API calls, no informers, no external systems  