# CronJob Pattern

**The Kubebuilder CronJob tutorial. Solved in YAML.**

The [Kubebuilder CronJob tutorial](https://book.kubebuilder.io/cronjob-tutorial/cronjob-tutorial)
is the canonical introduction to Kubernetes operator development. It walks
through building a CronJob controller from scratch — informers, reconcile
loops, finalizers, multi-version CRDs, conversion webhooks. Over ten pages
of Go, scaffolded code generation, and a separate webhook deployment.

This pattern solves the same problem. The Katalog is 120 lines of YAML.
There is no Go. There is no generated code. The conversion webhook is
Orkestra's `/convert` endpoint — already running, no extra deployment.

---

## The version change

Between v1 and v2, `schedule` changes from a cron string to a structured object:

```yaml
# v1 — one string
spec:
  schedule: "0 2 * * 1-5"

# v2 — named fields
spec:
  schedule:
    minute: "0"
    hour: "2"
    dayOfMonth: "*"
    month: "*"
    dayOfWeek: "1-5"   # Monday through Friday
```

The Kubebuilder tutorial requires handwritten conversion functions in Go to
translate between these formats. Orkestra expresses the same conversion as
template expressions using cron notes:

```yaml
# v1 → v2: cron notes extract each field from the string
- from: v1
  to: v2
  spec:
    schedule:
      minute:     "{{ cronMinute .spec.schedule }}"
      hour:       "{{ cronHour   .spec.schedule }}"
      dayOfMonth: "{{ cronDom    .spec.schedule }}"
      month:      "{{ cronMonth  .spec.schedule }}"
      dayOfWeek:  "{{ cronDow    .spec.schedule }}"

# v2 → v1: cronExpr reconstructs the canonical cron string
- from: v2
  to: v1
  spec:
    schedule: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
```

The round-trip is lossless. `"0 2 * * 1-5"` converts to
`{minute:"0", hour:"2", dom:"*", month:"*", dow:"1-5"}` and back to
`"0 2 * * 1-5"`. The cron notes handle `@`-macros (`@hourly` → `"0 * * * *"`)
transparently.

---

## What Kubebuilder required

The tutorial implementation needed:

| Component | Purpose | Size |
|---|---|---|
| `zz_generated_deepcopy.go` | Generated DeepCopy implementations | ~150 lines |
| `cronjob_types.go` (v1) | Type definitions | ~80 lines |
| `cronjob_types.go` (v2) | Type definitions | ~90 lines |
| `conversion.go` | Hub pattern, ConvertTo, ConvertFrom | ~60 lines |
| `conversion_test.go` | Round-trip tests | ~80 lines |
| `cronjob_controller.go` | The reconciler | ~200 lines |
| Webhook deployment | Separate pod for conversion | YAML + TLS setup |
| `cert-manager` or manual TLS | Certificate management | Configuration |
| `Makefile` targets | Build, generate, deploy | ~50 lines |

**Total: ~700 lines of code, two additional deployments, certificate management.**

What Orkestra requires: one `katalog.yaml`. RBAC is generated. The conversion
webhook is Orkestra's own `/convert` endpoint. No extra deployments.

---

## What the operator does

When you apply a CronJob CR, Orkestra:

1. **Validates** — ensures `spec.image` and `spec.schedule` are present. Deny rule blocks storage if they are missing.

2. **Mutates** — applies defaults (`concurrencyPolicy: Allow`, `successfulJobsHistoryLimit: 3`, `failedJobsHistoryLimit: 1`) before any validation runs.

3. **Converts** — if the CR is in v1 format, `/convert` splits the cron string into structured fields using cron notes and stores it as v2. If you read it back as v1, the cron string is reconstructed.

4. **Reconciles** — creates a Kubernetes `batch/v1 CronJob` with the schedule reconstructed from the structured fields. The child CronJob has owner references — it is garbage collected when the CR is deleted.

5. **Propagates status** — writes `phase`, `scheduleExpression`, `lastScheduleTime`, and `nextScheduleTime` to the CR's status after every successful reconcile. `phase` respects `spec.suspend`.

6. **Enforces suspension** — when `spec.suspend: true`, the child CronJob is suspended on the next reconcile cycle. When set back to false, it is unsuspended. No manual intervention.

7. **Corrects drift** — with `reconcile: true`, if someone edits the child CronJob directly, the next reconcile restores it to the declared state.

---

## Quick start

```bash
# 1. Install the CRD
kubectl apply -f crd.yaml

# 2. Create orkestra-system namespace (if not already present)
kubectl create namespace orkestra-system --dry-run=client -o yaml | kubectl apply -f -

# 3.Generate and Apply least-privilege RBAC
ork generate rbac -k komposer.yaml -o rbac.yaml

# or katalog
ork generate rbac -k katalog.yaml -o rbac.yaml    # either way, you generate the same exact RBAC
```

You can confirm with this:
ork diff rbac-kat.yaml rbac.yaml 

Expected:
```text
--- rbac-kat.yaml
+++ rbac.yaml
```
Finally, apply the RBAC:
```bash
kubectl apply -f rbac.yaml
```

# 4. Start Orkestra
ork run --katalog katalog.yaml

# 5. Apply a v2 CR
kubectl apply -f cr-v2.yaml

# 6. Watch it reconcile
kubectl get cronjobs
# NAME             SCHEDULE      PHASE    AGE
# print-hello-v2   */1 * * * *   Active   12s
# daily-backup     0 2 * * 1-5   Active   12s

# The schedule column shows the reconstructed cron expression
# from the structured fields — cronExpr in the status declaration

# 7. Apply a v1 CR — Orkestra converts it to v2 before storage
kubectl apply -f cr-v1.yaml

# Read it back as v1 — converted from v2 on the way out
kubectl get cronjob.v1.demo.orkestra.io print-hello-v1 -o yaml | grep schedule

# Read the same object as v2 — the stored format
kubectl get cronjob.v2.demo.orkestra.io print-hello-v1 -o yaml | grep -A5 schedule
```

---

## Observing conversions

```bash
# Port-forward to Orkestra's health API
kubectl port-forward svc/orkestra 8080:8080 -n orkestra-system &

# See live conversion metrics for the v2 CRD
curl localhost:8080/katalog/cronjob-v2 | jq '.conversion'
```

```json
{
  "enabled": true,
  "total": 47,
  "failures": 0,
  "avgLatencyMs": 0.4,
  "p95LatencyMs": 0.9
}
```

Or launch the Control Center to see conversions, queue depth, worker
utilisation, and every CRD's health in real time:

```bash
ork control start
```

---

## Suspending a CronJob

```bash
# Suspend all executions
kubectl patch cronjob daily-backup --type=merge -p '{"spec":{"suspend":true}}'

# Check status
kubectl get cronjob daily-backup -o jsonpath='{.status.phase}'
# Suspended

# The child Kubernetes CronJob is suspended on the next reconcile
kubectl get cronjob daily-backup -o jsonpath='{.spec.suspend}'
# true

# Resume
kubectl patch cronjob daily-backup --type=merge -p '{"spec":{"suspend":false}}'
```

The `phase` field in status uses the `ternary` note:
```yaml
- path: phase
  value: "{{ ternary .spec.suspend \"Suspended\" \"Active\" }}"
```

---

## Production deployment

```bash
# Generate RBAC from the production Komposer
ork generate rbac -k komposer.yaml -o rbac.yaml

# Apply everything
kubectl apply -f crd.yaml
kubectl apply -f rbac.yaml

# Deploy Orkestra with the Komposer as its Katalog
# (See examples/advanced/install.yaml for the full deployment)
```

---

## The notes that make this work

| Note | Used for |
|---|---|
| `cronMinute .spec.schedule` | Extract minute field from v1 cron string |
| `cronHour .spec.schedule` | Extract hour field |
| `cronDom .spec.schedule` | Extract day-of-month field |
| `cronMonth .spec.schedule` | Extract month field |
| `cronDow .spec.schedule` | Extract day-of-week field |
| `cronExpr min hr dom mon dow` | Reconstruct cron string from v2 fields |
| `ternary .spec.suspend "Suspended" "Active"` | Phase field in status |
| `default .spec.concurrencyPolicy "Allow"` | Default in conversion |

Every one of these was a Go function in the Kubebuilder tutorial.
In Orkestra they are notes — one word in a template expression.

---

## Files

| File | What it does |
|---|---|
| `crd.yaml` | v1 and v2 schemas, conversion webhook config |
| `katalog.yaml` | The complete operator declaration |
| `komposer.yaml` | Production overlay |
| `rbac.yaml` | Least-privilege RBAC (generated) |
| `cr-v1.yaml` | Example v1 CR |
| `cr-v2.yaml` | Example v2 CRs |
| `pattern.yaml` | Registry metadata |
| `README.md` | This file |
