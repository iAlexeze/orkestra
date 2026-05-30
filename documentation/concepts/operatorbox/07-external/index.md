# External

The `external:` block makes HTTP calls before any resource group runs. Results are injected into the template context as `.external.<name>.*` — every resource template, `when:` condition, and status field that follows can reference them.

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
    → cross-CRD reads
    → external calls  ← you are here
    → resource groups (deployments, services, configMaps, …)
    → read children → enrich
    → status fields
```

External runs after cross-CRD context is available (`.cross.*` is accessible in `url:` and `body:` fields) and before any resource is created or updated. Every call's result is available to everything that comes after it.

---

## The primary design decision

Every external call is either **required** or **optional**. This is controlled by `continueOnError:`.

| Behaviour | Setting | Use when |
|---|---|---|
| Failure halts the reconcile | `continueOnError: false` (default) | The call is a prerequisite — don't create anything if it fails |
| Failure is logged, reconcile continues | `continueOnError: true` | The call is optional — the system should keep running without it |

A failed required call records `Ready=False` on the CR condition and retries on the next resync. A failed optional call sets `.external.<name>.error` but the reconcile succeeds and resources are updated normally.

---

## Results in template context

After a call completes, four fields are available under `.external.<name>`:

| Field | Value |
|---|---|
| `.status` | HTTP status code string (`"200"`, `"404"`, `"503"`) |
| `.body` | First 4096 bytes of the response body |
| `.error` | Error message on failure; `""` on success |
| `.called` | `"true"` when the call ran; `"false"` when skipped by `when:` |

Access via dot notation in any template expression:

```yaml
# In a when: condition
- field: external.healthCheck.status
  equals: "200"

# In a value template
value: "{{ .external.appConfig.body }}"
```

---

## Try it

```bash
ork init --pack use-cases
cd external/01-health-gate     # gate a Deployment on a live health check
cd external/02-config-inject   # embed an API response body into a ConfigMap every reconcile
cd external/03-image-signing   # sign an image once, re-sign only when the image changes
cd external/04-chained         # chain two calls — second call uses the first call's token
ork run
```

---

## Best practices

### Gate calls with `when:` to avoid unnecessary API calls

External calls run on every reconcile by default. For calls that don't need to run every cycle, use a `when:` condition to skip them. Write the result to a status field on first success — subsequent reconciles check the status field instead of calling the API.

```yaml
external:
  - name: signImage
    url: "{{ .spec.serviceUrl }}/sign"
    method: POST
    when:
      # Only call when the image has not been signed yet.
      # status.signedImage is written after a successful sign (see status.fields below).
      - field: status.signedImage
        notEquals: "{{ .spec.image }}"

status:
  fields:
    - path: signedImage
      value: "{{ .spec.image }}"
      when:
        - field: external.signImage.status
          equals: "200"
```

The pattern: **call the API → write the result to status → gate future calls on the status field**. No annotations, no counters — just a status field and a condition.

### Use `continueOnError: false` for hard dependencies, `true` for optional enrichment

If a resource must not be created without the call succeeding, use `continueOnError: false`. If the call enriches or optimises but the system should keep running without it, use `continueOnError: true` and gate the enrichment path with `when: - field: external.<name>.error / operator: notExists`.

### Keep tokens in environment variables

Never put bearer tokens or API keys directly in the Katalog. Use `$ENV_VAR` in the `token:` field — the runtime expands it via `os.ExpandEnv` at call time.

```yaml
token: "$API_TOKEN"        # correct — read from env at runtime
token: "abc123secret"      # never — visible in the Katalog YAML
```

In production, mount the secret into the Orkestra runtime pod via `values.yaml`:

```yaml
# charts/orkestra/values.yaml
runtime:
  extraEnvFrom:
    - secretRef:
        name: external-api-credentials
  # or per-key:
  extraEnv:
    - name: API_TOKEN
      valueFrom:
        secretKeyRef:
          name: external-api-credentials
          key: API_TOKEN
```

Create the secret once:

```bash
kubectl create secret generic external-api-credentials \
  --from-literal=API_TOKEN=your-token-here
```

For local development, export the variable before `ork run`:

```bash
export API_TOKEN="your-token-here"
ork run
```

### Match `timeout:` to your resync period

If `resync: 15s` and the external call can take up to 10s, the operator spends most of each cycle waiting. Set `timeout:` to a fraction of the resync period — typically no more than 20–30%.

### Name calls with camelCase

Call names must be valid Go identifiers. Hyphens break template access.

```yaml
name: healthCheck      # correct   → {{ .external.healthCheck.status }}
name: health-check     # broken    → {{ .external.health-check.status }} fails to parse
```

### Treat the body as opaque text until parsed

`.external.<name>.body` is a raw string. Use `contains:` conditions to check for substrings, or inject it as-is into a ConfigMap for the app to parse. The operator does not parse JSON — that is intentional.


---

## Where to go next

- [Patterns](01-patterns.md) — health gates, config injection, image signing, chaining, notifications
- [Reference](02-reference.md) — full field table, constraints, `sleep:`, env var expansion

