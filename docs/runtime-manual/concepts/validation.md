# Validation

Orkestra validation declares constraints that CR fields must satisfy.
When a constraint is violated, Orkestra either blocks the operation
(`action: deny`) or surfaces an advisory (`action: warn`).

Validation runs at two enforcement points automatically — you declare the
rules once and they apply at both.

---

## Two enforcement points, one declaration

### Admission time

When `ENABLE_WEBHOOKS=true`, Orkestra registers a `ValidatingWebhookConfiguration`
with the Kubernetes API server covering each CRD that has validation rules.
The API server calls Orkestra's `/validate` endpoint synchronously during
every `kubectl apply`.

```
kubectl apply -f website.yaml
    │
    ▼
API server → POST /validate (Orkestra)
    │             evaluates validation rules
    │             action: deny  → reject immediately
    │             action: warn  → Warning header, allow
    │
    ▼
Object stored (or rejected)
```

The user sees rejection immediately in their terminal — before the object is
stored. No polling. No event watching.

### Reconcile time

The same validation rules run inside the reconcile loop, unconditionally —
regardless of whether `ENABLE_WEBHOOKS` is set. This catches:

- CRs that existed before the webhook was enabled
- CRs created via API calls that bypass the admission webhook
- CRs whose fields drift into violation after admission

At reconcile time, `action: deny` halts reconciliation until the spec is
corrected. `action: warn` surfaces the violation on the `/katalog/{crd}`
health endpoint as an active warning.

!!! tip "Gradual rollout"
    Start with `ENABLE_WEBHOOKS=false`. Deploy your validation rules and
    observe `controller_validation_violations_total` in Prometheus. You can
    see exactly which CRs would have been rejected before enabling admission
    interception. When you are confident in the rules, set `ENABLE_WEBHOOKS=true`.

---

## Declaring validation rules

Validation rules live in the Katalog under `reconciler.validation`:

```yaml
- name: website
  apiTypes:
    kind: Website
    ...
  reconciler:
    validation:
      - field: spec.image
        prefix: "myorg/"
        message: "image must be from the myorg registry"
        action: deny

      - field: metadata.labels.team
        operator: exists
        message: "all resources should declare a team owner"
        action: warn

      - field: spec.replicas
        max: "10"
        message: "replicas cannot exceed 10"
        # action omitted → defaults to deny
```

---

## Actions

### `action: deny` (default)

**At admission:** The API server rejects the operation. `kubectl apply` returns
an error immediately. The object is not stored.

```
$ kubectl apply -f website.yaml
Error from server: admission webhook "validate.orkestra.konductor.io" denied the request:
[orkestra] validation failed: field "spec.image": image must be from the myorg registry (got: "nginx:1.25")
```

**At reconcile:** Reconciliation halts. A `Warning` Kubernetes event is recorded
on the CR. The workqueue retries with backoff. Child resources are not created
or updated until the spec is corrected.

### `action: warn`

**At admission:** The API server allows the operation. A warning line is printed
to stderr by `kubectl`. The object is stored.

```
$ kubectl apply -f website.yaml
Warning: orkestra: field "metadata.labels.team": all resources should declare a team owner
website.demo.orkestra.io/my-site created
```

**At reconcile:** Reconciliation continues. The violation is recorded as an
active warning on the `/katalog/{crd}` health endpoint:

```json
{
  "validation": {
    "activeWarnings": [
      {
        "cr": "my-site",
        "namespace": "default",
        "field": "metadata.labels.team",
        "message": "all resources should declare a team owner",
        "since": "2026-03-20T10:00:00Z"
      }
    ]
  }
}
```

!!! note "Warn mode is not silent"
    `action: warn` is advisory — it does not block. But it is observable:
    warnings appear in `kubectl apply` output, in `controller_validation_violations_total`
    metrics, and in the health API. Use it for policies you are rolling out or
    for informational governance that should not block deployments.

---

## Rule evaluation

All rules are evaluated before returning a result — not fail-fast. A CR with
three violations receives all three error messages in one `kubectl apply`
response. The user can fix everything in one cycle rather than discovering
violations one at a time.

Rules are evaluated in declaration order. All deny rules and all warn rules
are collected independently. The response is denied if any deny rules failed,
regardless of how many warn rules also fired.

---

## Supported operators

| Shorthand | Operator | Description |
|---|---|---|
| `equals:` | `equals` | Field exactly matches value |
| `notEquals:` | `notEquals` | Field does not match value |
| `prefix:` | `prefix` | Field starts with value |
| `suffix:` | `suffix` | Field ends with value |
| `contains:` | `contains` | Field contains value as substring |
| `min:` | `gt` | Field is numerically >= value |
| `max:` | `lt` | Field is numerically <= value |
| — | `exists` | Field is present and non-empty |
| — | `notExists` | Field is absent or empty |

```yaml
# Shorthand form (most readable)
- field: spec.image
  prefix: "myorg/"
  message: "bad image"

# Explicit form
- field: spec.image
  operator: prefix
  value: "myorg/"
  message: "bad image"
```

Field paths use dot-notation: `spec.image`, `metadata.labels.tier`,
`spec.database.engine`. Nested paths work at any depth.

---

## Enabling admission-time validation

Admission-time validation requires an HTTPS server and TLS certificates.
Set the following environment variables:

```bash
# Required environment variables
ENABLE_WEBHOOKS=true     # starts the HTTPS server on :8443 and registers /validate and /mutate
TLS_CERT=/tls/tls.crt   # serving certificate (also used as CA bundle)
TLS_KEY=/tls/tls.key    # serving key
```

!!! tip "Combining with conversion"
    If you also use declarative CRD version conversion, set `ENABLE_CONVERSION=true`
    alongside `ENABLE_WEBHOOKS=true`. Both features share the same HTTPS server.
    Either flag alone is sufficient to start it.

Orkestra automatically creates the `ValidatingWebhookConfiguration` object
at startup. The webhook covers only CRDs that have validation rules declared
in the Katalog. CRDs without rules are not included — no unnecessary API
server calls are made.

!!! warning "RBAC requirement"
    Orkestra needs permission to create and update webhook configurations:

    ```yaml
    - apiGroups: ["admissionregistration.k8s.io"]
      resources:
        - validatingwebhookconfigurations
      verbs: ["get", "create", "update", "patch"]
    ```

    The Helm chart includes this automatically when `webhooks.enabled: true`.

---

## Observability

Four metrics cover validation:

```
controller_validation_total{crd, result="passed|denied|warned"}
```
Aggregate rate per CRD. Alert on elevated `denied` or `warned` rates.

```
controller_validation_violations_total{crd, field, rule, action="deny|warn"}
```
Per-field violation detail. Use to understand which rules fire most and
whether deny rules are too restrictive.

Both are emitted at reconcile time. Admission-time metrics use the same
counters — every admission call also triggers a reconcile via the informer
watch event.
