# Temporal 01 — Business Hours

One CR. A Deployment and Service that exist only Mon–Fri 09:00–18:00 UTC and are gone outside that window. No CronJobs. No runbooks. The operator owns the full lifecycle.

**What you learn:** `when: time:` and `when: dayOfWeek:` for resource lifecycle gates; user-defined notes composing built-ins into domain vocabulary (`inBusinessHours`, `nextBusinessHour`); status that always reflects the current time state.

---

## What gets created

When the window is open:

```
DevEnvironment/my-env   Active   Environment is running
Deployment/my-env       2/2 Running
Service/my-env-svc      ClusterIP
```

When the window is closed:

```
DevEnvironment/my-env   Suspended   Suspended — resumes 2026-07-15T09:00:00Z
```

No Deployment. No Service. Removed automatically when the window closed; re-created when it reopens.

---

## How it works

The Deployment and Service carry `when:` conditions. The reconciler evaluates them on every resync — if they pass, the resource exists; if they fail, it is deleted:

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    replicas: "{{ .spec.replicas | default 2 }}"
    reconcile: true
    when:
      - time: {after: "09:00", before: "18:00"}
      - dayOfWeek: {weekday: true}

services:
  - name: "{{ .metadata.name }}-svc"
    reconcile: true
    when:
      - time: {after: "09:00", before: "18:00"}
      - dayOfWeek: {weekday: true}
```

Two user-defined notes give the status fields their vocabulary:

```yaml
notes:
  functions:
    - name: inBusinessHours
      expression: '{{ and weekday (timeInWindow "09:00" "18:00") }}'

    - name: nextBusinessHour
      expression: '{{ nextCron "0 9 * * 1-5" }}'
```

The status updates on every resync — so Control Center always shows when the environment resumes, even at 3am on a Sunday.

---

## Step 1 — Validate and inspect notes

```bash
ork validate --notes
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

> **Tip:** if the Deployment and Service don't appear, you're outside the window. Edit the `after`/`before` values in [`katalog.yaml`](katalog.yaml) (and the `timeInWindow` expression in the `inBusinessHours` note) to bracket your current local time, then save and restart `ork run`.
Watch the resources appear on the next resync.

In a separate terminal, watch the resources:

```bash
kubectl get devenvironment,deployment,service
```

During business hours: Deployment and Service are present, CR phase is `Active`.
Outside the window: only the CR remains, phase is `Suspended — resumes <RFC3339>`.

---

## Step 5 — Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The Resources tab shows the CR status — `phase` and `message` reflecting the current window state, updating on every resync.

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

---

## Next

[02 — Maintenance Window](../02-maintenance-window/README.md) — scale a StatefulSet to zero on a recurring cron window, using user-defined notes that reference each other.
