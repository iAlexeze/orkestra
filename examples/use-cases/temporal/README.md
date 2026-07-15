# Temporal Examples

Three examples showing how Orkestra handles time-dependent workloads. No CronJobs. No external schedulers. The reconciler re-evaluates built-in time notes on every resync and converges the cluster to the correct state for *right now*.

| Example | Pattern | What you learn |
|---------|---------|----------------|
| [01-business-hours](./01-business-hours/README.md) | Provision on schedule | `when: time:` + `when: dayOfWeek:` for resource lifecycle; `inBusinessHours` and `nextBusinessHour` user-defined notes |
| [02-maintenance-window](./02-maintenance-window/README.md) | Drain and restore on cron | `inMaintenance` and `activeReplicas` notes; replica scaling to zero; ConfigMap signaling downstream load balancers |
| [03-regional-peak](./03-regional-peak/README.md) | Per-timezone replica scaling | Per-region `timeInWindow` notes; time-driven replica counts; one operator replacing a fleet of CronJobs |
| [04-autoscale](./04-autoscale/README.md) | Business hours autoscaling | `autoscale:` on a Deployment; `target` jump scaling; time conditions as scaling signals |

---

## Built-in time notes used

| Note | What it returns |
|------|----------------|
| `weekday` | `true` Mon–Fri UTC |
| `weekend` | `true` Sat–Sun UTC |
| `timeInWindow "HH:MM" "HH:MM"` | `true` when now is within the UTC window |
| `timeNotInWindow "HH:MM" "HH:MM"` | opposite |
| `nextCron "cron-expr"` | next fire time as RFC3339 string |

---

## Run all examples from root

Simulate the whole track at once:

```bash
ork simulate
```

Run all four operators together from one Komposer:

```bash
ork run
```

Run the full E2E suite:

```bash
ork e2e
```

---

## Further reading

- [Concepts: Time-Dependent Workloads](https://orkestra.sh/docs/concepts/temporal/)
- [Reference: time notes](https://orkestra.sh/docs/reference/orkestra-notes/)
