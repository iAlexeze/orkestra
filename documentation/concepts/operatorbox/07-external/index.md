# External

The `external:` block makes HTTP calls before any resource group runs. Results land in `.external.<name>.*` — every resource template, `when:` condition, and status field that follows can read them. One call can gate a Deployment, fill a ConfigMap, drive a replica count, or feed the token for the next call.

```yaml
operatorBox:
  onReconcile:
    external:
      - name: healthCheck
        url: "{{ .spec.serviceUrl }}/health"
        expectedStatus: 200
        continueOnError: true
        timeout: 5s

    deployments:
      - name: "{{ .metadata.name }}"
        when:
          - field: external.healthCheck.status
            equals: "200"
```

---

## Where external sits in the pipeline

```text
informer cache → DeepCopy → normalize → mutation → validation
    → cross-CRD reads                    (.cross.* available in url: and body:)
    → external calls                     ← you are here
    → resource groups (deployments, services, configMaps, …)
    → read children → enrich
    → status fields
```

External runs after `cross:` context is injected — `.cross.*` is already accessible in `url:` and `body:` fields. Every call's result is available to every resource group and every status field that comes after it.

---

## The primary design decision: required vs optional

Every external call is either a **hard prerequisite** or **optional enrichment**. This is `continueOnError`.

| Setting | Behaviour | Use when |
|---|---|---|
| `continueOnError: false` (default) | Failure halts the reconcile. `Ready=False` is written to the CR condition. The operator retries on the next resync. | The call is an infrastructure requirement — nothing should be created without it. |
| `continueOnError: true` | Failure is logged. `.error` is set. The reconcile succeeds and resources are updated normally. | The call is a policy or enrichment decision — the system should keep running and the rejection should be visible in status, not as a reconcile error. |

**The real distinction is where the failure should surface.**

`continueOnError: false` surfaces failure as a reconcile error — visible in the CR condition, retried automatically. Use this when the service not responding means you have no information to act on (e.g., auth token fetch for a chained call).

`continueOnError: true` surfaces failure in `.external.<name>.error` and through your status fields — a 403 from a signing service is a policy decision, not an infrastructure failure. The operator keeps reconciling, the status shows what happened, and the Deployment gate does the enforcement. Use this when the call result carries meaning even on failure.

---

## Results in template context

After a call completes, four fields are available under `.external.<name>`:

| Field | Type | Value |
|---|---|---|
| `.status` | `string` | HTTP status code: `"200"`, `"403"`, `"503"`. Empty if the call failed before receiving a response. |
| `.body` | `string` | First 4096 bytes of the response body. |
| `.error` | `string` | Error message on failure. `""` on success. |
| `.called` | `string` | `"true"` when the call ran. `"false"` when skipped by `when:`/`anyOf:`. |

Access in any template expression or condition:

```yaml
# Gate a resource
- field: external.healthCheck.status
  equals: "200"

# Embed the response body
value: "{{ .external.appConfig.body }}"

# Use a prior call's result in a later call
token: "{{ .external.tokenFetch.body }}"

# Compare against a CR field — template expressions resolve in equals: values
- field: status.signedImage
  equals: "{{ .spec.image }}"
```

---

## Try it — no real services needed

All five examples run against the built-in mock dev server. `--dev-server` starts it on `:9999` and serves every endpoint the examples use:

```bash
ork init --pack use-cases
```

```bash
cd external/01-health-gate     # gate a Deployment on a live health check
ork run --dev-server

cd external/02-config-inject   # embed an API response into a ConfigMap every reconcile
ork run --dev-server

cd external/03-image-signing   # sign once per image — 4xx locks retries, 5xx stays open
ork run --dev-server

cd external/04-chained         # chain two calls — second uses the first call's token
ork run --dev-server

cd external/05-feature-flags   # external call drives replica count, toggle live without restart
ork run --dev-server
```

To toggle the feature flag while the operator is running:
```bash
curl -X POST http://localhost:9999/flags/my-app/v2Enabled/toggle
```

---

## Best practices

### Gate calls with `when:` to avoid unnecessary API calls

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

### Distinguish transient failures from definitive rejections

Not all failures are equal. An HTTP 5xx means the service is unavailable — retry makes sense. An HTTP 4xx means the service made a decision — retrying the same request will always get the same answer.

Write a `rejectedImage` (or equivalent) status field only on 4xx. Use the `prefix:` operator to match status code ranges. Gate the call on both the success status and the rejection status:

```yaml
external:
  - name: signImage
    continueOnError: true
    when:
      - field: status.signedImage
        notEquals: "{{ .spec.image }}"
      - field: status.rejectedImage    # closed on definitive rejection
        notEquals: "{{ .spec.image }}"

status:
  fields:
    # Lock out retries on 4xx — the signing service made a policy decision
    - path: rejectedImage
      value: "{{ .spec.image }}"
      when:
        - field: external.signImage.called
          equals: "true"
        - field: external.signImage.status
          prefix: "4"

    # Leave the gate open on 5xx — the service may recover, retry next reconcile
    - path: phase
      value: "SigningUnavailable"
      when:
        - field: external.signImage.called
          equals: "true"
        - field: external.signImage.status
          prefix: "5"
```

The reconcile loop is the retry. Gate the call correctly and retries are automatic where appropriate, suppressed where not.

### Use `continueOnError: false` for hard dependencies, `true` for policy decisions

If a resource must never exist without the call succeeding, use `continueOnError: false` — the reconcile halts and the caller sees a clear `Ready=False` condition.

If the call carries meaningful information even when it fails — a rejection reason, a flag value, an unavailability signal — use `continueOnError: true`. Gate resources with `when:` conditions on the call result. Status fields surface the details. The reconcile succeeds and the CR tells the full story.

### External calls can drive resource attributes, not just gate conditions

`external.*` fields are available everywhere in the template context — including in resource spec fields, not just `when:` conditions. Combine with two deployment entries for clean replica-count switching:

```yaml
external:
  - name: flags
    url: "{{ .spec.serviceUrl }}/flags/{{ .metadata.name }}/v2Enabled"
    continueOnError: true

deployments:
  - name: "{{ .metadata.name }}"
    replicas: "{{ .spec.replicas }}"   # full capacity when flag is on
    reconcile: true
    when:
      - field: external.flags.body
        equals: "true"

  - name: "{{ .metadata.name }}"
    replicas: "1"                      # baseline when flag is off or service is down
    reconcile: true
    when:
      - field: external.flags.body
        notEquals: "true"              # also catches empty body on service outage
```

### Keep tokens in environment variables

Never put bearer tokens or API keys in the Katalog. Use `$ENV_VAR` in the `token:` field:

```yaml
token: "$API_TOKEN"        # correct — expanded via os.ExpandEnv at runtime
token: "abc123secret"      # never — visible in the Katalog YAML
```

In production, mount the secret into the Orkestra runtime pod via `values.yaml`:

```yaml
runtime:
  extraEnvFrom:
    - secretRef:
        name: external-api-credentials
```

For local development with `--dev-server`, no token is needed — the mock server ignores auth headers entirely.

### Match `timeout:` to your resync period

If `resync: 15s` and the external call can take up to 10s, the operator spends most of each cycle waiting. Set `timeout:` to a fraction of the resync period — typically no more than 20–30%.

### Name calls with camelCase

Call names must be valid Go identifiers. Hyphens break template access.

```yaml
name: healthCheck      # correct   → {{ .external.healthCheck.status }}
name: health-check     # broken    → {{ .external.health-check.status }} fails
```

---

## Where to go next

- [Patterns](01-patterns.md) — health gates, config injection, image signing with rejection tracking, chaining, feature flag rollouts
- [Reference](02-reference.md) — full field table, constraints, template expression support, env var expansion
