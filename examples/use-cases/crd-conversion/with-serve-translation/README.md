# CronJob Pattern — Solution 3: Serve Layer Field Translation

**The cron string never touches the CRD. The serve layer handles it.**

The CRD only accepts a structured schedule object. Callers submit a plain cron string
via `intent.yaml`. `serve.fields.values` fans the string out to the five structured
fields before the CR reaches the API server. No conversion webhook. No normalize. The
translation is declared in the Katalog's `serve` block and runs at the Gateway.

---

## The intent

```yaml
# intent.yaml — what the caller submits
target: cronjob-tutorial
name: daily-backup
schedule: "0 2 * * 1-5"
image: "gcr.io/google-containers/busybox:latest"
```

`target` is the name callers use — it does not have to match the CRD kind.
The Katalog maps `cronjob-tutorial` → `CronJob` (`demo.orkestra.io/v1`).

The caller speaks a human vocabulary. The CRD speaks a structured vocabulary.
The Katalog bridges them:

```yaml
serve:
  fields:
    schedule:
      values:
        schedule.minute:     '{{ cronMinute .value }}'
        schedule.hour:       '{{ cronHour   .value }}'
        schedule.dayOfMonth: '{{ cronDom    .value }}'
        schedule.month:      '{{ cronMonth  .value }}'
        schedule.dayOfWeek:  '{{ cronDow    .value }}'
```

What reaches the API server:

```yaml
spec:
  schedule:
    minute:     "0"
    hour:       "2"
    dayOfMonth: "*"
    month:      "*"
    dayOfWeek:  "1-5"
  image: gcr.io/google-containers/busybox:latest
```

The intent gate fires on the raw string, not the spec:

```yaml
validation:
  rules:
    - field: request.schedule
      operator: exists
      message: "schedule is required — use a cron expression (e.g. \"*/5 * * * *\")"
      action: deny
```

---

## What changes between approaches

| | with-webhooks | without-webhooks | with-serve-translation |
|---|---|---|---|
| CRD versions | 2 (v1 + v2) | 1 | 1 |
| Caller submits | kubectl YAML (v1 or v2) | kubectl YAML (string or map) | `ork serve apply` (flat string) |
| Translation point | API server (`/convert`) | Reconciler (`normalize`) | Gateway (`serve.fields.values`) |
| Extra infrastructure | None — Orkestra serves `/convert` | None | Gateway enabled |

---

## Steps

### 1. Install the ork CLI

```bash
curl get.orkestra.sh | bash
ork version
```

---

### 2. Validate the serve configuration

```bash
ork serve validate
ork serve validate --full
```

`--full` shows the resolved target/kind mapping and all field configs. Useful for
confirming the fanout expressions before testing against a cluster.

---

### 3. Test the translation locally

Verify the field fanout works against your `katalog.yaml`:

This reqires a token for authentication.

#### Set the token
```bash
export DEV_TOKEN=dev-token
```

#### Play the intent
```bash
ork serve play -i intent.yaml -t dev
# or
ork serve play -i intent.json -t dev
```

Both files contain the same payload — use whichever format fits your workflow.
This runs the full serve pipeline locally — `values` expressions are evaluated,
the built CR is printed, and the intent gate fires. No cluster needed.

Expected output shows `spec.schedule` as the structured object:

```json
"spec": {
  "image": "gcr.io/google-containers/busybox:latest",
  "schedule": {
    "dayOfMonth": "*",
    "dayOfWeek": "1-5",
    "hour": "2",
    "minute": "0",
    "month": "*"
  }
}
```

Try an invalid cron string to see the gate fire:

```bash
ork serve play -i intent-invalid.yaml -t dev
```

---

### 4. Simulate the operator

Run the full operator reconciliation loop locally without a cluster: Unlike the regular `ork simulate`, this runs step 3 above to create the CR, and then hands it off to simulate for reconcile time verification.

```bash
ork serve play -t dev --simulate
```

Verifies that the structured CR produces the expected Kubernetes CronJob.

> Note: _Without `--intent or -i` flag, ork serve reads the `intent.yaml` first or `intent.json` by default._

---

### 5. Apply the CRD

If you do not have a cluster yet, run:

```bash
ork create cluster
```

```bash
kubectl apply -f crd.yaml
```

### 6. Generate and apply the operator bundle

```bash
ork generate bundle -f katalog.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

### 7. Install Orkestra

`gateway.enabled=true` starts the Gateway API endpoint with the dev token as configured in [`values.yaml`](values.yaml):

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --wait --timeout 120s
```

### 8. Submit the intent

#### Proxy the Gateway API

```bash
ork proxy --for gateway
```

#### Apply the intent

```bash
ork serve apply --api http://localhost:8443 --token dev
```

The Gateway receives the flat cron string, runs `values` fanout, validates that
`request.schedule` exists, and applies the structured CR — all without the caller
knowing the CRD's internal schema.

### 9. Verify

```bash
kubectl get cronjobs.demo.orkestra.io -n default
```

```
NAME           SCHEDULE       PHASE    AGE
daily-backup   0 2 * * 1-5    Active   5s
```

The `SCHEDULE` column is driven by `cronFromAny .spec.schedule` in status —
reconstructs the cron string from the structured spec for display.

---

### 10. Run the end-to-end test

```bash
ork e2e
```

Spins up a kind cluster, installs Orkestra with gateway enabled, submits the intent
via the Gateway API, and verifies the structured spec fields and resulting CronJob.
Also asserts that an invalid cron string is rejected with a 422.

---

### Cleanup

```bash
./cleanup.sh
```

---

## Observing the translation

```bash
ork proxy --for gateway

curl localhost:8080/api/v1/schema/cronjob | jq '.fields.schedule'
```

The schema endpoint shows `schedule` as a string field (the intent surface),
not the structured spec. Callers see what they submit; the CRD's internal
structure is hidden behind the serve layer.

---

## Files

| File | Purpose |
|------|---------|
| `crd.yaml` | Single-version CRD — structured schedule object only |
| `katalog.yaml` | Operator with `serve.fields.values` fanout and validation gate |
| `serve/cronjob.yaml` | Serve field config — included by `katalog.yaml` |
| `intent.yaml` | Sample intent payload in YAML — flat cron string + image |
| `intent.json` | Same payload in JSON — for `curl` or API clients |
| `cr-default.yaml` | Pre-built structured CR — used by simulate and e2e |
| `simulate.yaml` | Local operator reconciliation test |
| `e2e.yaml` | Full end-to-end test including Gateway submission |
| `cleanup.sh` | Removes CRs and CRD from the cluster |
| `README.md` | This file |
