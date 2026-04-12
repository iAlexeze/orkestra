# CronJob Pattern — Solution 2: Without Webhooks

**The same CronJob operator. No conversion webhook. No TLS setup. No multi-version CRD.**

This solution uses Orkestra's `typeOf` condition to detect which schedule
format the user provided and branch accordingly — at reconcile time, inside
the operator, before any resource is created. The Kubernetes API server never
sees a v1 object. There is nothing to convert.

---

## The key insight

Conversion webhooks exist to solve one problem: the API server stores objects
in one version and clients request another. The webhook translates.

This approach eliminates the problem at the source. Orkestra detects which
schema the user wrote and always creates the right resource. One CRD. One version.
No conversion configuration. No webhook. No TLS.

```yaml
# Both of these are valid. Orkestra handles both.

# User writes a cron string (legacy format)
spec:
  schedule: "0 2 * * 1-5"

# User writes structured fields (modern format)
spec:
  schedule:
    minute: "0"
    hour: "2"
    dayOfMonth: "*"
    month: "*"
    dayOfWeek: "1-5"
```

---

## How it works

`typeOf .spec.schedule` returns `"string"` or `"map"` — the runtime type of the value.
`when:` conditions branch on this. Only one CronJob declaration fires per reconcile.

```yaml
onReconcile:
  cronJobs:
    # Path A — user wrote a cron string
    - name: "{{ .metadata.name }}"
      schedule: "{{ .spec.schedule }}"
      image: "{{ .spec.image }}"
      when:
        - field: spec.schedule
          operator: typeOf
          value: string

    # Path B — user wrote structured fields
    - name: "{{ .metadata.name }}"
      schedule: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
      image: "{{ .spec.image }}"
      when:
        - field: spec.schedule
          operator: typeOf
          value: map
```

The created Kubernetes CronJob is identical regardless of which format the user
chose. Orkestra normalizes at creation time. No stored v1 objects. No conversion
needed on read.

---

## Solution 1 vs Solution 2

| | Solution 1 (webhooks) | Solution 2 (no webhooks) |
|---|---|---|
| Kubernetes CRD versions | v1 + v2 (multi-version) | v1 only (single version) |
| Conversion webhook | Required — Orkestra `/convert` | Not needed |
| TLS certificates | Required | Not needed |
| Cluster prerequisites | Webhook support, TLS, RBAC | Nothing — `ork run` works |
| Bidirectional read | Yes — v1 reads reconstructed | No — one format, one CRD |
| Migration from v1 | Automatic on read | Objects stay as-is |
| Production deployment | Helm with webhook support | Helm, standard deployment |
| Best for | Public APIs, typed access | Internal operators, rapid iteration |

**Choose Solution 2 when:**
- You control who creates the CRs (internal platform, CI/CD)
- You want to iterate quickly without schema management overhead
- You are starting a new operator (no existing v1 objects to migrate)

**Choose Solution 1 when:**
- External clients use `kubectl` against a specific API version
- You have existing v1 objects in production that must remain readable as v1
- You need the Kubernetes API server to enforce the schema at admission time

---

## Quick start — `ork run`

Solution 2 requires nothing beyond a Kubernetes cluster with Orkestra.
No webhook configuration. No certificate generation. No extra steps.

```bash
# 1. Install the CRD (single version — no conversion config needed)
kubectl apply -f crd-single.yaml

# 2. Run the operator locally (for development)
ork run -k katalog-no-webhook.yaml

# That is all. Apply CRs in either format:
kubectl apply -f cr-string-schedule.yaml    # cron string format
kubectl apply -f cr-structured-schedule.yaml # structured format
```

For production, deploy using Helm with the same Katalog:

```bash
# Install Orkestra with this Katalog
helm install cronjob-operator orkestra/orkestra \
  --set katalog.configMap=cronjob-katalog \
  --namespace cronjob-system \
  --create-namespace

# Or add the Katalog to an existing Orkestra deployment
kubectl apply -f katalog-no-webhook.yaml -n orkestra-system
```

The Helm deployment uses the same `katalog-no-webhook.yaml` — no difference in
behaviour between `ork run` and the Helm-deployed version. The Katalog is the
complete operator definition in both cases.

---

## Complete Katalog

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: cronjob-operator
  description: >
    Single-version CronJob operator that accepts both cron string and structured
    schedule formats. No conversion webhook. No TLS. Runs with ork run.

spec:
  crds:
    cronjob:
      apiTypes:
        group: demo.orkestra.io
        version: v1
        kind: CronJob
        plural: cronjobs

      workers: 5
      resync: 15s

      mutation:
        mutateFirst: true
        rules:
          - field: spec.concurrencyPolicy
            default: "Allow"
          - field: spec.successfulJobsHistoryLimit
            default: 3
          - field: spec.failedJobsHistoryLimit
            default: 1
          - field: spec.suspend
            default: false

      validation:
        rules:
          - field: spec.image
            operator: exists
            message: "spec.image is required"
            action: deny
          - field: spec.schedule
            operator: exists
            message: "spec.schedule is required — either a cron string or a structured object"
            action: deny

      reconciler:
        default: true

        status:
          fields:
            # Phase driven by suspend flag
            - path: phase
              value: "{{ ternary .spec.suspend \"Suspended\" \"Active\" }}"

            # Detect and surface which schedule format the user provided.
            # typeOf returns "string" or "map" — directly useful for debugging.
            - path: scheduleFormat
              value: "{{ typeOf .spec.schedule }}"

            # The canonical cron expression — same regardless of input format.
            # String format: use as-is.
            # Structured format: reconstruct with cronExpr.
            - path: scheduleExpression
              value: "{{ if eq (typeOf .spec.schedule) \"map\" }}{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}{{ else }}{{ .spec.schedule }}{{ end }}"

            # When schedule is structured: show how many fields are defined.
            # Useful for validating partial structured schedules in the UI.
            - path: scheduleFieldsDefined
              value: "{{ if eq (typeOf .spec.schedule) \"map\" }}{{ len .spec.schedule }}{{ else }}1{{ end }}"

            - path: image
              value: "{{ .spec.image }}"
            - path: concurrencyPolicy
              value: "{{ .spec.concurrencyPolicy }}"
            - path: lastScheduleTime
              value: "{{ .children.cronjob.status.lastScheduleTime }}"

        onReconcile:
          cronJobs:
            # ── Path A: user wrote a cron string ─────────────────────────────
            # typeOf .spec.schedule == "string"
            # Use the schedule value directly — no transformation needed.
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              schedule: "{{ .spec.schedule }}"
              suspend: "{{ .spec.suspend }}"
              successfulJobsHistoryLimit: "{{ .spec.successfulJobsHistoryLimit }}"
              failedJobsHistoryLimit: "{{ .spec.failedJobsHistoryLimit }}"
              reconcile: true
              when:
                - field: spec.schedule
                  operator: typeOf
                  value: string

            # ── Path B: user wrote structured fields ──────────────────────────
            # typeOf .spec.schedule == "map"
            # Reconstruct the standard cron expression using cronExpr.
            # The Kubernetes CronJob always receives a plain cron string —
            # it does not know which format the user originally wrote.
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              schedule: "{{ cronExpr .spec.schedule.minute .spec.schedule.hour .spec.schedule.dayOfMonth .spec.schedule.month .spec.schedule.dayOfWeek }}"
              suspend: "{{ .spec.suspend }}"
              successfulJobsHistoryLimit: "{{ .spec.successfulJobsHistoryLimit }}"
              failedJobsHistoryLimit: "{{ .spec.failedJobsHistoryLimit }}"
              reconcile: true
              when:
                - field: spec.schedule
                  operator: typeOf
                  value: map
```

---

## Example CRs

```yaml
# cr-string-schedule.yaml — legacy format
apiVersion: demo.orkestra.io/v1
kind: CronJob
metadata:
  name: daily-backup
  namespace: default
spec:
  schedule: "0 2 * * 1-5"
  image: busybox:1.35

---
# cr-structured-schedule.yaml — modern format
apiVersion: demo.orkestra.io/v1
kind: CronJob
metadata:
  name: print-hello
  namespace: default
spec:
  schedule:
    minute: "*/1"
    hour: "*"
    dayOfMonth: "*"
    month: "*"
    dayOfWeek: "*"
  image: busybox:1.35
```

---

## What you see in the Control Center

Open `http://localhost:9090/controlcenter` after `ork run`.

The CR detail page shows:

```
phase:               Active
scheduleFormat:      map              ← typeOf in action
scheduleExpression:  */1 * * * *     ← cronExpr normalized it
scheduleFieldsDefined: 5             ← len of the structured schedule map
image:               busybox:1.35
lastScheduleTime:    2026-04-12T10:00:00Z
```

For a cron string input:

```
phase:               Active
scheduleFormat:      string           ← typeOf detected string format
scheduleExpression:  0 2 * * 1-5     ← passed through unchanged
scheduleFieldsDefined: 1             ← len returns 1 for a string
image:               busybox:1.35
```

The child CronJob is identical in both cases. The schedule format is an
input detail — not a storage detail.

---

## The upgrade path (v1 → v2, no migration)

When you want to "upgrade" users from the string format to the structured
format, you do not need a conversion webhook. You do not need to migrate
stored objects. You do not bump the CRD version.

You update the Katalog's status fields and documentation. Existing CRs
with string schedules continue to work. New users write structured schedules.
Both produce identical Kubernetes CronJobs.

When you eventually want to drop string schedule support, add a validation rule:

```yaml
validation:
  rules:
    - field: spec.schedule
      operator: typeOf
      value: map
      message: "spec.schedule must be a structured object — string format is deprecated. See migration guide."
      action: warn   # warn first, then deny after deprecation period
```

Old CRs still reconcile (the CronJob still runs). New CR creates emit a warning.
When you are ready, change `action: warn` to `action: deny`. No stored object
migration required at any point.

---

## The ConfigMap variant (zero CRD, zero versioning)

If you use a ConfigMap as your input surface instead of a custom CRD, the
versioning problem disappears entirely. ConfigMap `data` is `map[string]string`
— schema-free. Adding a new field is adding a new key. Old ConfigMaps continue
to work because absent keys evaluate as `notExists`.

```yaml
apiTypes:
  kind: ConfigMap
labelSelector:
  orkestra.io/katalog: cronjob-manager

onReconcile:
  cronJobs:
    - name: "{{ .metadata.name }}"
      schedule: "{{ .data.schedule }}"   # always a string in ConfigMap.data
      image: "{{ .data.image }}"
```

ConfigMap CRDs never suffer deletion cascades. Users never see `kubectl delete crd`
take down all their CronJob CRs. The ConfigMap is permanent infrastructure.

---

## Notes used in this solution

| Note | What it does | Used for |
|---|---|---|
| `typeOf .spec.schedule` | Returns `"string"` or `"map"` at runtime | Branching between schedule formats |
| `len .spec.schedule` | Returns field count for maps, char count for strings | Showing schedule completeness in status |
| `cronExpr min hr dom mon dow` | Reconstructs canonical cron string | Path B: normalizing structured schedule |
| `ternary .spec.suspend "Suspended" "Active"` | Conditional value | Phase field |
| `default .spec.concurrencyPolicy "Allow"` | Field default with fallback | Mutation defaults |

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Files

| File | Purpose |
|---|---|
| `crd-single.yaml` | Single-version CRD — no conversion webhook config |
| `katalog-no-webhook.yaml` | Complete operator — no conversion block |
| `cr-string-schedule.yaml` | CR using cron string format |
| `cr-structured-schedule.yaml` | CR using structured schedule object |
| `cleanup.sh` | Teardown — deletes CRs, CRD, and any child CronJobs |

For the webhook-based solution with bidirectional API version support, see `README.md`.