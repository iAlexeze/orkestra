# CronJob Pattern

**The Kubebuilder CronJob tutorial. Solved in YAML.**

The [Kubebuilder CronJob tutorial](https://book.kubebuilder.io/cronjob-tutorial/cronjob-tutorial)
is the canonical introduction to Kubernetes operator development. It walks
through building a CronJob controller from scratch — informers, reconcile loops,
finalizers, multi-version CRDs, conversion webhooks. Over ten pages of Go,
scaffolded code generation, and a separate webhook deployment.

This pattern solves the same problem. The Katalog is 120 lines of YAML.
There is no Go. There is no generated code. The conversion webhook is Orkestra's
`/convert` endpoint — already running, no extra deployment needed.

**Verified in production:**

| Metric | Value |
|---|---|
| v1 → v2 conversions | 5,024 |
| v2 → v1 conversions | 6,255 |
| Conversion failures | **0** |
| v1 → v2 p95 latency | 0.69 ms |
| v2 → v1 p95 latency | 0.49 ms |
| Reconcile total | 4,903 |
| Reconcile errors | 1 (transient) |

---

## The version change

Between v1 and v2, `schedule` changes from a cron string to a structured object:

```yaml
# v1 — one string
spec:
  schedule: "0 2 * * 1-5"

# v2 — named fields, self-documenting
spec:
  schedule:
    minute: "0"
    hour: "2"
    dayOfMonth: "*"
    month: "*"
    dayOfWeek: "1-5"   # Monday through Friday
```

Orkestra expresses the conversion as template expressions using cron notes:

```yaml
# v1 → v2: split the cron string into named fields
- from: v1
  to: v2
  spec:
    schedule:
      minute:     "{{ cronMinute .spec.schedule }}"
      hour:       "{{ cronHour   .spec.schedule }}"
      dayOfMonth: "{{ cronDom    .spec.schedule }}"
      month:      "{{ cronMonth  .spec.schedule }}"
      dayOfWeek:  "{{ cronDow    .spec.schedule }}"

# v2 → v1: reconstruct the cron string from named fields
- from: v2
  to: v1
  spec:
    schedule: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
```

Round-trip: `"0 2 * * 1-5"` → `{minute:"0", hour:"2", dom:"*", month:"*", dow:"1-5"}` → `"0 2 * * 1-5"`. Lossless. `@`-macros (`@hourly`, `@daily`) are expanded transparently.

---

## What Kubebuilder required vs what Orkestra requires

| Component | Kubebuilder | Orkestra |
|---|---|---|
| Type definitions (v1) | `cronjob_types.go` ~80 lines | CRD schema in `crd.yaml` |
| Type definitions (v2) | `cronjob_types.go` ~90 lines | CRD schema in `crd.yaml` |
| DeepCopy generation | `zz_generated_deepcopy.go` ~150 lines | Not needed |
| Conversion logic | `conversion.go` ~60 lines | Template expressions in `katalog.yaml` |
| Conversion tests | `conversion_test.go` ~80 lines | Live metrics from `/convert` endpoint |
| Controller | `cronjob_controller.go` ~200 lines | `reconciler` block in `katalog.yaml` |
| Webhook deployment | Separate pod + TLS setup | Orkestra's own `/convert` endpoint |
| Certificate management | `cert-manager` or manual | Same cert Orkestra already uses |
| Build tooling | `Makefile` ~50 lines | `ork generate bundle` |
| **Total** | **~700 lines + 2 extra deployments** | **1 Katalog** |

---

## What the operator does

When you apply a CronJob CR, Orkestra:

1. **Converts** — if the CR is v1, Orkestra's `/convert` endpoint splits the cron string into structured fields and stores it as v2. When you read it back as v1, the cron string is reconstructed.

2. **Validates** — ensures `spec.image` and `spec.schedule` are present. A deny rule blocks the object if either is missing.

3. **Mutates** — applies defaults (`concurrencyPolicy: Allow`, `successfulJobsHistoryLimit: 3`, `failedJobsHistoryLimit: 1`) before validation runs. `mutateFirst: true` ensures defaults exist before rules check them.

4. **Reconciles** — creates a Kubernetes `batch/v1 CronJob` with the schedule reconstructed by `cronExpr`. The child CronJob has owner references — garbage collected when the CR is deleted.

5. **Propagates status** — writes `phase`, `scheduleExpression`, `lastScheduleTime`, and `nextScheduleTime` after every successful reconcile. Phase respects `spec.suspend` via the `ternary` note.

6. **Corrects drift** — with `reconcile: true`, any external change to the child CronJob is restored on the next reconcile cycle.

---

## Steps

### 1. Generate TLS certificates for the conversion webhook

Orkestra's `/convert` endpoint requires TLS. For development, generate self-signed certificates:

```bash
chmod +x ../installation/generate-certs.sh
../installation/generate-certs.sh
```

This creates the `orkestra-tls` secret in `orkestra-system`.

Copy the CA bundle from `/tmp/tls/caBundle.txt` into `crd.yaml` under the conversion webhook:

```yaml
conversion:
  strategy: Webhook
  webhook:
    clientConfig:
      caBundle: <paste caBundle.txt content here>
```

### 2. Install the CRD

```bash
kubectl apply -f crd.yaml
```

### 3. Create the `orkestra-system` namespace

```bash
kubectl create namespace orkestra-system --dry-run=client -o yaml | kubectl apply -f -
```

### 4. Generate and apply the runtime bundle

The bundle contains the ConfigMap (Komposer YAML) and least-privilege RBAC:

```bash
ork generate bundle -k komposer.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

### 5. Deploy Orkestra with webhook support

```bash
kubectl apply -f ../installation/install-webhook-support.yaml

kubectl wait --for=condition=available deployment/orkestra \
  -n orkestra-system --timeout=60s
```

### 6. Apply the CRs

```bash
# v2 CR — stored directly, no conversion needed
kubectl apply -f cr-v2.yaml

# v1 CR — Orkestra converts to v2 before storage
kubectl apply -f cr-v1.yaml
```

### 7. Verify reconciliation

```bash
kubectl get cj
```

Expected:
```
NAME             SCHEDULE      PHASE    AGE
daily-backup     0 2 * * 1-5   Active   8s
print-hello-v1   */1 * * * *   Active   12s
print-hello-v2   */1 * * * *   Active   8s
```

The `SCHEDULE` column shows the cron expression reconstructed by `cronExpr`
from v2 structured fields — whether the CR was applied as v1 or v2.

### 8. Verify the round-trip conversion

```bash
# v1 CR read back as v1 — schedule is the original string
kubectl get cronjob.v1.demo.orkestra.io print-hello-v1 -n default -o yaml | grep schedule
# schedule: '*/1 * * * *'

# Same object read as v2 — schedule is the structured object
kubectl get cronjob.v2.demo.orkestra.io print-hello-v1 -n default -o yaml | grep -A6 'schedule:'
# schedule:
#   dayOfMonth: '*'
#   dayOfWeek: '*'
#   hour: '*'
#   minute: '*/1'
#   month: '*'
```

The object is stored once in v2. Orkestra converts on read when v1 is requested.

---

## Observing conversions

```bash
kubectl port-forward svc/orkestra 8080:8080 -n orkestra-system &

curl localhost:8080/katalog/cronjob-v2 | jq '.conversion'
```

```json
{
  "enabled": true,
  "total": 11279,
  "failures": 0,
  "avgLatencyMs": 0.59,
  "p95LatencyMs": 0.69
}
```

Or launch the Control Center for live visualisation:

```bash
ork control start
```

---

## Suspending a CronJob

Always patch the Orkestra-managed CR — not the Kubernetes CronJob directly.
Orkestra propagates the change to the child on the next reconcile:

```bash
# Suspend — patch the CR, not the child
kubectl patch cronjob.v2.demo.orkestra.io daily-backup -n default \
  --type=merge -p '{"spec":{"suspend":true}}'

# Check the CR status
kubectl get cronjob.v2.demo.orkestra.io daily-backup -n default \
  -o jsonpath='{.status.phase}'
# Suspended

# Verify the child Kubernetes CronJob is also suspended
kubectl get cronjob daily-backup -n default -o jsonpath='{.spec.suspend}'
# true

# Resume
kubectl patch cronjob.v2.demo.orkestra.io daily-backup -n default \
  --type=merge -p '{"spec":{"suspend":false}}'
```

> The `phase` field is driven by a note: `{{ ternary .spec.suspend "Suspended" "Active" }}`.
> Phase changes appear in status within one reconcile interval (15s default).

---

## The notes that made this possible

| Note | What it does | Replaces in Go |
|---|---|---|
| `cronMinute .spec.schedule` | Extract minute field from cron string | `strings.Split(s, " ")[0]` + nil checks |
| `cronHour .spec.schedule` | Extract hour field | `strings.Split(s, " ")[1]` + nil checks |
| `cronDom .spec.schedule` | Extract day-of-month | `strings.Split(s, " ")[2]` + nil checks |
| `cronMonth .spec.schedule` | Extract month | `strings.Split(s, " ")[3]` + nil checks |
| `cronDow .spec.schedule` | Extract day-of-week | `strings.Split(s, " ")[4]` + nil checks |
| `cronExpr min hr dom mon dow` | Reconstruct canonical cron string | `fmt.Sprintf("%s %s %s %s %s", ...)` |
| `ternary .spec.suspend "Suspended" "Active"` | Conditional status value | `if/else` block |
| `default .spec.concurrencyPolicy "Allow"` | Field default with fallback | nil check + default assignment |

Every one of these was Go code in the Kubebuilder tutorial.
In Orkestra they are notes — one word in a template expression.

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Files

| File | Purpose |
|---|---|
| `crd.yaml` | v1 and v2 schemas, conversion webhook config |
| `katalog.yaml` | Complete operator — reconciler, mutation, validation, status, conversion |
| `komposer.yaml` | Production overlay with tuned workers |
| `bundle.yaml` | Least-privilege RBAC and ConfigMap (regenerate: `ork generate bundle -k komposer.yaml`) |
| `cr-v1.yaml` | Example v1 CR |
| `cr-v2.yaml` | Example v2 CRs |
| `pattern.yaml` | Registry metadata |
| `README.md` | This file |