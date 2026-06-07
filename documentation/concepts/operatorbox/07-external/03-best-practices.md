# Best Practices

---

## Gate calls with `when:` to avoid unnecessary API calls

External calls run on every reconcile by default. For calls that don't need to run every cycle, use `when:` to skip them. Write the result to a status field on first success — subsequent reconciles check the status field instead of calling the API.

```yaml
external:
  - name: signImage
    url: "{{ .spec.serviceUrl }}/sign"
    method: POST
    when:
      - field: status.signedImage
        notEquals: "{{ .spec.image }}"   # skip if already signed

status:
  fields:
    - path: signedImage
      value: "{{ .spec.image }}"
      when:
        - field: external.signImage.status
          equals: "200"
```

The pattern: **call → write result to status → gate future calls on status**. No annotations, no counters — a status field and a condition.

---

## Distinguish transient failures from definitive rejections

Not all failures are equal. An HTTP 5xx means the service is unavailable — retry makes sense. An HTTP 4xx means the service made a decision — retrying the same request will always get the same answer.

Write a rejection-tracking status field only on 4xx. Use the `prefix:` operator to match status code ranges. Gate the call on both the success state and the rejection state:

```yaml
external:
  - name: signImage
    continueOnError: true
    when:
      - field: status.signedImage
        notEquals: "{{ .spec.image }}"
      - field: status.rejectedImage    # gate closes on definitive rejection
        notEquals: "{{ .spec.image }}"

status:
  fields:
    # 4xx — definitive rejection, close the gate
    - path: rejectedImage
      value: "{{ .spec.image }}"
      when:
        - field: external.signImage.called
          equals: "true"
        - field: external.signImage.status
          prefix: "4"

    # 5xx — transient, leave the gate open, retry next reconcile
    - path: phase
      value: "SigningUnavailable"
      when:
        - field: external.signImage.called
          equals: "true"
        - field: external.signImage.status
          prefix: "5"
```

The reconcile loop is the retry. Gate correctly and retries are automatic where appropriate, suppressed where they are not.

---

## Use `continueOnError: false` for hard dependencies, `true` for policy decisions

If a resource must never exist without the call succeeding, use `continueOnError: false` — the reconcile halts and `Ready=False` is written to the CR condition.

If the call carries meaningful information even when it fails — a rejection reason, a flag value, an unavailability signal — use `continueOnError: true`. Gate resources with `when:` conditions on the call result. Status fields surface the details. The reconcile succeeds and the CR tells the full story.

---

## Calls can drive resource attributes, not just gate conditions

`.external.*` fields are available in any template expression — including resource spec fields. This means a live flag value can set replica counts directly, not just control whether a resource is created:

```yaml
onCreate:
  # Full capacity when flag is on
  deployments:
    - name: "{{ .metadata.name }}"
      replicas: "{{ .spec.replicas }}"
      reconcile: true
      when:
        - field: external.flags.body
          equals: "true"

  # Baseline when flag is off or the flag service is unavailable
    - name: "{{ .metadata.name }}"
      replicas: "1"
      reconcile: true
      when:
        - field: external.flags.body
          notEquals: "true"   # also catches empty body on service outage
```

Both entries target the same Deployment name. Exactly one fires per reconcile. Declared under `onCreate`, `reconcile: true` is required — without it, `onCreate` would only create the Deployment once and never update it when the flag changes. Under `onReconcile`, drift correction happens automatically and `reconcile: true` is redundant.

---

## Keep tokens in environment variables

Never put bearer tokens or API keys in the Katalog. Use `$ENV_VAR` in the `token:` field — the value is expanded via `os.ExpandEnv` at call time:

```yaml
token: "$API_TOKEN"        # correct — expanded at runtime
token: "abc123secret"      # never — visible in the Katalog YAML
```

In production, mount the secret into the Orkestra pod via `values.yaml`:

```yaml
runtime:
  extraEnvFrom:
    - secretRef:
        name: external-api-credentials
```

For local development with `--dev-server`, no token is needed — the mock server ignores auth headers.

---

## Match `timeout:` to your resync period

If `resync: 15s` and the external call can take up to 10s, the operator spends most of each cycle waiting. Set `timeout:` to a fraction of the resync period — typically no more than 20–30%.

---

## Name calls with camelCase

Call names must be valid Go identifiers. Hyphens break template access:

```yaml
name: healthCheck      # correct   → {{ .external.healthCheck.status }}
name: health-check     # broken    → {{ .external.health-check.status }} fails to parse
```
