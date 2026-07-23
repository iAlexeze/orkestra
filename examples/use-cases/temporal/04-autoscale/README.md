# Temporal 04 — Business Hours Autoscale

One CR. A Deployment that scales to 8 replicas during business hours (Mon–Fri 09:00–18:00 UTC) and back to 2 outside them. No CronJobs. The reconciler evaluates time conditions on every resync and patches `spec.replicas` when the window opens or closes.

**What you learn:** `autoscale:` on a Deployment — `target` jump scaling driven by a user-defined note; how `inBusinessHours` composes into both `scaleUp` and `scaleDown` conditions; `cooldown` to prevent oscillation at window boundaries.

---

## What gets created

Always:

```
APIServer/my-api      Active   Deployment running
Deployment/my-api     2/2 Running   (off-peak)
```

During business hours:

```
APIServer/my-api      Peak     Deployment at peak
Deployment/my-api     8/8 Running
```

The replica count changes automatically on the next resync after the window opens or closes. No manual intervention.

---

## How it works

`inBusinessHours` is a user-defined note declared at the top of the Katalog:

```yaml
notes:
  functions:
    - name: inBusinessHours
      expression: '{{ and weekday (timeInWindow "09:00" "18:00") }}'
```

It composes two built-in time notes — `weekday` and `timeInWindow` — into a single named boolean. Both `scaleUp` and `scaleDown` reference it, so the business hours rule lives in one place.

The `autoscale:` block sits alongside the Deployment declaration. Conditions use the same `when:` / `anyOf:` engine as everything else in Orkestra:

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    replicas: 2
    reconcile: true
    autoscale:
      min: 2
      max: 8
      cooldown: 2m
      scaleUp:
        conditions:
          when:
            - field: "{{ inBusinessHours }}"
              equals: "true"
        target: 8
      scaleDown:
        conditions:
          when:
            - field: "{{ inBusinessHours }}"
              equals: "false"
        target: 2
```

`target` is jump scaling — when conditions pass, replicas are set to exactly that value in one reconcile. The `cooldown` prevents oscillation if the window boundary falls on a resync tick.

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
kubectl get apiserver,deployment
```

The `PEAK` printer column flips between `true` and `false`. The `REPLICAS` column shows the live count. The Deployment `READY` column tracks the actual pod count converging to the target.

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

