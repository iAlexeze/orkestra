# CronJob Pattern — Solution 1: With Conversion Webhooks

**The Kubebuilder CronJob tutorial. Solved in YAML.**

The [Kubebuilder CronJob tutorial](https://book.kubebuilder.io/cronjob-tutorial/cronjob-tutorial)
is the canonical introduction to Kubernetes operator development. It walks
through building a CronJob controller from scratch — informers, reconcile loops,
finalizers, multi-version CRDs, conversion webhooks. Over ten pages of Go,
scaffolded code generation, and a separate webhook deployment.

This pattern solves the same problem. The Katalog is 120 lines of YAML.
There is no Go. There is no generated code. The conversion webhook is Orkestra
Gateway's `/convert` endpoint — already running, no extra deployment needed.

**Verified in production:**

| Metric | Value |
|---|---|
| v1 → v2 conversions | 5,024 |
| v2 → v1 conversions | 6,255 |
| Conversion failures | **0** |
| v1 → v2 p95 latency | 0.69 ms |
| v2 → v1 p95 latency | 0.49 ms |
| Reconcile total | 4,903 |
| Reconcile errors | 0 |

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

Orkestra expresses the conversion as two one-liner notes:

```yaml
# v1 → v2: cronToMap splits the cron string into the structured map
- from: v1
  to: v2
  spec:
    schedule: "{{ cronToMap .spec.schedule }}"

# v2 → v1: cronFromMap reassembles the structured map back into a cron string
- from: v2
  to: v1
  spec:
    schedule: "{{ cronFromMap .spec.schedule }}"
```

Round-trip: `"0 2 * * 1-5"` → `{minute:"0", hour:"2", dom:"*", month:"*", dow:"1-5"}` → `"0 2 * * 1-5"`. Lossless. `@`-macros (`@hourly`, `@daily`) are expanded transparently.

`cronFromAny` is used separately in status fields — it accepts either a string or a map and produces the canonical cron string. This is what drives the `SCHEDULE` column in `kubectl get`.

---

## What Kubebuilder required vs what Orkestra requires

| Component | Kubebuilder | Orkestra |
|---|---|---|
| Type definitions (v1) | `cronjob_types.go` ~80 lines | CRD schema in `crd.yaml` |
| Type definitions (v2) | `cronjob_types.go` ~90 lines | CRD schema in `crd.yaml` |
| DeepCopy generation | `zz_generated_deepcopy.go` ~150 lines | Not needed |
| Conversion logic | `conversion.go` ~60 lines | Template expressions in `katalog.yaml` |
| Conversion tests | `conversion_test.go` ~80 lines | Live metrics from `/convert` endpoint |
| Controller | `cronjob_controller.go` ~200 lines | `operatorBox` block in `katalog.yaml` |
| Webhook deployment | Separate pod + TLS setup | Orkestra's own `/convert` endpoint |
| Certificate management | `cert-manager` or manual | Same cert Orkestra already uses |
| Build tooling | `Makefile` ~50 lines | `ork generate bundle` |
| **Total** | **~700 lines + 2 extra deployments** | **1 Katalog** |

---

## What the operator does

When you apply a CronJob CR, Orkestra:

1. **Converts** — if the CR is v1, Orkestra's `/convert` endpoint splits the cron string into structured fields and stores it as v2. When you read it back as v1, the cron string is reconstructed.

2. **Reconciles** — creates a Kubernetes `batch/v1 CronJob` with the schedule reconstructed by `cronFromAny`. The child CronJob has owner references — garbage collected when the CR is deleted.

3. **Propagates status** — writes `phase`, `scheduleExpression`, `lastScheduleTime`, and `nextScheduleTime` after every successful reconcile. Phase respects `spec.suspend` via the `ternary` note.

4. **Corrects drift** — with `reconcile: true`, any external change to the child CronJob is restored on the next reconcile cycle.

---

## Steps

### 1. Install the ork CLI

```bash
curl get.orkestra.sh | bash
ork version
```

---

### 2. Apply the CRD

If you do not have a cluster yet, run:

```bash
ork create cluster            # creates a kind cluster
```

```bash
kubectl apply -f crd.yaml
```

### 3. Generate and apply the operator bundle

The bundle contains the ConfigMap with the runtime `katalog.yaml` generated from the `komposer` and least-privilege RBAC:

```bash
ork generate bundle -f komposer.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

### 4. Install Orkestra

`gateway.enabled=true` starts the `/convert` endpoint and manages TLS automatically — no manual certificate generation or `caBundle` patching required:

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --set gateway.enabled=true \
  --wait --timeout 120s
```

### 5. Apply the CRs

```bash
# v2 CR — stored directly, no conversion needed
kubectl apply -f cr-v2.yaml

# v1 CR — Orkestra converts to v2 before storage
kubectl apply -f cr-v1.yaml
```

### 7. Verify reconciliation

```bash
kubectl get cronjob -n default 
```

Expected:
```
NAME             SCHEDULE      PHASE    AGE
daily-backup     0 2 * * 1-5   Active   8s
print-hello-v1   */1 * * * *   Active   12s
print-hello-v2   */1 * * * *   Active   8s
```

The `SCHEDULE` column shows the cron expression reconstructed by `cronFromAny`
from the v2 schedule field — whether the CR was applied as v1 or v2.

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
kubectl port-forward svc/orkestra-gateway 8080:8080 -n orkestra-system &

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
ork control
# username:password → orkestra
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

| Note | Where used | What it does | Replaces in Go |
|---|---|---|---|
| `cronToMap .spec.schedule` | Conversion (v1→v2) | Split cron string into structured map | `strings.Split` + five field assignments |
| `cronFromMap .spec.schedule` | Conversion (v2→v1) | Map → canonical cron string | `strings.Join(fields, " ")` |
| `cronFromAny .spec.schedule` | Status, onCreate | String or map → canonical cron string | `if/else` type check + `strings.Split` + nil checks |
| `ternary .spec.suspend "Suspended" "Active"` | Status | Conditional status value | `if/else` block |
| `default false .spec.suspend` | Conversion | Field default with fallback | nil check + default assignment |

Every one of these was Go code in the Kubebuilder tutorial.
In Orkestra they are notes — one word in a template expression.

---

## E2E

Run the full lifecycle in one command — installs Orkestra with Gateway, applies the multi-version CRD, applies the v1 CR, asserts it is readable via both API versions, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: v1 CronJob CR created
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get cronjobs.v1.demo.orkestra.io print-hello-v1
        exitCode: 0

  - name: v1 CR readable via v2 API
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get cronjobs.v2.demo.orkestra.io print-hello-v1
        exitCode: 0
```

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
| `katalog.yaml` | Complete operator — reconciler, status, conversion paths |
| `komposer.yaml` | Production overlay with tuned workers |
| `bundle.yaml` | Least-privilege RBAC and ConfigMap (regenerate: `ork generate bundle -k komposer.yaml`) |
| `cr-v1.yaml` | Example v1 CR |
| `cr-v2.yaml` | Example v2 CRs |
| `README.md` | This file |