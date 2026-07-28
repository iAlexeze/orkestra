# validation

Declarative validation rules evaluated at admission time (via webhook) and at reconcile time.
Declared on a CRDEntry or inside a Motif's `admission` block.

```yaml
validation:
  rules:
    - field: spec.replicas
      operator: lte
      value: "10"
      valueType: int
      message: "replicas must not exceed 10"

    - field: spec.engine
      equals: postgres
      message: "only postgres engine is supported"
      action: deny

    - field: spec.image
      prefix: "myregistry.example.com/"
      message: "image must come from the internal registry"
      action: warn

    - field: spec.productionApproval
      operator: exists
      message: "production deploys require an approval ticket"
      action: deny
      when:
        - field: spec.environment
          equals: production

    - field: spec.domain
      operator: exists
      message: "spec.domain is required for cert workloads"
      action: deny
      when:
        - field: spec.workloadType
          equals: cert
      anyOf:
        - field: spec.workloadType
          equals: cert
        - field: spec.workloadType
          equals: monitoring
```

## `validation.include`

Pull rules from an external file to keep the Katalog compact. The file contains a `rules:` key with the same list structure as inline rules.

```yaml
validation:
  include: ./admission/apprequest.yaml   # relative to the katalog file
  rules:
    - field: spec.tier                   # appended after included rules
      operator: in
      value: "free,pro,enterprise"
      action: deny
```

`admission/apprequest.yaml`:

```yaml
rules:
  - field: spec.team
    operator: exists
    message: "spec.team is required"
    action: deny
  - field: spec.environment
    operator: in
    value: "staging,production"
    message: "spec.environment must be staging or production"
    action: deny
```

Included rules come first. Inline `rules:` append after. The `include:` path is resolved relative to the katalog file's directory — the katalog can be run from any working directory. The field is cleared from the runtime bundle after expansion.

## `validation.rules`

Each rule describes one check. Rules are evaluated in order.

| Field | Required | Description |
|-------|----------|-------------|
| `field` | yes | Dot-notation path in the CR, or a Go template expression (see below). |
| `message` | yes | Error or warning message. Supports Go templates resolved against the CR and notes FuncMap. |
| `action` | no | `deny` (default) — reject; `warn` — allow but log a warning |
| `operator` + `value` | yes* | Explicit comparison (see operators). `value` supports Go templates. |
| `valueType` | no | `string` (default), `int`, `float`, `bool` |
| `when` | no | All conditions must pass for this rule to be evaluated (AND). Empty means unconditional. Conditions support Go template expressions via `EvaluateWhen`. |
| `anyOf` | no | At least one condition must pass for this rule to be evaluated (OR). When both `when` and `anyOf` are declared, both blocks must pass. |

*Use either an operator+value pair or a shorthand field.

`when` and `anyOf` use the same `Condition` type as resource templates — see [06-when-conditions.md](06-when-conditions.md) for the full operator reference.

### Required fields are enforced automatically

`required: true` on an `idp.fields` or `idp.additionalFields` entry isn't just a form hint — at katalog load time, every `required: true` entry gets an implicit `exists` rule synthesized automatically, with `message:` already matching the field's `label:`. This is enforced at the API server for every client — the Control Center form, `curl`, a CI pipeline, a custom UI, `kubectl apply` — not only the one that happens to render a required-field asterisk. You don't hand-write a `validation.rules` entry for plain "this field must be present" checks — marking the field required is enough:

```yaml
idp:
  fields:
    targetRevision:
      label: "Branch / Tag"
      required: true
# → synthesizes: { field: spec.targetRevision, operator: exists,
#                  message: "Branch / Tag is required", action: deny }
```

`idp.additionalFields.labels`/`.annotations` entries synthesize the same way, through `getLabel`/`getAnnotation` rather than a raw dot-path — required annotation keys with dots (the Kubernetes-recommended `prefix/name` shape) are handled correctly without you needing to know that dot-path resolution would otherwise misparse them.

### Enum fields are validated automatically

`type: enum` on an `idp.fields` or `idp.additionalFields` entry also synthesizes a rule — an `in` check against the declared `enum:` list, with the same label-matching message. Membership is checked only when the field has a value: an enum field that isn't also `required: true` can still be omitted entirely, it just can't be set to something outside the list.

```yaml
idp:
  fields:
    workloadType:
      label: "Workload Type"
      type: enum
      enum: [app, cert, monitoring, infra]
      required: true
# → synthesizes: { field: spec.workloadType, operator: in,
#                  value: "app,cert,monitoring,infra",
#                  message: "Workload Type must be one of: app, cert, monitoring, infra",
#                  action: deny }
```

`idp.fields` entries backed by a spec field don't infer `enum:` from the CRD's OpenAPI schema for this purpose — declare it explicitly to opt a field into this synthesis, the same way `idp.additionalFields` already requires `type`/`enum` since those have no CRD schema to infer from.

### IDP-aware messages for hand-written rules

Auto-synthesis covers plain existence and enum membership. For anything more specific — a range, a prefix, a cross-field comparison — you still write the `validation.rules` entry by hand, and there the same principle applies: write `message:` using the field's `label:` instead of its raw `spec.*` path. A developer submitting through the Control Center form saw the label, not the YAML path — an error that echoes the path back is a translation the developer has to do themselves.

```yaml
idp:
  fields:
    image:
      label: "Container Image"

validation:
  rules:
    # before — leaks the internal field name
    - field: spec.image
      prefix: "myorg/"
      message: "spec.image must start with myorg/"

    # after — matches what the developer actually saw on the form
    - field: spec.image
      prefix: "myorg/"
      message: "Container Image must start with myorg/"
```

This applies whether the rule fires from a form submission, `kubectl apply`, or a CI pipeline — the message is the same either way, so keep it in the vocabulary of the person reading it, not the API shape.

### Template expressions in rules

`field:`, comparison values (`equals:`, `prefix:`, `min:`, `value:`, …), and `message:` are all resolved as Go templates before evaluation. The full CR fields and notes FuncMap are available.

When `field:` is a template expression, the resolved value is used in the comparison directly — not looked up as a path in the CR. The original expression is preserved in violation messages.

```yaml
notes:
  functions:
    - name: inBusinessHours
      expression: '{{ and weekday (timeInWindow "09:00" "18:00") }}'
    - name: allowedRegistry
      expression: "myorg/"

validation:
  rules:
    - field: "{{ inBusinessHours }}"
      equals: "true"
      action: deny
      message: "deployments are only allowed during business hours"

    - field: spec.image
      prefix: "{{ allowedRegistry }}"
      action: deny
      message: "image must come from {{ allowedRegistry }}"
```

## `validation.external`

External HTTP calls can be declared directly under `validation:`. They fire before any rule is evaluated, and their results are available in `field:` expressions as `.external.<name>.*`.

```yaml
validation:
  external:
    - name: healthCheck
      url: "{{ .spec.healthCheckUrl }}/health"
      expectedStatus: 200
      continueOnError: true
      fires:
        reconcile: false   # admission-only — skip during reconcile resyncs

  rules:
    - field: "{{ .external.healthCheck.status }}"
      equals: "200"
      action: deny
      message: "health check failed — CR rejected"
```

`fires.reconcile: false` means the call only runs at `kubectl apply` time. When omitted (default), the call also runs on every reconcile — the result is available the same way as `onReconcile.external` calls.

See [13-external.md](13-external.md) for the full field reference.

## Operators

| Shorthand | Operator | Description |
|-----------|----------|-------------|
| `equals` | `eq` | Field equals value |
| `notEquals` | `neq` | Field does not equal value |
| `prefix` | `prefix` | Field starts with value |
| `suffix` | `suffix` | Field ends with value |
| `contains` | `contains` | Field contains substring |
| `min` | `gte` | Field is greater than or equal (numeric) |
| `max` | `lte` | Field is less than or equal (numeric) |
| `greaterThan` | `gt` | Field is greater than (numeric) |
| `lessThan` | `lt` | Field is less than (numeric) |

## `action`

| Value | Effect |
|-------|--------|
| `deny` (default) | Webhook returns a rejection; reconcile fails with an error. |
| `warn` | Webhook allows the operation; a warning is logged. |

## When validation runs

- **At admission**: if `security.webhooks.admission.enabled: true` and the CRD's `webhooks.validation: true`.
- **At reconcile**: always — even without a webhook, rules are checked during each cycle.

---
