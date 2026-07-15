# Workload Autoscaler 01 — Time-Based Scaling

> **Already done `use-cases/temporal` 04-autoscale?** The mechanics here are the same. Skip to [02-external-api](../02-external-api/) for the next new concept.

One CR. A worker service that scales to 10 replicas during business hours (Mon–Fri 09:00–18:00 UTC) and back to 2 outside them. No CronJobs. The reconciler evaluates time conditions on every resync and patches `spec.replicas` when the window opens or closes.

**What you learn:** `autoscale:` on a Deployment — `target` jump scaling; `when: time:` and `when: dayOfWeek:` as scaling signals; `negate: true` to invert a condition; `cooldown` to prevent oscillation at window boundaries.

---

## What gets created

Always:

```
WorkerService/my-worker      Steady   Deployment running
Deployment/my-worker         2/2 Running   (off-peak)
```

During business hours:

```
WorkerService/my-worker      Peak     Deployment at peak
Deployment/my-worker         10/10 Running
```

The replica count changes automatically on the next resync after the window opens or closes. No manual intervention.

---

## How it works

The `autoscale:` block sits alongside the Deployment declaration. Conditions use the same `when:` / `anyOf:` engine as everything else in Orkestra:

```yaml
autoscale:
  min: 2
  max: 10
  cooldown: 2m
  scaleUp:
    conditions:
      when:
        - dayOfWeek:
            weekday: true
        - time:
            after: "09:00"
            before: "18:00"
    target: 10
  scaleDown:
    conditions:
      when:
        - dayOfWeek:
            weekday: true
          negate: true
        - time:
            after: "09:00"
            before: "18:00"
          negate: true
    target: 2
```

`scaleUp` uses `when:` (AND) — both weekday and time window must be true. `scaleDown` negates both — not a weekday AND not in the window. `negate: true` inverts the result of any condition. `target` is jump scaling — replicas are set to exactly that value in one reconcile.

---

## Step 1 — Validate

```bash
ork validate
```

## Step 2 — Simulate

```bash
ork simulate
```

## Step 3 — Start the runtime

```bash
ork run
```

## Step 4 — Observe

> **Tip:** to trigger the scale-up without waiting for business hours, edit the `after`/`before` values in [`katalog.yaml`](katalog.yaml) to bracket your current local time and restart `ork run`.

In a separate terminal, watch the Deployment replica count and CR status:

```bash
kubectl get workerservice,deployment
```

The `PHASE` column flips between `Steady` and `Peak`. The `REPLICAS` column shows the live count. The Deployment `READY` column tracks the actual pod count converging to the target.

---

## Step 5 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The Resources tab shows the Deployment replica count updating on every resync as the autoscaler evaluates the time conditions.

---

## E2E

```bash
ork e2e
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
